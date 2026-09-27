package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/mocks"
)

type workspaceStoreStub struct {
	assignmentQueries []models.WorkspaceQuery
	ackQueries        []models.WorkspaceQuery
}

func (s *workspaceStoreStub) AssignmentSummary(query models.WorkspaceQuery) (models.WorkspaceAssignmentCounts, []models.WorkspaceAssignment, error) {
	s.assignmentQueries = append(s.assignmentQueries, query)
	return models.WorkspaceAssignmentCounts{New: 2, InProgress: 4, Overdue: 1}, []models.WorkspaceAssignment{}, nil
}

func (s *workspaceStoreStub) AcknowledgmentSummary(query models.WorkspaceQuery) (int, []models.WorkspaceAcknowledgment, error) {
	s.ackQueries = append(s.ackQueries, query)
	return 3, []models.WorkspaceAcknowledgment{}, nil
}

func (s *workspaceStoreStub) ListAcknowledgments([]models.WorkspaceQuery, int, int) (*models.PagedResult[models.WorkspaceAcknowledgment], error) {
	return &models.PagedResult[models.WorkspaceAcknowledgment]{Items: []models.WorkspaceAcknowledgment{}}, nil
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
	access := NewDocumentAccessService(principal, nil, nil, nil, permissions, nil,
		&userSubstitutionStoreStub{activePrincipals: substitutionIDs})
	store := &workspaceStoreStub{}
	return NewWorkspaceService(store, principal, access, nil), store, userID
}

func TestWorkspaceModesUsePermissionsAndSubstitution(t *testing.T) {
	kind := models.DocumentKindIncomingLetter
	subject := uuid.New()
	svc, store, userID := workspaceServiceFixture(t, true, map[models.DocumentKind]map[string]bool{
		kind: {"read": true, "assign": true, "acknowledge": true},
	}, subject)

	personal, err := svc.GetOverview("", "")
	require.NoError(t, err)
	require.Equal(t, []string{"execution", "control"}, personal.AssignmentModes)
	require.Equal(t, "execution", personal.AssignmentMode)
	require.Equal(t, "execution", personal.AcknowledgmentMode)
	require.Equal(t, 2, personal.AssignmentCounts.New)
	require.Equal(t, 4, personal.AssignmentCounts.InProgress)
	require.Equal(t, 3, personal.AcknowledgmentCount)
	require.Equal(t, []string{userID.String(), subject.String()}, store.assignmentQueries[0].SubjectIDs)
	require.Equal(t, []string{userID.String(), subject.String()}, store.ackQueries[0].SubjectIDs)

	controlled, err := svc.GetOverview("control", "control")
	require.NoError(t, err)
	require.Equal(t, "control", controlled.AssignmentMode)
	require.Equal(t, kind, store.assignmentQueries[1].Kind)
	require.False(t, store.assignmentQueries[1].ReadScope.Restricted)
	require.Equal(t, kind, store.ackQueries[1].Kind)
}

func TestWorkspaceRejectsUnavailableMode(t *testing.T) {
	svc, store, _ := workspaceServiceFixture(t, true, nil)
	_, err := svc.GetOverview("control", "")
	require.ErrorIs(t, err, models.ErrForbidden)
	require.Empty(t, store.assignmentQueries)

	_, err = svc.GetOverview("unexpected", "")
	require.Error(t, err)
}

func TestWorkspaceControlOnlyDoesNotRequestPersonalTasks(t *testing.T) {
	kind := models.DocumentKindOutgoingLetter
	svc, store, _ := workspaceServiceFixture(t, false, map[models.DocumentKind]map[string]bool{
		kind: {"read": true, "assign": true},
	})
	result, err := svc.GetOverview("", "")
	require.NoError(t, err)
	require.Equal(t, []string{"control"}, result.AssignmentModes)
	require.Equal(t, "control", result.AssignmentMode)
	require.Empty(t, result.AcknowledgmentModes)
	require.Len(t, store.assignmentQueries, 1)
	require.Equal(t, kind, store.assignmentQueries[0].Kind)
	require.Empty(t, store.ackQueries)
}

func TestWorkspaceControlKeepsParticipantDocumentScope(t *testing.T) {
	kind := models.DocumentKindIncomingLetter
	svc, store, userID := workspaceServiceFixture(t, true, map[models.DocumentKind]map[string]bool{
		kind: {"assign": true},
	})
	_, err := svc.GetOverview("control", "")
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
	_, err := svc.GetOverview("control", "")
	require.NoError(t, err)
	require.Len(t, store.assignmentQueries, 1)
	require.True(t, store.assignmentQueries[0].ReadScope.Restricted)
	require.Empty(t, store.assignmentQueries[0].ReadScope.AccessibleByUserIDs)
}
