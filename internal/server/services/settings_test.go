package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupServerSettingsService(t *testing.T) (*SettingsService, *mocks.SettingsStore) {
	t.Helper()
	store := mocks.NewSettingsStore(t)
	return NewSettingsService(store), store
}
func TestSettingsService_GetMaxFileSize(t *testing.T) {
	// Получение максимально допустимого размера загружаемых файлов в байтах
	t.Run("from settings", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "25"}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(25*1024*1024), size)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "max_file_size_mb").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(15*1024*1024), size)
	})

	t.Run("default on empty setting", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: " "}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(15*1024*1024), size)
	})

	t.Run("default on invalid setting", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "large"}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(15*1024*1024), size)
	})

	t.Run("default on out of range setting", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Key: "max_file_size_mb", Value: "2048"}, nil).Once()
		size, err := svc.GetMaxFileSize()
		require.NoError(t, err)
		assert.Equal(t, int64(DefaultAttachmentSizeMB*1024*1024), size)
	})
}

func TestSettingsService_GetAllowedFileTypes(t *testing.T) {
	// Получение списка разрешенных расширений загружаемых файлов
	t.Run("from settings", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ".pdf, .DOC, .txt"}, nil).Once()
		types, err := svc.GetAllowedFileTypes()
		require.NoError(t, err)
		assert.Equal(t, []string{".pdf", ".doc", ".txt"}, types)
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "allowed_file_types").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		types, err := svc.GetAllowedFileTypes()
		require.NoError(t, err)
		assert.Equal(t, []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, types)
	})

	t.Run("empty setting returns default list", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "allowed_file_types").Return(&models.SystemSetting{Key: "allowed_file_types", Value: ""}, nil).Once()
		types, err := svc.GetAllowedFileTypes()
		require.NoError(t, err)
		assert.Equal(t, []string{".pdf", ".doc", ".docx", ".odt", ".xls", ".xlsx", ".ods"}, types)
	})
}

func TestSettingsService_IsAssignmentCompletionAttachmentsEnabled(t *testing.T) {
	t.Run("from settings enabled", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{
			Key:   "assignment_completion_attachments_enabled",
			Value: "true",
		}, nil).Once()
		assert.True(t, svc.IsAssignmentCompletionAttachmentsEnabled())
	})

	t.Run("from settings disabled", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "assignment_completion_attachments_enabled").Return(&models.SystemSetting{
			Key:   "assignment_completion_attachments_enabled",
			Value: "false",
		}, nil).Once()
		assert.False(t, svc.IsAssignmentCompletionAttachmentsEnabled())
	})

	t.Run("default on error", func(t *testing.T) {
		svc, repo := setupServerSettingsService(t)
		repo.On("Get", "assignment_completion_attachments_enabled").Return((*models.SystemSetting)(nil), assert.AnError).Once()
		assert.False(t, svc.IsAssignmentCompletionAttachmentsEnabled())
	})
}

func TestSettingsServiceAttachmentSizeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		value  string
		wantMB int64
	}{
		{"0", DefaultAttachmentSizeMB}, {"-1", DefaultAttachmentSizeMB},
		{"1", 1}, {"1024", 1024}, {"1025", DefaultAttachmentSizeMB},
	} {
		t.Run(tc.value, func(t *testing.T) {
			svc, store := setupServerSettingsService(t)
			store.On("Get", "max_file_size_mb").Return(&models.SystemSetting{Value: tc.value}, nil).Once()
			size, err := svc.GetMaxFileSize()
			require.NoError(t, err)
			assert.Equal(t, tc.wantMB*1024*1024, size)
		})
	}
}
