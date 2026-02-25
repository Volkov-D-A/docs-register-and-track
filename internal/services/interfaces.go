package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
)

// UserStore — интерфейс для работы с пользователями.
type UserStore interface {
	GetByLogin(login string) (*models.User, error)
	GetByID(id uuid.UUID) (*models.User, error)
	GetAll() ([]models.User, error)
	Create(req models.CreateUserRequest) (*models.User, error)
	Update(req models.UpdateUserRequest) (*models.User, error)
	GetExecutors() ([]models.User, error)
	UpdatePassword(userID uuid.UUID, newPasswordHash string) error
	ResetPassword(userID uuid.UUID, newPassword string) error
	UpdateProfile(userID uuid.UUID, req models.UpdateProfileRequest) error
	CountUsers() (int, error)
}

// IncomingDocStore — интерфейс для работы с входящими документами.
type IncomingDocStore interface {
	GetList(filter models.DocumentFilter) (*models.PagedResult[models.IncomingDocument], error)
	GetByID(id uuid.UUID) (*models.IncomingDocument, error)
	Create(
		nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy uuid.UUID,
		incomingNumber string, incomingDate time.Time,
		outgoingNumberSender string, outgoingDateSender time.Time,
		intermediateNumber *string, intermediateDate *time.Time,
		subject, content string, pagesCount int,
		senderSignatory, senderExecutor, addressee string,
		resolution *string,
	) (*models.IncomingDocument, error)
	Update(
		id uuid.UUID,
		documentTypeID, senderOrgID, recipientOrgID uuid.UUID,
		outgoingNumberSender string, outgoingDateSender time.Time,
		intermediateNumber *string, intermediateDate *time.Time,
		subject, content string, pagesCount int,
		senderSignatory, senderExecutor, addressee string,
		resolution *string,
	) (*models.IncomingDocument, error)
	Delete(id uuid.UUID) error
	GetCount() (int, error)
}

// OutgoingDocStore — интерфейс для работы с исходящими документами.
type OutgoingDocStore interface {
	GetList(filter models.OutgoingDocumentFilter) (*models.PagedResult[models.OutgoingDocument], error)
	GetByID(id uuid.UUID) (*models.OutgoingDocument, error)
	Create(
		nomenclatureID, documentTypeID, senderOrgID, recipientOrgID, createdBy uuid.UUID,
		outgoingNumber string, outgoingDate time.Time,
		subject, content string, pagesCount int,
		senderSignatory, senderExecutor, addressee string,
	) (*models.OutgoingDocument, error)
	Update(
		id uuid.UUID,
		documentTypeID, senderOrgID, recipientOrgID uuid.UUID,
		outgoingDate time.Time,
		subject, content string, pagesCount int,
		senderSignatory, senderExecutor, addressee string,
	) (*models.OutgoingDocument, error)
	Delete(id uuid.UUID) error
	GetCount() (int, error)
}

// NomenclatureStore — интерфейс для работы с номенклатурой дел.
type NomenclatureStore interface {
	GetAll(year int, direction string) ([]models.Nomenclature, error)
	GetByID(id uuid.UUID) (*models.Nomenclature, error)
	Create(name, index string, year int, direction string) (*models.Nomenclature, error)
	Update(id uuid.UUID, name, index string, year int, direction string, isActive bool) (*models.Nomenclature, error)
	Delete(id uuid.UUID) error
	GetNextNumber(id uuid.UUID) (int, string, error)
	GetActiveByDirection(direction string, year int) ([]models.Nomenclature, error)
}

// ReferenceStore — интерфейс для справочников (типы документов, организации).
type ReferenceStore interface {
	GetAllDocumentTypes() ([]models.DocumentType, error)
	CreateDocumentType(name string) (*models.DocumentType, error)
	UpdateDocumentType(id uuid.UUID, name string) error
	DeleteDocumentType(id uuid.UUID) error
	GetAllOrganizations() ([]models.Organization, error)
	FindOrCreateOrganization(name string) (*models.Organization, error)
	SearchOrganizations(query string) ([]models.Organization, error)
	UpdateOrganization(id uuid.UUID, name string) error
	DeleteOrganization(id uuid.UUID) error
}

// AssignmentStore — интерфейс для работы с поручениями.
type AssignmentStore interface {
	Create(
		documentID uuid.UUID, documentType string,
		executorID uuid.UUID, content string,
		deadline *time.Time, coExecutorIDs []string,
	) (*models.Assignment, error)
	Update(
		id uuid.UUID, executorID uuid.UUID,
		content string, deadline *time.Time,
		status, report string, completedAt *time.Time,
		coExecutorIDs []string,
	) (*models.Assignment, error)
	Delete(id uuid.UUID) error
	GetByID(id uuid.UUID) (*models.Assignment, error)
	GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error)
}

// DepartmentStore — интерфейс для работы с подразделениями.
type DepartmentStore interface {
	GetAll() ([]models.Department, error)
	GetNomenclatureIDs(departmentID uuid.UUID) ([]string, error)
	Create(name string, nomenclatureIDs []string) (*models.Department, error)
	Update(id uuid.UUID, name string, nomenclatureIDs []string) (*models.Department, error)
	Delete(id uuid.UUID) error
}

// SettingsStore — интерфейс для системных настроек.
type SettingsStore interface {
	Get(key string) (*models.SystemSetting, error)
	GetAll() ([]models.SystemSetting, error)
	Update(key, value string) error
}

// AttachmentStore — интерфейс для работы с вложениями.
type AttachmentStore interface {
	Create(a *models.Attachment) error
	Delete(id uuid.UUID) error
	GetByID(id uuid.UUID) (*models.Attachment, error)
	GetByDocumentID(docID uuid.UUID) ([]models.Attachment, error)
	GetContent(id uuid.UUID) ([]byte, error)
}

// LinkStore — интерфейс для связей между документами.
type LinkStore interface {
	Create(ctx context.Context, link *models.DocumentLink) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByDocumentID(ctx context.Context, docID uuid.UUID) ([]models.DocumentLink, error)
	GetGraph(ctx context.Context, rootID uuid.UUID) ([]models.DocumentLink, error)
}

// AcknowledgmentStore — интерфейс для ознакомлений.
type AcknowledgmentStore interface {
	Create(a *models.Acknowledgment) error
	GetByDocumentID(documentID uuid.UUID) ([]models.Acknowledgment, error)
	GetPendingForUser(userID uuid.UUID) ([]models.Acknowledgment, error)
	GetAllActive() ([]models.Acknowledgment, error)
	MarkViewed(ackID, userID uuid.UUID) error
	MarkConfirmed(ackID, userID uuid.UUID) error
	Delete(id uuid.UUID) error
}

// DashboardStore — интерфейс для запросов дашборда.
type DashboardStore interface {
	GetExecutorStatusCounts(userID uuid.UUID) (newCount, inProgressCount int, err error)
	GetExecutorOverdueCount(userID uuid.UUID) (int, error)
	GetExecutorFinishedCounts(userID uuid.UUID) (finished, finishedLate int, err error)
	GetExpiringAssignments(userID *uuid.UUID, days int) ([]models.Assignment, error)
	GetDocCountsByPeriod(startDate, endDate time.Time) (incoming, outgoing int, err error)
	GetOverdueCountByPeriod(startDate, endDate time.Time) (int, error)
	GetFinishedCountsByPeriod(startDate, endDate time.Time) (finished, finishedLate int, err error)
	GetAdminUserCount() (int, error)
	GetAdminDocCounts() (incoming, outgoing int, err error)
	GetDBSize() string
}
