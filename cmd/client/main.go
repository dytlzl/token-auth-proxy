package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func handleTunneling(w http.ResponseWriter, req *http.Request) {
	proxyAddr := "localhost:8888"
	var err error
	destConn, err := net.DialTimeout("tcp", proxyAddr, 10*time.Second)
	req.Write(destConn)
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
	wg := sync.WaitGroup{}
	wg.Add(2)
	go transfer(destConn, clientConn, &wg)
	go transfer(clientConn, destConn, &wg)
	wg.Wait()
}

func transfer(dst io.Writer, src io.Reader, wg *sync.WaitGroup) error {
	defer wg.Done()
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Println(err)
	}
	return nil
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	proxyAddr := "localhost:8888"
	req.URL, _ = url.Parse(fmt.Sprintf("http://%s", proxyAddr))
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func main() {
	addr := ":8989"
	log.Printf("Listening on %s\n", addr)
	server := &http.Server{
		Addr: ":8989",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println(r)
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
