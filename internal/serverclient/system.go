package serverclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Volkov-D-A/docs-register-and-track/internal/dto"
)

const (
	SystemErrorTLS         = "tls_error"
	SystemErrorUnavailable = "server_unavailable"
	SystemErrorProtocol    = "protocol_error"
)

type SystemClient interface {
	SystemStatus(context.Context) (*dto.SystemStatus, error)
	Compatibility(context.Context, string) (*dto.CompatibilityResult, error)
}

type SystemRequestError struct {
	Kind string
	Err  error
}

func (e *SystemRequestError) Error() string { return e.Err.Error() }
func (e *SystemRequestError) Unwrap() error { return e.Err }

func SystemRequestErrorKind(err error) string {
	var requestErr *SystemRequestError
	if errors.As(err, &requestErr) {
		return requestErr.Kind
	}
	return SystemErrorProtocol
}

func (c *Client) SystemStatus(ctx context.Context) (*dto.SystemStatus, error) {
	var status dto.SystemStatus
	if err := c.doSystemGET(ctx, "/api/v1/system/status", &status); err != nil {
		return nil, err
	}
	if status.APIVersion != "v1" || status.ServerVersion == "" || (status.Status != "ready" && status.Status != "maintenance" && status.Status != "not_ready") {
		return nil, &SystemRequestError{Kind: SystemErrorProtocol, Err: fmt.Errorf("system status response violates API contract")}
	}
	return &status, nil
}

func (c *Client) Compatibility(ctx context.Context, clientVersion string) (*dto.CompatibilityResult, error) {
	query := url.Values{"clientVersion": []string{clientVersion}}
	var result dto.CompatibilityResult
	if err := c.doSystemGET(ctx, "/api/v1/system/compatibility?"+query.Encode(), &result); err != nil {
		return nil, err
	}
	validCode := result.Code == "compatible" || result.Code == "client_too_old" || result.Code == "client_too_new"
	if result.APIVersion != "v1" || result.ServerVersion == "" || !validCode || result.Compatible != (result.Code == "compatible") {
		return nil, &SystemRequestError{Kind: SystemErrorProtocol, Err: fmt.Errorf("compatibility response violates API contract")}
	}
	return &result, nil
}

func (c *Client) doSystemGET(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return &SystemRequestError{Kind: SystemErrorProtocol, Err: fmt.Errorf("create system request: %w", err)}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		kind := SystemErrorUnavailable
		var verificationErr *tls.CertificateVerificationError
		var unknownAuthority x509.UnknownAuthorityError
		var hostnameErr x509.HostnameError
		var invalidCertificate x509.CertificateInvalidError
		if errors.As(err, &verificationErr) || errors.As(err, &unknownAuthority) || errors.As(err, &hostnameErr) || errors.As(err, &invalidCertificate) {
			kind = SystemErrorTLS
		}
		return &SystemRequestError{Kind: kind, Err: fmt.Errorf("request system API: %w", err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &SystemRequestError{Kind: SystemErrorProtocol, Err: fmt.Errorf("system API returned %s", resp.Status)}
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(target); err != nil {
		return &SystemRequestError{Kind: SystemErrorProtocol, Err: fmt.Errorf("decode system API response: %w", err)}
	}
	return nil
}
