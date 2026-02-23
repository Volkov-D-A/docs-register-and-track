package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"docflow/internal/models"
	"docflow/internal/repository"
)

type LinkService struct {
	repo            *repository.LinkRepository
	incomingDocRepo *repository.IncomingDocumentRepository
	outgoingDocRepo *repository.OutgoingDocumentRepository
	authService     *AuthService
	ctx             context.Context
}

func NewLinkService(
	repo *repository.LinkRepository,
	incomingDocRepo *repository.IncomingDocumentRepository,
	outgoingDocRepo *repository.OutgoingDocumentRepository,
	authService *AuthService,
) *LinkService {
	return &LinkService{
		repo:            repo,
		incomingDocRepo: incomingDocRepo,
		outgoingDocRepo: outgoingDocRepo,
		authService:     authService,
	}
}

func (s *LinkService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// LinkDocuments — создать связь между двумя документами
func (s *LinkService) LinkDocuments(sourceIDStr, targetIDStr, sourceType, targetType, linkType string) (*models.DocumentLink, error) {
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

	if err := s.repo.Create(s.ctx, link); err != nil {
		return nil, fmt.Errorf("failed to create link: %w", err)
	}

	link.IDStr = link.ID.String()
	link.SourceIDStr = link.SourceID.String()
	link.TargetIDStr = link.TargetID.String()
	link.CreatedByStr = link.CreatedBy.String()

	return link, nil
}

// UnlinkDocument — удалить связь
func (s *LinkService) UnlinkDocument(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	// TODO: проверка прав доступа
	return s.repo.Delete(s.ctx, id)
}

// GetDocumentLinks — получить прямые связи документа
func (s *LinkService) GetDocumentLinks(docIDStr string) ([]models.DocumentLink, error) {
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	return s.repo.GetByDocumentID(s.ctx, docID)
}

// GraphNode — узел графа визуализации связей
type GraphNode struct {
	ID        string `json:"id"`
	Label     string `json:"label"` // Номер документа
	Type      string `json:"type"`  // входящий/исходящий
	Subject   string `json:"subject"`
	Date      string `json:"date"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
}

// GraphEdge — ребро графа визуализации связей
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"` // тип связи
}

// GraphData — данные графа (узлы и рёбра) для фронтенда
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GetDocumentFlow — получить данные графа для визуализации
func (s *LinkService) GetDocumentFlow(rootIDStr string) (*GraphData, error) {
	rootID, err := uuid.Parse(rootIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	links, err := s.repo.GetGraph(s.ctx, rootID)
	if err != nil {
		return nil, err
	}

	// Собираем уникальные ID документов для получения детальной информации
	docIDs := make(map[uuid.UUID]string) // ID -> тип документа

	// Если связей нет — возвращаем пустой граф
	if len(links) == 0 {
		return &GraphData{Nodes: []GraphNode{}, Edges: []GraphEdge{}}, nil
	}

	for _, l := range links {
		docIDs[l.SourceID] = l.SourceType
		docIDs[l.TargetID] = l.TargetType
	}

	// Получение деталей документов
	nodes := []GraphNode{}

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

		nodes = append(nodes, GraphNode{
			ID:        id.String(),
			Label:     label,
			Type:      docType,
			Subject:   subject,
			Date:      dateStr,
			Sender:    sender,
			Recipient: recipient,
		})
	}

	edges := []GraphEdge{}
	for _, l := range links {
		edges = append(edges, GraphEdge{
			ID:     l.IDStr,
			Source: l.SourceIDStr,
			Target: l.TargetIDStr,
			Label:  l.LinkType,
		})
	}

	return &GraphData{Nodes: nodes, Edges: edges}, nil
}
