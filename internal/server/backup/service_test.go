package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJobViewKeepsRetryCounterServerSide(t *testing.T) {
	job := Job{ID: uuid.NewString(), State: "staged", Attempts: 4, ArchiveSize: 1024}
	stored, err := json.Marshal(job)
	require.NoError(t, err)
	require.Contains(t, string(stored), `"attempts":4`)
	response, err := json.Marshal(job.View())
	require.NoError(t, err)
	require.NotContains(t, string(response), `"attempts"`)
	require.Contains(t, string(response), `"archiveSize":1024`)
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
