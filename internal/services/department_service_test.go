package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

type testDepartmentClient struct {
	items              []dto.Department
	result             *dto.Department
	err                error
	method             string
	id                 string
	name               string
	nomenclatureIDs    []string
	contextHasDeadline bool
}

func (c *testDepartmentClient) capture(ctx context.Context, method, id, name string, nomenclatureIDs []string) {
	c.method = method
	c.id = id
	c.name = name
	c.nomenclatureIDs = append([]string(nil), nomenclatureIDs...)
	_, c.contextHasDeadline = ctx.Deadline()
}

func (c *testDepartmentClient) ListDepartments(ctx context.Context) ([]dto.Department, error) {
	c.capture(ctx, "list", "", "", nil)
	return c.items, c.err
}

func (c *testDepartmentClient) CreateDepartment(ctx context.Context, name string, nomenclatureIDs []string) (*dto.Department, error) {
	c.capture(ctx, "create", "", name, nomenclatureIDs)
	return c.result, c.err
}

func (c *testDepartmentClient) UpdateDepartment(ctx context.Context, id, name string, nomenclatureIDs []string) (*dto.Department, error) {
	c.capture(ctx, "update", id, name, nomenclatureIDs)
	return c.result, c.err
}

func (c *testDepartmentClient) DeleteDepartment(ctx context.Context, id string) error {
	c.capture(ctx, "delete", id, "", nil)
	return c.err
}

func TestDepartmentServiceDelegatesCRUDToServer(t *testing.T) {
	id := uuid.NewString()
	nomenclatureIDs := []string{uuid.NewString(), uuid.NewString()}
	result := &dto.Department{ID: id, Name: "Legal", NomenclatureIDs: nomenclatureIDs}

	tests := []struct {
		name   string
		method string
		call   func(*DepartmentService) error
	}{
		{
			name:   "list",
			method: "list",
			call: func(service *DepartmentService) error {
				_, err := service.GetAllDepartments()
				return err
			},
		},
		{
			name:   "create",
			method: "create",
			call: func(service *DepartmentService) error {
				_, err := service.CreateDepartment("Legal", nomenclatureIDs)
				return err
			},
		},
		{
			name:   "update",
			method: "update",
			call: func(service *DepartmentService) error {
				_, err := service.UpdateDepartment(id, "Legal", nomenclatureIDs)
				return err
			},
		},
		{
			name:   "delete",
			method: "delete",
			call: func(service *DepartmentService) error {
				return service.DeleteDepartment(id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &testDepartmentClient{items: []dto.Department{*result}, result: result}
			service := NewDepartmentService()
			service.SetServerClient(client)

			require.NoError(t, tt.call(service))
			assert.Equal(t, tt.method, client.method)
			assert.True(t, client.contextHasDeadline)
		})
	}
}

func TestDepartmentServicePropagatesServerError(t *testing.T) {
	want := errors.New("server failed")
	client := &testDepartmentClient{err: want}
	service := NewDepartmentService()
	service.SetServerClient(client)

	result, err := service.CreateDepartment("Legal", nil)

	assert.Nil(t, result)
	require.ErrorIs(t, err, want)
}

func TestDepartmentServiceRequiresServerClient(t *testing.T) {
	service := NewDepartmentService()

	items, err := service.GetAllDepartments()
	assert.Nil(t, items)
	require.ErrorIs(t, err, errServerDepartmentClientNotConfigured)

	created, err := service.CreateDepartment("Legal", nil)
	assert.Nil(t, created)
	require.ErrorIs(t, err, errServerDepartmentClientNotConfigured)

	updated, err := service.UpdateDepartment(uuid.NewString(), "Legal", nil)
	assert.Nil(t, updated)
	require.ErrorIs(t, err, errServerDepartmentClientNotConfigured)

	require.ErrorIs(t, service.DeleteDepartment(uuid.NewString()), errServerDepartmentClientNotConfigured)
}
