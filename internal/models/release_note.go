package models

import "time"

// ReleaseNote contains the embedded current version and its local view state.
type ReleaseNote struct {
	Version    string              `json:"version"`
	ReleasedAt time.Time           `json:"releasedAt"`
	Changes    []ReleaseNoteChange `json:"changes"`
	IsViewed   bool                `json:"isViewed"`
}

// ReleaseNoteChange is an item in the embedded release's ordered change list.
type ReleaseNoteChange struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}
