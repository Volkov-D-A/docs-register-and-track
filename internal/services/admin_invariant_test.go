package services

import (
	"fmt"
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestActiveAdministratorInvariantConflict(t *testing.T) {
	t.Run("maps the database invariant error to conflict", func(t *testing.T) {
		err := fmt.Errorf("commit failed: %w", &pq.Error{Code: "P0001", Message: activeAdministratorInvariantMessage})

		mapped := activeAdministratorInvariantConflict(err)

		requireAppError(t, mapped, "CONFLICT", 409, "последнего активного администратора")
	})

	t.Run("preserves unrelated errors", func(t *testing.T) {
		err := models.NewForbidden("нет доступа")
		assert.Same(t, err, activeAdministratorInvariantConflict(err))
	})
}
