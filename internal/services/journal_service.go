package services

import (
	"context"

	"docflow/internal/dto"
	"docflow/internal/models"

	"github.com/google/uuid"
)

type JournalService struct {
	repo JournalStore
	auth *AuthService
}

func NewJournalService(repo JournalStore, auth *AuthService) *JournalService {
	return &JournalService{
		repo: repo,
		auth: auth,
	}
}

// GetByDocumentID возвращает список записей журнала для заданного документа.
// Этот метод предназначен для вызова из фронтенда Wails.
func (s *JournalService) GetByDocumentID(documentIDStr string, documentType string) ([]dto.JournalEntry, error) {
	_, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}

	docID, err := uuid.Parse(documentIDStr)
	if err != nil {
		return nil, err
	}

	entries, err := s.repo.GetByDocumentID(context.Background(), docID, documentType)
	if err != nil {
		return nil, err
	}

	return dto.MapJournalEntries(entries), nil
}

// LogAction — это внутренний вспомогательный метод для других сервисов, чтобы логировать действия (создать запись в журнале).
func (s *JournalService) LogAction(ctx context.Context, req models.CreateJournalEntryRequest) error {
	_, err := s.repo.Create(ctx, req)
	return err
}
