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

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "bytes", size: 512, want: "512 B"},
		{name: "kilobytes", size: 1536, want: "1.5 KB"},
		{name: "megabytes", size: 2 * 1024 * 1024, want: "2.0 MB"},
		{name: "gigabytes", size: 3 * 1024 * 1024 * 1024, want: "3.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSize(tt.size))
		})
	}
}
