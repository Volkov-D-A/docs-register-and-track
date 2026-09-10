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
	ID          string    `json:"id"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Error       string    `json:"error,omitempty"`
	Attempts    int       `json:"attempts"`
	ArchiveSize int64     `json:"archiveSize"`
}
