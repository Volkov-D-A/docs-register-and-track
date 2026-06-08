package services

import (
	"strings"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AdminNumberOverrideRequest struct {
	Mode   string `json:"mode"`
	Number int    `json:"number"`
	Suffix string `json:"suffix"`
}

type AdminDraftCreateRequest struct {
	NomenclatureID      string                      `json:"nomenclatureId"`
	RegistrationDate    string                      `json:"registrationDate"`
	AdminNumberOverride *AdminNumberOverrideRequest `json:"adminNumberOverride"`
}

const adminDraftPlaceholder = "Черновик. Требуется заполнение."

func buildAdminNumberOverride(req *AdminNumberOverrideRequest) (*models.AdminNumberOverride, error) {
	if req == nil {
		return nil, nil
	}
	mode := strings.TrimSpace(req.Mode)
	if mode != models.AdminNumberModeInsertShift && mode != models.AdminNumberModeLiteral {
		return nil, models.NewBadRequest("неверный режим административной нумерации")
	}
	if req.Number < 1 {
		return nil, models.NewBadRequest("укажите номер документа больше 0")
	}
	suffix := strings.TrimSpace(req.Suffix)
	if mode == models.AdminNumberModeInsertShift && suffix != "" {
		return nil, models.NewBadRequest("для вставки со сдвигом укажите номер без литеры")
	}
	return &models.AdminNumberOverride{
		Mode:   mode,
		Number: req.Number,
		Suffix: suffix,
	}, nil
}
