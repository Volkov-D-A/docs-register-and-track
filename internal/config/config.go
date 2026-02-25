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

// ConnectionString формирует строку подключения к базе данных.
func (d DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
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
