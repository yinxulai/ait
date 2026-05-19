package server

import (
	"fmt"
	"sync"
	"sync/atomic"
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

// callbackLevelRunner 包装 runner.Runner，在每次请求完成时调用回调，
// 使 turbo 运行也能逐请求采集详细指标数据。
type callbackLevelRunner struct {
	r  *runner.Runner
	cb runner.RequestDoneCallback
}

func (c *callbackLevelRunner) Run() (*types.ReportData, error) {
	return c.r.RunWithCallback(c.cb)
}

func (c *callbackLevelRunner) Stop() {
	c.r.Stop()
}

// snapshotState 返回 state 的深度拷贝（调用方须已持有 activeRun.mu 读锁）。
func (ar *activeRun) snapshotState() *RunState {
	s := ar.state
	snap := *s
	// 深拷贝切片
	if len(s.Requests) > 0 {
		snap.Requests = make([]*types.RequestMetrics, len(s.Requests))
		copy(snap.Requests, s.Requests)
	}
	if len(s.Levels) > 0 {
		snap.Levels = make([]types.TurboLevelResult, len(s.Levels))
		copy(snap.Levels, s.Levels)
	}
	return &snap
}

// mapRequestMetrics 将 client.ResponseMetrics 映射到 types.RequestMetrics。
func mapRequestMetrics(m *client.ResponseMetrics, idx int, err error) *types.RequestMetrics {
	rm := &types.RequestMetrics{Index: idx}
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

func requestPointers(requests []types.RequestMetrics) []*types.RequestMetrics {
	if len(requests) == 0 {
		return nil
	}
	pointers := make([]*types.RequestMetrics, 0, len(requests))
	for i := range requests {
		request := requests[i]
		pointers = append(pointers, &request)
	}
	return pointers
}

func buildStoredRunMetadata(taskDef types.TaskDefinition, snap *RunState) store.RunMetadata {
	var finishedAt *time.Time
	if snap.FinishedAt != nil {
		finished := *snap.FinishedAt
		finishedAt = &finished
	}
	return store.RunMetadata{
		RunID:      string(snap.RunID),
		TaskID:     snap.TaskID,
		Mode:       snap.Mode,
		Protocol:   taskDef.Input.NormalizedProtocol(),
		Model:      taskDef.Input.Model,
		Status:     string(snap.Status),
		StartedAt:  snap.StartedAt,
		FinishedAt: finishedAt,
	}
}

func buildStoredRunResult(snap *RunState) store.RunResult {
	result := store.RunResult{
		TotalReqs:      snap.TotalReqs,
		DoneReqs:       snap.DoneReqs,
		SuccessReqs:    snap.SuccessReqs,
		FailedReqs:     snap.FailedReqs,
		SuccessRate:    snap.SuccessRate,
		AvgTTFT:        snap.AvgTTFT,
		AvgTPS:         snap.AvgTPS,
		CacheHitRate:   snap.CacheHitRate,
		ErrorSummary:   snap.ErrorMsg,
		StandardResult: snap.StandardResult,
		TurboResult:    snap.TurboResult,
	}
	if snap.TurboResult != nil {
		result.MaxStableConcurrency = snap.TurboResult.MaxStableConcurrency
	} else if snap.CurrentLevel > 0 {
		result.MaxStableConcurrency = snap.CurrentLevel
	}
	return result
}

func buildRunStateFromStoredRun(run *store.StoredRun, requests []*types.RequestMetrics) *RunState {
	if run == nil {
		return nil
	}

	state := &RunState{
		RunID:     RunID(run.Metadata.RunID),
		TaskID:    run.Metadata.TaskID,
		Status:    RunStatus(run.Metadata.Status),
		Mode:      run.Metadata.Mode,
		StartedAt: run.Metadata.StartedAt,
		Requests:  requests,
	}
	if run.Metadata.FinishedAt != nil {
		finished := *run.Metadata.FinishedAt
		state.FinishedAt = &finished
	}
	if run.Result == nil {
		return state
	}

	state.TotalReqs = run.Result.TotalReqs
	state.DoneReqs = run.Result.DoneReqs
	state.SuccessReqs = run.Result.SuccessReqs
	state.FailedReqs = run.Result.FailedReqs
	state.SuccessRate = run.Result.SuccessRate
	state.AvgTTFT = run.Result.AvgTTFT
	state.AvgTPS = run.Result.AvgTPS
	state.CacheHitRate = run.Result.CacheHitRate
	state.StandardResult = run.Result.StandardResult
	state.TurboResult = run.Result.TurboResult
	state.ErrorMsg = run.Result.ErrorSummary
	state.CurrentLevel = run.Result.MaxStableConcurrency
	if state.DoneReqs == 0 && len(requests) > 0 {
		state.DoneReqs = len(requests)
	}
	if state.TotalReqs == 0 && len(requests) > 0 {
		state.TotalReqs = len(requests)
	}
	if run.Result.TurboResult != nil {
		state.Levels = run.Result.TurboResult.Levels
		state.CurrentLevel = run.Result.TurboResult.MaxStableConcurrency
	}
	return state
}

func buildRunningRunSummary(taskDef types.TaskDefinition, snap *RunState) types.TaskRunSummary {
	summary := types.TaskRunSummary{
		RunID:        string(snap.RunID),
		TaskID:       taskDef.ID,
		Mode:         snap.Mode,
		Status:       string(snap.Status),
		Protocol:     taskDef.Input.NormalizedProtocol(),
		Model:        taskDef.Input.Model,
		StartedAt:    snap.StartedAt,
		SuccessRate:  snap.SuccessRate,
		AvgTTFT:      snap.AvgTTFT,
		AvgTPS:       snap.AvgTPS,
		CacheHitRate: snap.CacheHitRate,
	}
	if snap.FinishedAt != nil {
		summary.FinishedAt = *snap.FinishedAt
	}
	if snap.ErrorMsg != "" {
		summary.ErrorSummary = snap.ErrorMsg
	}
	return summary
}

// StartRun 启动一次新的运行，立即返回 RunID。
func (s *serverImpl) StartRun(taskID string) (RunID, error) {
	s.mu.RLock()
	taskDef, ok := s.taskStore.Get(taskID)
	runStore := s.runStore
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
		RunID:     runID,
		TaskID:    taskID,
		Status:    RunStatusRunning,
		Mode:      mode,
		StartedAt: now,
	}
	if hydratedInput.Turbo {
		// turbo 模式：跨多个并发级别探测，请求总数不固定，动态追加
		state.TotalReqs = 0
	} else {
		// standard 模式：请求数固定，动态追加（按完成顺序）
		state.TotalReqs = hydratedInput.Count
	}

	ar := &activeRun{state: state}

	s.mu.Lock()
	s.activeRuns[runID] = ar
	s.mu.Unlock()

	if hydratedInput.Turbo {
		go s.runTurbo(ar, runID, taskDef, hydratedInput, runStore)
	} else {
		go s.runStandard(ar, runID, taskDef, hydratedInput, runStore)
	}

	return runID, nil
}

// runStandard 在 goroutine 中执行标准运行。
func (s *serverImpl) runStandard(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	rnr, err := runner.NewRunner(taskDef.ID, input)
	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
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
		_ = runStore.AppendRequest(taskDef.ID, string(runID), *rm)

		ar.mu.Lock()
		ar.state.Requests = append(ar.state.Requests, rm)
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
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}

	s.completeStandardRun(ar, runID, taskDef, runStore, reportData)
}

