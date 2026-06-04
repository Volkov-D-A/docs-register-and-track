package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
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
			"document_type", "document_type_name",
			"content", "pages_count",
			"sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01 — Дело 1",
			"ИСХ-123", now,
			models.DocumentTypeLetter, models.DocumentTypeLetter,
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
		IdempotencyKey: uuid.New(),
		CreatedBy:      uuid.New(),
		OutgoingNumber: "ИСХ-001",
		OutgoingDate:   now,
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        "Текст",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
		WithArgs(req.CreatedBy, models.DocumentKindOutgoingLetter, req.IdempotencyKey).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(req.NomenclatureID).
		WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
			AddRow("01-01", "/", "manual_only", 1, string(models.DocumentKindOutgoingLetter)))
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindOutgoingLetter, req.NomenclatureID, req.IdempotencyKey, req.OutgoingNumber, req.OutgoingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
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
		"document_type", "document_type_name",
		"content", "pages_count",
		"sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ИСХ-001", now,
		models.DocumentTypeLetter, models.DocumentTypeLetter, "Текст", 0, "", "",
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

func TestOutgoingDocumentRepository_CreateValidationErrors(t *testing.T) {
	t.Run("invalid document type", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewOutgoingDocumentRepository(&database.DB{DB: db})

		doc, err := repo.Create(models.CreateOutgoingDocRequest{DocumentTypeID: "unknown"})
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "неверный тип документа")
	})

	t.Run("missing idempotency key", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
		mock.ExpectBegin()
		mock.ExpectRollback()

		doc, err := repo.Create(models.CreateOutgoingDocRequest{
			NomenclatureID: uuid.New(),
			CreatedBy:      uuid.New(),
			DocumentTypeID: models.DocumentTypeLetter,
		})

		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "отсутствует ключ идемпотентности")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestOutgoingDocumentRepository_CreateRootInsertErrors(t *testing.T) {
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
			wantErr:   "failed to create document root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
			req := models.CreateOutgoingDocRequest{
				NomenclatureID: uuid.New(),
				IdempotencyKey: uuid.New(),
				CreatedBy:      uuid.New(),
				OutgoingNumber: "ИСХ-002",
				OutgoingDate:   time.Now(),
				DocumentTypeID: models.DocumentTypeLetter,
				Content:        "Текст",
				RecipientOrgID: uuid.New(),
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
				WithArgs(req.CreatedBy, models.DocumentKindOutgoingLetter, req.IdempotencyKey).
				WillReturnError(sql.ErrNoRows)
			mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
				WithArgs(req.NomenclatureID).
				WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
					AddRow("02-01", "/", "manual_only", 1, string(models.DocumentKindOutgoingLetter)))
			mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
				models.DocumentKindOutgoingLetter,
				req.NomenclatureID,
				req.IdempotencyKey,
				req.OutgoingNumber,
				req.OutgoingDate,
				req.DocumentTypeID,
				req.Content,
				req.PagesCount,
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
}

func TestOutgoingDocumentRepository_CreateDetailsInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	req := models.CreateOutgoingDocRequest{
		NomenclatureID: uuid.New(),
		IdempotencyKey: uuid.New(),
		CreatedBy:      uuid.New(),
		OutgoingNumber: "ИСХ-003",
		OutgoingDate:   time.Now(),
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        "Текст",
		RecipientOrgID: uuid.New(),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
		WithArgs(req.CreatedBy, models.DocumentKindOutgoingLetter, req.IdempotencyKey).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(req.NomenclatureID).
		WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
			AddRow("02-01", "/", "manual_only", 1, string(models.DocumentKindOutgoingLetter)))
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindOutgoingLetter,
		req.NomenclatureID,
		req.IdempotencyKey,
		req.OutgoingNumber,
		req.OutgoingDate,
		req.DocumentTypeID,
		req.Content,
		req.PagesCount,
		req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
	mock.ExpectExec(`INSERT INTO outgoing_document_details`).WithArgs(
		docID,
		req.OutgoingNumber,
		req.OutgoingDate,
		req.SenderSignatory,
		req.SenderExecutor,
		req.RecipientOrgID,
		req.Addressee,
	).WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	doc, err := repo.Create(req)

	require.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "failed to create outgoing document details")
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
			"document_type", "document_type_name",
			"content", "pages_count",
			"sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			uuid.New(), uuid.New(), "01-01", "ИСХ-001", now,
			models.DocumentTypeLetter, models.DocumentTypeLetter, "Текст", 0, "", "",
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

	t.Run("success with broad filters and pagination limits", func(t *testing.T) {
		filter := models.OutgoingDocumentFilter{
			AccessibleByUserID:     uuid.New().String(),
			AllowedNomenclatureIDs: []string{uuid.New().String()},
			NomenclatureIDs:        []string{uuid.New().String(), uuid.New().String()},
			DocumentTypeID:         string(models.DocumentTypeContract),
			OrgID:                  uuid.New().String(),
			DateFrom:               "2026-01-01",
			DateTo:                 "2026-12-31",
			Search:                 "важно",
			OutgoingNumber:         "ИСХ",
			RecipientName:          "Ромашка",
			Page:                   -1,
			PageSize:               500,
		}

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT(.*)FROM documents d(.*)JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nomenclature_id", "nomenclature_name",
				"outgoing_number", "outgoing_date",
				"document_type", "document_type_name",
				"content", "pages_count",
				"sender_signatory", "sender_executor",
				"recipient_org_id", "recipient_org_name", "addressee",
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

	t.Run("success with default pagination", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`SELECT(.*)FROM documents d(.*)JOIN outgoing_document_details out ON out.document_id = d.id(.*)`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nomenclature_id", "nomenclature_name",
				"outgoing_number", "outgoing_date",
				"document_type", "document_type_name",
				"content", "pages_count",
				"sender_signatory", "sender_executor",
				"recipient_org_id", "recipient_org_name", "addressee",
				"created_by", "created_by_name",
				"created_at", "updated_at",
			}))

		res, err := repo.GetList(models.OutgoingDocumentFilter{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 20, res.PageSize)
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
		ID:             docID,
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        "Обновленный текст",
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
		"document_type", "document_type_name",
		"content", "pages_count",
		"sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ИСХ-001", now,
		models.DocumentTypeLetter, models.DocumentTypeLetter, "Обновленный текст", 0, "", "",
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

func TestOutgoingDocumentRepository_UpdateErrors(t *testing.T) {
	t.Run("invalid document type", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewOutgoingDocumentRepository(&database.DB{DB: db})

		doc, err := repo.Update(models.UpdateOutgoingDocRequest{DocumentTypeID: "unknown"})
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "неверный тип документа")
	})

	t.Run("begin transaction error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		doc, err := repo.Update(models.UpdateOutgoingDocRequest{DocumentTypeID: models.DocumentTypeLetter})
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to begin transaction")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("root update error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
		req := models.UpdateOutgoingDocRequest{
			ID:             uuid.New(),
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        "Текст",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindOutgoingLetter).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Update(req)
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to update document root")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("details update error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewOutgoingDocumentRepository(&database.DB{DB: db})
		req := models.UpdateOutgoingDocRequest{
			ID:             uuid.New(),
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        "Текст",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindOutgoingLetter).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE outgoing_document_details SET`).
			WithArgs(
				req.OutgoingDate,
				req.SenderSignatory,
				req.SenderExecutor,
				req.RecipientOrgID,
				req.Addressee,
				req.ID,
			).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Update(req)
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to update outgoing document details")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
