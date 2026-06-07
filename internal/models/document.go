package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type DocumentKind string

const (
	DocumentKindIncomingLetter      DocumentKind = "incoming_letter"
	DocumentKindOutgoingLetter      DocumentKind = "outgoing_letter"
	DocumentKindCitizenAppeal       DocumentKind = "citizen_appeal"
	DocumentKindAdministrativeOrder DocumentKind = "administrative_order"
)

const (
	DocumentTypeLetter              = "Письмо"
	DocumentTypeContract            = "Договор"
	DocumentTypeAct                 = "Акт"
	DocumentTypeInvoice             = "Счёт"
	DocumentTypeRequest             = "Запрос"
	DocumentTypeReply               = "Ответ"
	DocumentTypeNotification        = "Уведомление"
	DocumentTypeCitizenAppeal       = "Обращение"
	DocumentTypeAdministrativeOrder = "Приказ"
)

var documentTypeSet = map[string]struct{}{
	DocumentTypeLetter:              {},
	DocumentTypeContract:            {},
	DocumentTypeAct:                 {},
	DocumentTypeInvoice:             {},
	DocumentTypeRequest:             {},
	DocumentTypeReply:               {},
	DocumentTypeNotification:        {},
	DocumentTypeCitizenAppeal:       {},
	DocumentTypeAdministrativeOrder: {},
}

func AllowedDocumentTypes() []string {
	return []string{
		DocumentTypeLetter,
		DocumentTypeContract,
		DocumentTypeAct,
		DocumentTypeInvoice,
		DocumentTypeRequest,
		DocumentTypeReply,
		DocumentTypeNotification,
		DocumentTypeCitizenAppeal,
		DocumentTypeAdministrativeOrder,
	}
}

func NormalizeDocumentType(value string) string {
	return strings.TrimSpace(value)
}

func IsAllowedDocumentType(value string) bool {
	_, ok := documentTypeSet[NormalizeDocumentType(value)]
	return ok
}

func (k DocumentKind) IsIncoming() bool {
	return k == DocumentKindIncomingLetter
}

func (k DocumentKind) IsOutgoing() bool {
	return k == DocumentKindOutgoingLetter
}

func (k DocumentKind) IsCitizenAppeal() bool {
	return k == DocumentKindCitizenAppeal
}

func (k DocumentKind) IsAdministrativeOrder() bool {
	return k == DocumentKindAdministrativeOrder
}

// Document — общая корневая сущность документа.
type Document struct {
	ID                 uuid.UUID    `json:"-"`
	Kind               DocumentKind `json:"kind"`
	NomenclatureID     uuid.UUID    `json:"-"`
	RegistrationNumber string       `json:"registrationNumber"`
	RegistrationDate   time.Time    `json:"registrationDate"`
	DocumentTypeID     string       `json:"-"`
	Content            string       `json:"content"`
	PagesCount         int          `json:"pagesCount"`
	CreatedBy          uuid.UUID    `json:"-"`
	CreatedAt          time.Time    `json:"createdAt"`
	UpdatedAt          time.Time    `json:"updatedAt"`
}

// IncomingDocument — входящий документ
type IncomingDocument struct {
	ID               uuid.UUID `json:"-"`
	NomenclatureID   uuid.UUID `json:"-"`
	NomenclatureName string    `json:"nomenclatureName,omitempty"`

	// Номера и даты
	IncomingNumber string    `json:"incomingNumber"`
	IncomingDate   time.Time `json:"incomingDate"`

	// О документе
	DocumentTypeID   string `json:"-"`
	DocumentTypeName string `json:"documentTypeName,omitempty"`
	Content          string `json:"content"`
	PagesCount       int    `json:"pagesCount"`

	// Корреспондентские регистрации
	Correspondents []DocumentCorrespondentRegistration `json:"correspondents,omitempty"`

	// Подписант
	SenderSignatory string `json:"senderSignatory"`

	// Резолюция
	Resolution          *string `json:"resolution,omitempty"`
	ResolutionAuthor    *string `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors *string `json:"resolutionExecutors,omitempty"`

	// Метаданные
	CreatedBy     uuid.UUID `json:"-"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// Связанные данные
	AttachmentsCount int `json:"attachmentsCount,omitempty"`
	AssignmentsCount int `json:"assignmentsCount,omitempty"`
}

