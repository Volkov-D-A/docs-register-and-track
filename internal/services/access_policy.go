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

// requireExecutorNomenclatureAccess проверяет доступ исполнителя к документу по номенклатуре.
func requireExecutorNomenclatureAccess(auth *AuthService, depRepo DepartmentStore, nomenclatureID uuid.UUID) error {
	if auth == nil {
		return models.ErrUnauthorized
	}

	if auth.HasActiveRole("clerk") {
		return nil
	}
	if !auth.HasActiveRole("executor") {
		return models.ErrForbidden
	}

	user, err := auth.GetCurrentUser()
	if err != nil {
		return err
	}
	if user == nil || user.Department == nil || user.Department.ID == "" {
		return models.ErrForbidden
	}

	departmentID, err := uuid.Parse(user.Department.ID)
	if err != nil {
		return models.ErrForbidden
	}

	allowedNomenclatures, err := depRepo.GetNomenclatureIDs(departmentID)
	if err != nil {
		return err
	}
	nomenclatureIDStr := nomenclatureID.String()
	for _, allowedID := range allowedNomenclatures {
		if allowedID == nomenclatureIDStr {
			return nil
		}
	}

	return models.ErrForbidden
}

// requireDocumentReadAccess проверяет доступ к документу по роли и номенклатуре.
func requireDocumentReadAccess(
	auth *AuthService,
	depRepo DepartmentStore,
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
		return requireExecutorNomenclatureAccess(auth, depRepo, doc.NomenclatureID)
	case "outgoing":
		doc, err := outgoingRepo.GetByID(documentID)
		if err != nil {
			return err
		}
		if doc == nil {
			return nil
		}
		return requireExecutorNomenclatureAccess(auth, depRepo, doc.NomenclatureID)
	default:
		return models.ErrForbidden
	}
}

// requireAnyDocumentReadAccess пытается определить тип документа по ID.
func requireAnyDocumentReadAccess(
	auth *AuthService,
	depRepo DepartmentStore,
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
		return requireExecutorNomenclatureAccess(auth, depRepo, doc.NomenclatureID)
	} else if err != nil {
		return err
	}

	if doc, err := outgoingRepo.GetByID(documentID); err == nil && doc != nil {
		return requireExecutorNomenclatureAccess(auth, depRepo, doc.NomenclatureID)
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
