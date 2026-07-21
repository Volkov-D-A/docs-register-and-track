// dbperf records a small, reproducible PostgreSQL performance baseline.
// It is deliberately a reporting tool: it never turns timing variance into a
// pass/fail result.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

const samples = 15

type snapshot struct {
	CreatedAt       time.Time `json:"createdAt"`
	PostgresVersion string    `json:"postgresVersion"`
	DocumentCount   int       `json:"documentCount"`
	PageSize        int       `json:"pageSize"`
	DeepPage        int       `json:"deepPage"`
	Metrics         []metric  `json:"metrics"`
}

type metric struct {
	Name                 string          `json:"name"`
	Samples              int             `json:"samples"`
	MedianMilliseconds   float64         `json:"medianMilliseconds"`
	P95Milliseconds      float64         `json:"p95Milliseconds"`
	ExplainMilliseconds  float64         `json:"explainMilliseconds"`
	PlanningMilliseconds float64         `json:"planningMilliseconds"`
	ActualRows           float64         `json:"actualRows"`
	SharedHitBlocks      float64         `json:"sharedHitBlocks"`
	SharedReadBlocks     float64         `json:"sharedReadBlocks"`
	PoolWaitCount        int64           `json:"poolWaitCount"`
	Plan                 explainDocument `json:"plan"`
}

type explainDocument struct {
	Plan          map[string]any `json:"Plan"`
	PlanningTime  float64        `json:"Planning Time"`
	ExecutionTime float64        `json:"Execution Time"`
}

