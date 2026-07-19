package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ConfigPathEnv             = "DOCFLOW_CONFIG_PATH"
	defaultConfigFileName     = "config.json"
	defaultConfigDirName      = "config"
	defaultDBConnectTimeout   = 10 * time.Second
	defaultDBStatementTimeout = 30 * time.Second
	defaultDBLockTimeout      = 5 * time.Second
)

// Config хранит основную конфигурацию приложения.
type Config struct {
	Database DatabaseConfig `json:"database"`
	Minio    MinioConfig    `json:"minio"`
	Seq      SeqConfig      `json:"seq"`
}

// SeqConfig хранит настройки подключения к Seq
type SeqConfig struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// DatabaseConfig хранит настройки подключения к базе данных PostgreSQL.
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// MinioConfig хранит настройки подключения к MinIO.
type MinioConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	UseSSL          bool   `json:"useSSL"`
	BucketName      string `json:"bucketName"`
}

// GetSecretAccessKey возвращает ключ доступа.
// Если он зашифрован (префикс ENC:), автоматически дешифрует его.
func (m MinioConfig) GetSecretAccessKey() string {
	secret := m.SecretAccessKey
	if decrypted, err := DecryptPassword(m.SecretAccessKey); err == nil {
		secret = decrypted
	}
	return secret
}

// ConnectionString формирует строку подключения к базе данных.
// Если пароль зашифрован (префикс ENC:), автоматически дешифрует его.
func (d DatabaseConfig) ConnectionString() string {
	password := d.Password
	if decrypted, err := DecryptPassword(d.Password); err == nil {
		password = decrypted
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d statement_timeout=%d lock_timeout=%d",
		d.Host, d.Port, d.User, password, d.DBName, d.SSLMode,
		int(defaultDBConnectTimeout/time.Second),
		defaultDBStatementTimeout.Milliseconds(),
		defaultDBLockTimeout.Milliseconds(),
	)
}

// Load загружает конфигурацию из файла по указанному пути.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetDefaultConfigPath возвращает путь к файлу конфигурации по умолчанию.
// Lookup order:
// 1. DOCFLOW_CONFIG_PATH, if set;
// 2. config/config.json next to the executable;
// 3. config/config.json in the current working directory for local development.
func GetDefaultConfigPath() string {
	if override := strings.TrimSpace(os.Getenv(ConfigPathEnv)); override != "" {
		return override
	}

	candidates := getDefaultConfigPathCandidates()
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return filepath.Join(defaultConfigDirName, defaultConfigFileName)
}

func getDefaultConfigPathCandidates() []string {
	executablePath, _ := os.Executable()
	workingDir, _ := os.Getwd()
	return buildDefaultConfigPathCandidates(executablePath, workingDir)
}

func buildDefaultConfigPathCandidates(executablePath, workingDir string) []string {
	var candidates []string
	seen := make(map[string]struct{})
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		candidates = append(candidates, clean)
	}

	if executablePath != "" {
		add(filepath.Join(filepath.Dir(executablePath), defaultConfigDirName, defaultConfigFileName))
	}
	if workingDir != "" {
		add(filepath.Join(workingDir, defaultConfigDirName, defaultConfigFileName))
	}
	add(filepath.Join(defaultConfigDirName, defaultConfigFileName))
	return candidates
}
