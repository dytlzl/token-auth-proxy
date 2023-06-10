package main

import (
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dytlzl/go-forward-proxy/pkg/common"
)

func verifyToken(r *http.Request) error {
	token := r.Header.Get(common.ProxyAuthorizationHeaderName)
	r.Header.Del(common.ProxyAuthorizationHeaderName)
	if token != "nekot" {
		return errors.New("invalid token")
	}
	return nil
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	if err := verifyToken(r); err != nil {
		http.Error(w, err.Error(), http.StatusProxyAuthRequired)
		return
	}
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "failed to hijack", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	defer clientConn.Close()
	common.TransferBidirectionally(destConn, clientConn)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	common.CopyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func main() {
	addr := ":8888"
	log.Printf("Listening on %s\n", addr)
	server := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodConnect:
				handleTunneling(w, r)
			default:
				switch r.URL.String() {
				case "/-/ready":
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte("OK"))
					if err != nil {
						log.Println(err)
					}
				default:
					handleHTTP(w, r)
				}
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.Fatal(server.ListenAndServeTLS("./server.crt", "./server.key"))
}
