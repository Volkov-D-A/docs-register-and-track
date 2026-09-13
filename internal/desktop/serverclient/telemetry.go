package serverclient

import (
	"context"
	"net/http"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type TelemetryClient interface {
	SendTechnicalLogs(context.Context, []models.TechnicalLogEvent) error
}

func (c *Client) SendTechnicalLogs(ctx context.Context, events []models.TechnicalLogEvent) error {
	if len(events) == 0 {
		return nil
	}
	return c.doUserRequest(ctx, http.MethodPost, "/api/v1/telemetry/logs", models.TechnicalLogBatch{Events: events}, http.StatusNoContent, nil)
}
