package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncomingDocumentRepository_GetByID(t *testing.T) {
	// Получение входящего документа по его ID с подгрузкой всех связей
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	expectedQuery := incomingDocSelectBase + " WHERE d.id = $1"

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"incoming_number", "incoming_date", "outgoing_number_sender", "outgoing_date_sender",
			"intermediate_number", "intermediate_date",
			"document_type_id", "document_type_name",
			"content", "pages_count",
			"sender_org_id", "sender_org_name", "sender_signatory",
			"resolution", "resolution_author", "resolution_executors",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01 — Дело 1",
			"ВХ-123", now, "ИСХ-123", now,
			"", nil,
			uuid.New(), "Тип 1",
			"Содержание документа", 5,
			uuid.New(), "Орг 1", "Иванов И.И.",
			"В работу", "Директор", "Петров П.П.",
			uuid.New(), "Создатель",
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

		doc, err := repo.GetByID(docID)
		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		assert.Equal(t, "ВХ-123", doc.IncomingNumber)
		assert.Equal(t, "Содержание документа", doc.Content)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnError(sql.ErrNoRows)

		doc, err := repo.GetByID(docID)
		require.NoError(t, err)
		require.Nil(t, doc)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIncomingDocumentRepository_GetCount(t *testing.T) {
	// Подсчет общего количества зарегистрированных входящих документов
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents WHERE kind = \$1`).WithArgs(models.DocumentKindIncomingLetter).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.GetCount()
	require.NoError(t, err)
	assert.Equal(t, 42, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncomingDocumentRepository_Create(t *testing.T) {
	// Регистрация нового входящего документа в БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	req := models.CreateIncomingDocRequest{
		NomenclatureID: uuid.New(),
		IncomingNumber: "ВХ-001",
		IncomingDate:   now,
		DocumentTypeID: uuid.New(),
		Content:        "Текст",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindIncomingLetter, req.NomenclatureID, req.IncomingNumber, req.IncomingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
	mock.ExpectExec(`INSERT INTO incoming_document_details`).WithArgs(
		docID, req.IncomingNumber, req.IncomingDate,
		req.OutgoingNumberSender, req.OutgoingDateSender,
		req.IntermediateNumber, req.IntermediateDate,
		req.SenderOrgID, req.SenderSignatory,
		req.Resolution, req.ResolutionAuthor, req.ResolutionExecutors,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// После Create идет вызов GetByID
	expectedQuery := incomingDocSelectBase + " WHERE d.id = $1"
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"incoming_number", "incoming_date", "outgoing_number_sender", "outgoing_date_sender",
		"intermediate_number", "intermediate_date",
		"document_type_id", "document_type_name",
		"content", "pages_count",
		"sender_org_id", "sender_org_name", "sender_signatory",
		"resolution", "resolution_author", "resolution_executors",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ВХ-001", now, "", now, "", now,
		uuid.New(), "Тип", "Текст", 0, uuid.New(), "", "",
		"", nil, nil, uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

	doc, err := repo.Create(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "ВХ-001", doc.IncomingNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncomingDocumentRepository_GetList(t *testing.T) {
	// Получение списка входящих документов с фильтрацией (по номенклатуре) и пагинацией
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	now := time.Now()

	t.Run("success with filters", func(t *testing.T) {
		filter := models.DocumentFilter{
			NomenclatureID: uuid.New().String(),
			Page:           1,
			PageSize:       10,
		}

		// Count query
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN incoming_document_details inc ON inc.document_id = d.id(.*)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Data query
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"incoming_number", "incoming_date", "outgoing_number_sender", "outgoing_date_sender",
			"intermediate_number", "intermediate_date",
			"document_type_id", "document_type_name",
			"content", "pages_count",
			"sender_org_id", "sender_org_name", "sender_signatory",
			"resolution", "resolution_author", "resolution_executors",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), "01-01", "ВХ-001", now, "", now, "", now,
			uuid.New(), "Тип", "Текст", 0, uuid.New(), "", "",
			"", nil, nil, uuid.New(), "", now, now,
		)
		mock.ExpectQuery(regexp.QuoteMeta(incomingDocSelectBase)).WillReturnRows(rows)

		res, err := repo.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, res.TotalCount)
		assert.Len(t, res.Items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count database error", func(t *testing.T) {
		filter := models.DocumentFilter{Search: "test"}
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN incoming_document_details inc ON inc.document_id = d.id(.*)`).WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count documents")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("data database error", func(t *testing.T) {
		filter := models.DocumentFilter{}
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN incoming_document_details inc ON inc.document_id = d.id(.*)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(incomingDocSelectBase)).WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get incoming documents")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIncomingDocumentRepository_Update(t *testing.T) {
	// Редактирование карточки входящего документа
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	req := models.UpdateIncomingDocRequest{
		ID:                   docID,
		OutgoingNumberSender: "ИСХ-001",
		Content:              "Обновленное содержание",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE documents SET`).WithArgs(
		req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindIncomingLetter,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE incoming_document_details SET`).WithArgs(
		req.OutgoingNumberSender, req.OutgoingDateSender,
		req.IntermediateNumber, req.IntermediateDate,
		req.SenderOrgID, req.SenderSignatory,
		req.Resolution, req.ResolutionAuthor, req.ResolutionExecutors,
		req.ID,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// После Update идет вызов GetByID
	expectedQuery := incomingDocSelectBase + " WHERE d.id = $1"
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"incoming_number", "incoming_date", "outgoing_number_sender", "outgoing_date_sender",
		"intermediate_number", "intermediate_date",
		"document_type_id", "document_type_name",
		"content", "pages_count",
		"sender_org_id", "sender_org_name", "sender_signatory",
		"resolution", "resolution_author", "resolution_executors",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ВХ-001", now, "ИСХ-001", now, "", now,
		uuid.New(), "Тип", "Обновленное содержание", 0, uuid.New(), "", "",
		"", nil, nil, uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

	doc, err := repo.Update(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "Обновленное содержание", doc.Content)
	require.NoError(t, mock.ExpectationsWereMet())
}
