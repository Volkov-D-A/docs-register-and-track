package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// IncomingDocStore — интерфейс для работы с входящими документами в хранилище.
type IncomingDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error)
	GetByID(id uuid.UUID) (*models.IncomingDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.IncomingDocument, error)
}

// CitizenAppealDocStore — интерфейс для работы с обращениями граждан в хранилище.
type CitizenAppealDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.CitizenAppealDocument], error)
	GetByID(id uuid.UUID) (*models.CitizenAppealDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.CitizenAppealDocument, error)
}

// AdministrativeOrderDocStore — интерфейс для работы с приказами в хранилище.
type AdministrativeOrderDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error)
	GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.AdministrativeOrderDocument, error)
	GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error)
	GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error)
}

// DocumentStore — интерфейс для общей корневой сущности документа.
type DocumentStore interface {
	GetByID(id uuid.UUID) (*models.Document, error)
	GetByIDs(ids []uuid.UUID) ([]models.Document, error)
}

// DocumentAccessStore — интерфейс для чтения матрицы доступа document-domain.
type DocumentAccessStore interface {
	HasPermission(kindCode, action string, departmentID, userID string) (bool, error)
}

// OutgoingDocStore — интерфейс для работы с исходящими документами в хранилище.
type OutgoingDocStore interface {
	GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error)
	GetByID(id uuid.UUID) (*models.OutgoingDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.OutgoingDocument, error)
}

// ReferenceStore — интерфейс для работы со справочниками организаций и исполнителей резолюции в хранилище.
type ReferenceStore interface {
	FindOrCreateOrganization(name string) (*models.Organization, error)
	FindOrCreateResolutionExecutor(name string) (*models.ResolutionExecutor, error)
}

// LinkStore — интерфейс для работы со связями между документами в хранилище.
type LinkStore interface {
	CreateWithOutbox(ctx context.Context, link *models.DocumentLink, effects []models.OutboxEvent) error
	DeleteWithOutbox(ctx context.Context, id uuid.UUID, effects []models.OutboxEvent) error
	CreateAndCancelOrderWithOutbox(ctx context.Context, link *models.DocumentLink, effects []models.OutboxEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentLink, error)
	GetByDocumentID(ctx context.Context, docID uuid.UUID) ([]models.DocumentLink, error)
	GetGraph(ctx context.Context, rootID uuid.UUID) ([]models.DocumentLink, error)
}

// UserEventStore — интерфейс для работы с персональными событиями.
type UserEventStore interface {
	GetList(userID uuid.UUID, filter models.UserEventFilter) (*models.PagedResult[models.UserEvent], error)
	CountUnread(userID uuid.UUID) (int, error)
	MarkDocumentRead(documentID, userID uuid.UUID, readAt time.Time) error
	MarkAllRead(userID uuid.UUID, readAt time.Time) error
}

// WorkspaceStore reads counts and bounded previews under server-resolved scopes.
type WorkspaceStore interface {
	RecentDocuments(map[models.DocumentKind]models.DocumentAccessScope) ([]models.WorkspaceDocument, error)
}

// StatisticsStore — интерфейс для получения аналитических данных раздела статистики.
type StatisticsStore interface {
	GetDocumentTotalByYear(yearStart, yearEnd time.Time) (int, error)
	GetMonthlyDocumentCountsByKind(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error)
	GetMonthlyDocumentCountsByRegistrar(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error)
	GetDocumentReport(startDate, endDate time.Time, groupBy, kindCode, nomenclatureID, userID string) ([]models.StatisticsReportRow, error)
	GetNomenclatureOptions() ([]models.StatisticsOption, error)
	GetUserOptions() ([]models.StatisticsOption, error)
	GetExecutorOptions() ([]models.StatisticsOption, error)
	GetDeadlineMonthlyOverview(yearStart, yearEnd time.Time) ([]models.DeadlineMonthlyPoint, error)
	GetDeadlineMonthlyByExecutor(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error)
	GetDeadlineOverdueRating(yearStart, yearEnd time.Time) ([]models.StatisticsReportRow, error)
	GetDeadlineStatusCounts() ([]models.StatisticsReportRow, error)
	GetDeadlineReport(startDate, endDate time.Time, onlyOverdue bool, userID string) ([]models.StatisticsReportRow, error)
	GetSystemUserCount() (int, error)
	GetSystemDocumentCount() (int, error)
	GetDBSize() string
}

// SystemDiagnosticsProvider supplies process and service metrics that do not
// belong to the business statistics repository.
type SystemDiagnosticsProvider interface {
	GetSystemDiagnostics() (*models.SystemDiagnostics, error)
}

// JournalStore — интерфейс для работы с журналом действий.
type JournalStore interface {
	GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]models.JournalEntry, error)
}

// AdminAuditLogStore — интерфейс для работы с журналом действий администраторов.
type AdminAuditLogStore interface {
	GetAll(limit, offset int) ([]models.AdminAuditLog, int, error)
}
