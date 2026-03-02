package dto

import (
	"docflow/internal/models"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMapUser(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapUser(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.User{ID: id, Login: "test", FullName: "Test User", IsActive: true, Roles: []string{"admin"}}
		d := MapUser(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "test", d.Login)
		assert.Equal(t, "Test User", d.FullName)
		assert.True(t, d.IsActive)
	})
}

func TestMapAssignment(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAssignment(nil))
	})

	t.Run("success with co-executors", func(t *testing.T) {
		id := uuid.New()
		coExec := models.User{ID: uuid.New(), Login: "co", FullName: "CoExec"}
		m := &models.Assignment{
			ID:          id,
			DocumentID:  uuid.New(),
			ExecutorID:  uuid.New(),
			Content:     "Do it",
			Status:      "new",
			CoExecutors: []models.User{coExec},
		}
		d := MapAssignment(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Len(t, d.CoExecutors, 1)
		assert.Equal(t, "CoExec", d.CoExecutors[0].FullName)
	})
}

func TestMapAcknowledgment(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapAcknowledgment(nil))
	})

	t.Run("success with users", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()
		u := models.AcknowledgmentUser{ID: uuid.New(), UserID: uuid.New(), UserName: "Tester", CreatedAt: now}
		m := &models.Acknowledgment{ID: id, DocumentID: uuid.New(), CreatorID: uuid.New(), Users: []models.AcknowledgmentUser{u}}
		d := MapAcknowledgment(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Len(t, d.Users, 1)
		assert.Equal(t, "Tester", d.Users[0].UserName)
	})
}

func TestMapIncomingDocument(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapIncomingDocument(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.IncomingDocument{ID: id, IncomingNumber: "01-01/1", Subject: "Test"}
		d := MapIncomingDocument(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "01-01/1", d.IncomingNumber)
	})
}

func TestMapOutgoingDocument(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapOutgoingDocument(nil))
	})

	t.Run("success", func(t *testing.T) {
		id := uuid.New()
		m := &models.OutgoingDocument{ID: id, OutgoingNumber: "02-01/1", Subject: "Test"}
		d := MapOutgoingDocument(m)
		assert.Equal(t, id.String(), d.ID)
		assert.Equal(t, "02-01/1", d.OutgoingNumber)
	})
}

func TestMapSlices(t *testing.T) {
	t.Run("MapUsers nil", func(t *testing.T) {
		assert.Nil(t, MapUsers(nil))
	})

	t.Run("MapAssignments nil", func(t *testing.T) {
		assert.Nil(t, MapAssignments(nil))
	})

	t.Run("MapAssignments non-nil", func(t *testing.T) {
		items := []models.Assignment{{ID: uuid.New(), Status: "new"}, {ID: uuid.New(), Status: "done"}}
		result := MapAssignments(items)
		assert.Len(t, result, 2)
	})

	t.Run("MapAcknowledgments nil", func(t *testing.T) {
		assert.Nil(t, MapAcknowledgments(nil))
	})

	t.Run("MapIncomingDocuments nil", func(t *testing.T) {
		assert.Nil(t, MapIncomingDocuments(nil))
	})

	t.Run("MapOutgoingDocuments nil", func(t *testing.T) {
		assert.Nil(t, MapOutgoingDocuments(nil))
	})

	t.Run("MapAttachments nil", func(t *testing.T) {
		assert.Nil(t, MapAttachments(nil))
	})

	t.Run("MapDocumentLinks nil", func(t *testing.T) {
		assert.Nil(t, MapDocumentLinks(nil))
	})
}

func TestMapDashboardStats(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, MapDashboardStats(nil))
	})

	t.Run("success", func(t *testing.T) {
		m := &models.DashboardStats{Role: "admin", UserCount: 5, TotalDocuments: 100, DBSize: "10MB"}
		d := MapDashboardStats(m)
		assert.Equal(t, "admin", d.Role)
		assert.Equal(t, 5, d.UserCount)
		assert.Equal(t, 100, d.TotalDocuments)
	})
}
