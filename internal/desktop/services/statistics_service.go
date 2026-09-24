package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/releaseassets"
	"github.com/Volkov-D-A/docs-register-and-track/internal/shared/buildinfo"
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

func (s *StatisticsService) GetAssignmentStatistics() (*models.AssignmentStatistics, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetAssignmentStatistics(ctx)
}

func (s *StatisticsService) GetAssignmentReport(startDateStr, endDateStr string, onlyOverdue bool, userID string) (*models.AssignmentStatisticsReport, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetAssignmentReport(ctx, startDateStr, endDateStr, onlyOverdue, userID)
}

func (s *StatisticsService) GetAssignmentFilterOptions() (*models.AssignmentStatisticsFilters, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetAssignmentFilterOptions(ctx)
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

func (s *StatisticsService) GetStorageStatisticsStatus() (*models.StorageStatisticsStatus, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.GetStorageStatisticsStatus(ctx)
}

func (s *StatisticsService) RetryStorageStatisticsRefresh() (*models.StorageStatisticsStatus, error) {
	if s.server == nil {
		return nil, errStatisticsClientNotConfigured
	}
	ctx, cancel := statisticsClientContext()
	defer cancel()
	return s.server.RetryStorageStatisticsRefresh(ctx)
}