// DocumentCorrespondentRegistration — регистрационные реквизиты корреспондента документа.
type DocumentCorrespondentRegistration struct {
	ID                 uuid.UUID `json:"-"`
	DocumentID         uuid.UUID `json:"-"`
	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	CorrespondentOrgID uuid.UUID `json:"-"`
	CorrespondentName  string    `json:"correspondentName,omitempty"`
	Position           int       `json:"position"`
}

// DocumentResolution — резолюция документа.
type DocumentResolution struct {
	ID                  uuid.UUID `json:"-"`
	DocumentID          uuid.UUID `json:"-"`
	Resolution          *string   `json:"resolution,omitempty"`
	ResolutionAuthor    *string   `json:"resolutionAuthor,omitempty"`
	ResolutionExecutors *string   `json:"resolutionExecutors,omitempty"`
	Position            int       `json:"position"`
}

// CitizenAppealDocument — обращения граждан.
type CitizenAppealDocument struct {
	ID               uuid.UUID `json:"-"`
	NomenclatureID   uuid.UUID `json:"-"`
	NomenclatureName string    `json:"nomenclatureName,omitempty"`

	RegistrationNumber string    `json:"registrationNumber"`
	RegistrationDate   time.Time `json:"registrationDate"`
	AppealDate         time.Time `json:"appealDate"`

	DocumentTypeID   string `json:"-"`
	DocumentTypeName string `json:"documentTypeName,omitempty"`
	Content          string `json:"content"`
	PagesCount       int    `json:"pagesCount"`

	ApplicantFullName    string `json:"applicantFullName"`
	RegistrationAddress  string `json:"registrationAddress"`
	AppealType           string `json:"appealType"`
	ApplicantCategory    string `json:"applicantCategory"`
	AppealPagesCount     int    `json:"appealPagesCount"`
	AttachmentPagesCount int    `json:"attachmentPagesCount"`
	HasEnvelope          bool   `json:"hasEnvelope"`
	ReceivedFromPOS      bool   `json:"receivedFromPos"`

	Correspondents []DocumentCorrespondentRegistration `json:"correspondents,omitempty"`
	Resolutions    []DocumentResolution                `json:"resolutions,omitempty"`

	CreatedBy     uuid.UUID `json:"-"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	AttachmentsCount int `json:"attachmentsCount,omitempty"`
	AssignmentsCount int `json:"assignmentsCount,omitempty"`
}

// OutgoingDocument — исходящий документ
type OutgoingDocument struct {
	ID               uuid.UUID `json:"-"`
	NomenclatureID   uuid.UUID `json:"-"`
	NomenclatureName string    `json:"nomenclatureName,omitempty"`

	// Номера и даты
	OutgoingNumber string    `json:"outgoingNumber"`
	OutgoingDate   time.Time `json:"outgoingDate"`

	// О документе
	DocumentTypeID   string `json:"-"`
	DocumentTypeName string `json:"documentTypeName,omitempty"`
	Content          string `json:"content"`
	PagesCount       int    `json:"pagesCount"`

	// Отправитель
	SenderSignatory string `json:"senderSignatory"`
	SenderExecutor  string `json:"senderExecutor"`

	// Получатель
	RecipientOrgID   uuid.UUID `json:"-"`
	RecipientOrgName string    `json:"recipientOrgName,omitempty"`
	Addressee        string    `json:"addressee"`

	// Метаданные
	CreatedBy     uuid.UUID `json:"-"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// Связанные данные
	AttachmentsCount int `json:"attachmentsCount,omitempty"`
}

