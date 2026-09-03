package serverclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedDocflowRootCAIsValid(t *testing.T) {
	block, rest := pem.Decode(embeddedDocflowRootCA)
	require.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)
	assert.Empty(t, rest)

	certificate, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	assert.True(t, certificate.IsCA)
	assert.True(t, certificate.BasicConstraintsValid)
}

func TestNewHTTPClientUsesEmbeddedRootCAAndSecureTLS(t *testing.T) {
	client, err := newHTTPClient()
	require.NoError(t, err)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)

	block, _ := pem.Decode(embeddedDocflowRootCA)
	certificate, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	_, err = certificate.Verify(x509.VerifyOptions{Roots: transport.TLSClientConfig.RootCAs})
	require.NoError(t, err)
}
