package services

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

func validatePageCounts(pagesCount, attachmentPagesCount int) error {
	if pagesCount < 1 {
		return models.NewBadRequest("количество листов должно быть не меньше 1")
	}
	if attachmentPagesCount < 0 {
		return models.NewBadRequest("количество листов приложения не может быть отрицательным")
	}
	return nil
}
