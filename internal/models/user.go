package models

import (
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/personalname"

	"github.com/google/uuid"
)

// User представляет собой сущность пользователя системы.
type User struct {
	ID                     uuid.UUID   `json:"-"`
	Login                  string      `json:"login"`
	PasswordHash           string      `json:"-"`
	LastName               string      `json:"lastName"`
	FirstName              string      `json:"firstName"`
	Patronymic             string      `json:"patronymic"`
	NoPatronymic           bool        `json:"noPatronymic"`
	IsDocumentParticipant  bool        `json:"isDocumentParticipant"`
	IsActive               bool        `json:"isActive"`
	FailedLoginAttempts    int         `json:"failedLoginAttempts"`
	PasswordChangedAt      *time.Time  `json:"passwordChangedAt,omitempty"`
	PasswordChangeRequired bool        `json:"passwordChangeRequired"`
	SystemPermissions      []string    `json:"systemPermissions"`
	DepartmentID           *uuid.UUID  `json:"-"`
	Department             *Department `json:"department,omitempty"`
}

// CreateUserRequest описывает полезную нагрузку для создания нового пользователя.
type CreateUserRequest struct {
	Login                  string `json:"login"`
	Password               string `json:"password"`
	LastName               string `json:"lastName"`
	FirstName              string `json:"firstName"`
	Patronymic             string `json:"patronymic"`
	NoPatronymic           bool   `json:"noPatronymic"`
	DepartmentID           string `json:"departmentId"`
	IsDocumentParticipant  bool   `json:"isDocumentParticipant"`
	PasswordChangeRequired bool   `json:"-"`
}

// UpdateUserRequest описывает полезную нагрузку для обновления данных существующего пользователя администратором.
type UpdateUserRequest struct {
	ID                    string `json:"id"`
	Login                 string `json:"login"`
	LastName              string `json:"lastName"`
	FirstName             string `json:"firstName"`
	Patronymic            string `json:"patronymic"`
	NoPatronymic          bool   `json:"noPatronymic"`
	IsActive              bool   `json:"isActive"`
	DepartmentID          string `json:"departmentId"`
	IsDocumentParticipant bool   `json:"isDocumentParticipant"`
}

// UpdateProfileRequest описывает полезную нагрузку для обновления профиля самим пользователем.
type UpdateProfileRequest struct {
	Login        string `json:"login"`
	LastName     string `json:"lastName"`
	FirstName    string `json:"firstName"`
	Patronymic   string `json:"patronymic"`
	NoPatronymic bool   `json:"noPatronymic"`
}

// InitialSetupRequest creates the first administrator with the fixed login admin.
type InitialSetupRequest struct {
	Password     string `json:"password"`
	LastName     string `json:"lastName"`
	FirstName    string `json:"firstName"`
	Patronymic   string `json:"patronymic"`
	NoPatronymic bool   `json:"noPatronymic"`
}

func (u User) FullName() string {
	return personalname.Display(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
}

func (u CreateUserRequest) FullName() string {
	return personalname.Display(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
}

func (u UpdateUserRequest) FullName() string {
	return personalname.Display(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
}

func (u *CreateUserRequest) NormalizeName() error {
	last, first, patronymic, err := personalname.Normalize(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
	if err != nil {
		return NewBadRequestWrapped(err.Error(), err)
	}
	u.LastName, u.FirstName, u.Patronymic = last, first, patronymic
	return nil
}

func (u *UpdateUserRequest) NormalizeName() error {
	last, first, patronymic, err := personalname.Normalize(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
	if err != nil {
		return NewBadRequestWrapped(err.Error(), err)
	}
	u.LastName, u.FirstName, u.Patronymic = last, first, patronymic
	return nil
}

func (u *UpdateProfileRequest) NormalizeName() error {
	last, first, patronymic, err := personalname.Normalize(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
	if err != nil {
		return NewBadRequestWrapped(err.Error(), err)
	}
	u.LastName, u.FirstName, u.Patronymic = last, first, patronymic
	return nil
}

func (u *InitialSetupRequest) NormalizeName() error {
	last, first, patronymic, err := personalname.Normalize(u.LastName, u.FirstName, u.Patronymic, u.NoPatronymic)
	if err != nil {
		return NewBadRequestWrapped(err.Error(), err)
	}
	u.LastName, u.FirstName, u.Patronymic = last, first, patronymic
	return nil
}
