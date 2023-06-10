package receiver

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/dytlzl/go-forward-proxy/pkg/auth"
	"github.com/dytlzl/go-forward-proxy/pkg/cert"
	"github.com/dytlzl/go-forward-proxy/pkg/common"
)

type receiver struct {
	config Config
}

func New(config Config) receiver {
	return receiver{
		config: config,
	}
}

func (a receiver) connectToProxy() (*tls.Conn, error) {
	tlsConf, err := cert.TLSClientConfig()
	if err != nil {
		return nil, err
	}
	return tls.Dial("tcp", a.config.ProxyAddr, tlsConf)
}

func (a receiver) handleTunneling(w http.ResponseWriter, req *http.Request) {
	destConn, err := a.connectToProxy()
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	auth.InjectToken(req, a.config.Token)
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

func (a receiver) handleHTTP(w http.ResponseWriter, req *http.Request) {
	tlsConf, err := cert.TLSClientConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	auth.InjectToken(req, a.config.Token)
	transport := &http.Transport{
		Proxy: func(_ *http.Request) (*url.URL, error) {
			return url.Parse(fmt.Sprintf("https://%s", a.config.ProxyAddr))
		},
		TLSClientConfig: tlsConf,
	}
	resp, err := transport.RoundTrip(req)
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

func (a receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodConnect:
		a.handleTunneling(w, r)
	default:
		a.handleHTTP(w, r)
	}
}

func (a receiver) Run() {
	addr := fmt.Sprintf(":%s", a.config.Port)
	log.Printf("Listening on %s\n", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: a,
		// Disable HTTP/2.
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}
	log.Fatal(server.ListenAndServe())
}
