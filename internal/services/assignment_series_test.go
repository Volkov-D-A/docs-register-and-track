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
	series               *models.AssignmentSeries
	history              []models.Assignment
	createCalls          int
	createdContent       string
	createdFirstDeadline time.Time
	createdIntervalUnit  string
	createdInterval      int
	createdDayRule       string
	createdDayOfMonth    int
	createdCoExecutors   []string
	createEffects        []models.OutboxEvent
	updateCalls          int
	updatedContent       string
	updatedIntervalUnit  string
	updatedInterval      int
	updatedDayRule       string
	updatedDayOfMonth    int
	updateEffects        []models.OutboxEvent
	cancelCalls          int
	cancelEffects        []models.OutboxEvent
	finishCalls          int
	nextDeadline         time.Time
	nextIteration        int
	finishEffects        []models.OutboxEvent
	nextEffects          []models.OutboxEvent
	finishResult         *models.Assignment
}

func (s *testAssignmentSeriesStore) CreateSeriesWithFirstAssignment(_, _, _, _, _ uuid.UUID, content string, firstDeadline time.Time, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error) {
	s.createCalls++
	s.createdContent = content
	s.createdFirstDeadline = firstDeadline
	s.createdIntervalUnit = intervalUnit
	s.createdInterval = intervalValue
	s.createdDayRule = dayRule
	s.createdDayOfMonth = dayOfMonth
	s.createdCoExecutors = append([]string(nil), coExecutorIDs...)
	s.createEffects = append([]models.OutboxEvent(nil), effects...)
	return s.series, nil
}
func (s *testAssignmentSeriesStore) GetAssignmentSeries(uuid.UUID) (*models.AssignmentSeries, error) {
	return s.series, nil
}
func (s *testAssignmentSeriesStore) GetAssignmentSeriesByAssignment(uuid.UUID) (*models.AssignmentSeries, error) {
	return s.series, nil
}
func (s *testAssignmentSeriesStore) UpdateAssignmentSeries(_, _ uuid.UUID, content, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, _ []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error) {
	s.updateCalls++
	s.updatedContent = content
	s.updatedIntervalUnit = intervalUnit
	s.updatedInterval = intervalValue
	s.updatedDayRule = dayRule
	s.updatedDayOfMonth = dayOfMonth
	s.updateEffects = append([]models.OutboxEvent(nil), effects...)
	return s.series, nil
}
func (s *testAssignmentSeriesStore) CancelAssignmentSeries(_, _ uuid.UUID, effects []models.OutboxEvent) error {
	s.cancelCalls++
	s.cancelEffects = append([]models.OutboxEvent(nil), effects...)
	return nil
}
func (s *testAssignmentSeriesStore) FinishSeriesIterationWithNext(_, _, _ uuid.UUID, _ time.Time, _ string, _ *time.Time, nextDeadline time.Time, nextIteration int, _ uuid.UUID, _ string, _ []string, currentEffects, nextEffects []models.OutboxEvent) (*models.Assignment, error) {
	s.finishCalls++
	s.nextDeadline = nextDeadline
	s.nextIteration = nextIteration
	s.finishEffects = append([]models.OutboxEvent(nil), currentEffects...)
	s.nextEffects = append([]models.OutboxEvent(nil), nextEffects...)
	return s.finishResult, nil
}
func (s *testAssignmentSeriesStore) GetAssignmentSeriesHistory(uuid.UUID) ([]models.Assignment, error) {
	return s.history, nil
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

func TestValidateAssignmentSeriesRequest(t *testing.T) {
	validExecutorID := uuid.New().String()
	valid := models.AssignmentSeriesRequest{
		ExecutorID:    validExecutorID,
		Content:       "Регулярный отчёт",
		FirstDeadline: "2026-03-31",
		IntervalUnit:  "month",
		IntervalValue: 1,
		DayRule:       "last_day",
	}
	tests := []struct {
		name   string
		mutate func(*models.AssignmentSeriesRequest)
	}{
		{name: "invalid executor", mutate: func(r *models.AssignmentSeriesRequest) { r.ExecutorID = "invalid" }},
		{name: "empty content", mutate: func(r *models.AssignmentSeriesRequest) { r.Content = "   " }},
		{name: "invalid unit", mutate: func(r *models.AssignmentSeriesRequest) { r.IntervalUnit = "hour" }},
		{name: "zero interval", mutate: func(r *models.AssignmentSeriesRequest) { r.IntervalValue = 0 }},
		{name: "invalid calendar rule", mutate: func(r *models.AssignmentSeriesRequest) { r.DayRule = "weekday" }},
		{name: "invalid day of month", mutate: func(r *models.AssignmentSeriesRequest) { r.DayRule, r.DayOfMonth = "fixed", 32 }},
		{name: "missing first deadline", mutate: func(r *models.AssignmentSeriesRequest) { r.FirstDeadline = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := valid
			tt.mutate(&request)
			_, _, _, _, _, err := validateAssignmentSeriesRequest(request, true)
			require.Error(t, err)
		})
	}

	daily := valid
	daily.IntervalUnit = "day"
	daily.DayRule = "fixed"
	daily.DayOfMonth = 15
	_, content, dayRule, interval, day, err := validateAssignmentSeriesRequest(daily, true)
	require.NoError(t, err)
	assert.Equal(t, "Регулярный отчёт", content)
	assert.Equal(t, "same_day", dayRule)
	assert.Equal(t, 1, interval)
	assert.Zero(t, day)
}

func TestAssignmentSeriesServiceLifecycle(t *testing.T) {
	documentID, executorID, coExecutorID := uuid.New(), uuid.New(), uuid.New()
	series := &models.AssignmentSeries{
		ID:               uuid.New(),
		DocumentID:       documentID,
		DocumentKind:     string(models.DocumentKindIncomingLetter),
		ExecutorID:       executorID,
		Content:          "Квартальный отчёт",
		IntervalUnit:     "month",
		IntervalValue:    3,
		DayRule:          "last_day",
		Active:           true,
		CurrentIteration: 1,
		UpdatedAt:        time.Now().UTC(),
	}
	svc, assignmentRepo, _, _, _ := setupAssignmentService(t, "clerk")
	store := &testAssignmentSeriesStore{AssignmentStore: assignmentRepo, series: series}
	svc.repo = store

	created, err := svc.CreateSeries(models.AssignmentSeriesRequest{
		DocumentID:    documentID.String(),
		ExecutorID:    executorID.String(),
		Content:       "  Квартальный отчёт  ",
		FirstDeadline: "2026-03-31",
		IntervalUnit:  "month",
		IntervalValue: 3,
		DayRule:       "last_day",
		CoExecutorIDs: []string{coExecutorID.String()},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, 1, store.createCalls)
	assert.Equal(t, "Квартальный отчёт", store.createdContent)
	assert.Equal(t, "2026-03-31", store.createdFirstDeadline.Format("2006-01-02"))
	assert.Equal(t, "last_day", store.createdDayRule)
	assert.Len(t, store.createEffects, 3)
	assert.Equal(t, models.OutboxEventJournal, store.createEffects[0].EventType)

	updated, err := svc.UpdateSeries(series.ID.String(), models.AssignmentSeriesRequest{
		ExecutorID:    executorID.String(),
		Content:       "  Годовой отчёт  ",
		IntervalUnit:  "year",
		IntervalValue: 1,
		DayRule:       "fixed",
		DayOfMonth:    15,
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 1, store.updateCalls)
	assert.Equal(t, "Годовой отчёт", store.updatedContent)
	assert.Equal(t, "year", store.updatedIntervalUnit)
	assert.Equal(t, "fixed", store.updatedDayRule)
	assert.Equal(t, 15, store.updatedDayOfMonth)
	assert.Len(t, store.updateEffects, 1)

	store.history = []models.Assignment{{ID: uuid.New(), DocumentID: documentID, ExecutorID: executorID, IterationNumber: 1, Status: "finished"}}
	history, err := svc.GetSeriesHistory(series.ID.String())
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, 1, history[0].IterationNumber)

	require.NoError(t, svc.CancelSeries(series.ID.String()))
	assert.Equal(t, 1, store.cancelCalls)
	assert.Len(t, store.cancelEffects, 1)
}

func TestAssignmentSeriesFinishCreatesNextIteration(t *testing.T) {
	documentID, assignmentID, executorID, seriesID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	plannedDeadline := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	series := &models.AssignmentSeries{
		ID:            seriesID,
		DocumentID:    documentID,
		ExecutorID:    executorID,
		Content:       "Квартальный отчёт",
		IntervalUnit:  "month",
		IntervalValue: 3,
		DayRule:       "last_day",
		Active:        true,
		UpdatedAt:     time.Now().UTC(),
	}
	existing := &models.Assignment{
		ID:              assignmentID,
		DocumentID:      documentID,
		DocumentKind:    string(models.DocumentKindIncomingLetter),
		ExecutorID:      executorID,
		Content:         "Квартальный отчёт",
		Status:          "completed",
		Report:          "готово",
		SeriesID:        &seriesID,
		IterationNumber: 1,
		PlannedDeadline: &plannedDeadline,
	}
	finished := *existing
	finished.Status = "finished"
	svc, assignmentRepo, _, _, _ := setupAssignmentService(t, "clerk")
	assignmentRepo.On("GetByID", assignmentID).Return(existing, nil).Once()
	store := &testAssignmentSeriesStore{AssignmentStore: assignmentRepo, series: series, finishResult: &finished}
	svc.repo = store

	result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, store.finishCalls)
	assert.Equal(t, 2, store.nextIteration)
	assert.Equal(t, "2026-06-30", store.nextDeadline.Format("2006-01-02"))
	assert.Len(t, store.finishEffects, 1)
	assert.Len(t, store.nextEffects, 2)
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
