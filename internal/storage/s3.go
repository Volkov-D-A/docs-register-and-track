package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const storageUsageRefreshTimeout = 30 * time.Second

// S3Storage предоставляет сервис для работы с объектным S3-хранилищем.
type S3Storage struct {
	client     *minio.Client
	bucketName string
}

// NewS3Storage создает новый экземпляр S3Storage.
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("check bucket exists failed: %w", err)
	}

	if !exists {
		slog.Info("Bucket does not exist, creating...", "bucket", cfg.BucketName)
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
		slog.Info("Bucket created", "bucket", cfg.BucketName)
	}

	return &S3Storage{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

// CheckS3 verifies that the configured bucket is reachable without creating
// or modifying storage. It is used by standalone server health checks.
func CheckS3(ctx context.Context, cfg config.S3Config) error {
	client, err := newS3Client(cfg)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	exists, err := client.BucketExists(checkCtx, cfg.BucketName)
	if err != nil {
		return fmt.Errorf("check bucket exists failed: %w", err)
	}
	if !exists {
		return fmt.Errorf("s3 bucket %q does not exist", cfg.BucketName)
	}
	return nil
}

func newS3Client(cfg config.S3Config) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.GetSecretAccessKey(), ""),
		Secure:       cfg.UseSSL,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init s3 client: %w", err)
	}
	return client, nil
}

// UploadFile загружает файл в объектное хранилище.
func (m *S3Storage) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to s3: %w", err)
	}
	return nil
}

// DownloadFileToWriter streams a bounded object directly to writer.
func (m *S3Storage) DownloadFileToWriter(ctx context.Context, objectName string, writer io.Writer, maxSize int64) error {
	info, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to stat object: %w", err)
	}
	if info.Size > maxSize {
		return fmt.Errorf("object size %d exceeds maximum allowed size %d", info.Size, maxSize)
	}
	obj, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from s3: %w", err)
	}
	defer obj.Close()

	limited := io.LimitReader(obj, maxSize+1)
	written, err := io.Copy(writer, limited)
	if err != nil {
		return fmt.Errorf("failed to read object data: %w", err)
	}
	if written > maxSize {
		return fmt.Errorf("object exceeds maximum allowed size %d", maxSize)
	}
	return nil
}

// DeleteFile удаляет файл из объектного хранилища.
func (m *S3Storage) DeleteFile(ctx context.Context, objectName string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object from s3: %w", err)
	}
	return nil
}

// RefreshStorageUsage performs a complete object scan and returns an exact
// byte count for the persisted aggregate.
func (m *S3Storage) RefreshStorageUsage(ctx context.Context) (objectCount int, totalBytes int64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	refreshCtx, cancel := context.WithTimeout(ctx, storageUsageRefreshTimeout)
	defer cancel()
	objectCh := m.client.ListObjects(refreshCtx, m.bucketName, minio.ListObjectsOptions{Recursive: true})
	for obj := range objectCh {
		if obj.Err != nil {
			return 0, 0, fmt.Errorf("failed to list objects in s3: %w", obj.Err)
		}
		objectCount++
		totalBytes += obj.Size
	}
	return objectCount, totalBytes, nil
}

// ListObjectNames is used by the read-only attachment reconciliation command.
func (m *S3Storage) ListObjectNames(ctx context.Context) ([]string, error) {
	objects := make([]string, 0)
	for object := range m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects in s3: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}
	return objects, nil
}
