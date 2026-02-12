package models

import (
	"time"

	"github.com/google/uuid"
)

// IncomingDocument — входящий документ
type IncomingDocument struct {
	ID                uuid.UUID `json:"-"`
	IDStr             string    `json:"id"`
	NomenclatureID    uuid.UUID `json:"-"`
	NomenclatureIDStr string    `json:"nomenclatureId"`
	NomenclatureName  string    `json:"nomenclatureName,omitempty"`

	// Номера и даты
	IncomingNumber       string     `json:"incomingNumber"`
	IncomingDate         time.Time  `json:"incomingDate"`
	OutgoingNumberSender string     `json:"outgoingNumberSender"`
	OutgoingDateSender   time.Time  `json:"outgoingDateSender"`
	IntermediateNumber   *string    `json:"intermediateNumber,omitempty"`
	IntermediateDate     *time.Time `json:"intermediateDate,omitempty"`

	// О документе
	DocumentTypeID    uuid.UUID `json:"-"`
	DocumentTypeIDStr string    `json:"documentTypeId"`
	DocumentTypeName  string    `json:"documentTypeName,omitempty"`
	Subject           string    `json:"subject"`
	PagesCount        int       `json:"pagesCount"`
	Content           string    `json:"content"`

	// Отправитель
	SenderOrgID     uuid.UUID `json:"-"`
	SenderOrgIDStr  string    `json:"senderOrgId"`
	SenderOrgName   string    `json:"senderOrgName,omitempty"`
	SenderSignatory string    `json:"senderSignatory"`
	SenderExecutor  string    `json:"senderExecutor"`

	// Получатель
	RecipientOrgID    uuid.UUID `json:"-"`
	RecipientOrgIDStr string    `json:"recipientOrgId"`
	RecipientOrgName  string    `json:"recipientOrgName,omitempty"`
	Addressee         string    `json:"addressee"`

	// Резолюция
	Resolution *string `json:"resolution,omitempty"`

	// Метаданные
	CreatedBy     uuid.UUID `json:"-"`
	CreatedByStr  string    `json:"createdBy"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// Связанные данные
	AttachmentsCount int `json:"attachmentsCount,omitempty"`
	AssignmentsCount int `json:"assignmentsCount,omitempty"`
}

func (d *IncomingDocument) FillIDStr() {
	d.IDStr = d.ID.String()
	d.NomenclatureIDStr = d.NomenclatureID.String()
	d.DocumentTypeIDStr = d.DocumentTypeID.String()
	d.SenderOrgIDStr = d.SenderOrgID.String()
	d.RecipientOrgIDStr = d.RecipientOrgID.String()
	d.CreatedByStr = d.CreatedBy.String()
}

// OutgoingDocument — исходящий документ
type OutgoingDocument struct {
	ID                uuid.UUID `json:"-"`
	IDStr             string    `json:"id"`
	NomenclatureID    uuid.UUID `json:"-"`
	NomenclatureIDStr string    `json:"nomenclatureId"`
	NomenclatureName  string    `json:"nomenclatureName,omitempty"`

	// Номера и даты
	OutgoingNumber string    `json:"outgoingNumber"`
	OutgoingDate   time.Time `json:"outgoingDate"`

	// О документе
	DocumentTypeID    uuid.UUID `json:"-"`
	DocumentTypeIDStr string    `json:"documentTypeId"`
	DocumentTypeName  string    `json:"documentTypeName,omitempty"`
	Subject           string    `json:"subject"`
	PagesCount        int       `json:"pagesCount"`
	Content           string    `json:"content"`

	// Отправитель
	SenderOrgID     uuid.UUID `json:"-"`
	SenderOrgIDStr  string    `json:"senderOrgId"`
	SenderOrgName   string    `json:"senderOrgName,omitempty"`
	SenderSignatory string    `json:"senderSignatory"`
	SenderExecutor  string    `json:"senderExecutor"`

	// Получатель
	RecipientOrgID    uuid.UUID `json:"-"`
	RecipientOrgIDStr string    `json:"recipientOrgId"`
	RecipientOrgName  string    `json:"recipientOrgName,omitempty"`
	Addressee         string    `json:"addressee"`

	// Метаданные
	CreatedBy     uuid.UUID `json:"-"`
	CreatedByStr  string    `json:"createdBy"`
	CreatedByName string    `json:"createdByName,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`

	// Связанные данные
	AttachmentsCount int `json:"attachmentsCount,omitempty"`
}

func (d *OutgoingDocument) FillIDStr() {
	d.IDStr = d.ID.String()
	d.NomenclatureIDStr = d.NomenclatureID.String()
	d.DocumentTypeIDStr = d.DocumentTypeID.String()
	d.SenderOrgIDStr = d.SenderOrgID.String()
	d.RecipientOrgIDStr = d.RecipientOrgID.String()
	d.CreatedByStr = d.CreatedBy.String()
}

// DocumentLink — связь между документами
type DocumentLink struct {
	ID           uuid.UUID `json:"-"`
	IDStr        string    `json:"id"`
	SourceType   string    `json:"sourceType"`
	SourceID     uuid.UUID `json:"-"`
	SourceIDStr  string    `json:"sourceId"`
	TargetType   string    `json:"targetType"`
	TargetID     uuid.UUID `json:"-"`
	TargetIDStr  string    `json:"targetId"`
	LinkType     string    `json:"linkType"`
	CreatedBy    uuid.UUID `json:"-"`
	CreatedByStr string    `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`

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

type OutgoingDocumentFilter struct {
	NomenclatureIDs []string `json:"nomenclatureIds,omitempty"`
	DocumentTypeID  string   `json:"documentTypeId,omitempty"`
	OrgID           string   `json:"orgId,omitempty"` // Recipient Org
	DateFrom        string   `json:"dateFrom,omitempty"`
	DateTo          string   `json:"dateTo,omitempty"`
	Search          string   `json:"search,omitempty"`
	OutgoingNumber  string   `json:"outgoingNumber,omitempty"`
	RecipientName   string   `json:"recipientName,omitempty"`
	Page            int      `json:"page"`
	PageSize        int      `json:"pageSize"`
}

// PagedResult — постраничный результат
type PagedResult struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"totalCount"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
}
