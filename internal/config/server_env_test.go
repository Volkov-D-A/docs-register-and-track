package config

import (
	"os"
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
	t.Setenv("S3_ENDPOINT", "https://seaweedfs:8333")
	t.Setenv("S3_ACCESS_KEY_ID", "docflow")
	t.Setenv("S3_SECRET_ACCESS_KEY", "secret")
	t.Setenv("S3_BUCKET", "docflow-attachments")
	t.Setenv("SEQ_URL", "http://seq:80")
	t.Setenv("SEQ_ENABLED", "true")
	t.Setenv("DOCFLOW_OUTBOX_BATCH_SIZE", "50")

	cfg, err := LoadServer()
	require.NoError(t, err)
	require.Equal(t, "postgres", cfg.Database.Host)
	require.Equal(t, 5432, cfg.Database.Port)
	require.Equal(t, "docflow", cfg.Database.User)
	require.Equal(t, "seaweedfs:8333", cfg.S3.Endpoint)
	require.True(t, cfg.S3.UseSSL)
	require.True(t, cfg.Seq.Enabled)
	require.Equal(t, 50, cfg.Outbox.BatchSize)
	require.Equal(t, 12, cfg.Server.SessionTTLHours)
}

func TestLoadServerRejectsInvalidEnvironmentValue(t *testing.T) {
	t.Setenv("POSTGRES_PORT", "not-a-port")

	_, err := LoadServer()
	require.ErrorContains(t, err, "POSTGRES_PORT must be an integer")
}

func TestS3EndpointSchemeOverridesSSLFlag(t *testing.T) {
	for _, tc := range []struct {
		endpoint, flag string
		secure         bool
	}{
		{"http://seaweedfs:8333", "true", false},
		{"https://storage.example", "false", true},
		{"storage.example:8333", "true", true},
	} {
		t.Run(tc.endpoint, func(t *testing.T) {
			t.Setenv("S3_ENDPOINT", tc.endpoint)
			t.Setenv("S3_USE_SSL", tc.flag)
			cfg, err := LoadServer()
			require.NoError(t, err)
			require.Equal(t, tc.secure, cfg.S3.UseSSL)
		})
	}
}

func TestS3RuntimeSecretFile(t *testing.T) {
	path := t.TempDir() + "/secret"
	require.NoError(t, os.WriteFile(path, []byte("literal $() ' secret\n"), 0600))
	t.Setenv("S3_SECRET_ACCESS_KEY_FILE", path)
	t.Setenv("S3_SECRET_ACCESS_KEY", "")
	cfg, err := LoadServer()
	require.NoError(t, err)
	require.Equal(t, "literal $() ' secret", cfg.S3.SecretAccessKey)
	t.Setenv("S3_SECRET_ACCESS_KEY", "conflict")
	_, err = LoadServer()
	require.ErrorContains(t, err, "set only one")
	t.Setenv("S3_SECRET_ACCESS_KEY", "")
	t.Setenv("S3_SECRET_ACCESS_KEY_FILE", path+"-absent")
	_, err = LoadServer()
	require.ErrorContains(t, err, "S3_SECRET_ACCESS_KEY_FILE")
}

func TestPostgresRuntimeSecretFile(t *testing.T) {
	path := t.TempDir() + "/password"
	require.NoError(t, os.WriteFile(path, []byte(" literal $() ' password \r\n"), 0600))
	t.Setenv("POSTGRES_PASSWORD_FILE", path)
	t.Setenv("POSTGRES_PASSWORD", "")
	cfg, err := LoadServer()
	require.NoError(t, err)
	require.Equal(t, " literal $() ' password ", cfg.Database.Password)
	t.Setenv("POSTGRES_PASSWORD", "conflict")
	_, err = LoadServer()
	require.ErrorContains(t, err, "set only one of POSTGRES_PASSWORD and POSTGRES_PASSWORD_FILE")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_PASSWORD_FILE", path+"-absent")
	_, err = LoadServer()
	require.ErrorContains(t, err, "POSTGRES_PASSWORD_FILE")
}
