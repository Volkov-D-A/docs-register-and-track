package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// Empty rows are ignored; each document kind decides whether a row is required.
func buildDocumentCorrespondent(references ports.ReferenceStore, number, date, name string, position int) (*models.DocumentCorrespondentRegistration, error) {
	number, date, name = strings.TrimSpace(number), strings.TrimSpace(date), strings.TrimSpace(name)
	if number == "" && date == "" && name == "" {
		return nil, nil
	}
	if number == "" {
		return nil, models.NewBadRequest("укажите регистрационный номер корреспондента")
	}
	if date == "" {
		return nil, models.NewBadRequest("укажите дату регистрации корреспондента")
	}
	if name == "" {
		return nil, models.NewBadRequest("укажите корреспондента")
	}
	registeredAt, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, models.NewBadRequest("неверный формат даты регистрации корреспондента")
	}
	organization, err := references.FindOrCreateOrganization(name)
	if err != nil {
		return nil, fmt.Errorf("ошибка корреспондента: %w", err)
	}
	return &models.DocumentCorrespondentRegistration{
		RegistrationNumber: number,
		RegistrationDate:   registeredAt,
		CorrespondentOrgID: organization.ID,
		CorrespondentName:  organization.Name,
		Position:           position,
	}, nil
}
