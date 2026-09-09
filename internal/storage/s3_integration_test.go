package storage_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrations3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
)

func TestS3OperationsIntegration(t *testing.T) {
	s, c, cfg := integrations3.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, storage.CheckS3(ctx, cfg))
	_, err := storage.NewS3Storage(cfg)
	require.NoError(t, err)
	data := []byte("attachment\x00\xff\n")
	require.NoError(t, s.UploadFile(ctx, "file.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream"))
	info, err := c.StatObject(ctx, cfg.BucketName, "file.bin", minio.StatObjectOptions{})
	require.NoError(t, err)
	require.Equal(t, "application/octet-stream", info.ContentType)
	var out bytes.Buffer
	require.NoError(t, s.DownloadFileToWriter(ctx, "file.bin", &out, int64(len(data))))
	require.Equal(t, sha256.Sum256(data), sha256.Sum256(out.Bytes()))
	require.Error(t, s.DownloadFileToWriter(ctx, "file.bin", io.Discard, 1))
	names, err := s.ListObjectNames(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"file.bin"}, names)
	count, size, err := s.RefreshStorageUsage(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, int64(len(data)), size)
	require.NoError(t, s.DeleteFile(ctx, "file.bin"))
	require.NoError(t, s.DeleteFile(ctx, "file.bin"))
	require.Error(t, s.DownloadFileToWriter(ctx, "file.bin", io.Discard, 100))
	missing := cfg
	missing.BucketName += "-absent"
	require.Error(t, storage.CheckS3(ctx, missing))
	exists, err := c.BucketExists(ctx, missing.BucketName)
	require.NoError(t, err)
	require.False(t, exists)
	for _, bad := range []struct{ key, secret string }{{"wrong", "wrong"}, {cfg.AccessKeyID, "wrong"}, {"", ""}} {
		invalid := cfg
		invalid.AccessKeyID = bad.key
		invalid.SecretAccessKey = bad.secret
		require.Error(t, storage.CheckS3(ctx, invalid))
	}
}

type observeTransport struct {
	base      http.RoundTripper
	parts     atomic.Int32
	checksums atomic.Int32
	cancel    context.CancelFunc
}

func (o *observeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodPut && r.URL.Query().Get("partNumber") != "" {
		o.parts.Add(1)
		if o.cancel != nil {
			o.cancel()
		}
	}
	for key := range r.Header {
		if strings.Contains(strings.ToLower(key), "checksum") {
			o.checksums.Add(1)
			break
		}
	}
	return o.base.RoundTrip(r)
}

func TestS3MultipartAndCancellationIntegration(t *testing.T) {
	s, c, cfg := integrations3.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// minio-go/v7 v7.3.0 defaults to 16 MiB parts; 65 MiB also exceeds older 64 MiB defaults.
	data := bytes.Repeat([]byte("0123456789abcdef"), 65*1024*1024/16)
	require.NoError(t, s.UploadFile(ctx, "adapter-multipart.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream"))
	var adapterResult bytes.Buffer
	require.NoError(t, s.DownloadFileToWriter(ctx, "adapter-multipart.bin", &adapterResult, int64(len(data))))
	require.Equal(t, sha256.Sum256(data), sha256.Sum256(adapterResult.Bytes()))
	observer := &observeTransport{base: http.DefaultTransport}
	client, err := minio.New(cfg.Endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath, Transport: observer, TrailingHeaders: true})
	require.NoError(t, err)
	_, err = client.PutObject(ctx, cfg.BucketName, "multipart.bin", bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: "application/octet-stream", Checksum: minio.ChecksumCRC32C})
	require.NoError(t, err)
	require.Greater(t, observer.parts.Load(), int32(1), "must exercise real multipart")
	require.Positive(t, observer.checksums.Load(), "must send checksum headers")
	var out bytes.Buffer
	require.NoError(t, s.DownloadFileToWriter(ctx, "multipart.bin", &out, int64(len(data))))
	require.Equal(t, sha256.Sum256(data), sha256.Sum256(out.Bytes()))
	cancelled, stop := context.WithCancel(ctx)
	defer stop()
	observer.cancel = stop
	_, err = client.PutObject(cancelled, cfg.BucketName, "cancelled.bin", bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{})
	require.Error(t, err)
	require.ErrorIs(t, cancelled.Err(), context.Canceled)
	_, err = c.StatObject(ctx, cfg.BucketName, "cancelled.bin", minio.StatObjectOptions{})
	require.Error(t, err)
	for upload := range c.ListIncompleteUploads(ctx, cfg.BucketName, "cancelled.bin", true) {
		require.NoError(t, upload.Err)
		require.NoError(t, c.RemoveIncompleteUpload(ctx, cfg.BucketName, upload.Key))
	}
}

func TestS3PaginationIntegration(t *testing.T) {
	s, _, _ := integrations3.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var total int64
	for i := 0; i < 1005; i++ {
		data := strings.Repeat("x", i%11+1)
		total += int64(len(data))
		require.NoError(t, s.UploadFile(ctx, fmt.Sprintf("%04d.bin", i), strings.NewReader(data), int64(len(data)), "application/octet-stream"))
	}
	names, err := s.ListObjectNames(ctx)
	require.NoError(t, err)
	require.Len(t, names, 1005)
	count, size, err := s.RefreshStorageUsage(ctx)
	require.NoError(t, err)
	require.Equal(t, 1005, count)
	require.Equal(t, total, size)
}
