package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/server/logger"
	"github.com/yinxulai/ait/internal/server/modes"
	"github.com/yinxulai/ait/internal/server/modes/integrity"
	"github.com/yinxulai/ait/internal/server/modes/standard"
	"github.com/yinxulai/ait/internal/server/modes/turbo"
	"github.com/yinxulai/ait/internal/server/report"
	"github.com/yinxulai/ait/internal/server/store"
	"github.com/yinxulai/ait/internal/server/task"
	"github.com/yinxulai/ait/internal/server/types"
	"github.com/yinxulai/ait/internal/server/upload"
)

// activeRun 持有一次正在执行的运行的全部运行时状态。
type activeRun struct {
	mu     sync.RWMutex
	state  *RunState
	ctx    context.Context
	cancel context.CancelFunc
	runner modes.Runner // 统一的模式执行器接口
	// 用于计算实时均值
	tpsSum    float64
	ttftSum   time.Duration
	cacheSum  float64
	tokenSum  int64 // 累计成功请求的输出 Token 数，用于计算 TPM
	doneCount int   // 与 state.DoneReqs 保持同步，方便不加锁时计算
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
	// 深拷贝模式状态
	if ar.runner != nil {
		if provider, ok := ar.runner.(modes.StateProvider); ok {
			// 通过 runner 获取最新快照
			snap.ModeState = map[string]any{"state": provider.GetState()}
			return &snap
		}
	}
	// 如果没有 StateProvider，手动深拷贝 ModeState
	if len(s.ModeState) > 0 {
		snap.ModeState = make(map[string]any, len(s.ModeState))
		for k, v := range s.ModeState {
			// 对已知的 slice 类型进行深拷贝
			switch val := v.(type) {
			case []types.TurboLevelResult:
				copied := make([]types.TurboLevelResult, len(val))
				copy(copied, val)
				snap.ModeState[k] = copied
			case []types.IntegrityCaseResult:
				copied := make([]types.IntegrityCaseResult, len(val))
				copy(copied, val)
				snap.ModeState[k] = copied
			case []types.AssertionResult:
				copied := make([]types.AssertionResult, len(val))
				copy(copied, val)
				snap.ModeState[k] = copied
			default:
				// 其他类型直接拷贝（基本类型或指针）
				snap.ModeState[k] = v
			}
		}
	}
	// 深拷贝请求状态映射
	if len(s.RequestStates) > 0 {
		snap.RequestStates = make(map[int]RequestState, len(s.RequestStates))
		for k, v := range s.RequestStates {
			snap.RequestStates[k] = v
		}
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

func loggerForInput(input types.Input) *logger.Logger {
	if !input.Log {
		return nil
	}
	return logger.New(input.Log)
}

func uploadRequest(taskID string, metrics *client.ResponseMetrics, input types.Input) {
	if metrics == nil || metrics.ErrorMessage != "" {
		return
	}
	upload.New().UploadReport(taskID, metrics, input)
}

func (s *serverImpl) startProgressTicker(ar *activeRun, runID RunID) chan struct{} {
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
	return stopTick
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
		ErrorSummary: snap.ErrorMsg,
		ModeResult:   snap.ModeResult,
	}
	if snap.ModeResult == nil && snap.TotalReqs > 0 {
		result.TotalReqs = snap.TotalReqs
	}
	// 对于 turbo 模式，从 ModeState 提取 current_level
	if snap.Mode == "turbo" && snap.ModeState != nil {
		if level, ok := snap.ModeState["current_level"].(int); ok && level > 0 {
			result.MaxStableConcurrency = level
		}
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

	// 恢复模式结果（根据 mode 判断类型）
	state.ModeResult = run.Result.ModeResult
	// 向后兼容：如果是旧格式存储，尝试从特定字段恢复
	if state.ModeResult == nil {
		switch state.Mode {
		case "standard":
			state.ModeResult = run.Result.StandardResult
		case "turbo":
			state.ModeResult = run.Result.TurboResult
		case "integrity":
			state.ModeResult = run.Result.IntegrityResult
		}
	}
	// 恢复模式状态（从结果中提取）
	switch state.Mode {
	case "turbo":
		if turboResult, ok := state.ModeResult.(*types.TurboResult); ok && turboResult != nil {
			state.ModeState = map[string]any{
				"config":        turboResult.Config,
				"levels":        turboResult.Levels,
				"current_level": turboResult.MaxStableConcurrency,
			}
		}
	case "integrity":
		if integrityResult, ok := state.ModeResult.(*types.IntegrityResult); ok && integrityResult != nil {
			state.ModeState = map[string]any{
				"suite":             types.IntegritySuite{ID: integrityResult.SuiteID},
				"cases":             integrityResult.Cases,
				"assertion_results": integrityResult.Assertions,
			}
		}
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

// StartRun 提交一次新的运行，立即返回 RunID。
func (s *serverImpl) StartRun(taskID string) (RunID, error) {
	taskDef, err := s.taskStore.Get(taskID)
	if err != nil {
		if errors.Is(err, store.ErrTaskNotFound) {
			return "", fmt.Errorf("task %q not found: %w", taskID, err)
		}
		return "", fmt.Errorf("get task %q: %w", taskID, err)
	}

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
	// 使用 Server 的生命周期 Context，这样运行可以响应 Server 关闭
	// 如果 Server 没有 ctx（测试场景），使用 Background
	parentCtx := s.ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)

	state := &RunState{
		RunID:     runID,
		TaskID:    taskID,
		Status:    RunStatusQueued,
		Mode:      mode,
		StartedAt: now,
	}
	switch mode {
	case "turbo", "integrity":
		state.TotalReqs = 0
	default:
		state.TotalReqs = hydratedInput.Count
	}

	ar := &activeRun{state: state, ctx: ctx, cancel: cancel}

	s.mu.Lock()
	if s.scheduler == nil {
		s.scheduler = newRunScheduler(1, s.dispatchQueuedRun)
	}
	s.activeRuns[runID] = ar
	s.mu.Unlock()

	s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunQueued, Payload: state})
	if err := s.scheduler.Enqueue(runQueueItem{RunID: runID, TaskID: taskID, TaskDef: taskDef, Input: hydratedInput, Mode: mode}); err != nil {
		cancel()
		s.removeActiveRun(runID)
		return "", err
	}

	return runID, nil
}

func (s *serverImpl) dispatchQueuedRun(item runQueueItem) {
	s.mu.RLock()
	ar, ok := s.activeRuns[item.RunID]
	runStore := s.runStore
	s.mu.RUnlock()
	if !ok {
		return
	}

	ar.mu.Lock()
	if ar.state.Status == RunStatusStopped {
		ar.mu.Unlock()
		return
	}
	ar.state.Status = RunStatusRunning
	ar.state.StartedAt = time.Now()
	ar.mu.Unlock()

	ar.mu.RLock()
	snap := ar.snapshotState()
	ar.mu.RUnlock()
	s.bus.publishRunEvent(Event{RunID: item.RunID, Kind: EventRunStarted, Payload: snap})

	switch item.Mode {
	case "turbo":
		s.runTurbo(ar, item.RunID, item.TaskDef, item.Input, runStore)
	case "integrity":
		s.runIntegrity(ar, item.RunID, item.TaskDef, item.Input, runStore)
	default:
		s.runStandard(ar, item.RunID, item.TaskDef, item.Input, runStore)
	}
}

// runStandard 在 goroutine 中执行标准运行。
func (s *serverImpl) runStandard(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	ctx := ar.ctx
	if ctx == nil {
		// 备用：使用 Server 的生命周期 Context
		ctx = s.ctx
		if ctx == nil {
			ctx = context.Background()
		}
	}
	loggerInstance := loggerForInput(input)
	modelClient, err := client.NewClient(input, loggerInstance)
	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}
	aggregator := newRunAggregator(s, ar, runID, taskDef, runStore)
	jobs := make([]RequestJob, 0, input.Count)
	for i := 0; i < input.Count; i++ {
		jobs = append(jobs, RequestJob{RunID: runID, Index: i, Input: input})
	}

	stopTick := s.startProgressTicker(ar, runID)
	results := make([]*client.ResponseMetrics, input.Count)
	start := time.Now()
	launched := RunRequestBatch(ctx, jobs, input.Concurrency, NewRequestExecutor(modelClient), RequestQueueHooks{
		OnQueued:  aggregator.MarkQueued,
		OnStarted: aggregator.MarkStarted,
		OnSkipped: aggregator.MarkSkipped,
		OnDone: func(result RequestResult) {
			if result.Metrics != nil {
				results[result.Job.Index] = result.Metrics
			}
			rm := aggregator.Complete(result)
			if rm.Success {
				uploadRequest(taskDef.ID, result.Metrics, input)
			}
		},
	})
	close(stopTick)

	reportData := standard.CalculateResult(input, results, time.Since(start), launched)
	s.completeStandardRun(ar, runID, taskDef, runStore, reportData)
}

