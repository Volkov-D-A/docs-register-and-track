package server

import (
	"errors"
	"net/http"
)

// requireReadySchema covers the entire request, including authentication and
// streaming, so migrations cannot race with an already admitted operation.
func (api *managementAPI) requireReadySchema(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		unavailable := func() {
			writeAPIError(w, http.StatusServiceUnavailable, "maintenance", errors.New("Сервис временно недоступен: обслуживание базы данных."))
		}
		// A pending migration also rejects new requests while existing ones drain.
		if api.backupPending.Load() || !api.schemaRequests.TryRLock() {
			unavailable()
			return
		}
		defer api.schemaRequests.RUnlock()
		if api.lifecycle != nil && api.lifecycle.CheckReady() != nil {
			unavailable()
			return
		}
		next.ServeHTTP(w, r)
	})
}
