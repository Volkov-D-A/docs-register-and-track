package repository

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"docflow/internal/database"
	"docflow/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncomingDocumentRepository_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	expectedQuery := `SELECT d.id, d.nomenclature_id, n.index || ' — ' || n.name,
		d.incoming_number, d.incoming_date, d.outgoing_number_sender, d.outgoing_date_sender,
		d.intermediate_number, d.intermediate_date,
		d.document_type_id, dt.name,
		d.subject, d.pages_count, d.content,
		d.sender_org_id, so.name, d.sender_signatory, d.sender_executor,
		d.recipient_org_id, ro.name, d.addressee,
		d.resolution,
		d.created_by, u.full_name,
		d.created_at, d.updated_at
	FROM incoming_documents d
	LEFT JOIN nomenclature n ON d.nomenclature_id = n.id
	LEFT JOIN document_types dt ON d.document_type_id = dt.id
	LEFT JOIN organizations so ON d.sender_org_id = so.id
	LEFT JOIN organizations ro ON d.recipient_org_id = ro.id
	LEFT JOIN users u ON d.created_by = u.id WHERE d.id = $1`

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "nomenclature_id", "nomenclature_name",
			"incoming_number", "incoming_date", "outgoing_number_sender", "outgoing_date_sender",
			"intermediate_number", "intermediate_date",
			"document_type_id", "document_type_name",
			"subject", "pages_count", "content",
			"sender_org_id", "sender_org_name", "sender_signatory", "sender_executor",
			"recipient_org_id", "recipient_org_name", "addressee",
			"resolution",
			"created_by", "created_by_name",
			"created_at", "updated_at",
		}).AddRow(
			docID, uuid.New(), "01-01 — Дело 1",
			"ВХ-123", now, "ИСХ-123", now,
			"", nil,
			uuid.New(), "Тип 1",
			"Тестовая тема", 5, "Содержание",
			uuid.New(), "Орг 1", "Иванов И.И.", "Петров П.П.",
			uuid.New(), "Орг 2", "Сидоров С.С.",
			"В работу",
			uuid.New(), "Создатель",
			now, now,
		)

		mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

		doc, err := repo.GetByID(docID)
		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, docID, doc.ID)
		assert.Equal(t, "ВХ-123", doc.IncomingNumber)
		assert.Equal(t, "Тестовая тема", doc.Subject)
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
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM incoming_documents`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := repo.GetCount()
	require.NoError(t, err)
	assert.Equal(t, 42, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncomingDocumentRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()

	mock.ExpectExec(`DELETE FROM incoming_documents WHERE id = \$1`).WithArgs(docID).WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Delete(docID)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestIncomingDocumentRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewIncomingDocumentRepository(&database.DB{DB: db})
	docID := uuid.New()
	now := time.Now()

	req := models.CreateIncomingDocRequest{
		NomenclatureID: uuid.New().String(),
		IncomingNumber: "ВХ-001",
		IncomingDate:   now,
		DocumentTypeID: uuid.New().String(),
		Subject:        "Тема",
		Content:        "Текст",
	}

	mock.ExpectQuery(`INSERT INTO incoming_documents`).WithArgs(
		req.NomenclatureID, req.IncomingNumber, req.IncomingDate,
		req.OutgoingNumberSender, req.OutgoingDateSender,
		req.IntermediateNumber, req.IntermediateDate,
		req.DocumentTypeID, req.Subject, req.PagesCount, req.Content,
		req.SenderOrgID, req.SenderSignatory, req.SenderExecutor,
		req.RecipientOrgID, req.Addressee, req.Resolution, req.CreatedBy,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(docID))

	// После Create идет вызов GetByID
	expectedQuery := incomingDocSelectBase + " WHERE d.id = $1"
	rows := sqlmock.NewRows([]string{
		"id", "nomenclature_id", "nomenclature_name",
		"incoming_number", "incoming_date", "outgoing_number_sender", "outgoing_date_sender",
		"intermediate_number", "intermediate_date",
		"document_type_id", "document_type_name",
		"subject", "pages_count", "content",
		"sender_org_id", "sender_org_name", "sender_signatory", "sender_executor",
		"recipient_org_id", "recipient_org_name", "addressee",
		"resolution",
		"created_by", "created_by_name",
		"created_at", "updated_at",
	}).AddRow(
		docID, uuid.New(), "01-01", "ВХ-001", now, "", nil, "", nil,
		uuid.New(), "Тип", "Тема", 0, "Текст", uuid.New(), "", "", "",
		uuid.New(), "", "", "", uuid.New(), "", now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(expectedQuery)).WithArgs(docID).WillReturnRows(rows)

	doc, err := repo.Create(req)
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "ВХ-001", doc.IncomingNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}
