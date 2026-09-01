package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/serverclient"
)

type fakeSystemClient struct {
	compatibility *dto.CompatibilityResult
	status        *dto.SystemStatus
	err           error
}

func (c fakeSystemClient) Compatibility(context.Context, string) (*dto.CompatibilityResult, error) {
	return c.compatibility, c.err
}
func (c fakeSystemClient) SystemStatus(context.Context) (*dto.SystemStatus, error) {
	return c.status, c.err
}

func TestSystemServiceAllowsReadyAndMaintenanceServers(t *testing.T) {
	for _, state := range []string{"ready", "maintenance"} {
		service := NewSystemServiceWithClient(fakeSystemClient{
			compatibility: &dto.CompatibilityResult{Compatible: true, Code: "compatible"},
			status:        &dto.SystemStatus{Status: state, Code: state},
		}, "1.0.6")

		result := service.GetBootstrapStatus()
		assert.Equal(t, state, result.State)
		assert.NotNil(t, result.System)
	}
}

func TestSystemServiceBlocksIncompatibleClient(t *testing.T) {
	service := NewSystemServiceWithClient(fakeSystemClient{
		compatibility: &dto.CompatibilityResult{Compatible: false, Code: "client_too_old"},
	}, "1.0.5")

	result := service.GetBootstrapStatus()
	assert.Equal(t, "client_too_old", result.State)
	assert.Nil(t, result.System)
}

func TestSystemServiceReturnsActionableConnectionFailure(t *testing.T) {
	service := NewSystemServiceWithClient(fakeSystemClient{err: &serverclient.SystemRequestError{
		Kind: serverclient.SystemErrorUnavailable,
		Err:  errors.New("connection refused"),
	}}, "1.0.6")

	result := service.GetBootstrapStatus()
	assert.Equal(t, "server_unavailable", result.State)
	assert.Contains(t, result.Message, "Сервер недоступен")
	assert.NotContains(t, result.Message, "connection refused")
}
