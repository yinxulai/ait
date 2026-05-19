package server

import (
	"fmt"

	"github.com/yinxulai/ait/internal/types"
)

// ListTasks 返回所有任务（最近更新排在前面）。
func (s *serverImpl) ListTasks() []TaskOverview {
	s.mu.RLock()
	tasks := s.taskStore.All()
	running := s.runningTaskSummariesLocked(tasks)
	s.mu.RUnlock()

	return s.buildTaskOverviews(tasks, running)
}

// GetTask 按 ID 查找任务。
func (s *serverImpl) GetTask(id string) (types.TaskDefinition, bool) {
	s.mu.RLock()
	task, ok := s.taskStore.Get(id)
	s.mu.RUnlock()
	if !ok {
		return types.TaskDefinition{}, false
	}
	return task, true
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
	if err := s.taskStore.Save(); err != nil {
		return err
	}
	return s.runStore.DeleteTask(id)
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

func (s *serverImpl) buildTaskOverviews(tasks []types.TaskDefinition, running map[string]types.TaskRunSummary) []TaskOverview {
	decorated := make([]TaskOverview, 0, len(tasks))
	for _, task := range tasks {
		decorated = append(decorated, s.buildTaskOverview(task, running))
	}
	return decorated
}


func (s *serverImpl) buildTaskOverview(task types.TaskDefinition, running map[string]types.TaskRunSummary) TaskOverview {
	overview := TaskOverview{TaskDefinition: task}
	latest, err := s.runStore.LatestByTask(task.ID)
	if err == nil && latest != nil {
		summary := latest.Summary()
		overview.LatestRun = &summary
	}
	if summary, ok := running[task.ID]; ok {
		runningSummary := summary
		overview.LatestRun = &runningSummary
	}
	return overview
}

func (s *serverImpl) runningTaskSummariesLocked(tasks []types.TaskDefinition) map[string]types.TaskRunSummary {
	if len(tasks) == 0 || len(s.activeRuns) == 0 {
		return nil
	}

	taskByID := make(map[string]types.TaskDefinition, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task
	}

	running := make(map[string]types.TaskRunSummary)
	for _, ar := range s.activeRuns {
		ar.mu.RLock()
		if ar.state == nil || ar.state.Status != RunStatusRunning {
			ar.mu.RUnlock()
			continue
		}
		taskDef, ok := taskByID[ar.state.TaskID]
		if !ok {
			ar.mu.RUnlock()
			continue
		}
		summary := buildRunningRunSummary(taskDef, ar.snapshotState())
		ar.mu.RUnlock()
		running[taskDef.ID] = summary
	}

	if len(running) == 0 {
		return nil
	}
	return running
}
