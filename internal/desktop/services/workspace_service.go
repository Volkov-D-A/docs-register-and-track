package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

// WorkspaceService exposes server-owned operations through HTTP.
type WorkspaceService struct{ server serverclient.WorkspaceClient }

func NewWorkspaceService(client serverclient.WorkspaceClient) *WorkspaceService {
	return &WorkspaceService{server: client}
}

var errWorkspaceServiceClientNotConfigured = errors.New("docflow-server workspace client is not configured")

func (s *WorkspaceService) GetOverview(assignmentMode string) (*dto.WorkspaceOverview, error) {
	if s.server == nil {
		return nil, errWorkspaceServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetWorkspaceOverview(ctx, assignmentMode)
}

func (s *WorkspaceService) GetRecentDocuments() (*dto.WorkspaceDocuments, error) {
	if s.server == nil {
		return nil, errWorkspaceServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetWorkspaceDocuments(ctx)
}
