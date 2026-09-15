package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevServerUpdatesOnlyAfterSuccessfulStableBuild(t *testing.T) {
	source, err := os.ReadFile("dev-server.sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"success", "failed-build", "changed-sources"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			write := func(name, content string) {
				t.Helper()
				path := filepath.Join(root, name)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0755); err != nil {
					t.Fatal(err)
				}
			}
			write("tools/dev-server.sh", string(source))
			write(".env", "DOCFLOW_SERVER_IMAGE_TAG=keep-existing-tag\n")
			write("build/bin/docflow-go", `#!/bin/sh
if test "$DOCFLOW_LOCAL_BUILD" != 1; then exit 9; fi
if test -f changed; then echo changed; else echo current; fi
`)
			write("bin/go", "#!/bin/sh\necho 1.0.7\n")
			write("bin/docker", `#!/bin/sh
printf '%s\n' "$*" >> calls
if test "$4" = build; then
 printf 'identity=%s version=%s local=%s\n' "$DOCFLOW_LOCAL_IDENTITY" "$DOCFLOW_LOCAL_VERSION" "$DOCFLOW_LOCAL_BUILD" >> calls
 case "$SCENARIO" in
  failed-build) exit 1;;
  changed-sources) touch changed;;
 esac
fi
`)
			cmd := exec.Command("bash", filepath.Join(root, "tools/dev-server.sh"))
			cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"), "SCENARIO="+scenario)
			output, err := cmd.CombinedOutput()
			if (err == nil) != (scenario == "success") {
				t.Fatalf("unexpected result: %v: %s", err, output)
			}
			calls, err := os.ReadFile(filepath.Join(root, "calls"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(calls), "identity=current version=1.0.7 local=1") {
				t.Fatalf("missing build identity: %s", calls)
			}
			composed := strings.Contains(string(calls), "compose -f docker-compose.yaml up -d --no-build --wait")
			if composed != (scenario == "success") {
				t.Fatalf("unexpected Compose update: %s", calls)
			}
			env, err := os.ReadFile(filepath.Join(root, ".env"))
			if err != nil {
				t.Fatal(err)
			}
			if string(env) != "DOCFLOW_SERVER_IMAGE_TAG=keep-existing-tag\n" {
				t.Fatal("modified .env")
			}
		})
	}
}
