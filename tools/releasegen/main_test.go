package main

import "testing"

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
