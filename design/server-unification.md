# AIT Server 统一化与泛化设计

> 从设计层面分析哪些逻辑可以统一化、泛化，从而简化代码

---

## 一、当前架构的两个"执行路径"

### 1.1 旧路径：`standard.Runner` 自主执行

`standard.Runner` 是一个独立、完整的执行器——拥有自己的请求队列、worker pool、进度跟踪：

```
standard.Runner.Run()
  ├─ runRequestQueue()          ← 自有的 worker pool 实现
  │    ├─ executeRequest()      ← 自有的请求执行
  │    └─ upload.UploadReport() ← 自有的上传
  └─ calculateResult()          ← 聚合统计
```

### 1.2 新路径：Server 统一执行

`serverImpl` 没有使用 `standard.Runner`，而是通过统一基础设施：

```
runStandard
  ├─ client.NewClient()         ← 统一客户端
  ├─ RunRequestBatch()          ← 统一 worker pool
  │    ├─ RequestExecutor       ← 统一请求执行
  │    └─ RequestQueueHooks     ← 生命周期钩子
  ├─ RunAggregator              ← 统一计数+持久化+事件
  └─ standard.CalculateResult() ← 仅用其统计函数
```

**关键发现**：`Run()` 和 `RunWithCallback()` 方法在 server 上下文中从未被调用。server 总是通过 `RunRequestBatch` + `queuedCaseRunner`/`queuedLevelRunner` 绕过 `standard.Runner` 的执行循环。

---

## 二、可统一的模块

### ✅ 2.1 `queuedCaseRunner` + `queuedLevelRunner` → `batchRunner`

**当前状态**（两个结构几乎相同）：

| | queuedCaseRunner | queuedLevelRunner |
|---|---|---|
| 字段 | ctx, input, runID, index, caseID, client, aggregator, stop | ctx, input, runID, level, client, aggregator, results, stop |
| 方法 | `RunWithCallback(cb)` → 1 job, concurrency=1 | `Run()` → N jobs, concurrency=N |
| 结果 | metrics 通过 cb 传出 | results 存入 r.results[] |
| Stop() | ✅ | ✅ |

**统一后**：

```go
// batchRunner 统一执行批量请求，替代 queuedCaseRunner 和 queuedLevelRunner。
type batchRunner struct {
    ctx        context.Context
    input      types.Input
    runID      RunID
    client     client.ModelClient
    aggregator *RunAggregator
    stop       context.CancelFunc

    // 作业参数
    level      int    // turbo 级别，0 表示非 turbo
    caseID     string // integrity case ID，空表示非 integrity
    startIndex int    // 起始索引，默认 0
    count      int    // 请求数，默认 input.Count

    // 收集结果（仅在无回调时使用）
    results []*client.ResponseMetrics
}

func newBatchRunner(parent context.Context, runID RunID, input types.Input,
    client client.ModelClient, agg *RunAggregator) *batchRunner {
    ctx, cancel := context.WithCancel(parent)
    return &batchRunner{
        ctx:        ctx,
        input:      input,
        runID:      runID,
        count:      input.Count,
        client:     client,
        aggregator: agg,
        results:    make([]*client.ResponseMetrics, input.Count),
        stop:       cancel,
    }
}

// WithLevel 设置 turbo 级别。
func (r *batchRunner) WithLevel(level int) *batchRunner { r.level = level; return r }
// WithCaseID 设置 integrity case ID。
func (r *batchRunner) WithCaseID(caseID string, index int) *batchRunner {
    r.caseID = caseID; r.startIndex = index; r.count = 1; return r
}

// Run 执行批量请求，可选回调。
func (r *batchRunner) Run(cb standard.RequestDoneCallback) (*types.ReportData, error) {
    jobs := make([]RequestJob, 0, r.count)
    for i := 0; i < r.count; i++ {
        jobs = append(jobs, RequestJob{
            RunID: r.runID, Index: r.startIndex + i,
            Input: r.input, Level: r.level, CaseID: r.caseID,
        })
    }
    start := time.Now()
    launched := RunRequestBatch(r.ctx, jobs, r.input.Concurrency,
        NewRequestExecutor(r.client), RequestQueueHooks{
            OnQueued:  r.aggregator.MarkQueued,
            OnStarted: r.aggregator.MarkStarted,
            OnSkipped: r.aggregator.MarkSkipped,
            OnDone: func(result RequestResult) {
                if result.Metrics != nil {
                    r.results[result.Job.Index-r.startIndex] = result.Metrics
                }
                rm := r.aggregator.Complete(result)
                if rm.Success {
                    uploadRequest(r.aggregator.taskDef.ID, result.Metrics, r.input)
                }
                if cb != nil {
                    cb(result.Metrics, result.Job.Index, result.Err)
                }
            },
        })
    return standard.CalculateResult(r.input, r.results, time.Since(start), launched), nil
}

func (r *batchRunner) Stop() {
    if r.stop != nil { r.stop() }
}
```

