package services

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// StatisticsService предоставляет бизнес-логику раздела статистики.
type StatisticsService struct {
	repo      StatisticsStore
	auth      *AuthService
	storage   StorageInfoProvider
	lifecycle *OperationLifecycle
}

// NewStatisticsService создает новый экземпляр StatisticsService.
func NewStatisticsService(repo StatisticsStore, auth *AuthService, storage StorageInfoProvider) *StatisticsService {
	return &StatisticsService{repo: repo, auth: auth, storage: storage}
}

func (s *StatisticsService) SetOperationLifecycle(lifecycle *OperationLifecycle) {
	s.lifecycle = lifecycle
}

// GetDocumentStatistics возвращает обзорную статистику по всем документам за текущий год.
func (s *StatisticsService) GetDocumentStatistics() (*models.DocumentStatistics, error) {
	if err := s.requirePermission(models.SystemPermissionStatsDocuments); err != nil {
		return nil, err
	}

	year, yearStart, yearEnd := currentYearRange()

	total, err := s.repo.GetDocumentTotalByYear(yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	byKind, err := s.repo.GetMonthlyDocumentCountsByKind(yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	byKind = completeMonthlySeries(withDocumentKindLabels(byKind), documentKindCategories())

	byRegistrar, err := s.repo.GetMonthlyDocumentCountsByRegistrar(yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	byRegistrar = completeMonthlySeries(withMonthPeriods(byRegistrar), categoriesFromPoints(byRegistrar))

	return &models.DocumentStatistics{
		Year:                        year,
		TotalYear:                   total,
		DocumentsByKindMonthly:      byKind,
		DocumentsByRegistrarMonthly: byRegistrar,
	}, nil
}

// GetDocumentReport возвращает документный отчет за период.
func (s *StatisticsService) GetDocumentReport(startDateStr, endDateStr, groupBy, kindCode, nomenclatureID, userID string) (*models.DocumentStatisticsReport, error) {
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
}

// GetDocumentFilterOptions возвращает значения фильтров для документной статистики.
func (s *StatisticsService) GetDocumentFilterOptions() (*models.DocumentStatisticsFilters, error) {
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
}

// GetAssignmentStatistics возвращает обзорную статистику по всем поручениям.
func (s *StatisticsService) GetAssignmentStatistics() (*models.AssignmentStatistics, error) {
	if err := s.requirePermission(models.SystemPermissionStatsAssignments); err != nil {
		return nil, err
	}

	year, yearStart, yearEnd := currentYearRange()

	monthlyTotals, err := s.repo.GetAssignmentMonthlyOverview(yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	for i := range monthlyTotals {
		monthlyTotals[i].Period = monthLabel(monthlyTotals[i].Month)
	}

	monthlyByExecutor, err := s.repo.GetAssignmentMonthlyByExecutor(yearStart, yearEnd)
	if err != nil {
		return nil, err
	}
	monthlyByExecutor = completeMonthlySeries(withMonthPeriods(monthlyByExecutor), categoriesFromPoints(monthlyByExecutor))

	overdueRating, err := s.repo.GetAssignmentOverdueRating(yearStart, yearEnd)
	if err != nil {
		return nil, err
	}

	statusCounts, err := s.repo.GetAssignmentStatusCounts()
	if err != nil {
		return nil, err
	}
	statusCounts = withAssignmentStatusLabels(statusCounts)

	return &models.AssignmentStatistics{
		Year:              year,
		MonthlyTotals:     monthlyTotals,
		MonthlyByExecutor: monthlyByExecutor,
		OverdueRating:     overdueRating,
		StatusCounts:      statusCounts,
	}, nil
}

// GetAssignmentReport возвращает отчет по поручениям за период.
func (s *StatisticsService) GetAssignmentReport(startDateStr, endDateStr string, onlyOverdue bool, userID string) (*models.AssignmentStatisticsReport, error) {
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
}

// GetAssignmentFilterOptions возвращает значения фильтров для статистики поручений.
func (s *StatisticsService) GetAssignmentFilterOptions() (*models.AssignmentStatisticsFilters, error) {
	if err := s.requirePermission(models.SystemPermissionStatsAssignments); err != nil {
		return nil, err
	}

	users, err := s.repo.GetUserOptions()
	if err != nil {
		return nil, err
	}

	return &models.AssignmentStatisticsFilters{Users: users}, nil
}

// GetSystemStatistics возвращает системную статистику.
func (s *StatisticsService) GetSystemStatistics() (*models.SystemStatistics, error) {
	ctx, release := serviceOperationContext(s.lifecycle)
	defer release()

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
		StorageSize:    "N/A",
	}

	if s.storage != nil {
		objectCount, totalSize, err := s.storage.GetStorageInfo(ctx)
		if err != nil {
			slog.Warn("failed to get storage statistics", "error", err)
		} else {
			result.StorageObjects = objectCount
			result.StorageSize = totalSize
		}
	}

	return result, nil
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
