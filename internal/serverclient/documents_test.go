package serverclient

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestDocumentQueryClientUsesTypedEndpoints(t *testing.T) {
	id := uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/documents/"+id, r.URL.Path)
			return response(http.StatusOK, `{"id":"`+id+`","kindCode":"incoming_letter"}`), nil
		case 2:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/documents/query", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "incoming_letter", body["kindCode"])
			filter := body["filter"].(map[string]any)
			assert.Equal(t, "needle", filter["search"])
			assert.NotContains(t, filter, "accessScope")
			return response(http.StatusOK, `{"items":[],"totalCount":0,"page":1,"pageSize":20,"hasMore":false}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	card, err := client.GetDocumentCard(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, card.ID)
	result, err := client.ListDocuments(context.Background(), "incoming_letter", models.DocumentFilter{
		Search: "needle", Page: 1, PageSize: 20,
		AccessScope: &models.DocumentAccessScope{Restricted: false},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Items)
}

func TestDocumentQueryClientValidatesInputBeforeRequest(t *testing.T) {
	requests := 0
	client := userClientWithToken(t, func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusInternalServerError, ""), nil
	})

	card, err := client.GetDocumentCard(context.Background(), "invalid")
	assert.Nil(t, card)
	require.Error(t, err)
	result, err := client.ListDocuments(context.Background(), " ", models.DocumentFilter{})
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Zero(t, requests)
}

func TestDocumentCommandClientUsesIdempotencyHeaders(t *testing.T) {
	key := uuid.NewString()
	documentID := uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.NotEmpty(t, r.Header.Get("Idempotency-Key"))
		switch requestNumber {
		case 1:
			assert.Equal(t, key, r.Header.Get("Idempotency-Key"))
			assert.Equal(t, "/api/v1/documents/incoming_letter", r.URL.Path)
			return response(http.StatusCreated, `{"id":"`+documentID+`"}`), nil
		case 2:
			assert.Equal(t, "/api/v1/documents/incoming_letter/"+documentID, r.URL.Path)
			return response(http.StatusOK, `{"id":"`+documentID+`"}`), nil
		case 3:
			assert.Equal(t, "/api/v1/documents/incoming_letter/admin-drafts", r.URL.Path)
			return response(http.StatusCreated, `{"id":"`+documentID+`"}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.RegisterDocument(context.Background(), "incoming_letter", map[string]any{"idempotencyKey": key})
	require.NoError(t, err)
	_, err = client.UpdateDocument(context.Background(), "incoming_letter", map[string]any{"id": documentID})
	require.NoError(t, err)
	_, err = client.CreateAdminDocumentDraft(context.Background(), "incoming_letter", map[string]any{"nomenclatureId": uuid.NewString()})
	require.NoError(t, err)
	assert.Equal(t, 3, requestNumber)
}
