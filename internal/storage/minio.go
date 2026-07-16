package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioService предоставляет сервис для работы с объектным хранилищем MinIO.
type MinioService struct {
	client     *minio.Client
	bucketName string
}

// NewMinioService создает новый экземпляр MinioService.
func NewMinioService(cfg config.MinioConfig) (*MinioService, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.GetSecretAccessKey(), ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init minio client: %w", err)
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

// DownloadFile скачивает файл из MinIO и возвращает его содержимое.
func (m *MinioService) DownloadFile(ctx context.Context, objectName string, maxSize int64) ([]byte, error) {
	var data bytes.Buffer
	if err := m.DownloadFileToWriter(ctx, objectName, &data, maxSize); err != nil {
		return nil, err
	}
	return data.Bytes(), nil
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

// GetStorageInfo возвращает количество объектов и суммарный размер хранилища в человекочитаемом формате.
func (m *MinioService) GetStorageInfo(ctx context.Context) (objectCount int, totalSize string, err error) {
	var count int
	var size int64

	objectCh := m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return 0, "", fmt.Errorf("failed to list objects in minio: %w", obj.Err)
		}
		count++
		size += obj.Size
	}

	return count, formatSize(size), nil
}

// formatSize форматирует размер в байтах в человекочитаемый формат.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
