package dto

import (
	"testing"
	"time"

	"docflow/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapUser(t *testing.T) {
	t.Run("nil user", func(t *testing.T) {
		assert.Nil(t, MapUser(nil))
	})

	t.Run("valid user without department", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()
		m := &models.User{
			ID:        id,
			Login:     "testuser",
			FullName:  "Test User",
			IsActive:  true,
			Roles:     []string{"admin"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		dto := MapUser(m)
		require.NotNil(t, dto)
		assert.Equal(t, id.String(), dto.ID)
		assert.Equal(t, "testuser", dto.Login)
		assert.Equal(t, "Test User", dto.FullName)
		assert.True(t, dto.IsActive)
		assert.Equal(t, []string{"admin"}, dto.Roles)
		assert.Equal(t, now, dto.CreatedAt)
		assert.Equal(t, now, dto.UpdatedAt)
		assert.Nil(t, dto.Department)
	})

	t.Run("valid user with department", func(t *testing.T) {
		userID := uuid.New()
		deptID := uuid.New()
		m := &models.User{
			ID:    userID,
			Login: "deptuser",
			Department: &models.Department{
				ID:   deptID,
				Name: "IT",
			},
		}

		dto := MapUser(m)
		require.NotNil(t, dto)
		assert.Equal(t, userID.String(), dto.ID)
		assert.Equal(t, "deptuser", dto.Login)
		require.NotNil(t, dto.Department)
		assert.Equal(t, deptID.String(), dto.Department.ID)
		assert.Equal(t, "IT", dto.Department.Name)
	})
}

func TestMapDepartment(t *testing.T) {
	t.Run("nil department", func(t *testing.T) {
		assert.Nil(t, MapDepartment(nil))
	})

	t.Run("valid department", func(t *testing.T) {
		id := uuid.New()
		m := &models.Department{
			ID:   id,
			Name: "HR",
		}

		dto := MapDepartment(m)
		require.NotNil(t, dto)
		assert.Equal(t, id.String(), dto.ID)
		assert.Equal(t, "HR", dto.Name)
	})
}

func TestMapAssignments(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		assert.Nil(t, MapAssignments(nil))
	})

	t.Run("empty slice", func(t *testing.T) {
		assert.Equal(t, []Assignment{}, MapAssignments([]models.Assignment{}))
	})

	t.Run("valid slice with items", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		modelsList := []models.Assignment{
			{ID: id1, Content: "Task 1", Status: "new"},
			{ID: id2, Content: "Task 2", Status: "in_progress"},
		}

		dtos := MapAssignments(modelsList)
		require.Len(t, dtos, 2)
		assert.Equal(t, id1.String(), dtos[0].ID)
		assert.Equal(t, "Task 1", dtos[0].Content)
		assert.Equal(t, "new", dtos[0].Status)
		assert.Equal(t, id2.String(), dtos[1].ID)
		assert.Equal(t, "Task 2", dtos[1].Content)
		assert.Equal(t, "in_progress", dtos[1].Status)
	})
}

func TestMapDashboardStats(t *testing.T) {
	t.Run("nil stats", func(t *testing.T) {
		assert.Nil(t, MapDashboardStats(nil))
	})

	t.Run("valid stats", func(t *testing.T) {
		m := &models.DashboardStats{
			Role:             "admin",
			UserCount:        10,
			TotalDocuments:   100,
			MyAssignmentsNew: 5,
			ExpiringAssignments: []models.Assignment{
				{Content: "Urgent Task"},
			},
		}

		dto := MapDashboardStats(m)
		require.NotNil(t, dto)
		assert.Equal(t, "admin", dto.Role)
		assert.Equal(t, 10, dto.UserCount)
		assert.Equal(t, 100, dto.TotalDocuments)
		assert.Equal(t, 5, dto.MyAssignmentsNew)
		require.Len(t, dto.ExpiringAssignments, 1)
		assert.Equal(t, "Urgent Task", dto.ExpiringAssignments[0].Content)
	})
}
