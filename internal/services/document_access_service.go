package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentAccessPrincipal provides the request-local identity used by document
// access checks. AuthService implements it for desktop workflows; the server
// supplies an immutable principal for each HTTP request.
type DocumentAccessPrincipal interface {
	RequireAuthenticated() error
	GetCurrentUser() (*dto.User, error)
	GetCurrentUserUUID() (uuid.UUID, error)
}

// DocumentAccessService инкапсулирует политику доступа к документному домену.
// Нужен как единая точка переиспользования для сервисов документов, файлов, журнала и связанных сущностей.
type DocumentAccessService struct {
	auth               DocumentAccessPrincipal
	desktopAuth        *AuthService
	depRepo            DepartmentStore
	assignmentRepo     AssignmentStore
	acknowledgmentRepo AcknowledgmentStore
	substitutionRepo   UserSubstitutionStore
	accessRepo         DocumentAccessStore
	documentRepo       DocumentStore
}

// NewDocumentAccessService создает сервис проверки доступа к документам.
func NewDocumentAccessService(
	auth DocumentAccessPrincipal,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	accessRepo DocumentAccessStore,
	documentRepo DocumentStore,
	substitutionRepos ...UserSubstitutionStore,
) *DocumentAccessService {
	svc := &DocumentAccessService{
		auth:               auth,
		depRepo:            depRepo,
		assignmentRepo:     assignmentRepo,
		acknowledgmentRepo: acknowledgmentRepo,
		accessRepo:         accessRepo,
		documentRepo:       documentRepo,
	}
	if desktopAuth, ok := auth.(*AuthService); ok {
		svc.desktopAuth = desktopAuth
	}
	if len(substitutionRepos) > 0 {
		svc.substitutionRepo = substitutionRepos[0]
	}
	return svc
}

// RequireDomainRead проверяет базовый доступ к document-domain.
func (s *DocumentAccessService) RequireDomainRead() error {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return err
	}
	if s.accessRepo == nil {
		return models.ErrForbidden
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return err
	}
	if user != nil && user.IsDocumentParticipant {
		return nil
	}

	subjectIDs, err := s.getCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return err
	}
	if len(subjectIDs) > 1 {
		return nil
	}

	hasDocumentPermissions, err := s.hasAnyDocumentPermission()
	if err != nil {
		return err
	}
	if !hasDocumentPermissions {
		return models.ErrForbidden
	}
	return nil
}

// GetDocument возвращает корневой документ по ID.
func (s *DocumentAccessService) GetDocument(documentID uuid.UUID) (*models.Document, error) {
	return s.documentRepo.GetByID(documentID)
}

func (s *DocumentAccessService) getDocumentsByIDs(documentIDs []uuid.UUID) ([]models.Document, error) {
	uniqueIDs := uniqueDocumentIDs(documentIDs)
	if len(uniqueIDs) == 0 {
		return []models.Document{}, nil
	}

	return s.documentRepo.GetByIDs(uniqueIDs)
}

func uniqueDocumentIDs(documentIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(documentIDs))
	result := make([]uuid.UUID, 0, len(documentIDs))
	for _, documentID := range documentIDs {
		if documentID == uuid.Nil {
			continue
		}
		if _, ok := seen[documentID]; ok {
			continue
		}
		seen[documentID] = struct{}{}
		result = append(result, documentID)
	}
	return result
}

// RequireExists проверяет существование документа.
func (s *DocumentAccessService) RequireExists(documentID uuid.UUID) (*models.Document, error) {
	doc, err := s.GetDocument(documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, models.NewNotFound("документ не найден")
	}
	return doc, nil
}

func (s *DocumentAccessService) getCurrentSubjects() (departmentID, userID string, err error) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", models.ErrUnauthorized
	}

	departmentID = ""
	if user.Department != nil {
		departmentID = user.Department.ID
	}

	return departmentID, user.ID, nil
}

func (s *DocumentAccessService) isCurrentUserDocumentParticipant() (bool, error) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, models.ErrUnauthorized
	}
	return user.IsDocumentParticipant, nil
}

