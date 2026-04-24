package services

import (
	"context"
	"fmt"
	"sort"
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
	if !s.authService.IsAuthenticated() {
		return nil, models.ErrUnauthorized
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
	sourceDoc, targetDoc, err := s.access.ResolveLink(sourceID, targetID)
	if err != nil {
		return nil, err
	}
	link := &models.DocumentLink{
		SourceKind: sourceDoc.Kind,
		SourceID:   sourceID,
		TargetKind: targetDoc.Kind,
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
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	link, err := s.repo.GetByID(context.Background(), id)
	if err != nil {
		return err
	}
	if link == nil {
		return nil
	}
	if err := s.access.RequireLink(link.SourceID, link.TargetID); err != nil {
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
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	if err := s.access.RequireDocumentAction(docID, "link"); err != nil {
		return nil, err
	}
	res, err := s.repo.GetByDocumentID(context.Background(), docID)
	if err != nil {
		return nil, err
	}

	docIDs := make([]uuid.UUID, 0, len(res)*2)
	for _, link := range res {
		docIDs = append(docIDs, link.SourceID, link.TargetID)
	}
	readableDocs, err := s.access.ResolveReadableDocuments(docIDs)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.DocumentLink, 0, len(res))
	for _, link := range res {
		if _, ok := readableDocs[link.SourceID]; !ok {
			continue
		}
		if _, ok := readableDocs[link.TargetID]; !ok {
			continue
		}
		filtered = append(filtered, link)
	}

	return dto.MapDocumentLinks(filtered), nil
}

// GetDocumentFlow возвращает граф связей для документа, включая связанные узлы (документы) и ребра (связи) для визуализации.
func (s *LinkService) GetDocumentFlow(rootIDStr string) (*models.GraphData, error) {
	rootID, err := uuid.Parse(rootIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	if err := s.access.RequireDocumentAction(rootID, "link"); err != nil {
		return nil, err
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
		docIDs[l.SourceID] = string(l.SourceKind)
		docIDs[l.TargetID] = string(l.TargetKind)
	}

	readableDocs, err := s.access.ResolveReadableDocuments(mapKeys(docIDs))
	if err != nil {
		return nil, err
	}
	links = keepReadableGraphLinksReachableFromRoot(rootID, links, readableDocs)
	if len(links) == 0 {
		return &models.GraphData{Nodes: []models.GraphNode{}, Edges: []models.GraphEdge{}}, nil
	}

	docIDs = make(map[uuid.UUID]string)
	for _, l := range links {
		docIDs[l.SourceID] = string(l.SourceKind)
		docIDs[l.TargetID] = string(l.TargetKind)
	}

	// Получение деталей документов
	nodes := []models.GraphNode{}

	// Получение информации о документах
	// Создаёт N запросов; можно оптимизировать через WHERE IN, но граф обычно маленький (< 20 узлов)
	for id, docType := range docIDs {
		var label, subject, dateStr, sender, recipient string

		switch models.NormalizeDocumentKind(docType) {
		case models.DocumentKindIncomingLetter:
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
		case models.DocumentKindOutgoingLetter:
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
			KindCode:  docType,
			Subject:   subject,
			Date:      dateStr,
			Sender:    sender,
			Recipient: recipient,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Date != nodes[j].Date {
			return nodes[i].Date < nodes[j].Date
		}
		if nodes[i].Label != nodes[j].Label {
			return nodes[i].Label < nodes[j].Label
		}
		if nodes[i].KindCode != nodes[j].KindCode {
			return nodes[i].KindCode < nodes[j].KindCode
		}
		return nodes[i].ID < nodes[j].ID
	})

	edges := []models.GraphEdge{}
	for _, l := range links {
		edges = append(edges, models.GraphEdge{
			ID:     l.ID.String(),
			Source: l.SourceID.String(),
			Target: l.TargetID.String(),
			Label:  l.LinkType,
		})
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		if edges[i].Target != edges[j].Target {
			return edges[i].Target < edges[j].Target
		}
		if edges[i].Label != edges[j].Label {
			return edges[i].Label < edges[j].Label
		}
		return edges[i].ID < edges[j].ID
	})

	return &models.GraphData{Nodes: nodes, Edges: edges}, nil
}

func mapKeys(values map[uuid.UUID]string) []uuid.UUID {
	keys := make([]uuid.UUID, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func keepReadableGraphLinksReachableFromRoot(rootID uuid.UUID, links []models.DocumentLink, readableDocs map[uuid.UUID]*models.Document) []models.DocumentLink {
	filtered := make([]models.DocumentLink, 0, len(links))
	for _, link := range links {
		if _, ok := readableDocs[link.SourceID]; !ok {
			continue
		}
		if _, ok := readableDocs[link.TargetID]; !ok {
			continue
		}
		filtered = append(filtered, link)
	}

	reachable := map[uuid.UUID]struct{}{rootID: {}}
	changed := true
	for changed {
		changed = false
		for _, link := range filtered {
			if _, ok := reachable[link.SourceID]; ok {
				if _, targetReachable := reachable[link.TargetID]; !targetReachable {
					reachable[link.TargetID] = struct{}{}
					changed = true
				}
			}
			if _, ok := reachable[link.TargetID]; ok {
				if _, sourceReachable := reachable[link.SourceID]; !sourceReachable {
					reachable[link.SourceID] = struct{}{}
					changed = true
				}
			}
		}
	}

	result := make([]models.DocumentLink, 0, len(filtered))
	for _, link := range filtered {
		if _, ok := reachable[link.SourceID]; !ok {
			continue
		}
		if _, ok := reachable[link.TargetID]; !ok {
			continue
		}
		result = append(result, link)
	}

	return result
}
