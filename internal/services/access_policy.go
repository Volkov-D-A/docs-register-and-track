package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// requireClerkDocumentRole ограничивает document-domain активной ролью clerk.
func requireClerkDocumentRole(auth *AuthService) error {
	if auth == nil {
		return models.ErrUnauthorized
	}

	return auth.RequireActiveRole("clerk")
}

// requireDocumentDomainReadRole ограничивает document-domain ролями clerk и executor.
func requireDocumentDomainReadRole(auth *AuthService) error {
	if auth == nil {
		return models.ErrUnauthorized
	}

	return auth.RequireAnyActiveRole("clerk", "executor")
}

func hasExecutorNomenclatureAccess(auth *AuthService, depRepo DepartmentStore, nomenclatureID uuid.UUID) (bool, error) {
	if auth == nil {
		return false, models.ErrUnauthorized
	}

	if auth.HasActiveRole("clerk") {
		return true, nil
	}
	if !auth.HasActiveRole("executor") {
		return false, models.ErrForbidden
	}

	user, err := auth.GetCurrentUser()
	if err != nil {
		return false, err
	}
	if user == nil || user.Department == nil || user.Department.ID == "" {
		return false, nil
	}

	departmentID, err := uuid.Parse(user.Department.ID)
	if err != nil {
		return false, nil
	}

	allowedNomenclatures, err := depRepo.GetNomenclatureIDs(departmentID)
	if err != nil {
		return false, err
	}
	nomenclatureIDStr := nomenclatureID.String()
	for _, allowedID := range allowedNomenclatures {
		if allowedID == nomenclatureIDStr {
			return true, nil
		}
	}

	return false, nil
}

// requireExecutorDocumentAccess проверяет доступ исполнителя к документу по номенклатуре либо по связанному поручению.
func requireExecutorDocumentAccess(
	auth *AuthService,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	documentID uuid.UUID,
	documentType string,
	nomenclatureID uuid.UUID,
) error {
	if auth == nil {
		return models.ErrUnauthorized
	}

	if auth.HasActiveRole("clerk") {
		return nil
	}
	if !auth.HasActiveRole("executor") {
		return models.ErrForbidden
	}

	allowedByDepartment, err := hasExecutorNomenclatureAccess(auth, depRepo, nomenclatureID)
	if err != nil {
		return err
	}
	if allowedByDepartment {
		return nil
	}

	currentUserID, err := auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}

	hasAssignmentAccess, err := assignmentRepo.HasDocumentAccess(currentUserID, documentID, documentType)
	if err != nil {
		return err
	}
	if hasAssignmentAccess {
		return nil
	}

	hasAcknowledgmentAccess, err := acknowledgmentRepo.HasDocumentAccess(currentUserID, documentID, documentType)
	if err != nil {
		return err
	}
	if hasAcknowledgmentAccess {
		return nil
	}

	return models.ErrForbidden
}

// requireDocumentReadAccess проверяет доступ к документу по роли и номенклатуре.
func requireDocumentReadAccess(
	auth *AuthService,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	incomingRepo IncomingDocStore,
	outgoingRepo OutgoingDocStore,
	documentType string,
	documentID uuid.UUID,
) error {
	if err := requireDocumentDomainReadRole(auth); err != nil {
		return err
	}

	if auth.HasActiveRole("clerk") {
		return nil
	}

	switch documentType {
	case "incoming":
		doc, err := incomingRepo.GetByID(documentID)
		if err != nil {
			return err
		}
		if doc == nil {
			return nil
		}
		return requireExecutorDocumentAccess(auth, depRepo, assignmentRepo, acknowledgmentRepo, doc.ID, "incoming", doc.NomenclatureID)
	case "outgoing":
		doc, err := outgoingRepo.GetByID(documentID)
		if err != nil {
			return err
		}
		if doc == nil {
			return nil
		}
		return requireExecutorDocumentAccess(auth, depRepo, assignmentRepo, acknowledgmentRepo, doc.ID, "outgoing", doc.NomenclatureID)
	default:
		return models.ErrForbidden
	}
}

// requireAnyDocumentReadAccess пытается определить тип документа по ID.
func requireAnyDocumentReadAccess(
	auth *AuthService,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	incomingRepo IncomingDocStore,
	outgoingRepo OutgoingDocStore,
	documentID uuid.UUID,
) error {
	if err := requireDocumentDomainReadRole(auth); err != nil {
		return err
	}

	if auth.HasActiveRole("clerk") {
		return nil
	}

	if doc, err := incomingRepo.GetByID(documentID); err == nil && doc != nil {
		return requireExecutorDocumentAccess(auth, depRepo, assignmentRepo, acknowledgmentRepo, doc.ID, "incoming", doc.NomenclatureID)
	} else if err != nil {
		return err
	}

	if doc, err := outgoingRepo.GetByID(documentID); err == nil && doc != nil {
		return requireExecutorDocumentAccess(auth, depRepo, assignmentRepo, acknowledgmentRepo, doc.ID, "outgoing", doc.NomenclatureID)
	} else if err != nil {
		return err
	}

	return nil
}

func isAssignmentAccessibleToExecutor(currentUserID string, assignment *models.Assignment) bool {
	if assignment == nil {
		return false
	}
	if assignment.ExecutorID.String() == currentUserID {
		return true
	}
	for _, coExecutorID := range assignment.CoExecutorIDs {
		if coExecutorID == currentUserID {
			return true
		}
	}
	return false
}