// AdministrativeOrderDocument — приказ.
type AdministrativeOrderDocument struct {
	ID               uuid.UUID `json:"-"`
	NomenclatureID   uuid.UUID `json:"-"`
	NomenclatureName string    `json:"nomenclatureName,omitempty"`

	OrderNumber string    `json:"orderNumber"`
	OrderDate   time.Time `json:"orderDate"`
	Title       string    `json:"title"`

	ExecutionController string     `json:"executionController"`
	ExecutionDeadline   *time.Time `json:"executionDeadline,omitempty"`
	IsActive            bool       `json:"isActive"`
	CancelledAt         *time.Time `json:"cancelledAt,omitempty"`

	AcknowledgmentPeople []AdministrativeOrderAcknowledgmentPerson `json:"acknowledgmentPeople,omitempty"`

	CreatedBy     uuid.UUID `json:"-"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	AttachmentsCount int `json:"attachmentsCount,omitempty"`
	AssignmentsCount int `json:"assignmentsCount,omitempty"`
}

// AdministrativeOrderAcknowledgmentPerson описывает строку листа ознакомления приказа.
type AdministrativeOrderAcknowledgmentPerson struct {
	ID                 uuid.UUID  `json:"-"`
	DocumentID         uuid.UUID  `json:"-"`
	FullName           string     `json:"fullName"`
	AcknowledgedAt     *time.Time `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy     *uuid.UUID `json:"-"`
	AcknowledgedByName string     `json:"acknowledgedByName,omitempty"`
	Position           int        `json:"position"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// DocumentLink — связь между документами
type DocumentLink struct {
	ID         uuid.UUID    `json:"-"`
	SourceKind DocumentKind `json:"sourceKind"`
	SourceID   uuid.UUID    `json:"-"`
	TargetKind DocumentKind `json:"targetKind"`
	TargetID   uuid.UUID    `json:"-"`
	LinkType   string       `json:"linkType"`
	CreatedBy  uuid.UUID    `json:"-"`
	CreatedAt  time.Time    `json:"createdAt"`

	SourceNumber  string `json:"sourceNumber,omitempty"`
	TargetNumber  string `json:"targetNumber,omitempty"`
	TargetSubject string `json:"targetSubject,omitempty"`
}

// DocumentFilter — фильтры для журналов
type DocumentFilter struct {
	NomenclatureID            string   `json:"nomenclatureId,omitempty"`
	NomenclatureIDs           []string `json:"nomenclatureIds,omitempty"`
	AllowedNomenclatureIDs    []string `json:"-"`
	AccessibleByUserID        string   `json:"-"`
	AccessibleByUserIDs       []string `json:"-"`
	KindCode                  string   `json:"kindCode,omitempty"`
	DocumentTypeID            string   `json:"documentTypeId,omitempty"`
	OrgID                     string   `json:"orgId,omitempty"`
	DateFrom                  string   `json:"dateFrom,omitempty"`
	DateTo                    string   `json:"dateTo,omitempty"`
	Search                    string   `json:"search,omitempty"`
	IncomingNumber            string   `json:"incomingNumber,omitempty"`
	RegistrationNumber        string   `json:"registrationNumber,omitempty"`
	OutgoingNumber            string   `json:"outgoingNumber,omitempty"`
	RecipientName             string   `json:"recipientName,omitempty"`
	SenderName                string   `json:"senderName,omitempty"`
	ApplicantName             string   `json:"applicantName,omitempty"`
	AppealType                string   `json:"appealType,omitempty"`
	AppealDateFrom            string   `json:"appealDateFrom,omitempty"`
	AppealDateTo              string   `json:"appealDateTo,omitempty"`
	OutgoingDateFrom          string   `json:"outgoingDateFrom,omitempty"`
	OutgoingDateTo            string   `json:"outgoingDateTo,omitempty"`
	Resolution                string   `json:"resolution,omitempty"`
	NoResolution              bool     `json:"noResolution,omitempty"`
	OrderNumber               string   `json:"orderNumber,omitempty"`
	ExecutionController       string   `json:"executionController,omitempty"`
	OnlyPendingAcknowledgment bool     `json:"onlyPendingAcknowledgment,omitempty"`
	OrderActiveStatus         string   `json:"orderActiveStatus,omitempty"`
	Page                      int      `json:"page"`
	PageSize                  int      `json:"pageSize"`
}

// OutgoingDocumentFilter описывает параметры фильтрации исходящих документов.
type OutgoingDocumentFilter struct {
	NomenclatureIDs        []string `json:"nomenclatureIds,omitempty"`
	AllowedNomenclatureIDs []string `json:"-"`
	AccessibleByUserID     string   `json:"-"`
	AccessibleByUserIDs    []string `json:"-"`
	KindCode               string   `json:"kindCode,omitempty"`
	DocumentTypeID         string   `json:"documentTypeId,omitempty"`
	OrgID                  string   `json:"orgId,omitempty"` // Организация-получатель
	DateFrom               string   `json:"dateFrom,omitempty"`
	DateTo                 string   `json:"dateTo,omitempty"`
	Search                 string   `json:"search,omitempty"`
	OutgoingNumber         string   `json:"outgoingNumber,omitempty"`
	RecipientName          string   `json:"recipientName,omitempty"`
	Page                   int      `json:"page"`
	PageSize               int      `json:"pageSize"`
}

// PagedResult — постраничный результат
type PagedResult[T any] struct {
	Items      []T `json:"items"`
	TotalCount int `json:"totalCount"`
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
}

// GraphNode — узел графа визуализации связей
type GraphNode struct {
	ID        string `json:"id"`
	Label     string `json:"label"` // Номер документа
	KindCode  string `json:"kindCode"`
	Subject   string `json:"subject"`
	Date      string `json:"date"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	IsActive  *bool  `json:"isActive,omitempty"`
}

