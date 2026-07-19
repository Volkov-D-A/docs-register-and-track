package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/repository"
)

func newOutboxAdminServiceForTest(t *testing.T, roles ...string) (*OutboxAdminService, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	userID := uuid.New()
	userStore := mocks.NewUserStore(t)
	userStore.On("GetByID", userID).Return(&models.User{ID: userID, IsActive: true}, nil).Maybe()
	auth := NewAuthService(nil, userStore)
	auth.currentUserID = userID
	auth.SetAccessStore(newRoleMappedDocumentAccessStore(roles...))
	service := NewOutboxAdminService(repository.NewOutboxRepository(&database.DB{DB: db}), auth)
	return service, mock, func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	}
}

func TestOutboxAdminServiceRequiresAdministratorPermission(t *testing.T) {
	service, mock, closeDB := newOutboxAdminServiceForTest(t, "clerk")
	defer closeDB()

	_, err := service.GetStats()
	require.ErrorIs(t, err, models.ErrForbidden)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxAdminServiceRequeueRejectsInvalidID(t *testing.T) {
	service, mock, closeDB := newOutboxAdminServiceForTest(t, models.SystemPermissionAdmin)
	defer closeDB()

	err := service.Requeue("not-a-uuid")
	appErr, ok := models.AsAppError(err)
	require.True(t, ok)
	require.Equal(t, "VALIDATION_ERROR", appErr.Kind)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxAdminServiceRequeuesTerminalEvent(t *testing.T) {
	service, mock, closeDB := newOutboxAdminServiceForTest(t, models.SystemPermissionAdmin)
	defer closeDB()
	eventID := uuid.New()
	mock.ExpectExec(`UPDATE event_outbox.*failed_at = NULL`).WithArgs(eventID).WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, service.Requeue(eventID.String()))
	require.NoError(t, mock.ExpectationsWereMet())
}
