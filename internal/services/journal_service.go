package services

import (
	"context"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
)

type JournalService struct {
	repo      JournalStore
	auth      *AuthService
	access    *DocumentAccessService
	lifecycle *OperationLifecycle
	outbox    *OutboxPublisher
}

func NewJournalService(repo JournalStore, auth *AuthService, access *DocumentAccessService) *JournalService {
	return &JournalService{
		repo:   repo,
		auth:   auth,
		access: access,
	}
}

func (s *JournalService) SetOperationLifecycle(lifecycle *OperationLifecycle) {
	s.lifecycle = lifecycle
}

// SetOutboxPublisher routes legacy journal records through durable delivery.
// Operations that do not expose a shared transaction are an intermediate
// migration: the entry is queued after their business commit.
func (s *JournalService) SetOutboxPublisher(publisher *OutboxPublisher) { s.outbox = publisher }

// GetByDocumentID возвращает список записей журнала для заданного документа.
// Этот метод предназначен для вызова из фронтенда Wails.
func (s *JournalService) GetByDocumentID(documentIDStr string) ([]dto.JournalEntry, error) {
	ctx, release := serviceOperationContext(s.lifecycle)
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

// LogAction — это внутренний вспомогательный метод для других сервисов, чтобы логировать действия (создать запись в журнале).
func (s *JournalService) LogAction(ctx context.Context, req models.CreateJournalEntryRequest) error {
	if s.outbox != nil {
		return s.outbox.PublishJournal("journal:"+uuid.NewString(), req)
	}
	if ctx == nil && s.lifecycle != nil {
		opCtx, release := s.lifecycle.OperationContext()
		defer release()
		ctx = opCtx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := s.repo.Create(ctx, req)
	return err
}
