package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

// statisticsQueryConcurrency keeps one statistics request from occupying the
// whole database pool (which is intentionally small for the desktop app).
const statisticsQueryConcurrency = 2

const (
	storageStatisticsRefreshInterval = 24 * time.Hour
	storageStatisticsLeaseDuration   = 2 * time.Minute
	storageStatisticsRefreshError    = "Не удалось выполнить сверку хранилища. Повторите попытку."
)

// StatisticsService предоставляет бизнес-логику раздела статистики.
type StatisticsService struct {
	repo      StatisticsStore
	auth      StatisticsPrincipal
	storage   StorageInfoProvider
	lifecycle *OperationLifecycle
	metrics   *observability.Registry
	server    serverclient.StatisticsClient
}

type StatisticsPrincipal interface {
	RequireAuthenticated() error
	HasSystemPermission(string) bool
}

// NewStatisticsService создает новый экземпляр StatisticsService.
func NewStatisticsService(repo StatisticsStore, auth StatisticsPrincipal, storage StorageInfoProvider) *StatisticsService {
	return &StatisticsService{repo: repo, auth: auth, storage: storage}
}

func NewStatisticsServiceWithClient(client serverclient.StatisticsClient) *StatisticsService {
	return &StatisticsService{server: client}
}

func statisticsClientContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}

func (s *StatisticsService) SetOperationLifecycle(lifecycle *OperationLifecycle) {
	s.lifecycle = lifecycle
}

func (s *StatisticsService) SetOperationMetrics(metrics *observability.Registry) { s.metrics = metrics }

// GetDocumentStatistics возвращает обзорную статистику по всем документам за текущий год.
func (s *StatisticsService) GetDocumentStatistics() (*models.DocumentStatistics, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetDocumentStatistics(ctx)
	}
	return measureOperation(s.metrics, "statistics.get_documents", func() (*models.DocumentStatistics, error) {
		if err := s.requirePermission(models.SystemPermissionStatsDocuments); err != nil {
			return nil, err
		}

		year, yearStart, yearEnd := currentYearRange()

		var total int
		var byKind, byRegistrar []models.StatisticsSeriesPoint
		if err := runStatisticsQueries(
			func() error {
				var err error
				total, err = s.repo.GetDocumentTotalByYear(yearStart, yearEnd)
				return err
			},
			func() error {
				var err error
				byKind, err = s.repo.GetMonthlyDocumentCountsByKind(yearStart, yearEnd)
				return err
			},
			func() error {
				var err error
				byRegistrar, err = s.repo.GetMonthlyDocumentCountsByRegistrar(yearStart, yearEnd)
				return err
			},
		); err != nil {
			return nil, err
		}
		byKind = completeMonthlySeries(withDocumentKindLabels(byKind), documentKindCategories())
		byRegistrar = completeMonthlySeries(withMonthPeriods(byRegistrar), categoriesFromPoints(byRegistrar))

		return &models.DocumentStatistics{
			Year:                        year,
			TotalYear:                   total,
			DocumentsByKindMonthly:      byKind,
			DocumentsByRegistrarMonthly: byRegistrar,
		}, nil
	})
}

// GetDocumentReport возвращает документный отчет за период.
func (s *StatisticsService) GetDocumentReport(startDateStr, endDateStr, groupBy, kindCode, nomenclatureID, userID string) (*models.DocumentStatisticsReport, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetDocumentReport(ctx, startDateStr, endDateStr, groupBy, kindCode, nomenclatureID, userID)
	}
	return measureOperation(s.metrics, "statistics.get_document_report", func() (*models.DocumentStatisticsReport, error) {
		if err := s.requirePermission(models.SystemPermissionStatsDocuments); err != nil {
			return nil, err
		}
		if groupBy == "" {
			groupBy = "kind"
		}
		if groupBy != "kind" && groupBy != "nomenclature" && groupBy != "user" {
			return nil, models.NewBadRequest("неподдерживаемая группировка статистики документов")
		}
		if kindCode != "" {
			if _, ok := models.GetDocumentKindSpec(models.DocumentKind(kindCode)); !ok {
				return nil, models.NewBadRequest("неизвестный вид документа")
			}
		}
		if err := validateOptionalUUID(nomenclatureID, "некорректная номенклатура"); err != nil {
			return nil, err
		}
		if err := validateOptionalUUID(userID, "некорректный пользователь"); err != nil {
			return nil, err
		}

		startDate, endDate, err := parseStatisticsDateRange(startDateStr, endDateStr)
		if err != nil {
			return nil, err
		}

		rows, err := s.repo.GetDocumentReport(startDate, endDate, groupBy, kindCode, nomenclatureID, userID)
		if err != nil {
			return nil, err
		}
		if groupBy == "kind" {
			rows = withDocumentKindReportLabels(rows)
		}

		return &models.DocumentStatisticsReport{
			StartDate: startDate.Format("2006-01-02"),
			EndDate:   endDate.Format("2006-01-02"),
			GroupBy:   groupBy,
			Rows:      rows,
			Total:     sumReportRows(rows),
		}, nil
	})
}

