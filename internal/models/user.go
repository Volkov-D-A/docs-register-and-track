package models

import (
	"time"

	"github.com/google/uuid"
)

// User представляет собой сущность пользователя системы.
type User struct {
	ID           uuid.UUID   `json:"-"`
	Login        string      `json:"login"`
	PasswordHash string      `json:"-"`
	FullName     string      `json:"fullName"`
	IsActive     bool        `json:"isActive"`
	Roles        []string    `json:"roles"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	DepartmentID *uuid.UUID  `json:"-"`
	Department   *Department `json:"department,omitempty"`
}

// UserRole представляет собой связь пользователя с его ролью в системе.
type UserRole struct {
	ID        uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"-"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateUserRequest описывает полезную нагрузку для создания нового пользователя.
type CreateUserRequest struct {
	Login        string   `json:"login"`
	Password     string   `json:"password"`
	FullName     string   `json:"fullName"`
	Roles        []string `json:"roles"`
	DepartmentID string   `json:"departmentId"`
}

// UpdateUserRequest описывает полезную нагрузку для обновления данных существующего пользователя администратором.
type UpdateUserRequest struct {
	ID           string   `json:"id"`
	Login        string   `json:"login"`
	FullName     string   `json:"fullName"`
	IsActive     bool     `json:"isActive"`
	Roles        []string `json:"roles"`
	DepartmentID string   `json:"departmentId"`
}

// UpdateProfileRequest описывает полезную нагрузку для обновления профиля самим пользователем.
type UpdateProfileRequest struct {
	Login    string `json:"login"`
	FullName string `json:"fullName"`
}

// LoginRequest описывает учетные данные пользователя для входа в систему.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// ChangePasswordRequest описывает запрос пользователя на смену пароля.
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
