package repository

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestDocumentRegistrationIdempotencyIntegration(t *testing.T) {
	sqlDB := openSafeIntegrationDB(t)
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	repo := NewOutgoingDocumentRepository(db)

	userID := uuid.New()
	nomID := uuid.New()
	orgID := uuid.New()

	execSQL(t, sqlDB, `
		INSERT INTO users (id, login, password_hash, full_name)
		VALUES ($1, 'regression_user', 'hash', 'Regression User')
	`, userID)
	execSQL(t, sqlDB, `
		INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode, next_number)
		VALUES ($1, 'Regression outgoing', '01-01', 2026, 'outgoing_letter', '/', 'index_and_number', 1)
	`, nomID)
	execSQL(t, sqlDB, `
		INSERT INTO organizations (id, name)
		VALUES ($1, 'Regression Org')
	`, orgID)

	firstKey := uuid.New()
	firstReq := models.CreateOutgoingDocRequest{
		NomenclatureID:  nomID,
		IdempotencyKey:  firstKey,
		DocumentTypeID:  models.DocumentTypeLetter,
		RecipientOrgID:  orgID,
		CreatedBy:       userID,
		OutgoingDate:    time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
		Content:         "first",
		PagesCount:      1,
		SenderSignatory: "Signer",
		SenderExecutor:  "Executor",
		Addressee:       "Addressee",
	}

	first, err := repo.Create(firstReq)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if first.OutgoingNumber != "01-01/1" {
		t.Fatalf("first number = %q, want 01-01/1", first.OutgoingNumber)
	}

	repeated, err := repo.Create(firstReq)
	if err != nil {
		t.Fatalf("repeat create: %v", err)
	}
	if repeated.ID != first.ID {
		t.Fatalf("repeat returned id %s, want %s", repeated.ID, first.ID)
	}
	assertScalar(t, sqlDB, `SELECT next_number FROM nomenclature WHERE id = $1`, []any{nomID}, 2)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM documents WHERE created_by = $1 AND kind = 'outgoing_letter'`, []any{userID}, 1)

	failingReq := firstReq
	failingReq.IdempotencyKey = uuid.New()
	failingReq.RecipientOrgID = uuid.New()
	_, err = repo.Create(failingReq)
	if err == nil {
		t.Fatalf("expected failing create with invalid recipient org")
	}
	assertScalar(t, sqlDB, `SELECT next_number FROM nomenclature WHERE id = $1`, []any{nomID}, 2)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM documents WHERE created_by = $1 AND kind = 'outgoing_letter'`, []any{userID}, 1)

	secondReq := firstReq
	secondReq.IdempotencyKey = uuid.New()
	secondReq.Content = "second"
	second, err := repo.Create(secondReq)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if second.OutgoingNumber != "01-01/2" {
		t.Fatalf("second number = %q, want 01-01/2", second.OutgoingNumber)
	}
}

