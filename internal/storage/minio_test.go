package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMinioServiceInvalidEndpoint(t *testing.T) {
	service, err := NewMinioService(config.MinioConfig{
		Endpoint:        "http://bad endpoint",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
		BucketName:      "bucket",
	})

	require.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "failed to init minio client")
}

func TestMinioServiceGetStorageInfoCachesAndInvalidates(t *testing.T) {
	now := time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)
	calls := 0
	service := &MinioService{now: func() time.Time { return now }}
	service.scanStorageInfo = func(context.Context) (int, string, error) {
		calls++
		return calls, "1 MB", nil
	}

	count, size, err := service.GetStorageInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, "1 MB", size)
	_, _, err = service.GetStorageInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, calls)

	service.invalidateStorageInfo()
	count, _, err = service.GetStorageInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, 2, calls)
}

func TestMinioServiceGetStorageInfoSharesConcurrentRefresh(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	calls := 0
	service := &MinioService{now: time.Now}
	service.scanStorageInfo = func(context.Context) (int, string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		close(started)
		<-release
		return 3, "3 MB", nil
	}

	results := make(chan error, 2)
	for range 2 {
		go func() { _, _, err := service.GetStorageInfo(context.Background()); results <- err }()
	}
	<-started
	close(release)
	require.NoError(t, <-results)
	require.NoError(t, <-results)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestMinioServiceGetStorageInfoDoesNotCacheErrorsOrCancelSharedRefresh(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	service := &MinioService{now: time.Now}
	service.scanStorageInfo = func(context.Context) (int, string, error) {
		calls++
		if calls == 1 {
			close(started)
			<-release
			return 0, "", errors.New("minio unavailable")
		}
		return 4, "4 MB", nil
	}

	firstDone := make(chan error, 1)
	go func() { _, _, err := service.GetStorageInfo(context.Background()); firstDone <- err }()
	<-started
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := service.GetStorageInfo(cancelled)
	assert.ErrorIs(t, err, context.Canceled)
	close(release)
	assert.EqualError(t, <-firstDone, "minio unavailable")
	count, _, err := service.GetStorageInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 4, count)
	assert.Equal(t, 2, calls)
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{name: "bytes", size: 512, want: "512 B"},
		{name: "kilobytes", size: 1536, want: "1.5 KB"},
		{name: "megabytes", size: 2 * 1024 * 1024, want: "2.0 MB"},
		{name: "gigabytes", size: 3 * 1024 * 1024 * 1024, want: "3.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSize(tt.size))
		})
	}
}
