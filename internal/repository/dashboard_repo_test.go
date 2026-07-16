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

	query := `SELECT a.id, a.content, a.deadline, a.status, a.document_id, d.kind, u.full_name as executor_name, d.registration_number as doc_number FROM assignments a JOIN documents d ON d.id = a.document_id LEFT JOIN users u ON a.executor_id = u.id WHERE a.status IN \('new', 'in_progress'\) AND a.deadline BETWEEN CURRENT_DATE AND \(CURRENT_DATE \+ INTERVAL '1 day' \* \$1\) AND \(d.kind = ANY\(\$2\) OR \(a.executor_id = ANY\(\$3::uuid\[\]\) OR EXISTS \(SELECT 1 FROM assignment_co_executors ce WHERE ce.assignment_id = a.id AND ce.user_id = ANY\(\$3::uuid\[\]\)\)\)\) ORDER BY a.deadline ASC`

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
