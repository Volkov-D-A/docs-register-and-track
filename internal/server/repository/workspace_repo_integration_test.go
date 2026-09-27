package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestWorkspaceCountsAndPreviewsRespectModesAndDocumentScope(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	owner, firstDocument := seedIntegrationDocument(t, sqlDB)
	other := insertIntegrationUser(t, sqlDB, "workspace_other")
	secondNom, secondDocument := uuid.New(), uuid.New()
	execSQL(t, sqlDB, `INSERT INTO nomenclature (id, name, index, year, kind_code, separator, numbering_mode)
		VALUES ($1, 'Other', 'OT', 2026, 'outgoing_letter', '/', 'index_and_number')`, secondNom)
	execSQL(t, sqlDB, `INSERT INTO documents (id, kind, nomenclature_id, idempotency_key, registration_number,
		registration_date, document_type, content, pages_count, created_by)
		SELECT $1, kind, $2, $3, 'OT/1', registration_date, document_type, 'other document', pages_count, created_by
		FROM documents WHERE id = $4`, secondDocument, secondNom, uuid.New(), firstDocument)

	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status)
		VALUES ($1, $2, 'late', CURRENT_DATE - 1, 'new')`, firstDocument, owner)
	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status)
		VALUES ($1, $2, 'soon', CURRENT_DATE + 2, 'in_progress')`, firstDocument, owner)
	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status)
		VALUES ($1, $2, 'accept', CURRENT_DATE - 1, 'completed')`, secondDocument, owner)
	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status)
		VALUES ($1, $2, 'unrelated', CURRENT_DATE, 'new')`, secondDocument, other)
	ackOwn, ackOther, ackSecond := uuid.New(), uuid.New(), uuid.New()
	for _, item := range []struct {
		id, document, recipient uuid.UUID
	}{{ackOwn, firstDocument, owner}, {ackOther, firstDocument, other}, {ackSecond, secondDocument, owner}} {
		execSQL(t, sqlDB, `INSERT INTO acknowledgments (id, document_id, creator_id, content) VALUES ($1, $2, $3, 'read')`, item.id, item.document, owner)
		execSQL(t, sqlDB, `INSERT INTO acknowledgment_users (id, acknowledgment_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), item.id, item.recipient)
	}

	repo := NewWorkspaceRepository(database.Wrap(sqlDB))
	personal := models.WorkspaceQuery{Mode: models.WorkspaceModeExecution, SubjectIDs: []string{owner.String()}, Limit: 5}
	counts, items, err := repo.AssignmentSummary(personal)
	require.NoError(t, err)
	require.Equal(t, models.WorkspaceAssignmentCounts{New: 1, Overdue: 1, DueSoon: 1}, counts)
	require.Len(t, items, 2)
	require.Equal(t, "late", items[0].Content)
	ackCount, ackItems, err := repo.AcknowledgmentSummary(personal)
	require.NoError(t, err)
	require.Equal(t, 2, ackCount)
	require.Len(t, ackItems, 2)

	var firstNom uuid.UUID
	require.NoError(t, sqlDB.QueryRow(`SELECT nomenclature_id FROM documents WHERE id = $1`, firstDocument).Scan(&firstNom))
	controlled := models.WorkspaceQuery{Mode: models.WorkspaceModeControl, Kind: models.DocumentKindOutgoingLetter,
		ReadScope: models.DocumentAccessScope{Restricted: true, AllowedNomenclatureIDs: []string{firstNom.String()}}, Limit: 5}
	counts, items, err = repo.AssignmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, models.WorkspaceAssignmentCounts{New: 1, Overdue: 1, DueSoon: 1}, counts)
	require.Len(t, items, 2)
	ackCount, _, err = repo.AcknowledgmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, 2, ackCount)

	assignments := NewAssignmentRepository(database.Wrap(sqlDB))
	list, err := assignments.GetList(models.AssignmentFilter{
		Mode: models.WorkspaceModeControl, Metric: "overdue", Page: 1, PageSize: 10,
		ControlScopes: map[models.DocumentKind]models.DocumentAccessScope{
			models.DocumentKindOutgoingLetter: controlled.ReadScope,
		},
	})
	require.NoError(t, err)
	require.Equal(t, counts.Overdue, list.TotalCount)
	require.Equal(t, "late", list.Items[0].Content)
	list, err = assignments.GetList(models.AssignmentFilter{
		Mode: models.WorkspaceModeExecution, Metric: "new", Page: 1, PageSize: 10,
		AccessibleByUserIDs: []string{owner.String()},
	})
	require.NoError(t, err)
	require.Equal(t, 1, list.TotalCount)
	require.Equal(t, "late", list.Items[0].Content)

	page, err := repo.ListAcknowledgments([]models.WorkspaceQuery{personal}, 1, 1)
	require.NoError(t, err)
	require.Equal(t, 2, page.TotalCount)
	require.Len(t, page.Items, 1)
	page, err = repo.ListAcknowledgments([]models.WorkspaceQuery{controlled}, 1, 10)
	require.NoError(t, err)
	require.Equal(t, 2, page.TotalCount)
}
