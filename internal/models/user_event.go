package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserEventEntityAssignment     = "assignment"
	UserEventEntityAcknowledgment = "acknowledgment"

	UserEventAssignmentCreated       = "assignment_created"
	UserEventAssignmentUpdated       = "assignment_updated"
	UserEventAssignmentCompleted     = "assignment_completed"
	UserEventAssignmentFinished      = "assignment_finished"
	UserEventAssignmentReturned      = "assignment_returned"
	UserEventAcknowledgmentCreated   = "acknowledgment_created"
	UserEventAcknowledgmentConfirmed = "acknowledgment_confirmed"
)

// UserEvent описывает персональное событие пользователя.
type UserEvent struct {
	ID              uuid.UUID  `json:"-"`
	RecipientUserID uuid.UUID  `json:"-"`
	ActorUserID     *uuid.UUID `json:"-"`
	ActorUserName   string     `json:"actorUserName,omitempty"`
	DocumentID      uuid.UUID  `json:"-"`
	DocumentKind    string     `json:"documentKind"`
	DocumentNumber  string     `json:"documentNumber,omitempty"`
	EntityType      string     `json:"entityType"`
	EntityID        uuid.UUID  `json:"-"`
	EventType       string     `json:"eventType"`
	Title           string     `json:"title"`
	Message         string     `json:"message"`
	Metadata        string     `json:"metadata,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	ReadAt          *time.Time `json:"readAt,omitempty"`
}

// CreateUserEventRequest описывает данные для создания события.
type CreateUserEventRequest struct {
	RecipientUserID uuid.UUID
	ActorUserID     *uuid.UUID
	DocumentID      uuid.UUID
	DocumentKind    string
	DocumentNumber  string
	EntityType      string
	EntityID        uuid.UUID
	EventType       string
	Title           string
	Message         string
	Metadata        string
}

// UserEventFilter описывает параметры списка событий.
type UserEventFilter struct {
	UnreadOnly bool `json:"unreadOnly"`
	Page       int  `json:"page"`
	PageSize   int  `json:"pageSize"`
}
