package services

import (
	"strconv"
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

const (
	DefaultAttachmentSizeMB = 15
	MaximumAttachmentSizeMB = 1024
)

// SettingsService interprets server-owned settings for business operations.
type SettingsService struct{ settingsStore ports.SettingsStore }

func NewSettingsService(store ports.SettingsStore) *SettingsService {
	return &SettingsService{settingsStore: store}
}

// GetMaxFileSize возвращает максимальный допустимый размер файла в байтах.
func (s *SettingsService) GetMaxFileSize() (int64, error) {
	setting, err := s.getSetting("max_file_size_mb")
	if err != nil {
		return DefaultAttachmentSizeMB * 1024 * 1024, nil
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return DefaultAttachmentSizeMB * 1024 * 1024, nil
	}
	mb, err := strconv.Atoi(setting.Value)
	if err != nil || mb < 1 || mb > MaximumAttachmentSizeMB {
		return DefaultAttachmentSizeMB * 1024 * 1024, nil
	}
	return int64(mb) * 1024 * 1024, nil
}

// GetAllowedFileTypes возвращает список разрешенных расширений файлов.
func (s *SettingsService) GetAllowedFileTypes() ([]string, error) {
	setting, err := s.getSetting("allowed_file_types")
	if err != nil {
		return []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, nil
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, nil
	}
	types := strings.Split(setting.Value, ",")
	result := make([]string, 0, len(types))
	for i, t := range types {
		types[i] = strings.TrimSpace(strings.ToLower(t))
		if types[i] != "" {
			result = append(result, types[i])
		}
	}
	return result, nil
}

// IsAssignmentCompletionAttachmentsEnabled возвращает признак доступности загрузки файлов при завершении поручения.
func (s *SettingsService) IsAssignmentCompletionAttachmentsEnabled() bool {
	setting, err := s.getSetting("assignment_completion_attachments_enabled")
	if err != nil || setting == nil || setting.Value == "" {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(setting.Value)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func (s *SettingsService) getSetting(key string) (*models.SystemSetting, error) {
	return s.settingsStore.Get(key)
}
