package services

import (
	"errors"
	"sort"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/ports"
)

// WorkspaceService resolves capabilities and work from the same server principal.
type WorkspaceService struct {
	repo    ports.WorkspaceStore
	auth    ports.DocumentAccessPrincipal
	access  *DocumentAccessService
	metrics *observability.Registry
}

func NewWorkspaceService(repo ports.WorkspaceStore, auth ports.DocumentAccessPrincipal, access *DocumentAccessService, metrics *observability.Registry) *WorkspaceService {
	return &WorkspaceService{repo: repo, auth: auth, access: access, metrics: metrics}
}

func selectWorkspaceMode(requested string, execution, control bool) (string, []string, error) {
	modes := []string{}
	if execution {
		modes = append(modes, models.WorkspaceModeExecution)
	}
	if control {
		modes = append(modes, models.WorkspaceModeControl)
	}
	if requested == "" && len(modes) > 0 {
		requested = modes[0]
	}
	if requested == "" {
		return "", modes, nil
	}
	if requested != models.WorkspaceModeExecution && requested != models.WorkspaceModeControl {
		return "", nil, models.NewBadRequest("неизвестный режим рабочего стола")
	}
	if (requested == models.WorkspaceModeExecution && !execution) || (requested == models.WorkspaceModeControl && !control) {
		return "", nil, models.ErrForbidden
	}
	return requested, modes, nil
}

func (s *WorkspaceService) GetOverview(assignmentMode string) (*dto.WorkspaceOverview, error) {
	return observability.Measure(s.metrics, "workspace.get_overview", func() (*dto.WorkspaceOverview, error) {
		if err := s.auth.RequireAuthenticated(); err != nil {
			return nil, err
		}
		if s.access == nil || s.repo == nil {
			return nil, models.ErrForbidden
		}
		user, err := s.auth.GetCurrentUser()
		if err != nil {
			return nil, err
		}
		subjectIDs, err := s.access.GetCurrentUserAndSubstitutionSubjectIDs()
		if err != nil {
			return nil, err
		}
		assignmentKinds, err := s.access.GetDocumentKindsWithAction("assign")
		if err != nil {
			return nil, err
		}
		canExecute := user.IsDocumentParticipant || len(subjectIDs) > 1
		selectedAssignmentMode, assignmentModes, err := selectWorkspaceMode(assignmentMode, canExecute, len(assignmentKinds) > 0)
		if err != nil {
			return nil, err
		}
		result := &dto.WorkspaceOverview{
			AssignmentModes: assignmentModes, AssignmentMode: selectedAssignmentMode,
			Assignments: []dto.WorkspaceAssignment{},
		}
		if selectedAssignmentMode != "" {
			queries, err := s.queries(selectedAssignmentMode, assignmentKinds, UUIDStrings(subjectIDs))
			if err != nil {
				return nil, err
			}
			all := []models.WorkspaceAssignment{}
			for _, query := range queries {
				counts, items, err := s.repo.AssignmentSummary(query)
				if err != nil {
					return nil, err
				}
				result.AssignmentCounts.New += counts.New
				result.AssignmentCounts.InProgress += counts.InProgress
				result.AssignmentCounts.Returned += counts.Returned
				result.AssignmentCounts.Overdue += counts.Overdue
				result.AssignmentCounts.DueSoon += counts.DueSoon
				result.AssignmentCounts.AwaitingAcceptance += counts.AwaitingAcceptance
				all = append(all, items...)
			}
			sort.SliceStable(all, func(i, j int) bool { return workspaceAssignmentLess(all[i], all[j]) })
			for _, item := range all[:min(len(all), 5)] {
				content := item.Content
				if item.Type == models.AssignmentTypeAcknowledgment {
					content = ""
				}
				result.Assignments = append(result.Assignments, dto.WorkspaceAssignment{
					ID: item.ID.String(), DocumentID: item.DocumentID.String(), DocumentKind: item.DocumentKind,
					DocumentNumber: item.DocumentNumber, DocumentDate: item.DocumentDate, Content: content,
					Deadline: item.Deadline, Status: item.Status, Type: item.Type, DocumentContent: item.DocumentContent,
				})
			}
		}
		return result, nil
	})
}

func (s *WorkspaceService) queries(mode string, kinds []models.DocumentKind, subjectIDs []string) ([]models.WorkspaceQuery, error) {
	if mode == models.WorkspaceModeExecution {
		return []models.WorkspaceQuery{{Mode: mode, SubjectIDs: subjectIDs, Limit: 5}}, nil
	}
	queries := make([]models.WorkspaceQuery, 0, len(kinds))
	for _, kind := range kinds {
		scope, err := s.access.ResolveReadScope(kind)
		if err != nil {
			return nil, err
		}
		queries = append(queries, models.WorkspaceQuery{Mode: mode, Kind: kind, ReadScope: *scope, Limit: 5})
	}
	return queries, nil
}

func workspaceAssignmentLess(a, b models.WorkspaceAssignment) bool {
	priority := func(item models.WorkspaceAssignment) int {
		if item.Status == "completed" {
			return 0
		}
		return 1
	}
	ap, bp := priority(a), priority(b)
	if ap != bp {
		return ap < bp
	}
	if a.Deadline == nil && b.Deadline == nil {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID.String() < b.ID.String()
	}
	if a.Deadline == nil {
		return false
	}
	if b.Deadline == nil {
		return true
	}
	if !a.Deadline.Equal(*b.Deadline) {
		return a.Deadline.Before(*b.Deadline)
	}
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return a.ID.String() < b.ID.String()
}

// GetRecentDocuments is independent of the assignment mode.
func (s *WorkspaceService) GetRecentDocuments() (*dto.WorkspaceDocuments, error) {
	if err := s.auth.RequireAuthenticated(); err != nil {
		return nil, err
	}
	if s.access == nil || s.repo == nil {
		return nil, models.ErrForbidden
	}
	result := &dto.WorkspaceDocuments{Items: []dto.WorkspaceDocument{}}
	if err := s.access.RequireDomainRead(); err != nil {
		if errors.Is(err, models.ErrForbidden) {
			return result, nil
		}
		return nil, err
	}
	scopes := make(map[models.DocumentKind]models.DocumentAccessScope)
	for _, spec := range models.AllDocumentKindSpecs() {
		scope, err := s.access.ResolveReadScope(spec.Code)
		if err != nil {
			return nil, err
		}
		scopes[spec.Code] = *scope
		if !scope.Restricted || len(scope.AllowedNomenclatureIDs) > 0 || scope.AccessibleByUserID != "" || len(scope.AccessibleByUserIDs) > 0 {
			result.Available = true
		}
	}
	if !result.Available {
		return result, nil
	}
	items, err := s.repo.RecentDocuments(scopes)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		correspondents := item.Correspondents
		if correspondents == nil {
			correspondents = []string{}
		}
		result.Items = append(result.Items, dto.WorkspaceDocument{
			ID: item.ID.String(), DocumentKind: item.DocumentKind,
			DocumentNumber: item.DocumentNumber, DocumentDate: item.DocumentDate,
			RegisteredAt: item.RegisteredAt, Description: item.Description, Correspondents: correspondents,
		})
	}
	return result, nil
}
