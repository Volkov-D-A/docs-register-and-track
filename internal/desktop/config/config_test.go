package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			},
			"server": {
				"url": "http://docflow.internal:8080",
				"allowInsecureHttp": true
			}
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Загружаем конфигурацию
		cfg, err := Load(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Проверяем значения
		assert.Equal(t, "http://docflow.internal:8080", cfg.Server.URL)
		assert.True(t, cfg.Server.AllowInsecureHTTP)
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

func TestDesktopConfigDoesNotRetainInfrastructureSettings(t *testing.T) {
	var cfg Config
	err := json.Unmarshal([]byte(`{"server":{"url":"https://docflow.example","listenAddress":":9000"},"database":{"password":"db-secret"},"s3":{"secretAccessKey":"storage-secret"},"seq":{"url":"https://seq.example"},"backup":{"directory":"/server-backup"}}`), &cfg)
	require.NoError(t, err)
	require.Equal(t, "https://docflow.example", cfg.Server.URL)
	encoded, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.JSONEq(t, `{"server":{"url":"https://docflow.example"}}`, string(encoded))
}
