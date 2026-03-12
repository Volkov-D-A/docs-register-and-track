package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
)

func TestSystemService_CheckDBConnection_Success(t *testing.T) {
	// Проверка успешной установки коннекта к базе данных и отправки ping
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil)

	wrappedDB := &database.DB{DB: db}
	service := services.NewSystemService(wrappedDB)
	service.Startup(context.Background())

	result := service.CheckDBConnection()

	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemService_CheckDBConnection_Failure(t *testing.T) {
	// Проверка случая, когда ping к базе возвращает ошибку подключения
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(errors.New("connection failed"))

	wrappedDB := &database.DB{DB: db}
	service := services.NewSystemService(wrappedDB)
	service.Startup(context.Background())

	result := service.CheckDBConnection()

	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemService_CheckDBConnection_NilDB(t *testing.T) {
	// Защита от panic при попытке проверить соединение в случае отсутствия инстанса DB
	service := services.NewSystemService(nil)
	service.Startup(context.Background())

	result := service.CheckDBConnection()
	assert.False(t, result)

	serviceWithNilWrappedDB := services.NewSystemService(&database.DB{DB: nil})
	result = serviceWithNilWrappedDB.CheckDBConnection()
	assert.False(t, result)
}
