package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config хранит основную конфигурацию приложения.
type Config struct {
	Database DatabaseConfig `json:"database"`
	Minio    MinioConfig    `json:"minio"`
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
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, password, d.DBName, d.SSLMode,
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
func GetDefaultConfigPath() string {
	return filepath.Join("config", "config.json")
}
