package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestDashboardRepository_GetExpiringAssignments(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewDashboardRepository(&database.DB{DB: db})
	userID := uuid.New()
	now := time.Now()

	query := `SELECT(.*)FROM assignments a(.*)WHERE a.status IN \('new', 'in_progress'\)(.*)assignment_series(.*)ORDER BY a.deadline ASC`

	rows := sqlmock.NewRows([]string{
		"id", "content", "deadline", "status", "document_id", "kind", "executor_name", "doc_number",
	}).AddRow(uuid.New(), "Content", now, "new", uuid.New(), "incoming_letter", "Executor", "doc-1")

	mock.ExpectQuery(query).WithArgs(3, pq.Array([]string{"incoming_letter"}), pq.Array([]string{userID.String()})).WillReturnRows(rows)

	res, err := repo.GetExpiringAssignments(models.DashboardAssignmentFilter{
		Days:                 3,
		AllowedDocumentKinds: []string{"incoming_letter"},
		AccessibleByUserIDs:  []string{userID.String()},
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