func main() {
	dsn := flag.String("dsn", os.Getenv("DOCFLOW_INTEGRATION_DSN"), "safe PostgreSQL DSN")
	outDir := flag.String("out", "build/performance", "directory for local snapshots")
	documentCount := flag.Int("documents", 500, "number of seeded outgoing documents (at least page-size * deep-page)")
	pageSize := flag.Int("page-size", 50, "list page size to measure")
	deepPage := flag.Int("deep-page", 0, "deep OFFSET page; defaults to the last full page")
	flag.Parse()
	if *documentCount < 1 || *pageSize < 1 {
		fail("-documents and -page-size must be positive")
	}
	if *deepPage == 0 {
		*deepPage = *documentCount / *pageSize
	}
	if *deepPage < 2 || *deepPage**pageSize > *documentCount {
		fail("-deep-page must be at least 2 and fit within -documents")
	}
	if *dsn == "" {
		fail("-dsn or DOCFLOW_INTEGRATION_DSN is required")
	}
	if err := integrationdb.ValidateDSN(*dsn); err != nil {
		fail("unsafe DSN: %v", err)
	}

	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		fail("open database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		fail("ping database: %v", err)
	}
	if _, err := db.Exec(`DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		fail("reset schema: %v", err)
	}
	if err := (&database.DB{DB: db}).RunMigrations(database.DefaultMigrationsPath); err != nil {
		fail("apply migrations: %v", err)
	}

	var version string
	if err := db.QueryRow(`SHOW server_version`).Scan(&version); err != nil {
		fail("read PostgreSQL version: %v", err)
	}
	nomenclatureID := seed(db, *documentCount)
	repoDB := &database.DB{DB: db}
	documentRepo := repository.NewOutgoingDocumentRepository(repoDB)
	statisticsRepo := repository.NewStatisticsRepository(repoDB)

	metrics := []metric{
		measureOutgoingList("outgoing_list_first_page", db, documentRepo, nomenclatureID, "", 1, *pageSize),
		measureOutgoingList("outgoing_search_first_page", db, documentRepo, nomenclatureID, "performance", 1, *pageSize),
		measureOutgoingList("outgoing_search_deep_offset", db, documentRepo, nomenclatureID, "performance", *deepPage, *pageSize),
		measureOutgoingCursor("outgoing_search_deep_cursor", db, documentRepo, nomenclatureID, "performance", *deepPage, *pageSize),
		measureOutgoingList("outgoing_search_selective", db, documentRepo, nomenclatureID, "needle", 1, *pageSize),
		measureOutgoingCount("outgoing_search_selective_count", db, nomenclatureID, "needle"),
		measureOutgoingList("outgoing_number_contains", db, documentRepo, nomenclatureID, "", 1, *pageSize),
		measure("monthly_document_statistics", db, func() error {
			items, err := statisticsRepo.GetMonthlyDocumentCountsByKind(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
			if err == nil && len(items) == 0 {
				return fmt.Errorf("got no statistics rows")
			}
			return err
		}, `SELECT EXTRACT(MONTH FROM registration_date)::int, kind, COUNT(*) FROM documents WHERE registration_date >= $1 AND registration_date < $2 GROUP BY 1, 2`, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	current := snapshot{CreatedAt: time.Now().UTC(), PostgresVersion: version, DocumentCount: *documentCount, PageSize: *pageSize, DeepPage: *deepPage, Metrics: metrics}
	previous := readSnapshot(filepath.Join(*outDir, "latest.json"))
	if err := os.MkdirAll(filepath.Join(*outDir, "history"), 0o755); err != nil {
		fail("create output directory: %v", err)
	}
	writeSnapshot(filepath.Join(*outDir, "latest.json"), current)
	writeSnapshot(filepath.Join(*outDir, "history", current.CreatedAt.Format("20060102T150405Z")+".json"), current)
	printSummary(current, previous)
}

func measureOutgoingList(name string, db *sql.DB, repo *repository.OutgoingDocumentRepository, nomenclatureID uuid.UUID, search string, page, pageSize int) metric {
	filter := models.OutgoingDocumentFilter{AllowedNomenclatureIDs: []string{nomenclatureID.String()}, Search: search, Page: page, PageSize: pageSize}
	if name == "outgoing_number_contains" {
		filter.OutgoingNumber = "PF/"
	}
	offset := (page - 1) * pageSize
	where := "d.kind = $1 AND d.nomenclature_id = $2"
	args := []any{models.DocumentKindOutgoingLetter, nomenclatureID}
	if filter.Search != "" {
		where += " AND (d.content ILIKE $3 OR out.outgoing_number ILIKE $3)"
		args = append(args, "%"+filter.Search+"%")
	} else if filter.OutgoingNumber != "" {
		where += " AND out.outgoing_number ILIKE $3"
		args = append(args, "%"+filter.OutgoingNumber+"%")
	}
	args = append(args, pageSize, offset)
	return measure(name, db, func() error {
		result, err := repo.GetList(filter)
		if err == nil && len(result.Items) == 0 {
			return fmt.Errorf("got no items")
		}
		return err
	}, fmt.Sprintf(`SELECT d.id FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id WHERE %s ORDER BY d.created_at DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
}

func measureOutgoingCount(name string, db *sql.DB, nomenclatureID uuid.UUID, search string) metric {
	args := []any{models.DocumentKindOutgoingLetter, nomenclatureID, "%" + search + "%"}
	query := `
		SELECT COUNT(*)
		FROM documents d
		JOIN outgoing_document_details out ON out.document_id = d.id
		WHERE d.kind = $1 AND d.nomenclature_id = $2
		  AND (d.content ILIKE $3 OR out.outgoing_number ILIKE $3)`
	return measure(name, db, func() error {
		var total int
		return db.QueryRow(query, args...).Scan(&total)
	}, query, args...)
}

func measureOutgoingCursor(name string, db *sql.DB, repo *repository.OutgoingDocumentRepository, nomenclatureID uuid.UUID, search string, page, pageSize int) metric {
	var createdAt time.Time
	var id uuid.UUID
	position := (page-1)*pageSize - 1
	err := db.QueryRow(`SELECT d.created_at, d.id
		FROM documents d WHERE d.kind = $1 AND d.nomenclature_id = $2
		ORDER BY d.created_at DESC, d.id DESC LIMIT 1 OFFSET $3`, models.DocumentKindOutgoingLetter, nomenclatureID, position).Scan(&createdAt, &id)
	if err != nil {
		fail("read cursor position: %v", err)
	}
	cursor, err := models.EncodeDocumentCursor(createdAt, id)
	if err != nil {
		fail("encode cursor: %v", err)
	}
	filter := models.OutgoingDocumentFilter{
		AllowedNomenclatureIDs: []string{nomenclatureID.String()}, Search: search,
		PageSize: pageSize, CursorPagination: true, Cursor: cursor,
	}
	args := []any{models.DocumentKindOutgoingLetter, nomenclatureID, "%" + search + "%", createdAt, id, pageSize + 1}
	query := `SELECT d.id FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id
		WHERE d.kind = $1 AND d.nomenclature_id = $2
		AND (d.content ILIKE $3 OR out.outgoing_number ILIKE $3)
		AND (d.created_at, d.id) < ($4, $5)
		ORDER BY d.created_at DESC, d.id DESC LIMIT $6`
	return measure(name, db, func() error {
		result, err := repo.GetList(filter)
		if err == nil && len(result.Items) == 0 {
			return fmt.Errorf("got no items")
		}
		return err
	}, query, args...)
}

func measure(name string, db *sql.DB, operation func() error, explainSQL string, args ...any) metric {
	durations := make([]float64, 0, samples)
	for range samples {
		started := time.Now()
		if err := operation(); err != nil {
			fail("%s: %v", name, err)
		}
		durations = append(durations, float64(time.Since(started).Microseconds())/1000)
	}
	sort.Float64s(durations)
	plan := explain(db, explainSQL, args...)
	stats := db.Stats()
	return metric{Name: name, Samples: samples, MedianMilliseconds: percentile(durations, 0.5), P95Milliseconds: percentile(durations, 0.95), ExplainMilliseconds: plan.ExecutionTime, PlanningMilliseconds: plan.PlanningTime, ActualRows: number(plan.Plan["Actual Rows"]), SharedHitBlocks: planValue(plan.Plan, "Shared Hit Blocks"), SharedReadBlocks: planValue(plan.Plan, "Shared Read Blocks"), PoolWaitCount: stats.WaitCount, Plan: plan}
}

func explain(db *sql.DB, query string, args ...any) explainDocument {
	var raw []byte
	if err := db.QueryRow(`EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) `+query, args...).Scan(&raw); err != nil {
		fail("explain query: %v", err)
	}
	var documents []explainDocument
	if err := json.Unmarshal(raw, &documents); err != nil || len(documents) != 1 {
		fail("decode explain JSON: %v", err)
	}
	return documents[0]
}

func seed(db *sql.DB, count int) uuid.UUID {
	userID, nomID, orgID := uuid.New(), uuid.New(), uuid.New()
	mustExec(db, `INSERT INTO users (id, login, password_hash, full_name) VALUES ($1, 'dbperf', 'hash', 'DB Perf')`, userID)
	mustExec(db, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode) VALUES ($1, 'Performance', 'PF', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomID)
	mustExec(db, `INSERT INTO organizations (id, name) VALUES ($1, 'Performance Org')`, orgID)
	tx, err := db.Begin()
	if err != nil {
		fail("begin seed transaction: %v", err)
	}
	defer tx.Rollback()
	for i := 0; i < count; i++ {
		id := uuid.New()
		date := time.Date(2026, time.Month(i%12+1), 1, 0, 0, 0, 0, time.UTC)
		number := fmt.Sprintf("PF/%d", i+1)
		content := fmt.Sprintf("performance searchable %d", i)
		if i%100 == 0 {
			content += " needle"
		}
		if _, err := tx.Exec(`INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by, created_at) VALUES ($1, 'outgoing_letter', $2, $3, $4, $5, $6, $7, 1, $8, $9)`, id, nomID, uuid.New(), number, date, models.DocumentTypeLetter, content, userID, date); err != nil {
			fail("seed document: %v", err)
		}
		if _, err := tx.Exec(`INSERT INTO outgoing_document_details (document_id, outgoing_number, outgoing_date, sender_signatory, sender_executor, recipient_org_id, addressee) VALUES ($1, $2, $3, 'Signer', 'Executor', $4, 'Addressee')`, id, number, date, orgID); err != nil {
			fail("seed outgoing details: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		fail("commit seed transaction: %v", err)
	}
	return nomID
}

func planValue(plan map[string]any, key string) float64 {
	value := number(plan[key])
	if children, ok := plan["Plans"].([]any); ok {
		for _, child := range children {
			if node, ok := child.(map[string]any); ok {
				value += planValue(node, key)
			}
		}
	}
	return value
}
func number(value any) float64 {
	if n, ok := value.(float64); ok {
		return n
	}
	return 0
}
func percentile(values []float64, p float64) float64 {
	return values[int(math.Ceil(p*float64(len(values))))-1]
}
func mustExec(db *sql.DB, query string, args ...any) {
	if _, err := db.Exec(query, args...); err != nil {
		fail("seed database: %v", err)
	}
}
func readSnapshot(path string) *snapshot {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var value snapshot
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	return &value
}
func writeSnapshot(path string, value snapshot) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail("encode snapshot: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		fail("write snapshot: %v", err)
	}
}
func printSummary(current snapshot, previous *snapshot) {
	fmt.Printf("\nPostgreSQL performance summary (%s, PostgreSQL %s)\n", current.CreatedAt.Format(time.RFC3339), current.PostgresVersion)
	fmt.Println("metric                         median    p95    explain  hit/read  rows    vs previous")
	old := map[string]metric{}
	if previous != nil {
		for _, item := range previous.Metrics {
			old[item.Name] = item
		}
	}
	for _, item := range current.Metrics {
		delta := "new"
		if before, ok := old[item.Name]; ok && before.MedianMilliseconds != 0 {
			delta = fmt.Sprintf("%+.1f%%", (item.MedianMilliseconds/before.MedianMilliseconds-1)*100)
		}
		fmt.Printf("%-30s %7.2fms %6.2fms %7.2fms %5.0f/%-5.0f %-7.0f %s\n", item.Name, item.MedianMilliseconds, item.P95Milliseconds, item.ExplainMilliseconds, item.SharedHitBlocks, item.SharedReadBlocks, item.ActualRows, delta)
	}
	fmt.Println("Snapshots: build/performance/latest.json and build/performance/history/")
}
func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "dbperf: "+format+"\n", args...)
	os.Exit(1)
}
