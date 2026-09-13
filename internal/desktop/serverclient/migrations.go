package serverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type MigrationClient interface {
	Status(context.Context) (*dto.MigrationStatus, error)
	Apply(context.Context, string, string) (*dto.MigrationStatus, error)
	Rollback(context.Context, string, string, models.RollbackMigrationRequest) (*dto.MigrationStatus, error)
}

type Client struct {
	eventsContext   context.Context
	onEvent         func(LiveEvent)
	baseURL         string
	http            *http.Client
	tokenMu         sync.RWMutex
	token           string
	sessionRevision uint64
	sessionUserID   string
	sessionReason   string
	loginAttempt    uint64
	sessionContext  context.Context
	sessionCancel   context.CancelFunc
	onSessionEnded  func(SessionState)
}

type Options struct {
	AllowInsecureHTTP bool
}

func New(rawURL string) (*Client, error) {
	return NewWithOptions(rawURL, Options{})
}

func NewWithOptions(rawURL string, options Options) (*Client, error) {
	rawURL = strings.TrimRight(strings.TrimSpace(rawURL), "/")
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("server URL must be an absolute http or https URL")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) && !options.AllowInsecureHTTP {
		return nil, fmt.Errorf("server URL must use https unless it points to localhost")
	}
	httpClient, err := newHTTPClient()
	if err != nil {
		return nil, err
	}

	return &Client{baseURL: rawURL, http: httpClient}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) Status(ctx context.Context) (*dto.MigrationStatus, error) {
	return c.do(ctx, http.MethodGet, "/api/v1/admin/migrations", "", "", nil)
}

func (c *Client) Apply(ctx context.Context, login, password string) (*dto.MigrationStatus, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/admin/migrations/apply", login, password, nil)
}

func (c *Client) Rollback(ctx context.Context, login, password string, req models.RollbackMigrationRequest) (*dto.MigrationStatus, error) {
	body := struct {
		BackupCompleted      bool   `json:"backupCompleted"`
		BackupReference      string `json:"backupReference"`
		AcknowledgedDataLoss bool   `json:"acknowledgedDataLoss"`
		Confirmation         string `json:"confirmation"`
	}{req.BackupCompleted, req.BackupReference, req.AcknowledgedDataLoss, req.Confirmation}
	return c.do(ctx, http.MethodPost, "/api/v1/admin/migrations/rollback", login, password, body)
}

func (c *Client) do(ctx context.Context, method, path, login, password string, body any) (*dto.MigrationStatus, error) {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode server request: %w", err)
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return nil, fmt.Errorf("create server request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if login != "" || password != "" {
		req.SetBasicAuth(login, password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docflow-server is unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, decodeAuthError(resp)
	}
	var status dto.MigrationStatus
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode docflow-server response: %w", err)
	}
	return &status, nil
}
