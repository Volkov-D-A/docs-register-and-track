package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
)

func TestServerDiagnosticsAggregatesRuntimeMetrics(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(20)
	metrics := observability.NewRegistry(32)
	metrics.Observe("http.request", 25*time.Millisecond, nil)
	metrics.Observe("http.request", 50*time.Millisecond, errors.New("HTTP 500"))
	metrics.Observe("http.request", 75*time.Millisecond, context.DeadlineExceeded)
	metrics.Observe("database.operation", 10*time.Millisecond, nil)
	metrics.AddCounter("http.responses.4xx", 2)
	metrics.AddCounter("http.responses.5xx", 2)
	metrics.AddCounter("outbox.processed", 7)
	metrics.AddCounter("outbox.retries", 1)
	metrics.SetGauge("http.in_flight", 3)
	metrics.SetGauge("attachments.reconciliation.missing", 2)

	diagnostics := &serverDiagnostics{app: &App{db: &database.DB{DB: sqlDB}, metrics: metrics}}
	result := &models.SystemDiagnostics{}
	diagnostics.addRuntimeMetrics(result)

	assert.EqualValues(t, 3, result.API.RequestsSinceStart)
	assert.EqualValues(t, 2, result.API.ServerErrorsSinceStart)
	assert.EqualValues(t, 1, result.API.DeadlineExceededSinceStart)
	assert.EqualValues(t, 2, result.API.ClientErrorsSinceStart)
	assert.EqualValues(t, 2, result.API.InFlight)
	assert.Equal(t, 32, result.API.SampleWindow)
	assert.EqualValues(t, 1, result.Database.OperationsSinceStart)
	assert.EqualValues(t, 7, result.Outbox.ProcessedSinceStart)
	assert.EqualValues(t, 1, result.Outbox.RetriesSinceStart)
	require.NotNil(t, result.Attachments.MissingObjects)
	assert.Equal(t, 2, *result.Attachments.MissingObjects)
	assert.Nil(t, result.Attachments.OrphanObjects)
}
