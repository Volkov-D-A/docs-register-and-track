package services

import (
	"fmt"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"

	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func setupAssignmentService(t *testing.T, role string) (
	*AssignmentService, *mocks.AssignmentStore, *mocks.UserStore, *attachmentPrincipalStub, *mocks.IncomingDocStore,
) {
	return setupAssignmentServiceWithAccess(t, role, newRoleMappedDocumentAccessStore(role), nil)
}

func setupAssignmentServiceWithAccess(t *testing.T, role string, accessStore ports.DocumentAccessStore, substitutions ports.UserSubstitutionStore) (
	*AssignmentService, *mocks.AssignmentStore, *mocks.UserStore, *attachmentPrincipalStub, *mocks.IncomingDocStore,
) {
	t.Helper()
	assignmentRepo := mocks.NewAssignmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := newAttachmentPrincipalStub(userRepo)

	user := &models.User{
		ID:    uuid.New(),
		Login: role + "_user",

		IsDocumentParticipant: role != "" && role != "admin",
		IsActive:              true,
	}
	auth.currentUserID = user.ID
	userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	userRepo.On("GetEligibleRecipientIDs", mock.Anything).Return(func(ids []uuid.UUID) map[uuid.UUID]struct{} {
		result := make(map[uuid.UUID]struct{}, len(ids))
		for _, id := range ids {
			result[id] = struct{}{}
		}
		return result
	}, nil).Maybe()

	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, nil, assignmentRepo, accessStore, &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo}, substitutions)

	svc := NewAssignmentService(assignmentRepo, userRepo, auth, accessSvc, substitutions, false)
	return svc, assignmentRepo, userRepo, auth, incomingRepo
}

func TestAssignmentOutboxKeySeparatesTransitionsAndRecipients(t *testing.T) {
	assignmentID := uuid.New()
	recipientA, recipientB := uuid.New(), uuid.New()
	revision1 := "2026-07-16T10:00:00Z"
	revision2 := "2026-07-16T10:01:00Z"

	created := assignmentOutboxKey(assignmentID, "created", "", &recipientA, "user_event")
	retry := assignmentOutboxKey(assignmentID, "created", "", &recipientA, "user_event")
	otherRecipient := assignmentOutboxKey(assignmentID, "created", "", &recipientB, "user_event")
	updated := assignmentOutboxKey(assignmentID, "updated", revision1, &recipientA, "user_event")
	laterUpdate := assignmentOutboxKey(assignmentID, "updated", revision2, &recipientA, "user_event")
	journal := assignmentOutboxKey(assignmentID, "created", "", nil, "journal")

	assert.Equal(t, created, retry)
	assert.NotEqual(t, created, otherRecipient)
	assert.NotEqual(t, created, updated)
	assert.NotEqual(t, updated, laterUpdate)
	assert.NotEqual(t, created, journal)
}

func setupAssignmentServiceNotAuth(t *testing.T) (*AssignmentService, *mocks.AssignmentStore) {
	t.Helper()
	assignmentRepo := mocks.NewAssignmentStore(t)
	userRepo := mocks.NewUserStore(t)
	auth := newAttachmentPrincipalStub(userRepo)
	incomingRepo := mocks.NewIncomingDocStore(t)
	outgoingRepo := mocks.NewOutgoingDocStore(t)
	incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
		return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
		return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
	}, nil).Maybe()
	assignmentRepo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
	accessSvc := NewDocumentAccessService(auth, nil, assignmentRepo, newRoleMappedDocumentAccessStore(), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})

	svc := NewAssignmentService(assignmentRepo, userRepo, auth, accessSvc, nil, false)
	return svc, assignmentRepo
}

type kindActionAccessStore struct {
	allowed map[string]map[string]bool
}

func newKindActionAccessStore(allowed map[string][]string) ports.DocumentAccessStore {
	store := &kindActionAccessStore{allowed: make(map[string]map[string]bool, len(allowed))}
	for kindCode, actions := range allowed {
		store.allowed[kindCode] = make(map[string]bool, len(actions))
		for _, action := range actions {
			store.allowed[kindCode][action] = true
		}
	}
	return store
}

func (s *kindActionAccessStore) HasPermission(kindCode, action string, departmentID, userID string) (bool, error) {
	return s.allowed[kindCode][action], nil
}

func (s *kindActionAccessStore) HasSystemPermission(permission, userID string) (bool, error) {
	return false, nil
}

func (s *kindActionAccessStore) GetUserAccessProfile(userID string) (*models.UserDocumentAccessProfile, error) {
	return &models.UserDocumentAccessProfile{}, nil
}

// ---------- TestAssignmentService_Create ----------

