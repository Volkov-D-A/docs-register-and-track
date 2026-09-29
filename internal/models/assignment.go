package models

import (
	"time"

	"github.com/google/uuid"
)

// Assignment — поручение по документу
type Assignment struct {
	Type         string                `json:"type"`
	CreatorID    uuid.UUID             `json:"-"`
	CreatorName  string                `json:"creatorName,omitempty"`
	Users        []AssignmentRecipient `json:"users,omitempty"`
	ID           uuid.UUID             `json:"-"`
	DocumentID   uuid.UUID             `json:"-"`
	DocumentKind string                `json:"documentKind"`

	ExecutorID   uuid.UUID `json:"-"`
	ExecutorName string    `json:"executorName,omitempty"`

	Content     string     `json:"content"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Status      string     `json:"status"` // new, in_progress, completed, cancelled, returned, finished
	Report      string     `json:"report,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	SeriesID        *uuid.UUID `json:"-"`
	IterationNumber int        `json:"iterationNumber,omitempty"`
	PlannedDeadline *time.Time `json:"plannedDeadline,omitempty"`
	IsSeriesCurrent bool       `json:"-"`

	DocumentNumber  string `json:"documentNumber,omitempty"`
	DocumentSubject string `json:"documentSubject,omitempty"`

	// Соисполнители
	CoExecutors   []User   `json:"coExecutors,omitempty"`
	CoExecutorIDs []string `json:"coExecutorIds,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AssignmentSeries describes the template and calendar rule for future
// iterations of a recurring assignment. Only users allowed to assign work for
// the document may read or mutate this model through the service layer.
type AssignmentSeries struct {
	ID                  uuid.UUID  `json:"-"`
	DocumentID          uuid.UUID  `json:"-"`
	DocumentKind        string     `json:"documentKind"`
	DocumentNumber      string     `json:"documentNumber,omitempty"`
	ExecutorID          uuid.UUID  `json:"-"`
	ExecutorName        string     `json:"executorName,omitempty"`
	Content             string     `json:"content"`
	IntervalUnit        string     `json:"intervalUnit"`
	IntervalValue       int        `json:"intervalValue"`
	DayRule             string     `json:"dayRule"` // same_day, fixed, last_day
	DayOfMonth          int        `json:"dayOfMonth,omitempty"`
	CurrentAssignmentID *uuid.UUID `json:"-"`
	CurrentIteration    int        `json:"currentIteration"`
	Active              bool       `json:"active"`
	CreatedBy           uuid.UUID  `json:"-"`
	CancelledBy         *uuid.UUID `json:"-"`
	CancelledAt         *time.Time `json:"cancelledAt,omitempty"`
	CoExecutors         []User     `json:"coExecutors,omitempty"`
	CoExecutorIDs       []string   `json:"coExecutorIds,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// AssignmentSeriesRequest contains the template and calendar rule supplied by
// the Wails client when a recurring series is created or updated.
type AssignmentSeriesRequest struct {
	DocumentID    string   `json:"documentId"`
	ExecutorID    string   `json:"executorId"`
	Content       string   `json:"content"`
	FirstDeadline string   `json:"firstDeadline"`
	IntervalUnit  string   `json:"intervalUnit"`
	IntervalValue int      `json:"intervalValue"`
	DayRule       string   `json:"dayRule"`
	DayOfMonth    int      `json:"dayOfMonth"`
	CoExecutorIDs []string `json:"coExecutorIds"`
}

// AssignmentFilter описывает параметры фильтрации поручений.
type AssignmentFilter struct {
	Types        []string `json:"types,omitempty"`
	Search       string   `json:"search,omitempty"`
	DocumentID   string   `json:"documentId,omitempty"`
	ExecutorID   string   `json:"executorId,omitempty"`
	Statuses     []string `json:"statuses,omitempty"`
	DateFrom     string   `json:"dateFrom,omitempty"`
	DateTo       string   `json:"dateTo,omitempty"`
	OverdueOnly  bool     `json:"overdueOnly"` // Фильтр просроченных
	ShowFinished bool     `json:"showFinished"`
	Mode         string   `json:"mode,omitempty"`
	Page         int      `json:"page"`
	PageSize     int      `json:"pageSize"`

	// Внутренний скоуп доступа: не принимается с клиента.
	AllowedDocumentKinds []string                             `json:"-"`
	AccessibleByUserID   string                               `json:"-"`
	AccessibleByUserIDs  []string                             `json:"-"`
	ControlScopes        map[DocumentKind]DocumentAccessScope `json:"-"`
}

const (
	AssignmentTypeExecution      = "execution"
	AssignmentTypeAcknowledgment = "acknowledgment"
)

// AssignmentRecipient records each participant's independent confirmation.
type AssignmentRecipient struct {
	ID           uuid.UUID  `json:"-"`
	AssignmentID uuid.UUID  `json:"-"`
	UserID       uuid.UUID  `json:"-"`
	UserName     string     `json:"userName,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmedAt,omitempty"`
	ConfirmedBy  *uuid.UUID `json:"-"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type AssignmentRequest struct {
	Type          string   `json:"type"`
	DocumentID    string   `json:"documentId"`
	ExecutorID    string   `json:"executorId,omitempty"`
	Content       string   `json:"content"`
	Deadline      string   `json:"deadline,omitempty"`
	CoExecutorIDs []string `json:"coExecutorIds,omitempty"`
	UserIDs       []string `json:"userIds,omitempty"`
}
