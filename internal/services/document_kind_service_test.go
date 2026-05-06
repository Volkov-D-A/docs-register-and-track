package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/security"
)

func setupDocumentKindService(t *testing.T, role string, isDocumentParticipant bool) (*DocumentKindService, *AuthService) {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	depRepo := mocks.NewDepartmentStore(t)
	assignmentRepo := mocks.NewAssignmentStore(t)
	ackRepo := mocks.NewAcknowledgmentStore(t)
	auth := NewAuthService(nil, userRepo)

	password := "Passw0rd!"
	hash, _ := security.HashPassword(password)
	user := &models.User{
		ID:                    uuid.New(),
		Login:                 role + "_kind",
		PasswordHash:          hash,
		IsActive:              true,
		IsDocumentParticipant: isDocumentParticipant,
	}

	auth.SetAccessStore(newRoleMappedDocumentAccessStore(role))
	userRepo.On("GetByLogin", user.Login).Return(user, nil).Once()
	_, err := auth.Login(user.Login, password)
	require.NoError(t, err)
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()

	access := NewDocumentAccessService(auth, depRepo, assignmentRepo, ackRepo, newRoleMappedDocumentAccessStore(role), nil, nil, nil)
	return NewDocumentKindService(access), auth
}

func TestDocumentKindService_GetCurrentAccessSummary(t *testing.T) {
	t.Parallel()

	t.Run("clerk gets document sections and registration kinds", func(t *testing.T) {
		service, auth := setupDocumentKindService(t, "clerk", false)
		defer auth.Logout()

		summary, err := service.GetCurrentAccessSummary()
		require.NoError(t, err)
		require.Len(t, summary.DocumentKinds, 4)
		assert.True(t, summary.DocumentDomainAccess)
		assert.True(t, summary.Sections.Dashboard)
		assert.True(t, summary.Sections.Incoming)
		assert.True(t, summary.Sections.Outgoing)
		assert.True(t, summary.Sections.Appeals)
		assert.True(t, summary.Sections.Orders)
		assert.True(t, summary.Sections.Assignments)
		assert.ElementsMatch(t, []string{"incoming_letter", "outgoing_letter", "citizen_appeal", "administrative_order"}, summary.RegistrationKinds)
		assert.True(t, summary.DocumentKinds[0].CanRegister)
		assert.True(t, summary.DocumentKinds[0].CanReadFull)
	})

	t.Run("participant opens document pages without full read", func(t *testing.T) {
		service, auth := setupDocumentKindService(t, "", true)
		defer auth.Logout()

		summary, err := service.GetCurrentAccessSummary()
		require.NoError(t, err)
		assert.True(t, summary.DocumentDomainAccess)
		assert.True(t, summary.Sections.Dashboard)
		assert.True(t, summary.Sections.Incoming)
		assert.True(t, summary.Sections.Outgoing)
		assert.True(t, summary.Sections.Appeals)
		assert.True(t, summary.Sections.Orders)
		assert.True(t, summary.Sections.Assignments)
		assert.Empty(t, summary.RegistrationKinds)
		assert.False(t, summary.DocumentKinds[0].CanRegister)
		assert.False(t, summary.DocumentKinds[0].CanReadFull)
	})

	t.Run("admin only gets settings section", func(t *testing.T) {
		service, auth := setupDocumentKindService(t, "admin", false)
		defer auth.Logout()

		summary, err := service.GetCurrentAccessSummary()
		require.NoError(t, err)
		assert.False(t, summary.DocumentDomainAccess)
		assert.False(t, summary.Sections.Dashboard)
		assert.False(t, summary.Sections.Incoming)
		assert.False(t, summary.Sections.Orders)
		assert.False(t, summary.Sections.Assignments)
		assert.True(t, summary.Sections.Settings)
		assert.ElementsMatch(t, []string{"admin"}, summary.SystemPermissions)
	})
}
