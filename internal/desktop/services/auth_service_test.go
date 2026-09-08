package services

import (
	"context"
	"errors"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type fakeServerAuthClient struct {
	user                        *dto.User
	loginCalls                  int
	logoutCalls                 int
	changePasswordCalls         int
	changeRequiredPasswordCalls int
	updateProfileCalls          int
	profileRequest              models.UpdateProfileRequest
}

func (f *fakeServerAuthClient) Login(context.Context, string, string) (*dto.User, error) {
	f.loginCalls++
	return f.user, nil
}
func (f *fakeServerAuthClient) Logout(context.Context) error {
	f.logoutCalls++
	return nil
}
func (f *fakeServerAuthClient) Me(context.Context) (*dto.User, error) { return f.user, nil }
func (f *fakeServerAuthClient) ChangePassword(context.Context, string, string) error {
	f.changePasswordCalls++
	return nil
}
func (f *fakeServerAuthClient) ChangeRequiredPassword(context.Context, string, string, string) error {
	f.changeRequiredPasswordCalls++
	return nil
}

func (f *fakeServerAuthClient) UpdateProfile(_ context.Context, req models.UpdateProfileRequest) error {
	f.updateProfileCalls++
	f.profileRequest = req
	return nil
}

func TestAuthServiceUsesRequiredServerSessionWhenConfigured(t *testing.T) {
	userID := uuid.New()
	client := &fakeServerAuthClient{user: &dto.User{ID: userID.String(), Login: "server-user", IsActive: true}}
	service := NewAuthService(client, nil, nil, nil)

	user, err := service.Login("server-user", "Passw0rd!")
	require.NoError(t, err)
	assert.Equal(t, userID.String(), user.ID)
	require.NoError(t, NewPrincipal(service, nil).RequireAuthenticated())
	require.NoError(t, service.Logout())
	assert.Equal(t, 1, client.loginCalls)
	assert.Equal(t, 1, client.logoutCalls)
}

func TestAuthServiceUsesServerForPasswordChangesWhenConfigured(t *testing.T) {
	userID := uuid.New()
	client := &fakeServerAuthClient{user: &dto.User{ID: userID.String(), Login: "server-user", IsActive: true}}
	service := NewAuthService(client, nil, nil, nil)
	_, err := service.Login("server-user", "Passw0rd!")
	require.NoError(t, err)

	require.NoError(t, service.ChangePassword("Passw0rd!", "NewPassw0rd!"))
	assert.Equal(t, 1, client.changePasswordCalls)
	assert.False(t, service.IsAuthenticated())

	require.NoError(t, service.ChangeRequiredPassword("server-user", "Passw0rd!", "NewPassw0rd!"))
	assert.Equal(t, 1, client.changeRequiredPasswordCalls)
}

func TestAuthServiceUsesServerForProfileUpdateWhenConfigured(t *testing.T) {
	client := &fakeServerAuthClient{user: &dto.User{ID: uuid.NewString(), Login: "server-user", IsActive: true}}
	service := NewAuthService(client, nil, nil, nil)
	req := models.UpdateProfileRequest{Login: "renamed", FullName: "Renamed User"}

	require.NoError(t, service.UpdateProfile(req))

	assert.Equal(t, 1, client.updateProfileCalls)
	assert.Equal(t, req, client.profileRequest)
}

func TestAuthServiceRequiresServerClient(t *testing.T) {
	service := NewAuthService(nil, nil, nil, nil)
	_, err := service.Login("user", "Passw0rd!")
	require.ErrorIs(t, err, errServerAuthNotConfigured)
	_, err = service.GetCurrentUser()
	require.ErrorIs(t, err, errServerAuthNotConfigured)
	require.ErrorIs(t, NewPrincipal(service, nil).RequireAuthenticated(), errServerAuthNotConfigured)
	require.ErrorIs(t, service.ChangePassword("old", "new"), errServerAuthNotConfigured)
	require.ErrorIs(t, service.ChangeRequiredPassword("user", "old", "new"), errServerAuthNotConfigured)
	require.ErrorIs(t, service.UpdateProfile(models.UpdateProfileRequest{}), errServerAuthNotConfigured)
	_, err = service.NeedsInitialSetup()
	require.ErrorIs(t, err, errServerAuthNotConfigured)
	require.ErrorIs(t, service.InitialSetup("Passw0rd!"), errServerAuthNotConfigured)
	require.False(t, NewPrincipal(service, nil).HasSystemPermission(models.SystemPermissionAdmin))
	id, name := NewPrincipal(service, nil).GetCurrentAuditInfo()
	require.Equal(t, uuid.Nil, id)
	require.Equal(t, "system", name)
}

type setupAuthClient struct {
	fakeServerAuthClient
	password string
	err      error
}

func (c *setupAuthClient) NeedsInitialSetup(context.Context) (bool, error) { return true, c.err }
func (c *setupAuthClient) InitialSetup(_ context.Context, password string) error {
	c.password = password
	return c.err
}
func TestAuthServiceForwardsBootstrapAndErrors(t *testing.T) {
	client := &setupAuthClient{}
	service := NewAuthService(client, client, nil, nil)
	required, err := service.NeedsInitialSetup()
	require.NoError(t, err)
	require.True(t, required)
	require.NoError(t, service.InitialSetup("Passw0rd!"))
	require.Equal(t, "Passw0rd!", client.password)
	client.err = errors.New("server unavailable")
	_, err = service.NeedsInitialSetup()
	require.ErrorIs(t, err, client.err)
	require.ErrorIs(t, service.InitialSetup("another"), client.err)
}

func TestAuthServiceMaintenanceKeepsLoginAvailableAndBlocksProtectedOperations(t *testing.T) {
	client := &fakeServerAuthClient{user: &dto.User{ID: uuid.NewString(), IsActive: true}}
	service := NewAuthService(client, nil, nil, nil)
	maintenance := errors.New("maintenance")
	principal := NewPrincipal(service, fakeReadiness{err: maintenance})
	_, err := service.Login("user", "Passw0rd!")
	require.NoError(t, err)
	require.ErrorIs(t, principal.RequireAuthenticated(), maintenance)
	_, err = principal.GetCurrentUserUUID()
	require.ErrorIs(t, err, maintenance)
	require.ErrorIs(t, principal.RequireSystemPermission(models.SystemPermissionAdmin), maintenance)
	require.NoError(t, service.Logout())
}

func (f *fakeServerAuthClient) SessionState() serverclient.SessionState {
	return serverclient.SessionState{}
}

type fakeReadiness struct{ err error }

func (f fakeReadiness) CheckReady() error { return f.err }
