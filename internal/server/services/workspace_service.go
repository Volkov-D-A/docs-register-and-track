package services

import (
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

func (s *WorkspaceService) GetOverview(assignmentMode, acknowledgmentMode string) (*dto.WorkspaceOverview, error) {
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
		acknowledgmentKinds, err := s.access.GetDocumentKindsWithAction("acknowledge")
		if err != nil {
			return nil, err
		}
		canExecute := user.IsDocumentParticipant || len(subjectIDs) > 1
		selectedAssignmentMode, assignmentModes, err := selectWorkspaceMode(assignmentMode, canExecute, len(assignmentKinds) > 0)
		if err != nil {
			return nil, err
		}
		selectedAcknowledgmentMode, acknowledgmentModes, err := selectWorkspaceMode(acknowledgmentMode, canExecute, len(acknowledgmentKinds) > 0)
		if err != nil {
			return nil, err
		}
		result := &dto.WorkspaceOverview{
			AssignmentModes: assignmentModes, AssignmentMode: selectedAssignmentMode,
			Assignments:         []dto.WorkspaceAssignment{},
			AcknowledgmentModes: acknowledgmentModes, AcknowledgmentMode: selectedAcknowledgmentMode,
			Acknowledgments: []dto.WorkspaceAcknowledgment{},
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
				result.AssignmentCounts.Overdue += counts.Overdue
				result.AssignmentCounts.DueSoon += counts.DueSoon
				result.AssignmentCounts.AwaitingAcceptance += counts.AwaitingAcceptance
				all = append(all, items...)
			}
			sort.Slice(all, func(i, j int) bool { return workspaceAssignmentLess(all[i], all[j]) })
			for _, item := range all[:min(len(all), 5)] {
				result.Assignments = append(result.Assignments, dto.WorkspaceAssignment{
					ID: item.ID.String(), DocumentID: item.DocumentID.String(), DocumentKind: item.DocumentKind,
					DocumentNumber: item.DocumentNumber, DocumentDate: item.DocumentDate, Content: item.Content,
					Deadline: item.Deadline, Status: item.Status,
				})
			}
		}
		if selectedAcknowledgmentMode != "" {
			queries, err := s.queries(selectedAcknowledgmentMode, acknowledgmentKinds, UUIDStrings(subjectIDs))
			if err != nil {
				return nil, err
			}
			all := []models.WorkspaceAcknowledgment{}
			for _, query := range queries {
				count, items, err := s.repo.AcknowledgmentSummary(query)
				if err != nil {
					return nil, err
				}
				result.AcknowledgmentCount += count
				all = append(all, items...)
			}
			sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
			for _, item := range all[:min(len(all), 5)] {
				result.Acknowledgments = append(result.Acknowledgments, dto.WorkspaceAcknowledgment{
					ID: item.ID.String(), DocumentID: item.DocumentID.String(), DocumentKind: item.DocumentKind,
					DocumentNumber: item.DocumentNumber, Content: item.Content, CreatedAt: item.CreatedAt,
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
	if a.Deadline == nil {
		return false
	}
	if b.Deadline == nil {
		return true
	}
	return a.Deadline.Before(*b.Deadline)
}

// ListAcknowledgments returns the same server-scoped work used by the overview.
func (s *WorkspaceService) ListAcknowledgments(mode string, page, pageSize int) (*dto.PagedResult[dto.WorkspaceAcknowledgment], error) {
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
	kinds, err := s.access.GetDocumentKindsWithAction("acknowledge")
	if err != nil {
		return nil, err
	}
	selected, _, err := selectWorkspaceMode(mode, user.IsDocumentParticipant || len(subjectIDs) > 1, len(kinds) > 0)
	if err != nil {
		return nil, err
	}
	if selected == "" {
		return nil, models.ErrForbidden
	}
	queries, err := s.queries(selected, kinds, UUIDStrings(subjectIDs))
	if err != nil {
		return nil, err
	}
	result, err := s.repo.ListAcknowledgments(queries, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]dto.WorkspaceAcknowledgment, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, dto.WorkspaceAcknowledgment{
			ID: item.ID.String(), DocumentID: item.DocumentID.String(), DocumentKind: item.DocumentKind,
			DocumentNumber: item.DocumentNumber, Content: item.Content, CreatedAt: item.CreatedAt,
		})
	}
	return &dto.PagedResult[dto.WorkspaceAcknowledgment]{Items: items, TotalCount: result.TotalCount, Page: result.Page, PageSize: result.PageSize}, nil
}
