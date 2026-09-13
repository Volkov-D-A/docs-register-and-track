package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"time"
)

// JournalService exposes server-owned journal reads through HTTP.
type JournalService struct{ server serverclient.JournalClient }

func NewJournalService(client serverclient.JournalClient) *JournalService {
	return &JournalService{server: client}
}

var errJournalServiceClientNotConfigured = errors.New("docflow-server journal client is not configured")

func (s *JournalService) GetByDocumentID(documentIDStr string) ([]dto.JournalEntry, error) {
	if s.server == nil {
		return nil, errJournalServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetDocumentJournal(ctx, documentIDStr)
}
