package ports

import (
	"time"

	"github.com/google/uuid"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type AssignmentOutboxStore interface {
	CreateWithOutbox(id, documentID, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error)
	UpdateDetailsWithOutbox(id, executorID uuid.UUID, content string, deadline *time.Time, coExecutorIDs []string, expectedUpdatedAt time.Time, effects []models.OutboxEvent) (*models.Assignment, error)
	UpdateWithOutbox(id, executorID uuid.UUID, content string, deadline *time.Time, status, report string, completedAt *time.Time, coExecutorIDs []string, effects []models.OutboxEvent) (*models.Assignment, error)
	DeleteWithOutbox(id uuid.UUID, effects []models.OutboxEvent) error
}

type AssignmentSeriesStore interface {
	CreateSeriesWithFirstAssignment(seriesID, assignmentID, documentID, executorID, createdBy uuid.UUID, content string, firstDeadline time.Time, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error)
	GetAssignmentSeries(id uuid.UUID) (*models.AssignmentSeries, error)
	GetAssignmentSeriesByAssignment(id uuid.UUID) (*models.AssignmentSeries, error)
	UpdateAssignmentSeries(id, executorID uuid.UUID, content, intervalUnit string, intervalValue int, dayRule string, dayOfMonth int, coExecutorIDs []string, effects []models.OutboxEvent) (*models.AssignmentSeries, error)
	CancelAssignmentSeries(id, actorID uuid.UUID, effects []models.OutboxEvent) error
	FinishSeriesIterationWithNext(currentID, seriesID, nextID uuid.UUID, expectedSeriesUpdatedAt time.Time, report string, completedAt *time.Time, nextDeadline time.Time, nextIteration int, executorID uuid.UUID, content string, coExecutorIDs []string, currentEffects, nextEffects []models.OutboxEvent) (*models.Assignment, error)
	GetAssignmentSeriesHistory(seriesID uuid.UUID) ([]models.Assignment, error)
}
