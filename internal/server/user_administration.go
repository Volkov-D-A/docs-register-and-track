package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type userAccessManagementStore interface {
	HasPermission(string, string, string, string) (bool, error)
	HasSystemPermission(string, string) (bool, error)
	GetUserAccessProfile(string) (*models.UserDocumentAccessProfile, error)
	ReplaceUserAccessProfileWithOutbox(string, []models.UserSystemPermissionRule, []models.UserDocumentPermissionRule, []models.OutboxEvent) error
}

type userSubstitutionManagementStore interface {
	GetByPrincipalID(uuid.UUID) (*models.UserSubstitution, error)
	GetActivePrincipalIDs(uuid.UUID) ([]uuid.UUID, error)
	ReplaceForPrincipalWithOutbox(uuid.UUID, *uuid.UUID, *time.Time, *time.Time, bool, *uuid.UUID, []models.OutboxEvent) (*models.UserSubstitution, error)
}

type departmentLookupStore interface {
	GetAll() ([]models.Department, error)
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
		kind := models.NormalizeDocumentKind(permission.KindCode)
		if _, exists := models.GetDocumentKindSpec(kind); !exists {
			writeUserError(w, models.NewBadRequest(fmt.Sprintf("неизвестный вид документа: %s", permission.KindCode)))
			return
		}
		if !kind.SupportsAction(permission.Action) {
			writeUserError(w, models.NewBadRequest(fmt.Sprintf("действие %q не поддерживается для вида документа %q", permission.Action, permission.KindCode)))
			return
		}
	}
	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "user-access:"+id.String()+":update:"+uuid.NewString(), "USER_ACCESS_UPDATE", fmt.Sprintf("Обновлены права пользователя «%s»", target.FullName))
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

func (api *managementAPI) getUserSubstitution(w http.ResponseWriter, r *http.Request) {
	id, _, ok := api.targetUser(w, r)
	if !ok {
		return
	}
	substitution, err := api.substitutions.GetByPrincipalID(id)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUserSubstitution(substitution))
}

func (api *managementAPI) updateUserSubstitution(w http.ResponseWriter, r *http.Request) {
	principalID, principal, ok := api.targetUser(w, r)
	if !ok {
		return
	}
	var req models.UpdateUserSubstitutionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	req.PrincipalUserID = principalID.String()
	api.saveUserSubstitution(w, r, principalID, principal, req, "USER_SUBSTITUTION_UPDATE")
}

func (api *managementAPI) saveUserSubstitution(w http.ResponseWriter, r *http.Request, principalID uuid.UUID, principal *models.User, req models.UpdateUserSubstitutionRequest, auditAction string) {
	startsAt, err := parseOptionalSubstitutionDate(req.StartsAt)
	if err != nil {
		writeUserError(w, err)
		return
	}
	endsAt, err := parseOptionalSubstitutionDate(req.EndsAt)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if startsAt != nil && endsAt != nil && startsAt.After(*endsAt) {
		writeUserError(w, models.NewBadRequest("дата начала замещения не может быть позже даты окончания"))
		return
	}

	var substituteID *uuid.UUID
	if req.SubstituteUserID != "" {
		if !principal.IsDocumentParticipant {
			writeUserError(w, models.NewBadRequest("замещение доступно только участникам документооборота"))
			return
		}
		parsed, parseErr := uuid.Parse(req.SubstituteUserID)
		if parseErr != nil {
			writeUserError(w, models.NewBadRequestWrapped("некорректный идентификатор замещающего", parseErr))
			return
		}
		if parsed == principalID {
			writeUserError(w, models.NewBadRequest("пользователь не может замещать самого себя"))
			return
		}
		substitute, getErr := api.userCommands.GetByID(parsed)
		if getErr != nil {
			writeUserError(w, getErr)
			return
		}
		if substitute == nil {
			writeUserError(w, models.NewNotFound("замещающий пользователь не найден"))
			return
		}
		if !substitute.IsActive {
			writeUserError(w, models.NewBadRequest("замещающий должен быть активным пользователем"))
			return
		}
		if principal.DepartmentID == nil {
			writeUserError(w, models.NewBadRequest("для замещаемого должно быть указано подразделение"))
			return
		}
		if substitute.DepartmentID == nil || *substitute.DepartmentID != *principal.DepartmentID {
			writeUserError(w, models.NewBadRequest("замещающий должен быть из подразделения замещаемого"))
			return
		}
		substituteID = &parsed
	}

	auth := authenticatedFromContext(r.Context())
	effect, err := userAuditEffect(auth.User, "user-substitution:"+principalID.String()+":update:"+uuid.NewString(), auditAction, fmt.Sprintf("Обновлено замещение пользователя «%s»", principal.FullName))
	if err != nil {
		writeUserError(w, err)
		return
	}
	createdBy := auth.User.ID
	if substituteID == nil {
		startsAt, endsAt = nil, nil
		req.IsActive = false
	}
	result, err := api.substitutions.ReplaceForPrincipalWithOutbox(principalID, substituteID, startsAt, endsAt, req.IsActive, &createdBy, []models.OutboxEvent{effect})
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapUserSubstitution(result))
}

