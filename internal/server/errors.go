package server

import (
	"log/slog"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

type apiErrorResponse struct {
	Code      string `json:"code"`
	Error     string `json:"error"`
	RequestID string `json:"requestId"`
	Status    string `json:"status,omitempty"`
}

// Only server-owned messages belong here. Unknown errors never supply public text.
var publicAPIMessages = map[string]string{
	"backup_unavailable":            "Резервирование не настроено на сервере.",
	"backup_settings_failed":        "Не удалось сохранить настройки резервирования. Проверьте поля и подключение ключа на сервере.",
	"backup_check_failed":           "Не удалось проверить SMB. Проверьте адрес, учётные данные и права на выбранную папку.",
	"backup_start_failed":           "Не удалось создать задание. Проверьте настройки, свободное место и незавершённые копии.",
	"backup_retry_failed":           "Не удалось повторить отправку. Проверьте наличие локального архива и совпадение SMB-направления.",
	"backup_cancel_failed":          "Задание уже завершено или ещё не выполняется.",
	"maintenance":                   "Сервис временно недоступен: обслуживание базы данных.",
	"dependency_not_ready":          "Сервис временно недоступен.",
	"migration_apply_failed":        "Не удалось применить миграции базы данных.",
	"migration_rollback_failed":     "Не удалось откатить миграцию базы данных.",
	"migration_lock_busy":           "Изменение схемы сейчас недоступно. Повторите попытку позже.",
	"worker_stop_failed":            "Не удалось подготовить сервер к изменению схемы.",
	"invalid_rollback_confirmation": "Подтвердите резервное копирование и последствия отката.",
	"authentication_required":       "Требуется авторизация.",
	"invalid_credentials":           "Неверный логин или пароль.",
	"authentication_rate_limited":   "Слишком много неудачных попыток. Повторите позже.",
}

func publicAPIMessage(status int, code string, err error) string {
	if status < 500 {
		if message, ok := models.PublicErrorMessage(err); ok {
			return message
		}
	}
	if message, ok := publicAPIMessages[code]; ok {
		return message
	}
	switch {
	case status >= 500:
		return "Произошла внутренняя ошибка сервера."
	case status == http.StatusConflict:
		return "Операция не может быть выполнена из-за конфликта."
	case status == http.StatusUnauthorized:
		return "Требуется авторизация."
	case status == http.StatusForbidden:
		return "Недостаточно прав для выполнения операции."
	case status == http.StatusNotFound:
		return "Запрошенный объект не найден."
	default:
		return "Проверьте параметры запроса."
	}
}

func ensureRequestID(w http.ResponseWriter) string {
	if id := w.Header().Get(requestIDHeader); id != "" {
		return id
	}
	id := uuid.NewString()
	w.Header().Set(requestIDHeader, id)
	return id
}

func writeAPIError(w http.ResponseWriter, status int, code string, err error) {
	writeAPIStateError(w, status, code, "", err)
}

func writeAPIStateError(w http.ResponseWriter, status int, code, state string, err error) {
	id := ensureRequestID(w)
	logAPIError(w, status, code, err)
	writeJSON(w, status, apiErrorResponse{Code: code, Error: publicAPIMessage(status, code, err), RequestID: id, Status: state})
}

func logAPIError(w http.ResponseWriter, status int, code string, err error) {
	// AppError.Error omits Internal. Preserve the chain as structured log data.
	slog.Warn("API operation failed", "request_id", ensureRequestID(w), "code", code, "status", status,
		"error", err, "error_causes", models.ErrorCauses(err))
}
