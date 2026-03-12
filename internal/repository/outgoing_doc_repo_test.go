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

func TestOutgoingDocumentRepository_GetByID(t *testing.T) {
	// Получение исходящего документа по его ID со всеми связанными справочниками
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			d.outgoing_number, d.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name as sender_org_name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name as recipient_org_name, d.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM outgoing_documents d
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations so ON d.sender_org_id = so.id
		JOIN organizations ro ON d.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id
		WHERE d.id = $1`

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"outgoing_number", "outgoing_date",
			"document_type_id", "document_type_name",
			"subject", "pages_count", "content",
			"sender_org_id", "sender_org_name", "sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01 — Дело 1",
			"ИСХ-123", now,
			uuid.New(), "Тип 1",
			"Тестовая тема", 5, "Содержание",
			uuid.New(), "Орг 1", "Иванов И.И.", "Петров П.П.",
			uuid.New(), "Орг 2", "Сидоров С.С.",
			uuid.New(), "Создатель",
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

		doc, err := repo.GetByID(docID)
		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		assert.Equal(t, "ИСХ-123", doc.OutgoingNumber)
		assert.Equal(t, "Тестовая тема", doc.Subject)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnError(sql.ErrNoRows)

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

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM outgoing_documents`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

	count, err := repo.GetCount()
	require.NoError(t, err)
	assert.Equal(t, 15, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutgoingDocumentRepository_Delete(t *testing.T) {
	// Удаление исходящего документа по его ID
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()

	mock.ExpectExec(`DELETE FROM outgoing_documents WHERE id = \$1`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(docID)
	require.NoError(t, err)
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
		Subject:        "Тема",
		Content:        "Текст",
	}

	mock.ExpectQuery(`INSERT INTO outgoing_documents`).WithArgs(
		req.NomenclatureID, req.OutgoingNumber, req.OutgoingDate,
		req.DocumentTypeID, req.Subject, req.PagesCount, req.Content,
		req.SenderOrgID, req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee, req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))

	// После Create идет вызов GetByID
	expectedQuery := `SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			d.outgoing_number, d.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name as sender_org_name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name as recipient_org_name, d.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM outgoing_documents d
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations so ON d.sender_org_id = so.id
		JOIN organizations ro ON d.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id
		WHERE d.id = $1`

	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"outgoing_number", "outgoing_date",
		"document_type_id", "document_type_name",
		"subject", "pages_count", "content",
		"sender_org_id", "sender_org_name", "sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ИСХ-001", now,
		uuid.New(), "Тип", "Тема", 0, "Текст", uuid.New(), "", "", "",
		uuid.New(), "", "", uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

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
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM outgoing_documents`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Data query
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"outgoing_number", "outgoing_date",
			"document_type_id", "document_type_name",
			"subject", "pages_count", "content",
			"sender_org_id", "sender_org_name", "sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), "01-01", "ИСХ-001", now,
			uuid.New(), "Тип", "Тема", 0, "Текст", uuid.New(), "", "", "",
			uuid.New(), "", "", uuid.New(), "", now, now,
		)
		
		expectedSelectBase := `SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			d.outgoing_number, d.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name as sender_org_name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name as recipient_org_name, d.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM outgoing_documents d
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations so ON d.sender_org_id = so.id
		JOIN organizations ro ON d.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id`
		mock.ExpectQuery(regexp.QuoteMeta(expectedSelectBase)).WillReturnRows(rows)

		res, err := repo.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, res.TotalCount)
		assert.Len(t, res.Items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count database error", func(t *testing.T) {
		filter := models.OutgoingDocumentFilter{Search: "test"}
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM outgoing_documents`).WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count documents")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	
	t.Run("data database error", func(t *testing.T) {
		filter := models.OutgoingDocumentFilter{}
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM outgoing_documents`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
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
		Subject: "Обновленная Тема",
	}

	mock.ExpectExec(`UPDATE outgoing_documents SET`).WithArgs(
		req.DocumentTypeID, req.Subject, req.PagesCount, req.Content,
		req.SenderOrgID, req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee, req.OutgoingDate,
		req.ID,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	// После Update идет вызов GetByID
	expectedQuery := `SELECT 
			d.id, d.nomenclature_id, n.index || ' — ' || n.name as nomenclature_name,
			d.outgoing_number, d.outgoing_date,
			d.document_type_id, dt.name as document_type_name,
			d.subject, d.pages_count, d.content,
			d.sender_org_id, so.name as sender_org_name, d.sender_signatory, d.sender_executor,
			d.recipient_org_id, ro.name as recipient_org_name, d.addressee,
			d.created_by, u.full_name as created_by_name,
			d.created_at, d.updated_at
		FROM outgoing_documents d
		JOIN nomenclature n ON d.nomenclature_id = n.id
		JOIN document_types dt ON d.document_type_id = dt.id
		JOIN organizations so ON d.sender_org_id = so.id
		JOIN organizations ro ON d.recipient_org_id = ro.id
		JOIN users u ON d.created_by = u.id
		WHERE d.id = $1`
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"outgoing_number", "outgoing_date",
		"document_type_id", "document_type_name",
		"subject", "pages_count", "content",
		"sender_org_id", "sender_org_name", "sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ИСХ-001", now,
		uuid.New(), "Тип", "Обновленная Тема", 0, "Текст", uuid.New(), "", "", "",
		uuid.New(), "", "", uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

	doc, err := repo.Update(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "Обновленная Тема", doc.Subject)
	require.NoError(t, mock.ExpectationsWereMet())
}
