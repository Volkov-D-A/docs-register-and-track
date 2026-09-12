package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIsAssignmentAccessibleToAnyExecutor(t *testing.T) {
	executorID := uuid.New()
	coExecutorID := uuid.New()
	assignment := &models.Assignment{
		ExecutorID:    executorID,
		CoExecutorIDs: []string{coExecutorID.String()},
	}

	assert.False(t, isAssignmentAccessibleToAnyExecutor([]string{executorID.String()}, nil))
	assert.True(t, isAssignmentAccessibleToAnyExecutor([]string{executorID.String()}, assignment))
	assert.True(t, isAssignmentAccessibleToAnyExecutor([]string{coExecutorID.String()}, assignment))
	assert.False(t, isAssignmentAccessibleToAnyExecutor(nil, assignment))
	assert.True(t, isAssignmentAccessibleToAnyExecutor([]string{uuid.New().String(), coExecutorID.String()}, assignment))
	assert.False(t, isAssignmentAccessibleToAnyExecutor([]string{uuid.New().String()}, assignment))
}
