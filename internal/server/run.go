package server

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/report"
	"github.com/yinxulai/ait/internal/runner"
	"github.com/yinxulai/ait/internal/store"
	"github.com/yinxulai/ait/internal/task"
	"github.com/yinxulai/ait/internal/turbo"
	"github.com/yinxulai/ait/internal/types"
)

// activeRun 持有一次正在执行的运行的全部运行时状态。
type activeRun struct {
	mu          sync.RWMutex
	state       *RunState
	rnr         *runner.Runner  // standard 模式使用
	turboEngine *turbo.Engine   // turbo 模式使用
	// 用于计算实时均值
	tpsSum       float64
	ttftSum      time.Duration
	cacheSum     float64
	doneCount    int // 与 state.DoneReqs 保持同步，方便不加锁时计算
}

// snapshotState 返回 state 的深度拷贝（调用方须已持有 activeRun.mu 读锁）。
func (ar *activeRun) snapshotState() *RunState {
	s := ar.state
	snap := *s
	// 深拷贝切片
	if len(s.Requests) > 0 {
		snap.Requests = make([]*RequestMetrics, len(s.Requests))
		copy(snap.Requests, s.Requests)
	}
	if len(s.Levels) > 0 {
		snap.Levels = make([]types.TurboLevelResult, len(s.Levels))
		copy(snap.Levels, s.Levels)
	}
	return &snap
}

// mapRequestMetrics 将 client.ResponseMetrics 映射到 server.RequestMetrics。
func mapRequestMetrics(m *client.ResponseMetrics, idx int, err error) *RequestMetrics {
	rm := &RequestMetrics{Index: idx}
	if m == nil {
		rm.Success = false
		if err != nil {
			rm.ErrorMessage = err.Error()
		}
		return rm
	}

	rm.Success = m.ErrorMessage == "" && err == nil
	rm.TotalTime = m.TotalTime
	rm.TTFT = m.TimeToFirstToken
	rm.PromptTokens = m.PromptTokens
	rm.CompletionTokens = m.CompletionTokens
	rm.CachedTokens = m.CachedInputTokens
	rm.DNSTime = m.DNSTime
	rm.ConnectTime = m.ConnectTime
	rm.TLSTime = m.TLSHandshakeTime
	rm.TargetIP = m.TargetIP
	rm.ErrorMessage = m.ErrorMessage
	if err != nil && rm.ErrorMessage == "" {
		rm.ErrorMessage = err.Error()
	}
	rm.RequestBody = m.RequestBody
	rm.ResponseBody = m.ResponseBody

	if m.TotalTime > 0 && m.CompletionTokens > 0 {
		rm.TPS = float64(m.CompletionTokens) / m.TotalTime.Seconds()
	}
	if m.PromptTokens > 0 {
		rm.CacheHitRate = float64(m.CachedInputTokens) / float64(m.PromptTokens)
	}
	return rm
}

// historyPath 返回指定任务的历史文件路径。
func historyPath(historyDir, taskID string) string {
	return filepath.Join(historyDir, taskID+".json")
}

// runStatePath 返回指定运行的完整状态快照文件路径（用于历史回放）。
func runStatePath(historyDir string, runID RunID) string {
	return filepath.Join(historyDir, "runs", string(runID)+".json")
}

// persistRunState 将完整 RunState 快照写入磁盘，供历史回放使用。
func persistRunState(historyDir string, snap *RunState) {
	st := store.NewJSONStore[*RunState](runStatePath(historyDir, snap.RunID))
	_ = st.Save(snap) // 失败不影响主流程
}

// StartRun 启动一次新的运行，立即返回 RunID。
func (s *serverImpl) StartRun(taskID string) (RunID, error) {
	s.mu.RLock()
	taskDef, ok := s.taskStore.Get(taskID)
	historyDir := s.historyDir
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("task %q not found", taskID)
	}

	// 解析 PromptSource（将 PromptText/PromptFile 转换为可调用的 PromptSource）
	hydratedInput, err := task.HydrateInput(taskDef.Input)
	if err != nil {
		return "", fmt.Errorf("hydrate input: %w", err)
	}

	runID := RunID(fmt.Sprintf("run_%d", time.Now().UnixNano()))
	now := time.Now()

	mode := "standard"
	if hydratedInput.Turbo {
		mode = "turbo"
	}

	state := &RunState{
		RunID:       runID,
		TaskID:      taskID,
		Status:      RunStatusRunning,
		Mode:        mode,
		StartedAt:   now,
		TotalReqs:   hydratedInput.Count,
		Requests:    make([]*RequestMetrics, hydratedInput.Count),
	}

	ar := &activeRun{state: state}

	s.mu.Lock()
	s.activeRuns[runID] = ar
	s.mu.Unlock()

	if hydratedInput.Turbo {
		go s.runTurbo(ar, runID, taskDef, hydratedInput, historyDir)
	} else {
		go s.runStandard(ar, runID, taskDef, hydratedInput, historyDir)
	}

	return runID, nil
}

