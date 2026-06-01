package main

import (
	"encoding/json"
	"testing"
)

func TestSelectCurrentReleaseChoosesLatestByDate(t *testing.T) {
	releases := []releaseEntry{
		{
			Version:    "1.0.1",
			ReleasedAt: "2026-03-29",
			Changes: []releaseItem{
				{Title: "First", Description: "Older release"},
			},
		},
		{
			Version:    "1.0.2",
			ReleasedAt: "2026-04-03",
			Changes: []releaseItem{
				{Title: "Second", Description: "Latest release"},
			},
		},
	}

	current, err := selectCurrentRelease(releases)
	if err != nil {
		t.Fatalf("selectCurrentRelease() error = %v", err)
	}

	if current.Version != "1.0.2" {
		t.Fatalf("expected latest version 1.0.2, got %s", current.Version)
	}
}

func TestSelectCurrentReleaseReturnsErrorForInvalidDate(t *testing.T) {
	releases := []releaseEntry{
		{
			Version:    "1.0.1",
			ReleasedAt: "invalid-date",
			Changes: []releaseItem{
				{Title: "Broken", Description: "Invalid date"},
			},
		},
	}

	if _, err := selectCurrentRelease(releases); err == nil {
		t.Fatal("expected error for invalid release date")
	}
}

func TestUpdateWailsProductVersion(t *testing.T) {
	source := []byte(`{
  "name": "docflow",
  "info": {
    "productName": "DocFlow",
    "productVersion": "1.0.1"
  }
}`)

	updated, err := updateWailsProductVersion(source, "1.0.4")
	if err != nil {
		t.Fatalf("updateWailsProductVersion() error = %v", err)
	}

	var config map[string]any
	if err := json.Unmarshal(updated, &config); err != nil {
		t.Fatalf("updated Wails config is not valid JSON: %v", err)
	}

	info, ok := config["info"].(map[string]any)
	if !ok {
		t.Fatal("updated Wails config does not contain info object")
	}
	if got := info["productVersion"]; got != "1.0.4" {
		t.Fatalf("productVersion = %v, want 1.0.4", got)
	}
}

func TestReadWailsProductVersion(t *testing.T) {
	source := []byte(`{"info":{"productVersion":"1.0.4"}}`)

	version, err := readWailsProductVersion(source)
	if err != nil {
		t.Fatalf("readWailsProductVersion() error = %v", err)
	}
	if version != "1.0.4" {
		t.Fatalf("version = %q, want 1.0.4", version)
	}
}

func TestReadWailsProductVersionRequiresVersion(t *testing.T) {
	source := []byte(`{"info":{}}`)

	if _, err := readWailsProductVersion(source); err == nil {
		t.Fatal("expected error for missing productVersion")
	}
}
