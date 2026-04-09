package services

import "github.com/google/uuid"

// DocumentAccessService инкапсулирует политику доступа к документному домену.
// Нужен как единая точка переиспользования для сервисов документов, файлов, журнала и связанных сущностей.
type DocumentAccessService struct {
	auth               *AuthService
	depRepo            DepartmentStore
	assignmentRepo     AssignmentStore
	acknowledgmentRepo AcknowledgmentStore
	incomingRepo       IncomingDocStore
	outgoingRepo       OutgoingDocStore
}

// NewDocumentAccessService создает сервис проверки доступа к документам.
func NewDocumentAccessService(
	auth *AuthService,
	depRepo DepartmentStore,
	assignmentRepo AssignmentStore,
	acknowledgmentRepo AcknowledgmentStore,
	incomingRepo IncomingDocStore,
	outgoingRepo OutgoingDocStore,
) *DocumentAccessService {
	return &DocumentAccessService{
		auth:               auth,
		depRepo:            depRepo,
		assignmentRepo:     assignmentRepo,
		acknowledgmentRepo: acknowledgmentRepo,
		incomingRepo:       incomingRepo,
		outgoingRepo:       outgoingRepo,
	}
}

// RequireDomainRead проверяет базовый доступ к document-domain.
func (s *DocumentAccessService) RequireDomainRead() error {
	return requireDocumentDomainReadRole(s.auth)
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
		documentType,
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
