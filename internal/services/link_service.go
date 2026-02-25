package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/dto"
	"docflow/internal/models"
)

type LinkService struct {
	repo            LinkStore
	incomingDocRepo IncomingDocStore
	outgoingDocRepo OutgoingDocStore
	authService     *AuthService
}

func NewLinkService(
	repo LinkStore,
	incomingDocRepo IncomingDocStore,
	outgoingDocRepo OutgoingDocStore,
	authService *AuthService,
) *LinkService {
	return &LinkService{
		repo:            repo,
		incomingDocRepo: incomingDocRepo,
		outgoingDocRepo: outgoingDocRepo,
		authService:     authService,
	}
}

// LinkDocuments — создать связь между двумя документами
func (s *LinkService) LinkDocuments(sourceIDStr, targetIDStr, sourceType, targetType, linkType string) (*dto.DocumentLink, error) {
	userIDStr := s.authService.GetCurrentUserID()
	if userIDStr == "" {
		return nil, fmt.Errorf("unauthorized")
	}
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

	link := &models.DocumentLink{
		SourceType: sourceType,
		SourceID:   sourceID,
		TargetType: targetType,
		TargetID:   targetID,
		LinkType:   linkType,
		CreatedBy:  userID,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(context.Background(), link); err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	return dto.MapDocumentLink(link), nil
}

// UnlinkDocument — удалить связь
func (s *LinkService) UnlinkDocument(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	return s.repo.Delete(context.Background(), id)
}

// GetDocumentLinks — получить прямые связи документа
func (s *LinkService) GetDocumentLinks(docIDStr string) ([]dto.DocumentLink, error) {
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	res, err := s.repo.GetByDocumentID(context.Background(), docID)
	return dto.MapDocumentLinks(res), err
}

// GetDocumentFlow — получить данные графа для визуализации
func (s *LinkService) GetDocumentFlow(rootIDStr string) (*models.GraphData, error) {
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

		if docType == "incoming" {
			doc, err := s.incomingDocRepo.GetByID(id)
			if err == nil && doc != nil {
				label = doc.IncomingNumber
				subject = doc.Subject
				dateStr = doc.IncomingDate.Format("02.01.2006")
				sender = doc.SenderOrgName
				if sender == "" {
					sender = "Неизвестно"
				}
				recipient = doc.RecipientOrgName // Обычно "Наша Организация"
			}
		} else if docType == "outgoing" {
			doc, err := s.outgoingDocRepo.GetByID(id)
			if err == nil && doc != nil {
				label = doc.OutgoingNumber
				subject = doc.Subject
				dateStr = doc.OutgoingDate.Format("02.01.2006")
				sender = doc.SenderOrgName // Обычно "Наша Организация"
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
