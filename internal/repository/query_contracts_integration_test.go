package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/testutil/integrationdb"
)

func TestDocumentListAccessScopeIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	owner := insertIntegrationUser(t, sqlDB, "scope_owner")
	other := insertIntegrationUser(t, sqlDB, "scope_other")
	allowedNom, deniedNom := uuid.New(), uuid.New()
	for _, item := range []struct {
		id    uuid.UUID
		index string
	}{{allowedNom, "AL"}, {deniedNom, "DN"}} {
		execSQL(t, sqlDB, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode) VALUES ($1, $2, $3, 2026, 'outgoing_letter', '/', 'index_and_number')`, item.id, item.index, item.index)
	}
	org := uuid.New()
	execSQL(t, sqlDB, `INSERT INTO organizations (id, name) VALUES ($1, 'Scope organization')`, org)
	insertScopedOutgoing(t, sqlDB, allowedNom, owner, org, "AL/1", "allowed searchable")
	insertScopedOutgoing(t, sqlDB, deniedNom, other, org, "DN/1", "denied searchable")

	items, err := NewOutgoingDocumentRepository(db).GetList(models.OutgoingDocumentFilter{AllowedNomenclatureIDs: []string{allowedNom.String()}, Search: "searchable", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatalf("get scoped outgoing list: %v", err)
	}
	if items.TotalCount != 1 || len(items.Items) != 1 || items.Items[0].OutgoingNumber != "AL/1" {
		t.Fatalf("scope leaked or hid data: %+v", items)
	}

	// The same document is visible via a real assignment access edge, but not
	// to an unrelated user. This protects the helper queries used by dashboard.
	var documentID uuid.UUID
	if err := sqlDB.QueryRow(`SELECT id FROM documents WHERE registration_number = 'AL/1'`).Scan(&documentID); err != nil {
		t.Fatal(err)
	}
	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status) VALUES ($1, $2, 'scope', CURRENT_DATE + 1, 'new')`, documentID, owner)
	dashboard := NewDashboardRepository(db)
	visible, err := dashboard.GetExpiringAssignments(models.DashboardAssignmentFilter{Days: 2, AccessibleByUserIDs: []string{owner.String()}})
	if err != nil || len(visible) != 1 {
		t.Fatalf("owner dashboard visibility=%d err=%v", len(visible), err)
	}
	hidden, err := dashboard.GetExpiringAssignments(models.DashboardAssignmentFilter{Days: 2, AccessibleByUserIDs: []string{other.String()}})
	if err != nil || len(hidden) != 0 {
		t.Fatalf("unrelated dashboard visibility=%d err=%v", len(hidden), err)
	}
}

func TestLinkGraphAndOutboxLifecycleIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	user, root := seedIntegrationDocument(t, sqlDB)
	nom := uuid.New()
	execSQL(t, sqlDB, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode) VALUES ($1, 'Graph', 'GR', 2026, 'outgoing_letter', '/', 'index_and_number')`, nom)
	ids := []uuid.UUID{root, uuid.New(), uuid.New(), uuid.New()}
	for i := 1; i < len(ids); i++ {
		execSQL(t, sqlDB, `INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by) VALUES ($1, 'outgoing_letter', $2, $3, $4, CURRENT_DATE, $5, 'graph', 1, $6)`, ids[i], nom, uuid.New(), "GR/"+string(rune('0'+i)), models.DocumentTypeLetter, user)
	}
	links := NewLinkRepository(db)
	for _, pair := range [][2]int{{0, 1}, {1, 2}, {2, 0}, {2, 3}} {
		if err := links.Create(context.Background(), &models.DocumentLink{SourceID: ids[pair[0]], TargetID: ids[pair[1]], LinkType: "related", CreatedBy: user}); err != nil {
			t.Fatalf("create graph edge: %v", err)
		}
	}
	graph, err := links.GetGraph(context.Background(), root)
	if err != nil {
		t.Fatalf("get graph: %v", err)
	}
	seen := map[uuid.UUID]struct{}{}
	for _, edge := range graph {
		if _, ok := seen[edge.ID]; ok {
			t.Fatalf("duplicate graph edge %s", edge.ID)
		}
		seen[edge.ID] = struct{}{}
	}
	if len(graph) != 4 {
		t.Fatalf("graph edges=%d, want 4", len(graph))
	}

	outbox := NewOutboxRepository(db)
	event := models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "lifecycle", Payload: `{}`}
	if err := outbox.Enqueue(event); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := outbox.Enqueue(event); err != nil {
		t.Fatalf("deduplicated enqueue: %v", err)
	}
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM event_outbox WHERE deduplication_key = 'lifecycle'`, nil, 1)
	claimed, err := outbox.ClaimPending(1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim pending: count=%d err=%v", len(claimed), err)
	}
	if err := outbox.MarkFailed(claimed[0].ID, 1, 0, 1, "terminal"); err != nil {
		t.Fatalf("mark terminal failure: %v", err)
	}
	if again, err := outbox.ClaimPending(1); err != nil || len(again) != 0 {
		t.Fatalf("terminal event claim=%d err=%v", len(again), err)
	}
	if err := outbox.Requeue(claimed[0].ID); err != nil {
		t.Fatalf("requeue: %v", err)
	}
	claimed, err = outbox.ClaimPending(1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim requeued: count=%d err=%v", len(claimed), err)
	}
	if err := outbox.ReleaseStaleClaims(time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("release stale claim: %v", err)
	}
	claimed, err = outbox.ClaimPending(1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim released stale event: count=%d err=%v", len(claimed), err)
	}
}

func TestUserSubstitutionAndReferenceMergeIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := &database.DB{DB: sqlDB}
	principal := insertIntegrationUser(t, sqlDB, "principal")
	substitute := insertIntegrationUser(t, sqlDB, "substitute")
	repo := NewUserSubstitutionRepository(db)
	if _, err := repo.ReplaceForPrincipal(principal, &substitute, nil, nil, true, nil); err != nil {
		t.Fatalf("set substitution: %v", err)
	}
	active, err := repo.IsActiveSubstitute(substitute, principal)
	if err != nil || !active {
		t.Fatalf("active substitution=%v err=%v", active, err)
	}
	if _, err := repo.ReplaceForPrincipal(principal, &principal, nil, nil, true, nil); err == nil {
		t.Fatal("self substitution accepted")
	}

	_, documentID := seedIntegrationDocument(t, sqlDB)
	source, target := uuid.New(), uuid.New()
	execSQL(t, sqlDB, `INSERT INTO organizations (id, name) VALUES ($1, 'Source'), ($2, 'Target')`, source, target)
	execSQL(t, sqlDB, `UPDATE outgoing_document_details SET recipient_org_id = $1 WHERE document_id = $2`, source, documentID)
	refs := NewReferenceRepository(db)
	if err := refs.MergeOrganizations(source, target); err != nil {
		t.Fatalf("merge organizations: %v", err)
	}
	var recipient uuid.UUID
	if err := sqlDB.QueryRow(`SELECT recipient_org_id FROM outgoing_document_details WHERE document_id = $1`, documentID).Scan(&recipient); err != nil || recipient != target {
		t.Fatalf("merged recipient=%s err=%v", recipient, err)
	}
	if err := sqlDB.QueryRow(`SELECT id FROM organizations WHERE id = $1`, source).Scan(new(uuid.UUID)); err != sql.ErrNoRows {
		t.Fatalf("source organization remains, err=%v", err)
	}
}

func insertScopedOutgoing(t *testing.T, db *sql.DB, nom, creator, org uuid.UUID, number, content string) {
	t.Helper()
	id := uuid.New()
	execSQL(t, db, `INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number, registration_date, document_type, content, pages_count, created_by) VALUES ($1, 'outgoing_letter', $2, $3, $4, CURRENT_DATE, $5, $6, 1, $7)`, id, nom, uuid.New(), number, models.DocumentTypeLetter, content, creator)
	execSQL(t, db, `INSERT INTO outgoing_document_details (document_id, outgoing_number, outgoing_date, sender_signatory, sender_executor, recipient_org_id, addressee) VALUES ($1, $2, CURRENT_DATE, 'Signer', 'Executor', $3, 'Addressee')`, id, number, org)
}
