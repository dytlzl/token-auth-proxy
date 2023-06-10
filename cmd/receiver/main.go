package main

import (
	"github.com/dytlzl/go-forward-proxy/pkg/receiver"
)

func main() {
	conf := receiver.Config{
		Port:      "8989",
		ProxyAddr: "localhost:8888",
		CACertPath: "./server.crt",
		Token:     "nekot",
	}
	receiver.New(conf).Run()
}
