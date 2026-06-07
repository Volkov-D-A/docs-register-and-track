package models

import (
	"time"

	"github.com/google/uuid"
)

// UserSubstitution описывает активное или запланированное замещение пользователя.
type UserSubstitution struct {
	ID               uuid.UUID  `json:"-"`
	PrincipalUserID  uuid.UUID  `json:"-"`
	SubstituteUserID uuid.UUID  `json:"-"`
	PrincipalName    string     `json:"principalName,omitempty"`
	SubstituteName   string     `json:"substituteName,omitempty"`
	StartsAt         *time.Time `json:"startsAt,omitempty"`
	EndsAt           *time.Time `json:"endsAt,omitempty"`
	IsActive         bool       `json:"isActive"`
	CreatedBy        *uuid.UUID `json:"-"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

// UpdateUserSubstitutionRequest описывает запрос на назначение замещающего.
type UpdateUserSubstitutionRequest struct {
	PrincipalUserID  string `json:"principalUserId,omitempty"`
	SubstituteUserID string `json:"substituteUserId,omitempty"`
	StartsAt         string `json:"startsAt,omitempty"`
	EndsAt           string `json:"endsAt,omitempty"`
	IsActive         bool   `json:"isActive"`
}
