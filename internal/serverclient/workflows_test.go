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

func TestWorkflowClientUsesTypedAuthenticatedEndpoints(t *testing.T) {
	id, requestNumber := uuid.NewString(), 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/acknowledgments", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "doc", body["documentId"])
			return response(http.StatusCreated, `{"id":"`+id+`"}`), nil
		case 2:
			assert.Equal(t, "/api/v1/acknowledgments", r.URL.Path)
			assert.Equal(t, "doc", r.URL.Query().Get("documentId"))
			return response(http.StatusOK, `[]`), nil
		case 3:
			assert.Equal(t, "/api/v1/acknowledgments/"+id+"/confirm", r.URL.Path)
			return response(http.StatusNoContent, ``), nil
		case 4:
			assert.Equal(t, "/api/v1/user-events/query", r.URL.Path)
			return response(http.StatusOK, `{"items":[],"totalCount":0,"page":1,"pageSize":20}`), nil
		case 5:
			assert.Equal(t, "/api/v1/user-events/unread-count", r.URL.Path)
			return response(http.StatusOK, `{"count":4}`), nil
		case 6:
			assert.Equal(t, "/api/v1/user-events/"+id+"/read", r.URL.Path)
			return response(http.StatusNoContent, ``), nil
		case 7:
			assert.Equal(t, "/api/v1/administrative-order-acknowledgments/"+id+"/confirm", r.URL.Path)
			return response(http.StatusOK, `{"id":"`+id+`"}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.CreateAcknowledgment(context.Background(), "doc", "read", []string{"user"})
	require.NoError(t, err)
	_, err = client.ListAcknowledgments(context.Background(), "doc")
	require.NoError(t, err)
	require.NoError(t, client.MarkAcknowledgmentConfirmed(context.Background(), id))
	_, err = client.ListUserEvents(context.Background(), models.UserEventFilter{Page: 1, PageSize: 20})
	require.NoError(t, err)
	count, err := client.GetUnreadUserEventCount(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 4, count)
	require.NoError(t, client.MarkUserEventRead(context.Background(), id))
	person, err := client.MarkAdministrativeOrderAcknowledged(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, person.ID)
	assert.Equal(t, 7, requestNumber)
}
