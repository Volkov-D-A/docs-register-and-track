package serverclient

import (
	"context"
	"encoding/json"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestDeadlineClientUsesVersionedCommands(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/deadline-controls/control-id/complete", r.URL.Path)
		var request map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, float64(4), request["version"])
		require.Equal(t, "2026-10-10", request["executedOn"])
		require.NotContains(t, request, "completedBy")
		return response(http.StatusOK, `{"id":"control-id","status":"completed","executedOn":"2026-10-10","version":5}`), nil
	})
	result, err := client.CompleteDeadlineControl(context.Background(), "control-id", models.CompleteDeadlineRequest{Version: 4, ExecutedOn: "2026-10-10"})
	require.NoError(t, err)
	require.Equal(t, int64(5), result.Version)
	require.Equal(t, models.DeadlineCompleted, result.Status)
}
func TestDeadlineClientPreservesConflict(t *testing.T) {
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		return response(http.StatusConflict, `{"code":"CONFLICT","message":"Перечитайте данные"}`), nil
	})
	_, err := client.ResumeDeadlineSeries(context.Background(), "series-id", models.DeadlineSeriesActionRequest{Version: 1, ExpectedNextDueDate: "2026-10-15"})
	require.Error(t, err)
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	require.Equal(t, http.StatusConflict, appErr.Code)
}