// GraphEdge — ребро графа визуализации связей
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"` // тип связи
}

// GraphData — данные графа (узлы и рёбра) для фронтенда
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// CreateIncomingDocRequest — запрос на создание входящего документа (уровень репозитория).
type CreateIncomingDocRequest struct {
	NomenclatureID      uuid.UUID
	IdempotencyKey      uuid.UUID
	DocumentTypeID      string
	CreatedBy           uuid.UUID
	IncomingNumber      string
	IncomingDate        time.Time
	Correspondents      []DocumentCorrespondentRegistration
	Content             string
	PagesCount          int
	SenderSignatory     string
	Resolution          *string
	ResolutionAuthor    *string
	ResolutionExecutors *string
}

// UpdateIncomingDocRequest — запрос на обновление входящего документа (уровень репозитория).
type UpdateIncomingDocRequest struct {
	ID                  uuid.UUID
	DocumentTypeID      string
	Correspondents      []DocumentCorrespondentRegistration
	Content             string
	PagesCount          int
	SenderSignatory     string
	Resolution          *string
	ResolutionAuthor    *string
	ResolutionExecutors *string
}

// CreateOutgoingDocRequest — запрос на создание исходящего документа (уровень репозитория).
type CreateOutgoingDocRequest struct {
	NomenclatureID  uuid.UUID
	IdempotencyKey  uuid.UUID
	DocumentTypeID  string
	RecipientOrgID  uuid.UUID
	CreatedBy       uuid.UUID
	OutgoingNumber  string
	OutgoingDate    time.Time
	Content         string
	PagesCount      int
	SenderSignatory string
	SenderExecutor  string
	Addressee       string
}

// UpdateOutgoingDocRequest — запрос на обновление исходящего документа (уровень репозитория).
type UpdateOutgoingDocRequest struct {
	ID              uuid.UUID
	DocumentTypeID  string
	RecipientOrgID  uuid.UUID
	OutgoingDate    time.Time
	Content         string
	PagesCount      int
	SenderSignatory string
	SenderExecutor  string
	Addressee       string
}

// CreateCitizenAppealDocRequest — запрос на создание обращения граждан.
type CreateCitizenAppealDocRequest struct {
	NomenclatureID       uuid.UUID
	IdempotencyKey       uuid.UUID
	CreatedBy            uuid.UUID
	RegistrationNumber   string
	RegistrationDate     time.Time
	AppealDate           time.Time
	Content              string
	ApplicantFullName    string
	RegistrationAddress  string
	AppealType           string
	ApplicantCategory    string
	AppealPagesCount     int
	AttachmentPagesCount int
	HasEnvelope          bool
	ReceivedFromPOS      bool
	Correspondents       []DocumentCorrespondentRegistration
	Resolutions          []DocumentResolution
}

// UpdateCitizenAppealDocRequest — запрос на обновление обращения граждан.
type UpdateCitizenAppealDocRequest struct {
	ID                   uuid.UUID
	RegistrationNumber   string
	RegistrationDate     time.Time
	AppealDate           time.Time
	Content              string
	ApplicantFullName    string
	RegistrationAddress  string
	AppealType           string
	ApplicantCategory    string
	AppealPagesCount     int
	AttachmentPagesCount int
	HasEnvelope          bool
	ReceivedFromPOS      bool
	Correspondents       []DocumentCorrespondentRegistration
	Resolutions          []DocumentResolution
}

// CreateAdministrativeOrderDocRequest — запрос на создание приказа.
type CreateAdministrativeOrderDocRequest struct {
	NomenclatureID          uuid.UUID
	IdempotencyKey          uuid.UUID
	CreatedBy               uuid.UUID
	OrderNumber             string
	OrderDate               time.Time
	Title                   string
	ExecutionController     string
	ExecutionDeadline       *time.Time
	IsActive                bool
	CancelledAt             *time.Time
	AcknowledgmentFullNames []string
}

// UpdateAdministrativeOrderDocRequest — запрос на обновление приказа.
type UpdateAdministrativeOrderDocRequest struct {
	ID                      uuid.UUID
	OrderDate               time.Time
	Title                   string
	ExecutionController     string
	ExecutionDeadline       *time.Time
	IsActive                bool
	CancelledAt             *time.Time
	AcknowledgmentFullNames []string
}
