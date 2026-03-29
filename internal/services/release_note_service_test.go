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
