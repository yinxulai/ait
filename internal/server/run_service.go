package server

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/server/modes/integrity"
	"github.com/yinxulai/ait/internal/server/modes/standard"
	"github.com/yinxulai/ait/internal/server/modes/turbo"
	"github.com/yinxulai/ait/internal/server/report"
	"github.com/yinxulai/ait/internal/server/store"
	"github.com/yinxulai/ait/internal/server/task"
	"github.com/yinxulai/ait/internal/server/types"
)

// activeRun 持有一次正在执行的运行的全部运行时状态。
type activeRun struct {
	mu                sync.RWMutex
	state             *RunState
	rnr               *standard.Runner    // standard 模式使用
	turboEngine       *turbo.Engine       // turbo 模式使用
	integrityExecutor *integrity.Executor // integrity 模式使用
	// 用于计算实时均值
	tpsSum    float64
	ttftSum   time.Duration
	cacheSum  float64
	tokenSum  int64 // 累计成功请求的输出 Token 数，用于计算 TPM
	doneCount int   // 与 state.DoneReqs 保持同步，方便不加锁时计算
}

// callbackLevelRunner 包装 standard.Runner，在每次请求完成时调用回调，
// 使 turbo 运行也能逐请求采集详细指标数据。
type callbackLevelRunner struct {
	r  *standard.Runner
	cb standard.RequestDoneCallback
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
	if len(s.IntegritySuite.Cases) > 0 {
		snap.IntegritySuite = s.IntegritySuite
		snap.IntegritySuite.Cases = make([]types.IntegrityCase, len(s.IntegritySuite.Cases))
		copy(snap.IntegritySuite.Cases, s.IntegritySuite.Cases)
	}
	if len(s.IntegrityCases) > 0 {
		snap.IntegrityCases = make([]types.IntegrityCaseResult, len(s.IntegrityCases))
		copy(snap.IntegrityCases, s.IntegrityCases)
	}
	if len(s.AssertionResults) > 0 {
		snap.AssertionResults = make([]types.AssertionResult, len(s.AssertionResults))
		copy(snap.AssertionResults, s.AssertionResults)
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
	if m.CachedInputTokens > 0 {
		rm.CacheHitRate = 1
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
		ErrorSummary:    snap.ErrorMsg,
		StandardResult:  snap.StandardResult,
		TurboResult:     snap.TurboResult,
		IntegrityResult: snap.IntegrityResult,
	}
	if snap.StandardResult == nil && snap.TurboResult == nil && snap.IntegrityResult == nil && snap.TotalReqs > 0 {
		result.TotalReqs = snap.TotalReqs
	}
	if snap.TurboResult == nil && snap.CurrentLevel > 0 {
		result.MaxStableConcurrency = snap.CurrentLevel
	}
	return result
}

func buildRunStateFromStoredRun(run *store.StoredRun, requests []types.RequestMetrics) *RunState {
	if run == nil {
		return nil
	}

	summary := run.Summary(requests)
	state := &RunState{
		RunID:        RunID(run.Metadata.RunID),
		TaskID:       run.Metadata.TaskID,
		Status:       RunStatus(run.Metadata.Status),
		Mode:         run.Metadata.Mode,
		StartedAt:    run.Metadata.StartedAt,
		Requests:     requestPointers(requests),
		AvgTTFT:      summary.AvgTTFT,
		AvgTPS:       summary.AvgTPS,
		SuccessRate:  summary.SuccessRate,
		CacheHitRate: summary.CacheHitRate,
		ErrorMsg:     summary.ErrorSummary,
		CurrentLevel: summary.MaxStableConcurrency,
	}
	if run.Metadata.FinishedAt != nil {
		finished := *run.Metadata.FinishedAt
		state.FinishedAt = &finished
	}
	state.DoneReqs = len(requests)
	for _, request := range requests {
		if request.Success {
			state.SuccessReqs++
		}
	}
	state.FailedReqs = state.DoneReqs - state.SuccessReqs
	state.TotalReqs = run.TotalReqs(requests)

	// 从存储数据重建 RPM/TPM
	end := time.Now()
	if run.Metadata.FinishedAt != nil {
		end = *run.Metadata.FinishedAt
	}
	if !run.Metadata.StartedAt.IsZero() {
		if elapsed := end.Sub(run.Metadata.StartedAt).Minutes(); elapsed > 0 {
			var tokenSum int64
			for _, r := range requests {
				if r.Success {
					tokenSum += int64(r.CompletionTokens)
				}
			}
			state.RPM = float64(state.DoneReqs) / elapsed
			state.TPM = float64(tokenSum) / elapsed
		}
	}
	if run.Result == nil {
		return state
	}

	state.StandardResult = run.Result.StandardResult
	state.TurboResult = run.Result.TurboResult
	state.IntegrityResult = run.Result.IntegrityResult
	if run.Result.TurboResult != nil {
		state.TurboConfig = run.Result.TurboResult.Config
		state.Levels = run.Result.TurboResult.Levels
		state.CurrentLevel = run.Result.TurboResult.MaxStableConcurrency
	}
	if run.Result.IntegrityResult != nil {
		state.IntegrityCases = run.Result.IntegrityResult.Cases
		state.AssertionResults = run.Result.IntegrityResult.Assertions
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
	taskDef, err := s.taskStore.Get(taskID)
	if err != nil {
		if errors.Is(err, store.ErrTaskNotFound) {
			return "", fmt.Errorf("task %q not found: %w", taskID, err)
		}
		return "", fmt.Errorf("get task %q: %w", taskID, err)
	}
	runStore := s.runStore

	// 解析 PromptSource（将 PromptText/PromptFile 转换为可调用的 PromptSource）
	hydratedInput, err := task.HydrateInput(taskDef.Input)
	if err != nil {
		return "", fmt.Errorf("hydrate input: %w", err)
	}

	// 若任务未单独配置代理，使用全局配置中的代理地址
	if hydratedInput.ProxyURL == "" {
		if cfg, err := config.Load(); err == nil {
			hydratedInput.ProxyURL = cfg.ProxyURL
		}
	}

	runID := RunID(fmt.Sprintf("run_%d", time.Now().UnixNano()))
	now := time.Now()

	mode := hydratedInput.RunMode()

	state := &RunState{
		RunID:     runID,
		TaskID:    taskID,
		Status:    RunStatusRunning,
		Mode:      mode,
		StartedAt: now,
	}
	switch mode {
	case "turbo":
		// turbo 模式：跨多个并发级别探测，请求总数不固定，动态追加
		state.TotalReqs = 0
		// 规范化并存储 TurboConfig，供 TUI 在运行开始时即可显示任务参数
		state.TurboConfig = turbo.NormalizeConfig(hydratedInput.TurboConfig, hydratedInput.Count)
	case "integrity":
		state.TotalReqs = 0
	default:
		// standard 模式：请求数固定，动态追加（按完成顺序）
		state.TotalReqs = hydratedInput.Count
	}

	ar := &activeRun{state: state}

	s.mu.Lock()
	s.activeRuns[runID] = ar
	s.mu.Unlock()

	switch mode {
	case "turbo":
		go s.runTurbo(ar, runID, taskDef, hydratedInput, runStore)
	case "integrity":
		go s.runIntegrity(ar, runID, taskDef, hydratedInput, runStore)
	default:
		go s.runStandard(ar, runID, taskDef, hydratedInput, runStore)
	}

	return runID, nil
}

// runStandard 在 goroutine 中执行标准运行。
func (s *serverImpl) runStandard(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	rnr, err := standard.NewRunner(taskDef.ID, input)
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
				s.bus.publishRunEvent(Event{RunID: runID, Kind: EventProgressTick, Payload: snap})
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

		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRequestDone, Payload: snap})
	})

	close(stopTick)

	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}

	s.completeStandardRun(ar, runID, taskDef, runStore, reportData)
}

