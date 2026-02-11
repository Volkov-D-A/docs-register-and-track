package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"-"`
	IDStr        string    `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"fullName"`
	IsActive     bool      `json:"isActive"`
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// FillIDStr заполняет строковое представление ID
func (u *User) FillIDStr() {
	u.IDStr = u.ID.String()
}

type UserRole struct {
	ID        uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateUserRequest struct {
	Login    string   `json:"login"`
	Password string   `json:"password"`
	FullName string   `json:"fullName"`
	Roles    []string `json:"roles"`
}

type UpdateUserRequest struct {
	ID       string   `json:"id"`
	Login    string   `json:"login"`
	FullName string   `json:"fullName"`
	IsActive bool     `json:"isActive"`
	Roles    []string `json:"roles"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
