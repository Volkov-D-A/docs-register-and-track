package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type documentAccessTestDeps struct {
	auth       *AuthService
	userRepo   *mocks.UserStore
	accessRepo *kindActionDocumentAccessStore
	depRepo    *documentAccessDepartmentStore
	assignRepo *documentAccessAssignmentStore
	ackRepo    *documentAccessAcknowledgmentStore
	docRepo    *documentAccessDocumentStore
	service    *DocumentAccessService
	user       *models.User
}

type documentAccessDepartmentStore struct {
	nomenclatureIDs []string
	err             error
}

func (s *documentAccessDepartmentStore) GetAll() ([]models.Department, error) {
	return nil, nil
}

func (s *documentAccessDepartmentStore) GetNomenclatureIDs(departmentID uuid.UUID) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.nomenclatureIDs, nil
}

func (s *documentAccessDepartmentStore) Create(name string, nomenclatureIDs []string) (*models.Department, error) {
	return nil, nil
}

func (s *documentAccessDepartmentStore) Update(id uuid.UUID, name string, nomenclatureIDs []string) (*models.Department, error) {
	return nil, nil
}

func (s *documentAccessDepartmentStore) Delete(id uuid.UUID) error {
	return nil
}

type documentAccessAssignmentStore struct {
	accessible map[uuid.UUID]struct{}
	err        error
}

func (s *documentAccessAssignmentStore) Create(documentID uuid.UUID, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string) (*models.Assignment, error) {
	return nil, nil
}

func (s *documentAccessAssignmentStore) Update(id uuid.UUID, executorID uuid.UUID, content string, deadline *time.Time, status, report string, completedAt *time.Time, coExecutorIDs []string) (*models.Assignment, error) {
	return nil, nil
}

func (s *documentAccessAssignmentStore) Delete(id uuid.UUID) error {
	return nil
}

func (s *documentAccessAssignmentStore) GetByID(id uuid.UUID) (*models.Assignment, error) {
	return nil, nil
}

func (s *documentAccessAssignmentStore) GetList(filter models.AssignmentFilter) (*models.PagedResult[models.Assignment], error) {
	return nil, nil
}

func (s *documentAccessAssignmentStore) HasDocumentAccess(userID, documentID uuid.UUID) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	_, ok := s.accessible[documentID]
	return ok, nil
}

func (s *documentAccessAssignmentStore) GetAccessibleDocumentIDs(userID uuid.UUID, documentIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[uuid.UUID]struct{})
	for _, documentID := range documentIDs {
		if _, ok := s.accessible[documentID]; ok {
			result[documentID] = struct{}{}
		}
	}
	return result, nil
}

type documentAccessAcknowledgmentStore struct {
	accessible map[uuid.UUID]struct{}
	err        error
}

func (s *documentAccessAcknowledgmentStore) Create(a *models.Acknowledgment) error {
	return nil
}

func (s *documentAccessAcknowledgmentStore) GetByID(id uuid.UUID) (*models.Acknowledgment, error) {
	return nil, nil
}

func (s *documentAccessAcknowledgmentStore) GetByDocumentID(documentID uuid.UUID) ([]models.Acknowledgment, error) {
	return nil, nil
}

func (s *documentAccessAcknowledgmentStore) GetPendingForUser(userID uuid.UUID) ([]models.Acknowledgment, error) {
	return nil, nil
}

func (s *documentAccessAcknowledgmentStore) GetAllActive() ([]models.Acknowledgment, error) {
	return nil, nil
}

func (s *documentAccessAcknowledgmentStore) HasDocumentAccess(userID, documentID uuid.UUID) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	_, ok := s.accessible[documentID]
	return ok, nil
}

func (s *documentAccessAcknowledgmentStore) GetAccessibleDocumentIDs(userID uuid.UUID, documentIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make(map[uuid.UUID]struct{})
	for _, documentID := range documentIDs {
		if _, ok := s.accessible[documentID]; ok {
			result[documentID] = struct{}{}
		}
	}
	return result, nil
}

func (s *documentAccessAcknowledgmentStore) MarkViewed(ackID, userID uuid.UUID) error {
	return nil
}

func (s *documentAccessAcknowledgmentStore) MarkConfirmed(ackID, userID uuid.UUID) error {
	return nil
}

