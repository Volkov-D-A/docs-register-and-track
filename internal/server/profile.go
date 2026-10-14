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
	if err := req.NormalizeName(); err != nil {
		writeUserError(w, err)
		return
	}
	if err := validateUserInput(req.Login, ""); err != nil {
		writeUserError(w, err)
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "profile:"+auth.User.ID.String()+":update:"+uuid.NewString(), "USER_PROFILE_UPDATE", fmt.Sprintf("Пользователь «%s» обновил собственный профиль", auth.User.FullName()), auth.User.ID)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.userCommands.UpdateProfileWithOutbox(auth.User.ID, req, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	user, err := api.userCommands.GetByID(auth.User.ID)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if user == nil {
		writeUserError(w, models.NewNotFound("пользователь не найден"))
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUser(user))
}
