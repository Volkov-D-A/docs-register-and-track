package serverclient

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"net/http"
	"time"
)

//go:embed certs/docflow-root-ca.crt
var embeddedDocflowRootCA []byte

func newHTTPClient() (*http.Client, error) {
	roots, err := x509.SystemCertPool()
	if err != nil || roots == nil {
		roots = x509.NewCertPool()
	}
	if ok := roots.AppendCertsFromPEM(embeddedDocflowRootCA); !ok {
		return nil, fmt.Errorf("embedded Docflow root CA is invalid")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   3 * time.Minute,
	}, nil
}
