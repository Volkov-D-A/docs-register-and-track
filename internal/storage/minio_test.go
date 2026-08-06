package storage

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMinioServiceInvalidEndpoint(t *testing.T) {
	service, err := NewMinioService(config.MinioConfig{
		Endpoint:        "http://bad endpoint",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
		BucketName:      "bucket",
	})

	require.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "failed to init minio client")
}
