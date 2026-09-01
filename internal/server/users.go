package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
	"github.com/Volkov-D-A/docs-register-and-track/internal/services"
)

const activeAdministratorInvariantMessage = "at least one active administrator must remain"

type userManagementStore interface {
	GetAll() ([]models.User, error)
	GetActiveUsers() ([]models.User, error)
	GetByID(uuid.UUID) (*models.User, error)
	CreateWithOutbox(models.CreateUserRequest, []models.OutboxEvent) (*models.User, error)
	UpdateWithOutbox(models.UpdateUserRequest, []models.OutboxEvent) (*models.User, error)
	ResetPasswordWithOutbox(uuid.UUID, string, []models.OutboxEvent) error
	UpdateProfileWithOutbox(uuid.UUID, models.UpdateProfileRequest, []models.OutboxEvent) error
}

type executorStore interface {
	GetExecutors() ([]models.User, error)
}

type resetPasswordResponse struct {
	TemporaryPassword string `json:"temporaryPassword"`
}

func (api *managementAPI) listUsers(w http.ResponseWriter, _ *http.Request) {
	users, err := api.userCommands.GetAll()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUsers(users))
}

func (api *managementAPI) listExecutors(w http.ResponseWriter, _ *http.Request) {
	if api.executors == nil {
		writeUserError(w, errors.New("executor store is not configured"))
		return
	}
	users, err := api.executors.GetExecutors()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUsers(users))
}

func (api *managementAPI) createUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if err := validateUserInput(req.Login, req.FullName, req.DepartmentID); err != nil {
		writeUserError(w, err)
		return
	}
	temporaryPassword := ""
	if req.Password == "" {
		password, err := security.GenerateTemporaryPassword()
		if err != nil {
			writeUserError(w, err)
			return
		}
		req.Password = password
		temporaryPassword = password
	}
	req.PasswordChangeRequired = true
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "user:"+uuid.NewString()+":create", "USER_CREATE", fmt.Sprintf("Создан пользователь «%s» (%s)", req.FullName, req.Login))
	if err != nil {
		writeUserError(w, err)
		return
	}
	user, err := api.userCommands.CreateWithOutbox(req, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, err)
		return
	}
	result := dto.MapUser(user)
	if result == nil {
		writeUserError(w, errors.New("user store returned an empty create result"))
		return
	}
	result.TemporaryPassword = temporaryPassword
	writeJSON(w, http.StatusCreated, result)
}

func (api *managementAPI) updateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "validation_error", models.NewBadRequest("некорректный идентификатор пользователя"))
		return
	}
	var req models.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	if err := validateUserInput(req.Login, req.FullName, req.DepartmentID); err != nil {
		writeUserError(w, err)
		return
	}
	req.ID = id.String()
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "user:"+req.ID+":update:"+uuid.NewString(), "USER_UPDATE", fmt.Sprintf("Обновлен пользователь «%s»", req.FullName))
	if err != nil {
		writeUserError(w, err)
		return
	}
	user, err := api.userCommands.UpdateWithOutbox(req, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, activeAdministratorConflict(err))
		return
	}
	result := dto.MapUser(user)
	if result == nil {
		writeUserError(w, errors.New("user store returned an empty update result"))
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func validateUserInput(login, fullName, departmentID string) error {
	if strings.TrimSpace(login) == "" || strings.TrimSpace(fullName) == "" {
		return models.NewBadRequest("логин и ФИО обязательны")
	}
	if departmentID != "" {
		if _, err := uuid.Parse(departmentID); err != nil {
			return models.NewBadRequest("некорректный идентификатор подразделения")
		}
	}
	return nil
}

func (api *managementAPI) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "validation_error", models.NewBadRequest("некорректный идентификатор пользователя"))
		return
	}
	user, err := api.userCommands.GetByID(id)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if user == nil {
		writeUserError(w, models.NewNotFound("пользователь не найден"))
		return
	}
	temporaryPassword, err := security.GenerateTemporaryPassword()
	if err != nil {
		writeUserError(w, err)
		return
	}
	targetName := user.FullName
	if targetName == "" {
		targetName = user.Login
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "user:"+id.String()+":password-reset:"+uuid.NewString(), "USER_PASSWORD_RESET", fmt.Sprintf("Сброшен пароль пользователя «%s»", targetName))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.userCommands.ResetPasswordWithOutbox(id, temporaryPassword, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resetPasswordResponse{TemporaryPassword: temporaryPassword})
}

func userAuditEffect(actor *models.User, key, action, details string) (models.OutboxEvent, error) {
	if actor == nil {
		return models.OutboxEvent{}, models.ErrUnauthorized
	}
	return services.NewAdminAuditOutboxEvent(key, models.CreateAdminAuditLogRequest{
		UserID: actor.ID, UserName: actor.FullName, Action: action, Details: details,
	})
}

func activeAdministratorConflict(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "P0001" && pqErr.Message == activeAdministratorInvariantMessage {
		return models.NewConflict("нельзя деактивировать или лишить права последнего активного администратора")
	}
	return err
}

func writeUserError(w http.ResponseWriter, err error) {
	if appErr, ok := models.AsAppError(err); ok {
		writeAPIError(w, appErr.StatusCode(), strings.ToLower(appErr.SafeKind()), errors.New(appErr.SafeMessage()))
		return
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		writeAPIError(w, http.StatusConflict, "conflict", errors.New("пользователь с таким логином уже существует"))
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "user_operation_failed", errors.New("не удалось выполнить операцию с пользователем"))
}
