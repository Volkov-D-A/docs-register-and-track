package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConfigConnectionString(t *testing.T) {
	dbCfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpassword",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpassword dbname=testdb sslmode=disable"
	assert.Equal(t, expected, dbCfg.ConnectionString())
}

func TestGetDefaultConfigPath(t *testing.T) {
	path := GetDefaultConfigPath()
	expected := filepath.Join("config", "config.json")
	assert.Equal(t, expected, path)
}

func TestLoadConfig(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		// Создаем временный файл конфигурации
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")
		configContent := `{
			"database": {
				"host": "db_host",
				"port": 5433,
				"user": "db_user",
				"password": "db_password",
				"dbname": "db_name",
				"sslmode": "require"
			}
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Загружаем конфигурацию
		cfg, err := Load(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Проверяем значения
		assert.Equal(t, "db_host", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "db_user", cfg.Database.User)
		assert.Equal(t, "db_password", cfg.Database.Password)
		assert.Equal(t, "db_name", cfg.Database.DBName)
		assert.Equal(t, "require", cfg.Database.SSLMode)
	})

	t.Run("file not found", func(t *testing.T) {
		cfg, err := Load("non_existent_config.json")
		require.Error(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("invalid json", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid_config.json")
		err := os.WriteFile(configPath, []byte("{invalid json}"), 0644)
		require.NoError(t, err)

		cfg, err := Load(configPath)
		require.Error(t, err)
		assert.Nil(t, cfg)
	})
}
