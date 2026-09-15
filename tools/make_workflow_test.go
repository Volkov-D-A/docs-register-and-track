package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseGateMakeTargetsRemainAvailable(t *testing.T) {
	script, err := os.ReadFile("release-gate.sh")
	if err != nil {
		t.Fatal(err)
	}
	steps := regexp.MustCompile(`make --no-print-directory ([a-z-]+)`).FindAllStringSubmatch(string(script), -1)
	if len(steps) != 11 {
		t.Fatalf("review gate coverage: %d steps", len(steps))
	}
	for _, step := range steps {
		cmd := exec.Command("make", "--no-print-directory", "-n", step[1])
		cmd.Dir = ".."
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", step[1], err, output)
		}
	}
}

func TestDockerPublicationRequiresSuccessfulStableCleanBuild(t *testing.T) {
	for _, scenario := range []string{"success", "failed-build", "changed-sources", "failed-push"} {
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
			for _, name := range []string{"Makefile", "build/make/checks.mk"} {
				content, err := os.ReadFile(filepath.Join("..", name))
				if err != nil {
					t.Fatal(err)
				}
				write(name, string(content))
			}
			write(".env", "DOCFLOW_SERVER_VERSION=1.0.7\n")
			write("build/bin/docflow-go", `#!/bin/sh
if test "$DOCFLOW_LOCAL_BUILD" != 0; then exit 9; fi
if test -f changed; then echo '364:other:'; else echo '363:revision:'; fi
`)
			write("bin/docker", `#!/bin/sh
printf '%s\n' "$*" >> calls
if test "$SCENARIO" = failed-push && test "$1" = push; then exit 1; fi
if test "$1" = build; then
 case "$SCENARIO" in
  failed-build) exit 1;;
  changed-sources) touch changed;;
 esac
fi
`)
			cmd := exec.Command("make", "--no-print-directory", "-o", "_build-compiler", "-o", "_check-docker", "-o", "_check-docflow-server-version", "docker-server-push", "DOCFLOW_LOCAL_BUILD=1")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"), "SCENARIO="+scenario)
			output, err := cmd.CombinedOutput()
			if (err == nil) != (scenario == "success") {
				t.Fatalf("%v: %s", err, output)
			}
			calls, err := os.ReadFile(filepath.Join(root, "calls"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(calls), "--build-arg LOCAL_BUILD=0") {
				t.Fatalf("not a clean build: %s", calls)
			}
			pushed := strings.Contains(string(calls), "push hehelf/docflow-service:1.0.7.363-revision")
			if pushed != (scenario == "success" || scenario == "failed-push") {
				t.Fatalf("unexpected publication: %s", calls)
			}
			latest := strings.Contains(string(calls), "push hehelf/docflow-service:latest")
			if latest != (scenario == "success") {
				t.Fatalf("unexpected latest publication: %s", calls)
			}
			if latest && !strings.Contains(string(calls), "push hehelf/docflow-service:1.0.7.363-revision\ntag hehelf/docflow-service:1.0.7.363-revision hehelf/docflow-service:latest\npush hehelf/docflow-service:latest") {
				t.Fatalf("latest must follow the exact image publication: %s", calls)
			}

		})
	}
}
