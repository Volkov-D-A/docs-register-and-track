package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// AssignmentService exposes assignments and recurring series through the HTTP API.
type AssignmentService struct{ server serverclient.AssignmentClient }

func NewAssignmentService(client serverclient.AssignmentClient) *AssignmentService {
	return &AssignmentService{server: client}
}

var errAssignmentClientNotConfigured = errors.New("docflow-server assignment client is not configured")

func assignmentClientContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func (s *AssignmentService) Create(
	documentID string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*dto.Assignment, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.CreateAssignment(ctx, documentID, executorID, content, deadline, coExecutorIDs)
}

func (s *AssignmentService) CreateSeries(request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.CreateAssignmentSeries(ctx, request)
}

func (s *AssignmentService) GetSeries(id string) (*dto.AssignmentSeries, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.GetAssignmentSeries(ctx, id)
}

func (s *AssignmentService) GetSeriesHistory(id string) ([]dto.Assignment, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.GetAssignmentSeriesHistory(ctx, id)
}

func (s *AssignmentService) UpdateSeries(id string, request models.AssignmentSeriesRequest) (*dto.AssignmentSeries, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.UpdateAssignmentSeries(ctx, id, request)
}

func (s *AssignmentService) CancelSeries(id string) error {
	if s.server == nil {
		return errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.CancelAssignmentSeries(ctx, id)
}

func (s *AssignmentService) Update(
	id string,
	executorID string,
	content string,
	deadline string,
	coExecutorIDs []string,
) (*dto.Assignment, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.UpdateAssignment(ctx, id, executorID, content, deadline, coExecutorIDs)
}

func (s *AssignmentService) UpdateStatus(id, status, report string) (*dto.Assignment, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.UpdateAssignmentStatus(ctx, id, status, report)
}

func (s *AssignmentService) GetByID(id string) (*dto.Assignment, error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.GetAssignment(ctx, id)
}

func (s *AssignmentService) GetList(filter models.AssignmentFilter) (*dto.PagedResult[dto.Assignment], error) {
	if s.server == nil {
		return nil, errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.ListAssignments(ctx, filter)
}

func (s *AssignmentService) Delete(id string) error {
	if s.server == nil {
		return errAssignmentClientNotConfigured
	}
	ctx, cancel := assignmentClientContext()
	defer cancel()
	return s.server.DeleteAssignment(ctx, id)
}
