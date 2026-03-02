package services

import (
	"docflow/internal/mocks"
	"docflow/internal/models"
	"docflow/internal/security"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupAckService(t *testing.T, role string) (
	*AcknowledgmentService, *mocks.AcknowledgmentStore, *mocks.UserStore, *AuthService,
) {
	t.Helper()
	ackRepo := mocks.NewAcknowledgmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:           uuid.New(),
		Login:        role + "_ack",
		PasswordHash: hash,
		IsActive:     true,
		Roles:        []string{role},
	}
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)

	svc := NewAcknowledgmentService(ackRepo, userRepo, auth)
	return svc, ackRepo, userRepo, auth
}

func setupAckServiceNotAuth(t *testing.T) *AcknowledgmentService {
	t.Helper()
	ackRepo := mocks.NewAcknowledgmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	return NewAcknowledgmentService(ackRepo, userRepo, auth)
}

func TestAcknowledgmentService_Create(t *testing.T) {
	docID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _ := setupAckService(t, "clerk")
		repo.On("Create", mock.AnythingOfType("*models.Acknowledgment")).Return(nil).Once()
		result, err := svc.Create(docID.String(), "incoming", "text", []string{user1.String(), user2.String()})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, docID.String(), result.DocumentID)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "executor")
		result, err := svc.Create(docID.String(), "incoming", "text", []string{user1.String()})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("no users selected", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "clerk")
		result, err := svc.Create(docID.String(), "incoming", "text", []string{"not-a-uuid"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не выбраны пользователи")
		assert.Nil(t, result)
	})

	t.Run("invalid document ID", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "clerk")
		result, err := svc.Create("not-a-uuid", "incoming", "text", []string{user1.String()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})
}

func TestAcknowledgmentService_GetList(t *testing.T) {
	docID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _ := setupAckService(t, "executor")
		acks := []models.Acknowledgment{{ID: uuid.New(), DocumentID: docID}}
		repo.On("GetByDocumentID", docID).Return(acks, nil).Once()
		result, err := svc.GetList(docID.String())
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc := setupAckServiceNotAuth(t)
		result, err := svc.GetList(docID.String())
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "executor")
		result, err := svc.GetList("not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid document ID")
		assert.Nil(t, result)
	})
}

func TestAcknowledgmentService_GetPendingForCurrentUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, _, auth := setupAckService(t, "executor")
		userUUID, _ := uuid.Parse(auth.GetCurrentUserID())
		acks := []models.Acknowledgment{{ID: uuid.New()}}
		repo.On("GetPendingForUser", userUUID).Return(acks, nil).Once()
		result, err := svc.GetPendingForCurrentUser()
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc := setupAckServiceNotAuth(t)
		result, err := svc.GetPendingForCurrentUser()
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
		assert.Nil(t, result)
	})
}

func TestAcknowledgmentService_GetAllActive(t *testing.T) {
	t.Run("success admin", func(t *testing.T) {
		svc, repo, _, _ := setupAckService(t, "admin")
		acks := []models.Acknowledgment{{ID: uuid.New()}, {ID: uuid.New()}}
		repo.On("GetAllActive").Return(acks, nil).Once()
		result, err := svc.GetAllActive()
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _ := setupAckService(t, "clerk")
		repo.On("GetAllActive").Return([]models.Acknowledgment{}, nil).Once()
		result, err := svc.GetAllActive()
		require.NoError(t, err)
		assert.Len(t, result, 0)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "executor")
		result, err := svc.GetAllActive()
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})
}

func TestAcknowledgmentService_MarkViewed(t *testing.T) {
	ackID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, auth := setupAckService(t, "executor")
		userUUID, _ := uuid.Parse(auth.GetCurrentUserID())
		repo.On("MarkViewed", ackID, userUUID).Return(nil).Once()
		err := svc.MarkViewed(ackID.String())
		require.NoError(t, err)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc := setupAckServiceNotAuth(t)
		err := svc.MarkViewed(ackID.String())
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
	})

	t.Run("invalid ID", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "executor")
		err := svc.MarkViewed("not-a-uuid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid acknowledgment ID")
	})
}

func TestAcknowledgmentService_MarkConfirmed(t *testing.T) {
	ackID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, auth := setupAckService(t, "executor")
		userUUID, _ := uuid.Parse(auth.GetCurrentUserID())
		repo.On("MarkConfirmed", ackID, userUUID).Return(nil).Once()
		err := svc.MarkConfirmed(ackID.String())
		require.NoError(t, err)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc := setupAckServiceNotAuth(t)
		err := svc.MarkConfirmed(ackID.String())
		require.Error(t, err)
		assert.Equal(t, ErrNotAuthenticated, err)
	})
}

func TestAcknowledgmentService_Delete(t *testing.T) {
	ackID := uuid.New()

	t.Run("success admin", func(t *testing.T) {
		svc, repo, _, _ := setupAckService(t, "admin")
		repo.On("Delete", ackID).Return(nil).Once()
		err := svc.Delete(ackID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, _, _, _ := setupAckService(t, "executor")
		err := svc.Delete(ackID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})
}
