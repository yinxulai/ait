package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

type persistedTaskDefinition struct {
	Name      string      `json:"name"`
	Input     types.Input `json:"input"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

var ErrTaskNotFound = errors.New("task not found")

// TaskStore 管理 ~/.ait/tasks/ 下的任务文件持久化。
// 它是无状态仓储：每次调用直接从磁盘读取或写入单任务文件。
type TaskStore struct {
	dir string
}

func NewTaskStore(dir string) *TaskStore {
	return &TaskStore{dir: dir}
}

func (s *TaskStore) List() ([]types.TaskDefinition, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return []types.TaskDefinition{}, nil
	}
	if err != nil {
		return nil, err
	}

	tasks := make([]types.TaskDefinition, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.dir, entry.Name())
		stored, err := NewJSONStore[persistedTaskDefinition](path).Load()
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, normalizeTaskDefinition(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), stored))
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	return tasks, nil
}

func (s *TaskStore) Get(id string) (types.TaskDefinition, error) {
	if strings.TrimSpace(id) == "" {
		return types.TaskDefinition{}, ErrTaskNotFound
	}

	path := s.taskPath(id)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return types.TaskDefinition{}, ErrTaskNotFound
	} else if err != nil {
		return types.TaskDefinition{}, err
	}

	stored, err := NewJSONStore[persistedTaskDefinition](path).Load()
	if err != nil {
		return types.TaskDefinition{}, err
	}
	return normalizeTaskDefinition(id, stored), nil
}

func (s *TaskStore) Create(task types.TaskDefinition) (types.TaskDefinition, error) {
	now := time.Now()
	if strings.TrimSpace(task.ID) == "" {
		task.ID = fmt.Sprintf("task_%d", now.UnixNano())
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	if err := s.writeTask(task); err != nil {
		return types.TaskDefinition{}, err
	}
	return task, nil
}

func (s *TaskStore) Update(task types.TaskDefinition) (types.TaskDefinition, error) {
	if strings.TrimSpace(task.ID) == "" {
		return types.TaskDefinition{}, ErrTaskNotFound
	}

	existing, err := s.Get(task.ID)
	if err != nil {
		return types.TaskDefinition{}, err
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = existing.CreatedAt
	}
	task.UpdatedAt = time.Now()
	if err := s.writeTask(task); err != nil {
		return types.TaskDefinition{}, err
	}
	return task, nil
}

func (s *TaskStore) Delete(id string) error {
	err := os.Remove(s.taskPath(id))
	if os.IsNotExist(err) {
		return ErrTaskNotFound
	}
	return err
}

func (s *TaskStore) taskPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *TaskStore) writeTask(task types.TaskDefinition) error {
	if strings.TrimSpace(task.ID) == "" {
		return fmt.Errorf("task id cannot be empty")
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	return NewJSONStore[persistedTaskDefinition](s.taskPath(task.ID)).Save(persistedTaskDefinition{
		Name:      task.Name,
		Input:     task.Input,
		CreatedAt: task.CreatedAt,
		UpdatedAt: task.UpdatedAt,
	})
}

func normalizeTaskDefinition(fallbackID string, stored persistedTaskDefinition) types.TaskDefinition {
	return types.TaskDefinition{
		ID:        fallbackID,
		Name:      stored.Name,
		Input:     stored.Input,
		CreatedAt: stored.CreatedAt,
		UpdatedAt: stored.UpdatedAt,
	}
}
