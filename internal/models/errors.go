package models

import (
	"errors"
	"fmt"
)

// AppError — структурированная ошибка приложения с HTTP-кодом.
type AppError struct {
	Code       int    `json:"status"`
	Kind       string `json:"code"`
	Message    string `json:"message"`
	Internal   error  `json:"-"`
	Production bool   `json:"-"`
}

// Error реализует интерфейс error, возвращая форматированное сообщение об ошибке.
func (e *AppError) Error() string {
	if e.Kind != "" {
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Internal
}

func (e *AppError) SafeMessage() string {
	if e.Message != "" {
		return e.Message
	}
	return "произошла ошибка"
}

func (e *AppError) SafeKind() string {
	if e.Kind != "" {
		return e.Kind
	}
	return "INTERNAL_ERROR"
}

func (e *AppError) StatusCode() int {
	if e.Code != 0 {
		return e.Code
	}
	return 500
}

// Предопределённые ошибки
var (
	ErrUnauthorized       = &AppError{Code: 401, Kind: "UNAUTHORIZED", Message: "требуется авторизация", Production: true}
	ErrInvalidCredentials = &AppError{Code: 401, Kind: "INVALID_CREDENTIALS", Message: "неверный логин или пароль", Production: true}
	ErrUserNotActive      = &AppError{Code: 403, Kind: "USER_INACTIVE", Message: "пользователь деактивирован", Production: true}
	ErrUserLocked         = &AppError{Code: 403, Kind: "USER_LOCKED", Message: "учетная запись заблокирована после 5 неверных попыток входа; обратитесь к администратору", Production: true}
	ErrForbidden          = &AppError{Code: 403, Kind: "FORBIDDEN", Message: "недостаточно прав", Production: true}
	ErrWrongPassword      = &AppError{Code: 400, Kind: "VALIDATION_ERROR", Message: "неверный текущий пароль", Production: true}
)

// NewBadRequest — ошибка 400 с кастомным сообщением.
func NewBadRequest(msg string) *AppError {
	return &AppError{Code: 400, Kind: "VALIDATION_ERROR", Message: msg, Production: true}
}

// NewUnauthorized — ошибка 401 с кастомным сообщением.
func NewUnauthorized(msg string) *AppError {
	return &AppError{Code: 401, Kind: "UNAUTHORIZED", Message: msg, Production: true}
}

// NewForbidden — ошибка 403 с кастомным сообщением.
func NewForbidden(msg string) *AppError {
	return &AppError{Code: 403, Kind: "FORBIDDEN", Message: msg, Production: true}
}

// NewNotFound — ошибка 404 с кастомным сообщением.
func NewNotFound(msg string) *AppError {
	return &AppError{Code: 404, Kind: "NOT_FOUND", Message: msg, Production: true}
}

func NewConflict(msg string) *AppError {
	return &AppError{Code: 409, Kind: "CONFLICT", Message: msg, Production: true}
}

func NewIdempotencyConflict(msg string) *AppError {
	return &AppError{Code: 409, Kind: "IDEMPOTENCY_CONFLICT", Message: msg, Production: true}
}

func NewInternal(msg string, err error) *AppError {
	return &AppError{Code: 500, Kind: "INTERNAL_ERROR", Message: msg, Internal: err}
}

func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
