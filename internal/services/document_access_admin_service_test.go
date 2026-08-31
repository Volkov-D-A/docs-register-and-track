package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type testDocumentAccessClient struct{ service *DocumentAccessAdminService }

func (c testDocumentAccessClient) GetUserAccessProfile(_ context.Context, userID string) (*models.UserDocumentAccessProfile, error) {
	if err := c.service.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return nil, err
	}
	if _, err := parseUUID(userID); err != nil {
		return nil, err
	}
	return c.service.accessRepo.GetUserAccessProfile(userID)
}

func (c testDocumentAccessClient) UpdateUserAccessProfile(_ context.Context, req models.UpdateUserDocumentAccessRequest) error {
	if err := c.service.auth.RequireSystemPermission(models.SystemPermissionAdmin); err != nil {
		return err
	}
	id, err := parseUUID(req.UserID)
	if err != nil {
		return err
	}
	user, err := c.service.userRepo.GetByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return models.NewNotFound("пользователь не найден")
	}
	for _, permission := range req.Permissions {
		kind := models.NormalizeDocumentKind(permission.KindCode)
		if _, ok := models.GetDocumentKindSpec(kind); !ok {
			return models.NewBadRequest(fmt.Sprintf("неизвестный вид документа: %s", permission.KindCode))
		}
		if !kind.SupportsAction(permission.Action) {
			return models.NewBadRequest(fmt.Sprintf("действие %q не поддерживается для вида документа %q", permission.Action, permission.KindCode))
		}
	}
	return activeAdministratorInvariantConflict(c.service.accessRepo.ReplaceUserAccessProfile(req.UserID, req.SystemPermissions, req.Permissions))
}

func newTestDocumentAccessAdminService(auth *AuthService, access DocumentAccessStore, users UserStore) *DocumentAccessAdminService {
	service := NewDocumentAccessAdminService(auth, access, users)
	service.SetServerClient(testDocumentAccessClient{service: service})
	return service
}

type spyDocumentAccessStore struct {
	replaceCalled bool
	profileUserID string
	profile       *models.UserDocumentAccessProfile
}

func (s *spyDocumentAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	return false, nil
}

func (s *spyDocumentAccessStore) HasSystemPermission(permission, userID string) (bool, error) {
	return permission == models.SystemPermissionAdmin, nil
}

func (s *spyDocumentAccessStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	s.profileUserID = userID
	if s.profile != nil {
		return s.profile, nil
	}
	return &models.UserDocumentAccessProfile{}, nil
}

func (s *spyDocumentAccessStore) ReplaceUserAccessProfile(userID string, systemPermissions []models.UserSystemPermissionRule, permissions []models.UserDocumentPermissionRule) error {
	s.replaceCalled = true
	return nil
}

func TestDocumentAccessAdminService_GetUserAccessProfile(t *testing.T) {
	t.Run("returns profile for admin", func(t *testing.T) {
		userRepo := mocks.NewUserStore(t)
		targetUserID := uuid.New().String()
		expected := &models.UserDocumentAccessProfile{
			SystemPermissions: []models.UserSystemPermissionRule{
				{Permission: models.SystemPermissionReferences, IsAllowed: true},
			},
			Permissions: []models.UserDocumentPermissionRule{
				{KindCode: string(models.DocumentKindIncomingLetter), Action: string(models.DocumentActionRead), IsAllowed: true},
			},
		}
		accessRepo := &spyDocumentAccessStore{profile: expected}
		auth := NewAuthService(nil, userRepo)
		auth.SetAccessStore(accessRepo)
		auth.currentUserID = uuid.New()
		userRepo.On("GetByID", auth.currentUserID).Return(&models.User{ID: auth.currentUserID, IsActive: true}, nil).Maybe()

		svc := newTestDocumentAccessAdminService(auth, accessRepo, userRepo)

		profile, err := svc.GetUserAccessProfile(targetUserID)

		require.NoError(t, err)
		assert.Same(t, expected, profile)
		assert.Equal(t, targetUserID, accessRepo.profileUserID)
	})

	t.Run("rejects invalid user id", func(t *testing.T) {
		userRepo := mocks.NewUserStore(t)
		accessRepo := &spyDocumentAccessStore{}
		auth := NewAuthService(nil, userRepo)
		auth.SetAccessStore(accessRepo)
		auth.currentUserID = uuid.New()
		userRepo.On("GetByID", auth.currentUserID).Return(&models.User{ID: auth.currentUserID, IsActive: true}, nil).Maybe()
		svc := newTestDocumentAccessAdminService(auth, accessRepo, userRepo)

		profile, err := svc.GetUserAccessProfile("bad-id")

		require.Error(t, err)
		assert.Nil(t, profile)
		assert.Empty(t, accessRepo.profileUserID)
	})

	t.Run("requires admin permission", func(t *testing.T) {
		userRepo := mocks.NewUserStore(t)
		accessRepo := newRoleMappedDocumentAccessStore(models.SystemPermissionReferences)
		auth := NewAuthService(nil, userRepo)
		auth.SetAccessStore(accessRepo)
		auth.currentUserID = uuid.New()
		userRepo.On("GetByID", auth.currentUserID).Return(&models.User{ID: auth.currentUserID, IsActive: true}, nil).Maybe()
		svc := newTestDocumentAccessAdminService(auth, accessRepo, userRepo)

		profile, err := svc.GetUserAccessProfile(uuid.New().String())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.Nil(t, profile)
	})
}

func TestDocumentAccessAdminService_UpdateUserAccessProfile_RejectsUnsupportedDocumentAction(t *testing.T) {
	userRepo := mocks.NewUserStore(t)
	accessRepo := &spyDocumentAccessStore{}
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(accessRepo)

	adminID := uuid.New()
	auth.currentUserID = adminID
	userRepo.On("GetByID", adminID).Return(&models.User{ID: adminID, IsActive: true}, nil).Maybe()

	targetUserID := uuid.New()
	userRepo.On("GetByID", targetUserID).Return(&models.User{ID: targetUserID, IsActive: true}, nil).Once()

	svc := newTestDocumentAccessAdminService(auth, accessRepo, userRepo)
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

func TestDocumentAccessAdminService_UpdateUserAccessProfile_ReturnsNotFoundForMissingUser(t *testing.T) {
	userRepo := mocks.NewUserStore(t)
	accessRepo := &spyDocumentAccessStore{}
	auth := NewAuthService(nil, userRepo)
	auth.SetAccessStore(accessRepo)
	auth.currentUserID = uuid.New()
	userRepo.On("GetByID", auth.currentUserID).Return(&models.User{ID: auth.currentUserID, IsActive: true}, nil).Maybe()

	targetUserID := uuid.New()
	userRepo.On("GetByID", targetUserID).Return(nil, nil).Once()

	svc := newTestDocumentAccessAdminService(auth, accessRepo, userRepo)
	err := svc.UpdateUserAccessProfile(models.UpdateUserDocumentAccessRequest{
		UserID: targetUserID.String(),
	})

	require.Error(t, err)
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, "NOT_FOUND", appErr.Kind)
	assert.Equal(t, 404, appErr.Code)
	assert.False(t, accessRepo.replaceCalled)
}
