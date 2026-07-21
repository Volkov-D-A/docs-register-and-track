// Package observability contains lightweight in-process performance metrics.
// It deliberately has no network exporter: desktop deployments can send its
// structured snapshots through the application's existing slog/Seq pipeline.
package observability

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"
)

const defaultWindowSize = 256

// Registry accumulates counters and a bounded latency window per operation.
// Operation names are supplied by application code and must be low-cardinality.
type Registry struct {
	mu         sync.Mutex
	windowSize int
	operations map[string]*operation
	gauges     map[string]float64
	counters   map[string]float64
}

type operation struct {
	count            int64
	errors           int64
	deadlineExceeded int64
	totalDuration    time.Duration
	samples          []time.Duration
	nextSample       int
}

// OperationSnapshot is a stable, serializable view of one operation's metrics.
type OperationSnapshot struct {
	Name             string        `json:"name"`
	Count            int64         `json:"count"`
	Errors           int64         `json:"errors"`
	DeadlineExceeded int64         `json:"deadlineExceeded"`
	TotalDuration    time.Duration `json:"totalDuration"`
	P50              time.Duration `json:"p50"`
	P95              time.Duration `json:"p95"`
	P99              time.Duration `json:"p99"`
}

// GaugeSnapshot contains the latest value of a low-cardinality instantaneous
// measurement, such as a database pool's number of in-use connections.
type GaugeSnapshot struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type CounterSnapshot struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// NewRegistry creates a registry which retains at most windowSize recent
// latency samples for each operation. Non-positive values use the default.
func NewRegistry(windowSize int) *Registry {
	if windowSize <= 0 {
		windowSize = defaultWindowSize
	}
	return &Registry{windowSize: windowSize, operations: make(map[string]*operation), gauges: make(map[string]float64), counters: make(map[string]float64)}
}

func (r *Registry) AddCounter(name string, value float64) {
	if r == nil || name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += value
}

// SetGauge records the latest value for a low-cardinality measurement.
func (r *Registry) SetGauge(name string, value float64) {
	if r == nil || name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
}

// Observe records one completed operation. Empty names are ignored so that a
// caller cannot accidentally create an anonymous metric series.
func (r *Registry) Observe(name string, duration time.Duration, err error) {
	if r == nil || name == "" {
		return
	}
	if duration < 0 {
		duration = 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	op := r.operations[name]
	if op == nil {
		op = &operation{samples: make([]time.Duration, r.windowSize)}
		r.operations[name] = op
	}
	op.count++
	op.totalDuration += duration
	if err != nil {
		op.errors++
		if errors.Is(err, context.DeadlineExceeded) {
			op.deadlineExceeded++
		}
	}
	op.samples[op.nextSample] = duration
	op.nextSample = (op.nextSample + 1) % len(op.samples)
}

// Snapshot returns metrics ordered by operation name. Percentiles use the
// nearest-rank method over the retained latency window.
func (r *Registry) Snapshot() []OperationSnapshot {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]OperationSnapshot, 0, len(r.operations))
	for name, op := range r.operations {
		sampleCount := int(op.count)
		if sampleCount > len(op.samples) {
			sampleCount = len(op.samples)
		}
		samples := append([]time.Duration(nil), op.samples[:sampleCount]...)
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		result = append(result, OperationSnapshot{
			Name:             name,
			Count:            op.count,
			Errors:           op.errors,
			DeadlineExceeded: op.deadlineExceeded,
			TotalDuration:    op.totalDuration,
			P50:              percentile(samples, 0.50),
			P95:              percentile(samples, 0.95),
			P99:              percentile(samples, 0.99),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// Gauges returns the latest gauge values ordered by name.
func (r *Registry) Gauges() []GaugeSnapshot {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]GaugeSnapshot, 0, len(r.gauges))
	for name, value := range r.gauges {
		result = append(result, GaugeSnapshot{Name: name, Value: value})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (r *Registry) Counters() []CounterSnapshot {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]CounterSnapshot, 0, len(r.counters))
	for name, value := range r.counters {
		result = append(result, CounterSnapshot{Name: name, Value: value})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// LogSnapshot writes one structured record for every collected operation.
func (r *Registry) LogSnapshot(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	for _, metric := range r.Snapshot() {
		logger.Info("operation metrics",
			"operation", metric.Name,
			"count", metric.Count,
			"errors", metric.Errors,
			"deadline_exceeded", metric.DeadlineExceeded,
			"total_duration", metric.TotalDuration,
			"p50", metric.P50,
			"p95", metric.P95,
			"p99", metric.P99,
		)
	}
	for _, gauge := range r.Gauges() {
		logger.Info("gauge metric", "metric", gauge.Name, "value", gauge.Value)
	}
	for _, counter := range r.Counters() {
		logger.Info("counter metric", "metric", counter.Name, "value", counter.Value)
	}
}

// LogPeriodically emits snapshots until ctx is cancelled. It is intended to
// run from application startup, using the existing structured log pipeline.
func LogPeriodically(ctx context.Context, registry *Registry, logger *slog.Logger, interval time.Duration) {
	if registry == nil || interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			registry.LogSnapshot(logger)
		}
	}
}

func percentile(samples []time.Duration, probability float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	index := int(probability*float64(len(samples)) + 0.9999999999)
	if index < 1 {
		index = 1
	}
	if index > len(samples) {
		index = len(samples)
	}
	return samples[index-1]
}