// GetDocumentFilterOptions возвращает значения фильтров для документной статистики.
func (s *StatisticsService) GetDocumentFilterOptions() (*models.DocumentStatisticsFilters, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetDocumentFilterOptions(ctx)
	}
	return measureOperation(s.metrics, "statistics.get_document_filters", func() (*models.DocumentStatisticsFilters, error) {
		if err := s.requirePermission(models.SystemPermissionStatsDocuments); err != nil {
			return nil, err
		}

		nomenclature, err := s.repo.GetNomenclatureOptions()
		if err != nil {
			return nil, err
		}
		users, err := s.repo.GetUserOptions()
		if err != nil {
			return nil, err
		}

		return &models.DocumentStatisticsFilters{
			Kinds:        documentKindOptions(),
			Nomenclature: nomenclature,
			Users:        users,
		}, nil
	})
}

// GetAssignmentStatistics возвращает обзорную статистику по всем поручениям.
func (s *StatisticsService) GetAssignmentStatistics() (*models.AssignmentStatistics, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetAssignmentStatistics(ctx)
	}
	return measureOperation(s.metrics, "statistics.get_assignments", func() (*models.AssignmentStatistics, error) {
		if err := s.requirePermission(models.SystemPermissionStatsAssignments); err != nil {
			return nil, err
		}

		year, yearStart, yearEnd := currentYearRange()

		var monthlyTotals []models.AssignmentMonthlyPoint
		var monthlyByExecutor []models.StatisticsSeriesPoint
		var overdueRating, statusCounts []models.StatisticsReportRow
		if err := runStatisticsQueries(
			func() error {
				var err error
				monthlyTotals, err = s.repo.GetAssignmentMonthlyOverview(yearStart, yearEnd)
				return err
			},
			func() error {
				var err error
				monthlyByExecutor, err = s.repo.GetAssignmentMonthlyByExecutor(yearStart, yearEnd)
				return err
			},
			func() error {
				var err error
				overdueRating, err = s.repo.GetAssignmentOverdueRating(yearStart, yearEnd)
				return err
			},
			func() error {
				var err error
				statusCounts, err = s.repo.GetAssignmentStatusCounts()
				return err
			},
		); err != nil {
			return nil, err
		}
		for i := range monthlyTotals {
			monthlyTotals[i].Period = monthLabel(monthlyTotals[i].Month)
		}
		monthlyByExecutor = completeMonthlySeries(withMonthPeriods(monthlyByExecutor), categoriesFromPoints(monthlyByExecutor))
		statusCounts = withAssignmentStatusLabels(statusCounts)

		return &models.AssignmentStatistics{
			Year:              year,
			MonthlyTotals:     monthlyTotals,
			MonthlyByExecutor: monthlyByExecutor,
			OverdueRating:     overdueRating,
			StatusCounts:      statusCounts,
		}, nil
	})
}

