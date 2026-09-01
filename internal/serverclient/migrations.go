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
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/database"
	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

type MigrationClient interface {
	Status(context.Context) (*database.MigrationStatus, error)
	Apply(context.Context, string, string) (*database.MigrationStatus, error)
	Rollback(context.Context, string, string, models.RollbackMigrationRequest) (*database.MigrationStatus, error)
}

type Client struct {
	baseURL string
	http    *http.Client
	tokenMu sync.RWMutex
	token   string
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
	return &Client{baseURL: rawURL, http: &http.Client{Timeout: 3 * time.Minute}}, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) Status(ctx context.Context) (*database.MigrationStatus, error) {
	return c.do(ctx, http.MethodGet, "/api/v1/admin/migrations", "", "", nil)
}

func (c *Client) Apply(ctx context.Context, login, password string) (*database.MigrationStatus, error) {
	return c.do(ctx, http.MethodPost, "/api/v1/admin/migrations/apply", login, password, nil)
}

func (c *Client) Rollback(ctx context.Context, login, password string, req models.RollbackMigrationRequest) (*database.MigrationStatus, error) {
	body := struct {
		BackupCompleted      bool   `json:"backupCompleted"`
		BackupReference      string `json:"backupReference"`
		AcknowledgedDataLoss bool   `json:"acknowledgedDataLoss"`
		Confirmation         string `json:"confirmation"`
	}{req.BackupCompleted, req.BackupReference, req.AcknowledgedDataLoss, req.Confirmation}
	return c.do(ctx, http.MethodPost, "/api/v1/admin/migrations/rollback", login, password, body)
}

func (c *Client) do(ctx context.Context, method, path, login, password string, body any) (*database.MigrationStatus, error) {
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
		var apiErr struct {
			Code  string `json:"code"`
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&apiErr)
		if apiErr.Error == "" {
			apiErr.Error = resp.Status
		}
		return nil, fmt.Errorf("docflow-server %s: %s", apiErr.Code, apiErr.Error)
	}
	var status database.MigrationStatus
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode docflow-server response: %w", err)
	}
	return &status, nil
}
