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

type testNomenclatureClient struct {
	items              []dto.Nomenclature
	item               *dto.Nomenclature
	err                error
	method             string
	id                 string
	name               string
	index              string
	year               int
	kindCode           string
	separator          string
	numberingMode      string
	startNumber        int
	isActive           bool
	contextHasDeadline bool
}

func (c *testNomenclatureClient) capture(ctx context.Context, method string) {
	c.method = method
	_, c.contextHasDeadline = ctx.Deadline()
}
func (c *testNomenclatureClient) ListNomenclature(ctx context.Context, year int, kindCode string) ([]dto.Nomenclature, error) {
	c.capture(ctx, "list")
	c.year, c.kindCode = year, kindCode
	return c.items, c.err
}
func (c *testNomenclatureClient) ListActiveNomenclature(ctx context.Context, kindCode string) ([]dto.Nomenclature, error) {
	c.capture(ctx, "active")
	c.kindCode = kindCode
	return c.items, c.err
}
func (c *testNomenclatureClient) CreateNomenclature(ctx context.Context, name, index string, year int, kindCode, separator, numberingMode string, startNumber int) (*dto.Nomenclature, error) {
	c.capture(ctx, "create")
	c.name, c.index, c.year, c.kindCode = name, index, year, kindCode
	c.separator, c.numberingMode, c.startNumber = separator, numberingMode, startNumber
	return c.item, c.err
}
func (c *testNomenclatureClient) UpdateNomenclature(ctx context.Context, id, name, index string, year int, kindCode, separator, numberingMode string, isActive bool) (*dto.Nomenclature, error) {
	c.capture(ctx, "update")
	c.id, c.name, c.index, c.year, c.kindCode = id, name, index, year, kindCode
	c.separator, c.numberingMode, c.isActive = separator, numberingMode, isActive
	return c.item, c.err
}
func (c *testNomenclatureClient) DeleteNomenclature(ctx context.Context, id string) error {
	c.capture(ctx, "delete")
	c.id = id
	return c.err
}

func TestNomenclatureServiceDelegatesToServer(t *testing.T) {
	id := uuid.NewString()
	item := &dto.Nomenclature{ID: id, Name: "Incoming", Index: "01-01", Year: 2026, KindCode: "incoming_letter"}
	client := &testNomenclatureClient{items: []dto.Nomenclature{*item}, item: item}
	service := NewNomenclatureService()
	service.SetServerClient(client)

	items, err := service.GetAll(2026, "incoming_letter")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "list", client.method)
	assert.Equal(t, 2026, client.year)

	items, err = service.GetActiveForKind("incoming_letter")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "active", client.method)

	created, err := service.Create("Incoming", "01-01", 2026, "incoming_letter", "/", "index_and_number", 3)
	require.NoError(t, err)
	assert.Equal(t, item, created)
	assert.Equal(t, "create", client.method)
	assert.Equal(t, 3, client.startNumber)

	updated, err := service.Update(id, "Incoming updated", "01-02", 2026, "incoming_letter", "-", "number_only", false)
	require.NoError(t, err)
	assert.Equal(t, item, updated)
	assert.Equal(t, "update", client.method)
	assert.False(t, client.isActive)

	require.NoError(t, service.Delete(id))
	assert.Equal(t, "delete", client.method)
	assert.Equal(t, id, client.id)
	assert.True(t, client.contextHasDeadline)
}

func TestNomenclatureServicePropagatesServerError(t *testing.T) {
	want := errors.New("server failed")
	service := NewNomenclatureService()
	service.SetServerClient(&testNomenclatureClient{err: want})

	items, err := service.GetAll(2026, "incoming_letter")
	assert.Nil(t, items)
	require.ErrorIs(t, err, want)
	require.ErrorIs(t, service.Delete(uuid.NewString()), want)
}

func TestNomenclatureServiceRequiresServerClient(t *testing.T) {
	service := NewNomenclatureService()

	items, err := service.GetAll(2026, "incoming_letter")
	assert.Nil(t, items)
	require.ErrorIs(t, err, errServerNomenclatureClientNotConfigured)
	created, err := service.Create("Incoming", "01-01", 2026, "incoming_letter", "/", "index_and_number", 1)
	assert.Nil(t, created)
	require.ErrorIs(t, err, errServerNomenclatureClientNotConfigured)
	require.ErrorIs(t, service.Delete(uuid.NewString()), errServerNomenclatureClientNotConfigured)
}
