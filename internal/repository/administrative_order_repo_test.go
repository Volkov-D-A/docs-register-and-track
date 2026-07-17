package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
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
	docID2 := uuid.New()
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
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

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
	).AddRow(
		docID2, uuid.New(), "01-01 — Приказы",
		"ПР-002", now, "О втором регламенте",
		"", nil, true, nil,
		uuid.New(), "Регистратор",
		now, now,
	)

	expectedListQuery := `SELECT(.*)ord\.order_number(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id(.*)acknowledgment_users au(.*)JOIN acknowledgments a ON au.acknowledgment_id = a.id(.*)ORDER BY d.created_at DESC`
	mock.ExpectQuery(expectedListQuery).
		WithArgs(models.DocumentKindAdministrativeOrder, userID, userID, 10, 0).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = ANY\(\$1\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
			"acknowledged_by_name", "position", "created_at",
		}))

	res, err := repo.GetList(filter)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 2, res.TotalCount)
	require.Len(t, res.Items, 2)
	assert.Equal(t, docID, res.Items[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdministrativeOrderRepository_GetListFiltersAndErrors(t *testing.T) {
	t.Run("success with broad filters and pagination limits", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		filter := models.DocumentFilter{
			AccessibleByUserID:        uuid.New().String(),
			AllowedNomenclatureIDs:    []string{uuid.New().String()},
			NomenclatureIDs:           []string{uuid.New().String(), uuid.New().String()},
			DateFrom:                  "2026-01-01",
			DateTo:                    "2026-12-31",
			Search:                    "регламент",
			OrderNumber:               "ПР",
			ExecutionController:       "Контроль",
			OnlyPendingAcknowledgment: true,
			OrderActiveStatus:         "active",
			Page:                      -1,
			PageSize:                  500,
		}

		mock.ExpectQuery(`SELECT COUNT\(\*\)(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nomenclature_id", "nomenclature_name",
				"order_number", "order_date", "title",
				"execution_controller", "execution_deadline", "is_active", "cancelled_at",
				"created_by", "created_by_name",
				"created_at", "updated_at",
			}))

		res, err := repo.GetList(filter)

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Items)
		assert.Equal(t, 0, res.TotalCount)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 100, res.PageSize)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with inactive status and defaults", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})

		mock.ExpectQuery(`SELECT COUNT\(\*\)(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nomenclature_id", "nomenclature_name",
				"order_number", "order_date", "title",
				"execution_controller", "execution_deadline", "is_active", "cancelled_at",
				"created_by", "created_by_name",
				"created_at", "updated_at",
			}))

		res, err := repo.GetList(models.DocumentFilter{OrderActiveStatus: "inactive"})

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Items)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count database error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		mock.ExpectQuery(`SELECT COUNT\(\*\)(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(models.DocumentFilter{Search: "test"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count administrative orders")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("data database error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		mock.ExpectQuery(`SELECT COUNT\(\*\)(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)FROM documents d(.*)JOIN administrative_order_details ord ON ord.document_id = d.id`).
			WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(models.DocumentFilter{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get administrative orders")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func administrativeOrderAcknowledgmentPeopleRows(documentID uuid.UUID, now time.Time) *sqlmock.Rows {
	acknowledgedAt := now.Add(time.Hour)
	acknowledgedBy := uuid.New()
	return sqlmock.NewRows([]string{
		"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
		"acknowledged_by_name", "position", "created_at",
	}).AddRow(
		uuid.New(), documentID, "Иванов И.И.", acknowledgedAt, acknowledgedBy,
		"Секретарь", 1, now,
	).AddRow(
		uuid.New(), documentID, "Петров П.П.", nil, nil,
		"", 2, now,
	)
}

func TestAdministrativeOrderRepository_GetAcknowledgmentPeople(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
	documentID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
		WithArgs(documentID).
		WillReturnRows(administrativeOrderAcknowledgmentPeopleRows(documentID, now))

	people, err := repo.GetAcknowledgmentPeople(documentID)

	require.NoError(t, err)
	require.Len(t, people, 2)
	assert.Equal(t, "Иванов И.И.", people[0].FullName)
	assert.NotNil(t, people[0].AcknowledgedAt)
	assert.NotNil(t, people[0].AcknowledgedBy)
	assert.Equal(t, "Петров П.П.", people[1].FullName)
	assert.Nil(t, people[1].AcknowledgedAt)
	assert.Nil(t, people[1].AcknowledgedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdministrativeOrderRepository_GetAcknowledgmentPersonByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		personID := uuid.New()
		documentID := uuid.New()
		now := time.Now()
		acknowledgedAt := now.Add(time.Hour)
		acknowledgedBy := uuid.New()

		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.id = \$1`).
			WithArgs(personID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
				"acknowledged_by_name", "position", "created_at",
			}).AddRow(personID, documentID, "Иванов И.И.", acknowledgedAt, acknowledgedBy, "Секретарь", 1, now))

		person, err := repo.GetAcknowledgmentPersonByID(personID)

		require.NoError(t, err)
		require.NotNil(t, person)
		assert.Equal(t, personID, person.ID)
		assert.Equal(t, documentID, person.DocumentID)
		assert.NotNil(t, person.AcknowledgedAt)
		assert.NotNil(t, person.AcknowledgedBy)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		personID := uuid.New()

		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.id = \$1`).
			WithArgs(personID).
			WillReturnError(sql.ErrNoRows)

		person, err := repo.GetAcknowledgmentPersonByID(personID)

		require.NoError(t, err)
		assert.Nil(t, person)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("fills missing acknowledgment fields from update result", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		personID := uuid.New()
		documentID := uuid.New()
		acknowledgedBy := uuid.New()
		now := time.Now()
		acknowledgedAt := now.Add(time.Hour)

		mock.ExpectQuery(`UPDATE administrative_order_acknowledgment_people`).
			WithArgs(personID, acknowledgedBy).
			WillReturnRows(sqlmock.NewRows([]string{"document_id", "acknowledged_at"}).AddRow(documentID, acknowledgedAt))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.id = \$1`).
			WithArgs(personID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
				"acknowledged_by_name", "position", "created_at",
			}).AddRow(personID, uuid.Nil, "Иванов И.И.", nil, nil, "", 1, now))

		person, err := repo.MarkAcknowledgmentPerson(personID, acknowledgedBy)

		require.NoError(t, err)
		require.NotNil(t, person)
		assert.Equal(t, documentID, person.DocumentID)
		require.NotNil(t, person.AcknowledgedAt)
		assert.Equal(t, acknowledgedAt, *person.AcknowledgedAt)
		require.NotNil(t, person.AcknowledgedBy)
		assert.Equal(t, acknowledgedBy, *person.AcknowledgedBy)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("person reload error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		personID := uuid.New()
		documentID := uuid.New()
		acknowledgedBy := uuid.New()
		acknowledgedAt := time.Now()

		mock.ExpectQuery(`UPDATE administrative_order_acknowledgment_people`).
			WithArgs(personID, acknowledgedBy).
			WillReturnRows(sqlmock.NewRows([]string{"document_id", "acknowledged_at"}).AddRow(documentID, acknowledgedAt))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.id = \$1`).
			WithArgs(personID).
			WillReturnError(sql.ErrConnDone)

		person, err := repo.MarkAcknowledgmentPerson(personID, acknowledgedBy)

		require.Error(t, err)
		assert.Nil(t, person)
		assert.Contains(t, err.Error(), "failed to get administrative order acknowledgment person")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdministrativeOrderRepository_MarkAcknowledgmentPerson(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		personID := uuid.New()
		documentID := uuid.New()
		acknowledgedBy := uuid.New()
		now := time.Now()
		acknowledgedAt := now.Add(time.Hour)

		mock.ExpectQuery(`UPDATE administrative_order_acknowledgment_people`).
			WithArgs(personID, acknowledgedBy).
			WillReturnRows(sqlmock.NewRows([]string{"document_id", "acknowledged_at"}).AddRow(documentID, acknowledgedAt))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.id = \$1`).
			WithArgs(personID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
				"acknowledged_by_name", "position", "created_at",
			}).AddRow(personID, documentID, "Иванов И.И.", acknowledgedAt, acknowledgedBy, "Секретарь", 1, now))

		person, err := repo.MarkAcknowledgmentPerson(personID, acknowledgedBy)

		require.NoError(t, err)
		require.NotNil(t, person)
		assert.Equal(t, personID, person.ID)
		assert.NotNil(t, person.AcknowledgedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		personID := uuid.New()
		acknowledgedBy := uuid.New()

		mock.ExpectQuery(`UPDATE administrative_order_acknowledgment_people`).
			WithArgs(personID, acknowledgedBy).
			WillReturnError(sql.ErrNoRows)

		person, err := repo.MarkAcknowledgmentPerson(personID, acknowledgedBy)

		require.NoError(t, err)
		assert.Nil(t, person)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdministrativeOrderRepositoryMarkAcknowledgmentPersonWithOutboxRollsBackOnEnqueueFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
	repo.SetOutbox(NewOutboxRepository(repo.db))
	personID, documentID, acknowledgedBy := uuid.New(), uuid.New(), uuid.New()
	event := models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "order-ack:" + personID.String(), Payload: `{"action":"ACK_CONFIRM"}`}

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE administrative_order_acknowledgment_people`).
		WithArgs(personID, acknowledgedBy).
		WillReturnRows(sqlmock.NewRows([]string{"document_id"}).AddRow(documentID))
	mock.ExpectExec(`INSERT INTO event_outbox`).WithArgs(event.EventType, event.DeduplicationKey, event.Payload).WillReturnError(assert.AnError)
	mock.ExpectRollback()

	_, err = repo.MarkAcknowledgmentPersonWithOutbox(personID, acknowledgedBy, []models.OutboxEvent{event})
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdministrativeOrderRepository_CancelByLink(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		cancelledAt := time.Now()

		mock.ExpectExec(`UPDATE documents d`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder, cancelledAt).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE administrative_order_details`).
			WithArgs(docID, cancelledAt).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = repo.CancelByLink(docID, cancelledAt)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("details update error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		cancelledAt := time.Now()

		mock.ExpectExec(`UPDATE documents d`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder, cancelledAt).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE administrative_order_details`).
			WithArgs(docID, cancelledAt).
			WillReturnError(sql.ErrConnDone)

		err = repo.CancelByLink(docID, cancelledAt)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cancel administrative order by link")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdministrativeOrderRepository_GetCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewAdministrativeOrderRepository(&database.DB{DB: db})

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents WHERE kind = \$1`).
		WithArgs(models.DocumentKindAdministrativeOrder).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(9))

	count, err := repo.GetCount()

	require.NoError(t, err)
	assert.Equal(t, 9, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func administrativeOrderRows(docID uuid.UUID, now time.Time) *sqlmock.Rows {
	deadline := now.AddDate(0, 0, 7)
	cancelledAt := now.AddDate(0, 0, 1)
	return sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"order_number", "order_date", "title",
		"execution_controller", "execution_deadline", "is_active", "cancelled_at",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "04-01 — Приказы",
		"ПР-002", now, "О внесении изменений",
		"Контрольный отдел", deadline, false, cancelledAt,
		uuid.New(), "Регистратор",
		now, now,
	)
}

