package main

import (
	"docflow/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForceMigration_InvalidPath(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "password",
		DBName:   "db",
	}

	// Это должно вызвать ошибку, так как путь к миграции недействителен
	err := ForceMigration(cfg, "invalid/path", 4)
	assert.Error(t, err)
}
