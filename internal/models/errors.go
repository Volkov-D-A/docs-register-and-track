package models

import "fmt"

// AppError — структурированная ошибка приложения с HTTP-кодом.
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error реализует интерфейс error, возвращая форматированное сообщение об ошибке.
func (e *AppError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}

// Предопределённые ошибки
var (
	ErrUnauthorized       = &AppError{Code: 401, Message: "требуется авторизация"}
	ErrInvalidCredentials = &AppError{Code: 401, Message: "неверный логин или пароль"}
	ErrUserNotActive      = &AppError{Code: 403, Message: "пользователь деактивирован"}
	ErrForbidden          = &AppError{Code: 403, Message: "недостаточно прав"}
	ErrWrongPassword      = &AppError{Code: 400, Message: "неверный текущий пароль"}
)

// NewBadRequest — ошибка 400 с кастомным сообщением.
func NewBadRequest(msg string) *AppError {
	return &AppError{Code: 400, Message: msg}
}

// NewUnauthorized — ошибка 401 с кастомным сообщением.
func NewUnauthorized(msg string) *AppError {
	return &AppError{Code: 401, Message: msg}
}

// NewForbidden — ошибка 403 с кастомным сообщением.
func NewForbidden(msg string) *AppError {
	return &AppError{Code: 403, Message: msg}
}

// NewNotFound — ошибка 404 с кастомным сообщением.
func NewNotFound(msg string) *AppError {
	return &AppError{Code: 404, Message: msg}
}
