package models

// SystemSetting представляет собой системную настройку.
type SystemSetting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"-"`
}

// RollbackMigrationRequest подтверждает production guardrails перед откатом миграции.
type RollbackMigrationRequest struct {
	BackupCompleted      bool   `json:"backupCompleted"`
	BackupReference      string `json:"backupReference"`
	AcknowledgedDataLoss bool   `json:"acknowledgedDataLoss"`
	Confirmation         string `json:"confirmation"`
	Password             string `json:"password"`
}
