package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	appDirName    = ".ait"
	configJSON    = "config.json"
	tasksJSON     = "tasks.json"
	historyDirName = "history"
)

type Config struct {
	SaveAPIKey         bool   `json:"save_api_key"`
	LastSelectedTaskID string `json:"last_selected_task_id,omitempty"`
	DefaultProtocol    string `json:"default_protocol,omitempty"`
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if _, err := EnsureAppDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func AppDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, appDirName), nil
}

func EnsureAppDir() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ConfigPath() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configJSON), nil
}

func TasksPath() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tasksJSON), nil
}

func HistoryDir() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, historyDirName), nil
}
