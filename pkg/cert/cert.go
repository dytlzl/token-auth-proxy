package cert

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func TLSClientConfig() (*tls.Config, error) {
	certPath := "./server.crt"
	pem, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(pem)
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
