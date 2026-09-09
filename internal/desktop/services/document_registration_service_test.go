package services

import (
	"context"
	"testing"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/observability"
	"github.com/Volkov-D-A/docs-register-and-track/internal/operations"
	"github.com/stretchr/testify/require"
)

type documentCommandClientStub struct {
	call func(context.Context, string, any) (any, error)
}

func (c documentCommandClientStub) RegisterDocument(ctx context.Context, kind string, req any) (any, error) {
	return c.call(ctx, kind, req)
}
func (c documentCommandClientStub) UpdateDocument(ctx context.Context, kind string, req any) (any, error) {
	return c.call(ctx, kind, req)
}
func (c documentCommandClientStub) CreateAdminDocumentDraft(ctx context.Context, kind string, req dto.AdminDraftCreateRequest) (any, error) {
	return c.call(ctx, kind, req)
}

func TestDocumentRegistrationServiceHTTPCommands(t *testing.T) {
	for _, operation := range []string{"register", "update", "draft"} {
		t.Run(operation, func(t *testing.T) {
			for _, serverErr := range []error{nil, models.ErrForbidden} {
				lifecycle := operations.NewLifecycle(time.Second)
				metrics := observability.NewRegistry(16)
				calls := 0
				client := documentCommandClientStub{call: func(ctx context.Context, kind string, req any) (any, error) {
					calls++
					require.NoError(t, ctx.Err())
					_, deadline := ctx.Deadline()
					require.True(t, deadline)
					require.Equal(t, "incoming_letter", kind)
					switch operation {
					case "register":
						require.Equal(t, dto.IncomingLetterRegisterRequest{Content: "test"}, req)
					case "update":
						require.Equal(t, dto.IncomingLetterUpdateRequest{ID: "doc-id"}, req)
					case "draft":
						require.Equal(t, dto.AdminDraftCreateRequest{}, req)
					}
					if serverErr != nil {
						return nil, serverErr
					}
					return "result", nil
				}}
				service := NewDocumentRegistrationService(client, lifecycle, metrics)
				invoke := func() (any, error) {
					switch operation {
					case "register":
						return service.Register("incoming_letter", map[string]any{"content": "test"})
					case "update":
						return service.Update("incoming_letter", map[string]any{"id": "doc-id"})
					default:
						return service.CreateAdminDraft("incoming_letter", dto.AdminDraftCreateRequest{})
					}
				}
				result, err := invoke()
				require.ErrorIs(t, err, serverErr)
				if serverErr == nil {
					require.Equal(t, "result", result)
				}
				require.Equal(t, 1, calls)
				require.Len(t, metrics.Snapshot(), 1)
				require.NoError(t, lifecycle.Shutdown(context.Background()))
				_, err = invoke()
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, 1, calls)
			}
		})
	}
}

func TestDocumentRegistrationServiceRejectsInvalidPayload(t *testing.T) {
	service := NewDocumentRegistrationService(documentCommandClientStub{call: func(context.Context, string, any) (any, error) {
		t.Fatal("invalid payload reached HTTP client")
		return nil, nil
	}}, nil, nil)
	_, err := service.Register("incoming_letter", map[string]any{"unknownField": true})
	require.Error(t, err)
	_, err = service.Update("incoming_letter", map[string]any{"unknownField": true})
	require.Error(t, err)
}

func TestDocumentRegistrationServiceRequiresClient(t *testing.T) {
	service := NewDocumentRegistrationService(nil, nil, nil)
	_, err := service.Register("incoming_letter", nil)
	require.Error(t, err)
	_, err = service.Update("incoming_letter", nil)
	require.Error(t, err)
	_, err = service.CreateAdminDraft("incoming_letter", dto.AdminDraftCreateRequest{})
	require.Error(t, err)
}

func TestDocumentRegistrationServiceShutdownCancelsInFlightCommand(t *testing.T) {
	lifecycle := operations.NewLifecycle(time.Second)
	started := make(chan struct{})
	finished := make(chan error, 1)
	client := documentCommandClientStub{call: func(ctx context.Context, _ string, _ any) (any, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	service := NewDocumentRegistrationService(client, lifecycle, nil)
	go func() {
		_, err := service.Register("incoming_letter", dto.IncomingLetterRegisterRequest{})
		finished <- err
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, lifecycle.Shutdown(ctx))
	require.ErrorIs(t, <-finished, context.Canceled)
}
