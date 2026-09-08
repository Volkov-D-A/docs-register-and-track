package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/mocks"
	serverservices "github.com/Volkov-D-A/docs-register-and-track/internal/server/services"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

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

func (s *documentAccessAcknowledgmentStore) GetAllActive(filter models.AcknowledgmentFilter) ([]models.Acknowledgment, error) {
	return nil, nil
}

func (s *documentAccessAcknowledgmentStore) GetUsersByAcknowledgmentID(ackID uuid.UUID) ([]models.AcknowledgmentUser, error) {
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

type documentAccessTestDeps struct {
	auth       *testPrincipal
	userRepo   *mocks.UserStore
	accessRepo *kindActionDocumentAccessStore
	depRepo    *documentAccessDepartmentStore
	assignRepo *documentAccessAssignmentStore
	ackRepo    *documentAccessAcknowledgmentStore
	docRepo    *documentAccessDocumentStore
	subRepo    *userSubstitutionStoreStub
	service    *serverservices.DocumentAccessService
	user       *models.User
}

func setupDocumentAccessService(t *testing.T, user *models.User, allowed map[models.DocumentKind]map[string]bool) *documentAccessTestDeps {
	t.Helper()

	userRepo := mocks.NewUserStore(t)
	auth := newTestPrincipal(userRepo)
	if user != nil {
		auth.currentUserID = user.ID
		userRepo.On("GetByID", user.ID).Return(user, nil).Maybe()
	}

	accessRepo := &kindActionDocumentAccessStore{allowed: allowed}
	depRepo := &documentAccessDepartmentStore{}
	assignRepo := &documentAccessAssignmentStore{accessible: map[uuid.UUID]struct{}{}}
	ackRepo := &documentAccessAcknowledgmentStore{accessible: map[uuid.UUID]struct{}{}}
	docRepo := &documentAccessDocumentStore{docs: map[uuid.UUID]models.Document{}}
	subRepo := &userSubstitutionStoreStub{}
	service := serverservices.NewDocumentAccessService(auth, depRepo, assignRepo, ackRepo, accessRepo, docRepo, subRepo)

	return &documentAccessTestDeps{
		auth:       auth,
		userRepo:   userRepo,
		accessRepo: accessRepo,
		depRepo:    depRepo,
		assignRepo: assignRepo,
		ackRepo:    ackRepo,
		docRepo:    docRepo,
		subRepo:    subRepo,
		service:    service,
		user:       user,
	}
}
