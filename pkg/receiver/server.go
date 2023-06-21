package receiver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/dytlzl/token-auth-proxy/pkg/auth"
	"github.com/dytlzl/token-auth-proxy/pkg/cert"
	"github.com/dytlzl/token-auth-proxy/pkg/common"
)

type receiver struct {
	config           Config
	targetDomainsSet map[string]struct{}
}

func New(config Config) receiver {
	targetDomainsSet := map[string]struct{}{}
	for _, domain := range config.TargetDomains {
		targetDomainsSet[domain] = struct{}{}
	}
	return receiver{
		config:           config,
		targetDomainsSet: targetDomainsSet,
	}
}

func (a receiver) connectToProxy() (*tls.Conn, error) {
	tlsConf, err := cert.TLSClientConfig(a.config.CACertPath)
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
	defer destConn.Close()
	auth.InjectToken(req, a.config.Token)
	err = req.Write(destConn)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
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
	tlsConf, err := cert.TLSClientConfig(a.config.CACertPath)
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
	common.TransferHTTPRequest(transport, w, req)
}

func (a receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r)
	if _, existing := a.targetDomainsSet[r.URL.Hostname()]; existing {
		switch r.Method {
		case http.MethodConnect:
			a.handleTunneling(w, r)
		default:
			a.handleHTTP(w, r)
		}
	} else {
		switch r.Method {
		case http.MethodConnect:
			common.HandleTunneling(w, r)
		default:
			common.TransferHTTPRequest(http.DefaultTransport, w, r)
		}
	}
}

func (a receiver) Run() {
	addr := fmt.Sprintf(":%s", a.config.Port)
	log.Printf("Listening on %s\n", addr)
	server := &http.Server{
		Addr:         addr,
		Handler:      a,
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	<-ctx.Done()
	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatalln(err)
	}
}
