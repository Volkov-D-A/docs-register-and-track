package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemovedRecoveryCommandsFailBeforeConfigurationOrStorage(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "recovery-required")
	require.NoError(t, os.WriteFile(marker, []byte("preserve"), 0600))
	t.Setenv("DOCFLOW_BACKUP_DIRECTORY", dir)
	t.Setenv("POSTGRES_CONTAINER", "invalid-do-not-connect")
	t.Setenv("S3_ENDPOINT", "invalid-do-not-connect")
	for _, args := range [][]string{{"recovery"}, {"restore", "missing.tar.gz"}} {
		var stdout, stderr bytes.Buffer
		err := run(args, &stdout, &stderr)
		require.ErrorContains(t, err, "unknown command")
		require.Empty(t, stdout.String())
		raw, err := os.ReadFile(marker)
		require.NoError(t, err)
		require.Equal(t, "preserve", string(raw))
	}
}
