package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentKindService предоставляет системные метаданные видов документов.
type DocumentKindService struct {
	access *DocumentAccessService
}

// NewDocumentKindService создает новый сервис метаданных видов документов.
func NewDocumentKindService(access *DocumentAccessService) *DocumentKindService {
	return &DocumentKindService{access: access}
}

// GetCurrentAccessSummary возвращает текущую access-модель для навигации и UI.
func (s *DocumentKindService) GetCurrentAccessSummary() (*dto.CurrentAccessSummary, error) {
	if s.access == nil || s.access.auth == nil {
		return nil, models.ErrUnauthorized
	}
	if !s.access.auth.IsAuthenticated() {
		return nil, models.ErrUnauthorized
	}

	user, err := s.access.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, models.ErrUnauthorized
	}

	specs := models.AllDocumentKindSpecs()
	documentKinds := make([]dto.DocumentKindAccessSummary, 0, len(specs))
	registrationKinds := make([]string, 0)
	hasAnyAction := false
	hasAnyAssign := false

	for _, spec := range specs {
		base := dto.MapDocumentKindSpec(spec)
		actions, err := s.access.GetAvailableActions(spec.Code)
		if err != nil {
			return nil, err
		}

		canRegister := containsAction(actions, string(models.DocumentActionCreate))
		canReadFull := containsAction(actions, string(models.DocumentActionRead))
		canAssign := containsAction(actions, string(models.DocumentActionAssign))
		canOpenPage := user.IsDocumentParticipant || canReadFull || canRegister

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
			Code:                 base.Code,
			Name:                 base.Name,
			RegistrationFormCode: base.RegistrationFormCode,
			RegistryGroup:        base.RegistryGroup,
			SupportedActions:     base.SupportedActions,
			AvailableActions:     actions,
			CanOpenPage:          canOpenPage,
			CanRegister:          canRegister,
			CanReadFull:          canReadFull,
		})
	}

	documentDomainAccess := user.IsDocumentParticipant || hasAnyAction
	systemPermissions := []string{}
	for _, permission := range []string{
		models.SystemPermissionAdmin,
		models.SystemPermissionReferences,
		models.SystemPermissionStatsDocuments,
		models.SystemPermissionStatsAssignments,
		models.SystemPermissionStatsSystem,
	} {
		if s.access.auth.HasSystemPermission(permission) {
			systemPermissions = append(systemPermissions, permission)
		}
	}

	return &dto.CurrentAccessSummary{
		IsDocumentParticipant: user.IsDocumentParticipant,
		DocumentDomainAccess:  documentDomainAccess,
		Sections: dto.AccessSections{
			Dashboard:   documentDomainAccess,
			Incoming:    canOpenDocumentKindPage(documentKinds, string(models.DocumentKindIncomingLetter)),
			Outgoing:    canOpenDocumentKindPage(documentKinds, string(models.DocumentKindOutgoingLetter)),
			Appeals:     canOpenDocumentKindPage(documentKinds, string(models.DocumentKindCitizenAppeal)),
			Assignments: user.IsDocumentParticipant || hasAnyAssign,
			References:  containsAction(systemPermissions, models.SystemPermissionReferences),
			Statistics:  containsAnyAction(systemPermissions, models.SystemPermissionStatsDocuments, models.SystemPermissionStatsAssignments, models.SystemPermissionStatsSystem),
			Settings:    containsAction(systemPermissions, models.SystemPermissionAdmin),
		},
		DocumentKinds:     documentKinds,
		RegistrationKinds: registrationKinds,
		SystemPermissions: systemPermissions,
	}, nil
}

func canOpenDocumentKindPage(items []dto.DocumentKindAccessSummary, kindCode string) bool {
	for _, item := range items {
		if item.Code == kindCode {
			return item.CanOpenPage
		}
	}
	return false
}

func containsAction(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func containsAnyAction(items []string, values ...string) bool {
	for _, value := range values {
		if containsAction(items, value) {
			return true
		}
	}
	return false
}