// runStandard 在 goroutine 中执行标准运行。
func (s *serverImpl) runStandard(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, historyDir string) {
	rnr, err := runner.NewRunner(taskDef.ID, input)
	if err != nil {
		s.failRun(ar, runID, taskDef, historyDir, err)
		return
	}

	ar.mu.Lock()
	ar.rnr = rnr
	ar.mu.Unlock()

	// 启动 500ms 进度快照 goroutine，定期向订阅者推送 EventProgressTick。
	stopTick := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ar.mu.RLock()
				snap := ar.snapshotState()
				ar.mu.RUnlock()
				s.bus.Publish(Event{RunID: runID, Kind: EventProgressTick, Payload: snap})
			case <-stopTick:
				return
			}
		}
	}()

	reportData, err := rnr.RunWithCallback(func(metrics *client.ResponseMetrics, idx int, cbErr error) {
		rm := mapRequestMetrics(metrics, idx, cbErr)

		ar.mu.Lock()
		if idx < len(ar.state.Requests) {
			ar.state.Requests[idx] = rm
		}
		ar.state.DoneReqs++
		if rm.Success {
			ar.state.SuccessReqs++
			ar.tpsSum += rm.TPS
			ar.ttftSum += rm.TTFT
			ar.cacheSum += rm.CacheHitRate
		} else {
			ar.state.FailedReqs++
		}
		successCount := ar.state.SuccessReqs
		done := ar.state.DoneReqs
		// 更新实时均值
		if successCount > 0 {
			ar.state.AvgTPS = ar.tpsSum / float64(successCount)
			ar.state.AvgTTFT = ar.ttftSum / time.Duration(successCount)
			ar.state.CacheHitRate = ar.cacheSum / float64(successCount)
		}
		if done > 0 {
			ar.state.SuccessRate = float64(successCount) / float64(done) * 100
		}
		snap := ar.snapshotState()
		ar.mu.Unlock()

		s.bus.Publish(Event{RunID: runID, Kind: EventRequestDone, Payload: snap})
	})

	close(stopTick)

	if err != nil {
		s.failRun(ar, runID, taskDef, historyDir, err)
		return
	}

	s.completeStandardRun(ar, runID, taskDef, historyDir, reportData)
}

// runTurbo 在 goroutine 中执行 Turbo 运行。
func (s *serverImpl) runTurbo(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, historyDir string) {
	engine := turbo.New(turbo.DefaultRunnerFactory(taskDef.ID))

	ar.mu.Lock()
	ar.turboEngine = engine
	ar.mu.Unlock()

	turboResult, err := engine.Run(input)
	if err != nil {
		s.failRun(ar, runID, taskDef, historyDir, err)
		return
	}

	s.completeTurboRun(ar, runID, taskDef, historyDir, turboResult)
}

