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
	documentScopes map[models.DocumentKind]models.DocumentAccessScope
	documents      []models.WorkspaceDocument
}

func workspaceServiceFixture(t *testing.T, participant bool) (*WorkspaceService, *workspaceStoreStub, uuid.UUID) {
	t.Helper()
	userID := uuid.New()
	user := &models.User{ID: userID, IsActive: true, IsDocumentParticipant: participant}
	users := mocks.NewUserStore(t)
	users.On("GetByID", userID).Return(user, nil).Maybe()
	principal := newDocumentPrincipalStub(users)
	principal.currentUserID = userID
	access := NewDocumentAccessService(principal, nil, nil)
	store := &workspaceStoreStub{}
	return NewWorkspaceService(store, principal, access, nil), store, userID
}

func (s *workspaceStoreStub) RecentDocuments(scopes map[models.DocumentKind]models.DocumentAccessScope) ([]models.WorkspaceDocument, error) {
	s.documentScopes = scopes
	return s.documents, nil
}

func TestWorkspaceRecentDocumentsUseReadScopesAndPreserveRegistrationTime(t *testing.T) {
	kind := models.DocumentKindIncomingLetter
	svc, store, userID := workspaceServiceFixture(t, true)
	item := models.WorkspaceDocument{ID: uuid.New(), DocumentKind: string(kind), DocumentNumber: "IT/125",
		DocumentDate: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC), RegisteredAt: time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC),
		Description: "Alpha", Correspondents: []string{"Alpha", "Beta"}}
	store.documents = []models.WorkspaceDocument{item}
	result, err := svc.GetRecentDocuments()
	require.NoError(t, err)
	require.True(t, result.Available)
	require.Len(t, store.documentScopes, 4)
	require.Equal(t, userID.String(), store.documentScopes[kind].UserID)
	restricted := store.documentScopes[models.DocumentKindOutgoingLetter]
	require.Equal(t, userID.String(), restricted.UserID)
	require.Len(t, result.Items, 1)
	require.Equal(t, item.ID.String(), result.Items[0].ID)
	require.Equal(t, item.DocumentDate, result.Items[0].DocumentDate)
	require.Equal(t, item.RegisteredAt, result.Items[0].RegisteredAt)
	require.Equal(t, item.Correspondents, result.Items[0].Correspondents)
}

func TestWorkspaceRecentDocumentsWithoutRootsStillQueryScopedStore(t *testing.T) {
	svc, store, userID := workspaceServiceFixture(t, false)
	result, err := svc.GetRecentDocuments()
	require.NoError(t, err)
	require.True(t, result.Available)
	require.Empty(t, result.Items)
	require.Len(t, store.documentScopes, 4)
	for _, scope := range store.documentScopes {
		require.Equal(t, userID.String(), scope.UserID)
	}
}
