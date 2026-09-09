package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	require.NoError(t, run([]string{"version"}, &stdout, &stderr))
	require.NotEmpty(t, strings.TrimSpace(stdout.String()))
	require.Empty(t, stderr.String())
}

func TestCheckConfigCommand(t *testing.T) {
	t.Setenv("POSTGRES_CONTAINER", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "docflow")
	t.Setenv("POSTGRES_PASSWORD", "test")
	t.Setenv("POSTGRES_DB", "docflow")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("S3_ENDPOINT", "http://seaweedfs:8333")
	t.Setenv("S3_ACCESS_KEY_ID", "docflow")
	t.Setenv("S3_SECRET_ACCESS_KEY", "test")
	t.Setenv("S3_BUCKET", "docflow-attachments")
	t.Setenv("SEQ_ENABLED", "false")
	t.Setenv("DOCFLOW_OUTBOX_BATCH_SIZE", "75")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	require.NoError(t, run([]string{"check-config"}, &stdout, &stderr))
	require.Equal(t, "configuration is valid\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestServerCommandsRejectConfigFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"check-config", "-config", "server.json"}, &stdout, &stderr)
	require.ErrorContains(t, err, "does not accept arguments")
}

func TestUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"unknown"}, &stdout, &stderr)
	require.ErrorContains(t, err, "unknown command")
	require.Contains(t, stderr.String(), "Usage:")
}
