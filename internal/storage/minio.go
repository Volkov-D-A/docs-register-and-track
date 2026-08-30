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

// MinioService предоставляет сервис для работы с объектным хранилищем MinIO.
type MinioService struct {
	client     *minio.Client
	bucketName string
}

// NewMinioService создает новый экземпляр MinioService.
func NewMinioService(cfg config.MinioConfig) (*MinioService, error) {
	client, err := newMinioClient(cfg)
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

	return &MinioService{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

// CheckMinio verifies that the configured bucket is reachable without creating
// or modifying storage. It is used by standalone server health checks.
func CheckMinio(ctx context.Context, cfg config.MinioConfig) error {
	client, err := newMinioClient(cfg)
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
		return fmt.Errorf("minio bucket %q does not exist", cfg.BucketName)
	}
	return nil
}

func newMinioClient(cfg config.MinioConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.GetSecretAccessKey(), ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init minio client: %w", err)
	}
	return client, nil
}

// UploadFile загружает файл в MinIO.
func (m *MinioService) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to minio: %w", err)
	}
	return nil
}

// DownloadFileToWriter streams a bounded object directly to writer.
func (m *MinioService) DownloadFileToWriter(ctx context.Context, objectName string, writer io.Writer, maxSize int64) error {
	info, err := m.client.StatObject(ctx, m.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to stat object: %w", err)
	}
	if info.Size > maxSize {
		return fmt.Errorf("object size %d exceeds maximum allowed size %d", info.Size, maxSize)
	}
	obj, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from minio: %w", err)
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

// DeleteFile удаляет файл из MinIO.
func (m *MinioService) DeleteFile(ctx context.Context, objectName string) error {
	err := m.client.RemoveObject(ctx, m.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object from minio: %w", err)
	}
	return nil
}

// RefreshStorageUsage performs a complete object scan and returns an exact
// byte count for the persisted aggregate.
func (m *MinioService) RefreshStorageUsage(ctx context.Context) (objectCount int, totalBytes int64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	refreshCtx, cancel := context.WithTimeout(ctx, storageUsageRefreshTimeout)
	defer cancel()
	objectCh := m.client.ListObjects(refreshCtx, m.bucketName, minio.ListObjectsOptions{Recursive: true})
	for obj := range objectCh {
		if obj.Err != nil {
			return 0, 0, fmt.Errorf("failed to list objects in minio: %w", obj.Err)
		}
		objectCount++
		totalBytes += obj.Size
	}
	return objectCount, totalBytes, nil
}

// ListObjectNames is used by the read-only attachment reconciliation command.
func (m *MinioService) ListObjectNames(ctx context.Context) ([]string, error) {
	objects := make([]string, 0)
	for object := range m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects in minio: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}
	return objects, nil
}
