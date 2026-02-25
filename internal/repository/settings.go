package repository

import (
	"docflow/internal/database"
	"docflow/internal/models"
)

// SettingsRepository предоставляет методы для работы с системными настройками в БД.
type SettingsRepository struct {
	db *database.DB
}

// NewSettingsRepository создает новый экземпляр SettingsRepository.
func NewSettingsRepository(db *database.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get возвращает значение системной настройки по её ключу.
func (r *SettingsRepository) Get(key string) (*models.SystemSetting, error) {
	var s models.SystemSetting
	err := r.db.QueryRow("SELECT key, value, description, updated_at FROM system_settings WHERE key = $1", key).
		Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetAll возвращает список всех системных настроек.
func (r *SettingsRepository) GetAll() ([]models.SystemSetting, error) {
	rows, err := r.db.Query("SELECT key, value, description, updated_at FROM system_settings ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]models.SystemSetting, 0)
	for rows.Next() {
		var s models.SystemSetting
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return settings, nil
}

// Update обновляет значение существующей системной настройки.
func (r *SettingsRepository) Update(key, value string) error {
	_, err := r.db.Exec("UPDATE system_settings SET value = $1, updated_at = NOW() WHERE key = $2", value, key)
	return err
}
