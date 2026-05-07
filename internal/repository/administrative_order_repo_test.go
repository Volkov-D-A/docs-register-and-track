package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestAdministrativeOrderRepository_GetListIncludesAcknowledgmentAccess(t *testing.T) {
	// При ограниченном доступе пользователь должен видеть приказ не только через поручение,
	// но и через задачу на ознакомление.
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
	docID := uuid.New()
	userID := uuid.New().String()
	now := time.Now()
	filter := models.DocumentFilter{
		AccessibleByUserID: userID,
		Page:               1,
		PageSize:           10,
	}

	expectedCountQuery := `SELECT COUNT\(\*\)(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id(.*)acknowledgment_users au(.*)JOIN acknowledgments a ON au.acknowledgment_id = a.id(.*)a.document_id = d.id`
	mock.ExpectQuery(expectedCountQuery).
		WithArgs(models.DocumentKindAdministrativeOrder, userID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"order_number", "order_date", "title",
		"execution_controller", "execution_deadline", "is_active", "cancelled_at",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01 — Приказы",
		"ПР-001", now, "О внутреннем регламенте",
		"", nil, true, nil,
		uuid.New(), "Регистратор",
		now, now,
	)

	expectedListQuery := `SELECT(.*)ord\.order_number(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id(.*)acknowledgment_users au(.*)JOIN acknowledgments a ON au.acknowledgment_id = a.id(.*)ORDER BY d.created_at DESC`
	mock.ExpectQuery(expectedListQuery).
		WithArgs(models.DocumentKindAdministrativeOrder, userID, userID, 10, 0).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
		WithArgs(docID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
			"acknowledged_by_name", "position", "created_at",
		}))

	res, err := repo.GetList(filter)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 1, res.TotalCount)
	require.Len(t, res.Items, 1)
	assert.Equal(t, docID, res.Items[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}