// runStatisticsQueries runs independent database queries with a deliberately
// small concurrency limit. Repositories do not yet accept contexts, so after
// an error it stops scheduling new work but lets already-running queries end.
func runStatisticsQueries(tasks ...func() error) error {
	semaphore := make(chan struct{}, statisticsQueryConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, task := range tasks {
		semaphore <- struct{}{}

		mu.Lock()
		shouldStop := firstErr != nil
		mu.Unlock()
		if shouldStop {
			<-semaphore
			break
		}

		wg.Add(1)
		go func(task func() error) {
			defer wg.Done()
			defer func() { <-semaphore }()

			if err := task(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(task)
	}

	wg.Wait()
	return firstErr
}

// GetAssignmentReport возвращает отчет по поручениям за период.
func (s *StatisticsService) GetAssignmentReport(startDateStr, endDateStr string, onlyOverdue bool, userID string) (*models.AssignmentStatisticsReport, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetAssignmentReport(ctx, startDateStr, endDateStr, onlyOverdue, userID)
	}
	return measureOperation(s.metrics, "statistics.get_assignment_report", func() (*models.AssignmentStatisticsReport, error) {
		if err := s.requirePermission(models.SystemPermissionStatsAssignments); err != nil {
			return nil, err
		}
		if err := validateOptionalUUID(userID, "некорректный пользователь"); err != nil {
			return nil, err
		}

		startDate, endDate, err := parseStatisticsDateRange(startDateStr, endDateStr)
		if err != nil {
			return nil, err
		}

		rows, err := s.repo.GetAssignmentReport(startDate, endDate, onlyOverdue, userID)
		if err != nil {
			return nil, err
		}

		return &models.AssignmentStatisticsReport{
			StartDate:   startDate.Format("2006-01-02"),
			EndDate:     endDate.Format("2006-01-02"),
			OnlyOverdue: onlyOverdue,
			UserID:      userID,
			Rows:        rows,
			Total:       sumReportRows(rows),
		}, nil
	})
}

// GetAssignmentFilterOptions возвращает значения фильтров для статистики поручений.
func (s *StatisticsService) GetAssignmentFilterOptions() (*models.AssignmentStatisticsFilters, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetAssignmentFilterOptions(ctx)
	}
	return measureOperation(s.metrics, "statistics.get_assignment_filters", func() (*models.AssignmentStatisticsFilters, error) {
		if err := s.requirePermission(models.SystemPermissionStatsAssignments); err != nil {
			return nil, err
		}

		users, err := s.repo.GetUserOptions()
		if err != nil {
			return nil, err
		}

		return &models.AssignmentStatisticsFilters{Users: users}, nil
	})
}

// GetSystemStatistics возвращает системную статистику.
func (s *StatisticsService) GetSystemStatistics() (*models.SystemStatistics, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetSystemStatistics(ctx)
	}
	return measureOperation(s.metrics, "statistics.get_system", func() (*models.SystemStatistics, error) {
		if err := s.requirePermission(models.SystemPermissionStatsSystem); err != nil {
			return nil, err
		}

		userCount, err := s.repo.GetSystemUserCount()
		if err != nil {
			return nil, err
		}
		documentCount, err := s.repo.GetSystemDocumentCount()
		if err != nil {
			return nil, err
		}

		result := &models.SystemStatistics{
			UserCount:      userCount,
			TotalDocuments: documentCount,
			DBSize:         s.repo.GetDBSize(),
			StorageSize:    "Нет данных",
		}

		if s.storage != nil {
			record, err := s.repo.GetStorageStatisticsRefreshRecord()
			if err != nil {
				slog.Warn("failed to get persisted storage statistics", "error", err)
			} else {
				snapshot := record.Snapshot
				result.StorageObjects = snapshot.ObjectCount
				result.StorageSize = formatStorageSize(snapshot.TotalBytes)
				if !snapshot.RefreshedAt.IsZero() {
					refreshedAt := snapshot.RefreshedAt
					result.StorageRefreshedAt = &refreshedAt
				}
				status, statusErr := s.ensureStorageStatisticsStatus(record)
				if statusErr != nil {
					slog.Warn("failed to get storage statistics refresh status", "error", statusErr)
				} else {
					result.StorageRefreshInProgress = status.State == models.StorageStatisticsRefreshPending || status.State == models.StorageStatisticsRefreshRunning
				}
			}
		}

		return result, nil
	})
}

// GetStorageStatisticsStatus returns only the storage snapshot and refresh
// lifecycle, making it cheap enough for bounded UI polling.
func (s *StatisticsService) GetStorageStatisticsStatus() (*models.StorageStatisticsStatus, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.GetStorageStatisticsStatus(ctx)
	}
	if err := s.requirePermission(models.SystemPermissionStatsSystem); err != nil {
		return nil, err
	}
	if s.storage == nil {
		return &models.StorageStatisticsStatus{StorageSize: "Нет данных", State: models.StorageStatisticsRefreshIdle}, nil
	}
	record, err := s.repo.GetStorageStatisticsRefreshRecord()
	if err != nil {
		return nil, err
	}
	return s.ensureStorageStatisticsStatus(record)
}

