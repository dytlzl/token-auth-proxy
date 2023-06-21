package main

import (
	"github.com/dytlzl/token-auth-proxy/pkg/receiver"
)

func main() {
	conf := receiver.Config{
		Port:          "8989",
		ProxyAddr:     "localhost:8888",
		CACertPath:    "./server.crt",
		Token:         "nekot",
		TargetDomains: []string{"www.google.com"},
	}
	receiver.New(conf).Run()
}
