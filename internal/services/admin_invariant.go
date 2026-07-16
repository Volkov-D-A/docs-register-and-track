package services

import (
	"errors"

	"github.com/lib/pq"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

const activeAdministratorInvariantMessage = "at least one active administrator must remain"

func activeAdministratorInvariantConflict(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "P0001" || pqErr.Message != activeAdministratorInvariantMessage {
		return err
	}
	return models.NewConflict("нельзя деактивировать или лишить права последнего активного администратора")
}
