package releaseassets

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed current_release.yaml
var currentReleaseSource []byte

func CurrentVersion() (string, error) {
	var release struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(currentReleaseSource, &release); err != nil {
		return "", fmt.Errorf("parse embedded release version: %w", err)
	}
	release.Version = strings.TrimSpace(release.Version)
	if release.Version == "" {
		return "", fmt.Errorf("embedded release version is empty")
	}
	return release.Version, nil
}
