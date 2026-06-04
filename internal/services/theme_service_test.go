package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestThemeServiceGetAndSetTheme(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "docflow", "theme.json")
	service := &ThemeService{statePath: statePath}

	theme, err := service.GetTheme()
	if err != nil {
		t.Fatalf("GetTheme() error = %v", err)
	}
	if theme != appThemeLight {
		t.Fatalf("expected default theme %s, got %s", appThemeLight, theme)
	}

	if err := service.SetTheme(appThemeDark); err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}

	theme, err = service.GetTheme()
	if err != nil {
		t.Fatalf("GetTheme() after save error = %v", err)
	}
	if theme != appThemeDark {
		t.Fatalf("expected theme %s, got %s", appThemeDark, theme)
	}

	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("expected theme state file to be created: %v", err)
	}
}

func TestNewThemeServiceUsesUserConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	service, err := NewThemeService()
	if err != nil {
		t.Fatalf("NewThemeService() error = %v", err)
	}

	expectedPath := filepath.Join(configDir, "docflow", "theme.json")
	if service.statePath != expectedPath {
		t.Fatalf("expected state path %s, got %s", expectedPath, service.statePath)
	}
}

func TestThemeServiceRejectsUnsupportedTheme(t *testing.T) {
	service := &ThemeService{statePath: filepath.Join(t.TempDir(), "docflow", "theme.json")}

	if err := service.SetTheme("blue"); err == nil {
		t.Fatal("expected unsupported theme error")
	}
}

func TestThemeServiceIgnoresInvalidState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "docflow", "theme.json")
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(statePath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := &ThemeService{statePath: statePath}

	theme, err := service.GetTheme()
	if err != nil {
		t.Fatalf("GetTheme() error = %v", err)
	}
	if theme != appThemeLight {
		t.Fatalf("expected fallback theme %s, got %s", appThemeLight, theme)
	}
}

func TestThemeServiceReadStateErrors(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "docflow")
	if err := os.MkdirAll(statePath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	service := &ThemeService{statePath: statePath}

	if _, err := service.GetTheme(); err == nil {
		t.Fatal("expected read error for directory state path")
	}
}

func TestThemeServiceWriteStateErrors(t *testing.T) {
	tempDir := t.TempDir()
	blockingFile := filepath.Join(tempDir, "docflow")
	if err := os.WriteFile(blockingFile, []byte("not a directory"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	service := &ThemeService{statePath: filepath.Join(blockingFile, "theme.json")}

	if err := service.SetTheme(appThemeDark); err == nil {
		t.Fatal("expected write error when parent path is a file")
	}
}

func TestNormalizeAppTheme(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{name: "trim and lowercase light", input: " LIGHT ", expected: appThemeLight, ok: true},
		{name: "trim and lowercase dark", input: " Dark ", expected: appThemeDark, ok: true},
		{name: "unsupported", input: "system", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, ok := normalizeAppTheme(tt.input)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v, got %v", tt.ok, ok)
			}
			if actual != tt.expected {
				t.Fatalf("expected theme %q, got %q", tt.expected, actual)
			}
		})
	}
}
