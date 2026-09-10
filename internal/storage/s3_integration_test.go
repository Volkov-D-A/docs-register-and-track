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
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	transfertypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	info, err := c.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(cfg.BucketName), Key: aws.String("file.bin")})
	require.NoError(t, err)
	require.Equal(t, "application/octet-stream", aws.ToString(info.ContentType))
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
	_, err = c.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(missing.BucketName)})
	require.Error(t, err)
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
	// Exceeds the transfer manager multipart threshold.
	data := bytes.Repeat([]byte("0123456789abcdef"), 65*1024*1024/16)
	require.NoError(t, s.UploadFile(ctx, "adapter-multipart.bin", bytes.NewReader(data), int64(len(data)), "application/octet-stream"))
	var adapterResult bytes.Buffer
	require.NoError(t, s.DownloadFileToWriter(ctx, "adapter-multipart.bin", &adapterResult, int64(len(data))))
	require.Equal(t, sha256.Sum256(data), sha256.Sum256(adapterResult.Bytes()))
	observer := &observeTransport{base: http.DefaultTransport}
	client, err := storage.NewS3Client(cfg, func(o *s3.Options) { o.HTTPClient = &http.Client{Transport: observer} })
	require.NoError(t, err)
	_, err = transfermanager.New(client, func(o *transfermanager.Options) { o.FailTimeout = 10 * time.Second }).UploadObject(ctx, &transfermanager.UploadObjectInput{Bucket: aws.String(cfg.BucketName), Key: aws.String("multipart.bin"), Body: bytes.NewReader(data), ContentLength: aws.Int64(int64(len(data))), ChecksumAlgorithm: transfertypes.ChecksumAlgorithmCrc32c})
	require.NoError(t, err)
	require.Greater(t, observer.parts.Load(), int32(1), "must exercise real multipart")
	require.Positive(t, observer.checksums.Load(), "must send checksum headers")
	var out bytes.Buffer
	require.NoError(t, s.DownloadFileToWriter(ctx, "multipart.bin", &out, int64(len(data))))
	require.Equal(t, sha256.Sum256(data), sha256.Sum256(out.Bytes()))
	cancelled, stop := context.WithCancel(ctx)
	defer stop()
	observer.cancel = stop
	_, err = transfermanager.New(client, func(o *transfermanager.Options) { o.FailTimeout = 10 * time.Second }).UploadObject(cancelled, &transfermanager.UploadObjectInput{Bucket: aws.String(cfg.BucketName), Key: aws.String("cancelled.bin"), Body: bytes.NewReader(data), ContentLength: aws.Int64(int64(len(data)))})
	require.Error(t, err)
	require.ErrorIs(t, cancelled.Err(), context.Canceled)
	_, err = c.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(cfg.BucketName), Key: aws.String("cancelled.bin")})
	require.Error(t, err)
	uploads, err := c.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{Bucket: aws.String(cfg.BucketName), Prefix: aws.String("cancelled.bin")})
	require.NoError(t, err)
	require.Empty(t, uploads.Uploads, "cancelled upload must be aborted")
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

func TestS3LegacyBackupRestoreIntegration(t *testing.T) {
	source, _, cfg := integrations3.Open(t)
	target, _, targetCfg := integrations3.Open(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	content := []byte("%PDF test legacy backup\x00\xff")
	key := "папка/file with spaces.pdf"
	require.NoError(t, source.UploadFile(ctx, key, bytes.NewReader(content), int64(len(content)), "application/pdf"))
	directory := t.TempDir()
	require.NoError(t, storage.ExportDirectory(ctx, cfg, directory))
	require.NoError(t, storage.ImportDirectory(ctx, targetCfg, directory))
	var result bytes.Buffer
	require.NoError(t, target.DownloadFileToWriter(ctx, key, &result, int64(len(content))))
	require.Equal(t, content, result.Bytes())
	require.ErrorContains(t, storage.ImportDirectory(ctx, targetCfg, directory), "empty bucket")
}
