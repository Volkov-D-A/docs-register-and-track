package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestActiveAdministratorInvariantIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	adminA := insertIntegrationUser(t, sqlDB, "admin_a")
	adminB := insertIntegrationUser(t, sqlDB, "admin_b")
	execSQL(t, sqlDB, `INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, 'admin', true), ($2, 'admin', true)`, adminA, adminB)

	// A profile replacement may temporarily have no permission inside its own
	// transaction; the deferred trigger must inspect the state at COMMIT.
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatalf("begin replacement: %v", err)
	}
	if _, err = tx.Exec(`DELETE FROM user_system_permissions WHERE user_id = $1`, adminA); err != nil {
		t.Fatalf("clear permission: %v", err)
	}
	if _, err = tx.Exec(`INSERT INTO user_system_permissions (user_id, permission, is_allowed) VALUES ($1, 'admin', true)`, adminA); err != nil {
		t.Fatalf("restore permission: %v", err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatalf("commit valid replacement: %v", err)
	}

	// Two concurrent transactions cannot both leave the system without an admin.
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []uuid.UUID{adminA, adminB} {
		wg.Add(1)
		go func(id uuid.UUID) {
			defer wg.Done()
			tx, err := sqlDB.Begin()
			if err == nil {
				_, err = tx.Exec(`UPDATE users SET is_active = false WHERE id = $1`, id)
			}
			<-start
			if err == nil {
				err = tx.Commit()
			} else if tx != nil {
				_ = tx.Rollback()
			}
			errs <- err
		}(id)
	}
	close(start)
	wg.Wait()
	close(errs)
	failures := 0
	for err := range errs {
		if err != nil {
			failures++
		}
	}
	if failures != 1 {
		t.Fatalf("concurrent administrator deactivation failures=%d, want 1", failures)
	}
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM users u JOIN user_system_permissions p ON p.user_id = u.id WHERE u.is_active AND p.permission = 'admin' AND p.is_allowed`, nil, 1)
}

func seedIntegrationDocument(t *testing.T, db *sql.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID := insertIntegrationUser(t, db, "workflow_user")
	nomID, docID := uuid.New(), uuid.New()
	orgID := uuid.New()
	execSQL(t, db, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode, next_number) VALUES ($1, 'Integration', 'IT', 2026, 'outgoing_letter', '/', 'index_and_number', 2)`, nomID)
	execSQL(t, db, `INSERT INTO organizations (id, name) VALUES ($1, 'Workflow Recipient')`, orgID)
	execSQL(t, db, `INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by) VALUES ($1, 'outgoing_letter', $2, $3, 'IT/1', CURRENT_DATE, $4, 'integration document', 1, $5)`, docID, nomID, uuid.New(), models.DocumentTypeLetter, userID)
	execSQL(t, db, `INSERT INTO outgoing_document_details (document_id, outgoing_number, outgoing_date, sender_signatory, sender_executor, recipient_org_id, addressee) VALUES ($1, 'IT/1', CURRENT_DATE, 'Signer', 'Executor', $2, 'Addressee')`, docID, orgID)
	return userID, docID
}

func insertIntegrationUser(t testing.TB, db *sql.DB, login string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	execSQL(t, db, `INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic) VALUES ($1, $2, 'hash', $3, 'User', TRUE)`, id, login, fmt.Sprintf("%s name", login))
	return id
}
