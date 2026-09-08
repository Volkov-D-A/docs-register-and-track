package models

import "fmt"

// MigrationCompatibilityError reports a schema state that the current binary
// must not operate against.
type MigrationCompatibilityError struct {
	CurrentVersion         uint
	LatestAvailableVersion uint
	Dirty                  bool
	SchemaTooNew           bool
}

func (e *MigrationCompatibilityError) Error() string {
	if e.SchemaTooNew {
		return fmt.Sprintf("database schema version %d is newer than embedded migrations %d", e.CurrentVersion, e.LatestAvailableVersion)
	}
	if e.Dirty {
		return fmt.Sprintf("database schema version %d is dirty", e.CurrentVersion)
	}
	return "database schema is incompatible with this binary"
}
