package models

// StatisticsOption описывает пункт фильтра для статистических отчетов.
type StatisticsOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// StatisticsSeriesPoint описывает точку временного ряда с категорией.
type StatisticsSeriesPoint struct {
	Month        int    `json:"month"`
	Period       string `json:"period"`
	CategoryKey  string `json:"categoryKey"`
	CategoryName string `json:"categoryName"`
	Value        int    `json:"value"`
}

// StatisticsReportRow описывает строку табличного статистического отчета.
type StatisticsReportRow struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// DocumentStatistics описывает обзорную статистику по документам.
type DocumentStatistics struct {
	Year                        int                     `json:"year"`
	TotalYear                   int                     `json:"totalYear"`
	DocumentsByKindMonthly      []StatisticsSeriesPoint `json:"documentsByKindMonthly"`
	DocumentsByRegistrarMonthly []StatisticsSeriesPoint `json:"documentsByRegistrarMonthly"`
}

// DocumentStatisticsFilters содержит значения фильтров для документной статистики.
type DocumentStatisticsFilters struct {
	Kinds        []StatisticsOption `json:"kinds"`
	Nomenclature []StatisticsOption `json:"nomenclature"`
	Users        []StatisticsOption `json:"users"`
}

// DocumentStatisticsReport описывает отчет по документам за период.
type DocumentStatisticsReport struct {
	StartDate string                `json:"startDate"`
	EndDate   string                `json:"endDate"`
	GroupBy   string                `json:"groupBy"`
	Total     int                   `json:"total"`
	Rows      []StatisticsReportRow `json:"rows"`
}

// AssignmentMonthlyPoint описывает помесячную статистику поручений.
type AssignmentMonthlyPoint struct {
	Month   int    `json:"month"`
	Period  string `json:"period"`
	Total   int    `json:"total"`
	Overdue int    `json:"overdue"`
}

// AssignmentStatistics описывает обзорную статистику по поручениям.
type AssignmentStatistics struct {
	Year              int                      `json:"year"`
	MonthlyTotals     []AssignmentMonthlyPoint `json:"monthlyTotals"`
	MonthlyByExecutor []StatisticsSeriesPoint  `json:"monthlyByExecutor"`
	OverdueRating     []StatisticsReportRow    `json:"overdueRating"`
	StatusCounts      []StatisticsReportRow    `json:"statusCounts"`
}

// AssignmentStatisticsFilters содержит значения фильтров для статистики поручений.
type AssignmentStatisticsFilters struct {
	Users []StatisticsOption `json:"users"`
}

// AssignmentStatisticsReport описывает отчет по поручениям за период.
type AssignmentStatisticsReport struct {
	StartDate   string                `json:"startDate"`
	EndDate     string                `json:"endDate"`
	OnlyOverdue bool                  `json:"onlyOverdue"`
	UserID      string                `json:"userId,omitempty"`
	Total       int                   `json:"total"`
	Rows        []StatisticsReportRow `json:"rows"`
}

// SystemStatistics описывает системную статистику.
type SystemStatistics struct {
	UserCount      int    `json:"userCount"`
	TotalDocuments int    `json:"totalDocuments"`
	DBSize         string `json:"dbSize"`
	StorageObjects int    `json:"storageObjects"`
	StorageSize    string `json:"storageSize"`
}
