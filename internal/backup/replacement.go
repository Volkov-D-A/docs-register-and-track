package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// replaceOperation keeps the downloaded source and safety archive even on
// failure. PostgreSQL and S3 are restored sequentially, not atomically.
func (s *Service) replaceOperation(ctx context.Context, op *operation) error {
	if s.Replace == nil || s.Reload == nil {
		return fmt.Errorf("replacement coordinator unavailable")
	}
	if copyFormat(op.VerificationID) != 3 {
		return fmt.Errorf("invalid verification identifier")
	}
	if err := s.checkStagingSpace(); err != nil {
		return err
	}
	work := filepath.Join(s.Directory, op.ID)
	if err := os.Mkdir(work, 0700); err != nil {
		return err
	}
	archive := filepath.Join(s.Directory, op.VerificationID, op.CopyID+".tar.gz")
	digest, size, err := FileDigest(archive)
	if err != nil {
		return err
	}
	if digest != op.Copy.SHA256 || size != op.Copy.Size {
		return fmt.Errorf("staged source checksum mismatch")
	}
	if err = s.operationPhase(op, "verifying", true); err != nil {
		return err
	}
	prepared, err := PrepareRestore(ctx, s.PostgreSQL, archive, filepath.Join(work, "source"), s.MaxBytes/8, s.Schema)
	if err != nil {
		return err
	}
	return s.Replace(ctx, func(ctx context.Context) error {
		if err := s.operationPhase(op, "safety_snapshot", false); err != nil {
			return err
		}
		safety, err := s.safetySnapshot(ctx, work)
		if err != nil {
			return err
		}
		return s.coordinateReplacement(ctx, op, prepared, safety, s.installReplacement, s.Reload)
	})
}

func (s *Service) safetySnapshot(ctx context.Context, work string) (PreparedRestore, error) {
	stage := filepath.Join(work, "safety-source")
	if err := os.Mkdir(stage, 0700); err != nil {
		return PreparedRestore{}, err
	}
	m := Manifest{Format: 3, ID: uuid.NewString(), CreatedAt: time.Now().UTC(), Version: s.Version, Objects: []Object{}}
	var dirty bool
	if err := s.DB.QueryRowContext(ctx, "SELECT version,dirty FROM schema_migrations").Scan(&m.Schema, &dirty); err != nil {
		return PreparedRestore{}, err
	}
	if dirty {
		return PreparedRestore{}, fmt.Errorf("schema is dirty")
	}
	dump := filepath.Join(stage, "database.dump")
	if err := s.PostgreSQL.Dump(ctx, dump, s.MaxBytes/8); err != nil {
		return PreparedRestore{}, err
	}
	var err error
	m.DatabaseSHA256, m.DatabaseSize, err = FileDigest(dump)
	if err != nil {
		return PreparedRestore{}, err
	}
	if err = SnapshotObjects(ctx, s.S3, stage, &m, s.MaxBytes/8); err != nil {
		return PreparedRestore{}, err
	}
	if err = ValidateReferences(ctx, s.DB, m); err != nil {
		return PreparedRestore{}, err
	}
	if err = s.requireDedicatedTarget(ctx, m); err != nil {
		return PreparedRestore{}, err
	}
	archive := filepath.Join(work, "safety.tar.gz")
	if err = Pack(ctx, stage, archive, m); err != nil {
		return PreparedRestore{}, err
	}
	return PrepareRestore(ctx, s.PostgreSQL, archive, filepath.Join(work, "safety-verified"), s.MaxBytes/8, s.Schema)
}

func (s *Service) installReplacement(ctx context.Context, op *operation, prepared PreparedRestore, rollback bool) error {
	phase := func(state string) error {
		if rollback {
			state = "rollback_" + state
		}
		return s.operationPhase(op, state, false)
	}
	if err := phase("clearing_database"); err != nil {
		return err
	}
	// Only dedicated Docflow databases and buckets are supported. The database
	// itself, roles, other databases, and other buckets are never removed.
	if _, err := s.DB.ExecContext(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		return err
	}
	if err := phase("clearing_objects"); err != nil {
		return err
	}
	client, err := storage.NewS3Client(s.S3)
	if err != nil {
		return err
	}
	for {
		page, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(s.S3.BucketName), MaxKeys: aws.Int32(1000)})
		if err != nil {
			return err
		}
		if len(page.Contents) == 0 {
			break
		}
		for _, object := range page.Contents {
			if _, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.S3.BucketName), Key: object.Key}); err != nil {
				return err
			}
		}
	}
	if err = phase("restoring"); err != nil {
		return err
	}
	if err = RestorePrepared(ctx, s.PostgreSQL, s.S3, prepared, s.Directory, s.Schema, false); err != nil {
		return err
	}
	if err = phase("migrating"); err != nil {
		return err
	}
	db, err := database.Connect(s.PostgreSQL.Config)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.RunMigrations(database.DefaultMigrationsPath); err != nil {
		return err
	}
	if !rollback {
		if _, err = db.ExecContext(ctx, `UPDATE server_sessions SET revoked_at=now() WHERE revoked_at IS NULL; UPDATE backup_settings SET settings=jsonb_set(jsonb_set(jsonb_set(settings,'{settings,enabled}','false'),'{settings,passwordSet}','false'),'{secret}','{}')`); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) coordinateReplacement(ctx context.Context, op *operation, prepared, safety PreparedRestore, install func(context.Context, *operation, PreparedRestore, bool) error, reload func(context.Context) error) error {
	var err error
	if err = ctx.Err(); err != nil {
		return err
	}
	// Cancellation and the irreversible phase transition share mu.
	s.mu.Lock()
	if err = ctx.Err(); err == nil {
		op.State = "replacing"
		op.CanCancel = false
		err = s.persistOperation(op)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	marker := filepath.Join(s.Directory, "recovery-required")
	if err = durableMarker(marker, []byte(op.ID+"\n")); err != nil {
		return err
	}
	// Client cancellation is no longer applicable; the process may still be
	// stopped, in which case the durable marker keeps startup closed.
	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), 6*time.Hour)
	defer cancel()
	err = install(bounded, op, prepared, false)
	if err == nil {
		err = reload(bounded)
	}
	if err != nil {
		original := err
		if e := s.operationPhase(op, "rolling_back", false); e != nil {
			return fmt.Errorf("replacement failed and rollback journal unavailable: %w", e)
		}
		rollback, stop := context.WithTimeout(context.Background(), 6*time.Hour)
		defer stop()
		if e := install(rollback, op, safety, true); e != nil {
			op.State = "rollback_failed"
			return fmt.Errorf("replacement failed: %v; rollback failed: %w", original, e)
		}
		if e := reload(rollback); e != nil {
			op.State = "rollback_failed"
			return fmt.Errorf("rollback reload failed: %w", e)
		}
		if e := s.operationPhase(op, "rolled_back", false); e != nil {
			return e
		}
		if e := os.Remove(marker); e != nil {
			return e
		}
		if e := syncDirectory(s.Directory); e != nil {
			return e
		}
		return fmt.Errorf("замена не выполнена; исходное состояние восстановлено: %w", original)
	}
	if err = s.operationPhase(op, "finalizing", false); err != nil {
		return err
	}
	if err = os.Remove(marker); err != nil {
		return err
	}
	if err = syncDirectory(s.Directory); err != nil {
		return err
	}
	op.State = "completed"
	return nil
}
