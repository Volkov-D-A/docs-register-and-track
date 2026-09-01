package dto

// SystemSchemaStatus describes whether the database schema is safe for this server.
type SystemSchemaStatus struct {
	CurrentVersion  uint `json:"currentVersion"`
	RequiredVersion uint `json:"requiredVersion"`
	Compatible      bool `json:"compatible"`
	Dirty           bool `json:"dirty"`
}

// SystemStatus is the public, non-sensitive server readiness contract.
type SystemStatus struct {
	Status        string             `json:"status"`
	Code          string             `json:"code"`
	APIVersion    string             `json:"apiVersion"`
	ServerVersion string             `json:"serverVersion"`
	Maintenance   bool               `json:"maintenance"`
	Schema        SystemSchemaStatus `json:"schema"`
}

// CompatibilityResult describes whether a desktop build can use this server.
type CompatibilityResult struct {
	Compatible           bool   `json:"compatible"`
	Code                 string `json:"code"`
	APIVersion           string `json:"apiVersion"`
	ServerVersion        string `json:"serverVersion"`
	MinimumClientVersion string `json:"minimumClientVersion"`
	MaximumClientVersion string `json:"maximumClientVersion"`
}

// BootstrapStatus is returned to the frontend before authentication starts.
type BootstrapStatus struct {
	State         string               `json:"state"`
	Code          string               `json:"code"`
	Message       string               `json:"message"`
	Compatibility *CompatibilityResult `json:"compatibility,omitempty"`
	System        *SystemStatus        `json:"system,omitempty"`
}
