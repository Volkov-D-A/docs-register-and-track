package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	appThemeLight = "light"
	appThemeDark  = "dark"
)

// ThemeService хранит локальную настройку темы интерфейса на устройстве.
type ThemeService struct {
	statePath string
	stateMu   sync.Mutex
}

type themeState struct {
	Theme string `json:"theme"`
}

// NewThemeService создает сервис локальных настроек темы.
func NewThemeService() (*ThemeService, error) {
	statePath, err := getThemeStatePath()
	if err != nil {
		return nil, err
	}

	return &ThemeService{statePath: statePath}, nil
}

// GetTheme возвращает локально выбранную пользователем тему интерфейса.
func (s *ThemeService) GetTheme() (string, error) {
	state, err := s.readState()
	if err != nil {
		return "", err
	}

	return resolveAppTheme(state.Theme), nil
}

// SetTheme сохраняет локально выбранную пользователем тему интерфейса.
func (s *ThemeService) SetTheme(theme string) error {
	normalizedTheme, ok := normalizeAppTheme(theme)
	if !ok {
		return fmt.Errorf("unsupported app theme %q", theme)
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	return s.writeStateLocked(themeState{Theme: normalizedTheme})
}

func getThemeStatePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}

	return filepath.Join(configDir, "docflow", "theme.json"), nil
}

func (s *ThemeService) readState() (themeState, error) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	payload, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return themeState{}, nil
		}
		return themeState{}, fmt.Errorf("failed to read theme state: %w", err)
	}

	var state themeState
	if err := json.Unmarshal(payload, &state); err != nil {
		return themeState{}, nil
	}

	return state, nil
}

func (s *ThemeService) writeStateLocked(state themeState) error {
	if err := os.MkdirAll(filepath.Dir(s.statePath), 0o755); err != nil {
		return fmt.Errorf("failed to create theme state directory: %w", err)
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode theme state: %w", err)
	}

	if err := os.WriteFile(s.statePath, payload, 0o600); err != nil {
		return fmt.Errorf("failed to save theme state: %w", err)
	}

	return nil
}

func resolveAppTheme(theme string) string {
	normalizedTheme, ok := normalizeAppTheme(theme)
	if !ok {
		return appThemeLight
	}

	return normalizedTheme
}

func normalizeAppTheme(theme string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case appThemeLight:
		return appThemeLight, true
	case appThemeDark:
		return appThemeDark, true
	default:
		return "", false
	}
}