func TestAssignmentService_Create(t *testing.T) {
	// Создание нового поручения по документу
	docID := uuid.New()
	execID := uuid.New()

	t.Run("success with deadline", func(t *testing.T) {
		svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		expected := &models.Assignment{
			ID:         uuid.New(),
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Выполнить",
			Status:     "new",
		}
		repo.On("CreateWithOutbox", mock.Anything, docID, execID, "Выполнить",
			mock.AnythingOfType("*time.Time"), []string(nil),
			mock.Anything).Return(expected, nil).Once()

		result, err := svc.createExecution(docID.String(), execID.String(), "Выполнить", "2025-12-31", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, expected.ID.String(), result.ID)
	})

	t.Run("success without deadline", func(t *testing.T) {
		svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		expected := &models.Assignment{
			ID:         uuid.New(),
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Выполнить",
			Status:     "new",
		}
		repo.On("CreateWithOutbox", mock.Anything, docID, execID, "Выполнить",
			(*time.Time)(nil), []string(nil),
			mock.Anything).Return(expected, nil).Once()

		result, err := svc.createExecution(docID.String(), execID.String(), "Выполнить", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc, _ := setupAssignmentServiceNotAuth(t)

		result, err := svc.createExecution(docID.String(), execID.String(), "Выполнить", "", nil)
		require.Error(t, err)
		assert.Equal(t, models.ErrUnauthorized, err)
		assert.Nil(t, result)
	})

	t.Run("invalid document ID", func(t *testing.T) {
		svc, _, _, _, _ := setupAssignmentService(t, "clerk")

		result, err := svc.createExecution("not-a-uuid", execID.String(), "Выполнить", "", nil)
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID документа")
		assert.Nil(t, result)
	})

	t.Run("invalid co-executor rejects request", func(t *testing.T) {
		svc, _, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
		result, err := svc.createExecution(docID.String(), execID.String(), "Выполнить", "", []string{uuid.NewString(), "not-a-uuid"})
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID соисполнителя")
		require.Nil(t, result)
	})

	t.Run("duplicate and main executor are removed", func(t *testing.T) {
		coExecID := uuid.New()
		svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
		repo.On("CreateWithOutbox", mock.Anything, docID, execID, "Выполнить", (*time.Time)(nil), []string{coExecID.String()}, mock.Anything).
			Return(&models.Assignment{ID: uuid.New(), DocumentID: docID, ExecutorID: execID, Status: "new"}, nil).Once()
		result, err := svc.createExecution(docID.String(), execID.String(), "Выполнить", "", []string{execID.String(), coExecID.String(), coExecID.String()})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, repo.Effects, 3)
	})

	t.Run("zero executor ID", func(t *testing.T) {
		svc, _, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
		result, err := svc.createExecution(docID.String(), uuid.Nil.String(), "Выполнить", "", nil)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID исполнителя")
		require.Nil(t, result)
	})

	t.Run("invalid executor ID", func(t *testing.T) {
		svc, _, _, _, incomingRepo := setupAssignmentService(t, "clerk")
		incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()

		result, err := svc.createExecution(docID.String(), "not-a-uuid", "Выполнить", "", nil)
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID исполнителя")
		assert.Nil(t, result)
	})
}

func TestAssignmentServiceCreatePassesJournalAndUserEffectsToAtomicStore(t *testing.T) {
	docID, executorID := uuid.New(), uuid.New()
	svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
	incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{ID: docID, NomenclatureID: uuid.New()}, nil).Maybe()
	atomicRepo := repo
	svc.repo = atomicRepo
	repo.On("CreateWithOutbox", mock.Anything, docID, executorID, "Выполнить", (*time.Time)(nil), []string(nil), mock.Anything).Return(&models.Assignment{ID: uuid.New(), DocumentID: docID, ExecutorID: executorID, Status: "new"}, nil).Once()

	_, err := svc.createExecution(docID.String(), executorID.String(), "Выполнить", "", nil)
	require.NoError(t, err)
	require.Len(t, atomicRepo.Effects, 2)
	assert.Equal(t, models.OutboxEventJournal, atomicRepo.Effects[0].EventType)
	assert.Equal(t, models.OutboxEventUserEvent, atomicRepo.Effects[1].EventType)
}

func TestAssignmentServiceUpdatePassesJournalAndUserEffectsToAtomicStore(t *testing.T) {
	docID, assignmentID, executorID := uuid.New(), uuid.New(), uuid.New()
	svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
	atomicRepo := repo
	svc.repo = atomicRepo
	repo.On("GetByID", assignmentID).Return(&models.Assignment{ID: assignmentID, DocumentID: docID, ExecutorID: executorID, Status: "new"}, nil).Once()
	repo.On("UpdateDetailsWithOutbox", assignmentID, executorID, "Исправить", (*time.Time)(nil), []string(nil), mock.Anything, mock.Anything).Return(&models.Assignment{ID: assignmentID, DocumentID: docID, ExecutorID: executorID, Status: "new"}, nil).Once()

	_, err := svc.Update(assignmentID.String(), executorID.String(), "Исправить", "", nil)
	require.NoError(t, err)
	require.Len(t, atomicRepo.Effects, 2)
	assert.Equal(t, models.OutboxEventJournal, atomicRepo.Effects[0].EventType)
	assert.Equal(t, models.OutboxEventUserEvent, atomicRepo.Effects[1].EventType)
}

func TestAssignmentServiceDeletePassesJournalEffectToAtomicStore(t *testing.T) {
	docID, assignmentID, executorID := uuid.New(), uuid.New(), uuid.New()
	svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
	atomicRepo := repo
	svc.repo = atomicRepo
	repo.On("GetByID", assignmentID).Return(&models.Assignment{ID: assignmentID, DocumentID: docID, ExecutorID: executorID, Status: "new"}, nil).Once()
	repo.On("DeleteWithOutbox", assignmentID, mock.Anything).Return(nil).Once()

	require.NoError(t, svc.Delete(assignmentID.String()))
	require.Len(t, atomicRepo.Effects, 1)
	assert.Equal(t, models.OutboxEventJournal, atomicRepo.Effects[0].EventType)
	var journal models.CreateJournalEntryRequest
	require.NoError(t, json.Unmarshal([]byte(atomicRepo.Effects[0].Payload), &journal))
	require.Equal(t, []uuid.UUID{executorID}, journal.PreviousReaderIDs)
}

