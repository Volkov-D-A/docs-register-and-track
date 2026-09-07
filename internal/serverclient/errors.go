package serverclient

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

func validRequestID(value string) string {
	id, err := uuid.Parse(value)
	if err != nil {
		return ""
	}
	return id.String()
}
func responseRequestID(resp *http.Response) string {
	return validRequestID(resp.Header.Get("X-Request-ID"))
}

func decodeAuthError(resp *http.Response) (result error) {
	var body struct {
		Code      string `json:"code"`
		Error     string `json:"error"`
		RequestID string `json:"requestId"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body)
	id := responseRequestID(resp)
	if id == "" {
		id = validRequestID(body.RequestID)
	}
	defer func() { result = models.WithRequestID(result, id) }()
	// A proxy or older server may still return an unsafe body. Never expose its
	// 5xx text, or let a misleading body turn a server failure into a session error.
	if resp.StatusCode >= 500 {
		message, kind := "Произошла внутренняя ошибка сервера.", "INTERNAL_ERROR"
		if resp.StatusCode == 503 && body.Code == "maintenance" {
			message, kind = "Сервис временно недоступен: обслуживание базы данных.", "MAINTENANCE"
		}
		return &models.AppError{Code: resp.StatusCode, Kind: kind, Message: message}
	}
	switch body.Code {
	case "invalid_credentials":
		return models.ErrInvalidCredentials
	case "user_locked":
		return models.ErrUserLocked
	case "user_inactive":
		return models.ErrUserNotActive
	case "password_change_required":
		return models.ErrPasswordChangeRequired
	case "wrong_password":
		return models.ErrWrongPassword
	case "invalid_request", "validation_error":
		return models.NewBadRequest(body.Error)
	case "password_change_not_required", "conflict":
		return models.NewConflict(body.Error)
	case "authentication_required", "session_invalid":
		return models.ErrUnauthorized
	case "forbidden":
		return models.ErrForbidden
	case "not_found":
		return models.NewNotFound(body.Error)
	default:
		if body.Error == "" {
			body.Error = "Не удалось выполнить запрос к серверу."
		}
		kind := strings.ToUpper(body.Code)
		if kind == "" {
			kind = "UNKNOWN_ERROR"
		}
		return &models.AppError{Code: resp.StatusCode, Kind: kind, Message: body.Error, Production: resp.StatusCode >= 400 && resp.StatusCode < 500}
	}
}
