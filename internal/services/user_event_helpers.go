package services

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func userEventMetadata(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func appendUniqueUserID(items []uuid.UUID, item uuid.UUID) []uuid.UUID {
	if item == uuid.Nil {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func collectUserIDsWithDocumentAction(
	userRepo UserStore,
	access *DocumentAccessService,
	kind string,
	action string,
	excluded map[uuid.UUID]struct{},
) ([]uuid.UUID, error) {
	if userRepo == nil || access == nil || access.accessRepo == nil {
		return nil, nil
	}

	users, err := userRepo.GetAll()
	if err != nil {
		return nil, err
	}

	recipients := make([]uuid.UUID, 0)
	for _, user := range users {
		if !user.IsActive {
			continue
		}
		if _, skip := excluded[user.ID]; skip {
			continue
		}

		departmentID := ""
		if user.DepartmentID != nil {
			departmentID = user.DepartmentID.String()
		} else if user.Department != nil {
			departmentID = user.Department.ID.String()
		}

		allowed, err := access.accessRepo.HasPermission(kind, action, departmentID, user.ID.String())
		if err != nil {
			return nil, err
		}
		if allowed {
			recipients = appendUniqueUserID(recipients, user.ID)
		}
	}

	return recipients, nil
}

func documentNumberLabel(number string) string {
	if number == "" {
		return "без номера"
	}
	return number
}

func eventActorID(auth *AuthService) *uuid.UUID {
	if auth == nil {
		return nil
	}
	currentUserID, err := auth.GetCurrentUserUUID()
	if err != nil || currentUserID == uuid.Nil {
		return nil
	}
	return &currentUserID
}

func eventActorExcluded(auth *AuthService) map[uuid.UUID]struct{} {
	excluded := make(map[uuid.UUID]struct{})
	actorID := eventActorID(auth)
	if actorID != nil {
		excluded[*actorID] = struct{}{}
	}
	return excluded
}

func createUserEventIfEnabled(events *UserEventService, req models.CreateUserEventRequest) {
	if events == nil {
		return
	}
	if _, err := events.create(req); err != nil {
		slog.Warn(
			"failed to create user event",
			"error", err,
			"recipient_user_id", req.RecipientUserID,
			"document_id", req.DocumentID,
			"document_kind", req.DocumentKind,
			"entity_type", req.EntityType,
			"entity_id", req.EntityID,
			"event_type", req.EventType,
		)
	}
}
