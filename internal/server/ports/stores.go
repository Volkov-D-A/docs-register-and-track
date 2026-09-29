package ports

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// UserStore — интерфейс для работы с пользователями в хранилище (БД).
type UserStore interface {
	GetByLogin(login string) (*models.User, error)
	GetByID(id uuid.UUID) (*models.User, error)
	GetSessionPrincipal(id uuid.UUID) (*models.SessionPrincipal, error)
	GetAll() ([]models.User, error)
	GetExecutors() ([]models.User, error)
	GetEligibleRecipientIDs(candidateIDs []uuid.UUID) (map[uuid.UUID]struct{}, error)
	GetActiveUsers() ([]models.User, error)
	UpdatePassword(userID uuid.UUID, newPasswordHash string) error
	ResetFailedLoginAttempts(userID uuid.UUID) error
	CountUsers() (int, error)
}

// UserSubstitutionStore — интерфейс для работы с замещениями пользователей.
type UserSubstitutionStore interface {
	GetByPrincipalID(principalUserID uuid.UUID) (*models.UserSubstitution, error)
	GetActivePrincipalIDs(substituteUserID uuid.UUID) ([]uuid.UUID, error)
	IsActiveSubstitute(substituteUserID, principalUserID uuid.UUID) (bool, error)
}

// IncomingDocStore — интерфейс для работы с входящими документами в хранилище.
type IncomingDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error)
	GetByID(id uuid.UUID) (*models.IncomingDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.IncomingDocument, error)
	GetCount() (int, error)
}

// CitizenAppealDocStore — интерфейс для работы с обращениями граждан в хранилище.
type CitizenAppealDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.CitizenAppealDocument], error)
	GetByID(id uuid.UUID) (*models.CitizenAppealDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.CitizenAppealDocument, error)
	GetCount() (int, error)
}

// AdministrativeOrderDocStore — интерфейс для работы с приказами в хранилище.
type AdministrativeOrderDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.AdministrativeOrderDocument], error)
	GetByID(id uuid.UUID) (*models.AdministrativeOrderDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.AdministrativeOrderDocument, error)
	GetAcknowledgmentPersonByID(id uuid.UUID) (*models.AdministrativeOrderAcknowledgmentPerson, error)
	GetAcknowledgmentPeople(documentID uuid.UUID) ([]models.AdministrativeOrderAcknowledgmentPerson, error)
	GetCount() (int, error)
}

// DocumentStore — интерфейс для общей корневой сущности документа.
type DocumentStore interface {
	GetByID(id uuid.UUID) (*models.Document, error)
	GetByIDs(ids []uuid.UUID) ([]models.Document, error)
}

// DocumentQueryReader — минимальный интерфейс для общего query-layer документов.
type DocumentQueryReader interface {
	GetByID(id string) (*dto.DocumentCard, error)
}

// DocumentAccessStore — интерфейс для чтения матрицы доступа document-domain.
type DocumentAccessStore interface {
	HasPermission(kindCode, action string, departmentID, userID string) (bool, error)
	HasSystemPermission(permission, userID string) (bool, error)
	GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error)
}

// OutgoingDocStore — интерфейс для работы с исходящими документами в хранилище.
type OutgoingDocStore interface {
	GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error)
	GetByID(id uuid.UUID) (*models.OutgoingDocument, error)
	GetByIDs(ids []uuid.UUID) ([]models.OutgoingDocument, error)
	GetCount() (int, error)
}

// NomenclatureStore — интерфейс для работы с номенклатурой дел в хранилище.
type NomenclatureStore interface {
	GetAll(year int, kindCode string) ([]models.Nomenclature, error)
	GetByID(id uuid.UUID) (*models.Nomenclature, error)
	GetActiveByKind(kindCode string, year int) ([]models.Nomenclature, error)
}

// ReferenceStore — интерфейс для работы со справочниками организаций и исполнителей резолюции в хранилище.
type ReferenceStore interface {
	GetAllOrganizations() ([]models.Organization, error)
	FindOrCreateOrganization(name string) (*models.Organization, error)
	SearchOrganizations(query string) ([]models.Organization, error)
	GetAllResolutionExecutors() ([]models.ResolutionExecutor, error)
	FindOrCreateResolutionExecutor(name string) (*models.ResolutionExecutor, error)
	SearchResolutionExecutors(query string) ([]models.ResolutionExecutor, error)
}

// AssignmentReader provides assignment queries and access checks.
type AssignmentReader interface {
	GetByID(id uuid.UUID) (*models.Assignment, error)
	GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error)
	HasDocumentAccess(userID, documentID uuid.UUID) (bool, error)
	GetAccessibleDocumentIDs(userID uuid.UUID, documentIDs []uuid.UUID) (map[uuid.UUID]struct{}, error)
}

