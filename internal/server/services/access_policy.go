package services

import "github.com/Volkov-D-A/docs-register-and-track/internal/models"

func isAssignmentAccessibleToAnyExecutor(userIDs []string, assignment *models.Assignment) bool {
	if assignment == nil {
		return false
	}
	for _, currentUserID := range userIDs {
		if assignment.ExecutorID.String() == currentUserID {
			return true
		}
		for _, coExecutorID := range assignment.CoExecutorIDs {
			if coExecutorID == currentUserID {
				return true
			}
		}
	}
	return false
}
