package serverclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNomenclatureClientUsesTypedEndpoints(t *testing.T) {
	id := uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/nomenclature", r.URL.Path)
			assert.Equal(t, "2026", r.URL.Query().Get("year"))
			assert.Equal(t, "incoming_letter", r.URL.Query().Get("kindCode"))
			return response(http.StatusOK, `[{"id":"`+id+`","name":"Incoming"}]`), nil
		case 2:
			assert.Equal(t, "/api/v1/nomenclature/active", r.URL.Path)
			return response(http.StatusOK, `[{"id":"`+id+`","name":"Incoming"}]`), nil
		case 3:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/nomenclature", r.URL.Path)
			return response(http.StatusCreated, `{"id":"`+id+`","name":"Incoming"}`), nil
		case 4:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/nomenclature/"+id, r.URL.Path)
			return response(http.StatusOK, `{"id":"`+id+`","name":"Updated"}`), nil
		case 5:
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/nomenclature/"+id, r.URL.Path)
			return response(http.StatusNoContent, ""), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	items, err := client.ListNomenclature(context.Background(), 2026, "incoming_letter")
	require.NoError(t, err)
	require.Len(t, items, 1)
	items, err = client.ListActiveNomenclature(context.Background(), "incoming_letter")
	require.NoError(t, err)
	require.Len(t, items, 1)
	created, err := client.CreateNomenclature(context.Background(), "Incoming", "01-01", 2026, "incoming_letter", "/", "index_and_number", 1)
	require.NoError(t, err)
	assert.Equal(t, id, created.ID)
	updated, err := client.UpdateNomenclature(context.Background(), id, "Updated", "01-02", 2026, "incoming_letter", "-", "number_only", false)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	require.NoError(t, client.DeleteNomenclature(context.Background(), id))
}

func TestNomenclatureClientRejectsInvalidIDBeforeRequest(t *testing.T) {
	requests := 0
	client := userClientWithToken(t, func(*http.Request) (*http.Response, error) {
		requests++
		return response(http.StatusInternalServerError, ""), nil
	})

	updated, err := client.UpdateNomenclature(context.Background(), "invalid", "Name", "01", 2026, "incoming_letter", "/", "index_and_number", true)
	assert.Nil(t, updated)
	require.Error(t, err)
	require.Error(t, client.DeleteNomenclature(context.Background(), "invalid"))
	assert.Zero(t, requests)
}
