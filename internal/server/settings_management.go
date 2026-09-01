package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
)

type settingsManagementStore interface {
	Get(string) (*models.SystemSetting, error)
	GetAll() ([]models.SystemSetting, error)
	UpdateWithOutbox(string, string, []models.OutboxEvent) error
}

type settingUpdateRequest struct {
	Value string `json:"value"`
}

func (api *managementAPI) listSettings(w http.ResponseWriter, _ *http.Request) {
	settings, err := api.settings.GetAll()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (api *managementAPI) getSetting(w http.ResponseWriter, r *http.Request) {
	key, ok := settingKey(w, r.PathValue("key"))
	if !ok {
		return
	}
	setting, err := api.settings.Get(key)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, setting)
}

func (api *managementAPI) updateSetting(w http.ResponseWriter, r *http.Request) {
	key, ok := settingKey(w, r.PathValue("key"))
	if !ok {
		return
	}
	var req settingUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if err := validateSystemSettingValue(key, req.Value); err != nil {
		writeUserError(w, err)
		return
	}

	current, err := api.settings.Get(key)
	if err == nil && current != nil && current.Value == req.Value {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	details := fmt.Sprintf("Изменена настройка %s: %s", settingAuditLabel(key, current), req.Value)
	effect, err := userAuditEffect(
		authenticatedFromContext(r.Context()).User,
		"setting:"+key+":update:"+uuid.NewString(),
		"SETTINGS_UPDATE",
		details,
	)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.settings.UpdateWithOutbox(key, req.Value, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func settingKey(w http.ResponseWriter, raw string) (string, bool) {
	key := strings.TrimSpace(raw)
	if key == "" {
		writeUserError(w, models.NewBadRequest("ключ настройки обязателен"))
		return "", false
	}
	return key, true
}

func validateSystemSettingValue(key, value string) error {
	if key == "max_file_size_mb" {
		megabytes, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || megabytes < 1 || megabytes > services.MaximumAttachmentSizeMB {
			return models.NewBadRequest(fmt.Sprintf("Максимальный размер файла должен быть целым числом от 1 до %d МБ", services.MaximumAttachmentSizeMB))
		}
		return nil
	}
	if key != "password_lifetime_days" {
		return nil
	}
	days, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || days < 0 {
		return models.NewBadRequest("Срок жизни пароля должен быть целым числом от 0 дней")
	}
	return nil
}

func settingAuditLabel(key string, current *models.SystemSetting) string {
	switch key {
	case "organization_name":
		return "Название организации"
	case "organization_short_name":
		return "Краткое название организации"
	case "max_file_size_mb":
		return "Максимальный размер файла"
	case "allowed_file_types":
		return "Разрешенные типы файлов"
	case "assignment_completion_attachments_enabled":
		return "Файлы при завершении поручения"
	case "password_lifetime_days":
		return "Срок жизни пароля"
	}
	if current != nil && strings.TrimSpace(current.Description) != "" {
		return current.Description
	}
	return fmt.Sprintf("«%s»", key)
}