func (s *documentAccessAcknowledgmentStore) Delete(id uuid.UUID) error {
	return nil
}

type documentAccessDocumentStore struct {
	docs map[uuid.UUID]models.Document
	err  error
}

func (s *documentAccessDocumentStore) GetByID(id uuid.UUID) (*models.Document, error) {
	if s.err != nil {
		return nil, s.err
	}
	doc, ok := s.docs[id]
	if !ok {
		return nil, nil
	}
	return &doc, nil
}

func (s *documentAccessDocumentStore) GetByIDs(ids []uuid.UUID) ([]models.Document, error) {
	if s.err != nil {
		return nil, s.err
	}
	docs := make([]models.Document, 0, len(ids))
	for _, id := range ids {
		if doc, ok := s.docs[id]; ok {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

func setupDocumentAccessService(t *testing.T, user *models.User, allowed map[models.DocumentKind]map[string]bool) *documentAccessTestDeps {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := NewAuthService(nil, userRepo)
	if user != nil {
		auth.currentUserID = user.ID
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	accessRepo := &kindActionDocumentAccessStore{allowed: allowed}
	depRepo := &documentAccessDepartmentStore{}
	assignRepo := &documentAccessAssignmentStore{accessible: map[uuid.UUID]struct{}{}}
	ackRepo := &documentAccessAcknowledgmentStore{accessible: map[uuid.UUID]struct{}{}}
	docRepo := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{}}
	service := NewDocumentAccessService(auth, depRepo, assignRepo, ackRepo, accessRepo, docRepo, nil, nil)

	return &documentAccessTestDeps{
		auth:       auth,
		userRepo:   userRepo,
		accessRepo: accessRepo,
		depRepo:    depRepo,
		assignRepo: assignRepo,
		ackRepo:    ackRepo,
		docRepo:    docRepo,
		service:    service,
		user:       user,
	}
}

func documentAccessUser(isParticipant bool, departmentID *uuid.UUID) *models.User {
	user := &models.User{
		ID:                    uuid.New(),
		Login:                 "access_user",
		IsActive:              true,
		IsDocumentParticipant: isParticipant,
	}
	if departmentID != nil {
		user.DepartmentID = departmentID
		user.Department = &models.Department{ID: *departmentID, Name: "Dept"}
	}
	return user
}

func allowDocumentActions(kind models.DocumentKind, actions ...string) map[models.DocumentKind]map[string]bool {
	allowed := map[models.DocumentKind]map[string]bool{kind: {}}
	for _, action := range actions {
		allowed[kind][action] = true
	}
	return allowed
}

func addDocumentActions(allowed map[models.DocumentKind]map[string]bool, kind models.DocumentKind, actions ...string) map[models.DocumentKind]map[string]bool {
	if allowed == nil {
		allowed = map[models.DocumentKind]map[string]bool{}
	}
	if allowed[kind] == nil {
		allowed[kind] = map[string]bool{}
	}
	for _, action := range actions {
		allowed[kind][action] = true
	}
	return allowed
}

func documentAccessDoc(id, nomenclatureID uuid.UUID, kind models.DocumentKind) models.Document {
	return models.Document{
		ID:             id,
		Kind:           kind,
		NomenclatureID: nomenclatureID,
	}
}

func TestDocumentAccessService_RequireDomainRead(t *testing.T) {
	t.Run("unauthorized user is rejected", func(t *testing.T) {
		deps := setupDocumentAccessService(t, nil, nil)

		err := deps.service.RequireDomainRead()

		require.ErrorIs(t, err, models.ErrUnauthorized)
	})

	t.Run("document participant is allowed without explicit document permissions", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(true, nil), nil)

		err := deps.service.RequireDomainRead()

		require.NoError(t, err)
	})

	t.Run("non participant without document permissions is forbidden", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(false, nil), nil)

		err := deps.service.RequireDomainRead()

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("non participant with any document permission is allowed", func(t *testing.T) {
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "upload"),
		)

		err := deps.service.RequireDomainRead()

		require.NoError(t, err)
	})
}