// RetryStorageStatisticsRefresh acknowledges the last failure and starts a new
// scan as soon as no attachment mutation is active.
func (s *StatisticsService) RetryStorageStatisticsRefresh() (*models.StorageStatisticsStatus, error) {
	if s.server != nil {
		ctx, cancel := statisticsClientContext()
		defer cancel()
		return s.server.RetryStorageStatisticsRefresh(ctx)
	}
	if err := s.requirePermission(models.SystemPermissionStatsSystem); err != nil {
		return nil, err
	}
	if s.storage == nil {
		return &models.StorageStatisticsStatus{StorageSize: "Нет данных", State: models.StorageStatisticsRefreshIdle}, nil
	}
	if err := s.repo.ClearStorageStatisticsRefreshError(); err != nil {
		return nil, err
	}
	record, err := s.repo.GetStorageStatisticsRefreshRecord()
	if err != nil {
		return nil, err
	}
	return s.ensureStorageStatisticsStatus(record)
}

func (s *StatisticsService) ensureStorageStatisticsStatus(record models.StorageStatisticsRefreshRecord) (*models.StorageStatisticsStatus, error) {
	stale := record.Snapshot.RefreshedAt.IsZero() || time.Since(record.Snapshot.RefreshedAt) >= storageStatisticsRefreshInterval
	if stale && !record.RefreshActive && !record.MutationActive && record.LastError == "" {
		token := uuid.New()
		started, err := s.repo.TryStartStorageStatisticsRefresh(token, time.Now().Add(storageStatisticsLeaseDuration))
		if err != nil {
			return nil, err
		}
		if started {
			record.RefreshActive = true
			go s.refreshStorageStatistics(token)
		} else {
			record, err = s.repo.GetStorageStatisticsRefreshRecord()
			if err != nil {
				return nil, err
			}
			stale = record.Snapshot.RefreshedAt.IsZero() || time.Since(record.Snapshot.RefreshedAt) >= storageStatisticsRefreshInterval
		}
	}

	state := models.StorageStatisticsRefreshIdle
	switch {
	case record.RefreshActive:
		state = models.StorageStatisticsRefreshRunning
	case record.MutationActive || (stale && record.LastError == ""):
		state = models.StorageStatisticsRefreshPending
	case record.LastError != "":
		state = models.StorageStatisticsRefreshFailed
	}
	status := &models.StorageStatisticsStatus{
		StorageObjects: record.Snapshot.ObjectCount,
		StorageSize:    formatStorageSize(record.Snapshot.TotalBytes),
		State:          state,
		LastError:      record.LastError,
	}
	if !record.Snapshot.RefreshedAt.IsZero() {
		refreshedAt := record.Snapshot.RefreshedAt
		status.RefreshedAt = &refreshedAt
	}
	if !record.FailedAt.IsZero() {
		failedAt := record.FailedAt
		status.FailedAt = &failedAt
	}
	return status, nil
}

func (s *StatisticsService) refreshStorageStatistics(token uuid.UUID) {
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()

	objectCount, totalBytes, err := s.storage.RefreshStorageUsage(ctx)
	if err != nil {
		slog.Warn("failed to refresh storage statistics", "error", err)
		if failureErr := s.repo.FailStorageStatisticsRefresh(token, storageStatisticsRefreshError, time.Now()); failureErr != nil {
			slog.Warn("failed to record storage statistics refresh failure", "error", failureErr)
		}
		return
	}

	if err := s.repo.SaveStorageStatisticsSnapshot(token, models.StorageStatisticsSnapshot{
		ObjectCount: objectCount,
		TotalBytes:  totalBytes,
		RefreshedAt: time.Now(),
	}); err != nil {
		slog.Warn("failed to save refreshed storage statistics", "error", err)
		if failureErr := s.repo.FailStorageStatisticsRefresh(token, storageStatisticsRefreshError, time.Now()); failureErr != nil {
			slog.Warn("failed to record storage statistics snapshot failure", "error", failureErr)
		}
	}
}

