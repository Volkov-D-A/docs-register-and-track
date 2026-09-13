package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigPathEnv         = "DOCFLOW_CONFIG_PATH"
	defaultConfigFileName = "config.json"
	defaultConfigDirName  = "config"
)

// Config contains only local desktop settings.
type Config struct {
	Server ServerConfig `json:"server"`
}
type ServerConfig struct {
	URL               string `json:"url,omitempty"`
	AllowInsecureHTTP bool   `json:"allowInsecureHttp,omitempty"`
}

// Load загружает конфигурацию из файла по указанному пути.
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetDefaultConfigPath возвращает путь к файлу конфигурации по умолчанию.
// Lookup order:
// 1. DOCFLOW_CONFIG_PATH, if set;
// 2. config/config.json next to the executable;
// 3. config/config.json in the current working directory for local development.
func GetDefaultConfigPath() string {
	if override := strings.TrimSpace(os.Getenv(ConfigPathEnv)); override != "" {
		return override
	}

	candidates := getDefaultConfigPathCandidates()
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return filepath.Join(defaultConfigDirName, defaultConfigFileName)
}

func getDefaultConfigPathCandidates() []string {
	executablePath, _ := os.Executable()
	workingDir, _ := os.Getwd()
	return buildDefaultConfigPathCandidates(executablePath, workingDir)
}

func buildDefaultConfigPathCandidates(executablePath, workingDir string) []string {
	var candidates []string
	seen := make(map[string]struct{})
	add := func(path string) {
		if strings.TrimSpace(path) == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		candidates = append(candidates, clean)
	}

	if executablePath != "" {
		add(filepath.Join(filepath.Dir(executablePath), defaultConfigDirName, defaultConfigFileName))
	}
	if workingDir != "" {
		add(filepath.Join(workingDir, defaultConfigDirName, defaultConfigFileName))
	}
	add(filepath.Join(defaultConfigDirName, defaultConfigFileName))
	return candidates
}
