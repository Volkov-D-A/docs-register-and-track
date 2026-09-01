package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type fakeAdminAuditAPI struct{ page, pageSize int }

func (f *fakeAdminAuditAPI) GetAll(page, pageSize int) (*dto.AdminAuditLogPage, error) {
	f.page, f.pageSize = page, pageSize
	return &dto.AdminAuditLogPage{Items: []dto.AdminAuditLog{}, Page: page}, nil
}

type fakeOutboxAdminAPI struct {
	limit    int
	requeued string
}

func (*fakeOutboxAdminAPI) GetStats() (models.OutboxStats, error) {
	return models.OutboxStats{Failed: 2}, nil
}
func (f *fakeOutboxAdminAPI) GetFailed(limit int) ([]models.FailedOutboxEvent, error) {
	f.limit = limit
	return []models.FailedOutboxEvent{}, nil
}
func (f *fakeOutboxAdminAPI) Requeue(id string) error { f.requeued = id; return nil }

func TestAdminOperationsRequireAdminAndUseRequestPrincipal(t *testing.T) {
	regularAPI, _, regularToken := authenticatedUserAPI(t, nil)
	regularRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit", nil)
	regularRequest.Header.Set("Authorization", "Bearer "+regularToken)
	regularResponse := httptest.NewRecorder()
	regularAPI.Handler().ServeHTTP(regularResponse, regularRequest)
	assert.Equal(t, http.StatusForbidden, regularResponse.Code)

	api, _, token := authenticatedUserAPI(t, []string{models.SystemPermissionAdmin})
	audit, outbox := &fakeAdminAuditAPI{}, &fakeOutboxAdminAPI{}
	var auditPrincipal, outboxPrincipal uuid.UUID
	api.adminAudit = func(user *models.User) adminAuditAPI { auditPrincipal = user.ID; return audit }
	api.outboxAdmin = func(user *models.User) outboxAdminAPI { outboxPrincipal = user.ID; return outbox }

	auditRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit?page=2&pageSize=20", nil)
	auditRequest.Header.Set("Authorization", "Bearer "+token)
	auditResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(auditResponse, auditRequest)
	require.Equal(t, http.StatusOK, auditResponse.Code, auditResponse.Body.String())
	assert.Equal(t, 2, audit.page)
	assert.Equal(t, 20, audit.pageSize)
	assert.NotEqual(t, uuid.Nil, auditPrincipal)

	id := uuid.NewString()
	requeue := httptest.NewRequest(http.MethodPost, "/api/v1/admin/outbox/"+id+"/requeue", nil)
	requeue.Header.Set("Authorization", "Bearer "+token)
	requeueResponse := httptest.NewRecorder()
	api.Handler().ServeHTTP(requeueResponse, requeue)
	require.Equal(t, http.StatusNoContent, requeueResponse.Code, requeueResponse.Body.String())
	assert.Equal(t, id, outbox.requeued)
	assert.NotEqual(t, uuid.Nil, outboxPrincipal)
}
