package tools

import (
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise cleanup of the actual administrative API smoke script, preserving
// failures and unrelated files. The retired CIFS scripts no longer exist.
func TestBackupSmokeCleanupPreservesFailureAndOnlyRemovesTestState(t *testing.T) {
	raw, err := os.ReadFile("../testing/scripts/integration-smoke.sh")
	require.NoError(t, err)
	source := string(raw)
	start := strings.Index(source, "cleanup() {")
	require.NotEqual(t, -1, start)
	end := strings.Index(source[start:], "\n}\n")
	require.NotEqual(t, -1, end)
	dir := t.TempDir()
	stage := filepath.Join(dir, "stage")
	evidence := filepath.Join(dir, "evidence")
	require.NoError(t, os.Mkdir(stage, 0700))
	require.NoError(t, os.Mkdir(evidence, 0700))
	script := `set -eu
docker() { printf '%s\n' "$*" >> "$CALL_LOG"; }
fake_compose() { printf '%s\n' "$*" >> "$CALL_LOG"; return 1; }
compose=(fake_compose)
compose_env=/dev/null
` + source[start:start+end+3] + "\ntrap cleanup EXIT\nexit 7\n"
	cmd := exec.Command("bash", "-c", script)
	log := filepath.Join(dir, "calls")
	cmd.Env = append(os.Environ(), "stage="+stage, "evidence="+evidence, "CALL_LOG="+log)
	out, err := cmd.CombinedOutput()
	require.Error(t, err)
	require.Equal(t, 7, cmd.ProcessState.ExitCode(), string(out))
	require.NoDirExists(t, stage)
	require.DirExists(t, evidence)
	calls, err := os.ReadFile(log)
	require.NoError(t, err)
	require.Contains(t, string(calls), "-p docflow-backup-test -f testing/compose/backup-test.yaml down -v --remove-orphans")
	require.NotContains(t, string(calls), "docker-compose.yaml")
}