// runIntegrity 在 goroutine 中执行接口完整性测试。
func (s *serverImpl) runIntegrity(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	suite, err := integrity.LoadSuite(input)
	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}

	ar.mu.Lock()
	ar.state.IntegritySuite = suite
	ar.state.TotalReqs = len(suite.Cases)
	ar.mu.Unlock()

	executor := integrity.NewExecutor(taskDef.ID, input, suite)
	ar.mu.Lock()
	ar.integrityExecutor = executor
	ar.mu.Unlock()
	executor.OnCaseStarted = func(c types.IntegrityCase) {
		ar.mu.Lock()
		ar.state.CurrentCaseID = c.ID
		snap := ar.snapshotState()
		ar.mu.Unlock()
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventIntegrityCaseStarted, Payload: snap})
	}
	executor.OnRequestDone = func(c types.IntegrityCase, metrics *client.ResponseMetrics, idx int, cbErr error, assertions []types.AssertionResult) {
		rm := mapRequestMetrics(metrics, idx, cbErr)
		_ = runStore.AppendRequest(taskDef.ID, string(runID), *rm)

		ar.mu.Lock()
		ar.state.Requests = append(ar.state.Requests, rm)
		ar.state.AssertionResults = append(ar.state.AssertionResults, assertions...)
		ar.state.DoneReqs++
		if rm.Success {
			ar.state.SuccessReqs++
			ar.tpsSum += rm.TPS
			ar.ttftSum += rm.TTFT
			ar.cacheSum += rm.CacheHitRate
			ar.tokenSum += int64(rm.CompletionTokens)
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

		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventAssertionResult, Payload: assertions})
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRequestDone, Payload: snap})
	}
	executor.OnCaseDone = func(result types.IntegrityCaseResult) {
		ar.mu.Lock()
		ar.state.IntegrityCases = append(ar.state.IntegrityCases, result)
		snap := ar.snapshotState()
		ar.mu.Unlock()
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventIntegrityCaseDone, Payload: snap})
	}

	result, err := executor.Run()
	if result != nil {
		result.Protocol = input.NormalizedProtocol()
		result.Model = input.Model
		result.EndpointURL = input.ResolvedEndpointURL()
		result.Timestamp = time.Now().Format(time.RFC3339)
	}
	if err != nil && result == nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}
	s.completeIntegrityRun(ar, runID, taskDef, runStore, result)
}

