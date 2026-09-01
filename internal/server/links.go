package server

import (
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type linkAPI interface {
	LinkDocuments(string, string, string) (*dto.DocumentLink, error)
	UnlinkDocument(string) error
	GetDocumentLinks(string) ([]dto.DocumentLink, error)
	GetDocumentFlow(string) (*models.GraphData, error)
}

type journalAPI interface {
	GetByDocumentID(string) ([]dto.JournalEntry, error)
}

type linkDocumentsRequest struct {
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	LinkType string `json:"linkType"`
}

func (api *managementAPI) linkService(r *http.Request) linkAPI {
	return api.links(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) journalService(r *http.Request) journalAPI {
	return api.journal(authenticatedFromContext(r.Context()).User)
}

func (api *managementAPI) createDocumentLink(w http.ResponseWriter, r *http.Request) {
	var req linkDocumentsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.linkService(r).LinkDocuments(req.SourceID, req.TargetID, req.LinkType)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) deleteDocumentLink(w http.ResponseWriter, r *http.Request) {
	if err := api.linkService(r).UnlinkDocument(r.PathValue("id")); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) listDocumentLinks(w http.ResponseWriter, r *http.Request) {
	result, err := api.linkService(r).GetDocumentLinks(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getDocumentLinkGraph(w http.ResponseWriter, r *http.Request) {
	result, err := api.linkService(r).GetDocumentFlow(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (api *managementAPI) getDocumentJournal(w http.ResponseWriter, r *http.Request) {
	result, err := api.journalService(r).GetByDocumentID(r.PathValue("id"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
