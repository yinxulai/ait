package server

import (
	"errors"
	"fmt"

	"github.com/yinxulai/ait/internal/server/config"
	storepkg "github.com/yinxulai/ait/internal/server/store"
	"github.com/yinxulai/ait/internal/server/types"
)

// ListTasks 返回所有任务（最近更新排在前面）。
func (s *serverImpl) ListTasks() ([]types.TaskOverview, error) {
	overviews, err := s.taskViews.List()
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	running := s.runningTaskSummariesLocked(overviews)
	s.mu.RUnlock()

	return s.overlayRunningTaskOverviews(overviews, running), nil
}

// GetTask 按 ID 查找任务。
func (s *serverImpl) GetTask(id string) (types.TaskDefinition, error) {
	return s.taskStore.Get(id)
}

// CreateTask 新建任务并持久化。
func (s *serverImpl) CreateTask(cfg TaskConfig) (types.TaskDefinition, error) {
	created, err := s.taskStore.Create(types.TaskDefinition{
		Name:  cfg.Name,
		Input: cfg.Input,
	})
	if err != nil {
		return types.TaskDefinition{}, fmt.Errorf("create task: %w", err)
	}
	return created, nil
}

// UpdateTask 更新指定任务，任务不存在时返回错误。
func (s *serverImpl) UpdateTask(id string, cfg TaskConfig) (types.TaskDefinition, error) {
	existing, err := s.taskStore.Get(id)
	if err != nil {
		if errors.Is(err, storepkg.ErrTaskNotFound) {
			return types.TaskDefinition{}, fmt.Errorf("task %q not found: %w", id, err)
		}
		return types.TaskDefinition{}, fmt.Errorf("get task %q: %w", id, err)
	}

	existing.Name = cfg.Name
	existing.Input = cfg.Input

	updated, err := s.taskStore.Update(existing)
	if err != nil {
		return types.TaskDefinition{}, fmt.Errorf("update task %q: %w", id, err)
	}
	return updated, nil
}

// DeleteTask 删除指定任务，任务不存在时返回错误。
func (s *serverImpl) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasRunningTaskLocked(id) {
		return fmt.Errorf("task %q is currently running", id)
	}
	if err := s.taskStore.Delete(id); err != nil {
		return err
	}
	return s.runStore.DeleteTaskRuns(id)
}

// DuplicateTask 复制指定任务（ID 和时间戳重置，名称加 " (copy)" 后缀）。
func (s *serverImpl) DuplicateTask(id string) (types.TaskDefinition, error) {
	src, err := s.taskStore.Get(id)
	if err != nil {
		if errors.Is(err, storepkg.ErrTaskNotFound) {
			return types.TaskDefinition{}, fmt.Errorf("task %q not found: %w", id, err)
		}
		return types.TaskDefinition{}, fmt.Errorf("get task %q: %w", id, err)
	}

	created, err := s.taskStore.Create(types.TaskDefinition{
		Name:  src.Name + " (copy)",
		Input: src.Input,
	})
	if err != nil {
		return types.TaskDefinition{}, fmt.Errorf("duplicate task %q: %w", id, err)
	}
	return created, nil
}

func (s *serverImpl) overlayRunningTaskOverviews(tasks []types.TaskOverview, running map[string]types.TaskRunSummary) []types.TaskOverview {
	if len(running) == 0 {
		return tasks
	}

	overlaid := make([]types.TaskOverview, len(tasks))
	copy(overlaid, tasks)
	for i := range overlaid {
		if summary, ok := running[overlaid[i].ID]; ok {
			runningSummary := summary
			overlaid[i].LatestRun = &runningSummary
		}
	}
	return overlaid
}

func (s *serverImpl) runningTaskSummariesLocked(tasks []types.TaskOverview) map[string]types.TaskRunSummary {
	if len(tasks) == 0 || len(s.activeRuns) == 0 {
		return nil
	}

	taskByID := make(map[string]types.TaskDefinition, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task.TaskDefinition
	}

	running := make(map[string]types.TaskRunSummary)
	for _, ar := range s.activeRuns {
		ar.mu.RLock()
		snapshot := ar.snapshotState()
		ar.mu.RUnlock()
		if snapshot == nil || (snapshot.Status != RunStatusQueued && snapshot.Status != RunStatusRunning) {
			continue
		}
		taskDef, ok := taskByID[snapshot.TaskID]
		if !ok {
			continue
		}
		running[taskDef.ID] = buildRunningRunSummary(taskDef, snapshot)
	}

	if len(running) == 0 {
		return nil
	}
	return running
}

func (s *serverImpl) hasRunningTaskLocked(taskID string) bool {
	for _, ar := range s.activeRuns {
		ar.mu.RLock()
		active := ar.state != nil && ar.state.TaskID == taskID && (ar.state.Status == RunStatusQueued || ar.state.Status == RunStatusRunning)
		ar.mu.RUnlock()
		if active {
			return true
		}
	}
	return false
}

// ─── 全局配置 ─────────────────────────────────────────────────────────────────

// GetAppConfig 返回当前全局配置。
func (s *serverImpl) GetAppConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return &config.Config{}, nil // 文件不存在时返回空配置
	}
	return cfg, nil
}

// UpdateProxyURL 更新并持久化全局代理 URL。
func (s *serverImpl) UpdateProxyURL(proxyURL string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	cfg.ProxyURL = proxyURL
	return cfg.Save()
}
