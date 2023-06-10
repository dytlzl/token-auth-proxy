package cert

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

func TLSClientConfig(caCertPath string) (*tls.Config, error) {
	pem, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(pem)
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
