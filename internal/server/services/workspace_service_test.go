package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
)

type workspaceStoreStub struct {
	assignmentQueries []models.WorkspaceQuery
	assignments       []models.WorkspaceAssignment
	documentScopes    map[models.DocumentKind]models.DocumentAccessScope
	documents         []models.WorkspaceDocument
}

func (s *workspaceStoreStub) AssignmentSummary(query models.WorkspaceQuery) (models.WorkspaceAssignmentCounts, []models.WorkspaceAssignment, error) {
	s.assignmentQueries = append(s.assignmentQueries, query)
	return models.WorkspaceAssignmentCounts{New: 2, InProgress: 4, Returned: 1, Overdue: 1}, s.assignments, nil
}

func workspaceServiceFixture(t *testing.T, participant bool, allowed map[models.DocumentKind]map[string]bool, substitutionIDs ...uuid.UUID) (*WorkspaceService, *workspaceStoreStub, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	user := &models.User{ID: userID, IsActive: true, IsDocumentParticipant: participant}
	users := mocks.NewUserStore(t)
	users.On("GetByID", userID).Return(user, nil).Maybe()
	principal := newAttachmentPrincipalStub(users)
	principal.currentUserID = userID
	permissions := &kindActionDocumentAccessStore{allowed: allowed}
	principal.SetAccessStore(permissions)
	access := NewDocumentAccessService(principal, nil, nil, permissions, nil,
		&userSubstitutionStoreStub{activePrincipals: substitutionIDs})
	store := &workspaceStoreStub{}
	return NewWorkspaceService(store, principal, access, nil), store, userID
}

func TestWorkspaceModesUsePermissionsAndSubstitution(t *testing.T) {
	kind := models.DocumentKindIncomingLetter
	subject := uuid.New()
	svc, store, userID := workspaceServiceFixture(t, true, map[models.DocumentKind]map[string]bool{
		kind: {"read": true, "assign": true},
	}, subject)

	personal, err := svc.GetOverview("")
	require.NoError(t, err)
	require.Equal(t, []string{"execution", "control"}, personal.AssignmentModes)
	require.Equal(t, "execution", personal.AssignmentMode)
	require.Equal(t, 2, personal.AssignmentCounts.New)
	require.Equal(t, 4, personal.AssignmentCounts.InProgress)
	require.Equal(t, 1, personal.AssignmentCounts.Returned)
	require.Equal(t, []string{userID.String(), subject.String()}, store.assignmentQueries[0].SubjectIDs)

	controlled, err := svc.GetOverview("control")
	require.NoError(t, err)
	require.Equal(t, "control", controlled.AssignmentMode)
	require.Equal(t, kind, store.assignmentQueries[1].Kind)
	require.False(t, store.assignmentQueries[1].ReadScope.Restricted)
}

func TestWorkspaceRejectsUnavailableMode(t *testing.T) {
	svc, store, _ := workspaceServiceFixture(t, true, nil)
	_, err := svc.GetOverview("control")
	require.ErrorIs(t, err, models.ErrForbidden)
	require.Empty(t, store.assignmentQueries)

	_, err = svc.GetOverview("unexpected")
	require.Error(t, err)
}

func TestWorkspaceControlOnlyDoesNotRequestPersonalTasks(t *testing.T) {
	kind := models.DocumentKindOutgoingLetter
	svc, store, _ := workspaceServiceFixture(t, false, map[models.DocumentKind]map[string]bool{
		kind: {"read": true, "assign": true},
	})
	result, err := svc.GetOverview("")
	require.NoError(t, err)
	require.Equal(t, []string{"control"}, result.AssignmentModes)
	require.Equal(t, "control", result.AssignmentMode)
	require.Len(t, store.assignmentQueries, 1)
	require.Equal(t, kind, store.assignmentQueries[0].Kind)
}

func TestWorkspaceControlKeepsParticipantDocumentScope(t *testing.T) {
	kind := models.DocumentKindIncomingLetter
	svc, store, userID := workspaceServiceFixture(t, true, map[models.DocumentKind]map[string]bool{
		kind: {"assign": true},
	})
	_, err := svc.GetOverview("control")
	require.NoError(t, err)
	require.Len(t, store.assignmentQueries, 1)
	require.True(t, store.assignmentQueries[0].ReadScope.Restricted)
	require.Equal(t, []string{userID.String()}, store.assignmentQueries[0].ReadScope.AccessibleByUserIDs)
}

func TestWorkspaceControlWithoutReadOrParticipationHasEmptyScope(t *testing.T) {
	kind := models.DocumentKindIncomingLetter
	svc, store, _ := workspaceServiceFixture(t, false, map[models.DocumentKind]map[string]bool{
		kind: {"assign": true},
	})
	_, err := svc.GetOverview("control")
	require.NoError(t, err)
	require.Len(t, store.assignmentQueries, 1)
	require.True(t, store.assignmentQueries[0].ReadScope.Restricted)
	require.Empty(t, store.assignmentQueries[0].ReadScope.AccessibleByUserIDs)
}

