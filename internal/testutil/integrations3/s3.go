// Package integrations3 creates unique disposable buckets in the isolated S3 stack.
package integrations3

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
)

func Open(t testing.TB) (*storage.S3Storage, *minio.Client, config.S3Config) {
	t.Helper()
	endpoint := os.Getenv("DOCFLOW_INTEGRATION_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("set DOCFLOW_INTEGRATION_S3_ENDPOINT (or run make integration-test) for real S3 tests")
	}
	cfg := config.S3Config{Endpoint: endpoint, AccessKeyID: "docflow_integration", SecretAccessKey: "docflow_integration_secret", BucketName: "docflow-test-" + uuid.NewString()}
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	require.NoError(t, err)
	s, err := storage.NewS3Storage(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for object := range client.ListObjects(ctx, cfg.BucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				t.Errorf("cleanup listing: %v", object.Err)
				break
			}
			if err := client.RemoveObject(ctx, cfg.BucketName, object.Key, minio.RemoveObjectOptions{}); err != nil {
				t.Errorf("cleanup object: %v", err)
			}
		}
		if err := client.RemoveBucket(ctx, cfg.BucketName); err != nil {
			t.Errorf("cleanup bucket: %v", err)
		}
	})
	return s, client, cfg
}
