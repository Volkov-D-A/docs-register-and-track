package server

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type userAccessManagementStore interface {
	GetAllowedActions(departmentID, userID string) (map[string]map[string]bool, error)
	GetUserAccessProfile(string) (*models.UserDocumentAccessProfile, error)
	ReplaceUserAccessProfileWithOutbox(string, []models.UserSystemPermissionRule, []models.UserDocumentPermissionRule, []models.OutboxEvent) error
}

type departmentManagementStore interface {
	GetAll() ([]models.Department, error)
	CreateWithOutbox(string, []string, []models.OutboxEvent) (*models.Department, error)
	UpdateWithOutbox(uuid.UUID, string, []string, []models.OutboxEvent) (*models.Department, error)
	DeleteWithOutbox(uuid.UUID, []models.OutboxEvent) error
}

type departmentMutationRequest struct {
	Name            string   `json:"name"`
	NomenclatureIDs []string `json:"nomenclatureIds"`
}

func (api *managementAPI) getUserAccessProfile(w http.ResponseWriter, r *http.Request) {
	id, user, ok := api.targetUser(w, r)
	if !ok {
		return
	}
	_ = user
	profile, err := api.userAccess.GetUserAccessProfile(id.String())
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (api *managementAPI) updateUserAccessProfile(w http.ResponseWriter, r *http.Request) {
	id, target, ok := api.targetUser(w, r)
	if !ok {
		return
	}
	var req models.UpdateUserDocumentAccessRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	req.UserID = id.String()
	for _, permission := range req.Permissions {
		kind := models.DocumentKind(permission.KindCode)
		if _, exists := models.GetDocumentKindSpec(kind); !exists {
			writeUserError(w, models.NewBadRequest(fmt.Sprintf("неизвестный вид документа: %s", permission.KindCode)))
			return
		}
		if !kind.SupportsAction(permission.Action) {
			writeUserError(w, models.NewBadRequest(fmt.Sprintf("действие %q не поддерживается для вида документа %q", permission.Action, permission.KindCode)))
			return
		}
	}
	for _, permission := range req.SystemPermissions {
		if permission.Permission != models.SystemPermissionAdmin && permission.Permission != models.SystemPermissionReferences {
			writeUserError(w, models.NewBadRequest("неизвестное системное право"))
			return
		}
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "user-access:"+id.String()+":update:"+uuid.NewString(), "USER_ACCESS_UPDATE", fmt.Sprintf("Обновлены права пользователя «%s»", target.FullName()), id)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.userAccess.ReplaceUserAccessProfileWithOutbox(id.String(), req.SystemPermissions, req.Permissions, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, activeAdministratorConflict(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) listDepartments(w http.ResponseWriter, _ *http.Request) {
	departments, err := api.departments.GetAll()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapDepartments(departments))
}

func (api *managementAPI) createDepartment(w http.ResponseWriter, r *http.Request) {
	var req departmentMutationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "department:"+uuid.NewString()+":create", "DEPT_CREATE", fmt.Sprintf("Создано подразделение «%s»", req.Name))
	if err != nil {
		writeUserError(w, err)
		return
	}
	department, err := api.departments.CreateWithOutbox(req.Name, req.NomenclatureIDs, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.MapDepartment(department))
}

func (api *managementAPI) updateDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeUserError(w, models.NewBadRequestWrapped("неверный ID отдела", err))
		return
	}
	var req departmentMutationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "department:"+id.String()+":update:"+uuid.NewString(), "DEPT_UPDATE", fmt.Sprintf("Обновлено подразделение «%s»", req.Name))
	if err != nil {
		writeUserError(w, err)
		return
	}
	department, err := api.departments.UpdateWithOutbox(id, req.Name, req.NomenclatureIDs, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapDepartment(department))
}

func (api *managementAPI) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeUserError(w, models.NewBadRequestWrapped("неверный ID отдела", err))
		return
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "department:"+id.String()+":delete:"+uuid.NewString(), "DEPT_DELETE", fmt.Sprintf("Удалено подразделение (ID: %s)", id))
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := api.departments.DeleteWithOutbox(id, []models.OutboxEvent{effect}); err != nil {
		writeUserError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *managementAPI) currentAccessSummary(w http.ResponseWriter, r *http.Request) {
	auth := authenticatedFromContext(r.Context())
	user := auth.User
	if api.lifecycle != nil {
		if err := api.lifecycle.CheckReady(); err != nil {
			if !contains(user.SystemPermissions, models.SystemPermissionAdmin) {
				writeAPIError(w, http.StatusServiceUnavailable, "maintenance", err)
				return
			}
			writeJSON(w, http.StatusOK, &dto.CurrentAccessSummary{
				Sections: dto.AccessSections{Settings: true}, DocumentKinds: []dto.DocumentKindAccessSummary{},
				SystemPermissions: []string{models.SystemPermissionAdmin},
			})
			return
		}
	}

	departmentID := ""
	if user.DepartmentID != nil {
		departmentID = user.DepartmentID.String()
	}
	userID := user.ID.String()
	allowedActions, err := api.userAccess.GetAllowedActions(departmentID, userID)
	if err != nil {
		writeUserError(w, err)
		return
	}
	specs := models.AllDocumentKindSpecs()
	documentKinds := make([]dto.DocumentKindAccessSummary, 0, len(specs))
	pageAccess := make(map[string]bool, len(specs))
	for _, spec := range specs {
		actions := make([]string, 0, len(spec.SupportedActions))
		for _, action := range spec.SupportedActions {
			if allowedActions[string(spec.Code)][string(action)] {
				actions = append(actions, string(action))
			}
		}
		pageAccess[string(spec.Code)] = true
		documentKinds = append(documentKinds, dto.DocumentKindAccessSummary{
			Code: string(spec.Code), Name: spec.Name, AvailableActions: actions,
		})
	}

	systemPermissions := make([]string, 0)
	for _, permission := range []string{models.SystemPermissionAdmin, models.SystemPermissionReferences} {
		if contains(user.SystemPermissions, permission) {
			systemPermissions = append(systemPermissions, permission)
		}
	}
	documentDomainAccess := true
	writeJSON(w, http.StatusOK, &dto.CurrentAccessSummary{
		Sections: dto.AccessSections{
			Dashboard:  documentDomainAccess,
			Incoming:   pageAccess[string(models.DocumentKindIncomingLetter)],
			Outgoing:   pageAccess[string(models.DocumentKindOutgoingLetter)],
			Appeals:    pageAccess[string(models.DocumentKindCitizenAppeal)],
			Orders:     pageAccess[string(models.DocumentKindAdministrativeOrder)],
			Controls:   true,
			References: contains(systemPermissions, models.SystemPermissionReferences),
			Statistics: true,
			Reports:    true,
			Settings:   contains(systemPermissions, models.SystemPermissionAdmin),
		},
		DocumentKinds: documentKinds, SystemPermissions: systemPermissions,
	})
}

func containsAny(items []string, values ...string) bool {
	for _, value := range values {
		if contains(items, value) {
			return true
		}
	}
	return false
}

func (api *managementAPI) targetUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, *models.User, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeUserError(w, models.NewBadRequest("некорректный идентификатор пользователя"))
		return uuid.Nil, nil, false
	}
	user, err := api.userCommands.GetByID(id)
	if err != nil {
		writeUserError(w, err)
		return uuid.Nil, nil, false
	}
	if user == nil {
		writeUserError(w, models.NewNotFound("пользователь не найден"))
		return uuid.Nil, nil, false
	}
	return id, user, true
}
