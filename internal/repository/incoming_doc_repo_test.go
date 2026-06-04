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
	"github.com/lib/pq"
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

func TestIncomingDocumentRepository_LoadResolutions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	firstID := uuid.New()
	secondID := uuid.New()

	mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors, position\s+FROM document_resolutions\s+WHERE document_id = \$1\s+ORDER BY position, created_at, id`).
		WithArgs(docID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "resolution", "resolution_author", "resolution_executors", "position",
		}).
			AddRow(firstID, docID, "Первая", "Автор 1", "Исполнитель 1", 1).
			AddRow(secondID, docID, "Вторая", "Автор 2", "Исполнитель 2", 2))

	resolutions, err := repo.loadResolutions(docID)
	require.NoError(t, err)
	require.Len(t, resolutions, 2)
	assert.Equal(t, firstID, resolutions[0].ID)
	require.NotNil(t, resolutions[0].Resolution)
	assert.Equal(t, "Первая", *resolutions[0].Resolution)
	assert.Equal(t, secondID, resolutions[1].ID)
	require.NotNil(t, resolutions[1].ResolutionExecutors)
	assert.Equal(t, "Исполнитель 2", *resolutions[1].ResolutionExecutors)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplaceResolution(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		docID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM document_resolutions`).
			WithArgs(docID).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		tx, err := db.Begin()
		require.NoError(t, err)
		err = replaceResolution(tx, docID, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clear document resolutions")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("insert error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		docID := uuid.New()
		resolution := "Рассмотреть"
		mock.ExpectBegin()
		mock.ExpectExec(`DELETE FROM document_resolutions`).
			WithArgs(docID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO document_resolutions`).
			WithArgs(docID, &resolution, nil, nil, 1).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		tx, err := db.Begin()
		require.NoError(t, err)
		err = replaceResolution(tx, docID, &resolution, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save document resolution")
		require.NoError(t, tx.Rollback())
		require.NoError(t, mock.ExpectationsWereMet())
	})
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
		IdempotencyKey: uuid.New(),
		CreatedBy:      uuid.New(),
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
	mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
		WithArgs(req.CreatedBy, models.DocumentKindIncomingLetter, req.IdempotencyKey).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(req.NomenclatureID).
		WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
			AddRow("01-01", "/", "manual_only", 1, string(models.DocumentKindIncomingLetter)))
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindIncomingLetter, req.NomenclatureID, req.IdempotencyKey, req.IncomingNumber, req.IncomingDate, req.DocumentTypeID, req.Content, req.PagesCount, req.CreatedBy,
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

func TestIncomingDocumentRepository_CreateValidationErrors(t *testing.T) {
	t.Run("invalid document type", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewIncomingDocumentRepository(&database.DB{DB: db})

		doc, err := repo.Create(models.CreateIncomingDocRequest{DocumentTypeID: "unknown"})
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "неверный тип документа")
	})

	t.Run("missing idempotency key", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewIncomingDocumentRepository(&database.DB{DB: db})
		mock.ExpectBegin()
		mock.ExpectRollback()

		doc, err := repo.Create(models.CreateIncomingDocRequest{
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

func TestIncomingDocumentRepository_CreateRootInsertErrors(t *testing.T) {
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

			repo := NewIncomingDocumentRepository(&database.DB{DB: db})
			req := models.CreateIncomingDocRequest{
				NomenclatureID: uuid.New(),
				IdempotencyKey: uuid.New(),
				CreatedBy:      uuid.New(),
				IncomingNumber: "ВХ-002",
				IncomingDate:   time.Now(),
				DocumentTypeID: models.DocumentTypeLetter,
				Content:        "Текст",
			}

			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
				WithArgs(req.CreatedBy, models.DocumentKindIncomingLetter, req.IdempotencyKey).
				WillReturnError(sql.ErrNoRows)
			mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
				WithArgs(req.NomenclatureID).
				WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
					AddRow("01-01", "/", "manual_only", 1, string(models.DocumentKindIncomingLetter)))
			mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
				models.DocumentKindIncomingLetter,
				req.NomenclatureID,
				req.IdempotencyKey,
				req.IncomingNumber,
				req.IncomingDate,
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

func TestIncomingDocumentRepository_CreateDetailsInsertError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	req := models.CreateIncomingDocRequest{
		NomenclatureID: uuid.New(),
		IdempotencyKey: uuid.New(),
		CreatedBy:      uuid.New(),
		IncomingNumber: "ВХ-003",
		IncomingDate:   time.Now(),
		DocumentTypeID: models.DocumentTypeLetter,
		Content:        "Текст",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
		WithArgs(req.CreatedBy, models.DocumentKindIncomingLetter, req.IdempotencyKey).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(req.NomenclatureID).
		WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
			AddRow("01-01", "/", "manual_only", 1, string(models.DocumentKindIncomingLetter)))
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindIncomingLetter,
		req.NomenclatureID,
		req.IdempotencyKey,
		req.IncomingNumber,
		req.IncomingDate,
		req.DocumentTypeID,
		req.Content,
		req.PagesCount,
		req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
	mock.ExpectExec(`INSERT INTO incoming_document_details`).WithArgs(
		docID,
		req.IncomingNumber,
		req.IncomingDate,
		req.SenderSignatory,
	).WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	doc, err := repo.Create(req)

	require.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "failed to create incoming document details")
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
		docID2 := uuid.New()
		filter := models.DocumentFilter{
			NomenclatureID: uuid.New().String(),
			Page:           1,
			PageSize:       10,
		}

		// Count query
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN incoming_document_details inc ON inc.document_id = d.id(.*)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

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
		).AddRow(
			docID2, uuid.New(), "01-02", "ВХ-002", now,
			models.DocumentTypeLetter, models.DocumentTypeLetter, "Текст 2", 1, "",
			uuid.New(), "", now, now,
		)
		mock.ExpectQuery(regexp.QuoteMeta(incomingDocSelectBase)).WillReturnRows(rows)
		mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
		}).AddRow(uuid.New(), docID, "ИСХ-001", now, uuid.New(), "Орг 1", 1))
		mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{
			"id", "document_id", "resolution", "resolution_author", "resolution_executors", "position",
		}))

		res, err := repo.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 2, res.TotalCount)
		assert.Len(t, res.Items, 2)
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

	t.Run("success with broad filters and pagination limits", func(t *testing.T) {
		filter := models.DocumentFilter{
			AccessibleByUserID:     uuid.New().String(),
			AllowedNomenclatureIDs: []string{uuid.New().String()},
			NomenclatureIDs:        []string{uuid.New().String(), uuid.New().String()},
			DocumentTypeID:         string(models.DocumentTypeContract),
			OrgID:                  uuid.New().String(),
			DateFrom:               "2026-01-01",
			DateTo:                 "2026-12-31",
			Search:                 "важно",
			IncomingNumber:         "ВХ",
			OutgoingNumber:         "ИСХ",
			SenderName:             "Ромашка",
			OutgoingDateFrom:       "2025-01-01",
			OutgoingDateTo:         "2025-12-31",
			NoResolution:           true,
			Page:                   -1,
			PageSize:               500,
		}

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN incoming_document_details inc ON inc.document_id = d.id(.*)`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(regexp.QuoteMeta(incomingDocSelectBase)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nomenclature_id", "nomenclature_name",
				"incoming_number", "incoming_date",
				"document_type", "document_type_name",
				"content", "pages_count",
				"sender_signatory",
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

	t.Run("success with single nomenclature and resolution filter", func(t *testing.T) {
		filter := models.DocumentFilter{
			NomenclatureID: uuid.New().String(),
			Resolution:     "исполнить",
		}

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents d JOIN incoming_document_details inc ON inc.document_id = d.id(.*)`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(regexp.QuoteMeta(incomingDocSelectBase)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "nomenclature_id", "nomenclature_name",
				"incoming_number", "incoming_date",
				"document_type", "document_type_name",
				"content", "pages_count",
				"sender_signatory",
				"created_by", "created_by_name",
				"created_at", "updated_at",
			}))

		res, err := repo.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Empty(t, res.Items)
		assert.Equal(t, 20, res.PageSize)
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

func TestIncomingDocumentRepository_UpdateErrors(t *testing.T) {
	t.Run("invalid document type", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewIncomingDocumentRepository(&database.DB{DB: db})

		doc, err := repo.Update(models.UpdateIncomingDocRequest{DocumentTypeID: "unknown"})
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "неверный тип документа")
	})

	t.Run("begin transaction error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewIncomingDocumentRepository(&database.DB{DB: db})
		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		doc, err := repo.Update(models.UpdateIncomingDocRequest{DocumentTypeID: models.DocumentTypeLetter})
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to begin transaction")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("root update error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewIncomingDocumentRepository(&database.DB{DB: db})
		req := models.UpdateIncomingDocRequest{
			ID:             uuid.New(),
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        "Текст",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindIncomingLetter).
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

		repo := NewIncomingDocumentRepository(&database.DB{DB: db})
		req := models.UpdateIncomingDocRequest{
			ID:             uuid.New(),
			DocumentTypeID: models.DocumentTypeLetter,
			Content:        "Текст",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).
			WithArgs(req.DocumentTypeID, req.Content, req.PagesCount, req.ID, models.DocumentKindIncomingLetter).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE incoming_document_details SET`).
			WithArgs(req.SenderSignatory, req.ID).
			WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Update(req)
		require.Error(t, err)
		assert.Nil(t, doc)
		assert.Contains(t, err.Error(), "failed to update incoming document details")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