func (api *managementAPI) listDepartments(w http.ResponseWriter, _ *http.Request) {
	departments, err := api.departments.GetAll()
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MapDepartments(departments))
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
				RegistrationKinds: []string{}, SystemPermissions: []string{models.SystemPermissionAdmin},
			})
			return
		}
	}

	principalIDs, err := api.substitutions.GetActivePrincipalIDs(user.ID)
	if err != nil {
		writeUserError(w, err)
		return
	}
	hasActiveSubstitution := len(principalIDs) > 0
	departmentID := ""
	if user.DepartmentID != nil {
		departmentID = user.DepartmentID.String()
	}
	userID := user.ID.String()
	specs := models.AllDocumentKindSpecs()
	documentKinds := make([]dto.DocumentKindAccessSummary, 0, len(specs))
	registrationKinds := make([]string, 0)
	hasAnyAction, hasAnyAssign := false, false
	for _, spec := range specs {
		base := dto.MapDocumentKindSpec(spec)
		actions := make([]string, 0, len(spec.SupportedActions))
		for _, action := range spec.SupportedActions {
			allowed, permissionErr := api.userAccess.HasPermission(string(spec.Code), string(action), departmentID, userID)
			if permissionErr != nil {
				writeUserError(w, permissionErr)
				return
			}
			if allowed {
				actions = append(actions, string(action))
			}
		}
		canRegister := contains(actions, string(models.DocumentActionCreate))
		canReadFull := contains(actions, string(models.DocumentActionRead))
		canAssign := contains(actions, string(models.DocumentActionAssign))
		canOpenPage := user.IsDocumentParticipant || hasActiveSubstitution || canReadFull || canRegister
		if len(actions) > 0 {
			hasAnyAction = true
		}
		if canRegister {
			registrationKinds = append(registrationKinds, string(spec.Code))
		}
		if canAssign {
			hasAnyAssign = true
		}
		documentKinds = append(documentKinds, dto.DocumentKindAccessSummary{
			Code: base.Code, Name: base.Name, RegistrationFormCode: base.RegistrationFormCode,
			RegistryGroup: base.RegistryGroup, SupportedActions: base.SupportedActions, AvailableActions: actions,
			CanOpenPage: canOpenPage, CanRegister: canRegister, CanReadFull: canReadFull,
		})
	}

	systemPermissions := make([]string, 0)
	for _, permission := range []string{models.SystemPermissionAdmin, models.SystemPermissionReferences, models.SystemPermissionStatsDocuments, models.SystemPermissionStatsAssignments, models.SystemPermissionStatsSystem} {
		allowed, permissionErr := api.userAccess.HasSystemPermission(permission, userID)
		if permissionErr != nil {
			writeUserError(w, permissionErr)
			return
		}
		if allowed {
			systemPermissions = append(systemPermissions, permission)
		}
	}
	documentDomainAccess := user.IsDocumentParticipant || hasActiveSubstitution || hasAnyAction
	writeJSON(w, http.StatusOK, &dto.CurrentAccessSummary{
		IsDocumentParticipant: user.IsDocumentParticipant, DocumentDomainAccess: documentDomainAccess,
		Sections: dto.AccessSections{
			Dashboard:   documentDomainAccess,
			Incoming:    canOpenDocumentKindPage(documentKinds, string(models.DocumentKindIncomingLetter)),
			Outgoing:    canOpenDocumentKindPage(documentKinds, string(models.DocumentKindOutgoingLetter)),
			Appeals:     canOpenDocumentKindPage(documentKinds, string(models.DocumentKindCitizenAppeal)),
			Orders:      canOpenDocumentKindPage(documentKinds, string(models.DocumentKindAdministrativeOrder)),
			Assignments: user.IsDocumentParticipant || hasActiveSubstitution || hasAnyAssign,
			References:  contains(systemPermissions, models.SystemPermissionReferences),
			Statistics:  containsAny(systemPermissions, models.SystemPermissionStatsDocuments, models.SystemPermissionStatsAssignments, models.SystemPermissionStatsSystem),
			Settings:    contains(systemPermissions, models.SystemPermissionAdmin),
		},
		DocumentKinds: documentKinds, RegistrationKinds: registrationKinds, SystemPermissions: systemPermissions,
	})
}

func canOpenDocumentKindPage(items []dto.DocumentKindAccessSummary, kindCode string) bool {
	for _, item := range items {
		if item.Code == kindCode {
			return item.CanOpenPage
		}
	}
	return false
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

func parseOptionalSubstitutionDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, models.NewBadRequestWrapped("неверный формат даты замещения", err)
	}
	return &parsed, nil
}
