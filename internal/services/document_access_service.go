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

	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return err
	}
	for _, role := range user.Roles {
		if role == "clerk" || role == "executor" {
			return nil
		}
	}

	return models.ErrForbidden
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

func (s *DocumentAccessService) getCurrentSubjects() (roles []string, departmentID, userID string, err error) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return nil, "", "", err
	}
	if user == nil {
		return nil, "", "", models.ErrUnauthorized
	}

	departmentID = ""
	if user.Department != nil {
		departmentID = user.Department.ID
	}

	return user.Roles, departmentID, user.ID, nil
}

func (s *DocumentAccessService) hasRole(role string) (bool, error) {
	user, err := s.auth.GetCurrentUser()
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, models.ErrUnauthorized
	}
	for _, currentRole := range user.Roles {
		if currentRole == role {
			return true, nil
		}
	}
	return false, nil
}

func (s *DocumentAccessService) hasPermission(kind models.DocumentKind, action string) (bool, error) {
	if s.accessRepo == nil {
		user, err := s.auth.GetCurrentUser()
		if err != nil {
			return false, err
		}
		if user == nil {
			return false, models.ErrUnauthorized
		}

		hasRole := func(expected string) bool {
			for _, role := range user.Roles {
				if role == expected {
					return true
				}
			}
			return false
		}

		if hasRole("clerk") {
			return true, nil
		}

		if hasRole("executor") {
			switch action {
			case "read", "upload", "view_journal":
				return true, nil
			default:
				return false, nil
			}
		}

		return false, nil
	}

	roles, departmentID, userID, err := s.getCurrentSubjects()
	if err != nil {
		return false, err
	}

	return s.accessRepo.HasPermission(string(kind), action, roles, departmentID, userID)
}

func (s *DocumentAccessService) getVisibilityChannels(kind models.DocumentKind) ([]string, error) {
	if s.accessRepo == nil {
		user, err := s.auth.GetCurrentUser()
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, models.ErrUnauthorized
		}
		for _, role := range user.Roles {
			if role == "executor" {
				return []string{"department_nomenclature", "assignment", "acknowledgment"}, nil
			}
			if role == "clerk" {
				return nil, nil
			}
		}
		return nil, nil
	}

	roles, departmentID, userID, err := s.getCurrentSubjects()
	if err != nil {
		return nil, err
	}

	return s.accessRepo.GetVisibilityChannels(string(kind), roles, departmentID, userID)
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

func (s *DocumentAccessService) hasChannel(channels []string, expected string) bool {
	for _, channel := range channels {
		if channel == expected {
			return true
		}
	}
	return false
}

func (s *DocumentAccessService) canReadResolved(doc *models.Document) (bool, error) {
	if doc == nil {
		return false, models.NewBadRequest("документ не найден")
	}

	allowed, err := s.hasPermission(doc.Kind, "read")
	if err != nil || !allowed {
		return allowed, err
	}

	channels, err := s.getVisibilityChannels(doc.Kind)
	if err != nil {
		return false, err
	}

	// Если для пользователя не настроены специальные каналы видимости,
	// достаточно самого разрешения на чтение.
	if len(channels) == 0 {
		return true, nil
	}

	if s.hasChannel(channels, "department_nomenclature") {
		ok, err := hasExecutorNomenclatureAccess(s.auth, s.depRepo, doc.NomenclatureID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return false, err
	}

	if s.hasChannel(channels, "assignment") {
		ok, err := s.assignmentRepo.HasDocumentAccess(currentUserID, doc.ID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	if s.hasChannel(channels, "acknowledgment") {
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
	if !allowed {
		return nil, models.ErrForbidden
	}

	channels, err := s.getVisibilityChannels(kind)
	if err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return &DocumentReadScope{}, nil
	}

	userID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return nil, err
	}

	scope := &DocumentReadScope{
		Restricted:         true,
		AccessibleByUserID: userID.String(),
	}
	if s.hasChannel(channels, "department_nomenclature") {
		scope.AllowedNomenclatureIDs, err = s.getDepartmentNomenclatureIDs()
		if err != nil {
			return nil, err
		}
	}

	return scope, nil
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

	if s.accessRepo == nil {
		isClerk, err := s.hasRole("clerk")
		if err != nil {
			return err
		}
		if isClerk {
			return nil
		}

		return requireExecutorDocumentAccess(
			s.auth,
			s.depRepo,
			s.assignmentRepo,
			s.acknowledgmentRepo,
			documentID,
			nomenclatureID,
		)
	}

	kind := models.NormalizeDocumentKind(documentKind)

	allowed, err := s.hasPermission(kind, "read")
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrForbidden
	}

	channels, err := s.getVisibilityChannels(kind)
	if err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}

	if s.hasChannel(channels, "department_nomenclature") {
		ok, err := hasExecutorNomenclatureAccess(s.auth, s.depRepo, nomenclatureID)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}

	currentUserID, err := s.auth.GetCurrentUserUUID()
	if err != nil {
		return err
	}
	if s.hasChannel(channels, "assignment") {
		ok, err := s.assignmentRepo.HasDocumentAccess(currentUserID, documentID)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	if s.hasChannel(channels, "acknowledgment") {
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
	if s.accessRepo == nil {
		if !s.auth.IsAuthenticated() {
			return models.ErrUnauthorized
		}
		user, err := s.auth.GetCurrentUser()
		if err != nil {
			return err
		}
		for _, role := range user.Roles {
			if role == "clerk" {
				return nil
			}
			if role == "executor" {
				switch action {
				case "read", "upload", "view_journal":
					return nil
				}
			}
		}
		return models.ErrForbidden
	}

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
	if s.accessRepo == nil {
		if !s.auth.IsAuthenticated() {
			return models.ErrUnauthorized
		}
		isClerk, err := s.hasRole("clerk")
		if err != nil {
			return err
		}
		if !isClerk {
			return models.ErrForbidden
		}
		return nil
	}

	sourceDoc, err := s.RequireExists(sourceID)
	if err != nil {
		return err
	}
	targetDoc, err := s.RequireExists(targetID)
	if err != nil {
		return err
	}

	if err := s.RequireReadResolved(sourceDoc); err != nil {
		return err
	}
	if err := s.RequireReadResolved(targetDoc); err != nil {
		return err
	}

	for _, doc := range []*models.Document{sourceDoc, targetDoc} {
		allowed, err := s.hasPermission(doc.Kind, "link")
		if err != nil {
			return err
		}
		if !allowed {
			return models.ErrForbidden
		}
	}

	return nil
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