func TestDocumentRegistrationConcurrencyIntegration(t *testing.T) {
	sqlDB := openSafeIntegrationDB(t)
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	repo := NewOutgoingDocumentRepository(db)

	userID := uuid.New()
	nomID := uuid.New()
	orgID := uuid.New()

	execSQL(t, sqlDB, `
		INSERT INTO users (id, login, password_hash, full_name)
		VALUES ($1, 'concurrent_user', 'hash', 'Concurrent User')
	`, userID)
	execSQL(t, sqlDB, `
		INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode, next_number)
		VALUES ($1, 'Concurrent outgoing', '02-02', 2026, 'outgoing_letter', '/', 'index_and_number', 1)
	`, nomID)
	execSQL(t, sqlDB, `
		INSERT INTO organizations (id, name)
		VALUES ($1, 'Concurrent Org')
	`, orgID)

	baseReq := models.CreateOutgoingDocRequest{
		NomenclatureID:  nomID,
		DocumentTypeID:  models.DocumentTypeLetter,
		RecipientOrgID:  orgID,
		CreatedBy:       userID,
		OutgoingDate:    time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
		PagesCount:      1,
		SenderSignatory: "Signer",
		SenderExecutor:  "Executor",
		Addressee:       "Addressee",
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	numbers := make(chan string, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := baseReq
			req.IdempotencyKey = uuid.New()
			req.Content = fmt.Sprintf("different-key-%d", i)
			doc, err := repo.Create(req)
			if err != nil {
				errs <- err
				return
			}
			numbers <- doc.OutgoingNumber
		}(i)
	}
	wg.Wait()
	close(errs)
	close(numbers)

	for err := range errs {
		if err != nil {
			t.Fatalf("parallel create with different keys: %v", err)
		}
	}

	gotNumbers := make([]string, 0, workers)
	for number := range numbers {
		gotNumbers = append(gotNumbers, number)
	}
	sort.Strings(gotNumbers)
	wantNumbers := []string{"02-02/1", "02-02/2", "02-02/3", "02-02/4", "02-02/5", "02-02/6", "02-02/7", "02-02/8"}
	if strings.Join(gotNumbers, ",") != strings.Join(wantNumbers, ",") {
		t.Fatalf("parallel numbers = %v, want %v", gotNumbers, wantNumbers)
	}
	assertScalar(t, sqlDB, `SELECT next_number FROM nomenclature WHERE id = $1`, []any{nomID}, 9)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM documents WHERE created_by = $1 AND kind = 'outgoing_letter'`, []any{userID}, workers)

	sharedKey := uuid.New()
	sharedReq := baseReq
	sharedReq.IdempotencyKey = sharedKey
	sharedReq.Content = "shared-key"

	const repeats = 6
	var repeatWG sync.WaitGroup
	repeatErrs := make(chan error, repeats)
	ids := make(chan uuid.UUID, repeats)

	for i := 0; i < repeats; i++ {
		repeatWG.Add(1)
		go func() {
			defer repeatWG.Done()
			doc, err := repo.Create(sharedReq)
			if err != nil {
				repeatErrs <- err
				return
			}
			ids <- doc.ID
		}()
	}
	repeatWG.Wait()
	close(repeatErrs)
	close(ids)

	for err := range repeatErrs {
		if err != nil {
			t.Fatalf("parallel create with same key: %v", err)
		}
	}

	var firstID uuid.UUID
	for id := range ids {
		if firstID == uuid.Nil {
			firstID = id
			continue
		}
		if id != firstID {
			t.Fatalf("same idempotency key returned ids %s and %s", firstID, id)
		}
	}
	assertScalar(t, sqlDB, `SELECT next_number FROM nomenclature WHERE id = $1`, []any{nomID}, 10)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM documents WHERE created_by = $1 AND kind = 'outgoing_letter'`, []any{userID}, workers+1)
}

// TestOutboxClaimConcurrencyIntegration verifies the mutual exclusion that a
// sqlmock test cannot prove: two independent workers connected to PostgreSQL
// must not receive the same outbox row.
func TestOutboxClaimConcurrencyIntegration(t *testing.T) {
	sqlDB := openSafeIntegrationDB(t)
	defer sqlDB.Close()

	db := &database.DB{DB: sqlDB}
	outbox := NewOutboxRepository(db)
	event := models.OutboxEvent{
		EventType:        models.OutboxEventJournal,
		DeduplicationKey: "integration-claim-" + uuid.NewString(),
		Payload:          `{"DocumentID":"00000000-0000-0000-0000-000000000000","UserID":"00000000-0000-0000-0000-000000000000","Action":"TEST","Details":"concurrent claim"}`,
	}
	if err := outbox.Enqueue(event); err != nil {
		t.Fatalf("enqueue event: %v", err)
	}

	start := make(chan struct{})
	results := make(chan []models.OutboxEvent, 2)
	errs := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			claimed, err := outbox.ClaimPending(1)
			errs <- err
			results <- claimed
		}()
	}
	close(start)
	workers.Wait()
	close(errs)
	close(results)

	claimedCount := 0
	for err := range errs {
		if err != nil {
			t.Fatalf("claim pending: %v", err)
		}
	}
	for claimed := range results {
		claimedCount += len(claimed)
	}
	if claimedCount != 1 {
		t.Fatalf("total claimed events = %d, want 1", claimedCount)
	}
}