func formatStorageSize(bytes int64) string {
	const (
		kb = int64(1024)
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func (s *StatisticsService) requirePermission(permission string) error {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return err
	}
	if !s.auth.HasSystemPermission(permission) {
		return models.NewForbidden("Недостаточно прав для просмотра статистики")
	}
	return nil
}

func currentYearRange() (int, time.Time, time.Time) {
	now := time.Now()
	yearStart := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
	return now.Year(), yearStart, yearStart.AddDate(1, 0, 0)
}

func parseStatisticsDateRange(startDateStr, endDateStr string) (time.Time, time.Time, error) {
	if startDateStr == "" || endDateStr == "" {
		now := time.Now()
		start := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(1, 0, -1), nil
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, models.NewBadRequestWrapped("неверный формат даты начала", err)
	}
	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return time.Time{}, time.Time{}, models.NewBadRequestWrapped("неверный формат даты окончания", err)
	}
	if endDate.Before(startDate) {
		return time.Time{}, time.Time{}, models.NewBadRequest("дата окончания не может быть раньше даты начала")
	}

	return startDate, endDate, nil
}

func validateOptionalUUID(value, message string) error {
	if value == "" {
		return nil
	}
	if _, err := uuid.Parse(value); err != nil {
		return models.NewBadRequest(message)
	}
	return nil
}

func documentKindOptions() []models.StatisticsOption {
	specs := models.AllDocumentKindSpecs()
	options := make([]models.StatisticsOption, 0, len(specs))
	for _, spec := range specs {
		options = append(options, models.StatisticsOption{
			Value: string(spec.Code),
			Label: spec.Name,
		})
	}
	return options
}

func documentKindCategories() map[string]string {
	categories := make(map[string]string)
	for _, option := range documentKindOptions() {
		categories[option.Value] = option.Label
	}
	return categories
}

func withDocumentKindLabels(points []models.StatisticsSeriesPoint) []models.StatisticsSeriesPoint {
	for i := range points {
		points[i].CategoryName = models.DocumentKind(points[i].CategoryKey).Label()
		points[i].Period = monthLabel(points[i].Month)
	}
	return points
}

func withDocumentKindReportLabels(rows []models.StatisticsReportRow) []models.StatisticsReportRow {
	for i := range rows {
		rows[i].Name = models.DocumentKind(rows[i].Key).Label()
	}
	return rows
}

func withAssignmentStatusLabels(rows []models.StatisticsReportRow) []models.StatisticsReportRow {
	labels := map[string]string{
		"new":         "Новое",
		"in_progress": "В работе",
		"completed":   "Исполнено",
		"finished":    "Завершено",
		"returned":    "Возврат",
		"cancelled":   "Отменено",
	}
	for i := range rows {
		if label, ok := labels[rows[i].Key]; ok {
			rows[i].Name = label
		}
	}
	return rows
}

func withMonthPeriods(points []models.StatisticsSeriesPoint) []models.StatisticsSeriesPoint {
	for i := range points {
		points[i].Period = monthLabel(points[i].Month)
	}
	return points
}

func categoriesFromPoints(points []models.StatisticsSeriesPoint) map[string]string {
	categories := make(map[string]string)
	for _, point := range points {
		categories[point.CategoryKey] = point.CategoryName
	}
	return categories
}

func completeMonthlySeries(points []models.StatisticsSeriesPoint, categories map[string]string) []models.StatisticsSeriesPoint {
	if len(categories) == 0 {
		return []models.StatisticsSeriesPoint{}
	}

	values := make(map[string]int)
	for _, point := range points {
		key := fmt.Sprintf("%d:%s", point.Month, point.CategoryKey)
		values[key] = point.Value
	}

	categoryKeys := make([]string, 0, len(categories))
	for categoryKey := range categories {
		categoryKeys = append(categoryKeys, categoryKey)
	}
	sort.Slice(categoryKeys, func(i, j int) bool {
		return categories[categoryKeys[i]] < categories[categoryKeys[j]]
	})

	result := make([]models.StatisticsSeriesPoint, 0, len(categories)*12)
	for _, categoryKey := range categoryKeys {
		categoryName := categories[categoryKey]
		for month := 1; month <= 12; month++ {
			result = append(result, models.StatisticsSeriesPoint{
				Month:        month,
				Period:       monthLabel(month),
				CategoryKey:  categoryKey,
				CategoryName: categoryName,
				Value:        values[fmt.Sprintf("%d:%s", month, categoryKey)],
			})
		}
	}
	return result
}

func sumReportRows(rows []models.StatisticsReportRow) int {
	total := 0
	for _, row := range rows {
		total += row.Count
	}
	return total
}

func monthLabel(month int) string {
	labels := []string{"Янв", "Фев", "Мар", "Апр", "Май", "Июн", "Июл", "Авг", "Сен", "Окт", "Ноя", "Дек"}
	if month < 1 || month > len(labels) {
		return ""
	}
	return labels[month-1]
}
