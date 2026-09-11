package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/backup/smb"
	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/liveevents"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

type Job struct {
	ID            string         `json:"id"`
	State         string         `json:"state"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	Error         string         `json:"error,omitempty"`
	Attempts      int            `json:"attempts"`
	ArchiveSize   int64          `json:"archiveSize"`
	ArchiveSHA256 string         `json:"archiveSha256"`
	Actor         string         `json:"actor"`
	ScheduledFor  time.Time      `json:"scheduledFor,omitempty"`
	Target        StoredSettings `json:"target"`
}
type JobView = models.BackupJob

func (j Job) View() JobView {
	return JobView{ID: j.ID, State: j.State, CreatedAt: j.CreatedAt, UpdatedAt: j.UpdatedAt, Error: j.Error, Attempts: j.Attempts, ArchiveSize: j.ArchiveSize}
}

type Service struct {
	Events     *liveevents.Bus
	DB         *sql.DB
	PostgreSQL PostgreSQL
	S3         config.S3Config
	Directory  string
	MaxBytes   int64
	Version    string
	Box        *SecretBox
	// Snapshot executes fn under the application's maintenance barrier.
	Snapshot func(context.Context, func(context.Context) error) error
	// Replace executes the destructive coordinator under full maintenance.
	Replace func(context.Context, func(context.Context) error) error
	Reload  func(context.Context) error
	Schema  int
	mu      sync.Mutex
	active  string
	cancel  context.CancelFunc
	done    chan struct{}
}

func (s *Service) Busy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active != "" || s.noPendingWork() != nil
}

func (s *Service) Ready() error {
	if s.Directory != "" {
		if _, err := os.Stat(filepath.Join(s.Directory, "recovery-required")); err == nil {
			return fmt.Errorf("незавершённое восстановление; обычные операции заблокированы")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if s.Box == nil {
		return fmt.Errorf("ключ настроек резервирования не подключён")
	}
	if s.Directory == "" || !filepath.IsAbs(s.Directory) {
		return fmt.Errorf("постоянный каталог резервирования не настроен")
	}
	if s.MaxBytes <= 0 {
		return fmt.Errorf("не задан лимит резервирования")
	}
	return nil
}
func (s *Service) Settings(ctx context.Context) (Settings, error) {
	v, err := (SettingsRepository{s.DB}).Load(ctx)
	return v.Settings, err
}
func (s *Service) SaveSettings(ctx context.Context, req SettingsUpdate, actor string) error {
	if err := s.Ready(); err != nil {
		return err
	}
	if err := req.Settings.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old, err := (SettingsRepository{s.DB}).Load(ctx)
	if err != nil {
		return err
	}
	if req.ClearPassword && req.Password != "" {
		return fmt.Errorf("choose a new password or clear it")
	}
	old.Settings = req.Settings
	if req.ClearPassword {
		old.Secret = EncryptedSecret{}
	}
	if req.Password != "" {
		old.Secret, err = s.Box.Encrypt(req.Password)
		if err != nil {
			return err
		}
	}
	old.Settings.PasswordSet = old.Secret.Data != ""
	if old.Settings.PasswordSet {
		if _, err = s.Box.Decrypt(old.Secret); err != nil {
			return err
		}
	}
	if old.Settings.Enabled && !old.Settings.PasswordSet {
		return fmt.Errorf("SMB password is required")
	}
	return (SettingsRepository{s.DB}).Save(ctx, old, actor)
}
func (s *Service) Check(ctx context.Context) error {
	if err := s.Ready(); err != nil {
		return err
	}
	cfg, err := (SettingsRepository{s.DB}).Load(ctx)
	if err != nil {
		return err
	}
	password, err := s.Box.Decrypt(cfg.Secret)
	if err != nil {
		return err
	}
	client, err := smb.Open(ctx, smb.Config(cfg.Settings.SMB), password)
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Check(ctx)
}
func (s *Service) Jobs() ([]JobView, error) {
	jobs, err := s.jobs()
	if err != nil {
		return nil, err
	}
	views := make([]JobView, 0, len(jobs))
	for _, job := range jobs {
		views = append(views, job.View())
	}
	ops, err := s.operations()
	if err != nil {
		return nil, err
	}
	for _, op := range ops {
		views = append(views, JobView{ID: op.ID, Kind: op.Kind, CopyID: op.CopyID, State: op.State, CreatedAt: op.CreatedAt, UpdatedAt: op.UpdatedAt, Error: op.Error, ArchiveSize: op.Copy.Size})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].CreatedAt.After(views[j].CreatedAt) })
	return views, nil
}
func (s *Service) jobs() ([]Job, error) {
	entries, err := os.ReadDir(s.Directory)
	if os.IsNotExist(err) {
		return []Job{}, nil
	}
	if err != nil {
		return nil, err
	}
	jobs := []Job{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".job.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.Directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		var job Job
		if err = json.Unmarshal(raw, &job); err != nil {
			return nil, fmt.Errorf("invalid local backup journal")
		}
		if _, err = uuid.Parse(job.ID); err != nil || entry.Name() != job.ID+".job.json" {
			return nil, fmt.Errorf("invalid backup identifier")
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.After(jobs[j].CreatedAt) })
	return jobs, nil
}
func (s *Service) persist(job *Job) error {
	job.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(job)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Directory, ".job-")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(raw)
	}
	if err == nil {
		err = tmp.Sync()
	}
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(name, filepath.Join(s.Directory, job.ID+".job.json")); err != nil {
		return err
	}
	dir, err := os.Open(s.Directory)
	if err != nil {
		return err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return err
	}
	s.Events.Publish("backups")
	return nil
}
func (s *Service) Start(ctx context.Context, actor string, due time.Time) (JobView, error) {
	if err := s.Ready(); err != nil {
		return JobView{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != "" {
		return JobView{}, fmt.Errorf("резервирование уже выполняется")
	}
	if err := s.noPendingOperations(); err != nil {
		return JobView{}, err
	}
	cfg, err := (SettingsRepository{s.DB}).Load(ctx)
	if err != nil {
		return JobView{}, err
	}
	if err = cfg.Settings.Validate(); err != nil {
		return JobView{}, err
	}
	if _, err = s.Box.Decrypt(cfg.Secret); err != nil {
		return JobView{}, err
	}
	if err = os.MkdirAll(s.Directory, 0700); err != nil {
		return JobView{}, err
	}
	jobs, err := s.jobs()
	if err != nil {
		return JobView{}, err
	}
	for _, job := range jobs {
		if !due.IsZero() && job.ScheduledFor.Equal(due) {
			return job.View(), nil
		}
		if job.State == "queued" || job.State == "snapshotting" || job.State == "staged" || job.State == "transferring" || job.State == "verifying" {
			return JobView{}, fmt.Errorf("есть незавершённая резервная копия")
		}
	}
	var used int64
	err = filepath.WalkDir(s.Directory, func(_ string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			info, err := e.Info()
			if err != nil {
				return err
			}
			used += info.Size()
		}
		return nil
	})
	if err != nil {
		return JobView{}, err
	}
	if used > s.MaxBytes/2 {
		return JobView{}, fmt.Errorf("недостаточно места в лимите staging; проверьте неотправленные копии")
	}
	job := Job{ID: uuid.NewString(), State: "queued", CreatedAt: time.Now().UTC(), Actor: actor, ScheduledFor: due, Target: cfg}
	if err = s.persist(&job); err != nil {
		return JobView{}, err
	}
	return job.View(), nil
}
func (s *Service) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != id || s.cancel == nil {
		return fmt.Errorf("задание сейчас не выполняется")
	}
	s.cancel()
	return nil
}
func (s *Service) Run(ctx context.Context) {
	s.done = make(chan struct{})
	defer close(s.done)
	// Only incomplete local snapshots are invalidated on restart. Complete staged
	// archives are retained for retransmission using the captured destination.
	if s.Directory != "" {
		jobs, err := s.jobs()
		if err == nil {
			for _, job := range jobs {
				if job.State == "snapshotting" {
					job.State = "interrupted"
					job.Error = "Снимок прерван перезапуском"
					os.RemoveAll(filepath.Join(s.Directory, job.ID))
					os.Remove(filepath.Join(s.Directory, job.ID+".tar.gz"))
					s.persist(&job)
				}
			}
		}
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if !s.runOperation(ctx) {
			s.tick(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (s *Service) tick(ctx context.Context) {
	if s.Ready() != nil {
		return
	}
	settings, err := s.Settings(ctx)
	if err != nil {
		return
	}
	if due, ok := settings.Due(time.Now()); ok {
		s.Start(ctx, "schedule", due)
	}
	s.mu.Lock()
	jobs, err := s.jobs()
	if err != nil {
		s.mu.Unlock()
		return
	}
	for _, job := range jobs {
		if job.State != "queued" && job.State != "staged" && job.State != "transferring" && job.State != "verifying" {
			continue
		}
		if job.Attempts >= 5 {
			continue
		}
		if job.Attempts > 0 && time.Since(job.UpdatedAt) < time.Duration(1<<job.Attempts)*time.Minute {
			continue
		}
		s.active = job.ID
		work, cancel := context.WithTimeout(ctx, 6*time.Hour)
		s.cancel = cancel
		s.mu.Unlock()
		err = s.execute(work, &job)
		workErr := work.Err()
		cancel()
		if err != nil && job.State != "completed" {
			// Detailed errors stay in the local protected journal. Never include settings.
			job.Error = err.Error()
			if job.ArchiveSHA256 != "" {
				job.State = "staged"
			} else {
				job.State = "failed"
				os.RemoveAll(filepath.Join(s.Directory, job.ID))
				os.Remove(filepath.Join(s.Directory, job.ID+".tar.gz"))
			}
			if errors.Is(workErr, context.Canceled) && ctx.Err() == nil {
				job.State = "cancelled"
			}
			s.persist(&job)
		}
		// History written after the snapshot barrier; it is not part of that snapshot.
		raw, _ := json.Marshal(job.View())
		bounded, stop := context.WithTimeout(context.Background(), 10*time.Second)
		s.DB.ExecContext(bounded, `INSERT INTO backup_jobs(id,state) VALUES($1,$2) ON CONFLICT(id) DO UPDATE SET state=excluded.state,updated_at=now()`, job.ID, raw)
		stop()
		s.mu.Lock()
		s.active = ""
		s.cancel = nil
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
}
func (s *Service) execute(ctx context.Context, job *Job) error {
	stage := filepath.Join(s.Directory, job.ID)
	archive := filepath.Join(s.Directory, job.ID+".tar.gz")
	if job.State == "queued" {
		job.State = "snapshotting"
		if err := s.persist(job); err != nil {
			return err
		}
		if err := os.Mkdir(stage, 0700); err != nil {
			return err
		}
		m := Manifest{Format: 3, ID: job.ID, CreatedAt: time.Now().UTC(), Version: s.Version, Objects: []Object{}}
		if s.Snapshot == nil {
			return fmt.Errorf("snapshot coordinator unavailable")
		}
		err := s.Snapshot(ctx, func(ctx context.Context) error {
			var dirty bool
			if err := s.DB.QueryRowContext(ctx, "SELECT version,dirty FROM schema_migrations").Scan(&m.Schema, &dirty); err != nil {
				return err
			}
			if dirty {
				return fmt.Errorf("schema is dirty")
			}
			file := filepath.Join(stage, "database.dump")
			if err := s.PostgreSQL.Dump(ctx, file, s.MaxBytes/2); err != nil {
				return err
			}
			var err error
			m.DatabaseSHA256, m.DatabaseSize, err = FileDigest(file)
			if err != nil {
				return err
			}
			if err = SnapshotObjects(ctx, s.S3, stage, &m, s.MaxBytes/2); err != nil {
				return err
			}
			return ValidateReferences(ctx, s.DB, m)
		})
		if err != nil {
			return err
		}
		if err = Pack(ctx, stage, archive, m); err != nil {
			return err
		}
		job.ArchiveSHA256, job.ArchiveSize, err = FileDigest(archive)
		if err != nil {
			return err
		}
		job.State = "staged"
		if err = s.persist(job); err != nil {
			return err
		}
		if err = os.RemoveAll(stage); err != nil {
			return err
		}
	}
	job.Attempts++
	job.State = "transferring"
	job.Error = ""
	if err := s.persist(job); err != nil {
		return err
	}
	password, err := s.Box.Decrypt(job.Target.Secret)
	if err != nil {
		return err
	}
	client, err := smb.Open(ctx, smb.Config(job.Target.Settings.SMB), password)
	if err != nil {
		return err
	}
	defer client.Close()
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	partial := "." + job.ID + "-" + uuid.NewString() + ".partial"
	if err = client.Write(ctx, partial, file); err != nil {
		return err
	}
	defer client.Remove(context.WithoutCancel(ctx), partial)
	job.State = "verifying"
	if err = s.persist(job); err != nil {
		return err
	}
	if err = client.Verify(ctx, partial, job.ArchiveSize, job.ArchiveSHA256); err != nil {
		return err
	}
	if err = client.Rename(ctx, partial, job.ID+".tar.gz"); err != nil {
		return err
	}
	// A small final marker is the only indication that the archive is complete.
	marker, _ := json.Marshal(RemoteCopy{Format: 3, ID: job.ID, Size: job.ArchiveSize, SHA256: job.ArchiveSHA256, CreatedAt: job.CreatedAt})
	markerTmp := partial + ".manifest"
	if err = client.Write(ctx, markerTmp, strings.NewReader(string(marker))); err != nil {
		return err
	}
	defer client.Remove(context.WithoutCancel(ctx), markerTmp)
	if err = client.Rename(ctx, markerTmp, job.ID+".manifest.json"); err != nil {
		return err
	}
	job.State = "completed"
	job.Target.Secret = EncryptedSecret{}
	if err = s.persist(job); err != nil {
		return err
	}
	if err = os.Remove(archive); err != nil {
		return err
	}
	// Retention errors do not invalidate the newly verified copy.
	if err = s.retention(ctx, client, job.Target.Settings, job.Actor); err != nil {
		job.Error = "Копия создана; очистка старых копий не выполнена"
		return s.persist(job)
	}
	return nil
}

func (s *Service) retention(ctx context.Context, client *smb.Client, settings Settings, actor string) error {
	release, err := client.Lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	copies, err := RemoteCopies(ctx, client)
	if err != nil {
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -settings.RetentionDays)
	for i := len(copies) - 1; i >= settings.KeepCopies; i-- {
		copy := copies[i]
		if copy.CreatedAt.IsZero() || !copy.CreatedAt.Before(cutoff) {
			continue
		}
		op := operation{BackupOperation: models.BackupOperation{ID: uuid.NewString(), CopyID: copy.ID, Kind: "delete", State: "deleting", CreatedAt: time.Now().UTC()}, Actor: actor, Copy: copy, Target: StoredSettings{Settings: settings}}
		if err = s.persistOperation(&op); err != nil {
			return err
		}
		err = s.deleteRemote(ctx, client, copy, settings.KeepCopies)
		if err == nil {
			err = s.markDeleted(copy.ID)
		}
		if err != nil {
			op.State = "failed"
			op.Error = err.Error()
		} else {
			op.State = "completed"
		}
		if journalErr := s.persistOperation(&op); journalErr != nil {
			return journalErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Retry keeps the archived snapshot and destination; refreshed credentials may
// be used only when the saved destination still matches the original job.
func (s *Service) Retry(ctx context.Context, id string) error {
	if err := s.Ready(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != "" {
		return fmt.Errorf("дождитесь текущего задания")
	}
	jobs, err := s.jobs()
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if job.ID != id {
			continue
		}
		if job.ArchiveSHA256 == "" || (job.State != "staged" && job.State != "cancelled") {
			return fmt.Errorf("нет готового локального архива для повторной отправки")
		}
		settings, err := (SettingsRepository{s.DB}).Load(ctx)
		if err != nil {
			return err
		}
		a, b := job.Target.Settings.SMB, settings.Settings.SMB
		if a.Host != b.Host || a.Port != b.Port || a.Share != b.Share || a.Directory != b.Directory {
			return fmt.Errorf("сохранённое SMB-направление отличается от направления задания")
		}
		if _, err = s.Box.Decrypt(settings.Secret); err != nil {
			return err
		}
		job.Target = settings
		job.Attempts = 0
		job.Error = ""
		job.State = "staged"
		return s.persist(&job)
	}
	return fmt.Errorf("задание не найдено")
}
