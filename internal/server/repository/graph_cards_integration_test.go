package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestGraphCardReadersLoadOnlyUsedAssociationsIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	userID := insertIntegrationUser(t, sqlDB, "graph_card_reader")
	db := database.Wrap(sqlDB)
	ids := map[models.DocumentKind]uuid.UUID{}
	for _, item := range []struct {
		kind    models.DocumentKind
		index   string
		docType string
	}{
		{models.DocumentKindIncomingLetter, "GI", "Письмо"},
		{models.DocumentKindCitizenAppeal, "GC", "Обращение"},
		{models.DocumentKindAdministrativeOrder, "GO", "Приказ"},
	} {
		nomID, docID := uuid.New(), uuid.New()
		ids[item.kind] = docID
		execSQL(t, sqlDB, `INSERT INTO nomenclature (id, name, index, year, kind_code) VALUES ($1, 'Graph card', $2, 2026, $3)`, nomID, item.index, item.kind)
		execSQL(t, sqlDB, `INSERT INTO documents (id, kind, nomenclature_id, registration_number, registration_date, document_type, content, created_by) VALUES ($1, $2, $3, $4, '2026-09-01', $5, 'Graph content', $6)`, docID, item.kind, nomID, item.index+"/1", item.docType, userID)
	}
	incomingID := ids[models.DocumentKindIncomingLetter]
	appealID := ids[models.DocumentKindCitizenAppeal]
	orderID := ids[models.DocumentKindAdministrativeOrder]
	execSQL(t, sqlDB, `INSERT INTO incoming_document_details (document_id, incoming_number, incoming_date, sender_signatory) VALUES ($1, 'GI/1', '2026-09-01', 'Signer')`, incomingID)
	execSQL(t, sqlDB, `INSERT INTO citizen_appeal_details (document_id, appeal_date, applicant_full_name, registration_address, appeal_type, applicant_category) VALUES ($1, '2026-09-01', 'Applicant', 'Address', 'заявление', 'person')`, appealID)
	execSQL(t, sqlDB, `INSERT INTO administrative_order_details (document_id, order_number, order_date, title, execution_controller) VALUES ($1, 'GO/1', '2026-09-01', 'Order title', 'Controller')`, orderID)
	orgID := uuid.New()
	execSQL(t, sqlDB, `INSERT INTO organizations (id, name) VALUES ($1, 'First correspondent')`, orgID)
	for _, docID := range []uuid.UUID{incomingID, appealID} {
		execSQL(t, sqlDB, `INSERT INTO document_correspondent_registrations (document_id, registration_number, registration_date, correspondent_org_id) VALUES ($1, 'CORR/1', '2026-09-01', $2)`, docID, orgID)
		execSQL(t, sqlDB, `INSERT INTO document_resolutions (document_id, resolution) VALUES ($1, 'Unused resolution')`, docID)
	}
	execSQL(t, sqlDB, `INSERT INTO administrative_order_acknowledgment_people (document_id, full_name) VALUES ($1, 'Unused person')`, orderID)

	incoming, err := NewIncomingDocumentRepository(db).GetByIDs([]uuid.UUID{incomingID})
	require.NoError(t, err)
	require.Len(t, incoming, 1)
	require.Equal(t, "GI/1", incoming[0].IncomingNumber)
	require.Len(t, incoming[0].Correspondents, 1)
	require.Equal(t, "First correspondent", incoming[0].Correspondents[0].CorrespondentName)
	require.Nil(t, incoming[0].Resolution)

	appeals, err := NewCitizenAppealRepository(db).GetByIDs([]uuid.UUID{appealID})
	require.NoError(t, err)
	require.Len(t, appeals, 1)
	require.Equal(t, "Applicant", appeals[0].ApplicantFullName)
	require.Empty(t, appeals[0].Correspondents)
	require.Empty(t, appeals[0].Resolutions)

	orders, err := NewAdministrativeOrderRepository(db).GetByIDs([]uuid.UUID{orderID})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, "GO/1", orders[0].OrderNumber)
	require.Equal(t, "Controller", orders[0].ExecutionController)
	require.Empty(t, orders[0].AcknowledgmentPeople)
}
