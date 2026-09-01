package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const (
	maxTechnicalLogBatch      = 50
	maxTechnicalLogMessage    = 4096
	maxTechnicalLogAttributes = 32
	maxTechnicalLogKey        = 64
	maxTechnicalLogValue      = 2048
)

func (api *managementAPI) ingestTechnicalLogs(w http.ResponseWriter, r *http.Request) {
	var batch models.TechnicalLogBatch
	if err := decodeJSONLimit(r, &batch, 256<<10); err != nil {
		writeUserError(w, models.NewBadRequestWrapped("некорректный пакет технических логов", err))
		return
	}
	if len(batch.Events) == 0 || len(batch.Events) > maxTechnicalLogBatch {
		writeUserError(w, models.NewBadRequest("пакет технических логов должен содержать от 1 до 50 событий"))
		return
	}
	user := authenticatedFromContext(r.Context()).User
	for _, event := range batch.Events {
		if err := validateTechnicalLogEvent(event); err != nil {
			writeUserError(w, err)
			return
		}
	}
	for _, event := range batch.Events {
		attrs := []slog.Attr{
			slog.String("source", "desktop"),
			slog.String("app_user_id", user.ID.String()),
		}
		if !event.Timestamp.IsZero() {
			attrs = append(attrs, slog.Time("desktop_timestamp", event.Timestamp.UTC()))
		}
		for key, value := range event.Attributes {
			if sensitiveTechnicalLogKey(key) || key == "app_user_id" || key == "app_user_name" {
				continue
			}
			attrs = append(attrs, slog.String("desktop_"+key, value))
		}
		slog.LogAttrs(r.Context(), technicalLogLevel(event.Level), event.Message, attrs...)
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateTechnicalLogEvent(event models.TechnicalLogEvent) error {
	message := strings.TrimSpace(event.Message)
	if message == "" || len(message) > maxTechnicalLogMessage || !utf8.ValidString(message) {
		return models.NewBadRequest("сообщение технического лога указано некорректно")
	}
	if len(event.Attributes) > maxTechnicalLogAttributes {
		return models.NewBadRequest("слишком много атрибутов технического лога")
	}
	if !event.Timestamp.IsZero() && (event.Timestamp.Before(time.Now().Add(-24*time.Hour)) || event.Timestamp.After(time.Now().Add(5*time.Minute))) {
		return models.NewBadRequest("время технического лога находится вне допустимого диапазона")
	}
	for key, value := range event.Attributes {
		if strings.TrimSpace(key) == "" || len(key) > maxTechnicalLogKey || len(value) > maxTechnicalLogValue || !utf8.ValidString(key) || !utf8.ValidString(value) {
			return models.NewBadRequest("атрибут технического лога указан некорректно")
		}
	}
	return nil
}

func sensitiveTechnicalLogKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "app_user_id" || key == "app_user_name" {
		return true
	}
	for _, fragment := range []string{"password", "passwd", "secret", "token", "authorization", "cookie", "credential"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func technicalLogLevel(level string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR", "FATAL":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
