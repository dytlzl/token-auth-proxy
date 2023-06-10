package main

import (
	"github.com/dytlzl/go-forward-proxy/pkg/auth"
	"github.com/dytlzl/go-forward-proxy/pkg/sender"
)

func main() {
	conf := sender.Config{
		Port:        "8888",
		TLSCertPath: "./server.crt",
		TLSKeyPath:  "./server.key",
	}
	sender.New(conf, auth.SingleTokenAuthorizer{
		Token: "nekot",
	}).Run()
}
