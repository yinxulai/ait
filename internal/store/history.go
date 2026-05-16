package store

import "github.com/yinxulai/ait/internal/types"

// HistoryStore 管理单个任务的运行历史文件（~/.ait/history/<task-id>.json）。
// 每个任务对应独立的 HistoryStore 实例和独立的文件。
type HistoryStore struct {
	store *JSONStore[[]types.TaskRunSummary]
}

// NewHistoryStore 创建持久化到 path 的 HistoryStore。
func NewHistoryStore(path string) *HistoryStore {
	return &HistoryStore{store: NewJSONStore[[]types.TaskRunSummary](path)}
}

// Append 追加一条运行摘要到历史文件。
func (s *HistoryStore) Append(run types.TaskRunSummary) error {
	runs, err := s.store.Load()
	if err != nil {
		return err
	}
	if runs == nil {
		runs = []types.TaskRunSummary{}
	}
	runs = append(runs, run)
	return s.store.Save(runs)
}

// Load 返回运行历史，最新的排在前面。limit <= 0 表示不限制条数。
func (s *HistoryStore) Load(limit int) ([]types.TaskRunSummary, error) {
	runs, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if runs == nil {
		return []types.TaskRunSummary{}, nil
	}

	// 反转（最新在前）
	reversed := make([]types.TaskRunSummary, len(runs))
	for i, r := range runs {
		reversed[len(runs)-1-i] = r
	}

	if limit > 0 && len(reversed) > limit {
		reversed = reversed[:limit]
	}
	return reversed, nil
}
