package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"time"
)

// LinkService exposes server-owned operations through HTTP.
type LinkService struct{ server serverclient.LinkClient }

func NewLinkService(client serverclient.LinkClient) *LinkService {
	return &LinkService{server: client}
}

var errLinkServiceClientNotConfigured = errors.New("docflow-server link client is not configured")

func (s *LinkService) LinkDocuments(sourceIDStr, targetIDStr, linkType string) (*dto.DocumentLink, error) {
	if s.server == nil {
		return nil, errLinkServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.LinkDocuments(ctx, sourceIDStr, targetIDStr, linkType)
}

func (s *LinkService) UnlinkDocument(idStr string) error {
	if s.server == nil {
		return errLinkServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.UnlinkDocument(ctx, idStr)
}

func (s *LinkService) GetDocumentLinks(docIDStr string) ([]dto.DocumentLink, error) {
	if s.server == nil {
		return nil, errLinkServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetDocumentLinks(ctx, docIDStr)
}

func (s *LinkService) GetDocumentFlow(rootIDStr string) (*models.GraphData, error) {
	if s.server == nil {
		return nil, errLinkServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetDocumentFlow(ctx, rootIDStr)
}
