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
	UserCount                int                        `json:"userCount"`
	TotalDocuments           int                        `json:"totalDocuments"`
	DBSize                   string                     `json:"dbSize"`
	DBSizeBytes              int64                      `json:"dbSizeBytes"`
	StorageObjects           int                        `json:"storageObjects"`
	StorageSize              string                     `json:"storageSize"`
	StorageBytes             int64                      `json:"storageBytes"`
	StorageRefreshedAt       *time.Time                 `json:"storageRefreshedAt,omitempty"`
	StorageRefreshInProgress bool                       `json:"storageRefreshInProgress"`
	GeneratedAt              time.Time                  `json:"generatedAt"`
	Service                  SystemServiceStatistics    `json:"service"`
	Usage                    SystemUsageStatistics      `json:"usage"`
	API                      SystemAPIStatistics        `json:"api"`
	Database                 SystemDatabaseStatistics   `json:"database"`
	Outbox                   SystemOutboxStatistics     `json:"outbox"`
	Attachments              SystemAttachmentStatistics `json:"attachments"`
}

type SystemServiceStatistics struct {
	Version               string    `json:"version"`
	APIVersion            string    `json:"apiVersion"`
	State                 string    `json:"state"`
	StartedAt             time.Time `json:"startedAt"`
	UptimeSeconds         int64     `json:"uptimeSeconds"`
	SchemaCurrentVersion  uint      `json:"schemaCurrentVersion"`
	SchemaRequiredVersion uint      `json:"schemaRequiredVersion"`
	SchemaCompatible      bool      `json:"schemaCompatible"`
	SchemaDirty           bool      `json:"schemaDirty"`
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
	SizeBytes                  int64 `json:"sizeBytes"`
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

type SystemAttachmentStatistics struct {
	MissingObjects *int `json:"missingObjects,omitempty"`
	OrphanObjects  *int `json:"orphanObjects,omitempty"`
}

// SystemDiagnostics is the server-owned operational part of system statistics.
type SystemDiagnostics struct {
	Service     SystemServiceStatistics    `json:"service"`
	Usage       SystemUsageStatistics      `json:"usage"`
	API         SystemAPIStatistics        `json:"api"`
	Database    SystemDatabaseStatistics   `json:"database"`
	Outbox      SystemOutboxStatistics     `json:"outbox"`
	Attachments SystemAttachmentStatistics `json:"attachments"`
}

// StorageStatisticsSnapshot is the persisted result of the last complete
// object-storage scan. The lease fields are intentionally not exposed to UI.
type StorageStatisticsSnapshot struct {
	ObjectCount int
	TotalBytes  int64
	RefreshedAt time.Time
}

type StorageStatisticsRefreshState string

const (
	StorageStatisticsRefreshIdle    StorageStatisticsRefreshState = "idle"
	StorageStatisticsRefreshPending StorageStatisticsRefreshState = "pending"
	StorageStatisticsRefreshRunning StorageStatisticsRefreshState = "running"
	StorageStatisticsRefreshFailed  StorageStatisticsRefreshState = "failed"
)

// StorageStatisticsStatus is a lightweight view used while the UI waits for
// a background MinIO scan. It deliberately excludes unrelated system counts.
type StorageStatisticsStatus struct {
	StorageObjects int                           `json:"storageObjects"`
	StorageSize    string                        `json:"storageSize"`
	RefreshedAt    *time.Time                    `json:"refreshedAt,omitempty"`
	State          StorageStatisticsRefreshState `json:"state"`
	LastError      string                        `json:"lastError,omitempty"`
	FailedAt       *time.Time                    `json:"failedAt,omitempty"`
}

// StorageStatisticsRefreshRecord is the persisted snapshot plus coordination
// state used by StatisticsService to decide whether a scan should be started.
type StorageStatisticsRefreshRecord struct {
	Snapshot       StorageStatisticsSnapshot
	RefreshActive  bool
	MutationActive bool
	LastError      string
	FailedAt       time.Time
}
