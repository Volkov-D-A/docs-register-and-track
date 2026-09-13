package services

import (
	"context"
	"errors"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/desktop/serverclient"
)

// DashboardService exposes server-owned operations through HTTP.
type DashboardService struct{ server serverclient.DashboardClient }

func NewDashboardService(client serverclient.DashboardClient) *DashboardService {
	return &DashboardService{server: client}
}

var errDashboardServiceClientNotConfigured = errors.New("docflow-server dashboard client is not configured")

func (s *DashboardService) GetActivity() (*dto.DashboardActivity, error) {
	if s.server == nil {
		return nil, errDashboardServiceClientNotConfigured
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.server.GetDashboardActivity(ctx)
}
