package config

import (
	"fmt"
	"time"
)

const (
	defaultDBConnectTimeout   = 10 * time.Second
	defaultDBStatementTimeout = 30 * time.Second
	defaultDBLockTimeout      = 5 * time.Second
)

type BackupConfig struct {
	Directory string `json:"-"`
	KeyFile   string `json:"-"`
	MaxBytes  int64  `json:"-"`
}

type Config struct {
	Backup   BackupConfig       `json:"-"`
	Database DatabaseConfig     `json:"database"`
	S3       S3Config           `json:"s3"`
	Seq      SeqConfig          `json:"seq"`
	Outbox   OutboxWorkerConfig `json:"outbox,omitempty"`
	Server   ServerConfig       `json:"server,omitempty"`
}

type ServerConfig struct {
	ListenAddress   string `json:"listenAddress,omitempty"`
	SessionTTLHours int    `json:"sessionTtlHours,omitempty"`
}

type OutboxWorkerConfig struct {
	PollingIntervalSeconds   int `json:"pollingIntervalSeconds,omitempty"`
	BatchSize                int `json:"batchSize,omitempty"`
	StaleClaimTimeoutSeconds int `json:"staleClaimTimeoutSeconds,omitempty"`
	ConsumerTimeoutSeconds   int `json:"consumerTimeoutSeconds,omitempty"`
	ProcessedRetentionDays   int `json:"processedRetentionDays,omitempty"`
	CleanupIntervalMinutes   int `json:"cleanupIntervalMinutes,omitempty"`
}

type SeqConfig struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

type S3Config struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	UseSSL          bool   `json:"useSSL"`
	BucketName      string `json:"bucketName"`
}

// GetSecretAccessKey возвращает ключ доступа, полученный из server environment.
func (m S3Config) GetSecretAccessKey() string {
	return m.SecretAccessKey
}

// ConnectionString формирует строку подключения к базе данных.
func (d DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d statement_timeout=%d lock_timeout=%d",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
		int(defaultDBConnectTimeout/time.Second),
		defaultDBStatementTimeout.Milliseconds(),
		defaultDBLockTimeout.Milliseconds(),
	)
}
