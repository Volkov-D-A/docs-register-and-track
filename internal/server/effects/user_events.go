package effects

import (
	"encoding/json"

	"github.com/google/uuid"

	ports "github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

func UserEventMetadata(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func DocumentNumberLabel(number string) string {
	if number == "" {
		return "без номера"
	}
	return number
}

func EventActorID(auth ports.DocumentAccessPrincipal) *uuid.UUID {
	if auth == nil {
		return nil
	}
	currentUserID, err := auth.GetCurrentUserUUID()
	if err != nil || currentUserID == uuid.Nil {
		return nil
	}
	return &currentUserID
}

func EventActorExcluded(auth ports.DocumentAccessPrincipal) map[uuid.UUID]struct{} {
	excluded := make(map[uuid.UUID]struct{})
	actorID := EventActorID(auth)
	if actorID != nil {
		excluded[*actorID] = struct{}{}
	}
	return excluded
}
