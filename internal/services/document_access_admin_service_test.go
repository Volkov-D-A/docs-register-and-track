package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type spyDocumentAccessStore struct {
	replaceCalled bool
}

func (s *spyDocumentAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	return false, nil
}

func (s *spyDocumentAccessStore) HasSystemPermission(permission, userID string) (bool, error) {
	return permission == models.SystemPermissionAdmin, nil
}

func (s *spyDocumentAccessStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	return &models.UserDocumentAccessProfile{}, nil
}

func (s *spyDocumentAccessStore) ReplaceUserAccessProfile(userID string, systemPermissions []models.UserSystemPermissionRule, permissions []models.UserDocumentPermissionRule) error {
	s.replaceCalled = true
	return nil
}

func TestDocumentAccessAdminService_UpdateUserAccessProfile_RejectsUnsupportedDocumentAction(t *testing.T) {
	userRepo := mocks.NewUserStore(t)
	accessRepo := &spyDocumentAccessStore{}
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(accessRepo)

	adminID := uuid.New()
	auth.currentUserID = adminID

	targetUserID := uuid.New()
	userRepo.On("GetByID", targetUserID).Return(&models.User{ID: targetUserID, IsActive: true}, nil).Once()

	svc := NewDocumentAccessAdminService(auth, accessRepo, userRepo)
	err := svc.UpdateUserAccessProfile(models.UpdateUserDocumentAccessRequest{
		UserID: targetUserID.String(),
		Permissions: []models.UserDocumentPermissionRule{
			{
				KindCode:  string(models.DocumentKindIncomingLetter),
				Action:    "delete",
				IsAllowed: true,
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `действие "delete" не поддерживается`)
	assert.False(t, accessRepo.replaceCalled)
}
