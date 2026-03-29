package models

import (
	"time"

	"github.com/google/uuid"
)

// IncomingDocument — входящий документ
type IncomingDocument struct {
	ID               uuid.UUID `json:"-"`
	NomenclatureID   uuid.UUID `json:"-"`
	NomenclatureName string    `json:"nomenclatureName,omitempty"`

	// Номера и даты
	IncomingNumber       string     `json:"incomingNumber"`
	IncomingDate         time.Time  `json:"incomingDate"`
	OutgoingNumberSender string     `json:"outgoingNumberSender"`
	OutgoingDateSender   time.Time  `json:"outgoingDateSender"`
	IntermediateNumber   *string    `json:"intermediateNumber,omitempty"`
	IntermediateDate     *time.Time `json:"intermediateDate,omitempty"`

	// О документе
	DocumentTypeID   uuid.UUID `json:"-"`
	DocumentTypeName string    `json:"documentTypeName,omitempty"`
	Content          string    `json:"content"`
	PagesCount       int       `json:"pagesCount"`

	// Отправитель
	SenderOrgID     uuid.UUID `json:"-"`
	SenderOrgName   string    `json:"senderOrgName,omitempty"`
	SenderSignatory string    `json:"senderSignatory"`

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

// OutgoingDocument — исходящий документ
type OutgoingDocument struct {
	ID               uuid.UUID `json:"-"`
	NomenclatureID   uuid.UUID `json:"-"`
	NomenclatureName string    `json:"nomenclatureName,omitempty"`

	// Номера и даты
	OutgoingNumber string    `json:"outgoingNumber"`
	OutgoingDate   time.Time `json:"outgoingDate"`

	// О документе
	DocumentTypeID   uuid.UUID `json:"-"`
	DocumentTypeName string    `json:"documentTypeName,omitempty"`
	Content          string    `json:"content"`
	PagesCount       int       `json:"pagesCount"`

	// Отправитель
	SenderOrgID     uuid.UUID `json:"-"`
	SenderOrgName   string    `json:"senderOrgName,omitempty"`
	SenderSignatory string    `json:"senderSignatory"`
	SenderExecutor  string    `json:"senderExecutor"`

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

// DocumentLink — связь между документами
type DocumentLink struct {
	ID         uuid.UUID `json:"-"`
	SourceType string    `json:"sourceType"`
	SourceID   uuid.UUID `json:"-"`
	TargetType string    `json:"targetType"`
	TargetID   uuid.UUID `json:"-"`
	LinkType   string    `json:"linkType"`
	CreatedBy  uuid.UUID `json:"-"`
	CreatedAt  time.Time `json:"createdAt"`

	SourceNumber  string `json:"sourceNumber,omitempty"`
	TargetNumber  string `json:"targetNumber,omitempty"`
	TargetSubject string `json:"targetSubject,omitempty"`
}

// DocumentFilter — фильтры для журналов
type DocumentFilter struct {
	NomenclatureID   string   `json:"nomenclatureId,omitempty"`
	NomenclatureIDs  []string `json:"nomenclatureIds,omitempty"`
	DocumentTypeID   string   `json:"documentTypeId,omitempty"`
	OrgID            string   `json:"orgId,omitempty"`
	DateFrom         string   `json:"dateFrom,omitempty"`
	DateTo           string   `json:"dateTo,omitempty"`
	Search           string   `json:"search,omitempty"`
	IncomingNumber   string   `json:"incomingNumber,omitempty"`
	OutgoingNumber   string   `json:"outgoingNumber,omitempty"`
	SenderName       string   `json:"senderName,omitempty"`
	OutgoingDateFrom string   `json:"outgoingDateFrom,omitempty"`
	OutgoingDateTo   string   `json:"outgoingDateTo,omitempty"`
	Resolution       string   `json:"resolution,omitempty"`
	NoResolution     bool     `json:"noResolution,omitempty"`
	Page             int      `json:"page"`
	PageSize         int      `json:"pageSize"`
}

// OutgoingDocumentFilter описывает параметры фильтрации исходящих документов.
type OutgoingDocumentFilter struct {
	NomenclatureIDs []string `json:"nomenclatureIds,omitempty"`
	DocumentTypeID  string   `json:"documentTypeId,omitempty"`
	OrgID           string   `json:"orgId,omitempty"` // Организация-получатель
	DateFrom        string   `json:"dateFrom,omitempty"`
	DateTo          string   `json:"dateTo,omitempty"`
	Search          string   `json:"search,omitempty"`
	OutgoingNumber  string   `json:"outgoingNumber,omitempty"`
	RecipientName   string   `json:"recipientName,omitempty"`
	Page            int      `json:"page"`
	PageSize        int      `json:"pageSize"`
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
	Type      string `json:"type"`  // входящий/исходящий
	Subject   string `json:"subject"`
	Date      string `json:"date"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
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
	NomenclatureID       uuid.UUID
	DocumentTypeID       uuid.UUID
	SenderOrgID          uuid.UUID
	CreatedBy            uuid.UUID
	IncomingNumber       string
	IncomingDate         time.Time
	OutgoingNumberSender string
	OutgoingDateSender   time.Time
	IntermediateNumber   *string
	IntermediateDate     *time.Time
	Content              string
	PagesCount           int
	SenderSignatory      string
	Resolution           *string
	ResolutionAuthor     *string
	ResolutionExecutors  *string
}

// UpdateIncomingDocRequest — запрос на обновление входящего документа (уровень репозитория).
type UpdateIncomingDocRequest struct {
	ID                   uuid.UUID
	DocumentTypeID       uuid.UUID
	SenderOrgID          uuid.UUID
	OutgoingNumberSender string
	OutgoingDateSender   time.Time
	IntermediateNumber   *string
	IntermediateDate     *time.Time
	Content              string
	PagesCount           int
	SenderSignatory      string
	Resolution           *string
	ResolutionAuthor     *string
	ResolutionExecutors  *string
}

// CreateOutgoingDocRequest — запрос на создание исходящего документа (уровень репозитория).
type CreateOutgoingDocRequest struct {
	NomenclatureID  uuid.UUID
	DocumentTypeID  uuid.UUID
	SenderOrgID     uuid.UUID
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
	DocumentTypeID  uuid.UUID
	SenderOrgID     uuid.UUID
	RecipientOrgID  uuid.UUID
	OutgoingDate    time.Time
	Content         string
	PagesCount      int
	SenderSignatory string
	SenderExecutor  string
	Addressee       string
}