// runIntegrity 在 goroutine 中执行接口完整性测试。
func (s *serverImpl) runIntegrity(ar *activeRun, runID RunID, taskDef types.TaskDefinition, input types.Input, runStore *store.RunStore) {
	suite, err := integrity.LoadSuiteWithManager(input, s.rulesManager)
	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}

	ctx := ar.ctx
	if ctx == nil {
		// 备用：使用 Server 的生命周期 Context
		ctx = s.ctx
		if ctx == nil {
			ctx = context.Background()
		}
	}
	aggregator := newRunAggregator(s, ar, runID, taskDef, runStore)
	caseIndex := 0

	executor := integrity.NewExecutor(taskDef.ID, input, suite)
	executor.RunnerFactory = func(caseInput types.Input, c types.IntegrityCase) (integrity.CaseRunner, error) {
		modelClient, err := client.NewClient(caseInput, loggerForInput(caseInput))
		if err != nil {
			return nil, err
		}
		idx := caseIndex
		caseIndex++
		return newQueuedCaseRunner(ctx, runID, caseInput, modelClient, aggregator, idx, c.ID), nil
	}
	ar.mu.Lock()
	ar.runner = executor
	ar.state.TotalReqs = len(suite.Cases)
	// 初始化模式状态
	ar.state.ModeState = map[string]any{
		"suite":             suite,
		"cases":             []types.IntegrityCaseResult{},
		"current_case_id":   "",
		"assertion_results": []types.AssertionResult{},
	}
	ar.mu.Unlock()
	executor.OnCaseStarted = func(c types.IntegrityCase) {
		ar.mu.Lock()
		if ar.state.ModeState == nil {
			ar.state.ModeState = make(map[string]any)
		}
		ar.state.ModeState["current_case_id"] = c.ID
		snap := ar.snapshotState()
		ar.mu.Unlock()
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventIntegrityCaseStarted, Payload: snap})
	}
	executor.OnRequestDone = func(c types.IntegrityCase, metrics *client.ResponseMetrics, idx int, cbErr error, assertions []types.AssertionResult) {
		aggregator.Complete(RequestResult{
			Job:     RequestJob{RunID: runID, Index: idx, Input: input, CaseID: c.ID},
			Metrics: metrics,
			Err:     cbErr,
		})

		ar.mu.Lock()
		if ar.state.ModeState != nil {
			if existing, ok := ar.state.ModeState["assertion_results"].([]types.AssertionResult); ok {
				ar.state.ModeState["assertion_results"] = append(existing, assertions...)
			}
		}
		ar.mu.Unlock()

		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventAssertionResult, Payload: assertions})
	}
	executor.OnCaseDone = func(result types.IntegrityCaseResult) {
		ar.mu.Lock()
		if ar.state.ModeState != nil {
			if existing, ok := ar.state.ModeState["cases"].([]types.IntegrityCaseResult); ok {
				ar.state.ModeState["cases"] = append(existing, result)
			}
		}
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
	ctx := ar.ctx
	if ctx == nil {
		// 备用：使用 Server 的生命周期 Context
		ctx = s.ctx
		if ctx == nil {
			ctx = context.Background()
		}
	}
	loggerInstance := loggerForInput(input)
	modelClient, err := client.NewClient(input, loggerInstance)
	if err != nil {
		s.failRun(ar, runID, taskDef, runStore, err)
		return
	}
	aggregator := newRunAggregator(s, ar, runID, taskDef, runStore)

	factory := func(levelInput types.Input) (turbo.LevelRunner, error) {
		return newQueuedLevelRunner(ctx, runID, levelInput, modelClient, aggregator, levelInput.Concurrency), nil
	}

	engine := turbo.New(factory)
	engine.SetOnLevelDone(func(level types.TurboLevelResult) {
		ar.mu.Lock()
		if ar.state.ModeState == nil {
			ar.state.ModeState = make(map[string]any)
		}
		if existing, ok := ar.state.ModeState["levels"].([]types.TurboLevelResult); ok {
			ar.state.ModeState["levels"] = append(existing, level)
		} else {
			ar.state.ModeState["levels"] = []types.TurboLevelResult{level}
		}
		ar.state.ModeState["current_level"] = level.Concurrency
		snap := ar.snapshotState()
		ar.mu.Unlock()
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventLevelDone, Payload: snap})
	})

	ar.mu.Lock()
	ar.runner = engine
	// 初始化 Turbo 模式状态
	ar.state.ModeState = map[string]any{
		"config":        turbo.NormalizeConfig(input.TurboConfig, input.Count),
		"levels":        []types.TurboLevelResult{},
		"current_level": 0,
	}
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
	if ar.state.Status != RunStatusStopped {
		ar.state.Status = RunStatusCompleted
	}
	ar.state.FinishedAt = &finishedAt
	ar.state.ModeResult = data
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

	if snap.Status == RunStatusStopped {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunStopped, Payload: snap})
	} else {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	}
	s.bus.closeRunEvents(runID)
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

