package repository

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceRecentDocumentsIntegration(t *testing.T) {
	db := integrationdb.Open(t)
	user, seed := seedIntegrationDocument(t, db)
	execSQL(t, db, `UPDATE documents SET created_at = '2026-09-01', updated_at = '2026-10-01' WHERE id = $1`, seed)
	incoming := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	outgoing := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	order := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	appeal := uuid.MustParse("40000000-0000-0000-0000-000000000001")
	older := uuid.MustParse("50000000-0000-0000-0000-000000000001")
	nomenclatures := map[uuid.UUID]uuid.UUID{}
	insert := func(id uuid.UUID, kind, number, documentDate, registeredAt string) {
		nom := uuid.New()
		nomenclatures[id] = nom
		execSQL(t, db, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
            VALUES ($1, 'Recent', $2, 2026, $3, '/', 'index_and_number')`, nom, number, kind)
		execSQL(t, db, `INSERT INTO documents (id, kind, nomenclature_id, registration_number, registration_date,
            document_type, content, created_by, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, 'Письмо', 'Content fallback', $6, $7, '2026-10-01')`, id, kind, nom, number, documentDate, user, registeredAt)
	}
	insert(incoming, "incoming_letter", "125", "2026-09-20", "2026-09-24T12:00:00Z")
	insert(outgoing, "outgoing_letter", "84", "2026-09-19", "2026-09-24T12:00:00Z")
	insert(order, "administrative_order", "41", "2026-09-22", "2026-09-23T12:00:00Z")
	insert(appeal, "citizen_appeal", "543", "2026-09-21", "2026-09-22T12:00:00Z")
	insert(older, "incoming_letter", "126", "2026-09-30", "2026-09-21T12:00:00Z")
	alpha, beta := insertOrganization(t, db, "Alpha"), insertOrganization(t, db, "Beta")
	execSQL(t, db, `INSERT INTO incoming_document_details (document_id, incoming_number, incoming_date, sender_signatory)
        VALUES ($1, '125', '2026-09-20', 'Signer')`, incoming)
	execSQL(t, db, `INSERT INTO document_correspondent_registrations (document_id, registration_number, registration_date, correspondent_org_id, position)
        VALUES ($1, 'external-999', '2026-09-18', $2, 2), ($1, 'external-998', '2026-09-17', $3, 1)`, incoming, beta, alpha)
	execSQL(t, db, `INSERT INTO outgoing_document_details (document_id, outgoing_number, outgoing_date, sender_signatory, sender_executor, recipient_org_id, addressee)
        VALUES ($1, '84', '2026-09-19', 'Signer', 'Executor', $2, 'Addressee')`, outgoing, alpha)
	execSQL(t, db, `INSERT INTO administrative_order_details (document_id, order_number, order_date, title)
        VALUES ($1, '41', '2026-09-22', 'О назначении')`, order)
	execSQL(t, db, `INSERT INTO citizen_appeal_details (document_id, appeal_date, applicant_full_name, registration_address, appeal_type, applicant_category)
        VALUES ($1, '2026-09-15', 'Петров А.В.', 'Address', 'заявление', 'Category')`, appeal)
	repo := NewWorkspaceRepository(database.Wrap(db))
	all := map[models.DocumentKind]models.DocumentAccessScope{}
	for _, spec := range models.AllDocumentKindSpecs() {
		all[spec.Code] = models.DocumentAccessScope{}
	}
	items, err := repo.RecentDocuments(all)
	require.NoError(t, err)
	require.Len(t, items, 4)
	ids := func(items []models.WorkspaceDocument) []uuid.UUID {
		result := []uuid.UUID{}
		for _, item := range items {
			result = append(result, item.ID)
		}
		return result
	}
	require.Equal(t, []uuid.UUID{outgoing, incoming, order, appeal}, ids(items))
	require.Equal(t, "125", items[1].DocumentNumber)
	require.Equal(t, "2026-09-20", items[1].DocumentDate.Format("2006-01-02"))
	require.Equal(t, "2026-09-24", items[1].RegisteredAt.Format("2006-01-02"))
	require.Equal(t, []string{"Alpha", "Beta"}, items[1].Correspondents)
	require.Equal(t, "Alpha", items[1].Description)
	require.Equal(t, "Alpha", items[0].Description)
	require.Equal(t, "О назначении", items[2].Description)
	require.Equal(t, "Петров А.В.", items[3].Description)
	// Read edges, substitution subjects and nomenclature access are combined before LIMIT.
	other := insertIntegrationUser(t, db, "recent-other")
	assn := uuid.New()
	execSQL(t, db, `INSERT INTO assignments (id, document_id, executor_id, content, status) VALUES ($1, $2, $3, 'Read edge', 'new')`, assn, order, other)
	execSQL(t, db, `INSERT INTO assignment_co_executors (assignment_id, user_id) VALUES ($1, $2)`, assn, user)
	execSQL(t, db, `INSERT INTO assignments (document_id, executor_id, content, status) VALUES ($1, $2, 'Substitution edge', 'new')`, appeal, other)
	ack := uuid.New()
	execSQL(t, db, `INSERT INTO assignments (id, document_id, creator_id, content, type) VALUES ($1, $2, $3, 'Read', 'acknowledgment')`, ack, incoming, user)
	execSQL(t, db, `INSERT INTO assignment_recipients (id, assignment_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), ack, user)
	unrelated := uuid.New()
	insert(unrelated, "outgoing_letter", "85", "2026-09-25", "2026-09-25T12:00:00Z")
	scopes := map[models.DocumentKind]models.DocumentAccessScope{}
	for _, spec := range models.AllDocumentKindSpecs() {
		scopes[spec.Code] = models.DocumentAccessScope{Restricted: true, AccessibleByUserIDs: []string{user.String(), other.String()}, AllowedNomenclatureIDs: []string{nomenclatures[incoming].String(), nomenclatures[outgoing].String()}}
	}
	items, err = repo.RecentDocuments(scopes)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{outgoing, incoming, order, appeal}, ids(items))
	require.NotContains(t, ids(items), unrelated)
	// Same document has both a nomenclature grant and acknowledgment; it remains one row.
	require.Len(t, items, 4)
	items, err = repo.RecentDocuments(map[models.DocumentKind]models.DocumentAccessScope{models.DocumentKindOutgoingLetter: {Restricted: true}})
	require.NoError(t, err)
	require.Empty(t, items)
	items, err = repo.RecentDocuments(map[models.DocumentKind]models.DocumentAccessScope{models.DocumentKindIncomingLetter: {}})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{incoming, older}, ids(items))
	require.Equal(t, "Content fallback", items[1].Description)
	require.Empty(t, items[1].Correspondents)
}
