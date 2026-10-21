package services

import (
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
)

type fakeStatisticsStore struct {
	documentTotal       int
	documentByKind      []models.StatisticsSeriesPoint
	documentByRegistrar []models.StatisticsSeriesPoint
	documentReport      []models.StatisticsReportRow
	nomenclatureOptions []models.StatisticsOption
	userOptions         []models.StatisticsOption
	assignmentMonthly   []models.DeadlineMonthlyPoint
	assignmentExecutor  []models.StatisticsSeriesPoint
	assignmentOverdue   []models.StatisticsReportRow
	assignmentStatuses  []models.StatisticsReportRow
	assignmentReport    []models.StatisticsReportRow
	systemUserCount     int
	systemDocumentCount int
	dbSize              string
	err                 error

	lastDocumentReportGroupBy string
	lastAssignmentOnlyOverdue bool
	lastAssignmentUserID      string
}

type fakeSystemDiagnostics struct {
	value *models.SystemDiagnostics
	err   error
}

func (d fakeSystemDiagnostics) GetSystemDiagnostics() (*models.SystemDiagnostics, error) {
	return d.value, d.err
}

func (s *fakeStatisticsStore) GetDocumentTotalByYear(yearStart, yearEnd time.Time) (int, error) {
	return s.documentTotal, s.err
}

func (s *fakeStatisticsStore) GetMonthlyDocumentCountsByKind(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	return s.documentByKind, s.err
}

func (s *fakeStatisticsStore) GetMonthlyDocumentCountsByRegistrar(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	return s.documentByRegistrar, s.err
}

func (s *fakeStatisticsStore) GetDocumentReport(startDate, endDate time.Time, groupBy, kindCode, nomenclatureID, userID string) ([]models.StatisticsReportRow, error) {
	s.lastDocumentReportGroupBy = groupBy
	return s.documentReport, s.err
}

func (s *fakeStatisticsStore) GetNomenclatureOptions() ([]models.StatisticsOption, error) {
	return s.nomenclatureOptions, s.err
}

func (s *fakeStatisticsStore) GetUserOptions() ([]models.StatisticsOption, error) {
	return s.userOptions, s.err
}

func (s *fakeStatisticsStore) GetDeadlineMonthlyOverview(yearStart, yearEnd time.Time) ([]models.DeadlineMonthlyPoint, error) {
	return s.assignmentMonthly, s.err
}

func (s *fakeStatisticsStore) GetDeadlineMonthlyByExecutor(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	return s.assignmentExecutor, s.err
}

func (s *fakeStatisticsStore) GetDeadlineOverdueRating(yearStart, yearEnd time.Time) ([]models.StatisticsReportRow, error) {
	return s.assignmentOverdue, s.err
}

func (s *fakeStatisticsStore) GetDeadlineStatusCounts() ([]models.StatisticsReportRow, error) {
	return s.assignmentStatuses, s.err
}

func (s *fakeStatisticsStore) GetDeadlineReport(startDate, endDate time.Time, onlyOverdue bool, userID string) ([]models.StatisticsReportRow, error) {
	s.lastAssignmentOnlyOverdue = onlyOverdue
	s.lastAssignmentUserID = userID
	return s.assignmentReport, s.err
}

func (s *fakeStatisticsStore) GetSystemUserCount() (int, error) {
	return s.systemUserCount, s.err
}

func (s *fakeStatisticsStore) GetSystemDocumentCount() (int, error) {
	return s.systemDocumentCount, s.err
}

func (s *fakeStatisticsStore) GetDBSize() string {
	return s.dbSize
}

func setupStatisticsService(t *testing.T, permissions ...string) (*StatisticsService, *fakeStatisticsStore, *statisticsPrincipalStub) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := &statisticsPrincipalStub{newDocumentPrincipalStub(userRepo)}
	auth.currentUserID = uuid.New()
	userRepo.On("GetByID", auth.currentUserID).Return(&models.User{ID: auth.currentUserID, IsActive: true}, nil).Maybe()
	auth.systemPermissions = permissions
	store := &fakeStatisticsStore{dbSize: "42 MB"}

	return NewStatisticsService(store, auth, StatisticsOptions{}), store, auth
}

func TestStatisticsService_GetDocumentStatistics(t *testing.T) {
	svc, store, _ := setupStatisticsService(t, models.SystemPermissionReferences)
	store.documentTotal = 7
	store.documentByKind = []models.StatisticsSeriesPoint{{
		Month:       1,
		CategoryKey: string(models.DocumentKindIncomingLetter),
		Value:       2,
	}}
	store.documentByRegistrar = []models.StatisticsSeriesPoint{{
		Month:        2,
		CategoryKey:  "user-1",
		CategoryName: "Регистратор",
		Value:        3,
	}}

	stats, err := svc.GetDocumentStatistics()

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, time.Now().Year(), stats.Year)
	assert.Equal(t, 7, stats.TotalYear)
	assert.Len(t, stats.DocumentsByKindMonthly, len(models.AllDocumentKindSpecs())*12)
	assert.Contains(t, stats.DocumentsByKindMonthly, models.StatisticsSeriesPoint{
		Month:        1,
		Period:       "Янв",
		CategoryKey:  string(models.DocumentKindIncomingLetter),
		CategoryName: models.DocumentKindIncomingLetter.Label(),
		Value:        2,
	})
	assert.Len(t, stats.DocumentsByRegistrarMonthly, 12)
	assert.Equal(t, "Фев", stats.DocumentsByRegistrarMonthly[1].Period)
	assert.Equal(t, 3, stats.DocumentsByRegistrarMonthly[1].Value)
}

func TestRunStatisticsQueries_LimitsConcurrentQueries(t *testing.T) {
	started := make(chan struct{}, statisticsQueryConcurrency)
	release := make(chan struct{})
	done := make(chan error, 1)
	var active, maximum atomic.Int32

	task := func() error {
		current := active.Add(1)
		for {
			observed := maximum.Load()
			if current <= observed || maximum.CompareAndSwap(observed, current) {
				break
			}
		}
		started <- struct{}{}
		<-release
		active.Add(-1)
		return nil
	}

	go func() {
		done <- runStatisticsQueries(task, task, task, task)
	}()

	<-started
	<-started
	assert.EqualValues(t, statisticsQueryConcurrency, maximum.Load())

	close(release)
	require.NoError(t, <-done)
	assert.EqualValues(t, 0, active.Load())
}

func TestStatisticsService_GetDocumentReport(t *testing.T) {
	svc, store, _ := setupStatisticsService(t, models.SystemPermissionReferences)
	store.documentReport = []models.StatisticsReportRow{
		{Key: string(models.DocumentKindIncomingLetter), Count: 2},
		{Key: string(models.DocumentKindOutgoingLetter), Count: 3},
	}

	report, err := svc.GetDocumentReport("2026-01-01", "2026-01-31", "", "", "", "")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "kind", store.lastDocumentReportGroupBy)
	assert.Equal(t, 5, report.Total)
	assert.Equal(t, models.DocumentKindIncomingLetter.Label(), report.Rows[0].Name)
	body, err := json.Marshal(report)
	require.NoError(t, err)
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Len(t, payload, 2)
	assert.Contains(t, payload, "total")
	assert.Contains(t, payload, "rows")

	report, err = svc.GetDocumentReport("2026-02-01", "2026-01-31", "kind", "", "", "")
	require.Error(t, err)
	assert.Nil(t, report)

	report, err = svc.GetDocumentReport("2026-01-01", "2026-01-31", "bad", "", "", "")
	require.Error(t, err)
	assert.Nil(t, report)

	report, err = svc.GetDocumentReport("2026-01-01", "2026-01-31", "kind", "unknown", "", "")
	require.Error(t, err)
	assert.Nil(t, report)
}

func TestStatisticsService_GetDocumentFilterOptions(t *testing.T) {
	svc, store, _ := setupStatisticsService(t, models.SystemPermissionReferences)
	store.nomenclatureOptions = []models.StatisticsOption{{Value: "nom-1", Label: "01-01"}}
	store.userOptions = []models.StatisticsOption{{Value: "user-1", Label: "Пользователь"}}

	filters, err := svc.GetDocumentFilterOptions()

	require.NoError(t, err)
	require.NotNil(t, filters)
	assert.Len(t, filters.Kinds, len(models.AllDocumentKindSpecs()))
	assert.Equal(t, store.nomenclatureOptions, filters.Nomenclature)
	assert.Equal(t, store.userOptions, filters.Users)
}

func TestStatisticsService_GetDeadlineStatistics(t *testing.T) {
	svc, store, _ := setupStatisticsService(t, models.SystemPermissionReferences)
	store.assignmentMonthly = []models.DeadlineMonthlyPoint{{Month: 3, Total: 4, Overdue: 1}}
	store.assignmentExecutor = []models.StatisticsSeriesPoint{{
		Month:        4,
		CategoryKey:  "user-1",
		CategoryName: "Исполнитель",
		Value:        2,
	}}
	store.assignmentOverdue = []models.StatisticsReportRow{{Key: "user-1", Name: "Исполнитель", Count: 1}}
	store.assignmentStatuses = []models.StatisticsReportRow{{Key: "completed", Count: 5}}

	stats, err := svc.GetDeadlineStatistics()

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, "Мар", stats.MonthlyTotals[0].Period)
	assert.Len(t, stats.MonthlyByExecutor, 12)
	assert.Equal(t, "Апр", stats.MonthlyByExecutor[3].Period)
	assert.Equal(t, "Исполнен", stats.StatusCounts[0].Name)
	assert.Equal(t, store.assignmentOverdue, stats.OverdueRating)
}

func TestStatisticsService_GetDeadlineReportAndFilters(t *testing.T) {
	svc, store, _ := setupStatisticsService(t, models.SystemPermissionReferences)
	userID := uuid.New().String()
	store.assignmentReport = []models.StatisticsReportRow{{Key: userID, Name: "Исполнитель", Count: 6}}
	store.userOptions = []models.StatisticsOption{{Value: userID, Label: "Исполнитель"}}

	report, err := svc.GetDeadlineReport("2026-01-01", "2026-01-31", true, userID)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 6, report.Total)
	assert.True(t, store.lastAssignmentOnlyOverdue)
	assert.Equal(t, userID, store.lastAssignmentUserID)
	body, err := json.Marshal(report)
	require.NoError(t, err)
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Len(t, payload, 2)
	assert.Contains(t, payload, "total")
	assert.Contains(t, payload, "rows")

	filters, err := svc.GetDeadlineFilterOptions()
	require.NoError(t, err)
	require.NotNil(t, filters)
	assert.Equal(t, store.userOptions, filters.Users)

	report, err = svc.GetDeadlineReport("2026-01-01", "2026-01-31", false, "bad-id")
	require.Error(t, err)
	assert.Nil(t, report)
}

func TestStatisticsService_GetSystemStatistics(t *testing.T) {
	svc, store, _ := setupStatisticsService(t, models.SystemPermissionAdmin)
	store.systemUserCount = 4
	store.systemDocumentCount = 11
	store.dbSize = "128 MB"
	stats, err := svc.GetSystemStatistics()
	require.NoError(t, err)
	require.Equal(t, 4, stats.UserCount)
	require.Equal(t, 11, stats.TotalDocuments)
	require.Equal(t, "128 MB", stats.DBSize)
	body, err := json.Marshal(stats)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	for _, key := range []string{"storageObjects", "storageSize", "storageRefreshedAt", "attachments"} {
		require.NotContains(t, payload, key)
	}
}

func TestStatisticsServiceIncludesServerDiagnostics(t *testing.T) {
	svc, _, _ := setupStatisticsService(t, models.SystemPermissionAdmin)
	svc.diagnostics = fakeSystemDiagnostics{value: &models.SystemDiagnostics{
		Service:  models.SystemServiceStatistics{Version: "1.0.6", State: "ready"},
		Usage:    models.SystemUsageStatistics{ActiveUsers15m: 3, ActiveSessions: 4},
		API:      models.SystemAPIStatistics{RequestsSinceStart: 100, P95Milliseconds: 12},
		Database: models.SystemDatabaseStatistics{PoolInUse: 2},
		Outbox:   models.SystemOutboxStatistics{Pending: 1},
	}}

	stats, err := svc.GetSystemStatistics()

	require.NoError(t, err)
	assert.Equal(t, "1.0.6", stats.Service.Version)
	assert.Equal(t, 3, stats.Usage.ActiveUsers15m)
	assert.EqualValues(t, 100, stats.API.RequestsSinceStart)
	assert.Equal(t, 2, stats.Database.PoolInUse)
	assert.Equal(t, 1, stats.Outbox.Pending)
	body, err := json.Marshal(stats)
	require.NoError(t, err)
	var payload struct {
		Service  map[string]json.RawMessage `json:"service"`
		Database map[string]json.RawMessage `json:"database"`
	}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.NotContains(t, payload.Service, "apiVersion")
	assert.NotContains(t, payload.Service, "schemaCompatible")
	assert.NotContains(t, payload.Service, "schemaDirty")
	assert.NotContains(t, payload.Database, "sizeBytes")
}

func TestParseStatisticsDateRange(t *testing.T) {
	t.Run("defaults to current year", func(t *testing.T) {
		start, end, err := parseStatisticsDateRange("", "")

		require.NoError(t, err)
		assert.Equal(t, time.Now().Year(), start.Year())
		assert.Equal(t, time.January, start.Month())
		assert.Equal(t, 1, start.Day())
		assert.Equal(t, time.December, end.Month())
		assert.Equal(t, 31, end.Day())
	})

	t.Run("valid range", func(t *testing.T) {
		start, end, err := parseStatisticsDateRange("2026-01-02", "2026-03-04")

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC), start)
		assert.Equal(t, time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC), end)
	})

	t.Run("invalid start date", func(t *testing.T) {
		start, end, err := parseStatisticsDateRange("bad-date", "2026-03-04")

		require.Error(t, err)
		assert.True(t, start.IsZero())
		assert.True(t, end.IsZero())
	})

	t.Run("invalid end date", func(t *testing.T) {
		start, end, err := parseStatisticsDateRange("2026-01-02", "bad-date")

		require.Error(t, err)
		assert.True(t, start.IsZero())
		assert.True(t, end.IsZero())
	})

	t.Run("end before start", func(t *testing.T) {
		start, end, err := parseStatisticsDateRange("2026-03-04", "2026-01-02")

		require.Error(t, err)
		assert.True(t, start.IsZero())
		assert.True(t, end.IsZero())
	})
}

func TestStatisticsServiceBusinessDataRequiresLoginAndDiagnosticsRequireAdmin(t *testing.T) {
	svc, _, auth := setupStatisticsService(t)
	_, err := svc.GetDocumentStatistics()
	require.NoError(t, err)
	_, err = svc.GetDeadlineStatistics()
	require.NoError(t, err)
	_, err = svc.GetSystemStatistics()
	requireAppError(t, err, "FORBIDDEN", 403, "Недостаточно прав для просмотра статистики")
	auth.currentUserID = uuid.Nil
	_, err = svc.GetDocumentStatistics()
	require.ErrorIs(t, err, models.ErrUnauthorized)
}

// statisticsPrincipalStub uses the same permission source as the server fixtures.
type statisticsPrincipalStub struct{ *documentPrincipalStub }

func (p *statisticsPrincipalStub) HasSystemPermission(permission string) bool {
	return p.RequireSystemPermission(permission) == nil
}

func (s *fakeStatisticsStore) GetExecutorOptions() ([]models.StatisticsOption, error) {
	return s.userOptions, s.err
}
