package ports

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type DocumentSearchStore interface {
	SearchDocuments(dto.DocumentSearchRequest, map[models.DocumentKind]models.DocumentAccessScope) (*dto.DocumentSearchResult, error)
}
