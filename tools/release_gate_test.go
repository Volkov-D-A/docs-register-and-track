package tools

import (
	"os"
	"strings"
	"testing"
)

func TestReleaseGateRequiresPostgreSQLIntegration(t *testing.T) {
	script, err := os.ReadFile("release-gate.sh")
	if err != nil {
		t.Fatalf("read release gate: %v", err)
	}
	content := string(script)
	goTest := strings.Index(content, `run_step "go-test"`)
	integration := strings.Index(content, `run_step "integration-test" "PostgreSQL integration" make --no-print-directory integration-test`)
	frontend := strings.Index(content, `run_step "frontend-ci"`)
	if goTest < 0 || integration < 0 || frontend < 0 {
		t.Fatalf("release gate stages are incomplete: go=%d integration=%d frontend=%d", goTest, integration, frontend)
	}
	if !(goTest < integration && integration < frontend) {
		t.Fatalf("PostgreSQL integration must run after Go tests and before frontend checks")
	}
}
