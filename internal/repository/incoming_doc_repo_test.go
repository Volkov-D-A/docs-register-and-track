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
			"incoming_number", "incoming_date",
			"document_type", "document_type_name",
			"content", "pages_count",
			"sender_signatory",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01 — Дело 1",
			"ВХ-123", now,
			models.DocumentTypeLetter, models.DocumentTypeLetter,
			"Содержание документа", 5,
			"Иванов И.И.",
			uuid.New(), "Создатель",
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)
		mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).WithArgs(docID).WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
		}).AddRow(uuid.New(), docID, "ИСХ-123", now, uuid.New(), "Орг 1", 1))
		mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).WithArgs(docID).WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "resolution", "resolution_author", "resolution_executors", "position",
		}).AddRow(uuid.New(), docID, "В работу", "Директор", "Петров П.П.", 1))

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
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        "Текст",
		Correspondents: []models.DocumentCorrespondentRegistration{{
			RegistrationNumber: "ИСХ-001",
			RegistrationDate:   now,
			CorrespondentOrgID: uuid.New(),
			Position:           1,
		}},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindIncomingLetter, req.NomenclatureID, req.IncomingNumber, req.IncomingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
	mock.ExpectExec(`INSERT INTO incoming_document_details`).WithArgs(
		docID, req.IncomingNumber, req.IncomingDate,
		req.SenderSignatory,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM document_correspondent_registrations`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO document_correspondent_registrations`).WithArgs(
		docID, req.Correspondents[0].RegistrationNumber, req.Correspondents[0].RegistrationDate, req.Correspondents[0].CorrespondentOrgID, req.Correspondents[0].Position,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM document_resolutions`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// После Create идет вызов GetByID
	expectedQuery := incomingDocSelectBase + " WHERE d.id = $1"
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"incoming_number", "incoming_date",
		"document_type", "document_type_name",
		"content", "pages_count",
		"sender_signatory",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ВХ-001", now,
		models.DocumentTypeLetter, models.DocumentTypeLetter, "Текст", 0, "",
		uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).WithArgs(docID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
	}).AddRow(uuid.New(), docID, "ИСХ-001", now, req.Correspondents[0].CorrespondentOrgID, "Орг 1", 1))
	mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).WithArgs(docID).WillReturnError(sql.ErrNoRows)

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
		docID := uuid.New()
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
			"incoming_number", "incoming_date",
			"document_type", "document_type_name",
			"content", "pages_count",
			"sender_signatory",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01", "ВХ-001", now,
			models.DocumentTypeLetter, models.DocumentTypeLetter, "Текст", 0, "",
			uuid.New(), "", now, now,
		)
		mock.ExpectQuery(regexp.QuoteMeta(incomingDocSelectBase)).WillReturnRows(rows)
		mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).WithArgs(docID).WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
		}).AddRow(uuid.New(), docID, "ИСХ-001", now, uuid.New(), "Орг 1", 1))
		mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).WithArgs(docID).WillReturnError(sql.ErrNoRows)

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
		ID:             docID,
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        "Обновленное содержание",
		Correspondents: []models.DocumentCorrespondentRegistration{{
			RegistrationNumber: "ИСХ-001",
			RegistrationDate:   now,
			CorrespondentOrgID: uuid.New(),
			Position:           1,
		}},
	}

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE documents SET`).WithArgs(
		req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindIncomingLetter,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE incoming_document_details SET`).WithArgs(
		req.SenderSignatory,
		req.ID,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM document_correspondent_registrations`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO document_correspondent_registrations`).WithArgs(
		docID, req.Correspondents[0].RegistrationNumber, req.Correspondents[0].RegistrationDate, req.Correspondents[0].CorrespondentOrgID, req.Correspondents[0].Position,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM document_resolutions`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	// После Update идет вызов GetByID
	expectedQuery := incomingDocSelectBase + " WHERE d.id = $1"
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"incoming_number", "incoming_date",
		"document_type", "document_type_name",
		"content", "pages_count",
		"sender_signatory",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ВХ-001", now,
		models.DocumentTypeLetter, models.DocumentTypeLetter, "Обновленное содержание", 0, "",
		uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)
	mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).WithArgs(docID).WillReturnRows(sqlmock.NewRows([]string{
		"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
	}).AddRow(uuid.New(), docID, "ИСХ-001", now, req.Correspondents[0].CorrespondentOrgID, "Орг 1", 1))
	mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).WithArgs(docID).WillReturnError(sql.ErrNoRows)

	doc, err := repo.Update(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "Обновленное содержание", doc.Content)
	require.NoError(t, mock.ExpectationsWereMet())
}
