package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentKindService предоставляет системные метаданные видов документов.
type DocumentKindService struct {
	access *DocumentAccessService
}

// NewDocumentKindService создает новый сервис метаданных видов документов.
func NewDocumentKindService(access *DocumentAccessService) *DocumentKindService {
	return &DocumentKindService{access: access}
}

// GetAll возвращает все системные виды документов.
func (s *DocumentKindService) GetAll() ([]dto.DocumentKind, error) {
	specs := models.AllDocumentKindSpecs()
	result := make([]dto.DocumentKind, 0, len(specs))

	for _, spec := range specs {
		result = append(result, *dto.MapDocumentKindSpec(spec))
	}

	return result, nil
}

// GetAvailableForRegistration возвращает системные виды документов, доступные пользователю для регистрации.
func (s *DocumentKindService) GetAvailableForRegistration() ([]dto.DocumentKind, error) {
	specs := models.AllDocumentKindSpecs()
	result := make([]dto.DocumentKind, 0, len(specs))

	for _, spec := range specs {
		if err := s.access.RequireCreate(spec.Code); err != nil {
			continue
		}

		result = append(result, *dto.MapDocumentKindSpec(spec))
	}

	return result, nil
}
