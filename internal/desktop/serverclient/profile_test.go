package serverclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestProfileClientUsesBearerProtectedTypedEndpoints(t *testing.T) {
	requestNumber := 0
	client := userClientWithToken(t, func(r *http.Request) (*http.Response, error) {
		requestNumber++
		assert.Equal(t, "Bearer session-token", r.Header.Get("Authorization"))
		switch requestNumber {
		case 1:
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "/api/v1/profile", r.URL.Path)
			return response(http.StatusOK, `{"id":"profile-user","login":"new","lastName":"New","firstName":"User","noPatronymic":true,"fullName":"New User"}`), nil
		default:
			t.Fatalf("unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	updatedUser, err := client.UpdateProfile(context.Background(), models.UpdateProfileRequest{Login: "new", LastName: "New", FirstName: "User", NoPatronymic: true})
	require.NoError(t, err)
	assert.Equal(t, "New User", updatedUser.FullName)
}
