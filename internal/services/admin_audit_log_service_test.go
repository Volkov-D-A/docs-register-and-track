package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"

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

func setupAdminAuditLogServiceWithRoles(t *testing.T, roles []string) (*AdminAuditLogService, *AuthService) {
	t.Helper()
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        "multi_audit_" + uuid.New().String(),
		PasswordHash: hash,
		IsActive:     true,
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
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