**收益**：
- 两个文件（`queued_case_runner.go`、`queued_level_runner.go`）合并为一个
- `runTurbo` 的 factory 从 `newQueuedLevelRunner(ctx, ...)` 变成 `newBatchRunner(ctx,...).WithLevel(concurrency)`
- `runIntegrity` 的 factory 从 `newQueuedCaseRunner(ctx,...)` 变成 `newBatchRunner(ctx,...).WithCaseID(c.ID, idx)`
- `runStandard` 也可以直接用 `newBatchRunner(ctx,...)` 替代内联的 jobs 创建

---

### ✅ 2.2 `completeStandardRun` + `completeTurboRun` + `completeIntegrityRun` → `finalizeRun`

**当前状态**：三个函数各 ~30 行，结构完全一致：

```
lock
  ├─ set Status (Completed/Stopped/Failed——integrity 特殊)
  ├─ set FinishedAt
  ├─ set ModeResult
  ├─ mode-specific ModeState 更新
  ├─ compute RPM/TPM
  └─ snapshot
unlock
├─ publish event (Complete/Failed/Stopped)
├─ closeRunEvents
├─ persistFinalRun
└─ removeActiveRun
```

**统一后**：

```go
type runFinalizer func(ar *activeRun, finishedAt time.Time) (kind EventKind, snap *RunState)

// finalizeRun 是完成运行的通用模板，差异通过 onLock 回调处理。
func (s *serverImpl) finalizeRun(
    ar *activeRun, runID RunID,
    taskDef types.TaskDefinition,
    runStore *store.RunStore,
    onLock runFinalizer,
) {
    finishedAt := time.Now()
    ar.mu.Lock()
    kind, snap := onLock(ar, finishedAt)
    ar.mu.Unlock()

    s.bus.publishRunEvent(Event{RunID: runID, Kind: kind, Payload: snap})
    s.bus.closeRunEvents(runID)
    if err := s.persistFinalRun(runStore, taskDef, snap); err == nil {
        s.removeActiveRun(runID)
    }
}
```

各模式只需提供 `onLock` 回调：

```go
func (s *serverImpl) completeStandardRun(ar *activeRun, runID RunID, ...) {
    s.finalizeRun(ar, runID, taskDef, runStore, func(ar *activeRun, t time.Time) (EventKind, *RunState) {
        if ar.state.Status != RunStatusStopped {
            ar.state.Status = RunStatusCompleted
        }
        ar.state.FinishedAt = &t
        ar.state.ModeResult = data
        if data != nil {
            ar.state.AvgTPS, ar.state.AvgTTFT = data.AvgTPS, data.AvgTTFT
            ar.state.SuccessRate, ar.state.CacheHitRate = data.SuccessRate, data.AvgCacheHitRate
        }
        if elapsed := t.Sub(ar.state.StartedAt).Minutes(); elapsed > 0 {
            ar.state.RPM = float64(ar.state.DoneReqs) / elapsed
            ar.state.TPM = float64(ar.tokenSum) / elapsed
        }
        snap := ar.snapshotState()
        kind := EventRunComplete
        if snap.Status == RunStatusStopped { kind = EventRunStopped }
        return kind, snap
    })
}
```

**收益**：
- 从 ~90 行重复代码 → ~15 行模板 + 3 个 ~15 行回调
- `failRun` 也自然纳入此模式
- 新增模式时只需实现回调

---