func TestAssignmentServiceUpdateStatusPassesJournalEffectToAtomicStore(t *testing.T) {
	docID, assignmentID := uuid.New(), uuid.New()
	svc, repo, _, auth, _ := setupAssignmentService(t, "executor")
	executorID, err := auth.GetCurrentUserUUID()
	require.NoError(t, err)
	atomicRepo := repo
	svc.repo = atomicRepo
	repo.On("GetByID", assignmentID).Return(&models.Assignment{ID: assignmentID, DocumentID: docID, ExecutorID: executorID, Status: "new"}, nil).Once()
	repo.On("UpdateWithOutbox", assignmentID, executorID, "", (*time.Time)(nil), "in_progress", "", (*time.Time)(nil), []string(nil), mock.Anything).Return(&models.Assignment{ID: assignmentID, DocumentID: docID, ExecutorID: executorID, Status: "in_progress"}, nil).Once()

	_, err = svc.UpdateStatus(assignmentID.String(), "in_progress", "")
	require.NoError(t, err)
	require.Len(t, atomicRepo.Effects, 1)
	assert.Equal(t, models.OutboxEventJournal, atomicRepo.Effects[0].EventType)
}

func TestAssignmentService_CreateEmitsUserEvents(t *testing.T) {
	docID := uuid.New()
	execID := uuid.New()
	coExecID := uuid.New()

	svc, repo, _, _, incomingRepo := setupAssignmentService(t, "clerk")
	svc = NewAssignmentService(svc.repo, svc.userRepo, svc.auth, svc.access, svc.substitutions, true)
	incomingRepo.On("GetByID", docID).Return(&models.IncomingDocument{
		ID:             docID,
		NomenclatureID: uuid.New(),
		IncomingNumber: "ВХ-1",
	}, nil).Maybe()

	expected := &models.Assignment{
		ID:             uuid.New(),
		DocumentID:     docID,
		DocumentKind:   "incoming_letter",
		DocumentNumber: "ВХ-1",
		ExecutorID:     execID,
		Content:        "Выполнить",
		Status:         "new",
		CoExecutorIDs:  []string{coExecID.String()},
	}
	repo.On("CreateWithOutbox", mock.Anything, docID, execID, "Выполнить",
		(*time.Time)(nil), []string{coExecID.String()},
		mock.Anything).Return(expected, nil).Once()

	result, err := svc.createExecution(docID.String(), execID.String(), "Выполнить", "", []string{coExecID.String()})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, repo.Effects, 3)
	assert.Equal(t, models.OutboxEventJournal, repo.Effects[0].EventType)
	assert.Equal(t, models.OutboxEventUserEvent, repo.Effects[1].EventType)
	assert.Equal(t, models.OutboxEventUserEvent, repo.Effects[2].EventType)
}

// ---------- TestAssignmentService_Update ----------

func TestAssignmentService_Update(t *testing.T) {
	// Обновление параметров существующего поручения (текст, исполнитель, дедлайн)
	assignmentID := uuid.New()
	execID := uuid.New()
	docID := uuid.New()

	existing := &models.Assignment{
		ID:         assignmentID,
		DocumentID: docID,
		ExecutorID: execID,
		Content:    "Старое",
		Status:     "new",
	}

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateDetailsWithOutbox", assignmentID, execID, "Новое",
			(*time.Time)(nil), []string(nil), mock.Anything,
			mock.Anything).Return(&models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Новое",
			Status:     "new",
		}, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "Новое", result.Content)
	})

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateDetailsWithOutbox", assignmentID, execID, "Обновлено",
			(*time.Time)(nil), []string(nil), mock.Anything,
			mock.Anything).Return(&models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Обновлено",
			Status:     "new",
		}, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Обновлено", "", nil)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("invalid co-executor rejects update", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", []string{"not-a-uuid"})
		requireAppError(t, err, "VALIDATION_ERROR", 400, "неверный ID соисполнителя")
		require.Nil(t, result)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "executor")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(nil, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
		assert.Nil(t, result)
	})

	t.Run("finished not admin", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		finished := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "finished",
		}

		repo.On("GetByID", assignmentID).Return(finished, nil).Once()

		result, err := svc.Update(assignmentID.String(), execID.String(), "Новое", "", nil)
		require.Error(t, err)
		requireAppError(t, err, "CONFLICT", 409, "завершённое")
		assert.Nil(t, result)
	})
}

// ---------- TestAssignmentService_UpdateStatus ----------

