package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

type RunMetadata struct {
	RunID      string     `json:"run_id"`
	TaskID     string     `json:"task_id"`
	Mode       string     `json:"mode"`
	Protocol   string     `json:"protocol"`
	Model      string     `json:"model"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type RunResult struct {
	TotalReqs            int                `json:"total_reqs"`
	DoneReqs             int                `json:"done_reqs"`
	SuccessReqs          int                `json:"success_reqs"`
	FailedReqs           int                `json:"failed_reqs"`
	SuccessRate          float64            `json:"success_rate"`
	AvgTTFT              time.Duration      `json:"avg_ttft"`
	AvgTPS               float64            `json:"avg_tps"`
	CacheHitRate         float64            `json:"cache_hit_rate"`
	MaxStableConcurrency int                `json:"max_stable_concurrency,omitempty"`
	ErrorSummary         string             `json:"error_summary,omitempty"`
	StandardResult       *types.ReportData  `json:"standard_result,omitempty"`
	TurboResult          *types.TurboResult `json:"turbo_result,omitempty"`
}

type StoredRun struct {
	Metadata RunMetadata
	Result   *RunResult
}

type RunStore struct {
	root string
}

func NewRunStore(root string) *RunStore {
	return &RunStore{root: root}
}

func (s *RunStore) TaskDir(taskID string) string {
	return filepath.Join(s.root, taskID)
}

func (s *RunStore) RunDir(taskID, runID string) string {
	return filepath.Join(s.TaskDir(taskID), runID)
}

func (s *RunStore) MetadataPath(taskID, runID string) string {
	return filepath.Join(s.RunDir(taskID, runID), "run.json")
}

func (s *RunStore) ResultPath(taskID, runID string) string {
	return filepath.Join(s.RunDir(taskID, runID), "result.json")
}

func (s *RunStore) RequestsPath(taskID, runID string) string {
	return filepath.Join(s.RunDir(taskID, runID), "requests.jsonl")
}

func (s *RunStore) AppendRequest(taskID, runID string, request types.RequestMetrics) error {
	if taskID == "" || runID == "" {
		return fmt.Errorf("task id and run id are required")
	}
	if err := os.MkdirAll(s.RunDir(taskID, runID), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(request)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.RequestsPath(taskID, runID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	_, err = f.Write([]byte{'\n'})
	return err
}

func (s *RunStore) LoadRequests(taskID, runID string) ([]types.RequestMetrics, error) {
	f, err := os.Open(s.RequestsPath(taskID, runID))
	if os.IsNotExist(err) {
		return []types.RequestMetrics{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const maxLineSize = 16 * 1024 * 1024
	buf := make([]byte, maxLineSize)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf, maxLineSize)

	requests := make([]types.RequestMetrics, 0)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var request types.RequestMetrics
		if err := json.Unmarshal(line, &request); err != nil {
			continue
		}
		requests = append(requests, request)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(requests, func(i, j int) bool {
		return requests[i].Index < requests[j].Index
	})
	return requests, nil
}

func (s *RunStore) SaveFinal(meta RunMetadata, result RunResult) error {
	if meta.TaskID == "" || meta.RunID == "" {
		return fmt.Errorf("task id and run id are required")
	}
	if err := os.MkdirAll(s.RunDir(meta.TaskID, meta.RunID), 0o755); err != nil {
		return err
	}
	if err := NewJSONStore[RunMetadata](s.MetadataPath(meta.TaskID, meta.RunID)).Save(meta); err != nil {
		return err
	}
	return NewJSONStore[RunResult](s.ResultPath(meta.TaskID, meta.RunID)).Save(result)
}

func (s *RunStore) SaveSummary(summary types.TaskRunSummary) error {
	if summary.TaskID == "" || summary.RunID == "" {
		return fmt.Errorf("task id and run id are required")
	}

	var finishedAt *time.Time
	if !summary.FinishedAt.IsZero() {
		finished := summary.FinishedAt
		finishedAt = &finished
	}

	return s.SaveFinal(RunMetadata{
		RunID:      summary.RunID,
		TaskID:     summary.TaskID,
		Mode:       summary.Mode,
		Protocol:   summary.Protocol,
		Model:      summary.Model,
		Status:     summary.Status,
		StartedAt:  summary.StartedAt,
		FinishedAt: finishedAt,
	}, RunResult{
		SuccessRate:          summary.SuccessRate,
		AvgTTFT:              summary.AvgTTFT,
		AvgTPS:               summary.AvgTPS,
		CacheHitRate:         summary.CacheHitRate,
		MaxStableConcurrency: summary.MaxStableConcurrency,
		ErrorSummary:         summary.ErrorSummary,
	})
}

func (s *RunStore) Load(taskID, runID string) (*StoredRun, error) {
	metaPath := s.MetadataPath(taskID, runID)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	meta, err := NewJSONStore[RunMetadata](metaPath).Load()
	if err != nil {
		return nil, err
	}

	resultPath := s.ResultPath(taskID, runID)
	var result *RunResult
	if _, err := os.Stat(resultPath); err == nil {
		loaded, err := NewJSONStore[RunResult](resultPath).Load()
		if err != nil {
			return nil, err
		}
		result = &loaded
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return &StoredRun{Metadata: meta, Result: result}, nil
}

func (s *RunStore) LoadByRunID(runID string) (*StoredRun, error) {
	taskEntries, err := os.ReadDir(s.root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	for _, taskEntry := range taskEntries {
		if !taskEntry.IsDir() {
			continue
		}
		candidate, err := s.Load(taskEntry.Name(), runID)
		if err != nil {
			return nil, err
		}
		if candidate != nil {
			return candidate, nil
		}
	}

	return nil, nil
}

func (s *RunStore) ListByTask(taskID string, limit int) ([]StoredRun, error) {
	runEntries, err := os.ReadDir(s.TaskDir(taskID))
	if os.IsNotExist(err) {
		return []StoredRun{}, nil
	}
	if err != nil {
		return nil, err
	}

	runs := make([]StoredRun, 0, len(runEntries))
	for _, runEntry := range runEntries {
		if !runEntry.IsDir() {
			continue
		}
		run, err := s.Load(taskID, runEntry.Name())
		if err != nil {
			return nil, err
		}
		if run == nil {
			continue
		}
		runs = append(runs, *run)
	}

	sort.Slice(runs, func(i, j int) bool {
		return runSortTime(runs[i]).After(runSortTime(runs[j]))
	})

	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (s *RunStore) LatestByTask(taskID string) (*StoredRun, error) {
	runs, err := s.ListByTask(taskID, 1)
	if err != nil {
		return nil, err
	}
	if len(runs) == 0 {
		return nil, nil
	}
	return &runs[0], nil
}

func (s *RunStore) DeleteTask(taskID string) error {
	return os.RemoveAll(s.TaskDir(taskID))
}

func (r StoredRun) Summary() types.TaskRunSummary {
	summary := types.TaskRunSummary{
		RunID:     r.Metadata.RunID,
		TaskID:    r.Metadata.TaskID,
		Mode:      r.Metadata.Mode,
		Status:    r.Metadata.Status,
		Protocol:  r.Metadata.Protocol,
		Model:     r.Metadata.Model,
		StartedAt: r.Metadata.StartedAt,
	}
	if r.Metadata.FinishedAt != nil {
		summary.FinishedAt = *r.Metadata.FinishedAt
	}
	if r.Result != nil {
		summary.SuccessRate = r.Result.SuccessRate
		summary.AvgTTFT = r.Result.AvgTTFT
		summary.AvgTPS = r.Result.AvgTPS
		summary.CacheHitRate = r.Result.CacheHitRate
		summary.MaxStableConcurrency = r.Result.MaxStableConcurrency
		summary.ErrorSummary = r.Result.ErrorSummary
	}
	return summary
}

func runSortTime(run StoredRun) time.Time {
	if run.Metadata.FinishedAt != nil && !run.Metadata.FinishedAt.IsZero() {
		return *run.Metadata.FinishedAt
	}
	return run.Metadata.StartedAt
}
