package smb

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestConfigRejectsEscapingPaths(t *testing.T) {
	base := Config{Host: "nas", Share: "backups", User: "backup"}
	require.NoError(t, base.Validate())
	for _, directory := range []string{"../outside", "/root", "a/../../b", "a\\b", "a:b", "a//b"} {
		cfg := base
		cfg.Directory = directory
		require.Error(t, cfg.Validate())
	}
	base.Directory = "docflow/копии"
	require.NoError(t, base.Validate())
}
func TestSMBRoundtripIntegration(t *testing.T) {
	host := os.Getenv("DOCFLOW_INTEGRATION_SMB_HOST")
	if host == "" {
		t.Skip("set isolated SMB connection variables")
	}
	port, _ := strconv.Atoi(os.Getenv("DOCFLOW_INTEGRATION_SMB_PORT"))
	cfg := Config{Host: host, Port: port, Share: os.Getenv("DOCFLOW_INTEGRATION_SMB_SHARE"), User: os.Getenv("DOCFLOW_INTEGRATION_SMB_USER"), Directory: os.Getenv("DOCFLOW_INTEGRATION_SMB_DIRECTORY")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := Open(ctx, cfg, os.Getenv("DOCFLOW_INTEGRATION_SMB_PASSWORD"))
	require.NoError(t, err)
	defer client.Close()
	require.NoError(t, client.Check(ctx))
	other, err := Open(ctx, cfg, os.Getenv("DOCFLOW_INTEGRATION_SMB_PASSWORD"))
	require.NoError(t, err)
	defer other.Close()
	release, err := client.Lock(ctx)
	require.NoError(t, err)
	_, err = other.Lock(ctx)
	require.Error(t, err, "a live SMB handle must prevent another operation")
	release()
	// The old lock file remains, but its closed handle no longer owns it.
	releaseOther, err := other.Lock(ctx)
	require.NoError(t, err)
	releaseOther()
}
