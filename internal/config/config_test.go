package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseConfigConnectionString(t *testing.T) {
	// Проверка формирования строки подключения (ConnectionString)
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

func TestMinioConfigGetSecretAccessKey(t *testing.T) {
	t.Run("plain secret", func(t *testing.T) {
		cfg := MinioConfig{SecretAccessKey: "plain-secret"}

		assert.Equal(t, "plain-secret", cfg.GetSecretAccessKey())
	})

	t.Run("encrypted secret", func(t *testing.T) {
		encrypted, err := EncryptPassword("encrypted-secret")
		require.NoError(t, err)

		cfg := MinioConfig{SecretAccessKey: encrypted}

		assert.Equal(t, "encrypted-secret", cfg.GetSecretAccessKey())
	})

	t.Run("invalid encrypted secret falls back to raw value", func(t *testing.T) {
		cfg := MinioConfig{SecretAccessKey: "ENC:not-valid-base64!!!"}

		assert.Equal(t, "ENC:not-valid-base64!!!", cfg.GetSecretAccessKey())
	})
}

func TestGetDefaultConfigPath(t *testing.T) {
	t.Run("env override", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "production.json")
		t.Setenv(ConfigPathEnv, configPath)

		assert.Equal(t, configPath, GetDefaultConfigPath())
	})

	t.Run("blank env override is ignored", func(t *testing.T) {
		t.Setenv(ConfigPathEnv, "   ")

		assert.NotEqual(t, "   ", GetDefaultConfigPath())
	})

	t.Run("executable relative before cwd", func(t *testing.T) {
		executablePath := filepath.Join(string(filepath.Separator), "opt", "docflow", "docflow")
		workingDir := filepath.Join(string(filepath.Separator), "tmp")

		candidates := buildDefaultConfigPathCandidates(executablePath, workingDir)

		require.Len(t, candidates, 3)
		assert.Equal(t, filepath.Join(string(filepath.Separator), "opt", "docflow", "config", "config.json"), candidates[0])
		assert.Equal(t, filepath.Join(string(filepath.Separator), "tmp", "config", "config.json"), candidates[1])
		assert.Equal(t, filepath.Join("config", "config.json"), candidates[2])
	})

	t.Run("deduplicates candidates and skips blanks", func(t *testing.T) {
		workingDir := filepath.Join(string(filepath.Separator), "opt", "docflow")
		executablePath := filepath.Join(workingDir, "docflow")

		candidates := buildDefaultConfigPathCandidates(executablePath, workingDir)

		require.Len(t, candidates, 2)
		assert.Equal(t, filepath.Join(workingDir, "config", "config.json"), candidates[0])
		assert.Equal(t, filepath.Join("config", "config.json"), candidates[1])

		assert.Equal(t, []string{filepath.Join("config", "config.json")}, buildDefaultConfigPathCandidates("", ""))
	})

	t.Run("returns existing cwd config", func(t *testing.T) {
		previousDir, err := os.Getwd()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, os.Chdir(previousDir))
		})

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config", "config.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
		require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0644))
		require.NoError(t, os.Chdir(tempDir))

		t.Setenv(ConfigPathEnv, "")

		assert.Equal(t, configPath, GetDefaultConfigPath())
	})
}

func TestLoadConfig(t *testing.T) {
	// Успешная загрузка конфигурации из существующего файла
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

	// Ошибка при отсутствии файла конфигурации
	t.Run("file not found", func(t *testing.T) {
		cfg, err := Load("non_existent_config.json")
		require.Error(t, err)
		assert.Nil(t, cfg)
	})

	// Ошибка при невалидном формате JSON
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
