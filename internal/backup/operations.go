package backup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

type operation struct {
	models.BackupOperation
	Actor          string         `json:"actor"`
	Target         StoredSettings `json:"target"`
	Copy           RemoteCopy     `json:"copy"`
	TokenHash      []byte         `json:"tokenHash"`
	ExpiresAt      time.Time      `json:"expiresAt"`
	VerificationID string         `json:"verificationId,omitempty"`
}

func (s *Service) operations() ([]operation, error) {
	entries, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return []operation{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := []operation{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".operation.json") {
			continue
		}
		op, err := s.operation(strings.TrimSuffix(entry.Name(), ".operation.json"))
		if err != nil {
			return nil, err
		}
		result = append(result, op)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result, nil
}

func (s *Service) operation(id string) (operation, error) {
	var op operation
	if copyFormat(id) != 3 {
		return op, fmt.Errorf("invalid operation identifier")
	}
	raw, err := os.ReadFile(filepath.Join(s.Directory, id+".operation.json"))
	if err != nil {
		return op, err
	}
	if err = json.Unmarshal(raw, &op); err != nil {
		return op, err
	}
	if op.ID != id || copyFormat(op.CopyID) == 0 {
		return op, fmt.Errorf("invalid operation journal")
	}
	return op, nil
}

func (s *Service) persistOperation(op *operation) error {
	op.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(op)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.Directory, ".operation-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	_, err = f.Write(raw)
	if err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	if err = os.Rename(name, filepath.Join(s.Directory, op.ID+".operation.json")); err != nil {
		return err
	}
	if err := syncDirectory(s.Directory); err != nil {
		return err
	}
	s.Events.Publish("operation:" + op.ID)
	s.Events.Publish("backups")
	return nil
}

func (s *Service) OperationStatus(id, token string) (models.BackupOperation, error) {
	op, err := s.operation(id)
	if err != nil {
		return models.BackupOperation{}, models.ErrForbidden
	}
	hash := sha256.Sum256([]byte(token))
	if token == "" || !time.Now().Before(op.ExpiresAt) || subtle.ConstantTimeCompare(hash[:], op.TokenHash) != 1 {
		return models.BackupOperation{}, models.ErrForbidden
	}
	return op.BackupOperation, nil
}

func (s *Service) StartOperation(ctx context.Context, kind string, req models.BackupOperationRequest, actor string) (models.BackupOperationStarted, error) {
	var out models.BackupOperationStarted
	if err := s.Ready(); err != nil {
		return out, err
	}
	if kind != "verify" && kind != "restore" && kind != "delete" {
		return out, fmt.Errorf("invalid operation")
	}
	if copyFormat(req.CopyID) == 0 {
		return out, fmt.Errorf("invalid copy identifier")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != "" {
		return out, fmt.Errorf("дождитесь текущего задания")
	}
	if err := s.noPendingWork(); err != nil {
		return out, err
	}
	op := operation{BackupOperation: models.BackupOperation{ID: uuid.NewString(), CopyID: req.CopyID, Kind: kind, State: "queued", CreatedAt: time.Now().UTC(), CanCancel: kind != "delete"}, Actor: actor, ExpiresAt: time.Now().UTC().Add(12 * time.Hour)}
	var err error
	if kind == "restore" {
		verified, e := s.operation(req.VerificationID)
		if e != nil {
			return out, fmt.Errorf("сначала проверьте выбранную копию")
		}
		if verified.Kind != "verify" || verified.State != "completed" || verified.CopyID != req.CopyID || time.Since(verified.UpdatedAt) > 24*time.Hour {
			return out, fmt.Errorf("требуется новая проверка копии")
		}
		op.Copy = verified.Copy
		op.Target = verified.Target
		op.VerificationID = verified.ID
	} else {
		op.Target, err = (SettingsRepository{s.DB}).Load(ctx)
		if err != nil {
			return out, err
		}
		password, e := s.Box.Decrypt(op.Target.Secret)
		if e != nil {
			return out, e
		}
		client, e := smb.Open(ctx, smb.Config(op.Target.Settings.SMB), password)
		if e != nil {
			return out, e
		}
		op.Copy, err = readCopyMarker(ctx, client, req.CopyID)
		if err != nil && kind == "delete" {
			op.Copy, err = readDeletion(ctx, client, req.CopyID)
		}
		client.Close()
		if err != nil {
			return out, err
		}
	}
	if kind == "delete" && op.Copy.Format != 3 {
		return out, fmt.Errorf("удаление v2 не поддерживается")
	}
	confirmationKind := kind
	if kind == "verify" {
		confirmationKind = "restore"
	} // Bind verification to the selected immutable set as well.
	if req.Confirmation != operationConfirmation(confirmationKind, op.Copy) {
		return out, fmt.Errorf("подтверждение должно содержать действие, ID и дату выбранной копии")
	}
	if err = os.MkdirAll(s.Directory, 0700); err != nil {
		return out, err
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return out, err
	}
	out.StatusToken = base64.RawURLEncoding.EncodeToString(secret)
	hash := sha256.Sum256([]byte(out.StatusToken))
	op.TokenHash = hash[:]
	if err = s.persistOperation(&op); err != nil {
		return models.BackupOperationStarted{}, err
	}
	out.Job = op.BackupOperation
	out.ExpiresAt = op.ExpiresAt
	return out, nil
}

func operationConfirmation(kind string, copy RemoteCopy) string {
	return fmt.Sprintf("%s %s %s %d %s", kind, copy.ID, copy.CreatedAt.UTC().Format(time.RFC3339Nano), copy.Size, copy.SHA256)
}

// Caller holds mu; this also rejects queued work between scheduler ticks.
func (s *Service) noPendingWork() error {
	jobs, err := s.jobs()
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if j.State == "queued" || j.State == "snapshotting" || j.State == "staged" || j.State == "transferring" || j.State == "verifying" {
			return fmt.Errorf("есть незавершённая резервная копия")
		}
	}
	entries, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".operation.json") {
			op, err := s.operation(strings.TrimSuffix(e.Name(), ".operation.json"))
			if err != nil {
				return err
			}
			if !operationFinished(op.State) {
				return fmt.Errorf("есть незавершённая операция с копией")
			}
		}
	}
	return nil
}