func TestAssignmentService_UpdateStatus(t *testing.T) {
	// Изменение статуса поручения (в работу, на проверку, завершено и т.д.)
	assignmentID := uuid.New()
	execID := uuid.New()
	docID := uuid.New()

	existing := &models.Assignment{
		ID:         assignmentID,
		DocumentID: docID,
		ExecutorID: execID,
		Content:    "Контент",
		Status:     "new",
	}

	t.Run("executor to in_progress", func(t *testing.T) {
		_, repo, userRepo, _, _ := setupAssignmentService(t, "executor")
		// Override auth currentUser ID to match executor

		executorUser := &models.User{
			ID:    execID,
			Login: "exec",

			IsDocumentParticipant: true,
			IsActive:              true,
		}
		authSvc := newAttachmentPrincipalStub(userRepo)
		authSvc.currentUserID = executorUser.ID
		userRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()

		incomingRepo := mocks.NewIncomingDocStore(t)
		outgoingRepo := mocks.NewOutgoingDocStore(t)
		incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
			return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
			return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		repo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		accessSvc := NewDocumentAccessService(authSvc, nil, repo, newRoleMappedDocumentAccessStore("executor"), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})
		svc2 := NewAssignmentService(repo, userRepo, authSvc, accessSvc, nil, false)

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, execID, "Контент",
			(*time.Time)(nil), "in_progress", "",
			(*time.Time)(nil), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "in_progress",
		}, nil).Once()

		result, err := svc2.UpdateStatus(assignmentID.String(), "in_progress", "")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "in_progress", result.Status)
	})

	t.Run("executor to completed", func(t *testing.T) {
		_, repo, userRepo, _, _ := setupAssignmentService(t, "executor")

		executorUser := &models.User{
			ID:    execID,
			Login: "exec2",

			IsDocumentParticipant: true,
			IsActive:              true,
		}
		authSvc := newAttachmentPrincipalStub(userRepo)
		authSvc.currentUserID = executorUser.ID
		userRepo.On("GetByID", executorUser.ID).Return(executorUser, nil).Maybe()

		incomingRepo := mocks.NewIncomingDocStore(t)
		outgoingRepo := mocks.NewOutgoingDocStore(t)
		incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
			return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
			return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		repo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		accessSvc := NewDocumentAccessService(authSvc, nil, repo, newRoleMappedDocumentAccessStore("executor"), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})
		svc2 := NewAssignmentService(repo, userRepo, authSvc, accessSvc, nil, false)

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, execID, "Контент",
			(*time.Time)(nil), "completed", "Отчёт",
			mock.AnythingOfType("*time.Time"), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "completed",
			Report: "Отчёт",
		}, nil).Once()

		result, err := svc2.UpdateStatus(assignmentID.String(), "completed", "Отчёт")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "completed", result.Status)
	})

	t.Run("active substitute can move assignment to in_progress", func(t *testing.T) {
		substituteID := uuid.New()
		_, repo, userRepo, _, _ := setupAssignmentService(t, "")

		substituteUser := &models.User{
			ID:    substituteID,
			Login: "substitute",

			IsDocumentParticipant: false,
			IsActive:              true,
		}
		authSvc := newAttachmentPrincipalStub(userRepo)
		authSvc.currentUserID = substituteUser.ID
		userRepo.On("GetByID", substituteUser.ID).Return(substituteUser, nil).Maybe()

		incomingRepo := mocks.NewIncomingDocStore(t)
		outgoingRepo := mocks.NewOutgoingDocStore(t)
		incomingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.IncomingDocument {
			return &models.IncomingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		outgoingRepo.On("GetByID", mock.Anything).Return(func(id uuid.UUID) *models.OutgoingDocument {
			return &models.OutgoingDocument{ID: id, NomenclatureID: uuid.New()}
		}, nil).Maybe()
		repo.On("HasDocumentAccess", mock.Anything, mock.Anything).Return(true, nil).Maybe()
		accessSvc := NewDocumentAccessService(authSvc, nil, repo, newRoleMappedDocumentAccessStore(), &kindBackedDocumentStore{incoming: incomingRepo, outgoing: outgoingRepo})
		svc2 := NewAssignmentService(repo, userRepo, authSvc, accessSvc, &userSubstitutionStoreStub{
			isActive: map[[2]uuid.UUID]bool{
				{substituteID, execID}: true,
			},
		}, false)

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, execID, "Контент",
			(*time.Time)(nil), "in_progress", "",
			(*time.Time)(nil), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "in_progress",
		}, nil).Once()

		result, err := svc2.UpdateStatus(assignmentID.String(), "in_progress", "")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "in_progress", result.Status)
	})

	t.Run("clerk to finished from completed", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		completedAt := time.Now()
		completedAssignment := &models.Assignment{
			ID:          assignmentID,
			DocumentID:  docID,
			ExecutorID:  execID,
			Content:     "Контент",
			Status:      "completed",
			Report:      "Отчёт исполнителя",
			CompletedAt: &completedAt,
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, execID, "Контент",
			(*time.Time)(nil), "finished", "Отчёт исполнителя",
			completedAssignment.CompletedAt, []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "finished",
			Report: "Отчёт исполнителя",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "finished", result.Status)
	})

	t.Run("clerk to returned requires reason", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		completedAssignment := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "completed",
			Report:     "Отчёт исполнителя",
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "returned", "")
		require.Error(t, err)
		requireAppError(t, err, "VALIDATION_ERROR", 400, "причина возврата обязательна")
		assert.Nil(t, result)
	})

	t.Run("clerk to returned with reason", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		completedAssignment := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "completed",
			Report:     "Отчёт исполнителя",
		}

		repo.On("GetByID", assignmentID).Return(completedAssignment, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, execID, "Контент",
			(*time.Time)(nil), "returned", "Нужно исправить замечания",
			(*time.Time)(nil), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:     assignmentID,
			Status: "returned",
			Report: "Нужно исправить замечания",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "returned", "Нужно исправить замечания")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "returned", result.Status)
		assert.Equal(t, "Нужно исправить замечания", result.Report)
	})

	for _, terminalStatus := range []string{"finished", "cancelled"} {
		t.Run("manager cannot reopen "+terminalStatus, func(t *testing.T) {
			svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
			terminalAssignment := &models.Assignment{
				ID:         assignmentID,
				DocumentID: docID,
				ExecutorID: execID,
				Content:    "Контент",
				Status:     terminalStatus,
			}
			repo.On("GetByID", assignmentID).Return(terminalAssignment, nil).Once()

			result, err := svc.UpdateStatus(assignmentID.String(), "in_progress", "")

			requireAppError(t, err, "CONFLICT", 409, "недопустимый переход")
			assert.Nil(t, result)
		})
	}

	t.Run("admin forbidden", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "admin")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "недостаточно прав")
		assert.Nil(t, result)
	})

	t.Run("forbidden", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "executor")
		// Executor with different ID than assignment executor — can't set "finished"
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestResolveAssignmentStatusUpdate(t *testing.T) {
	completedAt := time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name           string
		existingStatus string
		existingReport string
		status         string
		report         string
		canManage      bool
		canAct         bool
		wantReport     string
		wantCompleted  *time.Time
		wantErrKind    string
	}{
		{
			name:           "executor starts assignment",
			existingStatus: "new",
			status:         "in_progress",
			canAct:         true,
			wantReport:     "",
		},
		{
			name:           "executor completes with trimmed report",
			existingStatus: "in_progress",
			status:         "completed",
			report:         "  Готово  ",
			canAct:         true,
			wantReport:     "Готово",
		},
		{
			name:           "executor completion requires report",
			existingStatus: "in_progress",
			status:         "completed",
			canAct:         true,
			wantErrKind:    "VALIDATION_ERROR",
		},
		{
			name:           "manager finishes completed assignment with existing report",
			existingStatus: "completed",
			existingReport: "Отчет исполнителя",
			status:         "finished",
			canManage:      true,
			wantReport:     "Отчет исполнителя",
			wantCompleted:  &completedAt,
		},
		{
			name:           "manager return requires reason",
			existingStatus: "completed",
			status:         "returned",
			canManage:      true,
			wantErrKind:    "VALIDATION_ERROR",
		},
		{
			name:           "unprivileged actor is forbidden",
			existingStatus: "new",
			status:         "finished",
			wantErrKind:    "FORBIDDEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := &models.Assignment{
				Status:      tt.existingStatus,
				Report:      tt.existingReport,
				CompletedAt: &completedAt,
			}

			result, err := resolveAssignmentStatusUpdate(existing, tt.status, tt.report, tt.canManage, tt.canAct)
			if tt.wantErrKind != "" {
				require.Error(t, err)
				appErr, ok := models.AsAppError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantErrKind, appErr.Kind)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantReport, result.report)
			if tt.status == "completed" {
				assert.NotNil(t, result.completedAt)
			} else {
				assert.Equal(t, tt.wantCompleted, result.completedAt)
			}
		})
	}
}

