package tools

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupCredentialsAreLiteralFiles(t *testing.T) {
	stage := t.TempDir()
	secret := `a space ' " $(touch /tmp/docflow-secret-injection) ; $HOME`
	script := `set -eu
docker() { echo test-image; }
source ../scripts/smb_backup_lib.sh
prepare_s3_client
`
	cmd := exec.Command("bash", "-c", script)
	cmd.Env = append(os.Environ(), "TMP_DIR="+stage, "DOCFLOW_SERVER_CONTAINER=test", "S3_NETWORK=test-network", "S3_ENDPOINT=storage:8333", "S3_USE_SSL=true", "S3_ACCESS_KEY_ID=literal-access", "S3_SECRET_ACCESS_KEY="+secret, "S3_BUCKET=docflow-test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prepare: %v: %s", err, out)
	}
	path := filepath.Join(stage, "s3-secrets", "secret")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, secret, string(raw))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("secret config mode: %v", info.Mode())
	}
}

func TestBackupRestartsWriterButFailedRestoreLeavesItStopped(t *testing.T) {
	for _, name := range []string{"../backup_smb_tar.sh", "../restore_smb_tar.sh"} {
		t.Run(name, func(t *testing.T) {
			// Extract and execute the actual cleanup function, injecting a prior failure.
			raw, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			source := string(raw)
			start := strings.Index(source, "cleanup() {")
			end := strings.Index(source[start:], "\n}\n") + start + 3
			script := `set -eu
PARTIAL_ARCHIVE_PATH=""; PARTIAL_MANIFEST_PATH=""; ARCHIVE_PUBLISHED=0; MANIFEST_PUBLISHED=0; MOUNTED=0; TMP_DIR=""; SERVER_WAS_RUNNING=1
resume_server() { echo resumed; }
` + source[start:end] + "\ntrap cleanup EXIT\nexit 7\n"
			cmd := exec.Command("bash", "-c", script)
			out, err := cmd.CombinedOutput()
			if err == nil || cmd.ProcessState.ExitCode() != 7 {
				t.Fatalf("cleanup lost failure exit: %v %s", err, out)
			}
			resumed := strings.Contains(string(out), "resumed")
			if resumed != (name == "../backup_smb_tar.sh") {
				t.Fatalf("writer state: %s", out)
			}
		})
	}
}

func TestDevResetPreservesUnrelatedVolumesAndBuildsCurrentServer(t *testing.T) {
	for _, execute := range []bool{false, true} {
		t.Run(fmt.Sprint(execute), func(t *testing.T) {
			stage := t.TempDir()
			fake := `#!/bin/bash
set -eu
printf '%s\n' "$*" >> "$CALL_LOG"
case "$*" in
 'compose config --format json') echo '{"name":"docflow"}' ;;
 'volume ls '*com.docker.compose.volume=pgdata) echo docflow_pgdata ;;
 'volume ls '*com.docker.compose.volume=seaweedfs_data) echo docflow_seaweedfs_data ;;
esac
`
			require.NoError(t, os.WriteFile(filepath.Join(stage, "docker"), []byte(fake), 0700))
			args := []string{"../scripts/reset-dev-storage.sh"}
			if execute {
				args = append(args, "--execute")
			}
			cmd := exec.Command("bash", args...)
			log := filepath.Join(stage, "calls")
			cmd.Env = append(os.Environ(), "PATH="+stage+":"+os.Getenv("PATH"), "CALL_LOG="+log)
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
			data, err := os.ReadFile(log)
			require.NoError(t, err)
			calls := string(data)
			if execute {
				require.Contains(t, calls, "volume rm docflow_pgdata docflow_seaweedfs_data\n")
				require.Contains(t, calls, "compose up -d --build")
			} else {
				require.NotContains(t, calls, "volume rm")
				require.NotContains(t, calls, "compose down")
			}
			require.NotContains(t, calls, "down -v")
			require.NotContains(t, calls, "seq_data")
			require.NotContains(t, calls, "caddy_data")
		})
	}
}
