package models

import (
	"time"

	"github.com/google/uuid"
)

// ReleaseNote описывает релиз приложения и его метаданные.
type ReleaseNote struct {
	ID         uuid.UUID           `json:"id"`
	Version    string              `json:"version"`
	ReleasedAt time.Time           `json:"releasedAt"`
	IsCurrent  bool                `json:"isCurrent"`
	CreatedAt  time.Time           `json:"createdAt"`
	Changes    []ReleaseNoteChange `json:"changes"`
	IsViewed   bool                `json:"isViewed"`
}

// ReleaseNoteChange описывает отдельный пункт изменений релиза.
type ReleaseNoteChange struct {
	ID            uuid.UUID `json:"id"`
	ReleaseNoteID uuid.UUID `json:"releaseNoteId"`
	SortOrder     int       `json:"sortOrder"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
}

// ReleaseNoteChangeInput описывает изменение релиза при создании новой версии.
type ReleaseNoteChangeInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
