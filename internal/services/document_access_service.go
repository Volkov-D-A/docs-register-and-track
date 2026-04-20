package services

import (
	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// DocumentAccessService инкапсулирует политику доступа к документному домену.
// Нужен как единая точка переиспользования для сервисов документов, файлов, журнала и связанных сущностей.
type DocumentAccessService struct {
	auth               *AuthService
	depRepo            DepartmentStore
	assignmentRepo     AssignmentStore
	acknowledgmentRepo AcknowledgmentStore
	accessRepo         DocumentAccessStore
	documentRepo       DocumentStore
	incomingRepo       IncomingDocStore
	outgoingRepo       OutgoingDocStore
}

type DocumentReadScope struct {
	Restricted             bool
	AllowedNomenclatureIDs []string
	AccessibleByUserID     string
}

// NewDocumentAccessService создает сервис проверки доступа к документам.
func NewDocumentAccessService(
	auth *AuthService,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	accessRepo DocumentAccessStore,
	documentRepo DocumentStore,
	incomingRepo IncomingDocStore,
	outgoingRepo OutgoingDocStore,
) *DocumentAccessService {
	return &DocumentAccessService{
		auth:               auth,
		depRepo:            depRepo,
		assignmentRepo:     assignmentRepo,
		acknowledgmentRepo: acknowledgmentRepo,
		accessRepo:         accessRepo,
		documentRepo:       documentRepo,
		incomingRepo:       incomingRepo,
		outgoingRepo:       outgoingRepo,
	}
}

// RequireDomainRead проверяет базовый доступ к document-domain.
func (s *DocumentAccessService) RequireDomainRead() error {
	if !s.auth.IsAuthenticated() {
		return models.ErrUnauthorized
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
	if s.documentRepo != nil {
		return s.documentRepo.GetByID(documentID)
	}

	if s.incomingRepo != nil {
		incomingDoc, err := s.incomingRepo.GetByID(documentID)
		if err != nil {
			return nil, err
		}
		if incomingDoc != nil {
			return &models.Document{
				ID:                 incomingDoc.ID,
				Kind:               models.DocumentKindIncomingLetter,
				NomenclatureID:     incomingDoc.NomenclatureID,
				RegistrationNumber: incomingDoc.IncomingNumber,
				RegistrationDate:   incomingDoc.IncomingDate,
				DocumentTypeID:     incomingDoc.DocumentTypeID,
				Content:            incomingDoc.Content,
				PagesCount:         incomingDoc.PagesCount,
				CreatedBy:          incomingDoc.CreatedBy,
				CreatedAt:          incomingDoc.CreatedAt,
				UpdatedAt:          incomingDoc.UpdatedAt,
			}, nil
		}
	}

	if s.outgoingRepo != nil {
		outgoingDoc, err := s.outgoingRepo.GetByID(documentID)
		if err != nil {
			return nil, err
		}
		if outgoingDoc != nil {
			return &models.Document{
				ID:                 outgoingDoc.ID,
				Kind:               models.DocumentKindOutgoingLetter,
				NomenclatureID:     outgoingDoc.NomenclatureID,
				RegistrationNumber: outgoingDoc.OutgoingNumber,
				RegistrationDate:   outgoingDoc.OutgoingDate,
				DocumentTypeID:     outgoingDoc.DocumentTypeID,
				Content:            outgoingDoc.Content,
				PagesCount:         outgoingDoc.PagesCount,
				CreatedBy:          outgoingDoc.CreatedBy,
				CreatedAt:          outgoingDoc.CreatedAt,
				UpdatedAt:          outgoingDoc.UpdatedAt,
			}, nil
		}
	}

	return nil, nil
}

// RequireExists проверяет существование документа.
func (s *DocumentAccessService) RequireExists(documentID uuid.UUID) (*models.Document, error) {
	doc, err := s.GetDocument(documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, models.NewBadRequest("документ не найден")
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

func (s *DocumentAccessService) hasImplicitReadAccess(doc *models.Document) (bool, error) {
	if doc == nil {
		return false, models.NewBadRequest("документ не найден")
	}

	isParticipant, err := s.isCurrentUserDocumentParticipant()
	if err != nil {
		return false, err
	}
	if !isParticipant {
		return false, nil
	}

	ok, err := hasExecutorNomenclatureAccess(s.auth, s.depRepo, doc.NomenclatureID)
	if err == nil && ok {
		return true, nil
	}

	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return false, err
	}

	if s.assignmentRepo != nil {
		ok, err := s.assignmentRepo.HasDocumentAccess(currentUserID, doc.ID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	if s.acknowledgmentRepo != nil {
		ok, err := s.acknowledgmentRepo.HasDocumentAccess(currentUserID, doc.ID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

func (s *DocumentAccessService) canReadResolved(doc *models.Document) (bool, error) {
	if doc == nil {
		return false, models.NewBadRequest("документ не найден")
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
	if !s.auth.IsAuthenticated() {
		return models.ErrUnauthorized
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

func (s *DocumentAccessService) ResolveReadScope(kind models.DocumentKind) (*DocumentReadScope, error) {
	if err := s.RequireDomainRead(); err != nil {
		return nil, err
	}

	allowed, err := s.hasPermission(kind, "read")
	if err != nil {
		return nil, err
	}
	if allowed {
		return &DocumentReadScope{}, nil
	}

	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
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
		return &DocumentReadScope{Restricted: true, AccessibleByUserID: userID.String()}, nil
	}

	return &DocumentReadScope{
		Restricted:             true,
		AccessibleByUserID:     userID.String(),
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
	if !isParticipant {
		return models.ErrForbidden
	}

	ok, err := hasExecutorNomenclatureAccess(s.auth, s.depRepo, nomenclatureID)
	if err == nil && ok {
		return nil
	}

	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}
	if s.assignmentRepo != nil {
		ok, err := s.assignmentRepo.HasDocumentAccess(currentUserID, documentID)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	if s.acknowledgmentRepo != nil {
		ok, err := s.acknowledgmentRepo.HasDocumentAccess(currentUserID, documentID)
		if err != nil {
			return err
		}
		if ok {
			return nil
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
