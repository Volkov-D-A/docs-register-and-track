package models

import "time"

type SMBDestination struct {
	Port      int    `json:"port,omitempty"`
	Host      string `json:"host"`
	Share     string `json:"share"`
	Directory string `json:"directory"`
	User      string `json:"user"`
	Domain    string `json:"domain"`
}
type BackupSettings struct {
	SMB           SMBDestination `json:"smb"`
	Enabled       bool           `json:"enabled"`
	Time          string         `json:"time"`
	Timezone      string         `json:"timezone"`
	Weekdays      []int          `json:"weekdays"`
	RetentionDays int            `json:"retentionDays"`
	KeepCopies    int            `json:"keepCopies"`
	PasswordSet   bool           `json:"passwordSet"`
}
type BackupSettingsUpdate struct {
	Settings      BackupSettings `json:"settings"`
	Password      string         `json:"password"`
	ClearPassword bool           `json:"clearPassword"`
}
type BackupSettingsResponse struct {
	NextRun  string         `json:"nextRun"`
	Settings BackupSettings `json:"settings"`
	Issue    string         `json:"issue"`
}
type BackupJob struct {
	Kind        string    `json:"kind"`
	CopyID      string    `json:"copyId"`
	ID          string    `json:"id"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Error       string    `json:"error,omitempty"`
	Attempts    int       `json:"attempts"`
	ArchiveSize int64     `json:"archiveSize"`
}

// BackupCopy describes the remote set, independently of local job history.
type BackupCopy struct {
	RestoreConfirmation string    `json:"restoreConfirmation"`
	DeleteConfirmation  string    `json:"deleteConfirmation"`
	ID                  string    `json:"id"`
	Format              int       `json:"format"`
	CreatedAt           time.Time `json:"createdAt"`
	Size                int64     `json:"size"`
	SHA256              string    `json:"sha256"`
	Verification        string    `json:"verification"`
	Issue               string    `json:"issue"`
	CanDelete           bool      `json:"canDelete"`
}

type BackupOperationRequest struct {
	CopyID         string `json:"copyId"`
	VerificationID string `json:"verificationId"`
	Confirmation   string `json:"confirmation"`
}

type BackupOperation struct {
	ID        string    `json:"id"`
	CopyID    string    `json:"copyId"`
	Kind      string    `json:"kind"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Error     string    `json:"error"`
	CanCancel bool      `json:"canCancel"`
}

type BackupOperationStarted struct {
	Job         BackupOperation `json:"job"`
	StatusToken string          `json:"statusToken"`
	ExpiresAt   time.Time       `json:"expiresAt"`
}
