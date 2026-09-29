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
	execSQL(t, sqlDB, `UPDATE documents SET registration_date = '2026-09-28', content = 'Document content' WHERE id = $1`, firstDocument)
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
	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status)
        VALUES ($1, $2, 'redo', CURRENT_DATE + 10, 'returned')`, firstDocument, owner)
	ackOwn, ackOther, ackSecond := uuid.New(), uuid.New(), uuid.New()
	for _, item := range []struct {
		id, document, recipient uuid.UUID
	}{{ackOwn, firstDocument, owner}, {ackOther, firstDocument, other}, {ackSecond, secondDocument, owner}} {
		execSQL(t, sqlDB, `INSERT INTO assignments (id, document_id, creator_id, content, type) VALUES ($1, $2, $3, 'read', 'acknowledgment')`, item.id, item.document, owner)
		execSQL(t, sqlDB, `INSERT INTO assignment_recipients (id, assignment_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), item.id, item.recipient)
	}

	repo := NewWorkspaceRepository(database.Wrap(sqlDB))
	personal := models.WorkspaceQuery{Mode: models.WorkspaceModeExecution, SubjectIDs: []string{owner.String()}, Limit: 5}
	counts, items, err := repo.AssignmentSummary(personal)
	require.NoError(t, err)
	require.Equal(t, models.WorkspaceAssignmentCounts{New: 3, InProgress: 1, Returned: 1, Overdue: 1, DueSoon: 1}, counts)
	require.Len(t, items, 5)
	require.Equal(t, "late", items[0].Content)
	require.Equal(t, "IT/1", items[0].DocumentNumber)
	require.False(t, items[0].DocumentDate.IsZero())
	for _, item := range items {
		if item.Type != models.AssignmentTypeAcknowledgment {
			continue
		}
		require.Equal(t, "2026-09-28", item.DocumentDate.Format("2006-01-02"))
		require.Empty(t, item.Content)
		if item.DocumentID == firstDocument {
			require.Equal(t, "Document content", item.DocumentContent)
			require.Equal(t, "IT/1", item.DocumentNumber)
		} else {
			require.Equal(t, "other document", item.DocumentContent)
			require.Equal(t, "OT/1", item.DocumentNumber)
		}
	}

	var firstNom uuid.UUID
	require.NoError(t, sqlDB.QueryRow(`SELECT nomenclature_id FROM documents WHERE id = $1`, firstDocument).Scan(&firstNom))
	controlled := models.WorkspaceQuery{Mode: models.WorkspaceModeControl, Kind: models.DocumentKindOutgoingLetter,
		ReadScope: models.DocumentAccessScope{Restricted: true, AllowedNomenclatureIDs: []string{firstNom.String()}}, Limit: 5}
	counts, items, err = repo.AssignmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, models.WorkspaceAssignmentCounts{New: 3, InProgress: 1, Returned: 1, Overdue: 1, DueSoon: 1}, counts)
	require.Len(t, items, 5)

	assignments := NewAssignmentRepository(database.Wrap(sqlDB))
	list, err := assignments.GetList(models.AssignmentFilter{
		Mode: models.WorkspaceModeControl, OverdueOnly: true, Page: 1, PageSize: 10,
		ControlScopes: map[models.DocumentKind]models.DocumentAccessScope{
			models.DocumentKindOutgoingLetter: controlled.ReadScope,
		},
	})
	require.NoError(t, err)
	require.Equal(t, counts.Overdue, list.TotalCount)
	require.Equal(t, "late", list.Items[0].Content)
	list, err = assignments.GetList(models.AssignmentFilter{
		Mode: models.WorkspaceModeControl, Statuses: []string{"in_progress"}, Page: 1, PageSize: 10,
		ControlScopes: map[models.DocumentKind]models.DocumentAccessScope{
			models.DocumentKindOutgoingLetter: controlled.ReadScope,
		},
	})
	require.NoError(t, err)
	require.Equal(t, counts.InProgress, list.TotalCount)
	require.Equal(t, "soon", list.Items[0].Content)
	list, err = assignments.GetList(models.AssignmentFilter{
		Mode: models.WorkspaceModeExecution, Types: []string{}, Statuses: []string{"new"}, Page: 1, PageSize: 10,
		AccessibleByUserIDs: []string{owner.String()},
	})
	require.NoError(t, err)
	require.Equal(t, 3, list.TotalCount)
	require.Len(t, list.Items, 3)

	list, err = assignments.GetList(models.AssignmentFilter{
		Mode: models.WorkspaceModeExecution, Statuses: []string{"in_progress"}, Page: 1, PageSize: 10,
		AccessibleByUserIDs: []string{owner.String()},
	})
	require.NoError(t, err)
	require.Equal(t, 1, list.TotalCount)
	require.Equal(t, "soon", list.Items[0].Content)

	// Multiple values form a union within each group, with access applied to both types.
	multiFilter := models.AssignmentFilter{
		Types:    []string{models.AssignmentTypeExecution, models.AssignmentTypeAcknowledgment},
		Statuses: []string{"new", "in_progress", "finished"},
		Mode:     models.WorkspaceModeExecution, Page: 1, PageSize: 20,
		AccessibleByUserIDs: []string{owner.String()},
	}
	list, err = assignments.GetList(multiFilter)
	require.NoError(t, err)
	require.Equal(t, 4, list.TotalCount)
	for _, item := range list.Items {
		require.Contains(t, []string{"new", "in_progress"}, item.Status)
		require.NotEqual(t, ackOther, item.ID)
	}
	multiFilter.Types = []string{models.AssignmentTypeAcknowledgment}
	list, err = assignments.GetList(multiFilter)
	require.NoError(t, err)
	require.Equal(t, 2, list.TotalCount)
	multiFilter.Types = []string{models.AssignmentTypeExecution}
	list, err = assignments.GetList(multiFilter)
	require.NoError(t, err)
	require.Equal(t, 2, list.TotalCount)
	multiFilter.Types = nil
	multiFilter.Statuses = nil
	list, err = assignments.GetList(multiFilter)
	require.NoError(t, err)
	require.Equal(t, 5, list.TotalCount)
	multiFilter.Types = []string{models.AssignmentTypeExecution, models.AssignmentTypeAcknowledgment}
	multiFilter.Statuses = []string{"new", "in_progress"}
	multiFilter.Mode = models.WorkspaceModeControl
	multiFilter.ControlScopes = map[models.DocumentKind]models.DocumentAccessScope{
		models.DocumentKindOutgoingLetter: controlled.ReadScope,
	}
	list, err = assignments.GetList(multiFilter)
	require.NoError(t, err)
	require.Equal(t, 4, list.TotalCount)
	for _, item := range list.Items {
		require.Equal(t, firstDocument, item.DocumentID)
	}

	// A due acknowledgment participates in the common overdue metric.
	execSQL(t, sqlDB, `UPDATE assignments SET deadline = CURRENT_DATE - 2 WHERE id = $1`, ackOwn)
	counts, items, err = repo.AssignmentSummary(personal)
	require.NoError(t, err)
	require.Equal(t, 2, counts.Overdue)
	require.Equal(t, ackOwn, items[0].ID)
	list, err = assignments.GetList(models.AssignmentFilter{
		Types: []string{}, Mode: models.WorkspaceModeExecution, OverdueOnly: true, Page: 1, PageSize: 10,
		AccessibleByUserIDs: []string{owner.String()},
	})
	require.NoError(t, err)
	require.Equal(t, counts.Overdue, list.TotalCount)

	// A personally confirmed task disappears from execution while remaining under control.
	execSQL(t, sqlDB, `INSERT INTO assignment_recipients (id, assignment_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), ackOwn, other)
	execSQL(t, sqlDB, `UPDATE assignment_recipients SET confirmed_at = NOW(), confirmed_by = $1 WHERE assignment_id = $2 AND user_id = $1`, owner, ackOwn)
	counts, items, err = repo.AssignmentSummary(personal)
	require.NoError(t, err)
	require.Equal(t, 2, counts.New)
	for _, item := range items {
		require.NotEqual(t, ackOwn, item.ID)
	}
	counts, _, err = repo.AssignmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, 3, counts.New)

	execSQL(t, sqlDB, `UPDATE assignments SET status = 'finished', completed_at = NOW() WHERE id = $1`, ackOwn)
	counts, _, err = repo.AssignmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, 2, counts.New)

	// Returned work is counted even with a distant deadline, and the normal status filter agrees.
	counts, _, err = repo.AssignmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, 1, counts.Returned)
	controlScopes := map[models.DocumentKind]models.DocumentAccessScope{models.DocumentKindOutgoingLetter: controlled.ReadScope}
	list, err = assignments.GetList(models.AssignmentFilter{
		Types: []string{}, Mode: models.WorkspaceModeControl, Statuses: []string{"returned"}, Page: 1, PageSize: 10, ControlScopes: controlScopes,
	})
	require.NoError(t, err)
	require.Equal(t, counts.Returned, list.TotalCount)
	require.Equal(t, "redo", list.Items[0].Content)

	execSQL(t, sqlDB, `INSERT INTO assignments (document_id, executor_id, content, deadline, status)
        VALUES ($1, $2, 'accept soon', CURRENT_DATE + 2, 'completed'),
        ($1, $2, 'accept late', CURRENT_DATE - 1, 'completed'),
        ($1, $2, 'last second', (CURRENT_DATE + 3)::timestamp + INTERVAL '23 hours 59 minutes 59.5 seconds', 'new')`, firstDocument, owner)
	counts, _, err = repo.AssignmentSummary(controlled)
	require.NoError(t, err)
	require.Equal(t, 3, counts.DueSoon)
	require.Equal(t, 2, counts.AwaitingAcceptance)
	var dateFrom, dateTo string
	require.NoError(t, sqlDB.QueryRow(`SELECT CURRENT_DATE::text, (CURRENT_DATE + 3)::text`).Scan(&dateFrom, &dateTo))
	list, err = assignments.GetList(models.AssignmentFilter{
		Types: []string{}, Mode: models.WorkspaceModeControl, DateFrom: dateFrom, DateTo: dateTo, Page: 1, PageSize: 10, ControlScopes: controlScopes,
	})
	require.NoError(t, err)
	require.Equal(t, counts.DueSoon, list.TotalCount)
	list, err = assignments.GetList(models.AssignmentFilter{
		Types: []string{}, Mode: models.WorkspaceModeControl, Statuses: []string{"completed"}, Page: 1, PageSize: 10, ControlScopes: controlScopes,
	})
	require.NoError(t, err)
	require.Equal(t, counts.AwaitingAcceptance, list.TotalCount)
	list, err = assignments.GetList(models.AssignmentFilter{
		Types: []string{}, Mode: models.WorkspaceModeControl, OverdueOnly: true, Page: 1, PageSize: 10, ControlScopes: controlScopes,
	})
	require.NoError(t, err)
	require.Equal(t, counts.Overdue, list.TotalCount)

}
