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
	docRepo    *documentAccessDocumentStore
	service    *DocumentAccessService
	user       *models.User
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
	docRepo := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{}}
	service := NewDocumentAccessService(auth, accessRepo, docRepo)

	return &documentAccessTestDeps{
		auth:       auth,
		accessRepo: accessRepo,
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
