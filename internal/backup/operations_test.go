package backup

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testOperation(t *testing.T) (*Service, *operation, string) {
	t.Helper()
	s := &Service{Directory: t.TempDir()}
	token := "random-status-capability"
	hash := sha256.Sum256([]byte(token))
	op := &operation{BackupOperation: models.BackupOperation{ID: uuid.NewString(), CopyID: uuid.NewString(), Kind: "restore", State: "queued", CanCancel: true}, TokenHash: hash[:], ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, s.persistOperation(op))
	return s, op, token
}

func TestStatusCapabilitySurvivesRestartAndIsScoped(t *testing.T) {
	s, op, token := testOperation(t)
	restarted := &Service{Directory: s.Directory}
	_, err := restarted.OperationStatus(op.ID, token)
	require.NoError(t, err)
	for _, value := range []string{"", "wrong"} {
		_, err = restarted.OperationStatus(op.ID, value)
		require.ErrorIs(t, err, models.ErrForbidden)
	}
	_, err = restarted.OperationStatus(uuid.NewString(), token)
	require.ErrorIs(t, err, models.ErrForbidden)
	op.ExpiresAt = time.Now().Add(-time.Second)
	require.NoError(t, s.persistOperation(op))
	_, err = restarted.OperationStatus(op.ID, token)
	require.ErrorIs(t, err, models.ErrForbidden)
	raw, err := os.ReadFile(filepath.Join(s.Directory, op.ID+".operation.json"))
	require.NoError(t, err)
	require.NotContains(t, string(raw), token)
	info, err := os.Stat(filepath.Join(s.Directory, op.ID+".operation.json"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestReplacementRollbackAndDurableGate(t *testing.T) {
	for _, mode := range []string{"success", "install_failure", "rollback_failure", "reload_failure"} {
		t.Run(mode, func(t *testing.T) {
			s, op, _ := testOperation(t)
			calls := []bool{}
			reloads := 0
			install := func(ctx context.Context, got *operation, _ PreparedRestore, rollback bool) error {
				require.FileExists(t, filepath.Join(s.Directory, "recovery-required"))
				require.False(t, got.CanCancel)
				require.Error(t, s.CancelOperation(got.ID))
				calls = append(calls, rollback)
				if mode == "rollback_failure" || (mode == "install_failure" && !rollback) {
					return errors.New("injected failure")
				}
				return nil
			}
			reload := func(context.Context) error {
				reloads++
				if mode == "reload_failure" && reloads == 1 {
					return errors.New("reload failed")
				}
				return nil
			}
			err := s.coordinateReplacement(context.Background(), op, PreparedRestore{}, PreparedRestore{}, install, reload)
			if mode == "success" {
				require.NoError(t, err)
				require.Equal(t, []bool{false}, calls)
				require.Equal(t, "completed", op.State)
			} else {
				require.Error(t, err)
				require.Equal(t, []bool{false, true}, calls)
			}
			if mode == "rollback_failure" {
				require.FileExists(t, filepath.Join(s.Directory, "recovery-required"))
				require.Equal(t, "rollback_failed", op.State)
			} else {
				require.NoFileExists(t, filepath.Join(s.Directory, "recovery-required"))
			}
		})
	}
}

func TestCancelBeforeReplacementNeverInstalls(t *testing.T) {
	s, op, _ := testOperation(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.coordinateReplacement(ctx, op, PreparedRestore{}, PreparedRestore{}, func(context.Context, *operation, PreparedRestore, bool) error {
		t.Fatal("installed cancelled restore")
		return nil
	}, func(context.Context) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
	require.NoFileExists(t, filepath.Join(s.Directory, "recovery-required"))
	require.NoError(t, s.CancelOperation(op.ID))
	persisted, err := s.operation(op.ID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", persisted.State)
}

func TestRestoreConfirmationBindsCopyAndDate(t *testing.T) {
	s, verified, _ := testOperation(t)
	verified.Kind = "verify"
	verified.State = "completed"
	verified.Copy = RemoteCopy{ID: verified.CopyID, Format: 3, CreatedAt: time.Now().UTC()}
	require.NoError(t, s.persistOperation(verified))
	box, err := NewSecretBox(make([]byte, 32))
	require.NoError(t, err)
	s.Box = box
	s.MaxBytes = 1024
	req := models.BackupOperationRequest{CopyID: verified.CopyID, VerificationID: verified.ID, Confirmation: "restore " + verified.CopyID}
	_, err = s.StartOperation(context.Background(), "restore", req, "admin")
	require.ErrorContains(t, err, "подтверждение")
	req.Confirmation = operationConfirmation("restore", verified.Copy)
	started, err := s.StartOperation(context.Background(), "restore", req, "admin")
	require.NoError(t, err)
	require.NotEmpty(t, started.StatusToken)
	_, err = s.StartOperation(context.Background(), "restore", req, "admin")
	require.Error(t, err)
}

func TestPersistentMarkerPreventsSchedulerAndQueuedOperations(t *testing.T) {
	s, op, _ := testOperation(t)
	require.NoError(t, os.WriteFile(filepath.Join(s.Directory, "recovery-required"), []byte(op.ID), 0600))
	require.ErrorContains(t, s.Ready(), "незавершённое восстановление")
	require.True(t, s.runOperation(context.Background()))
	unchanged, err := s.operation(op.ID)
	require.NoError(t, err)
	require.Equal(t, "queued", unchanged.State)
}

func TestStagingSpaceRejectsExhaustedBudget(t *testing.T) {
	s := &Service{Directory: t.TempDir(), MaxBytes: 1024}
	require.NoError(t, os.WriteFile(filepath.Join(s.Directory, "retained-archive"), make([]byte, 600), 0600))
	require.ErrorContains(t, s.checkStagingSpace(), "недостаточно места")
}
