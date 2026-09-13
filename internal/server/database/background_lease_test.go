package database

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestBackgroundWorkerLeaseAcquireAndRelease(t *testing.T) {
	dbMock, mock, db := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(backgroundWorkerLeaseID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	lease, acquired, err := db.TryAcquireBackgroundWorkerLease(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)

	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(backgroundWorkerLeaseID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))
	require.NoError(t, lease.Release(context.Background()))
	require.NoError(t, lease.Release(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBackgroundWorkerLeaseReportsBusy(t *testing.T) {
	dbMock, mock, db := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(backgroundWorkerLeaseID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))
	lease, acquired, err := db.TryAcquireBackgroundWorkerLease(context.Background())
	require.NoError(t, err)
	require.False(t, acquired)
	require.Nil(t, lease)
	require.NoError(t, mock.ExpectationsWereMet())
}
