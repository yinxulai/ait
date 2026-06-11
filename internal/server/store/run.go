package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/yinxulai/ait/internal/server/types"
)

type RunMetadata struct {
	RunID      string     `json:"-"`
	TaskID     string     `json:"-"`
	Mode       string     `json:"mode"`
	Protocol   string     `json:"protocol"`
	Model      string     `json:"model"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type RunResult struct {
	TotalReqs            int    `json:"total_reqs,omitempty"`
	MaxStableConcurrency int    `json:"max_stable_concurrency,omitempty"`
	ErrorSummary         string `json:"error_summary,omitempty"`
	// ModeResult 存储模式特定的运行结果（泛型）
	// 根据 RunMetadata.Mode 判断具体类型：
	// - standard: types.ReportData
	// - turbo: types.TurboResult
	// - integrity: types.IntegrityResult
	ModeResult any `json:"mode_result,omitempty"`

	// 向后兼容：保留旧字段用于迁移
	StandardResult  *types.ReportData      `json:"standard_result,omitempty"`
	TurboResult     *types.TurboResult     `json:"turbo_result,omitempty"`
	IntegrityResult *types.IntegrityResult `json:"integrity_result,omitempty"`

	// 预计算汇总字段（避免每次加载任务列表时重新扫描 requests.jsonl）
	DoneReqs     int           `json:"done_reqs,omitempty"`
	SuccessReqs  int           `json:"success_reqs,omitempty"`
	FailedReqs   int           `json:"failed_reqs,omitempty"`
	SuccessRate  float64       `json:"success_rate,omitempty"`
	AvgTTFT      time.Duration `json:"avg_ttft,omitempty"`
	AvgTPS       float64       `json:"avg_tps,omitempty"`
	CacheHitRate float64       `json:"cache_hit_rate,omitempty"`
	RPM          float64       `json:"rpm,omitempty"`
	TPM          float64       `json:"tpm,omitempty"`
}

type StoredRun struct {
	Metadata RunMetadata
	Result   *RunResult
}

type RunStore struct {
	root string
	mu   sync.Mutex
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
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.RequestsPath(taskID, runID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
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
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var request types.RequestMetrics
		if err := json.Unmarshal(line, &request); err != nil {
			return nil, fmt.Errorf("parse requests jsonl line %d: %w", lineNo, err)
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

func (s *RunStore) SaveFinalRun(meta RunMetadata, result RunResult) error {
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
	meta.TaskID = taskID
	meta.RunID = runID

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

func (s *RunStore) DeleteTaskRuns(taskID string) error {
	return os.RemoveAll(s.TaskDir(taskID))
}

func (s *RunStore) LoadSummary(taskID, runID string) (*types.TaskRunSummary, error) {
	run, err := s.Load(taskID, runID)
	if err != nil || run == nil {
		return nil, err
	}
	// 优先从 result.json 预存字段生成摘要，不依赖 requests.jsonl
	summary := run.Summary(nil)
	return &summary, nil
}

func (s *RunStore) LatestSummaryByTask(taskID string) (*types.TaskRunSummary, error) {
	summaries, err := s.ListSummariesByTask(taskID, 1)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, nil
	}
	return &summaries[0], nil
}

func (s *RunStore) ListSummariesByTask(taskID string, limit int) ([]types.TaskRunSummary, error) {
	runs, err := s.ListByTask(taskID, limit)
	if err != nil {
		return nil, err
	}
	summaries := make([]types.TaskRunSummary, 0, len(runs))
	for _, run := range runs {
		// 优先从 result.json 中预存字段生成摘要，避免扫描 requests.jsonl
		summary := run.Summary(nil)
		// 对于旧运行，预存字段可能为空；此时回退到加载 requests.jsonl
		if summary.SuccessRate == 0 && summary.AvgTTFT == 0 && summary.AvgTPS == 0 {
			if requests, err := s.LoadRequests(run.Metadata.TaskID, run.Metadata.RunID); err == nil {
				summary = run.Summary(requests)
			}
			// 如果 requests.jsonl 损坏或不存在，保持零值，不阻断列表加载
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r StoredRun) Summary(requests []types.RequestMetrics) types.TaskRunSummary {
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

	// 优先使用预计算汇总字段（避免每次加载任务列表时重新扫描 requests.jsonl）
	if r.Result != nil {
		summary.ErrorSummary = r.Result.ErrorSummary
		summary.MaxStableConcurrency = r.Result.MaxStableConcurrency
		if r.Result.DoneReqs > 0 || r.Result.SuccessReqs > 0 {
			summary.SuccessRate = r.Result.SuccessRate
			summary.AvgTTFT = r.Result.AvgTTFT
			summary.AvgTPS = r.Result.AvgTPS
			summary.CacheHitRate = r.Result.CacheHitRate
			summary.RPM = r.Result.RPM
			summary.TPM = r.Result.TPM
		}
		// 模式特定字段（优先级高于预存汇总）
		if r.Result.ModeResult != nil {
			switch result := r.Result.ModeResult.(type) {
			case *types.ReportData:
				summary.SuccessRate = result.SuccessRate
				summary.AvgTTFT = result.AvgTTFT
				summary.AvgTPS = result.AvgTPS
				summary.CacheHitRate = result.AvgCacheHitRate
				summary.RPM = result.RPM
				summary.TPM = result.TPM
			case *types.TurboResult:
				summary.MaxStableConcurrency = result.MaxStableConcurrency
			case *types.IntegrityResult:
				if result.TotalCases > 0 {
					summary.SuccessRate = float64(result.PassedCases) / float64(result.TotalCases) * 100
				}
				if result.RequiredFailedCases > 0 || result.FailedCases > 0 {
					summary.ErrorSummary = fmt.Sprintf("%d/%d integrity cases failed", result.FailedCases, result.TotalCases)
				}
			}
		} else {
			// 向后兼容：尝试从旧字段读取
			if r.Result.StandardResult != nil {
				summary.SuccessRate = r.Result.StandardResult.SuccessRate
				summary.AvgTTFT = r.Result.StandardResult.AvgTTFT
				summary.AvgTPS = r.Result.StandardResult.AvgTPS
				summary.CacheHitRate = r.Result.StandardResult.AvgCacheHitRate
				summary.RPM = r.Result.StandardResult.RPM
				summary.TPM = r.Result.StandardResult.TPM
			}
			if r.Result.TurboResult != nil {
				summary.MaxStableConcurrency = r.Result.TurboResult.MaxStableConcurrency
			}
			if r.Result.IntegrityResult != nil {
				if r.Result.IntegrityResult.TotalCases > 0 {
					summary.SuccessRate = float64(r.Result.IntegrityResult.PassedCases) / float64(r.Result.IntegrityResult.TotalCases) * 100
				}
				if r.Result.IntegrityResult.RequiredFailedCases > 0 || r.Result.IntegrityResult.FailedCases > 0 {
					summary.ErrorSummary = fmt.Sprintf("%d/%d integrity cases failed", r.Result.IntegrityResult.FailedCases, r.Result.IntegrityResult.TotalCases)
				}
			}
		}
	}

	// 如果既没有预存字段也没有 requests，就返回已填充的基本字段
	if summary.SuccessRate == 0 && summary.AvgTTFT == 0 && summary.AvgTPS == 0 && len(requests) > 0 {
		derived := summarizeRequests(requests)
		summary.SuccessRate = derived.SuccessRate
		summary.AvgTTFT = derived.AvgTTFT
		summary.AvgTPS = derived.AvgTPS
		summary.CacheHitRate = derived.CacheHitRate

		// 从时间信息计算 RPM/TPM
		if !r.Metadata.StartedAt.IsZero() {
			end := time.Now()
			if r.Metadata.FinishedAt != nil {
				end = *r.Metadata.FinishedAt
			}
			if elapsed := end.Sub(r.Metadata.StartedAt).Minutes(); elapsed > 0 {
				var totalTokens int64
				for _, req := range requests {
					if req.Success {
						totalTokens += int64(req.CompletionTokens)
					}
				}
				summary.RPM = float64(len(requests)) / elapsed
				summary.TPM = float64(totalTokens) / elapsed
			}
		}
	}

	return summary
}

func (r StoredRun) TotalReqs(requests []types.RequestMetrics) int {
	if r.Result == nil {
		return len(requests)
	}
	// 优先使用 ModeResult 泛型字段
	if r.Result.ModeResult != nil {
		switch result := r.Result.ModeResult.(type) {
		case *types.ReportData:
			if result.TotalRequests > 0 {
				return result.TotalRequests
			}
		case *types.TurboResult:
			total := 0
			for _, level := range result.Levels {
				total += level.TotalRequests
			}
			if total > 0 {
				return total
			}
		case *types.IntegrityResult:
			if result.TotalCases > 0 {
				return result.TotalCases
			}
		}
	}
	// 向后兼容:尝试从旧字段读取
	if r.Result.StandardResult != nil && r.Result.StandardResult.TotalRequests > 0 {
		return r.Result.StandardResult.TotalRequests
	}
	if r.Result.TurboResult != nil {
		total := 0
		for _, level := range r.Result.TurboResult.Levels {
			total += level.TotalRequests
		}
		if total > 0 {
			return total
		}
	}
	if r.Result.IntegrityResult != nil && r.Result.IntegrityResult.TotalCases > 0 {
		return r.Result.IntegrityResult.TotalCases
	}
	if r.Result.TotalReqs > 0 {
		return r.Result.TotalReqs
	}
	return len(requests)
}

type requestSummary struct {
	DoneReqs     int
	SuccessReqs  int
	FailedReqs   int
	SuccessRate  float64
	AvgTTFT      time.Duration
	AvgTPS       float64
	CacheHitRate float64
}

func summarizeRequests(requests []types.RequestMetrics) requestSummary {
	summary := requestSummary{DoneReqs: len(requests)}
	var ttftSum time.Duration
	var tpsSum float64
	var cacheSum float64

	for _, request := range requests {
		if !request.Success {
			continue
		}
		summary.SuccessReqs++
		ttftSum += request.TTFT
		tpsSum += request.TPS
		cacheSum += request.CacheHitRate
	}

	summary.FailedReqs = summary.DoneReqs - summary.SuccessReqs
	if summary.DoneReqs > 0 {
		summary.SuccessRate = float64(summary.SuccessReqs) / float64(summary.DoneReqs) * 100
	}
	if summary.SuccessReqs > 0 {
		summary.AvgTTFT = ttftSum / time.Duration(summary.SuccessReqs)
		summary.AvgTPS = tpsSum / float64(summary.SuccessReqs)
		summary.CacheHitRate = cacheSum / float64(summary.SuccessReqs)
	}

	return summary
}

func runSortTime(run StoredRun) time.Time {
	if run.Metadata.FinishedAt != nil && !run.Metadata.FinishedAt.IsZero() {
		return *run.Metadata.FinishedAt
	}
	return run.Metadata.StartedAt
}
