package config

import (
	"os"
	"path/filepath"

	storepkg "github.com/yinxulai/ait/internal/server/store"
)

const (
	appDirName    = ".ait"
	configJSON    = "config.json"
	tasksDirName  = "tasks"
	runsDirName   = "runs"
	runMetaJSON   = "run.json"
	runResultJSON = "result.json"
	runReqsJSONL  = "requests.jsonl"
)

type Config struct {
	SaveAPIKey         bool   `json:"save_api_key"`
	LastSelectedTaskID string `json:"last_selected_task_id,omitempty"`
	DefaultProtocol    string `json:"default_protocol,omitempty"`
	ProxyURL           string `json:"proxy_url,omitempty"`
	Lang               string `json:"lang,omitempty"` // "zh" or "en", empty = zh
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	loaded, err := storepkg.NewJSONStore[Config](path).Load()
	if err != nil {
		return nil, err
	}
	return &loaded, nil
}

func (c *Config) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return storepkg.NewJSONStore[Config](path).Save(*c)
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

func TasksDir() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tasksDirName), nil
}

func RunsDir() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runsDirName), nil
}

func TaskPath(taskID string) (string, error) {
	dir, err := TasksDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, taskID+".json"), nil
}

func RunDir(taskID, runID string) (string, error) {
	dir, err := RunsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, taskID, runID), nil
}

func RunMetadataPath(taskID, runID string) (string, error) {
	dir, err := RunDir(taskID, runID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runMetaJSON), nil
}

func RunResultPath(taskID, runID string) (string, error) {
	dir, err := RunDir(taskID, runID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runResultJSON), nil
}

func RunRequestsPath(taskID, runID string) (string, error) {
	dir, err := RunDir(taskID, runID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runReqsJSONL), nil
}