func TestResolveAssignmentStatusUpdateTransitionMatrix(t *testing.T) {
	statuses := []string{"new", "in_progress", "completed", "returned", "finished", "cancelled"}
	roles := []struct {
		name      string
		canManage bool
		canAct    bool
		allowed   map[string]map[string]struct{}
	}{
		{name: "executor", canAct: true, allowed: executorAssignmentTransitions},
		{name: "manager", canManage: true, allowed: managerAssignmentTransitions},
		{name: "unprivileged", allowed: map[string]map[string]struct{}{}},
	}

	for _, role := range roles {
		for _, currentStatus := range statuses {
			for _, targetStatus := range statuses {
				t.Run(role.name+"_"+currentStatus+"_to_"+targetStatus, func(t *testing.T) {
					existing := &models.Assignment{Status: currentStatus, Report: "Отчет"}
					result, err := resolveAssignmentStatusUpdate(existing, targetStatus, "Отчет", role.canManage, role.canAct)

					expected := isAssignmentTransitionAllowed(role.allowed, currentStatus, targetStatus)
					if expected {
						require.NoError(t, err)
						require.NotNil(t, result)
						return
					}

					require.Error(t, err)
					assert.Nil(t, result)
					if role.canManage || role.canAct {
						requireAppError(t, err, "CONFLICT", 409, "недопустимый переход")
					} else {
						requireAppError(t, err, "FORBIDDEN", 403, "недостаточно прав")
					}
				})
			}
		}
	}
}

func TestAssignmentExecutorRecipientIDs(t *testing.T) {
	executorID := uuid.New()
	coExecutorID := uuid.New()
	otherCoExecutorID := uuid.New()

	tests := []struct {
		name       string
		assignment *models.Assignment
		want       []uuid.UUID
	}{
		{
			name: "nil assignment",
			want: nil,
		},
		{
			name: "deduplicates executor and coexecutors",
			assignment: &models.Assignment{
				ExecutorID: executorID,
				CoExecutorIDs: []string{
					coExecutorID.String(),
					"not-a-uuid",
					executorID.String(),
					coExecutorID.String(),
					otherCoExecutorID.String(),
				},
			},
			want: []uuid.UUID{executorID, coExecutorID, otherCoExecutorID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, assignmentExecutorRecipientIDs(tt.assignment))
		})
	}
}

