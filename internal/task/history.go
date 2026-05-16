package task

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/yinxulai/ait/internal/config"
	"github.com/yinxulai/ait/internal/types"
)

func AppendRun(taskID string, run types.TaskRunSummary) error {
	runs, err := loadHistoryFile(taskID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if run.TaskID == "" {
		run.TaskID = taskID
	}
	runs = append(runs, run)
	return saveHistoryFile(taskID, runs)
}

func LoadHistory(taskID string, limit int) ([]types.TaskRunSummary, error) {
	runs, err := loadHistoryFile(taskID)
	if errors.Is(err, os.ErrNotExist) {
		return []types.TaskRunSummary{}, nil
	}
	if err != nil {
		return nil, err
	}

	reversed := make([]types.TaskRunSummary, 0, len(runs))
	for i := len(runs) - 1; i >= 0; i-- {
		reversed = append(reversed, runs[i])
	}
	if limit > 0 && len(reversed) > limit {
		reversed = reversed[:limit]
	}
	return reversed, nil
}

func loadHistoryFile(taskID string) ([]types.TaskRunSummary, error) {
	path, err := historyPath(taskID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []types.TaskRunSummary{}, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}

	var runs []types.TaskRunSummary
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func saveHistoryFile(taskID string, runs []types.TaskRunSummary) error {
	dir, err := config.HistoryDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, taskID+".json")
	return os.WriteFile(path, data, 0o644)
}

func historyPath(taskID string) (string, error) {
	dir, err := config.HistoryDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, taskID+".json"), nil
}