func operationFinished(state string) bool {
	switch state {
	case "completed", "failed", "cancelled", "interrupted", "rolled_back", "rollback_failed", "recovery_required":
		return true
	}
	return false
}

func (s *Service) CancelOperation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	op, err := s.operation(id)
	if err != nil {
		return err
	}
	if !op.CanCancel || operationFinished(op.State) {
		return fmt.Errorf("отмена на этой фазе запрещена")
	}
	if s.active == id && s.cancel != nil {
		s.cancel()
		return nil
	}
	if op.State != "queued" {
		return fmt.Errorf("операция сейчас не выполняется")
	}
	op.State = "cancelled"
	op.CanCancel = false
	return s.persistOperation(&op)
}

func (s *Service) runOperation(ctx context.Context) bool {
	if _, err := os.Stat(filepath.Join(s.Directory, "recovery-required")); err == nil || !os.IsNotExist(err) {
		return true
	}
	s.mu.Lock()
	if s.active != "" {
		s.mu.Unlock()
		return true
	}
	entries, err := os.ReadDir(s.Directory)
	if err != nil {
		s.mu.Unlock()
		return false
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".operation.json") {
			continue
		}
		op, e := s.operation(strings.TrimSuffix(entry.Name(), ".operation.json"))
		if e != nil {
			s.mu.Unlock()
			return true
		}
		if operationFinished(op.State) {
			continue
		}
		if op.State != "queued" {
			// Never silently retry a partially destructive operation after restart.
			op.State = "interrupted"
			op.CanCancel = false
			op.Error = "Операция прервана перезапуском; проверьте локальный журнал"
			s.persistOperation(&op)
			s.mu.Unlock()
			return true
		}
		s.active = op.ID
		work, cancel := context.WithTimeout(ctx, 6*time.Hour)
		s.cancel = cancel
		s.mu.Unlock()
		err = s.executeOperation(work, &op)
		s.mu.Lock()
		if err != nil {
			op.Error = err.Error()
			if !operationFinished(op.State) {
				op.State = "failed"
			}
			if work.Err() == context.Canceled && op.CanCancel {
				op.State = "cancelled"
			}
		}
		if _, markerErr := os.Stat(filepath.Join(s.Directory, "recovery-required")); markerErr == nil || !os.IsNotExist(markerErr) {
			if op.State != "rollback_failed" {
				op.State = "recovery_required"
			}
		}
		op.CanCancel = false
		s.persistOperation(&op)
		cancel()
		s.active = ""
		s.cancel = nil
		s.mu.Unlock()
		return true
	}
	s.mu.Unlock()
	return false
}

func (s *Service) executeOperation(ctx context.Context, op *operation) error {
	password, err := s.Box.Decrypt(op.Target.Secret)
	if err != nil {
		return err
	}
	client, err := smb.Open(ctx, smb.Config(op.Target.Settings.SMB), password)
	if err != nil {
		return err
	}
	defer client.Close()
	release, err := client.Lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	if op.Kind == "restore" {
		actual, err := readCopyMarker(ctx, client, op.CopyID)
		if err != nil {
			return err
		}
		if !sameCopy(actual, op.Copy) {
			return fmt.Errorf("source changed after verification")
		}
		return s.replaceOperation(ctx, op)
	}
	if op.Kind == "delete" {
		return s.deleteOperation(ctx, client, op)
	}
	if err = s.operationPhase(op, "downloading", true); err != nil {
		return err
	}
	if err := s.checkStagingSpace(); err != nil {
		return err
	}
	work := filepath.Join(s.Directory, op.ID)
	if err = os.Mkdir(work, 0700); err != nil {
		return err
	}
	archive, err := downloadSelected(ctx, client, op.Copy, work, s.MaxBytes/8)
	if err != nil {
		return err
	}
	if err = s.operationPhase(op, "verifying", true); err != nil {
		return err
	}
	prepared, err := PrepareRestore(ctx, s.PostgreSQL, archive, filepath.Join(work, "contents"), s.MaxBytes/8, s.Schema)
	if err != nil {
		return err
	}
	if prepared.Manifest.Format == 3 && prepared.Manifest.ID != op.Copy.ID {
		return fmt.Errorf("archive identity mismatch")
	}
	return s.operationPhase(op, "completed", false)
}

func (s *Service) operationPhase(op *operation, state string, cancellable bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	op.State = state
	op.CanCancel = cancellable
	return s.persistOperation(op)
}

func (s *Service) noPendingOperations() error {
	entries, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".operation.json") {
			op, err := s.operation(strings.TrimSuffix(e.Name(), ".operation.json"))
			if err != nil {
				return err
			}
			if !operationFinished(op.State) {
				return fmt.Errorf("есть незавершённая операция с копией")
			}
		}
	}
	return nil
}
