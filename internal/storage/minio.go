package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	storageInfoTTL            = 5 * time.Minute
	storageInfoRefreshTimeout = 30 * time.Second
)

type cachedStorageInfo struct {
	objectCount int
	totalSize   string
	updatedAt   time.Time
}

type storageInfoCall struct {
	done chan struct{}
	info cachedStorageInfo
	err  error
}

// MinioService предоставляет сервис для работы с объектным хранилищем MinIO.
type MinioService struct {
	client     *minio.Client
	bucketName string

	storageInfoMu         sync.Mutex
	storageInfo           cachedStorageInfo
	storageInfoGeneration uint64
	storageInfoCall       *storageInfoCall
	now                   func() time.Time
	scanStorageInfo       func(context.Context) (int, string, error)
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

	service := &MinioService{
		client:     client,
		bucketName: cfg.BucketName,
		now:        time.Now,
	}
	service.scanStorageInfo = service.scanStorageInfoFromMinio
	return service, nil
}

// UploadFile загружает файл в MinIO.
func (m *MinioService) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	_, err := m.client.PutObject(ctx, m.bucketName, objectName, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	if err != nil {
		return fmt.Errorf("failed to upload file to minio: %w", err)
	}
	m.invalidateStorageInfo()

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
	m.invalidateStorageInfo()
	return nil
}

// GetStorageInfo returns a short-lived cached bucket summary. Concurrent cache
// misses share one scan so opening several statistics screens cannot multiply
// ListObjects calls.
func (m *MinioService) GetStorageInfo(ctx context.Context) (objectCount int, totalSize string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := m.now
	if now == nil {
		now = time.Now
	}

	m.storageInfoMu.Lock()
	if m.storageInfo.updatedAt.Add(storageInfoTTL).After(now()) {
		info := m.storageInfo
		m.storageInfoMu.Unlock()
		return info.objectCount, info.totalSize, nil
	}
	call := m.storageInfoCall
	if call == nil {
		call = &storageInfoCall{done: make(chan struct{})}
		m.storageInfoCall = call
		generation := m.storageInfoGeneration
		m.storageInfoMu.Unlock()

		refreshCtx, cancel := context.WithTimeout(context.Background(), storageInfoRefreshTimeout)
		count, size, scanErr := m.scanStorageInfo(refreshCtx)
		cancel()

		m.storageInfoMu.Lock()
		call.info = cachedStorageInfo{objectCount: count, totalSize: size, updatedAt: now()}
		call.err = scanErr
		if scanErr == nil && generation == m.storageInfoGeneration {
			m.storageInfo = call.info
		}
		m.storageInfoCall = nil
		close(call.done)
		m.storageInfoMu.Unlock()
	} else {
		m.storageInfoMu.Unlock()
	}

	select {
	case <-call.done:
		if call.err != nil {
			return 0, "", call.err
		}
		return call.info.objectCount, call.info.totalSize, nil
	case <-ctx.Done():
		return 0, "", ctx.Err()
	}
}

// RefreshStorageInfo invalidates this process's short-lived cache and performs
// a new bucket scan. Cross-process coordination is handled by the caller.
func (m *MinioService) RefreshStorageInfo(ctx context.Context) (objectCount int, totalSize string, err error) {
	m.invalidateStorageInfo()
	return m.GetStorageInfo(ctx)
}

// RefreshStorageUsage performs a complete object scan and returns an exact
// byte count for the persisted aggregate. It deliberately bypasses the
// display cache used by GetStorageInfo.
func (m *MinioService) RefreshStorageUsage(ctx context.Context) (objectCount int, totalBytes int64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	refreshCtx, cancel := context.WithTimeout(ctx, storageInfoRefreshTimeout)
	defer cancel()
	objectCh := m.client.ListObjects(refreshCtx, m.bucketName, minio.ListObjectsOptions{Recursive: true})
	for obj := range objectCh {
		if obj.Err != nil {
			return 0, 0, fmt.Errorf("failed to list objects in minio: %w", obj.Err)
		}
		objectCount++
		totalBytes += obj.Size
	}
	m.invalidateStorageInfo()
	return objectCount, totalBytes, nil
}

func (m *MinioService) scanStorageInfoFromMinio(ctx context.Context) (objectCount int, totalSize string, err error) {
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

func (m *MinioService) invalidateStorageInfo() {
	m.storageInfoMu.Lock()
	m.storageInfo = cachedStorageInfo{}
	m.storageInfoGeneration++
	m.storageInfoMu.Unlock()
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
