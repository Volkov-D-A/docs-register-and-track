package services

import (
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestIsAssignmentAccessibleToExecutor(t *testing.T) {
	executorID := uuid.New()
	coExecutorID := uuid.New()
	assignment := &models.Assignment{
		ExecutorID:    executorID,
		CoExecutorIDs: []string{coExecutorID.String()},
	}

	assert.False(t, isAssignmentAccessibleToExecutor(executorID.String(), nil))
	assert.True(t, isAssignmentAccessibleToExecutor(executorID.String(), assignment))
	assert.True(t, isAssignmentAccessibleToExecutor(coExecutorID.String(), assignment))
	assert.False(t, isAssignmentAccessibleToExecutor(uuid.New().String(), assignment))
}
