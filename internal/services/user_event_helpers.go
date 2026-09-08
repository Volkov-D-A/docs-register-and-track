package services

import (
	"encoding/json"

	"github.com/google/uuid"

	ports "github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
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

func documentNumberLabel(number string) string {
	if number == "" {
		return "без номера"
	}
	return number
}

func eventActorID(auth ports.DocumentAccessPrincipal) *uuid.UUID {
	if auth == nil {
		return nil
	}
	currentUserID, err := auth.GetCurrentUserUUID()
	if err != nil || currentUserID == uuid.Nil {
		return nil
	}
	return &currentUserID
}

func eventActorExcluded(auth ports.DocumentAccessPrincipal) map[uuid.UUID]struct{} {
	excluded := make(map[uuid.UUID]struct{})
	actorID := eventActorID(auth)
	if actorID != nil {
		excluded[*actorID] = struct{}{}
	}
	return excluded
}
