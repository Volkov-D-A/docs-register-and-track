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

// LinkDocuments creates a link between two documents
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

	// Basic validation: prevent self-linking
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

// UnlinkDocument removes a link
func (s *LinkService) UnlinkDocument(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}
	// Optional: check permissions
	return s.repo.Delete(s.ctx, id)
}

// GetDocumentLinks returns direct links for a document
func (s *LinkService) GetDocumentLinks(docIDStr string) ([]models.DocumentLink, error) {
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}
	return s.repo.GetByDocumentID(s.ctx, docID)
}

// GraphNode represents a node in the visualization graph
type GraphNode struct {
	ID        string `json:"id"`
	Label     string `json:"label"` // Document number
	Type      string `json:"type"`  // incoming/outgoing
	Subject   string `json:"subject"`
	Date      string `json:"date"`
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
}

// GraphEdge represents a link in the visualization graph
type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"` // link type
}

// GraphData holds nodes and edges for frontend
type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GetDocumentFlow returns the graph data for visualization
func (s *LinkService) GetDocumentFlow(rootIDStr string) (*GraphData, error) {
	rootID, err := uuid.Parse(rootIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid document ID: %w", err)
	}

	links, err := s.repo.GetGraph(s.ctx, rootID)
	if err != nil {
		return nil, err
	}

	// Collect unique document IDs to fetch details
	docIDs := make(map[uuid.UUID]string) // ID -> Type
	// Include root document in case it has no links but we want to show it?
	// Actually GetGraph relies on links. If no links, we might just return the root doc.

	// Check if links is empty, if so, just return root doc
	if len(links) == 0 {
		// Need to know type of root doc to fetch it.
		// We'll iterate links to find types.
		// If 0 links, we can try to find it in both tables or ask frontend to pass type.
		// For now, let's assume we return empty graph if no links, OR we can try to fetch root.
		// But we don't know the type.
		return &GraphData{Nodes: []GraphNode{}, Edges: []GraphEdge{}}, nil
	}

	for _, l := range links {
		docIDs[l.SourceID] = l.SourceType
		docIDs[l.TargetID] = l.TargetType
	}

	// Fetch document details
	nodes := []GraphNode{}

	// Helper to fetch doc info
	// This creates N queries, could be optimized with WHERE IN list if needed
	// But graph is usually small (< 20 nodes)
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
				recipient = doc.RecipientOrgName // Likely "Our Org"
			}
		} else if docType == "outgoing" {
			doc, err := s.outgoingDocRepo.GetByID(id)
			if err == nil && doc != nil {
				label = doc.OutgoingNumber
				subject = doc.Subject
				dateStr = doc.OutgoingDate.Format("02.01.2006")
				sender = doc.SenderOrgName // Likely "Our Org"
				recipient = doc.RecipientOrgName
				if recipient == "" {
					recipient = "Неизвестно"
				}
			}
		}

		if label == "" {
			label = "Unknown"
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
