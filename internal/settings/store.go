package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"file-transfer-client/internal/model"
)

type Store struct {
	path string
}

func NewStore(configPath string) *Store {
	if strings.TrimSpace(configPath) == "" {
		configPath = defaultConfigPath()
	}
	return &Store{path: configPath}
}

func (s *Store) Load() (model.Settings, error) {
	defaults := defaultSettings()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaults, nil
		}
		return defaults, fmt.Errorf("read settings: %w", err)
	}

	var current model.Settings
	if err := json.Unmarshal(data, &current); err != nil {
		return defaults, fmt.Errorf("decode settings: %w", err)
	}

	return normalizeSettings(defaults, current), nil
}

func (s *Store) Save(current model.Settings) error {
	normalized := normalizeSettings(defaultSettings(), current)

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("write settings temp file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("persist settings: %w", err)
	}
	return nil
}

func (s *Store) Update(mutator func(*model.Settings)) (model.Settings, error) {
	current, err := s.Load()
	if err != nil {
		return model.Settings{}, err
	}

	if mutator != nil {
		mutator(&current)
	}

	if err := s.Save(current); err != nil {
		return model.Settings{}, err
	}
	return normalizeSettings(defaultSettings(), current), nil
}

func defaultConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "config.json")
	}
	return filepath.Join(configDir, "file-transfer-client", "config.json")
}

func defaultSettings() model.Settings {
	return model.Settings{
		LastRemotePath:     "/",
		DefaultDownloadDir: defaultDownloadDir(),
		RecentServers:      []string{},
	}
}

func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}

	downloads := filepath.Join(home, "Downloads")
	info, err := os.Stat(downloads)
	if err == nil && info.IsDir() {
		return downloads
	}

	return home
}

func normalizeSettings(defaults, current model.Settings) model.Settings {
	if strings.TrimSpace(current.LastRemotePath) == "" {
		current.LastRemotePath = defaults.LastRemotePath
	}
	if strings.TrimSpace(current.DefaultDownloadDir) == "" {
		current.DefaultDownloadDir = defaults.DefaultDownloadDir
	}

	current.LastServerURL = strings.TrimSpace(current.LastServerURL)
	current.LastRemotePath = strings.TrimSpace(current.LastRemotePath)
	if current.LastRemotePath == "" {
		current.LastRemotePath = "/"
	}

	current.RecentServers = normalizeRecents(current.RecentServers, 6)
	return current
}

func normalizeRecents(items []string, limit int) []string {
	seen := make(map[string]struct{}, len(items))
	next := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		next = append(next, trimmed)
		if limit > 0 && len(next) >= limit {
			break
		}
	}
	return next
}
