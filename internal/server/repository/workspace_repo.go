package repository

import "github.com/Volkov-D-A/docs-register-and-track/internal/server/database"

// WorkspaceRepository reads document previews under server-resolved scopes.
type WorkspaceRepository struct{ db *database.DB }

func NewWorkspaceRepository(db *database.DB) *WorkspaceRepository {
	return &WorkspaceRepository{db: db}
}