func (s *DocumentAccessService) hasAnyDocumentPermission() (bool, error) {
	for _, spec := range models.AllDocumentKindSpecs() {
		for _, action := range spec.SupportedActions {
			allowed, err := s.hasPermission(spec.Code, string(action))
			if err != nil {
				return false, err
			}
			if allowed {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *DocumentAccessService) hasPermission(kind models.DocumentKind, action string) (bool, error) {
	if s.accessRepo == nil {
		return false, models.ErrForbidden
	}

	departmentID, userID, err := s.getCurrentSubjects()
	if err != nil {
		return false, err
	}

	return s.accessRepo.HasPermission(string(kind), action, departmentID, userID)
}

func (s *DocumentAccessService) getCurrentUserAndSubstitutionSubjectIDs() ([]uuid.UUID, error) {
	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}

	ids := []uuid.UUID{currentUserID}
	if s.substitutionRepo == nil {
		return ids, nil
	}

	principalIDs, err := s.substitutionRepo.GetActivePrincipalIDs(currentUserID)
	if err != nil {
		return nil, err
	}
	seen := map[uuid.UUID]struct{}{currentUserID: {}}
	for _, principalID := range principalIDs {
		if principalID == uuid.Nil {
			continue
		}
		if _, ok := seen[principalID]; ok {
			continue
		}
		seen[principalID] = struct{}{}
		ids = append(ids, principalID)
	}
	return ids, nil
}

func uuidStrings(ids []uuid.UUID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != uuid.Nil {
			result = append(result, id.String())
		}
	}
	return result
}

func (s *DocumentAccessService) GetAvailableActions(kind models.DocumentKind) ([]string, error) {
	spec, ok := models.GetDocumentKindSpec(kind)
	if !ok {
		return nil, nil
	}

	actions := make([]string, 0, len(spec.SupportedActions))
	for _, action := range spec.SupportedActions {
		allowed, err := s.hasPermission(kind, string(action))
		if err != nil {
			return nil, err
		}
		if allowed {
			actions = append(actions, string(action))
		}
	}

	return actions, nil
}

func (s *DocumentAccessService) HasDocumentAction(kind models.DocumentKind, action string) (bool, error) {
	return s.hasPermission(kind, action)
}

func (s *DocumentAccessService) HasAnyDocumentAction(action string) (bool, error) {
	for _, spec := range models.AllDocumentKindSpecs() {
		allowed, err := s.hasPermission(spec.Code, action)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (s *DocumentAccessService) GetDocumentKindsWithAction(action string) ([]models.DocumentKind, error) {
	kinds := make([]models.DocumentKind, 0)
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

func (s *DocumentAccessService) getDepartmentNomenclatureIDs() ([]string, error) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}
	if user == nil || user.Department == nil || user.Department.ID == "" {
		return nil, nil
	}

	departmentID, err := uuid.Parse(user.Department.ID)
	if err != nil {
		return nil, nil
	}

	return s.depRepo.GetNomenclatureIDs(departmentID)
}

func (s *DocumentAccessService) hasDepartmentNomenclatureAccess(nomenclatureID uuid.UUID) (bool, error) {
	allowedNomenclatures, err := s.getDepartmentNomenclatureIDs()
	if err != nil {
		return false, err
	}

	nomenclatureIDStr := nomenclatureID.String()
	for _, allowedID := range allowedNomenclatures {
		if allowedID == nomenclatureIDStr {
			return true, nil
		}
	}

	return false, nil
}

func (s *DocumentAccessService) hasImplicitReadAccess(doc *models.Document) (bool, error) {
	if doc == nil {
		return false, models.NewNotFound("документ не найден")
	}

	isParticipant, err := s.isCurrentUserDocumentParticipant()
	if err != nil {
		return false, err
	}

	subjectIDs, err := s.getCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return false, err
	}

	if isParticipant {
		ok, err := s.hasDepartmentNomenclatureAccess(doc.NomenclatureID)
		if err == nil && ok {
			return true, nil
		}
	} else if len(subjectIDs) <= 1 {
		return false, nil
	}

	if s.assignmentRepo != nil {
		for _, subjectID := range subjectIDs {
			ok, err := s.assignmentRepo.HasDocumentAccess(subjectID, doc.ID)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}

	if s.acknowledgmentRepo != nil {
		for _, subjectID := range subjectIDs {
			ok, err := s.acknowledgmentRepo.HasDocumentAccess(subjectID, doc.ID)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *DocumentAccessService) canReadResolved(doc *models.Document) (bool, error) {
	if doc == nil {
		return false, models.NewNotFound("документ не найден")
	}

	allowed, err := s.hasPermission(doc.Kind, "read")
	if err != nil {
		return false, err
	}
	if allowed {
		return true, nil
	}

	return s.hasImplicitReadAccess(doc)
}

func (s *DocumentAccessService) RequireCreate(kind models.DocumentKind) error {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return err
	}

	allowed, err := s.hasPermission(kind, "create")
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}
	return nil
}

func (s *DocumentAccessService) ResolveReadScope(kind models.DocumentKind) (*models.DocumentAccessScope, error) {
	if err := s.RequireDomainRead(); err != nil {
		return nil, err
	}

	allowed, err := s.hasPermission(kind, "read")
	if err != nil {
		return nil, err
	}
	if allowed {
		return &models.DocumentAccessScope{}, nil
	}

	subjectIDs, err := s.getCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return nil, err
	}
	subjectIDStrings := uuidStrings(subjectIDs)
	accessibleByUserID := ""
	if len(subjectIDStrings) > 0 {
		accessibleByUserID = subjectIDStrings[0]
	}

	allowedNomenclatureIDs, err := s.getDepartmentNomenclatureIDs()
	if err != nil {
		return nil, err
	}

	isParticipant, err := s.isCurrentUserDocumentParticipant()
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return &models.DocumentAccessScope{Restricted: true, AccessibleByUserID: accessibleByUserID, AccessibleByUserIDs: subjectIDStrings}, nil
	}

	return &models.DocumentAccessScope{
		Restricted:             true,
		AccessibleByUserID:     accessibleByUserID,
		AccessibleByUserIDs:    subjectIDStrings,
		AllowedNomenclatureIDs: allowedNomenclatureIDs,
	}, nil
}

