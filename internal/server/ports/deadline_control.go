package ports

import (
	"context"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/google/uuid"
)

type DeadlineTransaction interface {
	GetControl(string) (*models.DocumentDeadlineControl, error)
	GetSeries(string) (*models.DocumentDeadlineSeries, error)
	SaveControl(*models.DocumentDeadlineControl, bool) error
	SaveSeries(*models.DocumentDeadlineSeries, bool) error
	EligibleExecutors([]string) (bool, error)
	Emit([]models.OutboxEvent) error
	RecipientUserIDs([]string) ([]uuid.UUID, error)
	ControllerUserIDs() ([]uuid.UUID, error)
}

type DeadlineControlStore interface {
	ListExecutors(context.Context, bool) ([]models.DeadlineExecutor, error)
	SaveExternalExecutor(context.Context, uuid.UUID, string, models.ExternalDeadlineExecutorRequest) (*models.DeadlineExecutor, error)
	GetControl(context.Context, string) (*models.DocumentDeadlineControl, error)
	GetSeries(context.Context, string) (*models.DocumentDeadlineSeries, error)
	CanRead(context.Context, uuid.UUID, string) (bool, error)
	CanManage(context.Context, uuid.UUID, string) (bool, error)
	ListControls(context.Context, uuid.UUID, models.DeadlineControlFilter) (*models.DeadlineControlPage, error)
	WithDocument(context.Context, uuid.UUID, string, func(DeadlineTransaction) error) error
}
