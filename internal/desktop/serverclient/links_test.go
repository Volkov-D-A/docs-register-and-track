package serverclient

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkClientUsesTypedAuthenticatedEndpoints(t *testing.T) {
	id, requestNumber := uuid.NewString(), 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/document-links", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "source", body["sourceId"])
			assert.Equal(t, "target", body["targetId"])
			assert.NotContains(t, body, "userId")
			return response(http.StatusCreated, `{"id":"`+id+`"}`), nil
		case 2:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/documents/doc/links", r.URL.Path)
			return response(http.StatusOK, `[]`), nil
		case 3:
			assert.Equal(t, "/api/v1/documents/doc/link-graph", r.URL.Path)
			return response(http.StatusOK, `{"nodes":[],"edges":[]}`), nil
		case 4:
			assert.Equal(t, "/api/v1/documents/doc/journal", r.URL.Path)
			return response(http.StatusOK, `[]`), nil
		case 5:
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/document-links/"+id, r.URL.Path)
			return response(http.StatusNoContent, ``), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	created, err := client.LinkDocuments(context.Background(), "source", "target", "related")
	require.NoError(t, err)
	assert.Equal(t, id, created.ID)
	_, err = client.GetDocumentLinks(context.Background(), "doc")
	require.NoError(t, err)
	_, err = client.GetDocumentFlow(context.Background(), "doc")
	require.NoError(t, err)
	_, err = client.GetDocumentJournal(context.Background(), "doc")
	require.NoError(t, err)
	require.NoError(t, client.UnlinkDocument(context.Background(), id))
	assert.Equal(t, 5, requestNumber)
}
