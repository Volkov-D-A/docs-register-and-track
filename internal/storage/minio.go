package storage

import (
	"bytes"
	"context"
	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"fmt"
	"io"
	"log"

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

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("check bucket exists failed: %w", err)
	}

	if !exists {
		log.Printf("Bucket %s does not exist, creating...\n", cfg.BucketName)
		err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &MinioService{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

// UploadFile загружает файл в MinIO.
func (m *MinioService) UploadFile(ctx context.Context, objectName string, data []byte, contentType string) error {
	reader := bytes.NewReader(data)

	_, err := m.client.PutObject(ctx, m.bucketName, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to minio: %w", err)
	}

	return nil
}

// DownloadFile скачивает файл из MinIO и возвращает его содержимое.
func (m *MinioService) DownloadFile(ctx context.Context, objectName string) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from minio: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
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
