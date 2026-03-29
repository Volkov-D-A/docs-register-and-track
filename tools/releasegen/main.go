package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type releaseHistory struct {
	Releases []releaseEntry `yaml:"releases"`
}

type releaseEntry struct {
	Version    string        `yaml:"version"`
	ReleasedAt string        `yaml:"releasedAt"`
	Changes    []releaseItem `yaml:"changes"`
}

type releaseItem struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

func main() {
	sourcePath := flag.String("source", "", "Path to docs/releases.yaml")
	outputPath := flag.String("out", "", "Path to generated current_release.yaml")
	flag.Parse()

	if *sourcePath == "" || *outputPath == "" {
		exitWithError("source and out flags are required")
	}

	source, err := os.ReadFile(*sourcePath)
	if err != nil {
		exitWithError(fmt.Sprintf("failed to read source file: %v", err))
	}

	var history releaseHistory
	if err := yaml.Unmarshal(source, &history); err != nil {
		exitWithError(fmt.Sprintf("failed to parse source yaml: %v", err))
	}

	if len(history.Releases) == 0 {
		exitWithError("releases list must contain at least one item")
	}

	current := history.Releases[0]
	if err := validateRelease(current); err != nil {
		exitWithError(err.Error())
	}

	payload, err := yaml.Marshal(current)
	if err != nil {
		exitWithError(fmt.Sprintf("failed to encode current release: %v", err))
	}

	if err := os.WriteFile(*outputPath, payload, 0o644); err != nil {
		exitWithError(fmt.Sprintf("failed to write generated file: %v", err))
	}
}

func validateRelease(release releaseEntry) error {
	if strings.TrimSpace(release.Version) == "" {
		return fmt.Errorf("release version is required")
	}
	if strings.TrimSpace(release.ReleasedAt) == "" {
		return fmt.Errorf("release date is required")
	}
	if len(release.Changes) == 0 {
		return fmt.Errorf("release must contain at least one change")
	}

	for index, change := range release.Changes {
		if strings.TrimSpace(change.Title) == "" || strings.TrimSpace(change.Description) == "" {
			return fmt.Errorf("release change #%d must contain title and description", index+1)
		}
	}

	return nil
}

func exitWithError(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
