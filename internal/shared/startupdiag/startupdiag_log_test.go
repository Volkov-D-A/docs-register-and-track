package startupdiag

import (
	"bytes"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{}))

	Log(logger, Failure{
		Component:  " database ",
		ConfigPath: " /tmp/config.yaml ",
		Summary:    " cannot connect ",
		NextStep:   " check db ",
		Err:        errors.New("connection refused"),
	})

	got := buf.String()
	require.NotEmpty(t, got)
	assert.Contains(t, got, "Startup diagnostics")
	assert.Contains(t, got, "component=database")
	assert.Contains(t, got, "summary=\"cannot connect\"")
	assert.Contains(t, got, "next_step=\"check db\"")
	assert.Contains(t, got, "config_path=/tmp/config.yaml")
	assert.Contains(t, got, "error=\"connection refused\"")
}

func TestLogUsesDefaults(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	Log(logger, Failure{})

	got := buf.String()
	require.NotEmpty(t, got)
	assert.Contains(t, got, "component=startup")
	assert.Contains(t, got, "summary=\"Приложение не смогло завершить запуск.\"")
}
