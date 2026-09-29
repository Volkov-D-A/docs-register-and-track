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
	Type            string
	DocumentContent string
	ID              uuid.UUID
	DocumentID      uuid.UUID
	DocumentKind    string
	DocumentNumber  string
	DocumentDate    time.Time
	Content         string
	Deadline        *time.Time
	Status          string
	CreatedAt       time.Time
}

type WorkspaceAssignmentCounts struct {
	New                int
	InProgress         int
	Returned           int
	Overdue            int
	DueSoon            int
	AwaitingAcceptance int
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
