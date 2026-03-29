package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
)

// ReleaseNoteService предоставляет доступ к текущему релизу, вшитому в приложение.
type ReleaseNoteService struct {
	currentRelease models.ReleaseNote
	statePath      string
	stateMu        sync.Mutex
}

type embeddedRelease struct {
	Version    string                          `yaml:"version"`
	ReleasedAt string                          `yaml:"releasedAt"`
	Changes    []models.ReleaseNoteChangeInput `yaml:"changes"`
}

type releaseViewState struct {
	ViewedVersion string    `json:"viewedVersion"`
	ViewedAt      time.Time `json:"viewedAt"`
}

// NewReleaseNoteService создает сервис релизной информации из встроенного файла.
func NewReleaseNoteService(source []byte) (*ReleaseNoteService, error) {
	currentRelease, err := parseEmbeddedCurrentRelease(source)
	if err != nil {
		return nil, err
	}

	statePath, err := getReleaseStatePath()
	if err != nil {
		return nil, err
	}

	return &ReleaseNoteService{
		currentRelease: currentRelease,
		statePath:      statePath,
	}, nil
}

// GetCurrent возвращает текущий релиз и локальный статус его просмотра.
func (s *ReleaseNoteService) GetCurrent() (*models.ReleaseNote, error) {
	state, err := s.readState()
	if err != nil {
		return nil, err
	}

	current := s.currentRelease
	current.IsViewed = state.ViewedVersion == current.Version
	return &current, nil
}

// MarkCurrentViewed сохраняет факт ознакомления с текущей версией локально на устройстве.
func (s *ReleaseNoteService) MarkCurrentViewed() error {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.statePath), 0o755); err != nil {
		return fmt.Errorf("failed to create app state directory: %w", err)
	}

	state := releaseViewState{
		ViewedVersion: s.currentRelease.Version,
		ViewedAt:      time.Now().UTC(),
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode release state: %w", err)
	}

	if err := os.WriteFile(s.statePath, payload, 0o600); err != nil {
		return fmt.Errorf("failed to save release state: %w", err)
	}

	return nil
}

func parseEmbeddedCurrentRelease(source []byte) (models.ReleaseNote, error) {
	var release embeddedRelease
	if err := yaml.Unmarshal(source, &release); err != nil {
		return models.ReleaseNote{}, fmt.Errorf("failed to parse embedded current release: %w", err)
	}

	release.Version = strings.TrimSpace(release.Version)
	if release.Version == "" {
		return models.ReleaseNote{}, fmt.Errorf("current release version is required")
	}
	if len(release.Changes) == 0 {
		return models.ReleaseNote{}, fmt.Errorf("current release must contain at least one change")
	}

	releasedAt, err := time.Parse("2006-01-02", release.ReleasedAt)
	if err != nil {
		return models.ReleaseNote{}, fmt.Errorf("invalid release date: %w", err)
	}

	changes := make([]models.ReleaseNoteChange, 0, len(release.Changes))
	for index, change := range release.Changes {
		change.Title = strings.TrimSpace(change.Title)
		change.Description = strings.TrimSpace(change.Description)
		if change.Title == "" || change.Description == "" {
			return models.ReleaseNote{}, fmt.Errorf("release change #%d must contain title and description", index+1)
		}

		changes = append(changes, models.ReleaseNoteChange{
			SortOrder:   index + 1,
			Title:       change.Title,
			Description: change.Description,
		})
	}

	return models.ReleaseNote{
		Version:    release.Version,
		ReleasedAt: releasedAt,
		IsCurrent:  true,
		Changes:    changes,
	}, nil
}

func getReleaseStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}

	return filepath.Join(configDir, "docflow", "state.json"), nil
}

func (s *ReleaseNoteService) readState() (releaseViewState, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	payload, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return releaseViewState{}, nil
		}
		return releaseViewState{}, fmt.Errorf("failed to read release state: %w", err)
	}

	var state releaseViewState
	if err := json.Unmarshal(payload, &state); err != nil {
		return releaseViewState{}, nil
	}

	return state, nil
}
