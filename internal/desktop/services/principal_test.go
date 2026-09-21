package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrincipalPermissionUsesOneCurrentUserResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		user      *dto.User
		serverErr error
		want      error
	}{
		{name: "admin", user: &dto.User{ID: uuid.NewString(), IsActive: true, SystemPermissions: []string{models.SystemPermissionAdmin}}},
		{name: "no permission", user: &dto.User{ID: uuid.NewString(), IsActive: true}, want: models.ErrForbidden},
		{name: "inactive admin", user: &dto.User{ID: uuid.NewString(), SystemPermissions: []string{models.SystemPermissionAdmin}}, want: models.ErrUnauthorized},
		{name: "missing user", want: models.ErrUnauthorized},
		{name: "invalid ID", user: &dto.User{ID: "invalid", IsActive: true}, want: models.ErrUnauthorized},
		{name: "server error", serverErr: assert.AnError, want: assert.AnError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakeServerAuthClient{user: tc.user, meErr: tc.serverErr}
			principal := NewPrincipal(NewAuthService(client, nil, nil, nil))
			err := principal.RequireSystemPermission(models.SystemPermissionAdmin)
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.want)
			}
			require.Equal(t, 1, client.meCalls)
		})
	}
}