func TestAssignmentService_UpdateStatusEmitsUserEvents(t *testing.T) {
	assignmentID := uuid.New()
	docID := uuid.New()
	execID := uuid.New()

	t.Run("completed notifies controllers", func(t *testing.T) {
		svc, repo, userRepo, auth, _ := setupAssignmentServiceWithAccess(t, "executor", newKindActionAccessStore(map[string][]string{
			"incoming_letter": {"assign"},
		}), nil)
		executorID, _ := uuid.Parse(auth.currentUserID.String())
		svc = NewAssignmentService(svc.repo, svc.userRepo, svc.auth, svc.access, svc.substitutions, true)
		controllerID := uuid.New()
		userRepo.On("GetAll").Return([]models.User{
			{ID: controllerID, IsActive: true},
			{ID: executorID, IsActive: true},
		}, nil).Once()

		existing := &models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "in_progress",
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, executorID, "Контент",
			(*time.Time)(nil), "completed", "Отчет",
			mock.AnythingOfType("*time.Time"), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "completed",
			Report:       "Отчет",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "completed", "Отчет")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, repo.Effects, 3)
		assert.Equal(t, models.OutboxEventJournal, repo.Effects[0].EventType)
	})

	t.Run("completed notifies assignee with assign access", func(t *testing.T) {
		svc, repo, userRepo, auth, _ := setupAssignmentServiceWithAccess(t, "executor", newKindActionAccessStore(map[string][]string{
			"incoming_letter": {"assign"},
		}), nil)
		executorID, _ := uuid.Parse(auth.currentUserID.String())
		svc = NewAssignmentService(svc.repo, svc.userRepo, svc.auth, svc.access, svc.substitutions, true)
		userRepo.On("GetAll").Return([]models.User{
			{ID: executorID, IsActive: true},
		}, nil).Once()

		existing := &models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "in_progress",
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, executorID, "Контент",
			(*time.Time)(nil), "completed", "Отчет",
			mock.AnythingOfType("*time.Time"), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:           assignmentID,
			DocumentID:   docID,
			DocumentKind: "incoming_letter",
			ExecutorID:   executorID,
			Content:      "Контент",
			Status:       "completed",
			Report:       "Отчет",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "completed", "Отчет")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, repo.Effects, 2)
		assert.Equal(t, models.OutboxEventUserEvent, repo.Effects[1].EventType)
	})

	t.Run("completed notifies coexecutor with assign access", func(t *testing.T) {
		svc, repo, userRepo, auth, _ := setupAssignmentServiceWithAccess(t, "executor", newKindActionAccessStore(map[string][]string{
			"incoming_letter": {"assign"},
		}), nil)
		executorID, _ := uuid.Parse(auth.currentUserID.String())
		coExecutorID := uuid.New()
		svc = NewAssignmentService(svc.repo, svc.userRepo, svc.auth, svc.access, svc.substitutions, true)
		userRepo.On("GetAll").Return([]models.User{
			{ID: coExecutorID, IsActive: true},
			{ID: executorID, IsActive: true},
		}, nil).Once()

		existing := &models.Assignment{
			ID:            assignmentID,
			DocumentID:    docID,
			DocumentKind:  "incoming_letter",
			ExecutorID:    executorID,
			CoExecutorIDs: []string{coExecutorID.String()},
			Content:       "Контент",
			Status:        "in_progress",
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, executorID, "Контент",
			(*time.Time)(nil), "completed", "Отчет",
			mock.AnythingOfType("*time.Time"), []string{coExecutorID.String()},
			mock.Anything).Return(&models.Assignment{
			ID:            assignmentID,
			DocumentID:    docID,
			DocumentKind:  "incoming_letter",
			ExecutorID:    executorID,
			CoExecutorIDs: []string{coExecutorID.String()},
			Content:       "Контент",
			Status:        "completed",
			Report:        "Отчет",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "completed", "Отчет")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, repo.Effects, 3)
		assert.Equal(t, models.OutboxEventUserEvent, repo.Effects[2].EventType)
	})

	t.Run("returned notifies executor", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		svc = NewAssignmentService(svc.repo, svc.userRepo, svc.auth, svc.access, svc.substitutions, true)
		completedAt := time.Now()
		existing := &models.Assignment{
			ID:          assignmentID,
			DocumentID:  docID,
			ExecutorID:  execID,
			Content:     "Контент",
			Status:      "completed",
			Report:      "Отчет",
			CompletedAt: &completedAt,
		}
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("UpdateWithOutbox", assignmentID, execID, "Контент",
			(*time.Time)(nil), "returned", "Нужно доработать",
			(*time.Time)(nil), []string(nil),
			mock.Anything).Return(&models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "returned",
			Report:     "Нужно доработать",
		}, nil).Once()

		result, err := svc.UpdateStatus(assignmentID.String(), "returned", "Нужно доработать")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, repo.Effects, 2)
		assert.Equal(t, models.OutboxEventUserEvent, repo.Effects[1].EventType)
	})
}

// ---------- TestAssignmentService_GetList ----------