func TestJournalRetentionFKIntegration(t *testing.T) {
	sqlDB := openSafeIntegrationDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	nomID := uuid.New()
	docID := uuid.New()

	execSQL(t, sqlDB, `
		INSERT INTO users (id, login, password_hash, full_name)
		VALUES ($1, 'retention_user', 'hash', 'Retention User')
	`, userID)
	execSQL(t, sqlDB, `
		INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode, next_number)
		VALUES ($1, 'Retention outgoing', '03-03', 2026, 'outgoing_letter', '/', 'index_and_number', 1)
	`, nomID)
	execSQL(t, sqlDB, `
		INSERT INTO documents (
			id, kind, nomenclature_id, idempotency_key, registration_number, registration_date,
			document_type, content, pages_count, created_by
		) VALUES ($1, 'outgoing_letter', $2, $3, '03-03/1', DATE '2026-05-28', $4, 'retention', 1, $5)
	`, docID, nomID, uuid.New(), models.DocumentTypeLetter, userID)
	execSQL(t, sqlDB, `
		INSERT INTO outgoing_document_details (
			document_id, outgoing_number, outgoing_date, sender_signatory,
			sender_executor, recipient_org_id, addressee
		) VALUES ($1, '03-03/1', DATE '2026-05-28', 'Signer', 'Executor', $2, 'Addressee')
	`, docID, insertOrganization(t, sqlDB, "Retention Org"))
	execSQL(t, sqlDB, `
		INSERT INTO document_journal (document_id, user_id, action, details)
		VALUES ($1, $2, 'CREATE', 'retention check')
	`, docID, userID)
	execSQL(t, sqlDB, `
		INSERT INTO admin_audit_log (user_id, user_name, action, details)
		VALUES ($1, 'Retention User', 'USER_UPDATE', 'retention check')
	`, userID)

	if _, err := sqlDB.Exec(`DELETE FROM documents WHERE id = $1`, docID); err == nil {
		t.Fatalf("expected document delete to be restricted by document_journal")
	}
	if _, err := sqlDB.Exec(`DELETE FROM users WHERE id = $1`, userID); err == nil {
		t.Fatalf("expected user delete to be restricted by journal/audit")
	}

	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM document_journal WHERE document_id = $1`, []any{docID}, 1)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM admin_audit_log WHERE user_id = $1`, []any{userID}, 1)
}

func TestDatabaseConstraintsIntegration(t *testing.T) {
	sqlDB := openSafeIntegrationDB(t)
	defer sqlDB.Close()

	userID := uuid.New()
	nomID := uuid.New()
	orgID := uuid.New()
	docID := uuid.New()

	execSQL(t, sqlDB, `
		INSERT INTO users (id, login, password_hash, full_name)
		VALUES ($1, 'constraint_user', 'hash', 'Constraint User')
	`, userID)
	execSQL(t, sqlDB, `
		INSERT INTO organizations (id, name)
		VALUES ($1, 'Constraint Org')
	`, orgID)
	execSQL(t, sqlDB, `
		INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode, next_number)
		VALUES ($1, 'Constraint outgoing', '04-04', 2026, 'outgoing_letter', '/', 'index_and_number', 1)
	`, nomID)
	execSQL(t, sqlDB, `
		INSERT INTO documents (
			id, kind, nomenclature_id, idempotency_key, registration_number, registration_date,
			document_type, content, pages_count, created_by
		) VALUES ($1, 'outgoing_letter', $2, $3, '04-04/1', DATE '2026-05-28', $4, 'constraint', 1, $5)
	`, docID, nomID, uuid.New(), models.DocumentTypeLetter, userID)

	expectExecError(t, sqlDB, `
		INSERT INTO documents (
			kind, nomenclature_id, idempotency_key, registration_number, registration_date,
			document_type, content, pages_count, created_by
		) VALUES ('outgoing_letter', $1, $2, '04-04/1', DATE '2026-12-31', $3, 'duplicate number', 1, $4)
	`, nomID, uuid.New(), models.DocumentTypeLetter, userID)
	// idempotency_key has a database default, so direct writes remain safe even
	// if an older client omits the column.
	execSQL(t, sqlDB, `
		INSERT INTO documents (
			kind, nomenclature_id, registration_number, registration_date,
			document_type, content, pages_count, created_by
		) VALUES ('outgoing_letter', $1, '04-04/2', DATE '2026-05-28', $2, 'missing idempotency', 1, $3)
	`, nomID, models.DocumentTypeLetter, userID)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM documents WHERE created_by = $1 AND idempotency_key IS NOT NULL`, []any{userID}, 2)
	expectExecError(t, sqlDB, `
		INSERT INTO assignments (document_id, executor_id, content)
		VALUES ($1, $2, 'invalid document')
	`, uuid.New(), userID)
	expectExecError(t, sqlDB, `
		INSERT INTO assignments (document_id, executor_id, content)
		VALUES ($1, $2, 'invalid executor')
	`, docID, uuid.New())
	expectExecError(t, sqlDB, `
		INSERT INTO acknowledgments (id, document_id, creator_id, content)
		VALUES ($1, $2, $3, 'invalid document')
	`, uuid.New(), uuid.New(), userID)
	expectExecError(t, sqlDB, `
		INSERT INTO attachments (document_id, filename, file_size, storage_path, uploaded_by)
		VALUES ($1, 'invalid.pdf', 1, 'invalid.pdf', $2)
	`, docID, uuid.New())

	ackID := uuid.New()
	execSQL(t, sqlDB, `
		INSERT INTO acknowledgments (id, document_id, creator_id, content)
		VALUES ($1, $2, $3, 'ack')
	`, ackID, docID, userID)
	execSQL(t, sqlDB, `
		INSERT INTO acknowledgment_users (id, acknowledgment_id, user_id)
		VALUES ($1, $2, $3)
	`, uuid.New(), ackID, userID)
	expectExecError(t, sqlDB, `
		INSERT INTO acknowledgment_users (id, acknowledgment_id, user_id)
		VALUES ($1, $2, $3)
	`, uuid.New(), ackID, userID)

	execSQL(t, sqlDB, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT NOT NULL PRIMARY KEY,
			dirty BOOLEAN NOT NULL
		)
	`)
	execSQL(t, sqlDB, `DELETE FROM schema_migrations`)
	execSQL(t, sqlDB, `INSERT INTO schema_migrations (version, dirty) VALUES (10, true)`)
	db := &database.DB{DB: sqlDB}
	if err := db.CheckMigrationCompatibility(database.DefaultMigrationsPath); err == nil {
		t.Fatalf("expected dirty migration state to be rejected")
	}
}