// runTurbo 在 goroutine 中执行 Turbo 运行。
func (s *serverImpl) runTurbo(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	// 全局请求计数器（原子递增），确保跨多个并发级别的请求索引唯一
	var globalIdx int64

	factory := func(levelInput types.Input) (turbo.LevelRunner, error) {
		r, err := runner.NewRunner(taskDef.ID, levelInput)
		if err != nil {
			return nil, err
		}
		return &callbackLevelRunner{
			r: r,
			cb: func(metrics *client.ResponseMetrics, _ int, cbErr error) {
				gIdx := int(atomic.AddInt64(&globalIdx, 1)) - 1
				rm := mapRequestMetrics(metrics, gIdx, cbErr)
				_ = runStore.AppendRequest(taskDef.ID, string(runID), *rm)

				ar.mu.Lock()
				ar.state.Requests = append(ar.state.Requests, rm)
				ar.state.TotalReqs++
				ar.state.DoneReqs++
				if rm.Success {
					ar.state.SuccessReqs++
					ar.tpsSum += rm.TPS
					ar.ttftSum += rm.TTFT
					ar.cacheSum += rm.CacheHitRate
				} else {
					ar.state.FailedReqs++
				}
				if ar.state.SuccessReqs > 0 {
					ar.state.AvgTPS = ar.tpsSum / float64(ar.state.SuccessReqs)
					ar.state.AvgTTFT = ar.ttftSum / time.Duration(ar.state.SuccessReqs)
					ar.state.CacheHitRate = ar.cacheSum / float64(ar.state.SuccessReqs)
				}
				if ar.state.DoneReqs > 0 {
					ar.state.SuccessRate = float64(ar.state.SuccessReqs) / float64(ar.state.DoneReqs) * 100
				}
				snap := ar.snapshotState()
				ar.mu.Unlock()

				s.bus.Publish(Event{RunID: runID, Kind: EventRequestDone, Payload: snap})
			},
		}, nil
	}

	engine := turbo.New(factory)

	ar.mu.Lock()
	ar.turboEngine = engine
	ar.mu.Unlock()

	turboResult, err := engine.Run(input)
	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}

	s.completeTurboRun(ar, runID, taskDef, runStore, turboResult)
}

