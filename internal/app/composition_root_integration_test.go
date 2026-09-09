package app

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	desktopservices "github.com/Volkov-D-A/docs-register-and-track/internal/desktop/services"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/services"

	"github.com/stretchr/testify/require"
)

func TestCompositionRootStartupAndShutdownIntegration(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var closeLoggerCalls atomic.Int32

	appOptions, failure := newWailsOptionsWithDependencies(
		&config.Config{},
		WailsOptionsParams{
			ConfigPath: "integration-config.json",
			ReleaseNotesSource: []byte(`version: 1.0.0
releasedAt: 2026-08-06
changes:
  - title: Integration test
    description: Composition root lifecycle
`),
			CloseLogger: func() { closeLoggerCalls.Add(1) },
		},
		wailsOptionsDependencies{
			newThemeService: desktopservices.NewThemeService,
		},
	)
	require.Nil(t, failure)
	require.NotNil(t, appOptions)
	require.NotNil(t, appOptions.OnStartup)
	require.NotNil(t, appOptions.OnShutdown)

	boundTypes := make([]string, 0, len(appOptions.Bind))
	for _, binding := range appOptions.Bind {
		_, isServer := binding.(*services.ServerAttachmentService)
		require.False(t, isServer)
		boundTypes = append(boundTypes, fmt.Sprintf("%T", binding))
	}
	bindingOptions := NewBindingsWailsOptions()
	generatedBindingTypes := make([]string, 0, len(bindingOptions.Bind))
	boundConcreteTypes := make([]reflect.Type, 0, len(appOptions.Bind))
	for _, binding := range appOptions.Bind {
		boundConcreteTypes = append(boundConcreteTypes, reflect.TypeOf(binding))
	}
	generatedConcreteTypes := make([]reflect.Type, 0, len(bindingOptions.Bind))
	for _, binding := range bindingOptions.Bind {
		_, isServer := binding.(*services.ServerAttachmentService)
		require.False(t, isServer)
		generatedBindingTypes = append(generatedBindingTypes, fmt.Sprintf("%T", binding))
		generatedConcreteTypes = append(generatedConcreteTypes, reflect.TypeOf(binding))
	}
	require.ElementsMatch(t, boundConcreteTypes, generatedConcreteTypes)
	require.ElementsMatch(t, boundTypes, generatedBindingTypes)
	require.ElementsMatch(t, []string{
		"*services.AuthService",
		"*services.UserService",
		"*services.UserSubstitutionService",
		"*services.NomenclatureService",
		"*services.ReferenceService",
		"*services.DocumentAccessAdminService",
		"*services.DocumentKindService",
		"*services.DocumentQueryService",
		"*services.DocumentRegistrationService",
		"*services.AdministrativeOrderService",
		"*services.AssignmentService",
		"*services.DashboardService",
		"*services.StatisticsService",
		"*services.DepartmentService",
		"*services.SettingsService",
		"*services.AttachmentService",
		"*services.LinkService",
		"*services.AcknowledgmentService",
		"*services.SystemService",
		"*services.ReleaseNoteService",
		"*services.ThemeService",
		"*services.JournalService",
		"*services.AdminAuditLogService",
		"*services.UserEventService",
		"*services.OutboxAdminService",
	}, boundTypes)

	startupContext, cancelStartup := context.WithCancel(context.Background())
	defer cancelStartup()
	appOptions.OnStartup(startupContext)

	appOptions.OnShutdown(context.Background())
	require.Equal(t, int32(1), closeLoggerCalls.Load())
}

func TestDesktopCompositionNeedsOnlyServerConfiguration(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	appOptions, failure := newWailsOptionsWithDependencies(
		&config.Config{Server: config.ServerConfig{URL: "http://localhost:8080"}},
		WailsOptionsParams{
			ConfigPath: "integration-config.json",
			ReleaseNotesSource: []byte(`version: 1.0.0
releasedAt: 2026-08-06
changes:
  - title: Server-only desktop
    description: Desktop has no database or object-storage dependency
`),
		},
		wailsOptionsDependencies{
			newThemeService: desktopservices.NewThemeService,
		},
	)
	require.Nil(t, failure)
	require.NotNil(t, appOptions)
	appOptions.OnStartup(context.Background())
	appOptions.OnShutdown(context.Background())
}
