package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestWailsAdapterNoopMethods(t *testing.T) {
	adapter := NewWailsAdapter()
	if adapter == nil {
		t.Fatal("expected adapter")
	}

	adapter.Print("print")
	adapter.Trace("trace")
	adapter.Debug("debug")
	adapter.Info("info")
	adapter.Warning("warning")
}

func TestWailsAdapterErrorLogsMessage(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, nil)))

	NewWailsAdapter().Error("wails error")

	logLine := out.String()
	if !strings.Contains(logLine, `"msg":"wails error"`) {
		t.Fatalf("log line does not contain wails error message: %s", logLine)
	}
}
