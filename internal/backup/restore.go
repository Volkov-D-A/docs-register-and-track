package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const InstanceLeaseID int64 = 0x444f43464c4f5702

// Restore requires an empty target and leaves a persistent marker on any failure.
// Operators must preserve that marker until a successful recovery is verified.
func Restore(ctx context.Context, pg PostgreSQL, s3cfg config.S3Config, archive, staging string, maxBytes int64, schema int) error {
	if staging == "" || !filepath.IsAbs(staging) {
		return fmt.Errorf("persistent recovery directory is required")
	}
	if err := os.MkdirAll(staging, 0700); err != nil {
		return err
	}
	work, err := os.MkdirTemp(staging, "restore-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	prepared, err := PrepareRestore(ctx, pg, archive, filepath.Join(work, "contents"), maxBytes, schema)
	if err != nil {
		return err
	}
	db, err := pg.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var acquired bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", InstanceLeaseID).Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("stop the ordinary server before restoring")
	}
	defer func() {
		limited, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn.ExecContext(limited, "SELECT pg_advisory_unlock($1)", InstanceLeaseID)
	}()
	return RestorePrepared(ctx, pg, s3cfg, prepared, staging, schema, true)
}

// RestorePrepared is for a coordinator already holding the server lifetime
// lease. It still requires an empty database and bucket.
func RestorePrepared(ctx context.Context, pg PostgreSQL, s3cfg config.S3Config, prepared PreparedRestore, staging string, schema int, finalize bool) error {
	m := prepared.Manifest
	dump := filepath.Join(prepared.Directory, "database.dump")
	db, err := pg.Open(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = pg.RequireEmpty(ctx); err != nil {
		return err
	}
	client, err := storage.NewS3Client(s3cfg)
	if err != nil {
		return err
	}
	// ListBuckets distinguishes an absent bucket from authentication/network failure.
	buckets, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return err
	}
	exists := false
	for _, bucket := range buckets.Buckets {
		if aws.ToString(bucket.Name) == s3cfg.BucketName {
			exists = true
		}
	}
	if exists {
		objects, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(s3cfg.BucketName), MaxKeys: aws.Int32(1)})
		if err != nil {
			return err
		}
		if len(objects.Contents) > 0 {
			return fmt.Errorf("restore requires an empty bucket")
		}
	}
	marker := filepath.Join(staging, "recovery-required")
	if finalize {
		if err = durableMarker(marker, []byte(m.ID+"\n")); err != nil {
			return err
		}
	} else if _, err = os.Stat(marker); err != nil {
		return fmt.Errorf("replacement marker is required: %w", err)
	}
	if err = pg.Restore(ctx, dump); err != nil {
		return err
	}
	var restoredSchema int
	var dirty bool
	if err = db.QueryRowContext(ctx, "SELECT version,dirty FROM schema_migrations").Scan(&restoredSchema, &dirty); err != nil {
		return err
	}
	if dirty || restoredSchema < 1 || restoredSchema > schema || (m.Schema != 0 && m.Schema != restoredSchema) {
		return fmt.Errorf("restored schema is incompatible or dirty")
	}
	if err = ValidateReferences(ctx, db, m); err != nil {
		return err
	}
	store, err := storage.NewS3Storage(s3cfg)
	if err != nil {
		return err
	}
	for _, obj := range m.Objects {
		file, err := os.Open(filepath.Join(prepared.Directory, obj.File))
		if err != nil {
			return err
		}
		uploadErr := store.UploadFile(ctx, obj.Key, file, obj.Size, obj.ContentType)
		file.Close()
		if uploadErr != nil {
			return uploadErr
		}
		output, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s3cfg.BucketName), Key: aws.String(obj.Key)})
		if err != nil {
			return err
		}
		digest, size, err := digestReader(output.Body, obj.Size)
		output.Body.Close()
		if err != nil {
			return err
		}
		if digest != obj.SHA256 || size != obj.Size {
			return fmt.Errorf("restored object checksum mismatch")
		}
	}
	if !finalize {
		return nil
	}
	if _, err = db.ExecContext(ctx, "UPDATE server_sessions SET revoked_at=now() WHERE revoked_at IS NULL"); err != nil {
		return err
	}
	if restoredSchema >= 12 {
		if _, err = db.ExecContext(ctx, `UPDATE backup_settings SET settings=jsonb_set(jsonb_set(jsonb_set(settings,'{settings,enabled}','false'),'{settings,passwordSet}','false'),'{secret}','{}')`); err != nil {
			return err
		}
	}
	if err = writeJSONFile(filepath.Join(staging, "restore-"+m.ID+"-"+fmt.Sprint(time.Now().UnixNano())+".json"), map[string]any{"id": m.ID, "completedAt": time.Now().UTC(), "objects": len(m.Objects)}); err != nil {
		return err
	}
	if err = os.Remove(marker); err != nil {
		return err
	}
	return syncDirectory(staging)
}

func durableMarker(name string, data []byte) error {
	file, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	return syncDirectory(filepath.Dir(name))
}
func syncDirectory(name string) error {
	directory, err := os.Open(name)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
