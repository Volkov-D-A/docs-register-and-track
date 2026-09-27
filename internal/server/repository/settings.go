package repository

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
)

// SettingsRepository предоставляет методы для работы с системными настройками в БД.
type SettingsRepository struct {
	db     *database.DB
	outbox *OutboxRepository
}

func (r *SettingsRepository) SetOutbox(outbox *OutboxRepository) { r.outbox = outbox }

// NewSettingsRepository создает новый экземпляр SettingsRepository.
func NewSettingsRepository(db *database.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get возвращает значение системной настройки по её ключу.
func (r *SettingsRepository) Get(key string) (*models.SystemSetting, error) {
	var s models.SystemSetting
	err := r.db.QueryRow("SELECT key, value, description FROM system_settings WHERE key = $1", key).
		Scan(&s.Key, &s.Value, &s.Description)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetAll возвращает список всех системных настроек.
func (r *SettingsRepository) GetAll() ([]models.SystemSetting, error) {
	rows, err := r.db.Query("SELECT key, value FROM system_settings ORDER BY key")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]models.SystemSetting, 0)
	for rows.Next() {
		var s models.SystemSetting
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return settings, nil
}

// UpdateWithOutbox records a setting change and its audit event atomically.
func (r *SettingsRepository) UpdateWithOutbox(key, value string, effects []models.OutboxEvent) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO system_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value); err != nil {
		return err
	}
	if err := enqueueOutboxEffects(r.outbox, tx, effects); err != nil {
		return err
	}
	return tx.Commit()
}
