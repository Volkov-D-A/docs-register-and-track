package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"

	"github.com/google/uuid"
)

type JournalService struct {
	repo      ports.JournalStore
	access    *DocumentAccessService
	lifecycle *operations.Lifecycle
}

func NewJournalService(repo ports.JournalStore, access *DocumentAccessService, lifecycle *operations.Lifecycle) *JournalService {
	return &JournalService{
		repo:      repo,
		lifecycle: lifecycle,
		access:    access,
	}
}

// GetByDocumentID возвращает список записей журнала для заданного документа.
func (s *JournalService) GetByDocumentID(documentIDStr string) ([]dto.JournalEntry, error) {
	ctx, release := s.lifecycle.OperationContext()
	defer release()

	docID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, err
	}

	if s.access == nil {
		return nil, models.ErrForbidden
	}
	if err := s.access.RequireViewJournal(docID); err != nil {
		return nil, err
	}

	entries, err := s.repo.GetByDocumentID(ctx, docID)
	if err != nil {
		return nil, err
	}

	return dto.MapJournalEntries(entries), nil
}
