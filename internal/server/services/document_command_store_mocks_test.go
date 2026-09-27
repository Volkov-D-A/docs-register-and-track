package services

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// These test doubles record the atomic command call and its journal or outbox arguments.
type incomingAtomicStoreMock struct{ *mocks.IncomingDocStore }

type outgoingAtomicStoreMock struct{ *mocks.OutgoingDocStore }

var (
	_ ports.IncomingDocumentJournalStore = (*incomingAtomicStoreMock)(nil)
	_ ports.IncomingDocumentOutboxStore  = (*incomingAtomicStoreMock)(nil)
	_ ports.OutgoingDocumentJournalStore = (*outgoingAtomicStoreMock)(nil)
	_ ports.OutgoingDocumentOutboxStore  = (*outgoingAtomicStoreMock)(nil)
)

func newIncomingAtomicStoreMock(t *testing.T) *incomingAtomicStoreMock {
	return &incomingAtomicStoreMock{IncomingDocStore: mocks.NewIncomingDocStore(t)}
}

func newOutgoingAtomicStoreMock(t *testing.T) *outgoingAtomicStoreMock {
	return &outgoingAtomicStoreMock{OutgoingDocStore: mocks.NewOutgoingDocStore(t)}
}

func (m *incomingAtomicStoreMock) CreateWithJournal(req models.CreateIncomingDocRequest, action, detailsFormat string) (*models.IncomingDocument, error) {
	args := m.Mock.MethodCalled("CreateWithJournal", req, action, detailsFormat)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IncomingDocument), args.Error(1)
}

func (m *incomingAtomicStoreMock) UpdateWithOutbox(req models.UpdateIncomingDocRequest, effects []models.OutboxEvent) (*models.IncomingDocument, error) {
	args := m.Mock.MethodCalled("UpdateWithOutbox", req, effects)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.IncomingDocument), args.Error(1)
}

func (m *outgoingAtomicStoreMock) CreateWithJournal(req models.CreateOutgoingDocRequest, action, detailsFormat string) (*models.OutgoingDocument, error) {
	args := m.Mock.MethodCalled("CreateWithJournal", req, action, detailsFormat)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OutgoingDocument), args.Error(1)
}

func (m *outgoingAtomicStoreMock) UpdateWithOutbox(req models.UpdateOutgoingDocRequest, effects []models.OutboxEvent) (*models.OutgoingDocument, error) {
	args := m.Mock.MethodCalled("UpdateWithOutbox", req, effects)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OutgoingDocument), args.Error(1)
}

func journalUpdateEffects(documentID, userID uuid.UUID, kind string) any {
	return mock.MatchedBy(func(effects []models.OutboxEvent) bool {
		if len(effects) != 1 || effects[0].EventType != models.OutboxEventJournal {
			return false
		}
		if !strings.HasPrefix(effects[0].DeduplicationKey, kind+":"+documentID.String()+":update:") {
			return false
		}
		var entry models.CreateJournalEntryRequest
		if err := json.Unmarshal([]byte(effects[0].Payload), &entry); err != nil {
			return false
		}
		return entry.DocumentID == documentID &&
			entry.UserID == userID &&
			entry.Action == "UPDATE" &&
			entry.Details == "Документ отредактирован"
	})
}
