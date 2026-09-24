package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/operations"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

var errServerAuthNotConfigured = errors.New("docflow-server auth client is not configured")

// AuthSessionClient owns the session revision and invalidation state.
type AuthSessionClient interface {
	serverclient.AuthClient
	SessionState() serverclient.SessionState
}

// AuthService exposes only desktop UI operations. Session state belongs to the HTTP client.
type AuthService struct {
	serverAuth   AuthSessionClient
	initialSetup serverclient.InitialSetupClient
	lifecycle    *operations.Lifecycle
}

func NewAuthService(client AuthSessionClient, setup serverclient.InitialSetupClient, lifecycle *operations.Lifecycle) *AuthService {
	return &AuthService{serverAuth: client, initialSetup: setup, lifecycle: lifecycle}
}

func (s *AuthService) operationContext(timeout time.Duration) (context.Context, func()) {
	ctx, release := s.lifecycle.OperationContext()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return ctx, func() { cancel(); release() }
}

func (s *AuthService) Login(login, password string) (*dto.User, error) {
	if s.serverAuth == nil {
		return nil, errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	user, err := s.serverAuth.Login(ctx, login, password)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, models.NewInternal("Сервис не вернул пользователя", nil)
	}
	if _, err := uuid.Parse(user.ID); err != nil {
		return nil, models.NewInternal("Сервис вернул некорректный идентификатор пользователя", err)
	}
	return user, nil
}

func (s *AuthService) Logout() error {
	if s.serverAuth != nil {
		ctx, cancel := s.operationContext(15 * time.Second)
		defer cancel()
		if err := s.serverAuth.Logout(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthService) GetCurrentUser() (*dto.User, error) {
	if s.serverAuth == nil {
		return nil, errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	return s.serverAuth.Me(ctx)
}

func (s *AuthService) ChangePassword(oldPassword, newPassword string) error {
	if s.serverAuth == nil {
		return errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	return s.serverAuth.ChangePassword(ctx, oldPassword, newPassword)
}

func (s *AuthService) ChangeRequiredPassword(login, oldPassword, newPassword string) error {
	if s.serverAuth == nil {
		return errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	return s.serverAuth.ChangeRequiredPassword(ctx, login, oldPassword, newPassword)
}

func (s *AuthService) UpdateProfile(req models.UpdateProfileRequest) error {
	if s.serverAuth == nil {
		return errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	return s.serverAuth.UpdateProfile(ctx, req)
}

func (s *AuthService) GetSessionState() serverclient.SessionState {
	if s.serverAuth == nil {
		return serverclient.SessionState{}
	}
	return s.serverAuth.SessionState()
}

func (s *AuthService) NeedsInitialSetup() (bool, error) {
	if s.initialSetup == nil {
		return false, errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(15 * time.Second)
	defer cancel()
	return s.initialSetup.NeedsInitialSetup(ctx)
}

func (s *AuthService) InitialSetup(password string) error {
	if s.initialSetup == nil {
		return errServerAuthNotConfigured
	}

	ctx, cancel := s.operationContext(30 * time.Second)
	defer cancel()
	return s.initialSetup.InitialSetup(ctx, password)
}
