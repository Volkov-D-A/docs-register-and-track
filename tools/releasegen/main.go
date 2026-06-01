package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

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
	wailsConfigPath := flag.String("wails-config", "", "Optional path to wails.json to synchronize info.productVersion")
	checkOnly := flag.Bool("check", false, "Verify generated files are fresh without writing them")
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

	current, err := selectCurrentRelease(history.Releases)
	if err != nil {
		exitWithError(err.Error())
	}

	payload, err := yaml.Marshal(current)
	if err != nil {
		exitWithError(fmt.Sprintf("failed to encode current release: %v", err))
	}

	if *checkOnly {
		if err := checkGeneratedReleaseAsset(*outputPath, payload); err != nil {
			exitWithError(err.Error())
		}
		if *wailsConfigPath != "" {
			if err := checkWailsProductVersion(*wailsConfigPath, current.Version); err != nil {
				exitWithError(err.Error())
			}
		}
		return
	}

	if err := os.WriteFile(*outputPath, payload, 0o644); err != nil {
		exitWithError(fmt.Sprintf("failed to write generated file: %v", err))
	}

	if *wailsConfigPath != "" {
		if err := syncWailsProductVersion(*wailsConfigPath, current.Version); err != nil {
			exitWithError(err.Error())
		}
	}
}

func selectCurrentRelease(releases []releaseEntry) (releaseEntry, error) {
	current := releases[0]
	if err := validateRelease(current); err != nil {
		return releaseEntry{}, err
	}

	currentReleasedAt, err := time.Parse("2006-01-02", current.ReleasedAt)
	if err != nil {
		return releaseEntry{}, fmt.Errorf("invalid release date %q: %w", current.ReleasedAt, err)
	}

	for _, release := range releases[1:] {
		if err := validateRelease(release); err != nil {
			return releaseEntry{}, err
		}

		releasedAt, err := time.Parse("2006-01-02", release.ReleasedAt)
		if err != nil {
			return releaseEntry{}, fmt.Errorf("invalid release date %q: %w", release.ReleasedAt, err)
		}

		if releasedAt.After(currentReleasedAt) {
			current = release
			currentReleasedAt = releasedAt
		}
	}

	return current, nil
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

func syncWailsProductVersion(path, version string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read Wails config: %w", err)
	}

	updated, err := updateWailsProductVersion(source, version)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("failed to write Wails config: %w", err)
	}
	return nil
}

func checkGeneratedReleaseAsset(path string, expected []byte) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read generated release asset: %w", err)
	}
	if string(source) != string(expected) {
		return fmt.Errorf("generated release asset is stale; run make release-assets")
	}
	return nil
}

func checkWailsProductVersion(path, version string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read Wails config: %w", err)
	}

	productVersion, err := readWailsProductVersion(source)
	if err != nil {
		return err
	}
	if productVersion != version {
		return fmt.Errorf("Wails productVersion %q does not match current release version %q; run make release-assets", productVersion, version)
	}
	return nil
}

func updateWailsProductVersion(source []byte, version string) ([]byte, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return nil, fmt.Errorf("Wails product version is required")
	}

	var config map[string]any
	if err := json.Unmarshal(source, &config); err != nil {
		return nil, fmt.Errorf("failed to parse Wails config: %w", err)
	}

	info, ok := config["info"].(map[string]any)
	if !ok {
		info = map[string]any{}
		config["info"] = info
	}
	info["productVersion"] = version

	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode Wails config: %w", err)
	}
	payload = append(payload, '\n')
	return payload, nil
}

func readWailsProductVersion(source []byte) (string, error) {
	var config struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(source, &config); err != nil {
		return "", fmt.Errorf("failed to parse Wails config: %w", err)
	}
	if strings.TrimSpace(config.Info.ProductVersion) == "" {
		return "", fmt.Errorf("Wails productVersion is required")
	}
	return strings.TrimSpace(config.Info.ProductVersion), nil
}

func exitWithError(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
