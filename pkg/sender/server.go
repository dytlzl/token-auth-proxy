package sender

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dytlzl/token-auth-proxy/pkg/auth"
	"github.com/dytlzl/token-auth-proxy/pkg/common"
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
	return s.authorizer.Authorize(r.Context(), token)
}

func (s sender) handleTunneling(w http.ResponseWriter, r *http.Request) {
	if err := s.authorize(r); err != nil {
		http.Error(w, err.Error(), http.StatusProxyAuthRequired)
		return
	}
	common.HandleTunneling(w, r)
}

func (s sender) handleHTTP(w http.ResponseWriter, req *http.Request) {
	if err := s.authorize(req); err != nil {
		http.Error(w, err.Error(), http.StatusProxyAuthRequired)
		return
	}
	common.TransferHTTPRequest(http.DefaultTransport, w, req)
}

func (s sender) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r)
	switch r.Method {
	case http.MethodConnect:
		s.handleTunneling(w, r)
	default:
		switch r.URL.Path {
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
		Addr:         addr,
		Handler:      s,
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()
	go func() {
		err := server.ListenAndServeTLS(s.config.TLSCertPath, s.config.TLSKeyPath)
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
