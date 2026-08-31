package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

var errServerReferenceClientNotConfigured = errors.New("docflow-server reference client is not configured")

// ReferenceService предоставляет бизнес-логику для работы со справочниками.
type ReferenceService struct {
	auth   *AuthService
	server serverclient.ReferenceClient
}

func NewReferenceService(auth *AuthService) *ReferenceService {
	return &ReferenceService{auth: auth}
}

func (s *ReferenceService) SetServerClient(client serverclient.ReferenceClient) {
	s.server = client
}

// GetDocumentTypes возвращает неизменяемый список типов документов из кода.
func (s *ReferenceService) GetDocumentTypes() ([]dto.DocumentType, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	items := make([]dto.DocumentType, 0, len(models.AllowedDocumentTypes()))
	for _, name := range models.AllowedDocumentTypes() {
		items = append(items, dto.DocumentType{ID: name, Name: name})
	}
	return items, nil
}

func (s *ReferenceService) CreateDocumentType(string) (*dto.DocumentType, error) {
	return nil, models.NewBadRequest("типы документов заданы в коде и не редактируются")
}

func (s *ReferenceService) UpdateDocumentType(string, string) error {
	return models.NewBadRequest("типы документов заданы в коде и не редактируются")
}

func (s *ReferenceService) DeleteDocumentType(string) error {
	return models.NewBadRequest("типы документов заданы в коде и не редактируются")
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

func (s *ReferenceService) FindOrCreateResolutionExecutor(name string) (*dto.ResolutionExecutor, error) {
	if s.server == nil {
		return nil, errServerReferenceClientNotConfigured
	}
	ctx, cancel := referenceContext()
	defer cancel()
	return s.server.ResolveResolutionExecutor(ctx, name)
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