func TestAssignmentService_GetList(t *testing.T) {
	// Получение списка поручений с фильтрацией (для дашборда или списков)
	t.Run("success", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")

		filter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, Page: 1, PageSize: 20}
		filter.ExecutorID = auth.currentUserID.String()
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", filter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, 1, result.TotalCount)
	})

	t.Run("default pagination", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")

		// Filter with 0 page/pagesize — should default to 1/20
		filter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, Page: 0, PageSize: 0}
		expectedFilter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, Page: 1, PageSize: 20, ExecutorID: auth.currentUserID.String()}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{},
			TotalCount: 0,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("not authenticated", func(t *testing.T) {
		svc, _ := setupAssignmentServiceNotAuth(t)

		result, err := svc.GetList(models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}})
		require.Error(t, err)
		assert.Equal(t, models.ErrUnauthorized, err)
		assert.Nil(t, result)
	})

	t.Run("admin without document rights is forbidden", func(t *testing.T) {
		svc, _, _, _, _ := setupAssignmentService(t, "admin")
		result, err := svc.GetList(models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}})
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
		assert.Nil(t, result)
	})

	t.Run("document assignments allowed for clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		docID := uuid.New()
		filter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, DocumentID: docID.String(), Page: 1, PageSize: 100, ShowFinished: true}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), DocumentID: docID, Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   100,
		}
		repo.On("GetList", filter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})

	t.Run("document assignments scoped for executor without assign right", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")
		docID := uuid.New()
		filter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution},
			DocumentID:   docID.String(),
			Page:         1,
			PageSize:     100,
			ShowFinished: true,
		}
		expectedFilter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution},
			DocumentID:         docID.String(),
			Page:               1,
			PageSize:           100,
			ShowFinished:       true,
			ExecutorID:         auth.currentUserID.String(),
			AccessibleByUserID: auth.currentUserID.String(),
		}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), DocumentID: docID, ExecutorID: uuid.MustParse(auth.currentUserID.String()), Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   100,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})

	t.Run("substitute list ignores current user executor filter from client", func(t *testing.T) {
		principalID := uuid.New()
		substitutions := &userSubstitutionStoreStub{activePrincipals: []uuid.UUID{principalID}}
		svc, repo, _, auth, _ := setupAssignmentServiceWithAccess(t, "", newRoleMappedDocumentAccessStore(""), substitutions)
		currentUserID := auth.currentUserID.String()

		filter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution},
			Page:       1,
			PageSize:   20,
			ExecutorID: currentUserID,
		}
		expectedFilter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution},
			Page:                1,
			PageSize:            20,
			AccessibleByUserID:  currentUserID,
			AccessibleByUserIDs: []string{currentUserID, principalID.String()},
		}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), ExecutorID: principalID, Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
		assert.True(t, result.Items[0].CanAct)
	})

	t.Run("partial assignment rights are scoped by document kind and own assignments", func(t *testing.T) {
		svc, repo, _, auth, _ := setupAssignmentService(t, "executor")
		svc.access = NewDocumentAccessService(
			auth,
			nil,
			repo,

			newKindActionAccessStore(map[string][]string{
				string(models.DocumentKindIncomingLetter): {"assign"},
			}),
			nil,
		)

		filter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, Page: 1, PageSize: 20}
		expectedFilter := models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution},
			Page:                 1,
			PageSize:             20,
			AllowedDocumentKinds: []string{string(models.DocumentKindIncomingLetter)},
			AccessibleByUserID:   auth.currentUserID.String(),
		}
		repoResult := &models.PagedResult[models.Assignment]{
			Items:      []models.Assignment{{ID: uuid.New(), Status: "new"}},
			TotalCount: 1,
			Page:       1,
			PageSize:   20,
		}
		repo.On("GetList", expectedFilter).Return(repoResult, nil).Once()

		result, err := svc.GetList(filter)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Items, 1)
	})
}

// ---------- TestAssignmentService_Delete ----------

func TestAssignmentService_Delete(t *testing.T) {
	// Удаление поручения из системы
	assignmentID := uuid.New()
	execID := uuid.New()
	docID := uuid.New()

	existing := &models.Assignment{
		ID:         assignmentID,
		DocumentID: docID,
		ExecutorID: execID,
		Content:    "Контент",
		Status:     "new",
	}

	t.Run("success clerk", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")

		repo.On("GetByID", assignmentID).Return(existing, nil).Once()
		repo.On("DeleteWithOutbox", assignmentID, mock.Anything).Return(nil).Once()

		err := svc.Delete(assignmentID.String())
		require.NoError(t, err)
	})

	t.Run("forbidden executor", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "executor")
		repo.On("GetByID", assignmentID).Return(existing, nil).Once()

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		assert.Equal(t, models.ErrForbidden, err)
	})

	t.Run("finished not admin", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		finished := &models.Assignment{
			ID:         assignmentID,
			DocumentID: docID,
			ExecutorID: execID,
			Content:    "Контент",
			Status:     "finished",
		}

		repo.On("GetByID", assignmentID).Return(finished, nil).Once()

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		requireAppError(t, err, "CONFLICT", 409, "завершённое")
	})

	t.Run("not found", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		repo.On("GetByID", assignmentID).Return(nil, nil).Once()

		err := svc.Delete(assignmentID.String())
		require.Error(t, err)
		appErr, ok := models.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, "NOT_FOUND", appErr.Kind)
		assert.Equal(t, 404, appErr.Code)
	})
}

