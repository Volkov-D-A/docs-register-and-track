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
	documentRepo       DocumentStore
	incomingRepo       IncomingDocStore
	outgoingRepo       OutgoingDocStore
}

// NewDocumentAccessService создает сервис проверки доступа к документам.
func NewDocumentAccessService(
	auth *AuthService,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	documentRepo DocumentStore,
	incomingRepo IncomingDocStore,
	outgoingRepo OutgoingDocStore,
) *DocumentAccessService {
	return &DocumentAccessService{
		auth:               auth,
		depRepo:            depRepo,
		assignmentRepo:     assignmentRepo,
		acknowledgmentRepo: acknowledgmentRepo,
		documentRepo:       documentRepo,
		incomingRepo:       incomingRepo,
		outgoingRepo:       outgoingRepo,
	}
}

// RequireDomainRead проверяет базовый доступ к document-domain.
func (s *DocumentAccessService) RequireDomainRead() error {
	return requireDocumentDomainReadRole(s.auth)
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
				ID:             incomingDoc.ID,
				Kind:           models.DocumentKindIncoming,
				NomenclatureID: incomingDoc.NomenclatureID,
				DocumentTypeID: incomingDoc.DocumentTypeID,
				Content:        incomingDoc.Content,
				PagesCount:     incomingDoc.PagesCount,
				CreatedBy:      incomingDoc.CreatedBy,
				CreatedAt:      incomingDoc.CreatedAt,
				UpdatedAt:      incomingDoc.UpdatedAt,
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
				ID:             outgoingDoc.ID,
				Kind:           models.DocumentKindOutgoing,
				NomenclatureID: outgoingDoc.NomenclatureID,
				DocumentTypeID: outgoingDoc.DocumentTypeID,
				Content:        outgoingDoc.Content,
				PagesCount:     outgoingDoc.PagesCount,
				CreatedBy:      outgoingDoc.CreatedBy,
				CreatedAt:      outgoingDoc.CreatedAt,
				UpdatedAt:      outgoingDoc.UpdatedAt,
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

// RequireRead проверяет доступ к конкретному документу по его типу и ID.
func (s *DocumentAccessService) RequireRead(documentType string, documentID uuid.UUID) error {
	return requireDocumentReadAccess(
		s.auth,
		s.depRepo,
		s.assignmentRepo,
		s.acknowledgmentRepo,
		s.incomingRepo,
		s.outgoingRepo,
		documentType,
		documentID,
	)
}

// RequireResolvedRead проверяет доступ к уже загруженному документу без повторного чтения из репозитория.
func (s *DocumentAccessService) RequireResolvedRead(documentType string, documentID, nomenclatureID uuid.UUID) error {
	if err := s.RequireDomainRead(); err != nil {
		return err
	}

	if s.auth.HasActiveRole("clerk") {
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

// RequireReadAnyType проверяет доступ к документу, определяя тип по ID.
func (s *DocumentAccessService) RequireReadAnyType(documentID uuid.UUID) error {
	return requireAnyDocumentReadAccess(
		s.auth,
		s.depRepo,
		s.assignmentRepo,
		s.acknowledgmentRepo,
		s.incomingRepo,
		s.outgoingRepo,
		documentID,
	)
}
