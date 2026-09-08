package effects

import (
	"encoding/json"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOutboxEventContracts(t *testing.T) {
	id := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	journal := models.CreateJournalEntryRequest{DocumentID: id}
	audit := models.CreateAdminAuditLogRequest{UserID: id, Action: "TEST", Details: "details"}
	user := models.CreateUserEventRequest{RecipientUserID: id}
	tests := []struct {
		name   string
		kind   string
		build  func() (models.OutboxEvent, error)
		decode func(*testing.T, string)
	}{
		{"journal", models.OutboxEventJournal, func() (models.OutboxEvent, error) { return NewJournalOutboxEvent("key", journal) }, func(t *testing.T, payload string) {
			var got models.CreateJournalEntryRequest
			require.NoError(t, json.Unmarshal([]byte(payload), &got))
			require.Equal(t, journal, got)
		}},
		{"audit", models.OutboxEventAudit, func() (models.OutboxEvent, error) { return NewAdminAuditOutboxEvent("key", audit) }, func(t *testing.T, payload string) {
			var got models.CreateAdminAuditLogRequest
			require.NoError(t, json.Unmarshal([]byte(payload), &got))
			require.Equal(t, audit, got)
		}},
		{"user event", models.OutboxEventUserEvent, func() (models.OutboxEvent, error) { return NewUserEventOutboxEvent("key", user) }, func(t *testing.T, payload string) {
			var envelope map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(payload), &envelope))
			require.Len(t, envelope, 1)
			var got models.CreateUserEventRequest
			require.NoError(t, json.Unmarshal(envelope["request"], &got))
			require.Equal(t, user, got)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := tt.build()
			require.NoError(t, err)
			require.Equal(t, tt.kind, event.EventType)
			require.Equal(t, "key", event.DeduplicationKey)
			tt.decode(t, event.Payload)
		})
	}
}

func TestAdminAuditRequiresDeduplicationKey(t *testing.T) {
	_, err := NewAdminAuditOutboxEvent("", models.CreateAdminAuditLogRequest{})
	require.Error(t, err)
}