func (s *DocumentAccessService) RequireReadResolved(doc *models.Document) error {
	if err := s.RequireDomainRead(); err != nil {
		return err
	}

	allowed, err := s.canReadResolved(doc)
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}
	return nil
}

// ResolveReadableDocuments возвращает документы из переданного набора, доступные текущему пользователю на чтение.
func (s *DocumentAccessService) ResolveReadableDocuments(documentIDs []uuid.UUID) (map[uuid.UUID]*models.Document, error) {
	readable := make(map[uuid.UUID]*models.Document)
	uniqueIDs := uniqueDocumentIDs(documentIDs)
	if len(uniqueIDs) == 0 {
		return readable, nil
	}

	if err := s.RequireDomainRead(); err != nil {
		return nil, err
	}

	docs, err := s.getDocumentsByIDs(uniqueIDs)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return readable, nil
	}

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, models.ErrUnauthorized
	}

	userID := user.ID
	departmentID := ""
	if user.Department != nil {
		departmentID = user.Department.ID
	}
	subjectIDs, err := s.getCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return nil, err
	}

	allowedNomenclatures := make(map[uuid.UUID]struct{})
	if user.IsDocumentParticipant && s.depRepo != nil && user.Department != nil && user.Department.ID != "" {
		departmentUUID, err := uuid.Parse(user.Department.ID)
		if err == nil {
			nomenclatureIDs, err := s.depRepo.GetNomenclatureIDs(departmentUUID)
			if err != nil {
				return nil, err
			}
			for _, nomenclatureID := range nomenclatureIDs {
				parsedID, err := uuid.Parse(nomenclatureID)
				if err == nil {
					allowedNomenclatures[parsedID] = struct{}{}
				}
			}
		}
	}

	assignmentAccessibleDocuments := make(map[uuid.UUID]struct{})
	acknowledgmentAccessibleDocuments := make(map[uuid.UUID]struct{})
	if user.IsDocumentParticipant || len(subjectIDs) > 1 {
		for _, subjectID := range subjectIDs {
			if s.assignmentRepo != nil {
				ids, err := s.assignmentRepo.GetAccessibleDocumentIDs(subjectID, uniqueIDs)
				if err != nil {
					return nil, err
				}
				for documentID := range ids {
					assignmentAccessibleDocuments[documentID] = struct{}{}
				}
			}

			if s.acknowledgmentRepo != nil {
				ids, err := s.acknowledgmentRepo.GetAccessibleDocumentIDs(subjectID, uniqueIDs)
				if err != nil {
					return nil, err
				}
				for documentID := range ids {
					acknowledgmentAccessibleDocuments[documentID] = struct{}{}
				}
			}
		}
	}

	readPermissionByKind := make(map[models.DocumentKind]bool)
	for i := range docs {
		doc := &docs[i]

		allowed, ok := readPermissionByKind[doc.Kind]
		if !ok {
			allowed, err = s.accessRepo.HasPermission(string(doc.Kind), "read", departmentID, userID)
			if err != nil {
				return nil, err
			}
			readPermissionByKind[doc.Kind] = allowed
		}
		if allowed {
			readable[doc.ID] = doc
			continue
		}

		if !user.IsDocumentParticipant && len(subjectIDs) <= 1 {
			continue
		}

		if user.IsDocumentParticipant {
			if _, ok := allowedNomenclatures[doc.NomenclatureID]; ok {
				readable[doc.ID] = doc
				continue
			}
		}

		if s.assignmentRepo != nil {
			if _, ok := assignmentAccessibleDocuments[doc.ID]; ok {
				readable[doc.ID] = doc
				continue
			}
		}

		if s.acknowledgmentRepo != nil {
			if _, ok := acknowledgmentAccessibleDocuments[doc.ID]; ok {
				readable[doc.ID] = doc
			}
		}
	}

	return readable, nil
}

