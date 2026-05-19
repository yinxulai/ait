package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

type persistedTaskDefinition struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Input     types.Input `json:"input"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// TaskStore 管理 ~/.ait/tasks/ 下的任务文件持久化。
type TaskStore struct {
	dir  string
	data []types.TaskDefinition
}

// NewTaskStore 创建持久化到 dir 的 TaskStore（调用方需先调用 Load）。
func NewTaskStore(dir string) *TaskStore {
	return &TaskStore{dir: dir}
}

// Load 从磁盘加载任务列表，目录不存在时初始化为空列表。
func (s *TaskStore) Load() error {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		s.data = []types.TaskDefinition{}
		return nil
	}
	if err != nil {
		return err
	}

	tasks := make([]types.TaskDefinition, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(s.dir, entry.Name())
		stored, err := NewJSONStore[persistedTaskDefinition](path).Load()
		if err != nil {
			return err
		}
		if strings.TrimSpace(stored.ID) == "" {
			stored.ID = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		}
		tasks = append(tasks, types.TaskDefinition{
			ID:        stored.ID,
			Name:      stored.Name,
			Input:     stored.Input,
			CreatedAt: stored.CreatedAt,
			UpdatedAt: stored.UpdatedAt,
		})
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})

	s.data = tasks
	return nil
}

// Save 将当前内存中的任务列表持久化到磁盘，按任务拆分成独立文件。
func (s *TaskStore) Save() error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}

	keep := make(map[string]struct{}, len(s.data))
	for _, task := range s.data {
		if strings.TrimSpace(task.ID) == "" {
			return fmt.Errorf("task id cannot be empty")
		}
		keep[task.ID] = struct{}{}
		path := filepath.Join(s.dir, task.ID+".json")
		stored := persistedTaskDefinition{
			ID:        task.ID,
			Name:      task.Name,
			Input:     task.Input,
			CreatedAt: task.CreatedAt,
			UpdatedAt: task.UpdatedAt,
		}
		if err := NewJSONStore[persistedTaskDefinition](path).Save(stored); err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		taskID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if _, ok := keep[taskID]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// All 返回所有任务的副本，最近更新的排在前面。
func (s *TaskStore) All() []types.TaskDefinition {
	result := make([]types.TaskDefinition, len(s.data))
	copy(result, s.data)
	return result
}

// Get 按 ID 查找任务，返回副本。
func (s *TaskStore) Get(id string) (types.TaskDefinition, bool) {
	for _, t := range s.data {
		if t.ID == id {
			return t, true
		}
	}
	return types.TaskDefinition{}, false
}

// Upsert 新建或更新任务。
// - 若 task.ID 为空，自动生成唯一 ID。
// - 更新时将任务移至列表头部（最近活跃排序）。
func (s *TaskStore) Upsert(task types.TaskDefinition) {
	now := time.Now()
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", now.UnixNano())
	}

	for i, existing := range s.data {
		if existing.ID != task.ID {
			continue
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = existing.CreatedAt
		}
		task.UpdatedAt = now
		// 移至列表头部
		tasks := make([]types.TaskDefinition, 0, len(s.data))
		tasks = append(tasks, task)
		tasks = append(tasks, s.data[:i]...)
		tasks = append(tasks, s.data[i+1:]...)
		s.data = tasks
		return
	}

	// 新增
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	s.data = append([]types.TaskDefinition{task}, s.data...)
}

// Delete 按 ID 删除任务，任务不存在时返回错误。
func (s *TaskStore) Delete(id string) error {
	for i, t := range s.data {
		if t.ID == id {
			s.data = append(s.data[:i], s.data[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task %q not found", id)
}