func TestDocumentAccessService_RequireCreate(t *testing.T) {
	t.Run("requires authentication", func(t *testing.T) {
		deps := setupDocumentAccessService(t, nil, nil)

		err := deps.service.RequireCreate(models.DocumentKindIncomingLetter)

		require.ErrorIs(t, err, models.ErrUnauthorized)
	})

	t.Run("allows configured create permission", func(t *testing.T) {
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "create"),
		)

		err := deps.service.RequireCreate(models.DocumentKindIncomingLetter)

		require.NoError(t, err)
	})

	t.Run("rejects missing create permission", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(false, nil), nil)

		err := deps.service.RequireCreate(models.DocumentKindIncomingLetter)

		require.ErrorIs(t, err, models.ErrForbidden)
	})
}

func TestDocumentAccessService_ResolveReadScope(t *testing.T) {
	t.Run("full read permission returns unrestricted scope", func(t *testing.T) {
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)

		scope, err := deps.service.ResolveReadScope(models.DocumentKindIncomingLetter)

		require.NoError(t, err)
		require.NotNil(t, scope)
		assert.False(t, scope.Restricted)
		assert.Empty(t, scope.AccessibleByUserID)
		assert.Empty(t, scope.AllowedNomenclatureIDs)
	})

	t.Run("participant without read permission receives department and personal restricted scope", func(t *testing.T) {
		departmentID := uuid.New()
		nomenclatureID := uuid.New()
		user := documentAccessUser(true, &departmentID)
		deps := setupDocumentAccessService(t, user, nil)
		deps.depRepo.nomenclatureIDs = []string{nomenclatureID.String()}

		scope, err := deps.service.ResolveReadScope(models.DocumentKindIncomingLetter)

		require.NoError(t, err)
		require.NotNil(t, scope)
		assert.True(t, scope.Restricted)
		assert.Equal(t, user.ID.String(), scope.AccessibleByUserID)
		assert.Equal(t, []string{nomenclatureID.String()}, scope.AllowedNomenclatureIDs)
	})

	t.Run("non participant with domain access receives personal restricted scope only", func(t *testing.T) {
		user := documentAccessUser(false, nil)
		deps := setupDocumentAccessService(
			t,
			user,
			allowDocumentActions(models.DocumentKindIncomingLetter, "upload"),
		)

		scope, err := deps.service.ResolveReadScope(models.DocumentKindIncomingLetter)

		require.NoError(t, err)
		require.NotNil(t, scope)
		assert.True(t, scope.Restricted)
		assert.Equal(t, user.ID.String(), scope.AccessibleByUserID)
		assert.Empty(t, scope.AllowedNomenclatureIDs)
	})
}

func TestDocumentAccessService_ResolveReadableDocuments(t *testing.T) {
	departmentID := uuid.New()
	allowedNomenclatureID := uuid.New()
	deniedNomenclatureID := uuid.New()
	byDepartmentID := uuid.New()
	byAssignmentID := uuid.New()
	byAcknowledgmentID := uuid.New()
	deniedID := uuid.New()
	user := documentAccessUser(true, &departmentID)
	deps := setupDocumentAccessService(t, user, nil)
	deps.depRepo.nomenclatureIDs = []string{allowedNomenclatureID.String()}
	deps.assignRepo.accessible[byAssignmentID] = struct{}{}
	deps.ackRepo.accessible[byAcknowledgmentID] = struct{}{}
	deps.docRepo.docs = map[uuid.UUID]models.Document{
		byDepartmentID: {
			ID:             byDepartmentID,
			Kind:           models.DocumentKindIncomingLetter,
			NomenclatureID: allowedNomenclatureID,
		},
		byAssignmentID: {
			ID:             byAssignmentID,
			Kind:           models.DocumentKindIncomingLetter,
			NomenclatureID: deniedNomenclatureID,
		},
		byAcknowledgmentID: {
			ID:             byAcknowledgmentID,
			Kind:           models.DocumentKindOutgoingLetter,
			NomenclatureID: deniedNomenclatureID,
		},
		deniedID: {
			ID:             deniedID,
			Kind:           models.DocumentKindOutgoingLetter,
			NomenclatureID: deniedNomenclatureID,
		},
	}

	readable, err := deps.service.ResolveReadableDocuments([]uuid.UUID{
		uuid.Nil,
		byDepartmentID,
		byDepartmentID,
		byAssignmentID,
		byAcknowledgmentID,
		deniedID,
	})

	require.NoError(t, err)
	require.Len(t, readable, 3)
	assert.Contains(t, readable, byDepartmentID)
	assert.Contains(t, readable, byAssignmentID)
	assert.Contains(t, readable, byAcknowledgmentID)
	assert.NotContains(t, readable, deniedID)
}

