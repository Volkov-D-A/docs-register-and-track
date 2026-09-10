package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

// ProbeS3 checks signed read/write/delete operations before the application bucket exists.
func ProbeS3(ctx context.Context, cfg config.S3Config) error {
	client, err := NewS3Client(cfg)
	if err != nil {
		return err
	}
	cfg.BucketName = "docflow-probe-" + uuid.NewString()
	bucket := aws.String(cfg.BucketName)
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		client.DeleteObject(cleanup, &s3.DeleteObjectInput{Bucket: bucket, Key: aws.String("ready")})
		client.DeleteBucket(cleanup, &s3.DeleteBucketInput{Bucket: bucket})
	}()
	var last error
	for {
		attempt, cancel := context.WithTimeout(ctx, 10*time.Second)
		last = func() error {
			if _, err := client.CreateBucket(attempt, &s3.CreateBucketInput{Bucket: bucket}); err != nil {
				return err
			}
			store := &S3Storage{client: client, bucketName: cfg.BucketName}
			if err := store.UploadFile(attempt, "ready", strings.NewReader("docflow-ready"), 13, "text/plain"); err != nil {
				return err
			}
			var out bytes.Buffer
			if err := store.DownloadFileToWriter(attempt, "ready", &out, 13); err != nil {
				return err
			}
			if out.String() != "docflow-ready" {
				return fmt.Errorf("S3 probe content mismatch")
			}
			if err := store.DeleteFile(attempt, "ready"); err != nil {
				return err
			}
			_, err := client.DeleteBucket(attempt, &s3.DeleteBucketInput{Bucket: bucket})
			return err
		}()
		cancel()
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("S3 readiness failed: %w", last)
		case <-time.After(2 * time.Second):
		}
	}
}

// ExportDirectory writes the legacy v2 objects tree. Reject keys which cannot be
// represented losslessly as local paths. os.Root prevents symlink escape as well.
func ExportDirectory(ctx context.Context, cfg config.S3Config, directory string) error {
	client, err := NewS3Client(cfg)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	store := &S3Storage{client: client, bucketName: cfg.BucketName}
	return store.WalkObjects(ctx, func(obj types.Object) error {
		key := aws.ToString(obj.Key)
		if !safeObjectPath(key) {
			return fmt.Errorf("object key cannot be represented in v2 archive: %q", key)
		}
		if err := root.MkdirAll(filepath.FromSlash(path.Dir(key)), 0700); err != nil {
			return err
		}
		file, err := root.OpenFile(filepath.FromSlash(key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		copyErr := store.DownloadFileToWriter(ctx, key, file, aws.ToInt64(obj.Size))
		syncErr := file.Sync()
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		return closeErr
	})
}
func safeObjectPath(key string) bool {
	return key != "." && fs.ValidPath(key) && !strings.ContainsAny(key, "\\:\x00") && filepath.IsLocal(filepath.FromSlash(key))
}

// ImportDirectory restores v2 into an empty bucket only. It never deletes remote data.
func ImportDirectory(ctx context.Context, cfg config.S3Config, directory string) error {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	var keys []string
	err = fs.WalkDir(root.FS(), ".", func(key string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !safeObjectPath(key) || !entry.Type().IsRegular() {
			return fmt.Errorf("unsafe archive member %q", key)
		}
		keys = append(keys, key)
		return nil
	})
	if err != nil {
		return err
	}
	store, err := NewS3Storage(cfg)
	if err != nil {
		return err
	}
	objects, err := store.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(cfg.BucketName), MaxKeys: aws.Int32(1)})
	if err != nil {
		return err
	}
	if len(objects.Contents) > 0 {
		return fmt.Errorf("restore requires an empty bucket")
	}
	for _, key := range keys {
		err := func() error {
			file, err := root.Open(filepath.FromSlash(key))
			if err != nil {
				return err
			}
			defer file.Close()
			info, err := file.Stat()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("not a regular object file")
			}
			var header [512]byte
			n, err := file.Read(header[:])
			if err != nil && err != io.EOF {
				return err
			}
			if _, err = file.Seek(0, io.SeekStart); err != nil {
				return err
			}
			return store.UploadFile(ctx, key, file, info.Size(), http.DetectContentType(header[:n]))
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
