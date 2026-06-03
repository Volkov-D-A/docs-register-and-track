package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func citizenAppealRows(docID uuid.UUID, now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"registration_number", "registration_date",
		"appeal_date",
		"document_type", "document_type_name",
		"content", "pages_count",
		"applicant_full_name", "registration_address",
		"appeal_type", "applicant_category",
		"appeal_pages_count", "attachment_pages_count",
		"has_envelope", "received_from_pos",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "03-01 - Обращения",
		"ОГ-123", now,
		now.Add(-24*time.Hour),
		models.DocumentTypeCitizenAppeal, models.DocumentTypeCitizenAppeal,
		"Содержание обращения", 3,
		"Иван Иванов", "ул. Ленина, 1",
		"жалоба", "гражданин",
		2, 1,
		true, false,
		uuid.New(), "Регистратор",
		now, now,
	)
}

func emptyCitizenAppealCorrespondentRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
	})
}

func emptyDocumentResolutionRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "document_id", "resolution", "resolution_author", "resolution_executors", "position",
	})
}

func expectCitizenAppealHydrateEmpty(mock sqlmock.Sqlmock, docID uuid.UUID) {
	mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).
		WithArgs(docID).
		WillReturnRows(emptyCitizenAppealCorrespondentRows())
	mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).
		WithArgs(docID).
		WillReturnRows(emptyDocumentResolutionRows())
}

func TestCitizenAppealRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCitizenAppealRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()
	expectedQuery := citizenAppealSelectBase + " WHERE d.id = $1 AND d.kind = $2"

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
			WithArgs(docID, models.DocumentKindCitizenAppeal).
			WillReturnRows(citizenAppealRows(docID, now))
		mock.ExpectQuery(`SELECT cr\.id, cr\.document_id, cr\.registration_number, cr\.registration_date`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "registration_number", "registration_date", "correspondent_org_id", "name", "position",
			}).AddRow(uuid.New(), docID, "EXT-1", now, uuid.New(), "Администрация", 1))
		resolution := "Подготовить ответ"
		author := "Руководитель"
		executors := "Исполнитель"
		mock.ExpectQuery(`SELECT id, document_id, resolution, resolution_author, resolution_executors`).
			WithArgs(docID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "document_id", "resolution", "resolution_author", "resolution_executors", "position",
			}).AddRow(uuid.New(), docID, resolution, author, executors, 1))

		doc, err := repo.GetByID(docID)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		assert.Equal(t, "ОГ-123", doc.RegistrationNumber)
		assert.Equal(t, "Иван Иванов", doc.ApplicantFullName)
		assert.Len(t, doc.Correspondents, 1)
		assert.Len(t, doc.Resolutions, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
			WithArgs(docID, models.DocumentKindCitizenAppeal).
			WillReturnError(sql.ErrNoRows)

		doc, err := repo.GetByID(docID)

		require.NoError(t, err)
		assert.Nil(t, doc)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCitizenAppealRepository_GetCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCitizenAppealRepository(&database.DB{DB: db})

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM documents WHERE kind = \$1`).
		WithArgs(models.DocumentKindCitizenAppeal).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(17))

	count, err := repo.GetCount()

	require.NoError(t, err)
	assert.Equal(t, 17, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCitizenAppealRepository_GetList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCitizenAppealRepository(&database.DB{DB: db})
	now := time.Now()
	docID := uuid.New()

	t.Run("success with filters", func(t *testing.T) {
		filter := models.DocumentFilter{
			NomenclatureID:     uuid.New().String(),
			RegistrationNumber: "ОГ",
			ApplicantName:      "Иван",
			AppealType:         "жалоба",
			Page:               1,
			PageSize:           10,
		}

		mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM documents d\s+JOIN citizen_appeal_details ca ON ca.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(regexp.QuoteMeta(citizenAppealSelectBase)).
			WillReturnRows(citizenAppealRows(docID, now))
		expectCitizenAppealHydrateEmpty(mock, docID)

		res, err := repo.GetList(filter)

		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 1, res.TotalCount)
		assert.Equal(t, 1, res.Page)
		assert.Equal(t, 10, res.PageSize)
		require.Len(t, res.Items, 1)
		assert.Equal(t, docID, res.Items[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM documents d\s+JOIN citizen_appeal_details ca ON ca.document_id = d.id`).
			WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(models.DocumentFilter{Search: "test"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to count citizen appeals")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("data database error", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\)\s+FROM documents d\s+JOIN citizen_appeal_details ca ON ca.document_id = d.id`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT`).WillReturnError(sql.ErrConnDone)

		res, err := repo.GetList(models.DocumentFilter{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get citizen appeals")
		assert.Nil(t, res)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCitizenAppealRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewCitizenAppealRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()
	resolution := "Подготовить ответ"
	author := "Руководитель"
	executors := "Исполнитель"
	req := models.CreateCitizenAppealDocRequest{
		NomenclatureID:       uuid.New(),
		IdempotencyKey:       uuid.New(),
		CreatedBy:            uuid.New(),
		RegistrationNumber:   "ОГ-001",
		RegistrationDate:     now,
		AppealDate:           now.Add(-24 * time.Hour),
		Content:              "Текст",
		ApplicantFullName:    "Иван Иванов",
		RegistrationAddress:  "ул. Ленина, 1",
		AppealType:           "жалоба",
		ApplicantCategory:    "гражданин",
		AppealPagesCount:     2,
		AttachmentPagesCount: 1,
		HasEnvelope:          true,
		ReceivedFromPOS:      false,
		Correspondents: []models.DocumentCorrespondentRegistration{{
			RegistrationNumber: "EXT-1",
			RegistrationDate:   now.Add(-48 * time.Hour),
			CorrespondentOrgID: uuid.New(),
			Position:           1,
		}},
		Resolutions: []models.DocumentResolution{{
			Resolution:          &resolution,
			ResolutionAuthor:    &author,
			ResolutionExecutors: &executors,
			Position:            1,
		}},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id\s+FROM documents\s+WHERE created_by = \$1 AND kind = \$2 AND idempotency_key = \$3`).
		WithArgs(req.CreatedBy, models.DocumentKindCitizenAppeal, req.IdempotencyKey).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT index, separator, numbering_mode, next_number, kind_code\s+FROM nomenclature\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(req.NomenclatureID).
		WillReturnRows(sqlmock.NewRows([]string{"index", "separator", "numbering_mode", "next_number", "kind_code"}).
			AddRow("03-01", "/", "manual_only", 1, string(models.DocumentKindCitizenAppeal)))
	mock.ExpectQuery(`INSERT INTO documents`).WithArgs(
		models.DocumentKindCitizenAppeal,
		req.NomenclatureID,
		req.IdempotencyKey,
		req.RegistrationNumber,
		req.RegistrationDate,
		models.DocumentTypeCitizenAppeal,
		req.Content,
		3,
		req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))
	mock.ExpectExec(`INSERT INTO citizen_appeal_details`).WithArgs(
		docID,
		req.AppealDate,
		req.ApplicantFullName,
		req.RegistrationAddress,
		req.AppealType,
		req.ApplicantCategory,
		req.AppealPagesCount,
		req.AttachmentPagesCount,
		req.HasEnvelope,
		req.ReceivedFromPOS,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM document_correspondent_registrations`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO document_correspondent_registrations`).WithArgs(
		docID,
		req.Correspondents[0].RegistrationNumber,
		req.Correspondents[0].RegistrationDate,
		req.Correspondents[0].CorrespondentOrgID,
		req.Correspondents[0].Position,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM document_resolutions`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO document_resolutions`).WithArgs(
		docID,
		req.Resolutions[0].Resolution,
		req.Resolutions[0].ResolutionAuthor,
		req.Resolutions[0].ResolutionExecutors,
		req.Resolutions[0].Position,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	expectedQuery := citizenAppealSelectBase + " WHERE d.id = $1 AND d.kind = $2"
	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
		WithArgs(docID, models.DocumentKindCitizenAppeal).
		WillReturnRows(citizenAppealRows(docID, now))
	expectCitizenAppealHydrateEmpty(mock, docID)

	doc, err := repo.Create(req)

	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "ОГ-123", doc.RegistrationNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCitizenAppealRepository_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewCitizenAppealRepository(&database.DB{DB: db})
		docID := uuid.New()
		now := time.Now()
		resolution := "Повторно рассмотреть"
		author := "Руководитель"
		executors := "Исполнитель"
		req := models.UpdateCitizenAppealDocRequest{
			ID:                   docID,
			RegistrationNumber:   "ОГ-002",
			RegistrationDate:     now,
			AppealDate:           now.Add(-24 * time.Hour),
			Content:              "Обновленный текст",
			ApplicantFullName:    "Петр Петров",
			RegistrationAddress:  "ул. Мира, 2",
			AppealType:           "заявление",
			ApplicantCategory:    "пенсионер",
			AppealPagesCount:     0,
			AttachmentPagesCount: 0,
			HasEnvelope:          false,
			ReceivedFromPOS:      true,
			Correspondents: []models.DocumentCorrespondentRegistration{{
				RegistrationNumber: "EXT-2",
				RegistrationDate:   now.Add(-48 * time.Hour),
				CorrespondentOrgID: uuid.New(),
			}},
			Resolutions: []models.DocumentResolution{{
				Resolution:          &resolution,
				ResolutionAuthor:    &author,
				ResolutionExecutors: &executors,
			}},
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).WithArgs(
			req.RegistrationNumber,
			req.RegistrationDate,
			req.Content,
			1,
			req.ID,
			models.DocumentKindCitizenAppeal,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`UPDATE citizen_appeal_details SET`).WithArgs(
			req.AppealDate,
			req.ApplicantFullName,
			req.RegistrationAddress,
			req.AppealType,
			req.ApplicantCategory,
			req.AppealPagesCount,
			req.AttachmentPagesCount,
			req.HasEnvelope,
			req.ReceivedFromPOS,
			req.ID,
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM document_correspondent_registrations`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO document_correspondent_registrations`).WithArgs(
			docID,
			req.Correspondents[0].RegistrationNumber,
			req.Correspondents[0].RegistrationDate,
			req.Correspondents[0].CorrespondentOrgID,
			1,
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`DELETE FROM document_resolutions`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`INSERT INTO document_resolutions`).WithArgs(
			docID,
			req.Resolutions[0].Resolution,
			req.Resolutions[0].ResolutionAuthor,
			req.Resolutions[0].ResolutionExecutors,
			1,
		).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		expectedQuery := citizenAppealSelectBase + " WHERE d.id = $1 AND d.kind = $2"
		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).
			WithArgs(docID, models.DocumentKindCitizenAppeal).
			WillReturnRows(citizenAppealRows(docID, now))
		expectCitizenAppealHydrateEmpty(mock, docID)

		doc, err := repo.Update(req)

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("root update error rolls back", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		repo := NewCitizenAppealRepository(&database.DB{DB: db})
		req := models.UpdateCitizenAppealDocRequest{
			ID:                 uuid.New(),
			RegistrationNumber: "ОГ-003",
			RegistrationDate:   time.Now(),
			Content:            "Текст",
		}

		mock.ExpectBegin()
		mock.ExpectExec(`UPDATE documents SET`).WithArgs(
			req.RegistrationNumber,
			req.RegistrationDate,
			req.Content,
			1,
			req.ID,
			models.DocumentKindCitizenAppeal,
		).WillReturnError(sql.ErrConnDone)
		mock.ExpectRollback()

		doc, err := repo.Update(req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update citizen appeal root")
		assert.Nil(t, doc)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
