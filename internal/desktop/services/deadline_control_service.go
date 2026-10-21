package services

import (
	"context"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"time"
)

type DeadlineControlService struct {
	client    serverclient.DeadlineClient
	lifecycle *operations.Lifecycle
}

func NewDeadlineControlService(client serverclient.DeadlineClient, lifecycle *operations.Lifecycle) *DeadlineControlService {
	return &DeadlineControlService{client: client, lifecycle: lifecycle}
}
func (s *DeadlineControlService) context() (context.Context, func()) {
	parent, release := s.lifecycle.OperationContext()
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	return ctx, func() { cancel(); release() }
}
func (s *DeadlineControlService) ListExecutors(includeInactive bool) ([]models.DeadlineExecutor, error) {
	ctx, done := s.context()
	defer done()
	return s.client.ListDeadlineExecutors(ctx, includeInactive)
}
func (s *DeadlineControlService) SaveExternalExecutor(id string, req models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error) {
	ctx, done := s.context()
	defer done()
	return s.client.SaveExternalDeadlineExecutor(ctx, id, req)
}
func (s *DeadlineControlService) List(req models.DeadlineControlFilter) (*models.DeadlineControlPage, error) {
	ctx, done := s.context()
	defer done()
	return s.client.ListDeadlineControls(ctx, req)
}
func (s *DeadlineControlService) Create(req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	ctx, done := s.context()
	defer done()
	return s.client.CreateDeadlineControl(ctx, req)
}
func (s *DeadlineControlService) Update(id string, req models.DeadlineControlRequest) (*models.DocumentDeadlineControl, error) {
	ctx, done := s.context()
	defer done()
	return s.client.UpdateDeadlineControl(ctx, id, req)
}
func (s *DeadlineControlService) Complete(id string, req models.CompleteDeadlineRequest) (*models.DocumentDeadlineControl, error) {
	ctx, done := s.context()
	defer done()
	return s.client.CompleteDeadlineControl(ctx, id, req)
}
func (s *DeadlineControlService) Cancel(id string, req models.CancelDeadlineRequest) (*models.DocumentDeadlineControl, error) {
	ctx, done := s.context()
	defer done()
	return s.client.CancelDeadlineControl(ctx, id, req)
}
func (s *DeadlineControlService) CreateSeries(req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
	ctx, done := s.context()
	defer done()
	return s.client.CreateDeadlineSeries(ctx, req)
}
func (s *DeadlineControlService) GetSeries(id string) (*models.DocumentDeadlineSeries, error) {
	ctx, done := s.context()
	defer done()
	return s.client.GetDeadlineSeries(ctx, id)
}
func (s *DeadlineControlService) UpdateSeries(id string, req models.DeadlineSeriesRequest) (*models.DocumentDeadlineSeries, error) {
	ctx, done := s.context()
	defer done()
	return s.client.UpdateDeadlineSeries(ctx, id, req)
}
func (s *DeadlineControlService) StopSeries(id string, req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
	ctx, done := s.context()
	defer done()
	return s.client.StopDeadlineSeries(ctx, id, req)
}
func (s *DeadlineControlService) ResumeSeries(id string, req models.DeadlineSeriesActionRequest) (*models.DocumentDeadlineSeries, error) {
	ctx, done := s.context()
	defer done()
	return s.client.ResumeDeadlineSeries(ctx, id, req)
}

func (s *DeadlineControlService) PreviewResume(id string) (*models.DeadlineSeriesResumePreview, error) {
	ctx, done := s.context()
	defer done()
	return s.client.PreviewDeadlineSeriesResume(ctx, id)
}
