package storage

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3StorageInvalidEndpoint(t *testing.T) {
	service, err := NewS3Storage(config.S3Config{
		Endpoint:        "http://bad endpoint",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
		BucketName:      "bucket",
	})

	require.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "failed to init s3 client")
}

func TestLegacyObjectPaths(t *testing.T) {
	for _, key := range []string{"../escape", "/absolute", "a/../b", "a//b", "a\\b", "C:escape", ".", "a/", ""} {
		assert.False(t, safeObjectPath(key), key)
	}
	for _, key := range []string{"file.bin", "папка/имя с пробелами.pdf"} {
		assert.True(t, safeObjectPath(key), key)
	}
}
