package services

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

// isAssignmentAccessibleToExecutor оставлен как локальный helper для проверки участия
// пользователя в конкретном поручении без повторного обращения к access-layer.
func isAssignmentAccessibleToExecutor(currentUserID string, assignment *models.Assignment) bool {
	if assignment == nil {
		return false
	}
	if assignment.ExecutorID.String() == currentUserID {
		return true
	}
	for _, coExecutorID := range assignment.CoExecutorIDs {
		if coExecutorID == currentUserID {
			return true
		}
	}
	return false
}
