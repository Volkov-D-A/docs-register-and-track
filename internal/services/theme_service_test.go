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

func TestThemeServiceRejectsUnsupportedTheme(t *testing.T) {
	service := &ThemeService{statePath: filepath.Join(t.TempDir(), "docflow", "theme.json")}

	if err := service.SetTheme("blue"); err == nil {
		t.Fatal("expected unsupported theme error")
	}
}
