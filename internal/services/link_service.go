package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// LinkService предоставляет бизнес-логику для управления связями между документами.
type LinkService struct {
	repo            LinkStore
	incomingDocRepo IncomingDocStore
	outgoingDocRepo OutgoingDocStore
	access          *DocumentAccessService
	authService     *AuthService
	journal         *JournalService
}

// NewLinkService создает новый экземпляр LinkService.
func NewLinkService(
	repo LinkStore,
	incomingDocRepo IncomingDocStore,
	outgoingDocRepo OutgoingDocStore,
	access *DocumentAccessService,
	authService *AuthService,
	journal *JournalService,
) *LinkService {
	return &LinkService{
		repo:            repo,
		incomingDocRepo: incomingDocRepo,
		outgoingDocRepo: outgoingDocRepo,
		access:          access,
		authService:     authService,
		journal:         journal,
	}
}

// LinkDocuments создает связь указанного типа между двумя документами.
func (s *LinkService) LinkDocuments(sourceIDStr, targetIDStr, linkType string) (*dto.DocumentLink, error) {
	if err := requireClerkDocumentRole(s.authService); err != nil {
		return nil, err
	}
	userIDStr := s.authService.GetCurrentUserID()
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	sourceID, err := uuid.Parse(sourceIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid source ID: %w", err)
	}
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target ID: %w", err)
	}

	// Базовая валидация: запрет связи документа с самим собой
	if sourceID == targetID {
		return nil, fmt.Errorf("cannot link document to itself")
	}
	sourceDoc, err := s.access.RequireExists(sourceID)
	if err != nil {
		return nil, err
	}
	targetDoc, err := s.access.RequireExists(targetID)
	if err != nil {
		return nil, err
	}

	link := &models.DocumentLink{
		SourceType: string(sourceDoc.Kind),
		SourceID:   sourceID,
		TargetType: string(targetDoc.Kind),
		TargetID:   targetID,
		LinkType:   linkType,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(context.Background(), link); err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	// Логирование создания связи (для обоих документов)
	// Логирование создания связи (для обоих документов)
	// Упрощенный лог (без номеров, просто факт)
	// Упрощенный лог (без номеров, просто факт)
	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID: sourceID,
		UserID:     userID,
		Action:     "LINK_CREATE",
		Details:    "Создана связь с другим документом",
	})
	s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
		DocumentID: targetID,
		UserID:     userID,
		Action:     "LINK_CREATE",
		Details:    "Создана связь с другим документом",
	})

	return dto.MapDocumentLink(link), nil
}

// UnlinkDocument удаляет связь между документами по её ID.
func (s *LinkService) UnlinkDocument(idStr string) error {
	if err := requireClerkDocumentRole(s.authService); err != nil {
		return err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	link, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return err
	}

	err = s.repo.Delete(context.Background(), id)
	if err == nil {
		currentUserID, _ := s.authService.GetCurrentUserUUID()
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: link.SourceID,
			UserID:     currentUserID,
			Action:     "LINK_DELETE",
			Details:    "Удалена связь с документом",
		})
		s.journal.LogAction(context.Background(), models.CreateJournalEntryRequest{
			DocumentID: link.TargetID,
			UserID:     currentUserID,
			Action:     "LINK_DELETE",
			Details:    "Удалена связь с документом",
		})
	}
	return err
}

// GetDocumentLinks возвращает список всех прямых связей для указанного документа.
func (s *LinkService) GetDocumentLinks(docIDStr string) ([]dto.DocumentLink, error) {
	if err := requireClerkDocumentRole(s.authService); err != nil {
		return nil, err
	}

	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	res, err := s.repo.GetByDocumentID(context.Background(), docID)
	return dto.MapDocumentLinks(res), err
}

// GetDocumentFlow возвращает граф связей для документа, включая связанные узлы (документы) и ребра (связи) для визуализации.
func (s *LinkService) GetDocumentFlow(rootIDStr string) (*models.GraphData, error) {
	if err := requireClerkDocumentRole(s.authService); err != nil {
		return nil, err
	}

	rootID, err := uuid.Parse(rootIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	links, err := s.repo.GetGraph(context.Background(), rootID)
	if err != nil {
		return nil, err
	}

	// Собираем уникальные ID документов для получения детальной информации
	docIDs := make(map[uuid.UUID]string) // ID -> тип документа

	// Если связей нет — возвращаем пустой граф
	if len(links) == 0 {
		return &models.GraphData{Nodes: []models.GraphNode{}, Edges: []models.GraphEdge{}}, nil
	}

	for _, l := range links {
		docIDs[l.SourceID] = l.SourceType
		docIDs[l.TargetID] = l.TargetType
	}

	// Получение деталей документов
	nodes := []models.GraphNode{}

	// Получение информации о документах
	// Создаёт N запросов; можно оптимизировать через WHERE IN, но граф обычно маленький (< 20 узлов)
	for id, docType := range docIDs {
		var label, subject, dateStr, sender, recipient string

		switch docType {
		case "incoming":
			doc, err := s.incomingDocRepo.GetByID(id)
			if err == nil && doc != nil {
				label = doc.IncomingNumber
				subject = doc.Content
				dateStr = doc.IncomingDate.Format("02.01.2006")
				sender = doc.SenderOrgName
				if sender == "" {
					sender = "Неизвестно"
				}
			}
		case "outgoing":
			doc, err := s.outgoingDocRepo.GetByID(id)
			if err == nil && doc != nil {
				label = doc.OutgoingNumber
				subject = doc.Content
				dateStr = doc.OutgoingDate.Format("02.01.2006")
				recipient = doc.RecipientOrgName
				if recipient == "" {
					recipient = "Неизвестно"
				}
			}
		}

		if label == "" {
			label = "Неизвестно"
		}

		nodes = append(nodes, models.GraphNode{
			ID:        id.String(),
			Label:     label,
			Type:      docType,
			Subject:   subject,
			Date:      dateStr,
			Sender:    sender,
			Recipient: recipient,
		})
	}

	edges := []models.GraphEdge{}
	for _, l := range links {
		edges = append(edges, models.GraphEdge{
			ID:     l.ID.String(),
			Source: l.SourceID.String(),
			Target: l.TargetID.String(),
			Label:  l.LinkType,
		})
	}

	return &models.GraphData{Nodes: nodes, Edges: edges}, nil
}
