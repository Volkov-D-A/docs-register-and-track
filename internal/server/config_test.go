package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
)

func validConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host: "postgres", Port: 5432, User: "docflow", Password: "plain-test-password", DBName: "docflow", SSLMode: "require",
		},
		Minio: config.MinioConfig{
			Endpoint: "minio:9000", AccessKeyID: "docflow", SecretAccessKey: "plain-test-secret", BucketName: "docflow-attachments",
		},
		Seq: config.SeqConfig{Enabled: true, URL: "https://seq.example.test"},
	}
}

func TestValidateConfig(t *testing.T) {
	require.NoError(t, ValidateConfig(validConfig()))

	tests := []struct {
		name   string
		change func(*config.Config)
	}{
		{name: "missing database host", change: func(cfg *config.Config) { cfg.Database.Host = "" }},
		{name: "invalid database port", change: func(cfg *config.Config) { cfg.Database.Port = 70000 }},
		{name: "missing minio secret", change: func(cfg *config.Config) { cfg.Minio.SecretAccessKey = "" }},
		{name: "invalid seq URL", change: func(cfg *config.Config) { cfg.Seq.URL = "seq" }},
		{name: "invalid listen address", change: func(cfg *config.Config) { cfg.Server.ListenAddress = "8080" }},
		{name: "invalid polling interval", change: func(cfg *config.Config) { cfg.Outbox.PollingIntervalSeconds = 61 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.change(cfg)
			require.Error(t, ValidateConfig(cfg))
		})
	}
}

func TestResolveOutboxOptions(t *testing.T) {
	options, err := ResolveOutboxOptions(config.OutboxWorkerConfig{
		PollingIntervalSeconds:   2,
		BatchSize:                75,
		StaleClaimTimeoutSeconds: 120,
		ConsumerTimeoutSeconds:   45,
		ProcessedRetentionDays:   30,
		CleanupIntervalMinutes:   15,
	})
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, options.PollingInterval)
	require.Equal(t, 75, options.BatchSize)
	require.Equal(t, 2*time.Minute, options.StaleClaimTimeout)
	require.Equal(t, 45*time.Second, options.ConsumerTimeout)
	require.Equal(t, 30*24*time.Hour, options.ProcessedRetention)
	require.Equal(t, 15*time.Minute, options.CleanupInterval)
}