### ✅ 2.3 `RunAggregator` 去除 `serverImpl` 反向引用

**当前**：
```go
type RunAggregator struct {
    server   *serverImpl   // ← 仅为了 a.server.bus.publishRunEvent(...)
    ...
}
```

**统一后**：
```go
type RunAggregator struct {
    bus      *eventBus     // ← 直接注入依赖
    ...
}
```

**收益**：聚合器独立性 ↑，可测试性 ↑，心智负担 ↓。

---

### ✅ 2.4 统一 ModeState key 常量

**当前**：ModeState 的 key 散布在各处作为字符串字面量：

| key | 出现位置 |
|-----|---------|
| `"levels"` | runTurbo, completeTurboRun, snapshotState, buildRunStateFromStoredRun, StoredRun.Summary |
| `"current_level"` | runTurbo, completeTurboRun, buildStoredRunResult, buildRunStateFromStoredRun |
| `"cases"` | runIntegrity, completeIntegrityRun, snapshotState |
| `"current_case_id"` | runIntegrity |
| `"assertion_results"` | runIntegrity, completeIntegrityRun, snapshotState |
| `"suite"` | runIntegrity, StoredRun.Summary |
| `"suite_status"` | runIntegrity, handleRulesStatus |
| `"config"` | runTurbo, buildRunStateFromStoredRun |
| `"rules_status"` | handleRulesStatus, runIntegrity |
| `"state"` | snapshotState (StateProvider 路径) |

**统一后**：

```go
// ModeStateKey 定义 ModeState map 的标准键名。
const (
    ModeStateKeyLevels           = "levels"
    ModeStateKeyCurrentLevel     = "current_level"
    ModeStateKeyCases            = "cases"
    ModeStateKeyCurrentCaseID    = "current_case_id"
    ModeStateKeyAssertionResults = "assertion_results"
    ModeStateKeySuite            = "suite"
    ModeStateKeySuiteStatus      = "suite_status"
    ModeStateKeyConfig           = "config"
    ModeStateKeyRulesStatus      = "rules_status"
)
```

**收益**：拼写错误清零，IDE 补全，重构时一处修改。

---

### ✅ 2.5 统一 `mode` 字符串为 `RunMode` 类型常量

**当前**：代码中散布 `"standard"`、`"turbo"`、`"integrity"` 字符串（13+ 处）。

**统一后**：

```go
type RunMode string

const (
    ModeStandard  RunMode = "standard"
    ModeTurbo     RunMode = "turbo"
    ModeIntegrity RunMode = "integrity"
)
```

所有 `case "standard":` → `case ModeStandard:`。 同时可定义 `RunModes` slice 用于校验和遍历。

---

### ✅ 2.6 移除 `RunResult` 旧版兼容字段

**当前**：

```go
type RunResult struct {
    ModeResult       any                    // 新
    StandardResult  *types.ReportData      // 旧，仅向后兼容
    TurboResult     *types.TurboResult     // 旧，仅向后兼容
    IntegrityResult *types.IntegrityResult // 旧，仅向后兼容
    ...
}
```

这些旧字段在 3 处有回退读取逻辑（`buildRunStateFromStoredRun`、`GenerateRunReport`、`StoredRun.Summary`）。

**建议**：如果磁盘上已有旧格式数据的用户量小，直接移除；否则加一个迁移标记。无论如何，新代码不应再写入这些字段——当前 `buildStoredRunResult` 已只写 `ModeResult`。

---

### ✅ 2.7 消除 `ar.ctx` nil fallback

**当前**：`runStandard`、`runTurbo`、`runIntegrity` 中各有 5 行相同的 fallback：

```go
ctx := ar.ctx
if ctx == nil {
    ctx = s.ctx
    if ctx == nil { ctx = context.Background() }
}
```

**统一后**：在 `StartRun` 创建 `activeRun` 时保证 ctx 非 nil：

```go
parentCtx := ar.ctx
if parentCtx == nil {
    parentCtx = s.ctx
    if parentCtx == nil {
        parentCtx = context.Background()
    }
}
ar.ctx, ar.cancel = context.WithCancel(parentCtx)
```

然后所有 `run*` 方法直接使用 `ar.ctx`，无需 fallback。

---