// completeTurboRun 处理 Turbo 运行成功完成的后续工作。
func (s *serverImpl) completeTurboRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore, result *types.TurboResult) {
	finishedAt := time.Now()

	ar.mu.Lock()
	if ar.state.Status != RunStatusStopped {
		ar.state.Status = RunStatusCompleted
	}
	ar.state.FinishedAt = &finishedAt
	ar.state.ModeResult = result
	if result != nil {
		// 更新模式状态为最终结果
		if ar.state.ModeState == nil {
			ar.state.ModeState = make(map[string]any)
		}
		ar.state.ModeState["levels"] = result.Levels
		ar.state.ModeState["current_level"] = result.MaxStableConcurrency
	}
	// 使用完整运行时长计算最终稳定的 RPM/TPM
	if elapsed := finishedAt.Sub(ar.state.StartedAt).Minutes(); elapsed > 0 {
		ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
		ar.state.TPM = float64(ar.tokenSum) / elapsed
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	if snap.Status == RunStatusStopped {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunStopped, Payload: snap})
	} else {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunComplete, Payload: snap})
	}
	s.bus.closeRunEvents(runID)
	if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
		s.removeActiveRun(runID)
	}
}

// completeIntegrityRun 处理接口完整性测试成功完成的后续工作。
func (s *serverImpl) completeIntegrityRun(ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore, result *types.IntegrityResult) {
	finishedAt := time.Now()

	ar.mu.Lock()
	if ar.state.Status != RunStatusStopped {
		ar.state.Status = RunStatusCompleted
		if result != nil && result.Status == "failed" {
			ar.state.Status = RunStatusFailed
		}
	}
	ar.state.FinishedAt = &finishedAt
	ar.state.ModeResult = result
	if result != nil {
		// 更新模式状态为最终结果
		if ar.state.ModeState == nil {
			ar.state.ModeState = make(map[string]any)
		}
		ar.state.ModeState["cases"] = result.Cases
		ar.state.ModeState["assertion_results"] = result.Assertions
	}
	if elapsed := finishedAt.Sub(ar.state.StartedAt).Minutes(); elapsed > 0 {
		ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
		ar.state.TPM = float64(ar.tokenSum) / elapsed
	}
	snap := ar.snapshotState()
	ar.mu.Unlock()

	if snap.Status == RunStatusStopped {
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunStopped, Payload: snap})
	} else if snap.Status == RunStatusFailed {
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

	ar.mu.Lock()
	if ar.cancel != nil {
		ar.cancel()
	}
	if ar.state.Status == RunStatusQueued {
		now := time.Now()
		ar.state.Status = RunStatusStopped
		ar.state.FinishedAt = &now
		ar.mu.Unlock()
		ar.mu.RLock()
		snap := ar.snapshotState()
		ar.mu.RUnlock()
		s.bus.publishRunEvent(Event{RunID: runID, Kind: EventRunStopped, Payload: snap})
		s.bus.closeRunEvents(runID)
		if taskDef, err := s.taskStore.Get(snap.TaskID); err == nil {
			_ = s.persistFinalRun(s.runStore, taskDef, snap)
		}
		s.removeActiveRun(runID)
		return nil
	}
	if ar.state.Status == RunStatusRunning {
		ar.state.Status = RunStatusStopped
	}
	runner := ar.runner
	ar.mu.Unlock()

	if runner != nil {
		runner.Stop()
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
		if reportData, ok := ar.state.ModeResult.(*types.ReportData); ok {
			standardResult = reportData
		}
		ar.mu.RUnlock()
	} else {
		run, err := runStore.LoadByRunID(string(runID))
		if err != nil || run == nil {
			return "", fmt.Errorf("run %q not found", runID)
		}
		status = RunStatus(run.Metadata.Status)
		mode = run.Metadata.Mode
		if run.Result != nil {
			// 优先从 ModeResult 读取
			if reportData, ok := run.Result.ModeResult.(*types.ReportData); ok {
				standardResult = reportData
			} else if run.Result.StandardResult != nil {
				// 向后兼容：从旧字段读取
				standardResult = run.Result.StandardResult
			}
		}
	}

	if status == RunStatusQueued || status == RunStatusRunning {
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
