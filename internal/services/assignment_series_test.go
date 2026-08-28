package services

import (
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testAssignmentSeriesStore struct {
	*mocks.AssignmentStore
	series *models.AssignmentSeries
}

func (s *testAssignmentSeriesStore) CreateSeriesWithFirstAssignment(uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time, string, int, string, int, []string, []models.OutboxEvent) (*models.AssignmentSeries, error) {
	return s.series, nil
}
func (s *testAssignmentSeriesStore) GetAssignmentSeries(uuid.UUID) (*models.AssignmentSeries, error) {
	return s.series, nil
}
func (s *testAssignmentSeriesStore) GetAssignmentSeriesByAssignment(uuid.UUID) (*models.AssignmentSeries, error) {
	return s.series, nil
}
func (s *testAssignmentSeriesStore) UpdateAssignmentSeries(uuid.UUID, uuid.UUID, string, string, int, string, int, []string, []models.OutboxEvent) (*models.AssignmentSeries, error) {
	return s.series, nil
}
func (s *testAssignmentSeriesStore) CancelAssignmentSeries(uuid.UUID, uuid.UUID, []models.OutboxEvent) error {
	return nil
}
func (s *testAssignmentSeriesStore) FinishSeriesIterationWithNext(uuid.UUID, uuid.UUID, uuid.UUID, time.Time, string, *time.Time, time.Time, int, uuid.UUID, string, []string, []models.OutboxEvent, []models.OutboxEvent) (*models.Assignment, error) {
	return nil, nil
}
func (s *testAssignmentSeriesStore) GetAssignmentSeriesHistory(uuid.UUID) ([]models.Assignment, error) {
	return nil, nil
}

func TestNextSeriesDeadlineKeepsCalendarRule(t *testing.T) {
	tests := []struct {
		name          string
		current       string
		intervalUnit  string
		intervalValue int
		dayRule       string
		dayOfMonth    int
		want          string
	}{
		{name: "daily", current: "2026-01-01", intervalUnit: "day", intervalValue: 10, dayRule: "same_day", want: "2026-01-11"},
		{name: "weekly", current: "2026-01-01", intervalUnit: "week", intervalValue: 2, dayRule: "same_day", want: "2026-01-15"},
		{name: "first day monthly", current: "2026-01-01", intervalUnit: "month", intervalValue: 1, dayRule: "fixed", dayOfMonth: 1, want: "2026-02-01"},
		{name: "fifteenth monthly", current: "2026-01-15", intervalUnit: "month", intervalValue: 1, dayRule: "fixed", dayOfMonth: 15, want: "2026-02-15"},
		{name: "month end", current: "2026-01-31", intervalUnit: "month", intervalValue: 1, dayRule: "last_day", want: "2026-02-28"},
		{name: "quarter end", current: "2026-03-31", intervalUnit: "month", intervalValue: 3, dayRule: "last_day", want: "2026-06-30"},
		{name: "yearly", current: "2024-02-29", intervalUnit: "year", intervalValue: 1, dayRule: "last_day", want: "2025-02-28"},
		{name: "fixed thirty first clamps short month", current: "2026-01-31", intervalUnit: "month", intervalValue: 1, dayRule: "fixed", dayOfMonth: 31, want: "2026-02-28"},
		{name: "fixed thirty first recovers", current: "2026-02-28", intervalUnit: "month", intervalValue: 1, dayRule: "fixed", dayOfMonth: 31, want: "2026-03-31"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, err := time.Parse("2006-01-02", tt.current)
			require.NoError(t, err)
			assert.Equal(t, tt.want, nextSeriesDeadline(current, tt.intervalUnit, tt.intervalValue, tt.dayRule, tt.dayOfMonth).Format("2006-01-02"))
		})
	}
}

func TestDateMatchesSeriesRule(t *testing.T) {
	value, err := time.Parse("2006-01-02", "2026-02-28")
	require.NoError(t, err)
	assert.True(t, dateMatchesSeriesRule(value, "last_day", 0))
	assert.True(t, dateMatchesSeriesRule(value, "fixed", 31))
	assert.False(t, dateMatchesSeriesRule(value, "fixed", 15))
}

func TestAssignmentSeriesManagementRequiresAssignPermission(t *testing.T) {
	series := &models.AssignmentSeries{ID: uuid.New(), DocumentID: uuid.New(), DocumentKind: string(models.DocumentKindIncomingLetter), Active: true}

	t.Run("executor is forbidden", func(t *testing.T) {
		svc, assignmentRepo, _, _, _ := setupAssignmentService(t, "executor")
		svc.repo = &testAssignmentSeriesStore{AssignmentStore: assignmentRepo, series: series}
		result, err := svc.GetSeries(series.ID.String())
		assert.Nil(t, result)
		requireAppError(t, err, "FORBIDDEN", 403, "недостаточно прав")
	})

	t.Run("assignment manager can read", func(t *testing.T) {
		svc, assignmentRepo, _, _, _ := setupAssignmentService(t, "clerk")
		svc.repo = &testAssignmentSeriesStore{AssignmentStore: assignmentRepo, series: series}
		result, err := svc.GetSeries(series.ID.String())
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, series.ID.String(), result.ID)
	})
}
