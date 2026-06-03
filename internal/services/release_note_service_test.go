package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

func TestParseEmbeddedCurrentRelease(t *testing.T) {
	source := []byte(`version: 1.1.0
releasedAt: 2026-03-29
changes:
  - title: Новый блок
    description: Первая строка.
  - title: Исправление
    description: Короткое описание.
`)

	release, err := parseEmbeddedCurrentRelease(source)
	if err != nil {
		t.Fatalf("parseEmbeddedCurrentRelease() error = %v", err)
	}

	if release.Version != "1.1.0" {
		t.Fatalf("expected version 1.1.0, got %s", release.Version)
	}

	if len(release.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(release.Changes))
	}
}

func TestNewReleaseNoteServiceUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	service, err := NewReleaseNoteService([]byte(`version: 1.2.0
releasedAt: 2026-04-01
changes:
  - title: Новое
    description: Описание.
`))
	if err != nil {
		t.Fatalf("NewReleaseNoteService() error = %v", err)
	}

	if service.currentRelease.Version != "1.2.0" {
		t.Fatalf("expected version 1.2.0, got %s", service.currentRelease.Version)
	}

	expectedPath := filepath.Join(configDir, "docflow", "state.json")
	if service.statePath != expectedPath {
		t.Fatalf("expected state path %s, got %s", expectedPath, service.statePath)
	}
}

func TestParseEmbeddedCurrentReleaseErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "invalid yaml",
			source: `version:
  - broken
`,
		},
		{
			name: "missing version",
			source: `releasedAt: 2026-03-29
changes:
  - title: Новое
    description: Описание.
`,
		},
		{
			name: "missing changes",
			source: `version: 1.1.0
releasedAt: 2026-03-29
`,
		},
		{
			name: "invalid release date",
			source: `version: 1.1.0
releasedAt: 29.03.2026
changes:
  - title: Новое
    description: Описание.
`,
		},
		{
			name: "empty change title",
			source: `version: 1.1.0
releasedAt: 2026-03-29
changes:
  - title: " "
    description: Описание.
`,
		},
		{
			name: "empty change description",
			source: `version: 1.1.0
releasedAt: 2026-03-29
changes:
  - title: Новое
    description: " "
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseEmbeddedCurrentRelease([]byte(tt.source)); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}

func TestMarkCurrentViewedAndGetCurrent(t *testing.T) {
	service := &ReleaseNoteService{
		currentRelease: mustParseRelease(t, []byte(`version: 1.1.0
releasedAt: 2026-03-29
changes:
  - title: Новый блок
    description: Описание.
`)),
		statePath: filepath.Join(t.TempDir(), "docflow", "state.json"),
	}

	current, err := service.GetCurrent()
	if err != nil {
		t.Fatalf("GetCurrent() error = %v", err)
	}
	if current.IsViewed {
		t.Fatal("expected release to be unviewed before state save")
	}

	if err := service.MarkCurrentViewed(); err != nil {
		t.Fatalf("MarkCurrentViewed() error = %v", err)
	}

	current, err = service.GetCurrent()
	if err != nil {
		t.Fatalf("GetCurrent() after save error = %v", err)
	}
	if !current.IsViewed {
		t.Fatal("expected release to be viewed after state save")
	}

	if _, err := os.Stat(service.statePath); err != nil {
		t.Fatalf("expected state file to be created: %v", err)
	}
}

func mustParseRelease(t *testing.T, source []byte) models.ReleaseNote {
	t.Helper()

	release, err := parseEmbeddedCurrentRelease(source)
	if err != nil {
		t.Fatalf("parseEmbeddedCurrentRelease() error = %v", err)
	}

	return release
}
