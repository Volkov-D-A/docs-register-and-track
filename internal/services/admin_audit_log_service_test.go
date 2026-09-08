package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAdminAuditLogStore struct{}

func (s *stubAdminAuditLogStore) Create(req models.CreateAdminAuditLogRequest) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (s *stubAdminAuditLogStore) GetAll(limit, offset int) ([]models.AdminAuditLog, int, error) {
	return nil, 0, nil
}

func setupAdminAuditLogServiceWithRoles(t *testing.T, roles []string) (*AdminAuditLogService, *testPrincipal) {
	t.Helper()
	userRepo := mocks.NewUserStore(t)
	auth := newTestPrincipal(userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	user := &models.User{
		ID:    uuid.New(),
		Login: "multi_audit_" + uuid.New().String(),

		IsActive: true,
	}
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	return NewAdminAuditLogService(&stubAdminAuditLogStore{}, auth), auth
}

func TestAdminAuditLogService_GetAll_RequiresAdminRole(t *testing.T) {
	svc, _ := setupAdminAuditLogServiceWithRoles(t, []string{"admin", "clerk"})

	result, err := svc.GetAll(1, 10)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAdminAuditLogService_GetAll_ForbiddenWithoutAdminRole(t *testing.T) {
	svc, _ := setupAdminAuditLogServiceWithRoles(t, []string{"clerk"})

	result, err := svc.GetAll(1, 10)
	require.Error(t, err)
	assert.Equal(t, models.ErrForbidden, err)
	assert.Nil(t, result)
}
