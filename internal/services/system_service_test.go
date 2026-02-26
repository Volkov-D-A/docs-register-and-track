package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"docflow/internal/database"
	"docflow/internal/services"
)

func TestSystemService_CheckDBConnection_Success(t *testing.T) {
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
	service := services.NewSystemService(nil)
	service.Startup(context.Background())

	result := service.CheckDBConnection()
	assert.False(t, result)

	serviceWithNilWrappedDB := services.NewSystemService(&database.DB{DB: nil})
	result = serviceWithNilWrappedDB.CheckDBConnection()
	assert.False(t, result)
}

func TestSystemService_ReconnectDB_Success(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil)

	wrappedDB := &database.DB{DB: db}
	service := services.NewSystemService(wrappedDB)
	service.Startup(context.Background())

	result := service.ReconnectDB()

	assert.True(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemService_ReconnectDB_Failure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(errors.New("still disconnected"))

	wrappedDB := &database.DB{DB: db}
	service := services.NewSystemService(wrappedDB)
	service.Startup(context.Background())

	result := service.ReconnectDB()

	assert.False(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSystemService_ReconnectDB_NilDB(t *testing.T) {
	service := services.NewSystemService(nil)
	service.Startup(context.Background())

	result := service.ReconnectDB()
	assert.False(t, result)

	serviceWithNilWrappedDB := services.NewSystemService(&database.DB{DB: nil})
	result = serviceWithNilWrappedDB.ReconnectDB()
	assert.False(t, result)
}
