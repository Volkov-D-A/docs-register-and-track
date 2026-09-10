// Package integrations3 creates unique disposable buckets in the isolated S3 stack.
package integrations3

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Open(t testing.TB) (*storage.S3Storage, *s3.Client, config.S3Config) {
	t.Helper()
	endpoint := os.Getenv("DOCFLOW_INTEGRATION_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("set DOCFLOW_INTEGRATION_S3_ENDPOINT (or run make integration-test) for real S3 tests")
	}
	cfg := config.S3Config{Endpoint: endpoint, AccessKeyID: "docflow_integration", SecretAccessKey: "docflow_integration_secret", BucketName: "docflow-test-" + uuid.NewString()}
	client, err := storage.NewS3Client(cfg)
	require.NoError(t, err)
	s, err := storage.NewS3Storage(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		names, err := s.ListObjectNames(ctx)
		if err != nil {
			t.Errorf("cleanup listing: %v", err)
			return
		}
		for _, key := range names {
			if err := s.DeleteFile(ctx, key); err != nil {
				t.Errorf("cleanup object: %v", err)
			}
		}
		if _, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(cfg.BucketName)}); err != nil {
			t.Errorf("cleanup bucket: %v", err)
		}
	})
	return s, client, cfg
}
