package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRetryWaitsForFinalJournalAndHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	box, err := NewSecretBox(make([]byte, 32))
	require.NoError(t, err)
	service := &Service{DB: db, Box: box, Directory: t.TempDir(), MaxBytes: 1024}
	settings, err := json.Marshal(StoredSettings{Settings: DefaultSettings()})
	require.NoError(t, err)
	mock.ExpectQuery("SELECT settings FROM backup_settings").WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(settings))
	// Leave a window after final local persistence but before history completes.
	mock.ExpectExec("INSERT INTO backup_jobs").WillDelayFor(500 * time.Millisecond).WillReturnResult(sqlmock.NewResult(1, 1))
	job := Job{ID: uuid.NewString(), State: "staged", ArchiveSHA256: "staged", CreatedAt: time.Now(), Target: StoredSettings{Secret: EncryptedSecret{Data: "invalid"}}}
	require.NoError(t, service.persist(&job))
	done := make(chan struct{})
	go func() { defer close(done); service.tick(context.Background()) }()
	defer func() { <-done }()
	require.Eventually(t, func() bool {
		jobs, err := service.jobs()
		return err == nil && len(jobs) == 1 && jobs[0].State == "staged" && jobs[0].Error != ""
	}, time.Second, time.Millisecond)
	require.ErrorContains(t, service.Retry(context.Background(), job.ID), "дождитесь текущего задания")
	<-done
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRunInvalidatesInterruptedSnapshotAndRetainsStagedArchive(t *testing.T) {
	directory := t.TempDir()
	service := &Service{Directory: directory}
	snapshot := Job{ID: uuid.NewString(), State: "snapshotting", CreatedAt: time.Now()}
	staged := Job{ID: uuid.NewString(), State: "staged", CreatedAt: time.Now()}
	require.NoError(t, service.persist(&snapshot))
	require.NoError(t, service.persist(&staged))
	snapshotDir := filepath.Join(directory, snapshot.ID)
	require.NoError(t, os.Mkdir(snapshotDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(snapshotDir, "database.dump"), []byte("incomplete"), 0600))
	snapshotArchive := filepath.Join(directory, snapshot.ID+".tar.gz")
	stagedArchive := filepath.Join(directory, staged.ID+".tar.gz")
	require.NoError(t, os.WriteFile(snapshotArchive, []byte("incomplete"), 0600))
	require.NoError(t, os.WriteFile(stagedArchive, []byte("complete"), 0600))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service.Run(ctx)

	jobs, err := service.jobs()
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	byID := map[string]Job{jobs[0].ID: jobs[0], jobs[1].ID: jobs[1]}
	require.Equal(t, "interrupted", byID[snapshot.ID].State)
	require.Equal(t, "staged", byID[staged.ID].State)
	require.NoDirExists(t, snapshotDir)
	require.NoFileExists(t, snapshotArchive)
	require.FileExists(t, stagedArchive)
}
