package server

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/config"
	"github.com/Volkov-D-A/docs-register-and-track/internal/outbox"
)

func ValidateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is required")
	}

	db := cfg.Database
	if strings.TrimSpace(db.Host) == "" {
		return fmt.Errorf("database.host is required")
	}
	if db.Port < 1 || db.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535")
	}
	if strings.TrimSpace(db.User) == "" {
		return fmt.Errorf("database.user is required")
	}
	if strings.TrimSpace(db.Password) == "" {
		return fmt.Errorf("database.password is required")
	}
	if _, err := config.DecryptPassword(db.Password); err != nil {
		return fmt.Errorf("decrypt database.password: %w", err)
	}
	if strings.TrimSpace(db.DBName) == "" {
		return fmt.Errorf("database.dbname is required")
	}
	if strings.TrimSpace(db.SSLMode) == "" {
		return fmt.Errorf("database.sslmode is required")
	}

	minio := cfg.Minio
	if strings.TrimSpace(minio.Endpoint) == "" {
		return fmt.Errorf("minio.endpoint is required")
	}
	if strings.TrimSpace(minio.AccessKeyID) == "" {
		return fmt.Errorf("minio.accessKeyId is required")
	}
	if strings.TrimSpace(minio.SecretAccessKey) == "" {
		return fmt.Errorf("minio.secretAccessKey is required")
	}
	if _, err := config.DecryptPassword(minio.SecretAccessKey); err != nil {
		return fmt.Errorf("decrypt minio.secretAccessKey: %w", err)
	}
	if strings.TrimSpace(minio.BucketName) == "" {
		return fmt.Errorf("minio.bucketName is required")
	}

	if cfg.Seq.Enabled {
		parsed, err := url.ParseRequestURI(strings.TrimSpace(cfg.Seq.URL))
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("seq.url must be an absolute http or https URL when Seq is enabled")
		}
	}
	listenAddress := strings.TrimSpace(cfg.Server.ListenAddress)
	if listenAddress != "" {
		_, port, err := net.SplitHostPort(listenAddress)
		if err != nil || strings.TrimSpace(port) == "" {
			return fmt.Errorf("server.listenAddress must use host:port format")
		}
	}
	_, err := ResolveOutboxOptions(cfg.Outbox)
	return err
}

func ResolveOutboxOptions(cfg config.OutboxWorkerConfig) (outbox.Options, error) {
	if err := validateOptionalRange("outbox.pollingIntervalSeconds", cfg.PollingIntervalSeconds, 1, 60); err != nil {
		return outbox.Options{}, err
	}
	if err := validateOptionalRange("outbox.batchSize", cfg.BatchSize, 1, 1000); err != nil {
		return outbox.Options{}, err
	}
	if err := validateOptionalRange("outbox.staleClaimTimeoutSeconds", cfg.StaleClaimTimeoutSeconds, 60, 3600); err != nil {
		return outbox.Options{}, err
	}
	if err := validateOptionalRange("outbox.consumerTimeoutSeconds", cfg.ConsumerTimeoutSeconds, 1, 600); err != nil {
		return outbox.Options{}, err
	}
	if err := validateOptionalRange("outbox.processedRetentionDays", cfg.ProcessedRetentionDays, 1, 365); err != nil {
		return outbox.Options{}, err
	}
	if err := validateOptionalRange("outbox.cleanupIntervalMinutes", cfg.CleanupIntervalMinutes, 1, 1440); err != nil {
		return outbox.Options{}, err
	}

	options := outbox.Options{
		PollingInterval:    time.Duration(cfg.PollingIntervalSeconds) * time.Second,
		BatchSize:          cfg.BatchSize,
		StaleClaimTimeout:  time.Duration(cfg.StaleClaimTimeoutSeconds) * time.Second,
		ConsumerTimeout:    time.Duration(cfg.ConsumerTimeoutSeconds) * time.Second,
		ProcessedRetention: time.Duration(cfg.ProcessedRetentionDays) * 24 * time.Hour,
		CleanupInterval:    time.Duration(cfg.CleanupIntervalMinutes) * time.Minute,
	}.WithDefaults()
	if err := options.Validate(); err != nil {
		return outbox.Options{}, err
	}
	return options, nil
}

func validateOptionalRange(name string, value, minimum, maximum int) error {
	if value == 0 {
		return nil
	}
	if value < minimum || value > maximum {
		return fmt.Errorf("%s must be between %d and %d", name, minimum, maximum)
	}
	return nil
}
