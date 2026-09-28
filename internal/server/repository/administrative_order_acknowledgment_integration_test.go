package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/testutil/integrationdb"
)

func TestAdministrativeOrderAcknowledgmentRepeatedMarkerIntegration(t *testing.T) {
	sqlDB := integrationdb.Open(t)
	db := database.Wrap(sqlDB)
	firstMarker, secondMarker := uuid.New(), uuid.New()
	execSQL(t, sqlDB, `INSERT INTO users (id, login, password_hash, last_name, first_name, no_patronymic) VALUES ($1, 'order-first-marker', 'hash', 'First', 'Marker', TRUE), ($2, 'order-second-marker', 'hash', 'Second', 'Marker', TRUE)`, firstMarker, secondMarker)
	nomenclatureID, documentID, personID := uuid.New(), uuid.New(), uuid.New()
	execSQL(t, sqlDB, `INSERT INTO nomenclature (id, name, index, year, kind_code) VALUES ($1, 'Orders', 'ACK', 2026, 'administrative_order')`, nomenclatureID)
	execSQL(t, sqlDB, `INSERT INTO documents (id, kind, nomenclature_id, registration_number, registration_date, document_type, content, created_by) VALUES ($1, 'administrative_order', $2, 'ACK/1', '2026-09-01', 'Приказ', 'Order', $3)`, documentID, nomenclatureID, firstMarker)
	execSQL(t, sqlDB, `INSERT INTO administrative_order_details (document_id, order_number, order_date, title) VALUES ($1, 'ACK/1', '2026-09-01', 'Order')`, documentID)
	execSQL(t, sqlDB, `INSERT INTO administrative_order_acknowledgment_people (id, document_id, full_name) VALUES ($1, $2, 'Person')`, personID, documentID)

	repo := NewAdministrativeOrderRepository(db)
	repo.SetOutbox(NewOutboxRepository(db))
	firstEvent := models.OutboxEvent{EventType: models.OutboxEventJournal, DeduplicationKey: "administrative-order:" + documentID.String() + ":acknowledge:" + personID.String(), Payload: `{"marker":"first"}`}
	first, err := repo.MarkAcknowledgmentPersonWithOutbox(personID, firstMarker, []models.OutboxEvent{firstEvent})
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, first.AcknowledgedAt)
	require.NotNil(t, first.AcknowledgedBy)
	require.Equal(t, firstMarker, *first.AcknowledgedBy)

	secondEvent := firstEvent
	secondEvent.Payload = `{"marker":"second"}`
	repeated, err := repo.MarkAcknowledgmentPersonWithOutbox(personID, secondMarker, []models.OutboxEvent{secondEvent})
	require.NoError(t, err)
	require.NotNil(t, repeated)
	require.Equal(t, first.AcknowledgedAt, repeated.AcknowledgedAt)
	require.Equal(t, first.AcknowledgedBy, repeated.AcknowledgedBy)
	assertScalar(t, sqlDB, `SELECT COUNT(*) FROM event_outbox WHERE deduplication_key = $1`, []any{firstEvent.DeduplicationKey}, 1)

	missing, err := repo.MarkAcknowledgmentPersonWithOutbox(uuid.New(), secondMarker, []models.OutboxEvent{secondEvent})
	require.NoError(t, err)
	require.Nil(t, missing)
}
