package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

type statisticsClientStub struct {
	serverclient.StatisticsClient
	report func(context.Context, string, string, string, string, string, string) (*models.DocumentStatisticsReport, error)
}

func (c statisticsClientStub) GetDocumentReport(ctx context.Context, start, end, group, kind, nomenclature, user string) (*models.DocumentStatisticsReport, error) {
	return c.report(ctx, start, end, group, kind, nomenclature, user)
}
func TestStatisticsAdapterPreservesReportFiltersAndTimeout(t *testing.T) {
	var requestContext context.Context
	want := &models.DocumentStatisticsReport{Total: 7}
	service := NewStatisticsService(statisticsClientStub{report: func(ctx context.Context, start, end, group, kind, nomenclature, user string) (*models.DocumentStatisticsReport, error) {
		requestContext = ctx
		require.Equal(t, []string{"2026-01-01", "2026-09-13", "user", "incoming_letter", "nomenclature", "user"}, []string{start, end, group, kind, nomenclature, user})
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(2*time.Minute), deadline, time.Second)
		require.NoError(t, ctx.Err())
		return want, nil
	}})
	result, err := service.GetDocumentReport("2026-01-01", "2026-09-13", "user", "incoming_letter", "nomenclature", "user")
	require.NoError(t, err)
	require.Same(t, want, result)
	require.ErrorIs(t, requestContext.Err(), context.Canceled)
}

func TestStatisticsAdapterMissingClient(t *testing.T) {
	s := NewStatisticsService(nil)
	_, err := s.GetDocumentStatistics()
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
	_, err = s.GetDocumentReport("", "", "", "", "", "")
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
	_, err = s.GetDocumentFilterOptions()
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
	_, err = s.GetDeadlineStatistics()
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
	_, err = s.GetDeadlineReport("", "", false, "")
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
	_, err = s.GetDeadlineFilterOptions()
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
	_, err = s.GetSystemStatistics()
	require.ErrorIs(t, err, errStatisticsClientNotConfigured)
}
