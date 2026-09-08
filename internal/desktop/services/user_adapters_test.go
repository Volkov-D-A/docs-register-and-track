package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
)

// This client records calls only; authorization and validation belong to server tests.
type userAdapterClient struct {
	method   string
	input    any
	err      error
	deadline time.Time
}

func (c *userAdapterClient) record(ctx context.Context, method string, input any) error {
	c.method = method
	c.input = input
	c.deadline, _ = ctx.Deadline()
	return c.err
}
func (c *userAdapterClient) ListUsers(ctx context.Context) ([]dto.User, error) {
	return []dto.User{{ID: "server-user"}}, c.record(ctx, "users", nil)
}
func (c *userAdapterClient) ListExecutors(ctx context.Context) ([]dto.User, error) {
	return []dto.User{{ID: "server-user"}}, c.record(ctx, "executors", nil)
}
func (c *userAdapterClient) ListSubstitutionCandidates(ctx context.Context) ([]dto.User, error) {
	return []dto.User{{ID: "server-user"}}, c.record(ctx, "candidates", nil)
}
func (c *userAdapterClient) CreateUser(ctx context.Context, req models.CreateUserRequest) (*dto.User, error) {
	return &dto.User{ID: "server-user", TemporaryPassword: "server-password"}, c.record(ctx, "create", req)
}
func (c *userAdapterClient) UpdateUser(ctx context.Context, req models.UpdateUserRequest) (*dto.User, error) {
	return &dto.User{ID: "server-user"}, c.record(ctx, "update", req)
}
func (c *userAdapterClient) ResetUserPassword(ctx context.Context, id string) (string, error) {
	return "server-password", c.record(ctx, "reset", id)
}
func (c *userAdapterClient) GetUserAccessProfile(ctx context.Context, id string) (*models.UserDocumentAccessProfile, error) {
	return &models.UserDocumentAccessProfile{}, c.record(ctx, "access", id)
}
func (c *userAdapterClient) UpdateUserAccessProfile(ctx context.Context, req models.UpdateUserDocumentAccessRequest) error {
	return c.record(ctx, "update-access", req)
}
func (c *userAdapterClient) GetMySubstitution(ctx context.Context) (*dto.UserSubstitution, error) {
	return &dto.UserSubstitution{}, c.record(ctx, "my-substitution", nil)
}
func (c *userAdapterClient) UpdateMySubstitution(ctx context.Context, req models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	return &dto.UserSubstitution{}, c.record(ctx, "update-my-substitution", req)
}
func (c *userAdapterClient) GetUserSubstitution(ctx context.Context, id string) (*dto.UserSubstitution, error) {
	return &dto.UserSubstitution{}, c.record(ctx, "substitution", id)
}
func (c *userAdapterClient) UpdateUserSubstitution(ctx context.Context, req models.UpdateUserSubstitutionRequest) (*dto.UserSubstitution, error) {
	return &dto.UserSubstitution{}, c.record(ctx, "update-substitution", req)
}
func (c *userAdapterClient) GetCurrentAccessSummary(ctx context.Context) (*dto.CurrentAccessSummary, error) {
	return &dto.CurrentAccessSummary{}, c.record(ctx, "summary", nil)
}

func TestUserAdaptersDelegateWithoutLocalBusinessRules(t *testing.T) {
	// Deliberately incomplete requests must reach the server unchanged.
	create := models.CreateUserRequest{Login: "new-user"}
	update := models.UpdateUserRequest{ID: "target", Login: "renamed"}
	access := models.UpdateUserDocumentAccessRequest{UserID: "target"}
	substitution := models.UpdateUserSubstitutionRequest{PrincipalUserID: "target", SubstituteUserID: "substitute", IsActive: true}
	for _, tc := range []struct {
		name        string
		input, want any
		call        func(*userAdapterClient) (any, error)
	}{
		{"users", nil, []dto.User{{ID: "server-user"}}, func(c *userAdapterClient) (any, error) { return NewUserService(c).GetAllUsers() }},
		{"executors", nil, []dto.User{{ID: "server-user"}}, func(c *userAdapterClient) (any, error) { return NewUserService(c).GetExecutors() }},
		{"candidates", nil, []dto.User{{ID: "server-user"}}, func(c *userAdapterClient) (any, error) { return NewUserService(c).GetSubstitutionCandidates() }},
		{"create", create, &dto.User{ID: "server-user", TemporaryPassword: "server-password"}, func(c *userAdapterClient) (any, error) { return NewUserService(c).CreateUser(create) }},
		{"update", update, &dto.User{ID: "server-user"}, func(c *userAdapterClient) (any, error) { return NewUserService(c).UpdateUser(update) }},
		{"reset", "target", "server-password", func(c *userAdapterClient) (any, error) { return NewUserService(c).ResetPassword("target") }},
		{"access", "target", &models.UserDocumentAccessProfile{}, func(c *userAdapterClient) (any, error) {
			return NewDocumentAccessAdminService(c).GetUserAccessProfile("target")
		}},
		{"update-access", access, nil, func(c *userAdapterClient) (any, error) {
			return nil, NewDocumentAccessAdminService(c).UpdateUserAccessProfile(access)
		}},
		{"my-substitution", nil, &dto.UserSubstitution{}, func(c *userAdapterClient) (any, error) { return NewUserSubstitutionService(c).GetMySubstitution() }},
		{"update-my-substitution", substitution, &dto.UserSubstitution{}, func(c *userAdapterClient) (any, error) {
			return NewUserSubstitutionService(c).UpdateMySubstitution(substitution)
		}},
		{"substitution", "target", &dto.UserSubstitution{}, func(c *userAdapterClient) (any, error) {
			return NewUserSubstitutionService(c).GetUserSubstitution("target")
		}},
		{"update-substitution", substitution, &dto.UserSubstitution{}, func(c *userAdapterClient) (any, error) {
			return NewUserSubstitutionService(c).UpdateUserSubstitution(substitution)
		}},
		{"summary", nil, &dto.CurrentAccessSummary{}, func(c *userAdapterClient) (any, error) { return NewDocumentKindService(c).GetCurrentAccessSummary() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, serverErr := range []error{nil, models.ErrForbidden, models.ErrUnauthorized, errors.New("server unavailable")} {
				client := &userAdapterClient{err: serverErr}
				got, err := tc.call(client)
				if serverErr == nil {
					require.NoError(t, err)
					require.Equal(t, tc.want, got)
				} else {
					require.ErrorIs(t, err, serverErr)
				}
				require.Equal(t, tc.name, client.method)
				require.Equal(t, tc.input, client.input)
				require.WithinDuration(t, time.Now().Add(15*time.Second), client.deadline, time.Second)
			}
		})
	}
}
