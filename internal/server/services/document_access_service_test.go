package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

type documentAccessTestDeps struct {
	auth       *accessPrincipalStub
	accessRepo *kindActionDocumentAccessStore
	depRepo    *documentAccessDepartmentStore
	assignRepo *documentAccessAssignmentStore
	docRepo    *documentAccessDocumentStore
	subRepo    *userSubstitutionStoreStub
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

	auth := &accessPrincipalStub{user: user}

	accessRepo := &kindActionDocumentAccessStore{allowed: allowed, readable: map[uuid.UUID]bool{}}
	depRepo := &documentAccessDepartmentStore{}
	assignRepo := &documentAccessAssignmentStore{accessible: map[uuid.UUID]struct{}{}}
	docRepo := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{}}
	subRepo := &userSubstitutionStoreStub{}
	service := NewDocumentAccessService(auth, accessRepo, docRepo)

	return &documentAccessTestDeps{
		auth:       auth,
		accessRepo: accessRepo,
		depRepo:    depRepo,
		assignRepo: assignRepo,
		docRepo:    docRepo,
		subRepo:    subRepo,
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

// The request principal supplies identity; HTTP session validation is tested on the server.
type accessPrincipalStub struct {
	user *models.User
	err  error
}

func (p *accessPrincipalStub) GetCurrentUser() (*dto.User, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.user == nil {
		return nil, models.ErrUnauthorized
	}
	return dto.MapUser(p.user), nil
}
func (p *accessPrincipalStub) RequireAuthenticated() error { _, err := p.GetCurrentUser(); return err }
func (p *accessPrincipalStub) GetCurrentUserUUID() (uuid.UUID, error) {
	if err := p.RequireAuthenticated(); err != nil {
		return uuid.Nil, err
	}
	return p.user.ID, nil
}

type kindActionDocumentAccessStore struct {
	ports.DocumentAccessStore
	allowed       map[models.DocumentKind]map[string]bool
	readable      map[uuid.UUID]bool
	visibilityErr error
	visible       func(uuid.UUID) bool
}

func (s *kindActionDocumentAccessStore) HasPermission(kind, action, departmentID, userID string) (bool, error) {
	return s.allowed[models.DocumentKind(kind)][action], nil
}

type userSubstitutionStoreStub struct {
	isActive map[[2]uuid.UUID]bool
	ports.UserSubstitutionStore
	activePrincipals []uuid.UUID
}

func (s *userSubstitutionStoreStub) GetActivePrincipalIDs(uuid.UUID) ([]uuid.UUID, error) {
	return s.activePrincipals, nil
}

func (s *userSubstitutionStoreStub) IsActiveSubstitute(substituteID, principalID uuid.UUID) (bool, error) {
	return s.isActive[[2]uuid.UUID{substituteID, principalID}], nil
}

func (s *kindActionDocumentAccessStore) CanReadDocument(_ uuid.UUID, id uuid.UUID) (bool, error) {
	return s.readable[id] || (s.visible != nil && s.visible(id)), s.visibilityErr
}
func (s *kindActionDocumentAccessStore) GetReadableDocumentIDs(_ uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := map[uuid.UUID]bool{}
	for _, id := range ids {
		if s.readable[id] || (s.visible != nil && s.visible(id)) {
			result[id] = true
		}
	}
	return result, s.visibilityErr
}
func TestDocumentAccessUnifiedPolicy(t *testing.T) {
	deps := setupDocumentAccessService(t, documentAccessUser(false, nil), nil)
	require.NoError(t, deps.service.RequireDomainRead())
	scope, err := deps.service.ResolveReadScope(models.DocumentKindIncomingLetter)
	require.NoError(t, err)
	require.Equal(t, deps.user.ID.String(), scope.UserID)
	require.True(t, scope.Restricted)
	id := uuid.New()
	deps.docRepo.docs[id] = documentAccessDoc(id, uuid.New(), models.DocumentKindIncomingLetter)
	require.ErrorIs(t, deps.service.RequireReadAnyType(id), models.ErrForbidden)
	deps.accessRepo.readable[id] = true
	require.NoError(t, deps.service.RequireReadAnyType(id))
	require.NoError(t, deps.service.RequireViewJournal(id))
	require.ErrorIs(t, deps.service.RequireDocumentAction(id, "register"), models.ErrForbidden)
	deps.accessRepo.allowed = allowDocumentActions(models.DocumentKindIncomingLetter, "register")
	require.NoError(t, deps.service.RequireCreate(models.DocumentKindIncomingLetter))
	require.NoError(t, deps.service.RequireDocumentAction(id, "register"))
	deps.accessRepo.readable[id] = false
	require.ErrorIs(t, deps.service.RequireDocumentAction(id, "register"), models.ErrForbidden)
	deps.auth.err = models.ErrUnauthorized
	require.ErrorIs(t, deps.service.RequireDomainRead(), models.ErrUnauthorized)
}
func TestDocumentLinkNeedsOneRegistrationRightAndBothReaders(t *testing.T) {
	deps := setupDocumentAccessService(t, documentAccessUser(false, nil), allowDocumentActions(models.DocumentKindIncomingLetter, "register"))
	a, b := uuid.New(), uuid.New()
	deps.docRepo.docs[a] = documentAccessDoc(a, uuid.New(), models.DocumentKindIncomingLetter)
	deps.docRepo.docs[b] = documentAccessDoc(b, uuid.New(), models.DocumentKindOutgoingLetter)
	deps.accessRepo.readable[a] = true
	deps.accessRepo.readable[b] = true
	require.NoError(t, deps.service.RequireLink(a, b))
	require.NoError(t, deps.service.RequireLink(b, a))
	deps.accessRepo.readable[b] = false
	require.ErrorIs(t, deps.service.RequireLink(a, b), models.ErrForbidden)
	deps.accessRepo.readable[b] = true
	deps.accessRepo.allowed = nil
	require.ErrorIs(t, deps.service.RequireLink(a, b), models.ErrForbidden)
}
func TestDocumentVisibilityFailuresAreNotIgnored(t *testing.T) {
	deps := setupDocumentAccessService(t, documentAccessUser(false, nil), nil)
	id := uuid.New()
	deps.docRepo.docs[id] = documentAccessDoc(id, uuid.New(), models.DocumentKindIncomingLetter)
	cause := errors.New("visibility failed")
	deps.accessRepo.visibilityErr = cause
	require.ErrorIs(t, deps.service.RequireReadAnyType(id), cause)
	_, err := deps.service.ResolveReadableDocuments([]uuid.UUID{id})
	require.ErrorIs(t, err, cause)
	deps.service.visibility = nil
	require.ErrorIs(t, deps.service.RequireReadAnyType(id), models.ErrForbidden)
}
