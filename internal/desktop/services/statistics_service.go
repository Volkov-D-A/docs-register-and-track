package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/buildinfo"
	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/releaseassets"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// StatisticsService exposes server-owned reports and storage status through HTTP.
type StatisticsService struct{ server serverclient.StatisticsClient }

func NewStatisticsService(client serverclient.StatisticsClient) *StatisticsService {
	return &StatisticsService{server: client}
}

var errStatisticsClientNotConfigured = errors.New("docflow-server statistics client is not configured")

func statisticsClientContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

func (s *StatisticsService) GetDocumentStatistics() (*models.DocumentStatistics, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetDocumentStatistics(ctx)
}

func (s *StatisticsService) GetDocumentReport(startDateStr, endDateStr, groupBy, kindCode, nomenclatureID, userID string) (*models.DocumentStatisticsReport, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetDocumentReport(ctx, startDateStr, endDateStr, groupBy, kindCode, nomenclatureID, userID)
}

func (s *StatisticsService) GetDocumentFilterOptions() (*models.DocumentStatisticsFilters, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetDocumentFilterOptions(ctx)
}

func (s *StatisticsService) GetDeadlineStatistics() (*models.DeadlineStatistics, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetDeadlineStatistics(ctx)
}

func (s *StatisticsService) GetDeadlineReport(startDateStr, endDateStr string, onlyOverdue bool, userID string) (*models.DeadlineStatisticsReport, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetDeadlineReport(ctx, startDateStr, endDateStr, onlyOverdue, userID)
}

func (s *StatisticsService) GetDeadlineFilterOptions() (*models.DeadlineStatisticsFilters, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetDeadlineFilterOptions(ctx)
}

func (s *StatisticsService) GetSystemStatistics() (*models.SystemStatistics, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	stats, err := s.server.GetSystemStatistics(ctx)
	if err != nil || stats == nil {
		return stats, err
	}
	release, err := releaseassets.CurrentVersion()
	if err != nil {
		return nil, err
	}
	identity := buildinfo.Current()
	stats.ClientBuildVersion = identity.Version(release)
	stats.ClientRevision = identity.Revision
	stats.ClientDirty = identity.Fingerprint != ""
	return stats, nil
}
