// Package smb connects directly to an SMB share; no kernel mount is involved.
package smb

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	smb2 "github.com/cloudsoda/go-smb2"
	"github.com/google/uuid"
)

type Config models.SMBDestination

func (c Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid SMB port")
	}
	if strings.TrimSpace(c.Host) == "" || strings.ContainsAny(c.Host, "/\\@\x00\r\n") {
		return fmt.Errorf("invalid SMB host")
	}
	if c.Share == "" || strings.ContainsAny(c.Share, "/\\:\x00\r\n") || c.Share == "." || c.Share == ".." {
		return fmt.Errorf("invalid SMB share")
	}
	if c.User == "" {
		return fmt.Errorf("SMB user is required")
	}
	if c.Directory != "" && !validPath(c.Directory) {
		return fmt.Errorf("invalid SMB subdirectory")
	}
	return nil
}
func validPath(name string) bool {
	if name == "" || name == "." || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

type Client struct {
	session   *smb2.Session
	share     *smb2.Share
	directory string
}

func Open(ctx context.Context, cfg Config, password string) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	dialer := smb2.Dialer{Negotiator: smb2.Negotiator{RequireMessageSigning: true}, Initiator: &smb2.NTLMInitiator{User: cfg.User, Password: password, Domain: cfg.Domain}}
	port := cfg.Port
	if port == 0 {
		port = 445
	}
	address := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	connection, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect SMB: %w", err)
	}
	session, err := dialer.DialConn(ctx, connection, address)
	if err != nil {
		connection.Close()
		return nil, fmt.Errorf("connect SMB: %w", err)
	}
	share, err := session.WithContext(ctx).Mount(cfg.Share)
	if err != nil {
		session.Logoff()
		return nil, fmt.Errorf("open SMB share: %w", err)
	}
	return &Client{session: session, share: share, directory: cfg.Directory}, nil
}
func (c *Client) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return errors.Join(c.share.WithContext(ctx).Umount(), c.session.WithContext(ctx).Logoff())
}
func (c *Client) name(name string) (string, error) {
	if !validPath(name) {
		return "", fmt.Errorf("invalid backup filename")
	}
	return path.Join(c.directory, name), nil
}

// Check exercises the same operations required for archive publication.
func (c *Client) Check(ctx context.Context) error {
	name := ".docflow-check-" + uuid.NewString()
	payload := []byte("docflow SMB verification")
	if err := c.Write(ctx, name, bytes.NewReader(payload)); err != nil {
		return err
	}
	defer c.Remove(context.WithoutCancel(ctx), name)
	renamed := name + "-renamed"
	if err := c.Rename(ctx, name, renamed); err != nil {
		return err
	}
	defer c.Remove(context.WithoutCancel(ctx), renamed)
	if err := c.Verify(ctx, renamed, int64(len(payload)), fmt.Sprintf("%x", sha256.Sum256(payload))); err != nil {
		return err
	}
	return c.Remove(ctx, renamed)
}
func (c *Client) Write(ctx context.Context, name string, source io.Reader) error {
	full, err := c.name(name)
	if err != nil {
		return err
	}
	share := c.share.WithContext(ctx)
	// The administrator creates/selects the backup directory; do not create arbitrary paths.
	file, err := share.OpenFile(full, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, source)
	syncErr := file.Sync()
	closeErr := file.Close()
	err = errors.Join(copyErr, syncErr, closeErr)
	if err != nil {
		_ = c.Remove(context.WithoutCancel(ctx), name)
	}
	return err
}
func (c *Client) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	full, err := c.name(name)
	if err != nil {
		return nil, err
	}
	return c.share.WithContext(ctx).Open(full)
}
func (c *Client) Verify(ctx context.Context, name string, size int64, digest string) error {
	reader, err := c.Open(ctx, name)
	if err != nil {
		return err
	}
	defer reader.Close()
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(reader, size+1))
	if err != nil {
		return err
	}
	if n != size || fmt.Sprintf("%x", hash.Sum(nil)) != digest {
		return fmt.Errorf("remote backup checksum mismatch")
	}
	return nil
}
func (c *Client) Rename(ctx context.Context, old, new string) error {
	a, err := c.name(old)
	if err != nil {
		return err
	}
	b, err := c.name(new)
	if err != nil {
		return err
	}
	return c.share.WithContext(ctx).Rename(a, b)
}
func (c *Client) Remove(ctx context.Context, name string) error {
	full, err := c.name(name)
	if err != nil {
		return err
	}
	limited, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.share.WithContext(limited).Remove(full)
}

func (c *Client) List(ctx context.Context) ([]os.FileInfo, error) {
	directory := c.directory
	if directory == "" {
		directory = "."
	}
	return c.share.WithContext(ctx).ReadDir(directory)
}
