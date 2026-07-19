package repository

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

// BenchmarkIntegrationDocumentList is intentionally a baseline, not a CI
// budget. Its verbose output includes PostgreSQL's real execution plan for
// comparison when a query or index changes.
func BenchmarkIntegrationDocumentList(b *testing.B) {
	sqlDB := integrationdb.Open(b)
	db := &database.DB{DB: sqlDB}
	nomID, owner, orgID := seedPerformanceDocuments(b, sqlDB, 250)
	repo := NewOutgoingDocumentRepository(db)
	filter := models.OutgoingDocumentFilter{AllowedNomenclatureIDs: []string{nomID.String()}, Search: "performance", Page: 1, PageSize: 50}
	logExplain(b, sqlDB, `SELECT id FROM documents WHERE nomenclature_id = $1 AND content ILIKE $2 ORDER BY created_at DESC LIMIT 50`, nomID, "%performance%")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := repo.GetList(filter)
		if err != nil || len(result.Items) != 50 {
			b.Fatalf("list result=%d err=%v", len(result.Items), err)
		}
	}
	_ = owner
	_ = orgID
}

func BenchmarkIntegrationStatistics(b *testing.B) {
	sqlDB := integrationdb.Open(b)
	db := &database.DB{DB: sqlDB}
	seedPerformanceDocuments(b, sqlDB, 500)
	repo := NewStatisticsRepository(db)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)
	logExplain(b, sqlDB, `SELECT EXTRACT(MONTH FROM registration_date)::int, kind, COUNT(*) FROM documents WHERE registration_date >= $1 AND registration_date < $2 GROUP BY 1, 2`, start, end)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		points, err := repo.GetMonthlyDocumentCountsByKind(start, end)
		if err != nil || len(points) == 0 {
			b.Fatalf("statistics points=%d err=%v", len(points), err)
		}
	}
}

func seedPerformanceDocuments(tb testing.TB, db *sql.DB, count int) (uuid.UUID, uuid.UUID, uuid.UUID) {
	tb.Helper()
	owner := insertIntegrationUser(tb, db, "performance_owner")
	nomID, orgID := uuid.New(), uuid.New()
	execSQL(tb, db, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode) VALUES ($1, 'Performance', 'PF', 2026, 'outgoing_letter', '/', 'index_and_number')`, nomID)
	execSQL(tb, db, `INSERT INTO organizations (id, name) VALUES ($1, 'Performance Org')`, orgID)
	for i := 0; i < count; i++ {
		id := uuid.New()
		date := time.Date(2026, time.Month(i%12+1), 1, 0, 0, 0, 0, time.UTC)
		number := fmt.Sprintf("PF/%d", i+1)
		execSQL(tb, db, `INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by, created_at) VALUES ($1, 'outgoing_letter', $2, $3, $4, $5, $6, $7, 1, $8, $9)`, id, nomID, uuid.New(), number, date, models.DocumentTypeLetter, fmt.Sprintf("performance searchable %d", i), owner, date)
		execSQL(tb, db, `INSERT INTO outgoing_document_details (document_id, outgoing_number, outgoing_date, sender_signatory, sender_executor, recipient_org_id, addressee) VALUES ($1, $2, $3, 'Signer', 'Executor', $4, 'Addressee')`, id, number, date, orgID)
	}
	return nomID, owner, orgID
}

func logExplain(b *testing.B, db *sql.DB, query string, args ...any) {
	b.Helper()
	var plan string
	if err := db.QueryRow(`EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) `+query, args...).Scan(&plan); err != nil {
		b.Fatalf("explain benchmark query: %v", err)
	}
	b.Logf("postgres plan: %s", plan)
	stats := db.Stats()
	b.Logf("connection pool: open=%d in_use=%d idle=%d wait_count=%d wait_duration=%s", stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, stats.WaitDuration)
}