func TestDocumentAccessService_RequireResolvedRead(t *testing.T) {
	t.Run("allows explicit read permission", func(t *testing.T) {
		documentID := uuid.New()
		nomenclatureID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)

		err := deps.service.RequireResolvedRead(
			string(models.DocumentKindIncomingLetter),
			documentID,
			nomenclatureID,
		)

		require.NoError(t, err)
	})

	t.Run("allows participant by department nomenclature", func(t *testing.T) {
		departmentID := uuid.New()
		nomenclatureID := uuid.New()
		deps := setupDocumentAccessService(t, documentAccessUser(true, &departmentID), nil)
		deps.depRepo.nomenclatureIDs = []string{nomenclatureID.String()}

		err := deps.service.RequireResolvedRead(
			string(models.DocumentKindIncomingLetter),
			uuid.New(),
			nomenclatureID,
		)

		require.NoError(t, err)
	})

	t.Run("allows participant by assignment access", func(t *testing.T) {
		documentID := uuid.New()
		departmentID := uuid.New()
		deps := setupDocumentAccessService(t, documentAccessUser(true, &departmentID), nil)
		deps.assignRepo.accessible[documentID] = struct{}{}

		err := deps.service.RequireResolvedRead(
			string(models.DocumentKindIncomingLetter),
			documentID,
			uuid.New(),
		)

		require.NoError(t, err)
	})

	t.Run("allows participant by acknowledgment access", func(t *testing.T) {
		documentID := uuid.New()
		departmentID := uuid.New()
		deps := setupDocumentAccessService(t, documentAccessUser(true, &departmentID), nil)
		deps.ackRepo.accessible[documentID] = struct{}{}

		err := deps.service.RequireResolvedRead(
			string(models.DocumentKindIncomingLetter),
			documentID,
			uuid.New(),
		)

		require.NoError(t, err)
	})

	t.Run("rejects participant without implicit access", func(t *testing.T) {
		departmentID := uuid.New()
		deps := setupDocumentAccessService(t, documentAccessUser(true, &departmentID), nil)

		err := deps.service.RequireResolvedRead(
			string(models.DocumentKindIncomingLetter),
			uuid.New(),
			uuid.New(),
		)

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("propagates assignment access errors", func(t *testing.T) {
		documentID := uuid.New()
		departmentID := uuid.New()
		expectedErr := errors.New("assignment lookup failed")
		deps := setupDocumentAccessService(t, documentAccessUser(true, &departmentID), nil)
		deps.assignRepo.err = expectedErr

		err := deps.service.RequireResolvedRead(
			string(models.DocumentKindIncomingLetter),
			documentID,
			uuid.New(),
		)

		require.ErrorIs(t, err, expectedErr)
	})
}

func TestDocumentAccessService_RequireRead(t *testing.T) {
	t.Run("loads document and checks resolved read", func(t *testing.T) {
		documentID := uuid.New()
		nomenclatureID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)
		deps.docRepo.docs[documentID] = documentAccessDoc(
			documentID,
			nomenclatureID,
			models.DocumentKindIncomingLetter,
		)

		err := deps.service.RequireRead(string(models.DocumentKindIncomingLetter), documentID)

		require.NoError(t, err)
	})

	t.Run("returns not found for missing document", func(t *testing.T) {
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)

		err := deps.service.RequireRead(string(models.DocumentKindIncomingLetter), uuid.New())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "документ не найден")
	})

	t.Run("propagates document repository errors", func(t *testing.T) {
		expectedErr := errors.New("document lookup failed")
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)
		deps.docRepo.err = expectedErr

		err := deps.service.RequireRead(string(models.DocumentKindIncomingLetter), uuid.New())

		require.ErrorIs(t, err, expectedErr)
	})
}

