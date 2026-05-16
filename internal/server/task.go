package server

import (
	"fmt"

	"github.com/yinxulai/ait/internal/types"
)

// ListTasks 返回所有任务（最近更新排在前面）。
func (s *serverImpl) ListTasks() []types.TaskDefinition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.taskStore.All()
}

// GetTask 按 ID 查找任务。
func (s *serverImpl) GetTask(id string) (types.TaskDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.taskStore.Get(id)
}

// CreateTask 新建任务并持久化。
func (s *serverImpl) CreateTask(cfg TaskConfig) (types.TaskDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := types.TaskDefinition{
		Name:  cfg.Name,
		Input: cfg.Input,
	}
	s.taskStore.Upsert(task)
	if err := s.taskStore.Save(); err != nil {
		return types.TaskDefinition{}, fmt.Errorf("save tasks: %w", err)
	}

	// 返回已生成 ID 和时间戳的最新状态
	all := s.taskStore.All()
	if len(all) > 0 {
		return all[0], nil
	}
	return task, nil
}

// UpdateTask 更新指定任务，任务不存在时返回错误。
func (s *serverImpl) UpdateTask(id string, cfg TaskConfig) (types.TaskDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.taskStore.Get(id)
	if !ok {
		return types.TaskDefinition{}, fmt.Errorf("task %q not found", id)
	}

	existing.Name = cfg.Name
	existing.Input = cfg.Input
	s.taskStore.Upsert(existing)

	if err := s.taskStore.Save(); err != nil {
		return types.TaskDefinition{}, fmt.Errorf("save tasks: %w", err)
	}

	updated, _ := s.taskStore.Get(id)
	return updated, nil
}

// DeleteTask 删除指定任务，任务不存在时返回错误。
func (s *serverImpl) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.taskStore.Delete(id); err != nil {
		return err
	}
	return s.taskStore.Save()
}

// CopyTask 复制指定任务（ID 和时间戳重置，名称加 " (copy)" 后缀）。
func (s *serverImpl) CopyTask(id string) (types.TaskDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	src, ok := s.taskStore.Get(id)
	if !ok {
		return types.TaskDefinition{}, fmt.Errorf("task %q not found", id)
	}

	copied := types.TaskDefinition{
		Name:  src.Name + " (copy)",
		Input: src.Input,
	}
	s.taskStore.Upsert(copied)

	if err := s.taskStore.Save(); err != nil {
		return types.TaskDefinition{}, fmt.Errorf("save tasks: %w", err)
	}

	all := s.taskStore.All()
	if len(all) > 0 {
		return all[0], nil
	}
	return copied, nil
}
