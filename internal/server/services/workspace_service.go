package services

import (
	"errors"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// WorkspaceService resolves capabilities and work from the same server principal.
type WorkspaceService struct {
	repo    ports.WorkspaceStore
	auth    ports.DocumentAccessPrincipal
	access  *DocumentAccessService
	metrics *observability.Registry
}

func NewWorkspaceService(repo ports.WorkspaceStore, auth ports.DocumentAccessPrincipal, access *DocumentAccessService, metrics *observability.Registry) *WorkspaceService {
	return &WorkspaceService{repo: repo, auth: auth, access: access, metrics: metrics}
}

// GetRecentDocuments is independent of the assignment mode.
func (s *WorkspaceService) GetRecentDocuments() (*dto.WorkspaceDocuments, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	if s.access == nil || s.repo == nil {
		return nil, models.ErrForbidden
	}
	result := &dto.WorkspaceDocuments{Items: []dto.WorkspaceDocument{}}
	if err := s.access.RequireDomainRead(); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return result, nil
		}
		return nil, err
	}
	scopes := make(map[models.DocumentKind]models.DocumentAccessScope)
	for _, spec := range models.AllDocumentKindSpecs() {
		scope, err := s.access.ResolveReadScope(spec.Code)
		if err != nil {
			return nil, err
		}
		scopes[spec.Code] = *scope
		if scope.UserID != "" {
			result.Available = true
		}
	}
	if !result.Available {
		return result, nil
	}
	items, err := s.repo.RecentDocuments(scopes)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		correspondents := item.Correspondents
		if correspondents == nil {
			correspondents = []string{}
		}
		result.Items = append(result.Items, dto.WorkspaceDocument{
			ID: item.ID.String(), DocumentKind: item.DocumentKind,
			DocumentNumber: item.DocumentNumber, DocumentDate: item.DocumentDate,
			RegisteredAt: item.RegisteredAt, Description: item.Description, Correspondents: correspondents,
		})
	}
	return result, nil
}
