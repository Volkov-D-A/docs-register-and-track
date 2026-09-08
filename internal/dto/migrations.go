package dto

// MigrationStatus содержит информацию о текущем состоянии миграций БД.
type MigrationStatus struct {
	CurrentVersion         uint `json:"currentVersion"`
	Dirty                  bool `json:"dirty"`
	AvailableCount         int  `json:"availableCount"`
	LatestAvailableVersion uint `json:"latestAvailableVersion"`
	UpToDate               bool `json:"upToDate"`
	SchemaTooNew           bool `json:"schemaTooNew"`
	Compatible             bool `json:"compatible"`
}
