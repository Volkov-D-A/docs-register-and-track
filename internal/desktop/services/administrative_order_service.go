package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"time"
)

// AdministrativeOrderService exposes server-owned operations through HTTP.
type AdministrativeOrderService struct {
	server serverclient.AdministrativeOrderAcknowledgmentClient
}

func NewAdministrativeOrderService(client serverclient.AdministrativeOrderAcknowledgmentClient) *AdministrativeOrderService {
	return &AdministrativeOrderService{server: client}
}

var errAdministrativeOrderServiceClientNotConfigured = errors.New("docflow-server administrative order client is not configured")

func (s *AdministrativeOrderService) MarkAcknowledged(personIDStr string) (*dto.AdministrativeOrderAcknowledgmentPerson, error) {
	if s.server == nil {
		return nil, errAdministrativeOrderServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.MarkAdministrativeOrderAcknowledged(ctx, personIDStr)
}
