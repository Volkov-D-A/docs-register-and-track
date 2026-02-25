package models

import "time"

// SystemSetting представляет собой системную настройку.
type SystemSetting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