func TestWorkspaceAcknowledgmentsIncludeDocumentDetailsInOverview(t *testing.T) {
	svc, store, _ := workspaceServiceFixture(t, true, nil)
	item := models.WorkspaceAssignment{
		Type: models.AssignmentTypeAcknowledgment, Status: "new", ID: uuid.New(), DocumentID: uuid.New(), DocumentKind: "incoming_letter",
		DocumentNumber: "IT/43", DocumentDate: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
		DocumentContent: "Document content", Content: "Resolution", CreatedAt: time.Now(),
	}
	store.assignments = []models.WorkspaceAssignment{item}
	overview, err := svc.GetOverview("")
	require.NoError(t, err)
	require.Len(t, overview.Assignments, 1)
	acknowledgment := overview.Assignments[0]
	require.Equal(t, item.DocumentNumber, acknowledgment.DocumentNumber)
	require.Equal(t, item.DocumentDate, acknowledgment.DocumentDate)
	require.Equal(t, item.DocumentContent, acknowledgment.DocumentContent)
	require.Empty(t, acknowledgment.Content)
	require.Equal(t, models.AssignmentTypeAcknowledgment, acknowledgment.Type)
}

func (s *workspaceStoreStub) RecentDocuments(scopes map[models.DocumentKind]models.DocumentAccessScope) ([]models.WorkspaceDocument, error) {
	s.documentScopes = scopes
	return s.documents, nil
}

func TestWorkspaceRecentDocumentsUseReadScopesAndPreserveRegistrationTime(t *testing.T) {
	subject := uuid.New()
	kind := models.DocumentKindIncomingLetter
	svc, store, userID := workspaceServiceFixture(t, true, map[models.DocumentKind]map[string]bool{kind: {"read": true}}, subject)
	item := models.WorkspaceDocument{ID: uuid.New(), DocumentKind: string(kind), DocumentNumber: "IT/125",
		DocumentDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), RegisteredAt: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC),
		Description: "Alpha", Correspondents: []string{"Alpha", "Beta"}}
	store.documents = []models.WorkspaceDocument{item}
	result, err := svc.GetRecentDocuments()
	require.NoError(t, err)
	require.True(t, result.Available)
	require.Len(t, store.documentScopes, 4)
	require.False(t, store.documentScopes[kind].Restricted)
	restricted := store.documentScopes[models.DocumentKindOutgoingLetter]
	require.True(t, restricted.Restricted)
	require.Equal(t, []string{userID.String(), subject.String()}, restricted.AccessibleByUserIDs)
	require.Len(t, result.Items, 1)
	require.Equal(t, item.ID.String(), result.Items[0].ID)
	require.Equal(t, item.DocumentDate, result.Items[0].DocumentDate)
	require.Equal(t, item.RegisteredAt, result.Items[0].RegisteredAt)
	require.Equal(t, item.Correspondents, result.Items[0].Correspondents)
}

func TestWorkspaceRecentDocumentsHideWhenReadIsUnavailable(t *testing.T) {
	for _, permissions := range []map[models.DocumentKind]map[string]bool{nil, {models.DocumentKindIncomingLetter: {"create": true, "assign": true}}} {
		svc, store, _ := workspaceServiceFixture(t, false, permissions)
		result, err := svc.GetRecentDocuments()
		require.NoError(t, err)
		require.False(t, result.Available)
		require.Empty(t, result.Items)
		require.Nil(t, store.documentScopes)
	}
}

func TestWorkspaceUnifiedPreviewLimitsBothTypesTogether(t *testing.T) {
	svc, store, _ := workspaceServiceFixture(t, true, nil)
	now := time.Now()
	for i := 0; i < 7; i++ {
		taskType := models.AssignmentTypeExecution
		if i%2 == 0 {
			taskType = models.AssignmentTypeAcknowledgment
		}
		store.assignments = append(store.assignments, models.WorkspaceAssignment{
			ID: uuid.New(), Type: taskType, Status: "new", CreatedAt: now.Add(time.Duration(i) * time.Minute),
		})
	}
	overview, err := svc.GetOverview("")
	require.NoError(t, err)
	require.Len(t, overview.Assignments, 5)
	require.Equal(t, store.assignments[6].ID.String(), overview.Assignments[0].ID)
	require.Equal(t, models.AssignmentTypeAcknowledgment, overview.Assignments[0].Type)
	require.Equal(t, models.AssignmentTypeExecution, overview.Assignments[1].Type)
}