// DepartmentStore — интерфейс для работы с подразделениями в хранилище.
type DepartmentStore interface {
	GetAll() ([]models.Department, error)
	GetNomenclatureIDs(departmentID uuid.UUID) ([]string, error)
}

// SettingsStore — интерфейс для работы с системными настройками в хранилище.
type SettingsStore interface {
	Get(key string) (*models.SystemSetting, error)
	GetAll() ([]models.SystemSetting, error)
}

// AttachmentStore — интерфейс для работы с вложениями (файлами) в хранилище.
type AttachmentStore interface {
	CreateWithOutbox(a *models.Attachment, effects []models.OutboxEvent) error
	MarkDeletingWithEffects(attachment models.Attachment, effects []models.OutboxEvent) error
	MarkDeletingMultipleWithOutbox(attachments []models.Attachment, effects []models.OutboxEvent) error
	GetByID(id uuid.UUID) (*models.Attachment, error)
	GetByDocumentID(docID uuid.UUID) ([]models.Attachment, error)
	GetOlderThan(date time.Time) ([]models.Attachment, error)
}

// FileStorage — интерфейс для работы с внешним файловым хранилищем через S3.
type FileStorage interface {
	// UploadFile загружает файл в хранилище.
	UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error
	DownloadFileToWriter(ctx context.Context, objectName string, writer io.Writer, maxSize int64) error
	// DeleteFile удаляет файл из хранилища.
	DeleteFile(ctx context.Context, objectName string) error
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
	MarkRead(id, userID uuid.UUID, readAt time.Time) error
	MarkDocumentRead(documentID, userID uuid.UUID, readAt time.Time) error
	MarkAllRead(userID uuid.UUID, readAt time.Time) error
}

// WorkspaceStore reads counts and bounded previews under server-resolved scopes.
type WorkspaceStore interface {
	RecentDocuments(map[models.DocumentKind]models.DocumentAccessScope) ([]models.WorkspaceDocument, error)
	AssignmentSummary(models.WorkspaceQuery) (models.WorkspaceAssignmentCounts, []models.WorkspaceAssignment, error)
}

// StatisticsStore — интерфейс для получения аналитических данных раздела статистики.
type StatisticsStore interface {
	GetDocumentTotalByYear(yearStart, yearEnd time.Time) (int, error)
	GetMonthlyDocumentCountsByKind(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error)
	GetMonthlyDocumentCountsByRegistrar(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error)
	GetDocumentReport(startDate, endDate time.Time, groupBy, kindCode, nomenclatureID, userID string) ([]models.StatisticsReportRow, error)
	GetNomenclatureOptions() ([]models.StatisticsOption, error)
	GetUserOptions() ([]models.StatisticsOption, error)
	GetAssignmentMonthlyOverview(yearStart, yearEnd time.Time) ([]models.AssignmentMonthlyPoint, error)
	GetAssignmentMonthlyByExecutor(yearStart, yearEnd time.Time) ([]models.StatisticsSeriesPoint, error)
	GetAssignmentOverdueRating(yearStart, yearEnd time.Time) ([]models.StatisticsReportRow, error)
	GetAssignmentStatusCounts() ([]models.StatisticsReportRow, error)
	GetAssignmentReport(startDate, endDate time.Time, onlyOverdue bool, userID string) ([]models.StatisticsReportRow, error)
	GetSystemUserCount() (int, error)
	GetSystemDocumentCount() (int, error)
	GetDBSize() string
	GetStorageStatisticsRefreshRecord() (models.StorageStatisticsRefreshRecord, error)
	TryStartStorageStatisticsRefresh(token uuid.UUID, leaseUntil time.Time) (bool, error)
	SaveStorageStatisticsSnapshot(token uuid.UUID, snapshot models.StorageStatisticsSnapshot) error
	FailStorageStatisticsRefresh(token uuid.UUID, message string) error
	ClearStorageStatisticsRefreshError() error
}

// SystemDiagnosticsProvider supplies process and service metrics that do not
// belong to the business statistics repository.
type SystemDiagnosticsProvider interface {
	GetSystemDiagnostics() (*models.SystemDiagnostics, error)
}

// StorageInfoProvider — интерфейс для получения информации о файловом хранилище.
type StorageInfoProvider interface {
	RefreshStorageUsage(ctx context.Context) (objectCount int, totalBytes int64, err error)
}

// JournalStore — интерфейс для работы с журналом действий.
type JournalStore interface {
	Create(ctx context.Context, req models.CreateJournalEntryRequest) (uuid.UUID, error)
	GetByDocumentID(ctx context.Context, documentID uuid.UUID) ([]models.JournalEntry, error)
}

// AdminAuditLogStore — интерфейс для работы с журналом действий администраторов.
type AdminAuditLogStore interface {
	Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error)
	GetAll(limit, offset int) ([]models.AdminAuditLog, int, error)
}
