package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/dytlzl/go-forward-proxy/pkg/common"
)

func connectToProxy() (*tls.Conn, error) {
	proxyAddr := "localhost:8888"
	certPath := "./server.crt"
	pem, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(pem)
	return tls.Dial("tcp", proxyAddr, &tls.Config{
		RootCAs: caCertPool,
	})
}

func handleTunneling(w http.ResponseWriter, req *http.Request) {
	destConn, err := connectToProxy()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	req.Header.Add(common.ProxyAuthorizationHeaderName, "nekot")
	err = req.Write(destConn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Println(err)
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	defer clientConn.Close()
	common.TransferBidirectionally(destConn, clientConn)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	proxyAddr := "localhost:8888"
	req.URL, _ = url.Parse(fmt.Sprintf("https://%s", proxyAddr))
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
	addr := ":8989"
	log.Printf("Listening on %s\n", addr)
	server := &http.Server{
		Addr: ":8989",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodConnect:
				handleTunneling(w, r)
			default:
				handleHTTP(w, r)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}

	log.Fatal(server.ListenAndServe())
}
