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
	cfg := &Config{Server: ServerConfig{ListenAddress: ":8080", SessionTTLHours: 12}}
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
	for _, secret := range []struct {
		name   string
		target *string
	}{
		{"S3_ACCESS_KEY_ID", &cfg.S3.AccessKeyID}, {"S3_SECRET_ACCESS_KEY", &cfg.S3.SecretAccessKey},
	} {
		if err := secretValue(secret.name, secret.target); err != nil {
			return err
		}
	}
	stringValue("S3_BUCKET", &cfg.S3.BucketName)
	stringValue("SEQ_URL", &cfg.Seq.URL)
	stringValue("DOCFLOW_SERVER_LISTEN_ADDRESS", &cfg.Server.ListenAddress)

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
		{"DOCFLOW_AUTH_SESSION_TTL_HOURS", &cfg.Server.SessionTTLHours},
	}
	for _, value := range intValues {
		if err := intValue(value.name, value.target); err != nil {
			return err
		}
	}

	if err := boolValue("S3_USE_SSL", &cfg.S3.UseSSL); err != nil {
		return err
	}
	if err := s3EndpointValue("S3_ENDPOINT", &cfg.S3); err != nil {
		return err
	}
	if err := boolValue("SEQ_ENABLED", &cfg.Seq.Enabled); err != nil {
		return err
	}
	return nil
}

func s3EndpointValue(name string, target *S3Config) error {
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
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
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

// secretValue supports runtime-mounted secrets, without legacy configuration aliases.
func secretValue(name string, target *string) error {
	path := os.Getenv(name + "_FILE")
	if path == "" {
		stringValue(name, target)
		return nil
	}
	if os.Getenv(name) != "" {
		return fmt.Errorf("set only one of %s and %s_FILE", name, name)
	}
	value, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s_FILE: %w", name, err)
	}
	*target = strings.TrimRight(string(value), "\r\n")
	return nil
}
