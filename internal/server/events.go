package server

import (
	"encoding/json"
	"fmt"
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