// RequireRead проверяет доступ к конкретному документу по его виду и ID.
func (s *DocumentAccessService) RequireRead(documentKind string, documentID uuid.UUID) error {
	doc, err := s.RequireExists(documentID)
	if err != nil {
		return err
	}
	return s.RequireReadResolved(doc)
}

// RequireResolvedRead проверяет доступ к уже загруженному документу без повторного чтения из репозитория.
func (s *DocumentAccessService) RequireResolvedRead(documentKind string, documentID, nomenclatureID uuid.UUID) error {
	if err := s.RequireDomainRead(); err != nil {
		return err
	}

	kind := models.NormalizeDocumentKind(documentKind)

	allowed, err := s.hasPermission(kind, "read")
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}

	isParticipant, err := s.isCurrentUserDocumentParticipant()
	if err != nil {
		return err
	}

	subjectIDs, err := s.getCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return err
	}

	if isParticipant {
		ok, err := s.hasDepartmentNomenclatureAccess(nomenclatureID)
		if err == nil && ok {
			return nil
		}
	} else if len(subjectIDs) <= 1 {
		return models.ErrForbidden
	}
	if s.assignmentRepo != nil {
		for _, subjectID := range subjectIDs {
			ok, err := s.assignmentRepo.HasDocumentAccess(subjectID, documentID)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}
	if s.acknowledgmentRepo != nil {
		for _, subjectID := range subjectIDs {
			ok, err := s.acknowledgmentRepo.HasDocumentAccess(subjectID, documentID)
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
		}
	}

	return models.ErrForbidden
}

func (s *DocumentAccessService) RequireDocumentAction(documentID uuid.UUID, action string) error {
	doc, err := s.RequireExists(documentID)
	if err != nil {
		return err
	}

	allowed, err := s.hasPermission(doc.Kind, action)
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}

	return s.RequireReadResolved(doc)
}

func (s *DocumentAccessService) RequireLink(sourceID, targetID uuid.UUID) error {
	_, _, err := s.ResolveLink(sourceID, targetID)
	return err
}

func (s *DocumentAccessService) HasAssignmentAccess(documentID uuid.UUID) (bool, error) {
	if err := s.RequireDomainRead(); err != nil {
		return false, err
	}
	if s.assignmentRepo == nil {
		return false, nil
	}

	subjectIDs, err := s.getCurrentUserAndSubstitutionSubjectIDs()
	if err != nil {
		return false, err
	}

	for _, subjectID := range subjectIDs {
		ok, err := s.assignmentRepo.HasDocumentAccess(subjectID, documentID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (s *DocumentAccessService) ResolveLink(sourceID, targetID uuid.UUID) (*models.Document, *models.Document, error) {
	sourceDoc, err := s.RequireExists(sourceID)
	if err != nil {
		return nil, nil, err
	}
	targetDoc, err := s.RequireExists(targetID)
	if err != nil {
		return nil, nil, err
	}

	if err := s.RequireReadResolved(sourceDoc); err != nil {
		return nil, nil, err
	}
	if err := s.RequireReadResolved(targetDoc); err != nil {
		return nil, nil, err
	}

	for _, doc := range []*models.Document{sourceDoc, targetDoc} {
		allowed, err := s.hasPermission(doc.Kind, "link")
		if err != nil {
			return nil, nil, err
		}
		if !allowed {
			return nil, nil, models.ErrForbidden
		}
	}

	return sourceDoc, targetDoc, nil
}

func (s *DocumentAccessService) RequireViewJournal(documentID uuid.UUID) error {
	if s.accessRepo == nil {
		return s.RequireDomainRead()
	}

	doc, err := s.RequireExists(documentID)
	if err != nil {
		return err
	}

	allowed, err := s.hasPermission(doc.Kind, "view_journal")
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}

	return s.RequireReadResolved(doc)
}

// RequireReadAnyType проверяет доступ к документу, определяя тип по ID.
func (s *DocumentAccessService) RequireReadAnyType(documentID uuid.UUID) error {
	doc, err := s.RequireExists(documentID)
	if err != nil {
		return err
	}
	return s.RequireReadResolved(doc)
}
