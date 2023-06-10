package sender

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/dytlzl/go-forward-proxy/pkg/auth"
	"github.com/dytlzl/go-forward-proxy/pkg/common"
)

type sender struct {
	config     Config
	authorizer auth.Authorizer
}

func New(config Config, authorizer auth.Authorizer) sender {
	return sender{
		config:     config,
		authorizer: authorizer,
	}
}

func (s sender) authorize(r *http.Request) error {
	token, err := auth.ExtractToken(r)
	if err != nil {
		return err
	}
	return s.authorizer.Authorize(token)
}

func (s sender) handleTunneling(w http.ResponseWriter, r *http.Request) {
	if err := s.authorize(r); err != nil {
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

func (s sender) handleHTTP(w http.ResponseWriter, req *http.Request) {
	if err := s.authorize(req); err != nil {
		http.Error(w, err.Error(), http.StatusProxyAuthRequired)
		return
	}
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

func (s sender) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r)
	switch r.Method {
	case http.MethodConnect:
		s.handleTunneling(w, r)
	default:
		switch r.URL.String() {
		case "/-/ready":
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("OK"))
			if err != nil {
				log.Println(err)
			}
		default:
			s.handleHTTP(w, r)
		}
	}
}

func (s sender) Run() {
	addr := fmt.Sprintf(":%s", s.config.Port)
	log.Printf("Listening on %s\n", addr)
	server := &http.Server{
		Addr:    addr,
		Handler: s,
		// Disable HTTP/2.
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}
	log.Fatal(server.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath))
}
