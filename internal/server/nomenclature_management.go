package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type nomenclatureManagementStore interface {
	GetAll(int, string) ([]models.Nomenclature, error)
	GetActiveByKind(string, int) ([]models.Nomenclature, error)
	CreateWithOutbox(string, string, int, string, string, string, int, []models.OutboxEvent) (*models.Nomenclature, error)
	UpdateWithOutbox(uuid.UUID, string, string, int, string, string, string, bool, []models.OutboxEvent) (*models.Nomenclature, error)
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

type nomenclatureCreateRequest struct {
	Name          string `json:"name"`
	Index         string `json:"index"`
	Year          int    `json:"year"`
	KindCode      string `json:"kindCode"`
	Separator     string `json:"separator"`
	NumberingMode string `json:"numberingMode"`
	StartNumber   int    `json:"startNumber"`
}

type nomenclatureUpdateRequest struct {
	Name          string `json:"name"`
	Index         string `json:"index"`
	Year          int    `json:"year"`
	KindCode      string `json:"kindCode"`
	Separator     string `json:"separator"`
	NumberingMode string `json:"numberingMode"`
	IsActive      bool   `json:"isActive"`
}

func (api *managementAPI) listNomenclature(w http.ResponseWriter, r *http.Request) {
	year, err := optionalQueryInt(r, "year")
	if err != nil {
		writeUserError(w, models.NewBadRequestWrapped("год должен быть целым числом", err))
		return
	}
	items, err := api.nomenclature.GetAll(year, strings.TrimSpace(r.URL.Query().Get("kindCode")))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapNomenclatures(items))
}

func (api *managementAPI) listActiveNomenclature(w http.ResponseWriter, r *http.Request) {
	items, err := api.nomenclature.GetActiveByKind(strings.TrimSpace(r.URL.Query().Get("kindCode")), time.Now().Year())
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapNomenclatures(items))
}

func (api *managementAPI) createNomenclature(w http.ResponseWriter, r *http.Request) {
	var req nomenclatureCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if req.StartNumber < 1 {
		req.StartNumber = 1
	}
	effect, err := nomenclatureAuditEffect(r, "nomenclature:"+uuid.NewString()+":create", "NOMENCLATURE_CREATE", fmt.Sprintf("Создано дело «%s» (%s), вид: %s, год: %d, стартовый номер: %d", req.Name, req.Index, req.KindCode, req.Year, req.StartNumber))
	if err != nil {
		writeUserError(w, err)
		return
	}
	item, err := api.nomenclature.CreateWithOutbox(req.Name, req.Index, req.Year, req.KindCode, req.Separator, req.NumberingMode, req.StartNumber, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.MapNomenclature(item))
}

func (api *managementAPI) updateNomenclature(w http.ResponseWriter, r *http.Request) {
	id, ok := parseNomenclatureID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var req nomenclatureUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	effect, err := nomenclatureAuditEffect(r, "nomenclature:"+id.String()+":update:"+uuid.NewString(), "NOMENCLATURE_UPDATE", fmt.Sprintf("Обновлено дело «%s» (%s)", req.Name, req.Index))
	if err != nil {
		writeUserError(w, err)
		return
	}
	item, err := api.nomenclature.UpdateWithOutbox(id, req.Name, req.Index, req.Year, req.KindCode, req.Separator, req.NumberingMode, req.IsActive, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapNomenclature(item))
}

func (api *managementAPI) deleteNomenclature(w http.ResponseWriter, r *http.Request) {
	id, ok := parseNomenclatureID(w, r.PathValue("id"))
	if !ok {
		return
	}
	effect, err := nomenclatureAuditEffect(r, "nomenclature:"+id.String()+":delete", "NOMENCLATURE_DELETE", fmt.Sprintf("Удалено дело (ID: %s)", id))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.nomenclature.DeleteWithOutbox(id, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func optionalQueryInt(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func parseNomenclatureID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		writeUserError(w, models.NewBadRequestWrapped("неверный ID номенклатуры", err))
		return uuid.Nil, false
	}
	return id, true
}

func nomenclatureAuditEffect(r *http.Request, key, action, details string) (models.OutboxEvent, error) {
	auth := authenticatedFromContext(r.Context())
	return userAuditEffect(auth.User, key, action, details)
}
