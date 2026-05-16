package store

import (
	"fmt"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

type taskStoreData struct {
	Tasks []types.TaskDefinition `json:"tasks"`
}

// TaskStore 管理 ~/.ait/tasks.json 的任务列表持久化。
type TaskStore struct {
	store *JSONStore[taskStoreData]
	data  taskStoreData
}

// NewTaskStore 创建持久化到 path 的 TaskStore（调用方需先调用 Load）。
func NewTaskStore(path string) *TaskStore {
	return &TaskStore{store: NewJSONStore[taskStoreData](path)}
}

// Load 从磁盘加载任务列表，文件不存在时初始化为空列表。
func (s *TaskStore) Load() error {
	data, err := s.store.Load()
	if err != nil {
		return err
	}
	if data.Tasks == nil {
		data.Tasks = []types.TaskDefinition{}
	}
	s.data = data
	return nil
}

// Save 将当前内存中的任务列表持久化到磁盘。
func (s *TaskStore) Save() error {
	return s.store.Save(s.data)
}

// All 返回所有任务的副本，最近更新的排在前面。
func (s *TaskStore) All() []types.TaskDefinition {
	result := make([]types.TaskDefinition, len(s.data.Tasks))
	copy(result, s.data.Tasks)
	return result
}

// Get 按 ID 查找任务，返回副本。
func (s *TaskStore) Get(id string) (types.TaskDefinition, bool) {
	for _, t := range s.data.Tasks {
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

	for i, existing := range s.data.Tasks {
		if existing.ID != task.ID {
			continue
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = existing.CreatedAt
		}
		task.UpdatedAt = now
		// 移至列表头部
		tasks := make([]types.TaskDefinition, 0, len(s.data.Tasks))
		tasks = append(tasks, task)
		tasks = append(tasks, s.data.Tasks[:i]...)
		tasks = append(tasks, s.data.Tasks[i+1:]...)
		s.data.Tasks = tasks
		return
	}

	// 新增
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	s.data.Tasks = append([]types.TaskDefinition{task}, s.data.Tasks...)
}

// Delete 按 ID 删除任务，任务不存在时返回错误。
func (s *TaskStore) Delete(id string) error {
	for i, t := range s.data.Tasks {
		if t.ID == id {
			s.data.Tasks = append(s.data.Tasks[:i], s.data.Tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task %q not found", id)
}
