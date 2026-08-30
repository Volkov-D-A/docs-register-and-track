package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func ProbeLive(ctx context.Context, listenAddress string) error {
	address := strings.TrimSpace(listenAddress)
	if address == "" {
		address = ":8080"
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("parse management API listen address: %w", err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	probeURL := "http://" + net.JoinHostPort(host, port) + "/health/live"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call management API liveness endpoint: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("management API liveness returned %s", resp.Status)
	}
	return nil
}