// completeStandardRun 处理标准运行成功完成的后续工作。
func (s *serverImpl) completeStandardRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, historyDir string, data *types.ReportData) {
	finishedAt := time.Now()

	ar.mu.Lock()
	ar.state.Status = RunStatusCompleted
	ar.state.FinishedAt = &finishedAt
	ar.state.StandardResult = data
	if data != nil {
		ar.state.AvgTPS = data.AvgTPS
		ar.state.AvgTTFT = data.AvgTTFT
		ar.state.SuccessRate = data.SuccessRate
		ar.state.CacheHitRate = data.AvgCacheHitRate
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	s.bus.Publish(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	s.bus.CloseRun(runID)

	// 将完整运行状态持久化到磁盘，供历史详情页回放
	persistRunState(historyDir, snap)

	summary := types.TaskRunSummary{
		RunID:       string(runID),
		TaskID:      taskDef.ID,
		Mode:        "standard",
		Status:      string(RunStatusCompleted),
		Protocol:    taskDef.Input.NormalizedProtocol(),
		Model:       taskDef.Input.Model,
		StartedAt:   snap.StartedAt,
		FinishedAt:  finishedAt,
		SuccessRate: snap.SuccessRate,
		AvgTTFT:     snap.AvgTTFT,
		AvgTPS:      snap.AvgTPS,
		CacheHitRate: snap.CacheHitRate,
	}

	s.persistRunResult(taskDef.ID, historyDir, summary)
}

// completeTurboRun 处理 Turbo 运行成功完成的后续工作。
func (s *serverImpl) completeTurboRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, historyDir string, result *types.TurboResult) {
	finishedAt := time.Now()

	ar.mu.Lock()
	ar.state.Status = RunStatusCompleted
	ar.state.FinishedAt = &finishedAt
	ar.state.TurboResult = result
	if result != nil {
		ar.state.Levels = result.Levels
		ar.state.CurrentLevel = result.MaxStableConcurrency
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	s.bus.Publish(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	s.bus.CloseRun(runID)

	// 将完整运行状态持久化到磁盘，供历史详情页回放
	persistRunState(historyDir, snap)

	var maxStable int
	var peakTPS float64
	if result != nil {
		maxStable = result.MaxStableConcurrency
		peakTPS = result.PeakTPS
	}

	summary := types.TaskRunSummary{
		RunID:                string(runID),
		TaskID:               taskDef.ID,
		Mode:                 "turbo",
		Status:               string(RunStatusCompleted),
		Protocol:             taskDef.Input.NormalizedProtocol(),
		Model:                taskDef.Input.Model,
		StartedAt:            snap.StartedAt,
		FinishedAt:           finishedAt,
		MaxStableConcurrency: maxStable,
		AvgTPS:               peakTPS,
	}

	s.persistRunResult(taskDef.ID, historyDir, summary)
}

// failRun 处理运行失败的后续工作。
func (s *serverImpl) failRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, historyDir string, runErr error) {
	finishedAt := time.Now()

	ar.mu.Lock()
	ar.state.Status = RunStatusFailed
	ar.state.FinishedAt = &finishedAt
	ar.state.ErrorMsg = runErr.Error()
	snap := ar.snapshotState()
	ar.mu.Unlock()

	s.bus.Publish(Event{RunID: runID, Kind: EventRunFailed, Payload: runErr})
	s.bus.CloseRun(runID)

	// 将完整运行状态持久化到磁盘，供历史详情页回放
	persistRunState(historyDir, snap)

	summary := types.TaskRunSummary{
		RunID:        string(runID),
		TaskID:       taskDef.ID,
		Mode:         ar.state.Mode,
		Status:       string(RunStatusFailed),
		Protocol:     taskDef.Input.NormalizedProtocol(),
		Model:        taskDef.Input.Model,
		StartedAt:    snap.StartedAt,
		FinishedAt:   finishedAt,
		ErrorSummary: runErr.Error(),
	}

	s.persistRunResult(taskDef.ID, historyDir, summary)
}

// persistRunResult 将运行摘要写入历史文件，并更新任务的 LastRunAt/LastRunSummary。
func (s *serverImpl) persistRunResult(taskID, historyDir string, summary types.TaskRunSummary) {
	// 写历史文件
	hs := store.NewHistoryStore(historyPath(historyDir, taskID))
	_ = hs.Append(summary) // 历史记录失败不影响主流程

	// 更新任务的最后运行时间和摘要
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.taskStore.Get(taskID)
	if ok {
		existing.LastRunAt = &summary.FinishedAt
		existing.LastRunSummary = &summary
		s.taskStore.Upsert(existing)
		_ = s.taskStore.Save()
	}
}

// StopRun 请求停止指定运行。
func (s *serverImpl) StopRun(runID RunID) error {
	s.mu.RLock()
	ar, ok := s.activeRuns[runID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("run %q not found or already finished", runID)
	}

	ar.mu.RLock()
	rnr := ar.rnr
	eng := ar.turboEngine
	ar.mu.RUnlock()

	if rnr != nil {
		rnr.Stop()
	}
	if eng != nil {
		eng.Stop()
	}
	return nil
}

// GetRunState 返回指定运行的当前状态快照。
// 先查内存中的 activeRuns；若不存在，再尝试从磁盘加载持久化的快照（历史回放）。
func (s *serverImpl) GetRunState(runID RunID) (*RunState, bool) {
	s.mu.RLock()
	ar, ok := s.activeRuns[runID]
	historyDir := s.historyDir
	s.mu.RUnlock()

	if ok {
		ar.mu.RLock()
		snap := ar.snapshotState()
		ar.mu.RUnlock()
		return snap, true
	}

	// 不在内存中，尝试从磁盘加载持久化的 RunState 快照
	st := store.NewJSONStore[*RunState](runStatePath(historyDir, runID))
	snap, err := st.Load()
	if err != nil || snap == nil {
		return nil, false
	}
	return snap, true
}

// Subscribe 订阅指定运行的事件流。
func (s *serverImpl) Subscribe(runID RunID) (<-chan Event, CancelFunc) {
	return s.bus.Subscribe(runID)
}

// GetHistory 返回任务的历史运行摘要，最新在前。
func (s *serverImpl) GetHistory(taskID string, limit int) ([]types.TaskRunSummary, error) {
	s.mu.RLock()
	historyDir := s.historyDir
	s.mu.RUnlock()

	hs := store.NewHistoryStore(historyPath(historyDir, taskID))
	return hs.Load(limit)
}

// GenerateReport 为已完成的标准运行生成报告文件。
func (s *serverImpl) GenerateReport(runID RunID, format ReportFormat) (string, error) {
	s.mu.RLock()
	ar, ok := s.activeRuns[runID]
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("run %q not found", runID)
	}

	ar.mu.RLock()
	status := ar.state.Status
	mode := ar.state.Mode
	standardResult := ar.state.StandardResult
	ar.mu.RUnlock()

	if status == RunStatusRunning {
		return "", fmt.Errorf("run %q is still in progress", runID)
	}

	if mode == "turbo" {
		return "", fmt.Errorf("report generation for turbo runs is not yet supported")
	}

	if standardResult == nil {
		return "", fmt.Errorf("no result data available for run %q", runID)
	}

	rm := report.NewReportManager()
	paths, err := rm.GenerateReports([]types.ReportData{*standardResult}, []string{string(format)})
	if err != nil {
		return "", fmt.Errorf("generate report: %w", err)
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no report file generated")
	}
	return paths[0], nil
}
