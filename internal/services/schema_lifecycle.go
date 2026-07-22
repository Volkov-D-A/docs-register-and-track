package services

// SchemaLifecycle coordinates application components whose operation depends
// on the database schema being fully up to date.
type SchemaLifecycle interface {
	ReconcileSchema()
	PrepareRollback() error
	CompleteRollback(success bool)
	CheckReady() error
}

// ConfigureSchemaLifecycle wires the composition-level schema coordinator
// without exposing infrastructure setters as Wails service methods.
func ConfigureSchemaLifecycle(auth *AuthService, settings *SettingsService, lifecycle SchemaLifecycle) {
	auth.schemaLifecycle = lifecycle
	settings.schemaLifecycle = lifecycle
}
