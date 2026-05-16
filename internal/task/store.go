package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/yinxulai/ait/internal/config"
	"github.com/yinxulai/ait/internal/types"
)

type TaskStore struct {
	Tasks []types.TaskDefinition `json:"tasks"`
}

func LoadTasks() (*TaskStore, error) {
	path, err := config.TasksPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &TaskStore{Tasks: []types.TaskDefinition{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var store TaskStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Tasks == nil {
		store.Tasks = []types.TaskDefinition{}
	}
	return &store, nil
}

func (s *TaskStore) Save() error {
	if _, err := config.EnsureAppDir(); err != nil {
		return err
	}
	path, err := config.TasksPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *TaskStore) Upsert(task types.TaskDefinition) {
	now := time.Now()
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", now.UnixNano())
	}

	for i, existing := range s.Tasks {
		if existing.ID != task.ID {
			continue
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = existing.CreatedAt
		}
		task.UpdatedAt = now
		updated := append([]types.TaskDefinition{task}, append(s.Tasks[:i], s.Tasks[i+1:]...)...)
		s.Tasks = updated
		return
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	s.Tasks = append([]types.TaskDefinition{task}, s.Tasks...)
}

func (s *TaskStore) Delete(taskID string) error {
	for i, task := range s.Tasks {
		if task.ID != taskID {
			continue
		}
		s.Tasks = append(s.Tasks[:i], s.Tasks[i+1:]...)
		return nil
	}
	return os.ErrNotExist
}

func (s *TaskStore) Get(taskID string) (types.TaskDefinition, bool) {
	for _, task := range s.Tasks {
		if task.ID == taskID {
			return task, true
		}
	}
	return types.TaskDefinition{}, false
}