// runTurbo 在 goroutine 中执行 Turbo 运行。
func (s *serverImpl) runTurbo(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	// 全局请求计数器（原子递增），确保跨多个并发级别的请求索引唯一
	var globalIdx int64

	factory := func(levelInput types.Input) (turbo.LevelRunner, error) {
		// 每级别开始时更新 CurrentLevel，TUI 实时反映当前探测的并发度
		ar.mu.Lock()
		ar.state.CurrentLevel = levelInput.Concurrency
		ar.mu.Unlock()

		r, err := standard.NewRunner(taskDef.ID, levelInput)
		if err != nil {
			return nil, err
		}
		return &callbackLevelRunner{
			r: r,
			cb: func(metrics *client.ResponseMetrics, _ int, cbErr error) {
				gIdx := int(atomic.AddInt64(&globalIdx, 1)) - 1
				rm := mapRequestMetrics(metrics, gIdx, cbErr)
				rm.Level = levelInput.Concurrency
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
					ar.tokenSum += int64(rm.CompletionTokens)
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
				// 更新 RPM/TPM（基于运行时长）
				if elapsed := time.Since(ar.state.StartedAt).Minutes(); elapsed > 0 {
					ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
					ar.state.TPM = float64(ar.tokenSum) / elapsed
				}
				snap := ar.snapshotState()
				ar.mu.Unlock()

				s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRequestDone, Payload: snap})
			},
		}, nil
	}

	engine := turbo.New(factory)
	engine.SetOnLevelDone(func(level types.TurboLevelResult) {
		ar.mu.Lock()
		ar.state.Levels = append(ar.state.Levels, level)
		snap := ar.snapshotState()
		ar.mu.Unlock()
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventLevelDone, Payload: snap})
	})

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
	// 使用完整运行时长计算最终稳定的 RPM/TPM
	if elapsed := finishedAt.Sub(ar.state.StartedAt).Minutes(); elapsed > 0 {
		ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
		ar.state.TPM = float64(ar.tokenSum) / elapsed
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	s.bus.closeRunEvents(runID)
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
	// 使用完整运行时长计算最终稳定的 RPM/TPM
	if elapsed := finishedAt.Sub(ar.state.StartedAt).Minutes(); elapsed > 0 {
		ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
		ar.state.TPM = float64(ar.tokenSum) / elapsed
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	s.bus.closeRunEvents(runID)
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

// completeIntegrityRun 处理接口完整性测试成功完成的后续工作。
func (s *serverImpl) completeIntegrityRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore, result *types.IntegrityResult) {
	finishedAt := time.Now()

	ar.mu.Lock()
	ar.state.Status = RunStatusCompleted
	if result != nil && result.Status == "failed" {
		ar.state.Status = RunStatusFailed
	}
	ar.state.FinishedAt = &finishedAt
	ar.state.IntegrityResult = result
	if result != nil {
		ar.state.IntegrityCases = result.Cases
		ar.state.AssertionResults = result.Assertions
	}
	if elapsed := finishedAt.Sub(ar.state.StartedAt).Minutes(); elapsed > 0 {
		ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
		ar.state.TPM = float64(ar.tokenSum) / elapsed
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	if snap.Status == RunStatusFailed {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunFailed, Payload: snap})
	} else {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	}
	s.bus.closeRunEvents(runID)
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

	s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunFailed, Payload: snap})
	s.bus.closeRunEvents(runID)
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

func (s *serverImpl) persistFinalRun(runStore *store.RunStore, taskDef types.TaskDefinition, snap *RunState) error {
	return runStore.SaveFinalRun(buildStoredRunMetadata(taskDef, snap), buildStoredRunResult(snap))
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
	integrityExecutor := ar.integrityExecutor
	ar.mu.RUnlock()

	if rnr != nil {
		rnr.Stop()
	}
	if eng != nil {
		eng.Stop()
	}
	if integrityExecutor != nil {
		integrityExecutor.Stop()
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
	return buildRunStateFromStoredRun(run, requests), true
}

// SubscribeRunEvents 订阅指定运行的事件流。
func (s *serverImpl) SubscribeRunEvents(runID RunID) (<-chan Event, CancelFunc) {
	return s.bus.subscribeRunEvents(runID)
}

// ListTaskRunHistory 返回任务的历史运行摘要，最新在前。
func (s *serverImpl) ListTaskRunHistory(taskID string, limit int) ([]types.TaskRunSummary, error) {
	return s.runStore.ListSummariesByTask(taskID, limit)
}

// GenerateRunReport 为已完成的标准运行生成报告文件。
// 先查内存中的 activeRuns，若不存在则从最终结果文件加载（支持跨 session 历史运行）。
func (s *serverImpl) GenerateRunReport(runID RunID, format ReportFormat) (string, error) {
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
