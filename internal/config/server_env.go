package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// LoadServer loads the standalone service configuration exclusively from
// deployment environment variables. Desktop JSON configuration is
// intentionally not used by the server process.
func LoadServer() (*Config, error) {
	cfg := &Config{Server: ServerConfig{ListenAddress: ":8080"}}
	if err := applyServerEnvironment(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyServerEnvironment(cfg *Config) error {
	stringValue("POSTGRES_CONTAINER", &cfg.Database.Host)
	stringValue("POSTGRES_USER", &cfg.Database.User)
	stringValue("POSTGRES_PASSWORD", &cfg.Database.Password)
	stringValue("POSTGRES_DB", &cfg.Database.DBName)
	stringValue("POSTGRES_SSLMODE", &cfg.Database.SSLMode)
	stringValue("MINIO_ROOT_USER", &cfg.Minio.AccessKeyID)
	stringValue("MINIO_ROOT_PASSWORD", &cfg.Minio.SecretAccessKey)
	stringValue("MINIO_BUCKET", &cfg.Minio.BucketName)
	stringValue("SEQ_URL", &cfg.Seq.URL)
	stringValue("DOCFLOW_SERVER_LISTEN_ADDRESS", &cfg.Server.ListenAddress)
	if err := minioEndpointValue("MINIO_ENDPOINT", &cfg.Minio); err != nil {
		return err
	}

	intValues := []struct {
		name   string
		target *int
	}{
		{"POSTGRES_PORT", &cfg.Database.Port},
		{"DOCFLOW_OUTBOX_POLLING_INTERVAL_SECONDS", &cfg.Outbox.PollingIntervalSeconds},
		{"DOCFLOW_OUTBOX_BATCH_SIZE", &cfg.Outbox.BatchSize},
		{"DOCFLOW_OUTBOX_STALE_CLAIM_TIMEOUT_SECONDS", &cfg.Outbox.StaleClaimTimeoutSeconds},
		{"DOCFLOW_OUTBOX_CONSUMER_TIMEOUT_SECONDS", &cfg.Outbox.ConsumerTimeoutSeconds},
		{"DOCFLOW_OUTBOX_PROCESSED_RETENTION_DAYS", &cfg.Outbox.ProcessedRetentionDays},
		{"DOCFLOW_OUTBOX_CLEANUP_INTERVAL_MINUTES", &cfg.Outbox.CleanupIntervalMinutes},
	}
	for _, value := range intValues {
		if err := intValue(value.name, value.target); err != nil {
			return err
		}
	}

	if err := boolValue("MINIO_USE_SSL", &cfg.Minio.UseSSL); err != nil {
		return err
	}
	if err := boolValue("SEQ_ENABLED", &cfg.Seq.Enabled); err != nil {
		return err
	}
	return nil
}

func minioEndpointValue(name string, target *MinioConfig) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	if !strings.Contains(value, "://") {
		target.Endpoint = value
		return nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be an HTTP(S) URL without a path", name)
	}
	switch parsed.Scheme {
	case "http":
		target.UseSSL = false
	case "https":
		target.UseSSL = true
	default:
		return fmt.Errorf("%s must use http or https", name)
	}
	target.Endpoint = parsed.Host
	return nil
}

func stringValue(name string, target *string) {
	if value, ok := os.LookupEnv(name); ok {
		*target = strings.TrimSpace(value)
	}
}

func intValue(name string, target *int) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("%s must be an integer: %w", name, err)
	}
	*target = parsed
	return nil
}

func boolValue(name string, target *bool) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("%s must be true or false: %w", name, err)
	}
	*target = parsed
	return nil
}