func TestValidateIntegrationDSNRequiresSafeDatabaseName(t *testing.T) {
	for _, dsn := range []string{
		"postgres://user:pass@localhost:5432/docflow_test_123?sslmode=disable",
		"host=localhost port=5432 user=user password=pass dbname=docflow_regression sslmode=disable",
	} {
		if err := validateIntegrationDSN(dsn); err != nil {
			t.Fatalf("expected safe DSN %q, got error: %v", dsn, err)
		}
	}

	for _, dsn := range []string{
		"postgres://user:pass@localhost:5432/docflow?sslmode=disable",
		"host=localhost port=5432 user=user password=pass dbname=postgres sslmode=disable",
		"host=localhost port=5432 user=user password=pass sslmode=disable",
	} {
		if err := validateIntegrationDSN(dsn); err == nil {
			t.Fatalf("expected unsafe DSN %q to be rejected", dsn)
		}
	}
}

func openSafeIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DOCFLOW_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("set DOCFLOW_INTEGRATION_DSN to run PostgreSQL integration test")
	}
	if err := validateIntegrationDSN(dsn); err != nil {
		t.Fatalf("unsafe DOCFLOW_INTEGRATION_DSN: %v", err)
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}

	if err := applyIntegrationMigrations(sqlDB); err != nil {
		sqlDB.Close()
		t.Fatalf("apply migrations: %v", err)
	}
	return sqlDB
}

func validateIntegrationDSN(dsn string) error {
	dbName, err := integrationDBName(dsn)
	if err != nil {
		return err
	}
	if isSafeIntegrationDBName(dbName) {
		return nil
	}
	return fmt.Errorf("database name %q must start with docflow_test or docflow_regression", dbName)
}

func integrationDBName(dsn string) (string, error) {
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		name := strings.TrimPrefix(u.Path, "/")
		if name == "" {
			return "", fmt.Errorf("database name is empty")
		}
		return name, nil
	}

	for _, match := range regexp.MustCompile(`(?:^|\s)dbname=('[^']*'|"[^"]*"|[^\s]+)`).FindAllStringSubmatch(dsn, -1) {
		if len(match) == 2 {
			return strings.Trim(match[1], `'"`), nil
		}
	}
	return "", fmt.Errorf("database name not found")
}

func isSafeIntegrationDBName(name string) bool {
	return strings.HasPrefix(name, "docflow_test") || strings.HasPrefix(name, "docflow_regression")
}

func applyIntegrationMigrations(db *sql.DB) error {
	if _, err := db.Exec(`DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		return err
	}

	files, err := filepath.Glob("../../internal/database/migrations/*.up.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return err
		}
	}
	return nil
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %s: %v", strings.TrimSpace(query), err)
	}
}

func expectExecError(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err == nil {
		t.Fatalf("expected query to fail: %s", strings.TrimSpace(query))
	}
}

func insertOrganization(t *testing.T, db *sql.DB, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	execSQL(t, db, `
		INSERT INTO organizations (id, name)
		VALUES ($1, $2)
	`, id, name)
	return id
}

func assertScalar(t *testing.T, db *sql.DB, query string, args []any, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("query scalar %s: %v", strings.TrimSpace(query), err)
	}
	if got != want {
		t.Fatalf("scalar %s = %d, want %d", strings.TrimSpace(query), got, want)
	}
}