func TestAssignmentServiceRejectsIneligibleExecutorsBeforeWrite(t *testing.T) {
	documentID, assignmentID, executorID, coExecutorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		name      string
		operation string
		eligible  map[uuid.UUID]struct{}
		want      string
	}{
		{"create main", "create", map[uuid.UUID]struct{}{}, "исполнитель недоступен"},
		{"create coexecutor", "create", map[uuid.UUID]struct{}{executorID: {}}, "соисполнитель недоступен"},
		{"update main", "update", map[uuid.UUID]struct{}{coExecutorID: {}}, "исполнитель недоступен"},
		{"update coexecutor", "update", map[uuid.UUID]struct{}{executorID: {}}, "соисполнитель недоступен"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
			users := mocks.NewUserStore(t)
			users.On("GetEligibleRecipientIDs", []uuid.UUID{executorID, coExecutorID}).Return(tc.eligible, nil).Once()
			svc.userRepo = users
			if tc.operation == "update" {
				repo.On("GetByID", assignmentID).Return(&models.Assignment{ID: assignmentID, DocumentID: documentID, ExecutorID: executorID, Status: "new"}, nil).Once()
			}
			var err error
			if tc.operation == "create" {
				_, err = svc.createExecution(documentID.String(), executorID.String(), "Поручение", "", []string{coExecutorID.String()})
			} else {
				_, err = svc.Update(assignmentID.String(), executorID.String(), "Поручение", "", []string{coExecutorID.String()})
			}
			requireAppError(t, err, "VALIDATION_ERROR", 400, tc.want)
		})
	}
}

type failingAssignmentAccessStore struct {
	ports.DocumentAccessStore
	err error
}

func (s failingAssignmentAccessStore) HasPermission(kindCode, action, departmentID, userID string) (bool, error) {
	return false, s.err
}

func TestAssignmentStatusPropagatesAccessStoreError(t *testing.T) {
	failure := assert.AnError
	for _, assignedToActor := range []bool{false, true} {
		t.Run(map[bool]string{false: "manager", true: "executor"}[assignedToActor], func(t *testing.T) {
			svc, repo, _, auth, _ := setupAssignmentServiceWithAccess(t, "clerk", failingAssignmentAccessStore{err: failure}, nil)
			assignmentID, documentID := uuid.New(), uuid.New()
			executorID := uuid.New()
			if assignedToActor {
				executorID = auth.currentUserID
			}
			repo.On("GetByID", assignmentID).Return(&models.Assignment{ID: assignmentID, DocumentID: documentID, ExecutorID: executorID, Status: "completed"}, nil).Once()
			result, err := svc.UpdateStatus(assignmentID.String(), "finished", "")
			require.Nil(t, result)
			require.ErrorIs(t, err, failure)
		})
	}
}

func TestAssignmentListPropagatesDocumentAccessError(t *testing.T) {
	svc, _, _, _, _ := setupAssignmentServiceWithAccess(t, "clerk", failingAssignmentAccessStore{err: assert.AnError}, nil)
	result, err := svc.GetList(models.AssignmentFilter{Types: []string{models.AssignmentTypeExecution}, DocumentID: uuid.New().String()})
	require.Nil(t, result)
	require.ErrorIs(t, err, assert.AnError)
}

func TestAssignmentServiceCreateTaskRequiresExplicitKnownType(t *testing.T) {
	service := &AssignmentService{}
	for _, typ := range []string{"", "unknown"} {
		result, err := service.CreateTask(models.AssignmentRequest{Type: typ})
		require.Nil(t, result)
		require.ErrorContains(t, err, "неизвестный тип поручения")
	}
}

func TestAssignmentServiceRejectsUnknownListFilterValues(t *testing.T) {
	for _, filter := range []models.AssignmentFilter{
		{Types: []string{"execution", "unknown"}},
		{Statuses: []string{"new", "unknown"}},
	} {
		t.Run(fmt.Sprint(filter), func(t *testing.T) {
			_, err := (&AssignmentService{}).GetList(filter)
			var appErr *models.AppError
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, 400, appErr.Code)
		})
	}
}

func TestAssignmentAcknowledgmentIgnoresContent(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		var created *models.Assignment
		repo.On("CreateRecipientTaskWithOutbox", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			created = args.Get(0).(*models.Assignment)
			require.Empty(t, created.Content)
		}).Return(nil).Once()
		repo.On("GetByID", mock.Anything).Return(func(uuid.UUID) *models.Assignment { return created }, nil).Once()
		result, err := svc.CreateTask(models.AssignmentRequest{
			Type: models.AssignmentTypeAcknowledgment, DocumentID: uuid.NewString(),
			Content: "Legacy client comment", UserIDs: []string{uuid.NewString()},
		})
		require.NoError(t, err)
		require.Empty(t, result.Content)
	})
	t.Run("update", func(t *testing.T) {
		svc, repo, _, _, _ := setupAssignmentService(t, "clerk")
		existing := &models.Assignment{ID: uuid.New(), DocumentID: uuid.New(), Type: models.AssignmentTypeAcknowledgment, Status: "new", Content: "Historical comment"}
		repo.On("GetByID", existing.ID).Return(existing, nil)
		due := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
		repo.On("UpdateRecipientTaskDeadline", existing.ID, &due, mock.Anything).Return(nil).Once()
		result, err := svc.Update(existing.ID.String(), "", "New comment", "2026-12-31", nil)
		require.NoError(t, err)
		require.Empty(t, result.Content)
		require.Equal(t, "Historical comment", existing.Content)
	})
}
