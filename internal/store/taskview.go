package store

import "github.com/yinxulai/ait/internal/types"

// TaskViewStore 负责聚合任务定义和最近一次已完成运行，用于列表读取。
type TaskViewStore struct {
	tasks *TaskStore
	runs  *RunStore
}

func NewTaskViewStore(tasks *TaskStore, runs *RunStore) *TaskViewStore {
	return &TaskViewStore{tasks: tasks, runs: runs}
}

func (s *TaskViewStore) List() ([]types.TaskOverview, error) {
	tasks, err := s.tasks.List()
	if err != nil {
		return nil, err
	}

	overviews := make([]types.TaskOverview, 0, len(tasks))
	for _, task := range tasks {
		overview := types.TaskOverview{TaskDefinition: task}
		latest, err := s.runs.LatestSummaryByTask(task.ID)
		if err != nil {
			return nil, err
		}
		if latest != nil {
			overview.LatestRun = latest
		}
		overviews = append(overviews, overview)
	}

	return overviews, nil
}
