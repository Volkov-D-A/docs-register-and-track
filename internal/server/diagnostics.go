package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
)

type diagnosticsOutbox interface {
	QueueStats() (models.OutboxStats, error)
}

type diagnosticsSessions interface {
	Activity(time.Time, time.Time) (int, int, error)
}

type serverDiagnostics struct {
	app      *App
	outbox   diagnosticsOutbox
	sessions diagnosticsSessions
	now      func() time.Time
}

func newServerDiagnostics(app *App, outbox diagnosticsOutbox, sessions diagnosticsSessions) *serverDiagnostics {
	return &serverDiagnostics{app: app, outbox: outbox, sessions: sessions, now: time.Now}
}

func (d *serverDiagnostics) GetSystemDiagnostics() (*models.SystemDiagnostics, error) {
	now := d.now().UTC()
	result := &models.SystemDiagnostics{}
	result.Service = d.serviceStatistics(now)
	if err := d.app.db.QueryRow("SELECT pg_database_size(current_database())").Scan(&result.Database.SizeBytes); err != nil {
		slog.Warn("failed to get database size in bytes", "error", err)
	}

	if d.sessions != nil {
		sessions, users, err := d.sessions.Activity(now, now.Add(-15*time.Minute))
		if err != nil {
			return nil, fmt.Errorf("read session diagnostics: %w", err)
		}
		result.Usage.ActiveSessions = sessions
		result.Usage.ActiveUsers15m = users
	}
	if d.outbox != nil {
		stats, err := d.outbox.QueueStats()
		if err != nil {
			return nil, fmt.Errorf("read outbox diagnostics: %w", err)
		}
		result.Outbox.Pending = stats.Pending
		result.Outbox.Processing = stats.Processing
		result.Outbox.Failed = stats.Failed
	}
	d.addRuntimeMetrics(result)
	return result, nil
}

func (d *serverDiagnostics) serviceStatistics(now time.Time) models.SystemServiceStatistics {
	result := models.SystemServiceStatistics{Version: d.app.version, APIVersion: systemAPIVersion, State: "not_ready", StartedAt: d.app.startedAt}
	if !d.app.startedAt.IsZero() {
		result.UptimeSeconds = max(0, int64(now.Sub(d.app.startedAt).Seconds()))
	}
	status, err := d.app.db.GetMigrationStatus(database.DefaultMigrationsPath)
	if err != nil || status == nil {
		return result
	}
	result.SchemaCurrentVersion = status.CurrentVersion
	result.SchemaRequiredVersion = status.LatestAvailableVersion
	result.SchemaCompatible = status.Compatible
	result.SchemaDirty = status.Dirty
	if d.app.lifecycle != nil {
		if err := d.app.lifecycle.CheckReady(); err != nil {
			result.State = "maintenance"
			return result
		}
	}
	if status.UpToDate && status.Compatible && !status.Dirty {
		result.State = "ready"
	}
	return result
}

func (d *serverDiagnostics) addRuntimeMetrics(result *models.SystemDiagnostics) {
	if d.app == nil || d.app.db == nil {
		return
	}
	dbStats := d.app.db.Stats()
	result.Database.PoolOpen = dbStats.OpenConnections
	result.Database.PoolInUse = dbStats.InUse
	result.Database.PoolIdle = dbStats.Idle
	result.Database.PoolMax = dbStats.MaxOpenConnections
	result.Database.WaitCountSinceStart = dbStats.WaitCount
	result.Database.WaitMillisecondsSinceStart = dbStats.WaitDuration.Milliseconds()

	metrics := d.app.metrics
	if metrics == nil {
		return
	}
	operations := operationMetrics(metrics.Snapshot())
	if metric, ok := operations["http.request"]; ok {
		result.API.RequestsSinceStart = metric.Count
		result.API.DeadlineExceededSinceStart = metric.DeadlineExceeded
		result.API.P95Milliseconds = metric.P95.Milliseconds()
	}
	if metric, ok := operations["database.operation"]; ok {
		result.Database.OperationsSinceStart = metric.Count
		result.Database.OperationErrorsSinceStart = metric.Errors
		result.Database.OperationP95Milliseconds = metric.P95.Milliseconds()
	}
	result.API.SampleWindow = metrics.WindowSize()

	counters := counterMetrics(metrics.Counters())
	result.API.ClientErrorsSinceStart = int64(counters["http.responses.4xx"])
	result.API.ServerErrorsSinceStart = int64(counters["http.responses.5xx"])
	result.Outbox.ProcessedSinceStart = int64(counters["outbox.processed"])
	result.Outbox.RetriesSinceStart = int64(counters["outbox.retries"])
	gauges := gaugeMetrics(metrics.Gauges())
	// The diagnostics request itself is in flight while the snapshot is built.
	result.API.InFlight = max(0, int64(gauges["http.in_flight"])-1)
	result.Attachments.MissingObjects = optionalGauge(gauges, "attachments.reconciliation.missing")
	result.Attachments.OrphanObjects = optionalGauge(gauges, "attachments.reconciliation.orphan")
}

func operationMetrics(values []observability.OperationSnapshot) map[string]observability.OperationSnapshot {
	result := make(map[string]observability.OperationSnapshot, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func counterMetrics(values []observability.CounterSnapshot) map[string]float64 {
	result := make(map[string]float64, len(values))
	for _, value := range values {
		result[value.Name] = value.Value
	}
	return result
}

func gaugeMetrics(values []observability.GaugeSnapshot) map[string]float64 {
	result := make(map[string]float64, len(values))
	for _, value := range values {
		result[value.Name] = value.Value
	}
	return result
}

func optionalGauge(values map[string]float64, name string) *int {
	value, ok := values[name]
	if !ok {
		return nil
	}
	result := int(value)
	return &result
}
