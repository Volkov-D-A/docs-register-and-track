package models

import "time"

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
	Total int                   `json:"total"`
	Rows  []StatisticsReportRow `json:"rows"`
}

// DeadlineMonthlyPoint описывает помесячную статистику контролей.
type DeadlineMonthlyPoint struct {
	Month   int    `json:"month"`
	Period  string `json:"period"`
	Total   int    `json:"total"`
	Overdue int    `json:"overdue"`
}

// DeadlineStatistics описывает обзорную статистику по контролям.
type DeadlineStatistics struct {
	Year              int                     `json:"year"`
	MonthlyTotals     []DeadlineMonthlyPoint  `json:"monthlyTotals"`
	MonthlyByExecutor []StatisticsSeriesPoint `json:"monthlyByExecutor"`
	OverdueRating     []StatisticsReportRow   `json:"overdueRating"`
	StatusCounts      []StatisticsReportRow   `json:"statusCounts"`
}

// DeadlineStatisticsFilters содержит значения фильтров для статистики контролей.
type DeadlineStatisticsFilters struct {
	Users []StatisticsOption `json:"users"`
}

// DeadlineStatisticsReport описывает отчет по контролям за период.
type DeadlineStatisticsReport struct {
	Total int                   `json:"total"`
	Rows  []StatisticsReportRow `json:"rows"`
}

// SystemStatistics описывает системную статистику.
type SystemStatistics struct {
	ClientBuildVersion string                   `json:"clientBuildVersion"`
	ClientRevision     string                   `json:"clientRevision"`
	ClientDirty        bool                     `json:"clientDirty"`
	UserCount          int                      `json:"userCount"`
	TotalDocuments     int                      `json:"totalDocuments"`
	DBSize             string                   `json:"dbSize"`
	GeneratedAt        time.Time                `json:"generatedAt"`
	Service            SystemServiceStatistics  `json:"service"`
	Usage              SystemUsageStatistics    `json:"usage"`
	API                SystemAPIStatistics      `json:"api"`
	Database           SystemDatabaseStatistics `json:"database"`
	Outbox             SystemOutboxStatistics   `json:"outbox"`
}

type SystemServiceStatistics struct {
	BuildVersion          string    `json:"buildVersion"`
	SourceRevision        string    `json:"sourceRevision"`
	SourceDirty           bool      `json:"sourceDirty"`
	Version               string    `json:"version"`
	State                 string    `json:"state"`
	StartedAt             time.Time `json:"startedAt"`
	UptimeSeconds         int64     `json:"uptimeSeconds"`
	SchemaCurrentVersion  uint      `json:"schemaCurrentVersion"`
	SchemaRequiredVersion uint      `json:"schemaRequiredVersion"`
}

type SystemUsageStatistics struct {
	ActiveUsers15m int `json:"activeUsers15m"`
	ActiveSessions int `json:"activeSessions"`
}

type SystemAPIStatistics struct {
	RequestsSinceStart         int64 `json:"requestsSinceStart"`
	ClientErrorsSinceStart     int64 `json:"clientErrorsSinceStart"`
	ServerErrorsSinceStart     int64 `json:"serverErrorsSinceStart"`
	DeadlineExceededSinceStart int64 `json:"deadlineExceededSinceStart"`
	P95Milliseconds            int64 `json:"p95Milliseconds"`
	InFlight                   int64 `json:"inFlight"`
	SampleWindow               int   `json:"sampleWindow"`
}

type SystemDatabaseStatistics struct {
	PoolOpen                   int   `json:"poolOpen"`
	PoolInUse                  int   `json:"poolInUse"`
	PoolIdle                   int   `json:"poolIdle"`
	PoolMax                    int   `json:"poolMax"`
	WaitCountSinceStart        int64 `json:"waitCountSinceStart"`
	WaitMillisecondsSinceStart int64 `json:"waitMillisecondsSinceStart"`
	OperationsSinceStart       int64 `json:"operationsSinceStart"`
	OperationErrorsSinceStart  int64 `json:"operationErrorsSinceStart"`
	OperationP95Milliseconds   int64 `json:"operationP95Milliseconds"`
}

type SystemOutboxStatistics struct {
	Pending             int   `json:"pending"`
	Processing          int   `json:"processing"`
	Failed              int   `json:"failed"`
	ProcessedSinceStart int64 `json:"processedSinceStart"`
	RetriesSinceStart   int64 `json:"retriesSinceStart"`
}

type SystemDiagnostics struct {
	Service  SystemServiceStatistics  `json:"service"`
	Usage    SystemUsageStatistics    `json:"usage"`
	API      SystemAPIStatistics      `json:"api"`
	Database SystemDatabaseStatistics `json:"database"`
	Outbox   SystemOutboxStatistics   `json:"outbox"`
}
