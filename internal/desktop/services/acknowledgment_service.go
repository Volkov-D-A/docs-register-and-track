package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

// AcknowledgmentService exposes acknowledgment operations through the HTTP API.
type AcknowledgmentService struct {
	server serverclient.AcknowledgmentClient
}

func NewAcknowledgmentService(client serverclient.AcknowledgmentClient) *AcknowledgmentService {
	return &AcknowledgmentService{server: client}
}

var errAcknowledgmentClientNotConfigured = errors.New("docflow-server acknowledgment client is not configured")

func acknowledgmentClientContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func (s *AcknowledgmentService) Create(
	documentID string,
	content string,
	userIds []string,
) (*dto.Acknowledgment, error) {
	if s.server == nil {
		return nil, errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.CreateAcknowledgment(ctx, documentID, content, userIds)
}

func (s *AcknowledgmentService) GetList(documentID string) ([]dto.Acknowledgment, error) {
	if s.server == nil {
		return nil, errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.ListAcknowledgments(ctx, documentID)
}

func (s *AcknowledgmentService) GetPendingForCurrentUser() ([]dto.Acknowledgment, error) {
	if s.server == nil {
		return nil, errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.ListPendingAcknowledgments(ctx)
}

func (s *AcknowledgmentService) GetCurrentUserPendingByDocument(documentID string) ([]dto.Acknowledgment, error) {
	if s.server == nil {
		return nil, errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.ListPendingAcknowledgmentsByDocument(ctx, documentID)
}

func (s *AcknowledgmentService) GetAllActive() ([]dto.Acknowledgment, error) {
	if s.server == nil {
		return nil, errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.ListActiveAcknowledgments(ctx)
}

func (s *AcknowledgmentService) MarkConfirmed(ackID string) error {
	if s.server == nil {
		return errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.MarkAcknowledgmentConfirmed(ctx, ackID)
}

func (s *AcknowledgmentService) Delete(id string) error {
	if s.server == nil {
		return errAcknowledgmentClientNotConfigured
	}
	ctx, cancel := acknowledgmentClientContext()
	defer cancel()
	return s.server.DeleteAcknowledgment(ctx, id)
}