// completeStandardRun 处理标准运行成功完成的后续工作。
func (s *serverImpl) completeStandardRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore, data *types.ReportData) {
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
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

// completeTurboRun 处理 Turbo 运行成功完成的后续工作。
func (s *serverImpl) completeTurboRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore, result *types.TurboResult) {
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
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

// failRun 处理运行失败的后续工作。
func (s *serverImpl) failRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore, runErr error) {
	finishedAt := time.Now()

	ar.mu.Lock()
	ar.state.Status = RunStatusFailed
	ar.state.FinishedAt = &finishedAt
	ar.state.ErrorMsg = runErr.Error()
	snap := ar.snapshotState()
	ar.mu.Unlock()

	s.bus.Publish(Event{RunID: runID, Kind: EventRunFailed, Payload: snap})
	s.bus.CloseRun(runID)
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

func (s *serverImpl) persistFinalRun(runStore *store.RunStore, taskDef types.TaskDefinition, snap *RunState) error {
	return runStore.SaveFinal(buildStoredRunMetadata(taskDef, snap), buildStoredRunResult(snap))
}

func (s *serverImpl) persistRunResult(summary types.TaskRunSummary) error {
	return s.runStore.SaveSummary(summary)
}

func (s *serverImpl) removeActiveRun(runID RunID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeRuns, runID)
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
// 先查内存中的 activeRuns；若不存在，再尝试从磁盘加载最终运行结果（历史回放）。
func (s *serverImpl) GetRunState(runID RunID) (*RunState, bool) {
	s.mu.RLock()
	ar, ok := s.activeRuns[runID]
	runStore := s.runStore
	s.mu.RUnlock()

	if ok {
		ar.mu.RLock()
		snap := ar.snapshotState()
		ar.mu.RUnlock()
		return snap, true
	}

	run, err := runStore.LoadByRunID(string(runID))
	if err != nil || run == nil {
		return nil, false
	}
	requests, err := runStore.LoadRequests(run.Metadata.TaskID, string(runID))
	if err != nil {
		return nil, false
	}
	return buildRunStateFromStoredRun(run, requestPointers(requests)), true
}

// Subscribe 订阅指定运行的事件流。
func (s *serverImpl) Subscribe(runID RunID) (<-chan Event, CancelFunc) {
	return s.bus.Subscribe(runID)
}

// GetHistory 返回任务的历史运行摘要，最新在前。
func (s *serverImpl) GetHistory(taskID string, limit int) ([]types.TaskRunSummary, error) {
	runs, err := s.runStore.ListByTask(taskID, limit)
	if err != nil {
		return nil, err
	}
	history := make([]types.TaskRunSummary, 0, len(runs))
	for _, run := range runs {
		history = append(history, run.Summary())
	}
	return history, nil
}

// GenerateReport 为已完成的标准运行生成报告文件。
// 先查内存中的 activeRuns，若不存在则从最终结果文件加载（支持跨 session 历史运行）。
func (s *serverImpl) GenerateReport(runID RunID, format ReportFormat) (string, error) {
	s.mu.RLock()
	ar, ok := s.activeRuns[runID]
	runStore := s.runStore
	s.mu.RUnlock()

	var status RunStatus
	var mode string
	var standardResult *types.ReportData

	if ok {
		ar.mu.RLock()
		status = ar.state.Status
		mode = ar.state.Mode
		standardResult = ar.state.StandardResult
		ar.mu.RUnlock()
	} else {
		run, err := runStore.LoadByRunID(string(runID))
		if err != nil || run == nil {
			return "", fmt.Errorf("run %q not found", runID)
		}
		status = RunStatus(run.Metadata.Status)
		mode = run.Metadata.Mode
		if run.Result != nil {
			standardResult = run.Result.StandardResult
		}
	}

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