### ✅ 2.8 提取 `RunRequestBatch` 常见模式

`runStandard` 内联创建 jobs 数组的模式也存在于 `batchRunner.Run()`。统一后 `runStandard` 可简化为：

```go
func (s *serverImpl) runStandard(ar *activeRun, ...) {
    modelClient, err := client.NewClient(input, loggerForInput(input))
    if err != nil { s.failRun(...); return }

    agg := newRunAggregator(...)
    stopTick := s.startProgressTicker(ar, runID)
    defer close(stopTick)

    br := newBatchRunner(ar.ctx, runID, input, modelClient, agg)
    reportData, err := br.Run(nil)
    if err != nil { s.failRun(...); return }

    s.completeStandardRun(ar, runID, taskDef, runStore, reportData)
}
```

Turbo 和 Integrity 的 factory 也相应简化。

---

## 三、不建议统一的模块

### ❌ `runStandard` / `runTurbo` / `runIntegrity` 的完整融合

三个 `run*` 方法的执行模型本质不同：

| | standard | turbo | integrity |
|---|---|---|---|
| 执行模型 | 一次 N 并发请求 | 自适应多级探测 | 顺序 case → 每 case 1 请求 |
| 客户端 | 1 个 client | 1 个 client 复用 | 每 case 新 client |
| 调度 | 直接 RunRequestBatch | 通过 Engine.Run | 通过 Executor.Run |
| 进度 ticker | ✅ | ❌ | ❌ |
| 初始化 | 无 | ModeState["config"] | ModeState["suite"]..."suite_status" |
| 事件 | 仅 Request 级 | 有 LevelDone | 有 CaseStarted/CaseDone/AssertionResult |

强行统一 → 80% 的回调代码 → 可读性 ↓，不如保持三个函数各司其职。

### ❌ `standard.Runner` 与 server `RunRequestBatch` 合并

`standard.Runner` 支持独立使用（不依赖 server），保留此能力有价值。合并会迫使独立使用者引入 server 依赖。

### ❌ 将 `RunState` 泛型化

`RunState` 被 TUI/Web/event/disk 广泛使用，泛型化会传染到 `Event`、`Snapshot`、`JSONStore`——复杂度/收益比极差。

---

## 四、优先级排序

| 优先级 | 统一项 | 影响范围 | 收益 |
|--------|--------|----------|------|
| **P0** | 2.2 complete*Run → finalizeRun | run_service.go | 消除 ~60 行重复 |
| **P0** | 2.1 queuedRunner 合并 | 2 个文件 → 1 个 | 消除重复结构体 |
| **P1** | 2.3 RunAggregator 去反向引用 | run_aggregator.go | 解耦 |
| **P1** | 2.4 ModeState key 常量 | 6 个文件 | 类型安全 |
| **P1** | 2.5 RunMode 类型常量 | 10+ 个文件 | 类型安全 |
| **P1** | 2.7 ar.ctx nil fallback | run_service.go | 消除防御代码 |
| **P2** | 2.6 移除 RunResult 旧字段 | store/run.go | 简化数据结构 |
| **P2** | 2.8 runStandard 用 batchRunner | run_service.go | 代码一致 |

---

## 五、统一后的架构概览

```
serverImpl
  │
  ├─ StartRun → scheduler.Enqueue
  │
  ├─ dispatchQueuedRun
  │    ├─ client.NewClient()
  │    ├─ newRunAggregator(bus, active, ...)       ← 注入 eventBus
  │    ├─ newBatchRunner(ctx, runID, input, ...)    ← 统一 batch runner
  │    │    ├─ .WithLevel(lvl)  (turbo)
  │    │    └─ .WithCaseID(id)  (integrity)
  │    └─ switch mode:
  │         ├─ runStandard  → br.Run()  → finalizeRun(onStandardComplete)
  │         ├─ runTurbo     → engine.Run → finalizeRun(onTurboComplete)
  │         └─ runIntegrity → executor.Run → finalizeRun(onIntegrityComplete)
  │
  └─ finalizeRun(onLock)
       ├─ lock → onLock(ar) → snap
       ├─ publish event
       ├─ closeRunEvents
       ├─ persistFinalRun
       └─ removeActiveRun
```