func TestDocumentAccessService_RequireDocumentAction(t *testing.T) {
	t.Run("requires requested action and read access", func(t *testing.T) {
		documentID := uuid.New()
		nomenclatureID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read", "update"),
		)
		deps.docRepo.docs[documentID] = documentAccessDoc(
			documentID,
			nomenclatureID,
			models.DocumentKindIncomingLetter,
		)

		err := deps.service.RequireDocumentAction(documentID, "update")

		require.NoError(t, err)
	})

	t.Run("rejects missing requested action even when document is readable", func(t *testing.T) {
		documentID := uuid.New()
		nomenclatureID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)
		deps.docRepo.docs[documentID] = documentAccessDoc(
			documentID,
			nomenclatureID,
			models.DocumentKindIncomingLetter,
		)

		err := deps.service.RequireDocumentAction(documentID, "update")

		require.ErrorIs(t, err, models.ErrForbidden)
	})

	t.Run("rejects action when read access is missing", func(t *testing.T) {
		documentID := uuid.New()
		nomenclatureID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "update"),
		)
		deps.docRepo.docs[documentID] = documentAccessDoc(
			documentID,
			nomenclatureID,
			models.DocumentKindIncomingLetter,
		)

		err := deps.service.RequireDocumentAction(documentID, "update")

		require.ErrorIs(t, err, models.ErrForbidden)
	})
}

func TestDocumentAccessService_DocumentActionQueries(t *testing.T) {
	allowed := addDocumentActions(nil, models.DocumentKindIncomingLetter, "read", "upload")
	allowed = addDocumentActions(allowed, models.DocumentKindOutgoingLetter, "read")
	deps := setupDocumentAccessService(t, documentAccessUser(false, nil), allowed)

	actions, err := deps.service.GetAvailableActions(models.DocumentKindIncomingLetter)
	require.NoError(t, err)
	assert.Equal(t, []string{"read", "upload"}, actions)

	hasUpload, err := deps.service.HasAnyDocumentAction("upload")
	require.NoError(t, err)
	assert.True(t, hasUpload)

	kinds, err := deps.service.GetDocumentKindsWithAction("read")
	require.NoError(t, err)
	assert.ElementsMatch(t, []models.DocumentKind{
		models.DocumentKindIncomingLetter,
		models.DocumentKindOutgoingLetter,
	}, kinds)

	hasAssign, err := deps.service.HasDocumentAction(models.DocumentKindIncomingLetter, "assign")
	require.NoError(t, err)
	assert.False(t, hasAssign)
}

func TestDocumentAccessService_HasAssignmentAccess(t *testing.T) {
	t.Run("returns assignment access for authenticated domain user", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)
		deps.assignRepo.accessible[documentID] = struct{}{}

		ok, err := deps.service.HasAssignmentAccess(documentID)

		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("rejects users without document domain access", func(t *testing.T) {
		deps := setupDocumentAccessService(t, documentAccessUser(false, nil), nil)

		ok, err := deps.service.HasAssignmentAccess(uuid.New())

		require.ErrorIs(t, err, models.ErrForbidden)
		assert.False(t, ok)
	})
}

func TestDocumentAccessService_RequireViewJournal(t *testing.T) {
	t.Run("requires journal permission and read access", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read", "view_journal"),
		)
		deps.docRepo.docs[documentID] = documentAccessDoc(
			documentID,
			uuid.New(),
			models.DocumentKindIncomingLetter,
		)

		err := deps.service.RequireViewJournal(documentID)

		require.NoError(t, err)
	})

	t.Run("rejects missing journal permission", func(t *testing.T) {
		documentID := uuid.New()
		deps := setupDocumentAccessService(
			t,
			documentAccessUser(false, nil),
			allowDocumentActions(models.DocumentKindIncomingLetter, "read"),
		)
		deps.docRepo.docs[documentID] = documentAccessDoc(
			documentID,
			uuid.New(),
			models.DocumentKindIncomingLetter,
		)

		err := deps.service.RequireViewJournal(documentID)

		require.ErrorIs(t, err, models.ErrForbidden)
	})
}

func TestDocumentAccessService_RequireReadAnyType(t *testing.T) {
	documentID := uuid.New()
	deps := setupDocumentAccessService(
		t,
		documentAccessUser(false, nil),
		allowDocumentActions(models.DocumentKindOutgoingLetter, "read"),
	)
	deps.docRepo.docs[documentID] = documentAccessDoc(
		documentID,
		uuid.New(),
		models.DocumentKindOutgoingLetter,
	)

	err := deps.service.RequireReadAnyType(documentID)

	require.NoError(t, err)
}
