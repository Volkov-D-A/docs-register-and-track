package models

import (
	"time"

	"github.com/google/uuid"
)

// UserSubstitution описывает активное или запланированное замещение пользователя.
type UserSubstitution struct {
	SubstituteUserID uuid.UUID  `json:"-"`
	StartsAt         *time.Time `json:"startsAt,omitempty"`
	EndsAt           *time.Time `json:"endsAt,omitempty"`
	IsActive         bool       `json:"isActive"`
}

// UpdateUserSubstitutionRequest описывает запрос на назначение замещающего.
type UpdateUserSubstitutionRequest struct {
	PrincipalUserID  string `json:"principalUserId,omitempty"`
	SubstituteUserID string `json:"substituteUserId,omitempty"`
	StartsAt         string `json:"startsAt,omitempty"`
	EndsAt           string `json:"endsAt,omitempty"`
	IsActive         bool   `json:"isActive"`
}
