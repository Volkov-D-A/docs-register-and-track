package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutgoingDocumentRepository_GetByID(t *testing.T) {
	// Получение исходящего документа по его ID со всеми связанными справочниками
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT(.*)FROM documents d(.*)JOIN outgoing_document_details out ON out.document_id = d.id(.*)WHERE d.id = \$1 AND d.kind = \$2`

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"outgoing_number", "outgoing_date",
			"document_type_id", "document_type_name",
			"content", "pages_count",
			"sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01 — Дело 1",
			"ИСХ-123", now,
			uuid.New(), "Тип 1",
			"Содержание", 5,
			"Иванов И.И.", "Петров П.П.",
			uuid.New(), "Орг 2", "Сидоров С.С.",
			uuid.New(), "Создатель",
			now, now,
		)

		mock.ExpectQuery(expectedQuery).WithArgs(docID, models.DocumentKindOutgoingLetter).WillReturnRows(rows)

		doc, err := repo.GetByID(docID)
		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		assert.Equal(t, "ИСХ-123", doc.OutgoingNumber)
		assert.Equal(t, "Содержание", doc.Content)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(expectedQuery).WithArgs(docID, models.DocumentKindOutgoingLetter).WillReturnError(sql.ErrNoRows)

		doc, err := repo.GetByID(docID)
		require.NoError(t, err)
		require.Nil(t, doc)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOutgoingDocumentRepository_GetCount(t *testing.T) {
	// Подсчет общего количества зарегистрированных исходящих документов
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents WHERE kind = \$1`).WithArgs(models.DocumentKindOutgoingLetter).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

	count, err := repo.GetCount()
	require.NoError(t, err)
	assert.Equal(t, 15, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutgoingDocumentRepository_Create(t *testing.T) {
	// Создание новой карточки исходящего документа
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	req := models.CreateOutgoingDocRequest{
		NomenclatureID: uuid.New(),
		OutgoingNumber: "ИСХ-001",
		OutgoingDate:   now,
		DocumentTypeID: uuid.New(),
		Content:        "Текст",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindOutgoingLetter, req.NomenclatureID, req.OutgoingNumber, req.OutgoingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
	mock.ExpectExec(`INSERT INTO outgoing_document_details`).WithArgs(
		docID, req.OutgoingNumber, req.OutgoingDate,
		req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// После Create идет вызов GetByID
	expectedQuery := `SELECT(.*)FROM documents d(.*)JOIN outgoing_document_details out ON out.document_id = d.id(.*)WHERE d.id = \$1 AND d.kind = \$2`

	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"outgoing_number", "outgoing_date",
		"document_type_id", "document_type_name",
		"content", "pages_count",
		"sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ИСХ-001", now,
		uuid.New(), "Тип", "Текст", 0, "", "",
		uuid.New(), "", "", uuid.New(), "", now, now,
	)

	mock.ExpectQuery(expectedQuery).WithArgs(docID, models.DocumentKindOutgoingLetter).WillReturnRows(rows)

	doc, err := repo.Create(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "ИСХ-001", doc.OutgoingNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutgoingDocumentRepository_GetList(t *testing.T) {
	// Получение списка исходящих документов с фильтрацией (по номенклатуре) и пагинацией
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	now := time.Now()

	t.Run("success with filters", func(t *testing.T) {
		filter := models.OutgoingDocumentFilter{
			NomenclatureIDs: []string{uuid.New().String()},
			Page:            1,
			PageSize:        10,
		}

		// Count query
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Data query
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"outgoing_number", "outgoing_date",
			"document_type_id", "document_type_name",
			"content", "pages_count",
			"sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), "01-01", "ИСХ-001", now,
			uuid.New(), "Тип", "Текст", 0, "", "",
			uuid.New(), "", "", uuid.New(), "", now, now,
		)

		expectedSelectBase := `SELECT(.*)FROM documents d(.*)JOIN outgoing_document_details out ON out.document_id = d.id(.*)`
		mock.ExpectQuery(expectedSelectBase).WillReturnRows(rows)

		res, err := repo.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, res.TotalCount)
		assert.Len(t, res.Items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count database error", func(t *testing.T) {
		filter := models.OutgoingDocumentFilter{Search: "test"}
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count documents")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("data database error", func(t *testing.T) {
		filter := models.OutgoingDocumentFilter{}
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get outgoing documents")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOutgoingDocumentRepository_Update(t *testing.T) {
	// Обновление данных существующего исходящего документа
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	req := models.UpdateOutgoingDocRequest{
		ID:      docID,
		Content: "Обновленный текст",
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE documents SET`).WithArgs(
		req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindOutgoingLetter,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE outgoing_document_details SET`).WithArgs(
		req.OutgoingDate, req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee, req.ID,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// После Update идет вызов GetByID
	expectedQuery := `SELECT(.*)FROM documents d(.*)JOIN outgoing_document_details out ON out.document_id = d.id(.*)WHERE d.id = \$1 AND d.kind = \$2`
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"outgoing_number", "outgoing_date",
		"document_type_id", "document_type_name",
		"content", "pages_count",
		"sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ИСХ-001", now,
		uuid.New(), "Тип", "Обновленный текст", 0, "", "",
		uuid.New(), "", "", uuid.New(), "", now, now,
	)

	mock.ExpectQuery(expectedQuery).WithArgs(docID, models.DocumentKindOutgoingLetter).WillReturnRows(rows)

	doc, err := repo.Update(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "Обновленный текст", doc.Content)
	require.NoError(t, mock.ExpectationsWereMet())
}
