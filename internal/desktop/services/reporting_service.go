package services

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type ReportingService struct {
	client    serverclient.ReportingClient
	lifecycle *operations.Lifecycle
}

func NewReportingService(client serverclient.ReportingClient, lifecycle *operations.Lifecycle) *ReportingService {
	return &ReportingService{client: client, lifecycle: lifecycle}
}

func (s *ReportingService) operationContext(timeout time.Duration) (context.Context, func()) {
	base, release := s.lifecycle.OperationContext()
	ctx, cancel := context.WithTimeout(base, timeout)
	return ctx, func() { cancel(); release() }
}

func (s *ReportingService) Run(req models.ReportRequest) (*models.ReportResult, error) {
	ctx, cancel := s.operationContext(2 * time.Minute)
	defer cancel()
	return s.client.RunReport(ctx, req)
}

func (s *ReportingService) GetFilters() (*models.ReportFilters, error) {
	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.client.GetReportFilters(ctx)
}

func (s *ReportingService) Export(req models.ReportRequest, format string) (string, error) {
	ctx, cancel := s.operationContext(2 * time.Minute)
	defer cancel()
	filename, content, err := s.client.ExportReport(ctx, req, format)
	if err != nil {
		return "", err
	}
	defer content.Close()
	directory, err := defaultDownloadDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	return writeDownloadFileFromStorage(directory, filename, func(file *os.File) error {
		_, err := io.Copy(file, content)
		return err
	})
}

func (s *ReportingService) CreateSchedule(req models.ReportScheduleRequest) (*models.ReportSchedule, error) {
	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.client.CreateReportSchedule(ctx, req)
}

func (s *ReportingService) ListSchedules() ([]models.ReportSchedule, error) {
	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.client.ListReportSchedules(ctx)
}

func (s *ReportingService) DisableSchedule(id string) error {
	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.client.DisableReportSchedule(ctx, id)
}

func (s *ReportingService) ListRuns(id string) ([]models.ReportRun, error) {
	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.client.ListReportRuns(ctx, id)
}

func (s *ReportingService) DownloadRun(id string) (string, error) {
	ctx, cancel := s.operationContext(2 * time.Minute)
	defer cancel()
	filename, content, err := s.client.DownloadReportRun(ctx, id)
	if err != nil {
		return "", err
	}
	defer content.Close()
	directory, err := defaultDownloadDir()
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}
	return writeDownloadFileFromStorage(directory, filename, func(file *os.File) error { _, err := io.Copy(file, content); return err })
}

func (s *ReportingService) RetryRun(id string) error {
	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.client.RetryReportRun(ctx, id)
}
