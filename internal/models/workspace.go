package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	WorkspaceModeExecution = "execution"
	WorkspaceModeControl   = "control"
)

// WorkspaceQuery is resolved on the server. Client input never supplies its scope.
type WorkspaceQuery struct {
	Mode       string
	Kind       DocumentKind
	ReadScope  DocumentAccessScope
	SubjectIDs []string
	Limit      int
}

type WorkspaceAssignment struct {
	ID             uuid.UUID
	DocumentID     uuid.UUID
	DocumentKind   string
	DocumentNumber string
	DocumentDate   time.Time
	Content        string
	Deadline       *time.Time
	Status         string
}

type WorkspaceAssignmentCounts struct {
	New                int
	InProgress         int
	Overdue            int
	DueSoon            int
	AwaitingAcceptance int
}

type WorkspaceAcknowledgment struct {
	ID              uuid.UUID
	DocumentID      uuid.UUID
	DocumentKind    string
	DocumentNumber  string
	DocumentDate    time.Time
	DocumentContent string
	Content         string
	CreatedAt       time.Time
}

// WorkspaceDocument is a bounded document preview under server-resolved read scopes.
type WorkspaceDocument struct {
	ID             uuid.UUID
	DocumentKind   string
	DocumentNumber string
	DocumentDate   time.Time
	RegisteredAt   time.Time
	Description    string
	Correspondents []string
}
