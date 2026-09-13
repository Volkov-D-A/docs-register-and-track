package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
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

	expected := "host=localhost port=5432 user=testuser password=testpassword dbname=testdb sslmode=disable connect_timeout=10 statement_timeout=30000 lock_timeout=5000"
	assert.Equal(t, expected, dbCfg.ConnectionString())
}

func TestS3ConfigGetSecretAccessKey(t *testing.T) {
	cfg := S3Config{SecretAccessKey: "plain-secret"}

	assert.Equal(t, "plain-secret", cfg.GetSecretAccessKey())
}
