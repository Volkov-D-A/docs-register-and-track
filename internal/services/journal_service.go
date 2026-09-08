package services

import (
	"context"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	serverservices "github.com/Volkov-D-A/docs-register-and-track/internal/server/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"

	"github.com/google/uuid"
)

type JournalService struct {
	repo      ports.JournalStore
	auth      ports.DocumentAccessPrincipal
	access    *serverservices.DocumentAccessService
	lifecycle *operations.Lifecycle
	server    serverclient.JournalClient
}

func NewJournalService(repo ports.JournalStore, auth ports.DocumentAccessPrincipal, access *serverservices.DocumentAccessService) *JournalService {
	return &JournalService{
		repo:   repo,
		auth:   auth,
		access: access,
	}
}

// NewJournalServiceWithClient creates the desktop adapter for server-owned journal reads.
func NewJournalServiceWithClient(client serverclient.JournalClient) *JournalService {
	return &JournalService{server: client}
}

func (s *JournalService) SetOperationLifecycle(lifecycle *operations.Lifecycle) {
	s.lifecycle = lifecycle
}

// GetByDocumentID возвращает список записей журнала для заданного документа.
// Этот метод предназначен для вызова из фронтенда Wails.
func (s *JournalService) GetByDocumentID(documentIDStr string) ([]dto.JournalEntry, error) {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.GetDocumentJournal(ctx, documentIDStr)
	}
	ctx, release := s.lifecycle.OperationContext()
	defer release()

	docID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, err
	}

	if s.access != nil {
		if err := s.access.RequireViewJournal(docID); err != nil {
			return nil, err
		}
	} else {
		_, err := s.auth.GetCurrentUser()
		if err != nil {
			return nil, err
		}
	}

	entries, err := s.repo.GetByDocumentID(ctx, docID)
	if err != nil {
		return nil, err
	}

	return dto.MapJournalEntries(entries), nil
}
