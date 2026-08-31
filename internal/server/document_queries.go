package server

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type documentQueryAPI interface {
	GetByID(string) (*dto.DocumentCard, error)
	GetList(string, models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error)
}

type documentListRequest struct {
	KindCode string                `json:"kindCode"`
	Filter   models.DocumentFilter `json:"filter"`
}

type requestDocumentPrincipal struct {
	user *models.User
}

func (p requestDocumentPrincipal) RequireAuthenticated() error {
	if p.user == nil || !p.user.IsActive {
		return models.ErrUnauthorized
	}
	return nil
}

func (p requestDocumentPrincipal) GetCurrentUser() (*dto.User, error) {
	if err := p.RequireAuthenticated(); err != nil {
		return nil, err
	}
	return dto.MapUser(p.user), nil
}

func (p requestDocumentPrincipal) GetCurrentUserUUID() (uuid.UUID, error) {
	if err := p.RequireAuthenticated(); err != nil {
		return uuid.Nil, err
	}
	return p.user.ID, nil
}

func (api *managementAPI) getDocumentCard(w http.ResponseWriter, r *http.Request) {
	query := api.documentQueries(authenticatedFromContext(r.Context()).User)
	card, err := query.GetByID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, card)
}

func (api *managementAPI) listDocuments(w http.ResponseWriter, r *http.Request) {
	var req documentListRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	// Never accept an access scope from the client, even if the model becomes
	// serializable later.
	req.Filter.AccessScope = nil
	query := api.documentQueries(authenticatedFromContext(r.Context()).User)
	result, err := query.GetList(req.KindCode, req.Filter)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
