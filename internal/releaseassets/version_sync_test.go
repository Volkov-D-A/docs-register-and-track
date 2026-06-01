package releaseassets

import (
	"encoding/json"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWailsProductVersionMatchesCurrentRelease(t *testing.T) {
	currentReleaseSource, err := os.ReadFile("current_release.yaml")
	if err != nil {
		t.Fatalf("failed to read generated release asset: %v", err)
	}

	var currentRelease struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(currentReleaseSource, &currentRelease); err != nil {
		t.Fatalf("failed to parse generated release asset: %v", err)
	}

	wailsConfigSource, err := os.ReadFile("../../wails.json")
	if err != nil {
		t.Fatalf("failed to read Wails config: %v", err)
	}

	var wailsConfig struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsConfigSource, &wailsConfig); err != nil {
		t.Fatalf("failed to parse Wails config: %v", err)
	}

	if currentRelease.Version == "" {
		t.Fatal("generated release asset has empty version")
	}
	if wailsConfig.Info.ProductVersion != currentRelease.Version {
		t.Fatalf("Wails productVersion = %q, current release version = %q", wailsConfig.Info.ProductVersion, currentRelease.Version)
	}
}