func TestAdministrativeOrderRepository_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()

		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)WHERE d.id = \$1 AND d.kind = \$2`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder).
			WillReturnRows(administrativeOrderRows(docID, now))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnRows(administrativeOrderAcknowledgmentPeopleRows(docID, now))

		doc, err := repo.GetByID(docID)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		assert.Equal(t, "ПР-002", doc.OrderNumber)
		assert.Equal(t, "О внесении изменений", doc.Title)
		assert.NotNil(t, doc.ExecutionDeadline)
		assert.NotNil(t, doc.CancelledAt)
		require.Len(t, doc.AcknowledgmentPeople, 2)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()

		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)WHERE d.id = \$1 AND d.kind = \$2`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder).
			WillReturnError(sql.ErrNoRows)

		doc, err := repo.GetByID(docID)

		require.NoError(t, err)
		assert.Nil(t, doc)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("acknowledgment people error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()

		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)WHERE d.id = \$1 AND d.kind = \$2`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder).
			WillReturnRows(administrativeOrderRows(docID, now))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnError(sql.ErrConnDone)

		doc, err := repo.GetByID(docID)

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to get administrative order acknowledgment people")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdministrativeOrderRepository_Update(t *testing.T) {
	t.Run("success preserves acknowledged people", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()
		deadline := now.AddDate(0, 0, 10)
		cancelledAt := now.AddDate(0, 0, 1)
		req := models.UpdateAdministrativeOrderDocRequest{
			ID:                  docID,
			OrderDate:           now,
			Title:               "Обновленный приказ",
			ExecutionController: "Контрольный отдел",
			ExecutionDeadline:   &deadline,
			IsActive:            false,
			CancelledAt:         &cancelledAt,
			AcknowledgmentFullNames: []string{
				"  Иванов И.И.  ",
				"",
				"Сидоров С.С.",
			},
		}

		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnRows(administrativeOrderAcknowledgmentPeopleRows(docID, now))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.OrderDate, req.Title, req.ID, models.DocumentKindAdministrativeOrder).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE administrative_order_details SET`).
			WithArgs(
				req.OrderDate,
				req.Title,
				req.ExecutionController,
				req.ExecutionDeadline,
				req.IsActive,
				req.CancelledAt,
				req.ID,
			).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM administrative_order_acknowledgment_people WHERE document_id = \$1`).
			WithArgs(docID).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec(`INSERT INTO administrative_order_acknowledgment_people`).
			WithArgs(docID, "Иванов И.И.", sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO administrative_order_acknowledgment_people`).
			WithArgs(docID, "Сидоров С.С.", nil, nil, 2).
			WillReturnResult(sqlmock.NewResult(2, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)WHERE d.id = \$1 AND d.kind = \$2`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder).
			WillReturnRows(administrativeOrderRows(docID, now))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnRows(administrativeOrderAcknowledgmentPeopleRows(docID, now))

		doc, err := repo.Update(req)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rejects deleting acknowledged person", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()
		req := models.UpdateAdministrativeOrderDocRequest{
			ID:                      docID,
			OrderDate:               now,
			Title:                   "Обновленный приказ",
			ExecutionController:     "Контрольный отдел",
			IsActive:                true,
			AcknowledgmentFullNames: []string{"Петров П.П."},
		}

		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnRows(administrativeOrderAcknowledgmentPeopleRows(docID, now))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.OrderDate, req.Title, req.ID, models.DocumentKindAdministrativeOrder).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE administrative_order_details SET`).
			WithArgs(
				req.OrderDate,
				req.Title,
				req.ExecutionController,
				req.ExecutionDeadline,
				req.IsActive,
				req.CancelledAt,
				req.ID,
			).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectRollback()

		doc, err := repo.Update(req)

		require.Error(t, err)
		assert.Nil(t, doc)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "VALIDATION_ERROR", appErr.Kind)
		assert.Contains(t, appErr.Message, "нельзя удалить ФИО уже ознакомленного")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdministrativeOrderRepository_UpdateErrors(t *testing.T) {
	t.Run("existing people load error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnError(sql.ErrConnDone)

		doc, err := repo.Update(models.UpdateAdministrativeOrderDocRequest{ID: docID})

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to get administrative order acknowledgment people")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin transaction error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
				"acknowledged_by_name", "position", "created_at",
			}))
		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		doc, err := repo.Update(models.UpdateAdministrativeOrderDocRequest{ID: docID})

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to begin transaction")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("root update error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		req := models.UpdateAdministrativeOrderDocRequest{
			ID:        uuid.New(),
			OrderDate: time.Now(),
			Title:     "Приказ",
			IsActive:  true,
		}

		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(req.ID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
				"acknowledged_by_name", "position", "created_at",
			}))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.OrderDate, req.Title, req.ID, models.DocumentKindAdministrativeOrder).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Update(req)

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to update administrative order root")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("details update error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		req := models.UpdateAdministrativeOrderDocRequest{
			ID:                  uuid.New(),
			OrderDate:           time.Now(),
			Title:               "Приказ",
			ExecutionController: "Контроль",
			IsActive:            true,
		}

		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(req.ID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "full_name", "acknowledged_at", "acknowledged_by",
				"acknowledged_by_name", "position", "created_at",
			}))
		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.OrderDate, req.Title, req.ID, models.DocumentKindAdministrativeOrder).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE administrative_order_details SET`).
			WithArgs(
				req.OrderDate,
				req.Title,
				req.ExecutionController,
				req.ExecutionDeadline,
				req.IsActive,
				req.CancelledAt,
				req.ID,
			).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Update(req)

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to update administrative order details")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAdministrativeOrderRepository_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()
		deadline := now.AddDate(0, 0, 14)
		req := models.CreateAdministrativeOrderDocRequest{
			NomenclatureID:          uuid.New(),
			IdempotencyKey:          uuid.New(),
			CreatedBy:               uuid.New(),
			OrderNumber:             "ПР-003",
			OrderDate:               now,
			Title:                   "О назначении ответственных",
			ExecutionController:     "Контрольный отдел",
			ExecutionDeadline:       &deadline,
			IsActive:                true,
			AcknowledgmentFullNames: []string{"Иванов И.И.", "  ", "Петров П.П."},
		}

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(req.CreatedBy, models.DocumentKindAdministrativeOrder, req.IdempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(req.NomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
				AddRow("04-01", "/", "manual_only", 1, string(models.DocumentKindAdministrativeOrder)))
		mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
			models.DocumentKindAdministrativeOrder,
			req.NomenclatureID,
			req.IdempotencyKey,
			req.OrderNumber,
			req.OrderDate,
			models.DocumentTypeAdministrativeOrder,
			req.Title,
			req.CreatedBy,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
		mock.ExpectExec(`INSERT INTO administrative_order_details`).WithArgs(
			docID,
			req.OrderNumber,
			req.OrderDate,
			req.Title,
			req.ExecutionController,
			req.ExecutionDeadline,
			req.IsActive,
			req.CancelledAt,
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`DELETE FROM administrative_order_acknowledgment_people WHERE document_id = \$1`).
			WithArgs(docID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(`INSERT INTO administrative_order_acknowledgment_people`).
			WithArgs(docID, "Иванов И.И.", nil, nil, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`INSERT INTO administrative_order_acknowledgment_people`).
			WithArgs(docID, "Петров П.П.", nil, nil, 2).
			WillReturnResult(sqlmock.NewResult(2, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`SELECT(.*)ord\.order_number(.*)WHERE d.id = \$1 AND d.kind = \$2`).
			WithArgs(docID, models.DocumentKindAdministrativeOrder).
			WillReturnRows(administrativeOrderRows(docID, now))
		mock.ExpectQuery(`SELECT(.*)FROM administrative_order_acknowledgment_people p(.*)WHERE p.document_id = \$1`).
			WithArgs(docID).
			WillReturnRows(administrativeOrderAcknowledgmentPeopleRows(docID, now))

		doc, err := repo.Create(req)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("details insert error rolls back", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()
		req := models.CreateAdministrativeOrderDocRequest{
			NomenclatureID: uuid.New(),
			IdempotencyKey: uuid.New(),
			CreatedBy:      uuid.New(),
			OrderNumber:    "ПР-004",
			OrderDate:      now,
			Title:          "О назначении ответственных",
			IsActive:       true,
		}

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
			WithArgs(req.CreatedBy, models.DocumentKindAdministrativeOrder, req.IdempotencyKey).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
			WithArgs(req.NomenclatureID).
			WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
				AddRow("04-01", "/", "manual_only", 1, string(models.DocumentKindAdministrativeOrder)))
		mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
			models.DocumentKindAdministrativeOrder,
			req.NomenclatureID,
			req.IdempotencyKey,
			req.OrderNumber,
			req.OrderDate,
			models.DocumentTypeAdministrativeOrder,
			req.Title,
			req.CreatedBy,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
		mock.ExpectExec(`INSERT INTO administrative_order_details`).WithArgs(
			docID,
			req.OrderNumber,
			req.OrderDate,
			req.Title,
			req.ExecutionController,
			req.ExecutionDeadline,
			req.IsActive,
			req.CancelledAt,
		).WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Create(req)

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to create administrative order details")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("missing idempotency key", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
		mock.ExpectBegin()
		mock.ExpectRollback()

		doc, err := repo.Create(models.CreateAdministrativeOrderDocRequest{
			NomenclatureID: uuid.New(),
			CreatedBy:      uuid.New(),
		})

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "отсутствует ключ идемпотентности")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("root insert errors", func(t *testing.T) {
		tests := []struct {
			name       string
			insertErr  error
			wantErr    string
			wantAppErr bool
		}{
			{
				name:       "registration number conflict",
				insertErr:  &pq.Error{Code: "23505", Constraint: "idx_documents_kind_registration_number_year"},
				wantErr:    "документ с таким регистрационным номером уже существует",
				wantAppErr: true,
			},
			{
				name:      "generic root insert error",
				insertErr: sql.ErrConnDone,
				wantErr:   "failed to create administrative order root",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				db, mock, err := sqlmock.New()
				require.NoError(t, err)
				defer db.Close()

				repo := NewAdministrativeOrderRepository(&database.DB{DB: db})
				req := models.CreateAdministrativeOrderDocRequest{
					NomenclatureID: uuid.New(),
					IdempotencyKey: uuid.New(),
					CreatedBy:      uuid.New(),
					OrderNumber:    "ПР-005",
					OrderDate:      time.Now(),
					Title:          "О назначении ответственных",
				}

				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
					WithArgs(req.CreatedBy, models.DocumentKindAdministrativeOrder, req.IdempotencyKey).
					WillReturnError(sql.ErrNoRows)
				mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
					WithArgs(req.NomenclatureID).
					WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
						AddRow("04-01", "/", "manual_only", 1, string(models.DocumentKindAdministrativeOrder)))
				mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
					models.DocumentKindAdministrativeOrder,
					req.NomenclatureID,
					req.IdempotencyKey,
					req.OrderNumber,
					req.OrderDate,
					models.DocumentTypeAdministrativeOrder,
					req.Title,
					req.CreatedBy,
				).WillReturnError(tt.insertErr)
				mock.ExpectRollback()

				doc, err := repo.Create(req)

				require.Error(t, err)
				assert.Nil(t, doc)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantAppErr {
					appErr, ok := models.AsAppError(err)
					require.True(t, ok)
					assert.Equal(t, "CONFLICT", appErr.Kind)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	})
}
