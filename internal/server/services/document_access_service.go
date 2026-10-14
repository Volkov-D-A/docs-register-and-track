package services

import (
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
	"github.com/google/uuid"
)

// DocumentAccessService uses the same database visibility closure as controls.
// Executor eligibility and authorship never grant or restrict departmental read.
type DocumentAccessService struct {
	auth         ports.DocumentAccessPrincipal
	accessRepo   ports.DocumentAccessStore
	documentRepo ports.DocumentStore
	visibility   ports.DocumentVisibilityStore
}

func NewDocumentAccessService(auth ports.DocumentAccessPrincipal, access ports.DocumentAccessStore, documents ports.DocumentStore) *DocumentAccessService {
	visibility, _ := access.(ports.DocumentVisibilityStore)
	return &DocumentAccessService{auth: auth, accessRepo: access, documentRepo: documents, visibility: visibility}
}
func (s *DocumentAccessService) RequireDomainRead() error { return s.auth.RequireAuthenticated() }
func (s *DocumentAccessService) GetDocument(id uuid.UUID) (*models.Document, error) {
	return s.documentRepo.GetByID(id)
}
func uniqueDocumentIDs(ids []uuid.UUID) []uuid.UUID {
	result := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if id != uuid.Nil && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}
func (s *DocumentAccessService) RequireExists(id uuid.UUID) (*models.Document, error) {
	doc, err := s.GetDocument(id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, models.NewNotFound("документ не найден")
	}
	return doc, nil
}
func (s *DocumentAccessService) getCurrentSubjects() (string, string, error) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", models.ErrUnauthorized
	}
	department := ""
	if user.Department != nil {
		department = user.Department.ID
	}
	return department, user.ID, nil
}
func (s *DocumentAccessService) hasPermission(kind models.DocumentKind, action string) (bool, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return false, err
	}
	if s.accessRepo == nil || !kind.SupportsAction(action) {
		return false, nil
	}
	department, user, err := s.getCurrentSubjects()
	if err != nil {
		return false, err
	}
	return s.accessRepo.HasPermission(string(kind), action, department, user)
}

// Read and event delivery use only the current identity.
func (s *DocumentAccessService) GetCurrentReadSubjectIDs() ([]uuid.UUID, error) {
	id, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	return []uuid.UUID{id}, nil
}
func (s *DocumentAccessService) GetDocumentKindsWithAction(action string) ([]models.DocumentKind, error) {
	kinds := []models.DocumentKind{}
	for _, spec := range models.AllDocumentKindSpecs() {
		allowed, err := s.hasPermission(spec.Code, action)
		if err != nil {
			return nil, err
		}
		if allowed {
			kinds = append(kinds, spec.Code)
		}
	}
	return kinds, nil
}
func (s *DocumentAccessService) RequireCreate(kind models.DocumentKind) error {
	allowed, err := s.hasPermission(kind, string(models.DocumentActionRegister))
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}
	return nil
}
func (s *DocumentAccessService) ResolveReadScope(_ models.DocumentKind) (*models.DocumentAccessScope, error) {
	if err := s.RequireDomainRead(); err != nil {
		return nil, err
	}
	id, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	return &models.DocumentAccessScope{Restricted: true, UserID: id.String()}, nil
}
func (s *DocumentAccessService) RequireReadResolved(doc *models.Document) error {
	if err := s.RequireDomainRead(); err != nil {
		return err
	}
	if doc == nil {
		return models.NewNotFound("документ не найден")
	}
	if s.visibility == nil {
		return models.ErrForbidden
	}
	id, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}
	allowed, err := s.visibility.CanReadDocument(id, doc.ID)
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}
	return nil
}
func (s *DocumentAccessService) ResolveReadableDocuments(ids []uuid.UUID) (map[uuid.UUID]*models.Document, error) {
	result := map[uuid.UUID]*models.Document{}
	ids = uniqueDocumentIDs(ids)
	if len(ids) == 0 {
		return result, nil
	}
	if err := s.RequireDomainRead(); err != nil {
		return nil, err
	}
	if s.visibility == nil {
		return result, nil
	}
	user, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}
	allowed, err := s.visibility.GetReadableDocumentIDs(user, ids)
	if err != nil {
		return nil, err
	}
	selected := []uuid.UUID{}
	for _, id := range ids {
		if allowed[id] {
			selected = append(selected, id)
		}
	}
	if len(selected) == 0 {
		return result, nil
	}
	documents, err := s.documentRepo.GetByIDs(selected)
	if err != nil {
		return nil, err
	}
	for i := range documents {
		result[documents[i].ID] = &documents[i]
	}
	return result, nil
}
func (s *DocumentAccessService) RequireRead(_ string, id uuid.UUID) error {
	return s.RequireReadAnyType(id)
}
func (s *DocumentAccessService) RequireReadAnyType(id uuid.UUID) error {
	doc, err := s.RequireExists(id)
	if err != nil {
		return err
	}
	return s.RequireReadResolved(doc)
}
func (s *DocumentAccessService) RequireDocumentAction(id uuid.UUID, action string) error {
	doc, err := s.RequireExists(id)
	if err != nil {
		return err
	}
	if err = s.RequireReadResolved(doc); err != nil {
		return err
	}
	allowed, err := s.hasPermission(doc.Kind, action)
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}
	return nil
}
func (s *DocumentAccessService) RequireLink(source, target uuid.UUID) error {
	_, _, err := s.ResolveLink(source, target)
	return err
}
func (s *DocumentAccessService) ResolveLink(source, target uuid.UUID) (*models.Document, *models.Document, error) {
	a, err := s.RequireExists(source)
	if err != nil {
		return nil, nil, err
	}
	b, err := s.RequireExists(target)
	if err != nil {
		return nil, nil, err
	}
	if err = s.RequireReadResolved(a); err != nil {
		return nil, nil, err
	}
	if err = s.RequireReadResolved(b); err != nil {
		return nil, nil, err
	}
	for _, doc := range []*models.Document{a, b} {
		allowed, err := s.hasPermission(doc.Kind, string(models.DocumentActionRegister))
		if err != nil {
			return nil, nil, err
		}
		if allowed {
			return a, b, nil
		}
	}
	return nil, nil, models.ErrForbidden
}
func (s *DocumentAccessService) RequireViewJournal(id uuid.UUID) error {
	return s.RequireReadAnyType(id)
}

// The removed execution workflow must not be inferred from ordinary read.
// This temporary dependency is removed with the legacy attachment endpoints.
func (s *DocumentAccessService) HasAssignmentAccess(_ uuid.UUID) (bool, error) {
	return false, s.RequireDomainRead()
}
func (s *DocumentAccessService) CollectUserIDsWithDocumentAction(users ports.UserStore, kind, action string, excluded map[uuid.UUID]struct{}) ([]uuid.UUID, error) {
	if users == nil || s.accessRepo == nil || !models.DocumentKind(kind).SupportsAction(action) {
		return nil, nil
	}
	all, err := users.GetAll()
	if err != nil {
		return nil, err
	}
	result := []uuid.UUID{}
	for _, user := range all {
		if !user.IsActive {
			continue
		}
		if _, skip := excluded[user.ID]; skip {
			continue
		}
		department := ""
		if user.DepartmentID != nil {
			department = user.DepartmentID.String()
		}
		allowed, err := s.accessRepo.HasPermission(kind, action, department, user.ID.String())
		if err != nil {
			return nil, err
		}
		if allowed {
			result = AppendUniqueUserID(result, user.ID)
		}
	}
	return result, nil
}
