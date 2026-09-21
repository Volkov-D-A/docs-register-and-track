package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

var errServerReferenceClientNotConfigured = errors.New("docflow-server reference client is not configured")

// ReferenceService предоставляет UI операции со справочниками.
type ReferenceService struct {
	server serverclient.ReferenceClient
}

func NewReferenceService(client serverclient.ReferenceClient) *ReferenceService {
	return &ReferenceService{server: client}
}

func (s *ReferenceService) GetOrganizations() ([]dto.Organization, error) {
	if s.server == nil {
		return nil, errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.ListOrganizations(ctx, "")
}

func (s *ReferenceService) SearchOrganizations(query string) ([]dto.Organization, error) {
	if s.server == nil {
		return nil, errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.ListOrganizations(ctx, query)
}

func (s *ReferenceService) FindOrCreateOrganization(name string) (*dto.Organization, error) {
	if s.server == nil {
		return nil, errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.ResolveOrganization(ctx, name)
}

func (s *ReferenceService) UpdateOrganization(id, name string) error {
	if s.server == nil {
		return errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.UpdateOrganization(ctx, id, name)
}

func (s *ReferenceService) DeleteOrganization(id string) error {
	if s.server == nil {
		return errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.DeleteOrganization(ctx, id)
}

func (s *ReferenceService) MergeOrganizations(sourceID, targetID string) error {
	if s.server == nil {
		return errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.MergeOrganizations(ctx, sourceID, targetID)
}

func (s *ReferenceService) GetResolutionExecutors() ([]dto.ResolutionExecutor, error) {
	if s.server == nil {
		return nil, errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.ListResolutionExecutors(ctx, "")
}

func (s *ReferenceService) SearchResolutionExecutors(query string) ([]dto.ResolutionExecutor, error) {
	if s.server == nil {
		return nil, errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.ListResolutionExecutors(ctx, query)
}

func (s *ReferenceService) UpdateResolutionExecutor(id, name string) error {
	if s.server == nil {
		return errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.UpdateResolutionExecutor(ctx, id, name)
}

func (s *ReferenceService) DeleteResolutionExecutor(id string) error {
	if s.server == nil {
		return errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.DeleteResolutionExecutor(ctx, id)
}

func referenceContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}
