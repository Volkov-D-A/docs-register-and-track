package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/liveevents"
	"github.com/google/uuid"
	"net/http"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// Authorize only under short-lived barriers. Never hold them while streaming.
func (api *managementAPI) eventPrincipal(w http.ResponseWriter, r *http.Request) *authenticatedRequest {
	if api.replacementPending.Load() || !api.replacementRequests.TryRLock() {
		writeAPIError(w, 503, "maintenance", models.ErrForbidden)
		return nil
	}
	defer api.replacementRequests.RUnlock()
	var auth *authenticatedRequest
	api.requireReadySchema(api.requireSession(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		auth = authenticatedFromContext(r.Context())
	}))).ServeHTTP(w, r)
	return auth
}

type eventValidationWriter struct{ header http.Header }

func (w *eventValidationWriter) Header() http.Header       { return w.header }
func (*eventValidationWriter) WriteHeader(int)             {}
func (*eventValidationWriter) Write(p []byte) (int, error) { return len(p), nil }

func beginEvents(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
}
func writeEvent(w http.ResponseWriter, name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	controller := http.NewResponseController(w)
	// Refresh the server write deadline for each event; slow clients are bounded.
	_ = controller.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return err
	}
	return controller.Flush()
}

func (api *managementAPI) sessionEvents(w http.ResponseWriter, r *http.Request) {
	auth := api.eventPrincipal(w, r)
	if auth == nil {
		return
	}
	userID := auth.User.ID
	users, stopUsers := api.events.Subscribe("user:" + userID.String())
	defer stopUsers()
	changes, drainChanges, stopChanges := api.events.SubscribeChanges()
	defer stopChanges()
	backups, stopBackups := api.events.Subscribe("backups")
	defer stopBackups()
	beginEvents(w)
	if writeEvent(w, "resync", nil) != nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		topic := "heartbeat"
		select {
		case <-r.Context().Done():
			return
		case <-users:
			topic = "user-events"
		case <-changes:
			current := api.eventPrincipal(&eventValidationWriter{header: make(http.Header)}, r)
			if current == nil || current.User.ID != userID {
				return
			}
			visible, err := api.eventChanges(current.User, drainChanges())
			if err != nil {
				return
			}
			for _, change := range visible {
				if change.Resource == "resync" {
					if writeEvent(w, "resync", nil) != nil {
						return
					}
				} else if change.Resource == "access" {
					if writeEvent(w, "access-changed", nil) != nil {
						return
					}
				} else if writeEvent(w, "document-changed", change) != nil {
					return
				}
			}
			continue
		case <-backups:
			topic = "backups"
		case <-ticker.C:
		}
		current := api.eventPrincipal(&eventValidationWriter{header: make(http.Header)}, r)
		if current == nil || current.User.ID != userID {
			return
		}
		if topic == "backups" && !contains(current.User.SystemPermissions, models.SystemPermissionAdmin) {
			continue
		}
		if writeEvent(w, topic, nil) != nil {
			return
		}
	}
}

func (api *managementAPI) operationEvents(w http.ResponseWriter, r *http.Request) {
	if !api.backupAvailable(w) {
		return
	}
	id := r.PathValue("id")
	token, _ := bearerToken(r.Header.Get("Authorization"))
	changes, stop := api.backupService.Events.Subscribe("operation:" + id)
	defer stop()
	status, err := api.backupService.OperationStatus(id, token)
	if err != nil {
		writeAPIError(w, 403, "forbidden", models.ErrForbidden)
		return
	}
	beginEvents(w)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		if writeEvent(w, "operation", status) != nil {
			return
		}
		if operationTerminal(status.State) {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-changes:
		case <-ticker.C:
		}
		// Revalidate the scoped capability, including expiry, without touching the DB.
		status, err = api.backupService.OperationStatus(id, token)
		if err != nil {
			return
		}
	}
}

func operationTerminal(state string) bool {
	switch state {
	case "completed", "failed", "cancelled", "interrupted", "rolled_back", "rollback_failed", "recovery_required":
		return true
	}
	return false
}

// Use the same read policy as HTTP queries, including active substitutions.
type eventDocumentAccess interface {
	ResolveReadableDocuments([]uuid.UUID) (map[uuid.UUID]*models.Document, error)
	GetCurrentUserAndSubstitutionSubjectIDs() ([]uuid.UUID, error)
}

func (api *managementAPI) eventChanges(user *models.User, changes []liveevents.Change) ([]liveevents.Change, error) {
	if api.replacementPending.Load() || !api.replacementRequests.TryRLock() {
		return nil, models.ErrForbidden
	}
	defer api.replacementRequests.RUnlock()
	if api.backupPending.Load() || !api.schemaRequests.TryRLock() {
		return nil, models.ErrForbidden
	}
	defer api.schemaRequests.RUnlock()
	return api.readableChanges(user, changes)
}

func (api *managementAPI) readableChanges(user *models.User, changes []liveevents.Change) ([]liveevents.Change, error) {
	result := make([]liveevents.Change, 0, len(changes))
	ids := make([]uuid.UUID, 0, len(changes))
	for _, change := range changes {
		if change.Resource == "resync" {
			return []liveevents.Change{change}, nil
		}
		if id, err := uuid.Parse(change.DocumentID); err == nil {
			ids = append(ids, id)
		}
	}
	if api.eventAccess == nil {
		for _, change := range changes {
			if change.Resource == "access" && len(change.Audience) == 0 {
				result = append(result, change)
			}
		}
		return result, nil
	}
	access := api.eventAccess(user)
	readable := make(map[uuid.UUID]*models.Document)
	var err error
	if len(ids) > 0 {
		readable, err = access.ResolveReadableDocuments(ids)
	}
	if err != nil && !errors.Is(err, models.ErrForbidden) {
		return nil, err
	}
	subjects, err := access.GetCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return nil, err
	}
	for _, change := range changes {
		if change.Resource == "access" {
			allowed := len(change.Audience) == 0
			for _, recipient := range change.Audience {
				for _, subject := range subjects {
					if recipient == subject {
						allowed = true
					}
				}
			}
			if allowed {
				result = append(result, change)
			}
			continue
		}
		id, err := uuid.Parse(change.DocumentID)
		if err != nil {
			continue
		}
		_, allowed := readable[id]
		for _, previous := range change.PreviousReaders {
			for _, subject := range subjects {
				if previous == subject {
					allowed = true
				}
			}
		}
		if allowed {
			if document := readable[id]; document != nil {
				change.DocumentKind = string(document.Kind)
			}
			result = append(result, change)
		}
	}
	return result, nil
}
