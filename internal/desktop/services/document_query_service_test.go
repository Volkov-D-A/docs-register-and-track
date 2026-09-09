package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
)

type fakeServerDocumentQueryClient struct {
	card       *dto.DocumentCard
	list       *dto.PagedResult[dto.DocumentListItem]
	cardErr    error
	listErr    error
	lastID     string
	lastKind   string
	lastFilter models.DocumentFilter
}

func (c *fakeServerDocumentQueryClient) GetDocumentCard(_ context.Context, id string) (*dto.DocumentCard, error) {
	c.lastID = id
	return c.card, c.cardErr
}

func (c *fakeServerDocumentQueryClient) ListDocuments(_ context.Context, kind string, filter models.DocumentFilter) (*dto.PagedResult[dto.DocumentListItem], error) {
	c.lastKind, c.lastFilter = kind, filter
	return c.list, c.listErr
}

func TestDocumentQueryServiceUsesServerClient(t *testing.T) {
	client := &fakeServerDocumentQueryClient{
		card: &dto.DocumentCard{ID: uuid.NewString()},
		list: &dto.PagedResult[dto.DocumentListItem]{Items: []dto.DocumentListItem{{ID: uuid.NewString()}}, TotalCount: 1},
	}
	metrics := observability.NewRegistry(16)
	service := NewDocumentQueryService(client, metrics)

	card, err := service.GetByID(client.card.ID)
	require.NoError(t, err)
	assert.Same(t, client.card, card)
	assert.Equal(t, client.card.ID, client.lastID)

	filter := models.DocumentFilter{Search: "contract", Page: 2, PageSize: 25}
	result, err := service.GetList(string(models.DocumentKindIncomingLetter), filter)
	require.NoError(t, err)
	assert.Same(t, client.list, result)
	assert.Equal(t, string(models.DocumentKindIncomingLetter), client.lastKind)
	assert.Equal(t, filter, client.lastFilter)
	assert.Len(t, metrics.Snapshot(), 2)
	assert.Equal(t, []observability.CounterSnapshot{{Name: "documents.list.items", Value: 1}}, metrics.Counters())
}

func TestDocumentQueryServiceRequiresServerClient(t *testing.T) {
	service := NewDocumentQueryService(nil, nil)
	card, err := service.GetByID(uuid.NewString())
	assert.Nil(t, card)
	require.ErrorIs(t, err, errServerDocumentQueryClientNotConfigured)
	list, err := service.GetList("incoming_letter", models.DocumentFilter{})
	assert.Nil(t, list)
	require.ErrorIs(t, err, errServerDocumentQueryClientNotConfigured)
}

func TestDocumentQueryServicePropagatesServerErrors(t *testing.T) {
	service := NewDocumentQueryService(&fakeServerDocumentQueryClient{cardErr: models.ErrForbidden, listErr: models.ErrUnauthorized}, nil)
	card, err := service.GetByID(uuid.NewString())
	require.ErrorIs(t, err, models.ErrForbidden)
	assert.Nil(t, card)
	list, err := service.GetList(string(models.DocumentKindIncomingLetter), models.DocumentFilter{})
	require.ErrorIs(t, err, models.ErrUnauthorized)
	assert.Nil(t, list)
}
