package database

import (
	"docflow/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect_Failure(t *testing.T) {
	// Передаем недействительный порт для вызова ошибки ping
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     33333, // вероятно, недействительный порт
		User:     "invalid_user",
		Password: "invalid_password",
		DBName:   "invalid_db",
		SSLMode:  "disable",
	}

	db, err := Connect(cfg)

	// Сам Open может не завершиться ошибкой, так как он только проверяет аргументы,
	// но Ping должен точно вернуть ошибку.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ping database")
	assert.Nil(t, db)
}

func TestMigrations_InvalidPath(t *testing.T) {
	// Мы можем проверить, что передача несуществующего пути в RunMigrations
	// вернет nil, так как она пропускает миграции при отсутствии директории.
	db := &DB{} // Пустая структура DB только для вызова метода

	// Создаем пустую обертку DB с nil sql.DB - если метод попытается использовать ее досрочно, произойдет паника
	// Однако RunMigrations сначала проверяет наличие пути.
	err := db.RunMigrations("/invalid/non/existent/path")
	assert.NoError(t, err) // код говорит: "Migration directory not found. Skipping migrations. return nil"
}
