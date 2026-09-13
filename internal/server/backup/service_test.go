package backup

import (
	"context"
	"encoding/json"
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
