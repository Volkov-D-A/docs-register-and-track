package models

import (
	"time"

	"github.com/google/uuid"
)

// Assignment — поручение по документу
type Assignment struct {
	ID            uuid.UUID `json:"-"`
	IDStr         string    `json:"id"`
	DocumentID    uuid.UUID `json:"-"`
	DocumentIDStr string    `json:"documentId"`
	DocumentType  string    `json:"documentType"` // 'incoming' or 'outgoing'

	ExecutorID    uuid.UUID `json:"-"`
	ExecutorIDStr string    `json:"executorId"`
	ExecutorName  string    `json:"executorName,omitempty"`

	Content     string     `json:"content"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Status      string     `json:"status"` // new, in_progress, completed, cancelled, returned, finished
	Report      string     `json:"report,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	DocumentNumber  string `json:"documentNumber,omitempty"`
	DocumentSubject string `json:"documentSubject,omitempty"`

	// Co-executors
	CoExecutors   []User   `json:"coExecutors,omitempty"`
	CoExecutorIDs []string `json:"coExecutorIds,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (a *Assignment) FillIDStr() {
	a.IDStr = a.ID.String()
	a.DocumentIDStr = a.DocumentID.String()
	a.ExecutorIDStr = a.ExecutorID.String()
}

type AssignmentFilter struct {
	Search       string `json:"search,omitempty"`
	DocumentID   string `json:"documentId,omitempty"`
	ExecutorID   string `json:"executorId,omitempty"`
	Status       string `json:"status,omitempty"`
	DateFrom     string `json:"dateFrom,omitempty"`
	DateTo       string `json:"dateTo,omitempty"`
	OverdueOnly  bool   `json:"overdueOnly"` // Added field
	ShowFinished bool   `json:"showFinished"`
	Page         int    `json:"page"`
	PageSize     int    `json:"pageSize"`
}
