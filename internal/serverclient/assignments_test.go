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

func TestAssignmentClientUsesTypedEndpointsAndStripsAccessScope(t *testing.T) {
	id := uuid.NewString()
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/assignments", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, []any{"co"}, body["coExecutorIds"])
			return response(http.StatusCreated, `{"id":"`+id+`"}`), nil
		case 2:
			assert.Equal(t, "/api/v1/assignments/query", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.NotContains(t, body, "allowedDocumentKinds")
			assert.NotContains(t, body, "accessibleByUserId")
			return response(http.StatusOK, `{"items":[],"totalCount":0,"page":1,"pageSize":20}`), nil
		case 3:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/assignments/"+id+"/status", r.URL.Path)
			return response(http.StatusOK, `{"id":"`+id+`"}`), nil
		case 4:
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/assignment-series", r.URL.Path)
			return response(http.StatusCreated, `{"id":"`+id+`"}`), nil
		case 5:
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/api/v1/assignment-series/"+id+"/history", r.URL.Path)
			return response(http.StatusOK, `[]`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.CreateAssignment(context.Background(), "doc", "executor", "work", "", []string{"co"})
	require.NoError(t, err)
	_, err = client.ListAssignments(context.Background(), models.AssignmentFilter{AllowedDocumentKinds: []string{"incoming_letter"}, AccessibleByUserID: uuid.NewString()})
	require.NoError(t, err)
	_, err = client.UpdateAssignmentStatus(context.Background(), id, "completed", "done")
	require.NoError(t, err)
	_, err = client.CreateAssignmentSeries(context.Background(), models.AssignmentSeriesRequest{DocumentID: "doc"})
	require.NoError(t, err)
	history, err := client.GetAssignmentSeriesHistory(context.Background(), id)
	require.NoError(t, err)
	assert.Empty(t, history)
	assert.Equal(t, 5, requestNumber)
}
