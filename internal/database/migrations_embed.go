package database

import "embed"

const (
	// DefaultMigrationsPath is the canonical migration path used by the app.
	// At runtime this path is served from the embedded filesystem, so packaged
	// builds do not depend on the source tree being present near the executable.
	DefaultMigrationsPath = "internal/database/migrations"

	embeddedMigrationsPath = "migrations"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS
