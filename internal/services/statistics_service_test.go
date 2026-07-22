package services

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeStatisticsStore struct {
	documentTotal       int
	documentByKind      []models.StatisticsSeriesPoint
	documentByRegistrar []models.StatisticsSeriesPoint
	documentReport      []models.StatisticsReportRow
	nomenclatureOptions []models.StatisticsOption
	userOptions         []models.StatisticsOption
	assignmentMonthly   []models.AssignmentMonthlyPoint
	assignmentExecutor  []models.StatisticsSeriesPoint
	assignmentOverdue   []models.StatisticsReportRow
	assignmentStatuses  []models.StatisticsReportRow
	assignmentReport    []models.StatisticsReportRow
	systemUserCount     int
	systemDocumentCount int
	dbSize              string
	storageSnapshot     models.StorageStatisticsSnapshot
	refreshLeaseGranted bool
	refreshLeaseActive  bool
	storageSnapshotErr  error
	err                 error

	lastDocumentReportGroupBy string
	lastAssignmentOnlyOverdue bool
	lastAssignmentUserID      string
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

func (s *fakeStatisticsStore) GetAssignmentMonthlyOverview(yearStart, yearEnd time.Time) ([]models.AssignmentMonthlyPoint, error) {
	return s.assignmentMonthly, s.err
}

func (s *fakeStatisticsStore) GetAssignmentMonthlyByExecutor(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error) {
	return s.assignmentExecutor, s.err
}

func (s *fakeStatisticsStore) GetAssignmentOverdueRating(yearStart, yearEnd time.Time) ([]models.StatisticsReportRow, error) {
	return s.assignmentOverdue, s.err
}

func (s *fakeStatisticsStore) GetAssignmentStatusCounts() ([]models.StatisticsReportRow, error) {
	return s.assignmentStatuses, s.err
}

func (s *fakeStatisticsStore) GetAssignmentReport(startDate, endDate time.Time, onlyOverdue bool, userID string) ([]models.StatisticsReportRow, error) {
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

func (s *fakeStatisticsStore) GetStorageStatisticsSnapshot() (models.StorageStatisticsSnapshot, error) {
	return s.storageSnapshot, s.storageSnapshotErr
}

func (s *fakeStatisticsStore) TryStartStorageStatisticsRefresh(_ uuid.UUID, _ time.Time) (bool, error) {
	if s.err != nil || !s.refreshLeaseGranted || s.refreshLeaseActive {
		return false, s.err
	}
	s.refreshLeaseActive = true
	return true, nil
}

func (s *fakeStatisticsStore) SaveStorageStatisticsSnapshot(_ uuid.UUID, snapshot models.StorageStatisticsSnapshot) error {
	s.storageSnapshot = snapshot
	s.refreshLeaseActive = false
	return s.err
}

func (s *fakeStatisticsStore) ReleaseStorageStatisticsRefresh(_ uuid.UUID) error {
	s.refreshLeaseActive = false
	return s.err
}

type fakeStatisticsStorage struct {
	objectCount int
	totalSize   string
	err         error
}

func (s *fakeStatisticsStorage) GetStorageInfo(ctx context.Context) (int, string, error) {
	return s.objectCount, s.totalSize, s.err
}

func (s *fakeStatisticsStorage) RefreshStorageInfo(ctx context.Context) (int, string, error) {
	return s.GetStorageInfo(ctx)
}

func setupStatisticsService(t *testing.T, permissions ...string) (*StatisticsService, *fakeStatisticsStore, *fakeStatisticsStorage, *AuthService) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.currentUserID = uuid.New()
	userRepo.On("GetByID", auth.currentUserID).Return(&models.User{ID: auth.currentUserID, IsActive: true}, nil).Maybe()
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(permissions...))
	store := &fakeStatisticsStore{dbSize: "42 MB"}
	storage := &fakeStatisticsStorage{objectCount: 5, totalSize: "10 MB"}

	return NewStatisticsService(store, auth, storage), store, storage, auth
}

func TestStatisticsService_GetDocumentStatistics(t *testing.T) {
	svc, store, _, _ := setupStatisticsService(t, models.SystemPermissionStatsDocuments)
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
	svc, store, _, _ := setupStatisticsService(t, models.SystemPermissionStatsDocuments)
	store.documentReport = []models.StatisticsReportRow{
		{Key: string(models.DocumentKindIncomingLetter), Count: 2},
		{Key: string(models.DocumentKindOutgoingLetter), Count: 3},
	}

	report, err := svc.GetDocumentReport("2026-01-01", "2026-01-31", "", "", "", "")

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "kind", report.GroupBy)
	assert.Equal(t, "kind", store.lastDocumentReportGroupBy)
	assert.Equal(t, 5, report.Total)
	assert.Equal(t, models.DocumentKindIncomingLetter.Label(), report.Rows[0].Name)

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
	svc, store, _, _ := setupStatisticsService(t, models.SystemPermissionStatsDocuments)
	store.nomenclatureOptions = []models.StatisticsOption{{Value: "nom-1", Label: "01-01"}}
	store.userOptions = []models.StatisticsOption{{Value: "user-1", Label: "Пользователь"}}

	filters, err := svc.GetDocumentFilterOptions()

	require.NoError(t, err)
	require.NotNil(t, filters)
	assert.Len(t, filters.Kinds, len(models.AllDocumentKindSpecs()))
	assert.Equal(t, store.nomenclatureOptions, filters.Nomenclature)
	assert.Equal(t, store.userOptions, filters.Users)
}

func TestStatisticsService_GetAssignmentStatistics(t *testing.T) {
	svc, store, _, _ := setupStatisticsService(t, models.SystemPermissionStatsAssignments)
	store.assignmentMonthly = []models.AssignmentMonthlyPoint{{Month: 3, Total: 4, Overdue: 1}}
	store.assignmentExecutor = []models.StatisticsSeriesPoint{{
		Month:        4,
		CategoryKey:  "user-1",
		CategoryName: "Исполнитель",
		Value:        2,
	}}
	store.assignmentOverdue = []models.StatisticsReportRow{{Key: "user-1", Name: "Исполнитель", Count: 1}}
	store.assignmentStatuses = []models.StatisticsReportRow{{Key: "completed", Count: 5}}

	stats, err := svc.GetAssignmentStatistics()

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, "Мар", stats.MonthlyTotals[0].Period)
	assert.Len(t, stats.MonthlyByExecutor, 12)
	assert.Equal(t, "Апр", stats.MonthlyByExecutor[3].Period)
	assert.Equal(t, "Исполнено", stats.StatusCounts[0].Name)
	assert.Equal(t, store.assignmentOverdue, stats.OverdueRating)
}

func TestStatisticsService_GetAssignmentReportAndFilters(t *testing.T) {
	svc, store, _, _ := setupStatisticsService(t, models.SystemPermissionStatsAssignments)
	userID := uuid.New().String()
	store.assignmentReport = []models.StatisticsReportRow{{Key: userID, Name: "Исполнитель", Count: 6}}
	store.userOptions = []models.StatisticsOption{{Value: userID, Label: "Исполнитель"}}

	report, err := svc.GetAssignmentReport("2026-01-01", "2026-01-31", true, userID)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.True(t, report.OnlyOverdue)
	assert.Equal(t, userID, report.UserID)
	assert.Equal(t, 6, report.Total)
	assert.True(t, store.lastAssignmentOnlyOverdue)
	assert.Equal(t, userID, store.lastAssignmentUserID)

	filters, err := svc.GetAssignmentFilterOptions()
	require.NoError(t, err)
	require.NotNil(t, filters)
	assert.Equal(t, store.userOptions, filters.Users)

	report, err = svc.GetAssignmentReport("2026-01-01", "2026-01-31", false, "bad-id")
	require.Error(t, err)
	assert.Nil(t, report)
}

func TestStatisticsService_GetSystemStatistics(t *testing.T) {
	svc, store, _, _ := setupStatisticsService(t, models.SystemPermissionStatsSystem)
	store.systemUserCount = 4
	store.systemDocumentCount = 11
	store.dbSize = "128 MB"
	store.storageSnapshot = models.StorageStatisticsSnapshot{ObjectCount: 9, TotalSize: "256 MB", RefreshedAt: time.Now()}

	stats, err := svc.GetSystemStatistics()

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 4, stats.UserCount)
	assert.Equal(t, 11, stats.TotalDocuments)
	assert.Equal(t, "128 MB", stats.DBSize)
	assert.Equal(t, 9, stats.StorageObjects)
	assert.Equal(t, "256 MB", stats.StorageSize)

	store.storageSnapshotErr = errors.New("storage failed")
	stats, err = svc.GetSystemStatistics()
	require.NoError(t, err)
	assert.Equal(t, "N/A", stats.StorageSize)
	assert.Zero(t, stats.StorageObjects)
}

func TestStatisticsService_GetSystemStatisticsStartsStaleStorageRefreshInBackground(t *testing.T) {
	svc, store, storage, _ := setupStatisticsService(t, models.SystemPermissionStatsSystem)
	store.storageSnapshot = models.StorageStatisticsSnapshot{
		ObjectCount: 3,
		TotalSize:   "3 MB",
		RefreshedAt: time.Now().Add(-storageStatisticsRefreshInterval - time.Second),
	}
	store.refreshLeaseGranted = true
	storage.objectCount = 8
	storage.totalSize = "8 MB"

	stats, err := svc.GetSystemStatistics()

	require.NoError(t, err)
	assert.Equal(t, 3, stats.StorageObjects)
	assert.Equal(t, "3 MB", stats.StorageSize)
	assert.True(t, stats.StorageRefreshInProgress)
	require.Eventually(t, func() bool {
		return store.storageSnapshot.ObjectCount == 8 && store.storageSnapshot.TotalSize == "8 MB" && !store.refreshLeaseActive
	}, time.Second, 10*time.Millisecond)
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

func TestStatisticsService_RejectsMissingPermission(t *testing.T) {
	svc, _, _, _ := setupStatisticsService(t)

	stats, err := svc.GetDocumentStatistics()

	require.Error(t, err)
	assert.Nil(t, stats)
}
