package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type testReferenceClient struct {
	organizations      []dto.Organization
	organization       *dto.Organization
	executors          []dto.ResolutionExecutor
	executor           *dto.ResolutionExecutor
	err                error
	method             string
	query              string
	id                 string
	targetID           string
	name               string
	contextHasDeadline bool
}

func (c *testReferenceClient) capture(ctx context.Context, method string) {
	c.method = method
	_, c.contextHasDeadline = ctx.Deadline()
}

func (c *testReferenceClient) ListOrganizations(ctx context.Context, query string) ([]dto.Organization, error) {
	c.capture(ctx, "list-organizations")
	c.query = query
	return c.organizations, c.err
}
func (c *testReferenceClient) ResolveOrganization(ctx context.Context, name string) (*dto.Organization, error) {
	c.capture(ctx, "resolve-organization")
	c.name = name
	return c.organization, c.err
}
func (c *testReferenceClient) UpdateOrganization(ctx context.Context, id, name string) error {
	c.capture(ctx, "update-organization")
	c.id, c.name = id, name
	return c.err
}
func (c *testReferenceClient) DeleteOrganization(ctx context.Context, id string) error {
	c.capture(ctx, "delete-organization")
	c.id = id
	return c.err
}
func (c *testReferenceClient) MergeOrganizations(ctx context.Context, sourceID, targetID string) error {
	c.capture(ctx, "merge-organizations")
	c.id, c.targetID = sourceID, targetID
	return c.err
}
func (c *testReferenceClient) ListResolutionExecutors(ctx context.Context, query string) ([]dto.ResolutionExecutor, error) {
	c.capture(ctx, "list-executors")
	c.query = query
	return c.executors, c.err
}
func (c *testReferenceClient) ResolveResolutionExecutor(ctx context.Context, name string) (*dto.ResolutionExecutor, error) {
	c.capture(ctx, "resolve-executor")
	c.name = name
	return c.executor, c.err
}
func (c *testReferenceClient) UpdateResolutionExecutor(ctx context.Context, id, name string) error {
	c.capture(ctx, "update-executor")
	c.id, c.name = id, name
	return c.err
}
func (c *testReferenceClient) DeleteResolutionExecutor(ctx context.Context, id string) error {
	c.capture(ctx, "delete-executor")
	c.id = id
	return c.err
}

func TestReferenceServiceDelegatesDirectoriesToServer(t *testing.T) {
	id, targetID := uuid.NewString(), uuid.NewString()
	client := &testReferenceClient{
		organizations: []dto.Organization{{ID: id, Name: "Legal"}},
		organization:  &dto.Organization{ID: id, Name: "Legal"},
		executors:     []dto.ResolutionExecutor{{ID: id, Name: "Executor"}},
		executor:      &dto.ResolutionExecutor{ID: id, Name: "Executor"},
	}
	service := NewReferenceService(nil)
	service.SetServerClient(client)

	organizations, err := service.GetOrganizations()
	require.NoError(t, err)
	require.Len(t, organizations, 1)
	assert.Equal(t, "list-organizations", client.method)
	assert.Empty(t, client.query)

	_, err = service.SearchOrganizations("Leg")
	require.NoError(t, err)
	assert.Equal(t, "Leg", client.query)
	_, err = service.FindOrCreateOrganization("Legal")
	require.NoError(t, err)
	assert.Equal(t, "resolve-organization", client.method)
	require.NoError(t, service.UpdateOrganization(id, "Compliance"))
	assert.Equal(t, "update-organization", client.method)
	require.NoError(t, service.DeleteOrganization(id))
	assert.Equal(t, "delete-organization", client.method)
	require.NoError(t, service.MergeOrganizations(id, targetID))
	assert.Equal(t, "merge-organizations", client.method)
	assert.Equal(t, targetID, client.targetID)

	executors, err := service.GetResolutionExecutors()
	require.NoError(t, err)
	require.Len(t, executors, 1)
	assert.Equal(t, "list-executors", client.method)
	_, err = service.SearchResolutionExecutors("Exec")
	require.NoError(t, err)
	assert.Equal(t, "Exec", client.query)
	_, err = service.FindOrCreateResolutionExecutor("Executor")
	require.NoError(t, err)
	assert.Equal(t, "resolve-executor", client.method)
	require.NoError(t, service.UpdateResolutionExecutor(id, "Chief"))
	assert.Equal(t, "update-executor", client.method)
	require.NoError(t, service.DeleteResolutionExecutor(id))
	assert.Equal(t, "delete-executor", client.method)
	assert.True(t, client.contextHasDeadline)
}

func TestReferenceServicePropagatesServerError(t *testing.T) {
	want := errors.New("server failed")
	service := NewReferenceService(nil)
	service.SetServerClient(&testReferenceClient{err: want})

	items, err := service.GetOrganizations()
	assert.Nil(t, items)
	require.ErrorIs(t, err, want)
	require.ErrorIs(t, service.DeleteResolutionExecutor(uuid.NewString()), want)
}

func TestReferenceServiceRequiresServerClient(t *testing.T) {
	service := NewReferenceService(nil)

	items, err := service.GetOrganizations()
	assert.Nil(t, items)
	require.ErrorIs(t, err, errServerReferenceClientNotConfigured)
	require.ErrorIs(t, service.UpdateOrganization(uuid.NewString(), "Legal"), errServerReferenceClientNotConfigured)

	executors, err := service.GetResolutionExecutors()
	assert.Nil(t, executors)
	require.ErrorIs(t, err, errServerReferenceClientNotConfigured)
}

func TestReferenceServiceKeepsDocumentTypesLocalAndReadOnly(t *testing.T) {
	userRepo := mocks.NewUserStore(t)
	user := &models.User{ID: uuid.New(), IsActive: true}
	userRepo.On("GetByID", user.ID).Return(user, nil).Once()
	auth := NewAuthService(nil, userRepo)
	auth.currentUserID = user.ID
	service := NewReferenceService(auth)

	items, err := service.GetDocumentTypes()
	require.NoError(t, err)
	require.Len(t, items, len(models.AllowedDocumentTypes()))
	assert.Equal(t, models.DocumentTypeLetter, items[0].ID)

	created, err := service.CreateDocumentType("Custom")
	assert.Nil(t, created)
	require.Error(t, err)
	require.Error(t, service.UpdateDocumentType("Письмо", "Custom"))
	require.Error(t, service.DeleteDocumentType("Письмо"))
}
