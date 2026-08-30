package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadServerFromEnvironment(t *testing.T) {
	t.Setenv("POSTGRES_CONTAINER", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "docflow")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "docflow")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("MINIO_ENDPOINT", "https://minio:9000")
	t.Setenv("MINIO_ROOT_USER", "docflow")
	t.Setenv("MINIO_ROOT_PASSWORD", "secret")
	t.Setenv("MINIO_BUCKET", "docflow-attachments")
	t.Setenv("SEQ_URL", "http://seq:80")
	t.Setenv("SEQ_ENABLED", "true")
	t.Setenv("DOCFLOW_OUTBOX_BATCH_SIZE", "50")

	cfg, err := LoadServer()
	require.NoError(t, err)
	require.Equal(t, "postgres", cfg.Database.Host)
	require.Equal(t, 5432, cfg.Database.Port)
	require.Equal(t, "docflow", cfg.Database.User)
	require.Equal(t, "minio:9000", cfg.Minio.Endpoint)
	require.True(t, cfg.Minio.UseSSL)
	require.True(t, cfg.Seq.Enabled)
	require.Equal(t, 50, cfg.Outbox.BatchSize)
}

func TestLoadServerRejectsInvalidEnvironmentValue(t *testing.T) {
	t.Setenv("POSTGRES_PORT", "not-a-port")

	_, err := LoadServer()
	require.ErrorContains(t, err, "POSTGRES_PORT must be an integer")
}
