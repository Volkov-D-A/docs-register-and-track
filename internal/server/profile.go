package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func (api *managementAPI) updateOwnProfile(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if strings.TrimSpace(req.Login) == "" || strings.TrimSpace(req.FullName) == "" {
		writeUserError(w, models.NewBadRequest("логин и ФИО обязательны"))
		return
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "profile:"+auth.User.ID.String()+":update:"+uuid.NewString(), "USER_PROFILE_UPDATE", fmt.Sprintf("Пользователь «%s» обновил собственный профиль", auth.User.FullName))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.userCommands.UpdateProfileWithOutbox(auth.User.ID, req, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) listOwnSubstitutionCandidates(w http.ResponseWriter, _ *http.Request) {
	users, err := api.userCommands.GetActiveUsers()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUsers(users))
}

func (api *managementAPI) getOwnSubstitution(w http.ResponseWriter, r *http.Request) {
	auth := authenticatedFromContext(r.Context())
	item, err := api.substitutions.GetByPrincipalID(auth.User.ID)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUserSubstitution(item))
}

func (api *managementAPI) updateOwnSubstitution(w http.ResponseWriter, r *http.Request) {
	auth := authenticatedFromContext(r.Context())
	var req models.UpdateUserSubstitutionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	req.PrincipalUserID = auth.User.ID.String()
	api.saveUserSubstitution(w, r, auth.User.ID, auth.User, req, "USER_SUBSTITUTION_SELF_UPDATE")
}
