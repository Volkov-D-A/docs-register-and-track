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
	ExecutorName   string
	Content        string
	Deadline       *time.Time
	Status         string
}

type WorkspaceAssignmentCounts struct {
	New                int
	Overdue            int
	DueSoon            int
	AwaitingAcceptance int
}

type WorkspaceAcknowledgment struct {
	ID             uuid.UUID
	DocumentID     uuid.UUID
	DocumentKind   string
	DocumentNumber string
	Content        string
	CreatedAt      time.Time
}
