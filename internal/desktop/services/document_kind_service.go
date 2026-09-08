package services

import (
	"context"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// DocumentKindService предоставляет системные метаданные видов документов.
type DocumentKindService struct {
	server serverclient.CurrentAccessClient
}

func NewDocumentKindService(client serverclient.CurrentAccessClient) *DocumentKindService {
	return &DocumentKindService{server: client}
}

// GetCurrentAccessSummary возвращает текущую access-модель для навигации и UI.
func (s *DocumentKindService) GetCurrentAccessSummary() (*dto.CurrentAccessSummary, error) {
	if s.server == nil {
		return nil, errServerUserAdministrationNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.GetCurrentAccessSummary(ctx)
}
