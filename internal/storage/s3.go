package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

const storageUsageRefreshTimeout = 30 * time.Second

type S3Storage struct {
	client     *s3.Client
	bucketName string
}

// NewS3Client constructs a path-style, statically authenticated client for our S3 endpoint.
func NewS3Client(cfg config.S3Config, options ...func(*s3.Options)) (*s3.Client, error) {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	endpoint := scheme + "://" + cfg.Endpoint
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(cfg.Endpoint, "://") {
		return nil, fmt.Errorf("failed to init s3 client: invalid endpoint")
	}
	if cfg.AccessKeyID == "" || cfg.GetSecretAccessKey() == "" {
		return nil, fmt.Errorf("failed to init s3 client: credentials are required")
	}
	return s3.New(s3.Options{BaseEndpoint: aws.String(endpoint), Region: "us-east-1", UsePathStyle: true,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.GetSecretAccessKey(), ""), RetryMaxAttempts: 3}, options...), nil
}

func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	client, err := NewS3Client(cfg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if _, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.BucketName)}); err != nil {
		var response *smithyhttp.ResponseError
		if !errors.As(err, &response) || response.HTTPStatusCode() != 404 {
			return nil, fmt.Errorf("check bucket exists failed: %w", err)
		}
		if _, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(cfg.BucketName)}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}
	return &S3Storage{client: client, bucketName: cfg.BucketName}, nil
}

func CheckS3(ctx context.Context, cfg config.S3Config) error {
	client, err := NewS3Client(cfg)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(cfg.BucketName)})
	if err != nil {
		return fmt.Errorf("check bucket exists failed: %w", err)
	}
	return nil
}

func (m *S3Storage) UploadFile(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	if size < 0 {
		return fmt.Errorf("object size must be nonnegative")
	}
	_, err := transfermanager.New(m.client, func(o *transfermanager.Options) { o.Concurrency = 2; o.FailTimeout = 10 * time.Second }).UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket: aws.String(m.bucketName), Key: aws.String(key), Body: data, ContentLength: aws.Int64(size), ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to s3: %w", err)
	}
	return nil
}

func (m *S3Storage) DownloadFileToWriter(ctx context.Context, key string, writer io.Writer, maxSize int64) error {
	if maxSize < 0 {
		return fmt.Errorf("maximum size must be nonnegative")
	}
	obj, err := m.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(m.bucketName), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("failed to get object from s3: %w", err)
	}
	defer obj.Body.Close()
	if obj.ContentLength == nil || *obj.ContentLength > maxSize {
		return fmt.Errorf("object exceeds maximum allowed size %d", maxSize)
	}
	// Read up to the limit, then probe one byte without overflowing maxSize+1.
	n, err := io.Copy(writer, io.LimitReader(obj.Body, maxSize))
	if err != nil {
		return fmt.Errorf("failed to read object data: %w", err)
	}
	var extra [1]byte
	more, err := io.ReadFull(obj.Body, extra[:])
	if more != 0 {
		return fmt.Errorf("object exceeds maximum allowed size %d", maxSize)
	}
	if err != io.EOF {
		return fmt.Errorf("failed to finish object read: %w", err)
	}
	if n != *obj.ContentLength {
		return fmt.Errorf("object size mismatch")
	}
	return nil
}

func (m *S3Storage) DeleteFile(ctx context.Context, key string) error {
	_, err := m.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(m.bucketName), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("failed to remove object from s3: %w", err)
	}
	return nil
}

func (m *S3Storage) WalkObjects(ctx context.Context, visit func(types.Object) error) error {
	pages := s3.NewListObjectsV2Paginator(m.client, &s3.ListObjectsV2Input{Bucket: aws.String(m.bucketName)})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects in s3: %w", err)
		}
		for _, obj := range page.Contents {
			if err := visit(obj); err != nil {
				return err
			}
		}
	}
	return nil
}
func (m *S3Storage) RefreshStorageUsage(ctx context.Context) (count int, total int64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, storageUsageRefreshTimeout)
	defer cancel()
	err = m.WalkObjects(ctx, func(obj types.Object) error { count++; total += aws.ToInt64(obj.Size); return nil })
	if err != nil {
		return 0, 0, err
	}
	return
}
func (m *S3Storage) ListObjectNames(ctx context.Context) ([]string, error) {
	names := make([]string, 0)
	err := m.WalkObjects(ctx, func(obj types.Object) error { names = append(names, aws.ToString(obj.Key)); return nil })
	return names, err
}
