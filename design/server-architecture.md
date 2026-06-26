# AIT Server 内部架构分析

> 生成日期: 2026-06-26  
> 范围: `internal/server/` 目录（含 `store/` 子包）

---

## 一、整体架构

```
┌──────────────────────────────────────────────────────────┐
│  TUI (internal/tui/)  │  Web (internal/web/)             │
│                       │                                  │
│  通过 Server 接口交互 (18 个方法)                          │
└───────────────────────┬──────────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────────┐
│  serverImpl                                              │
│  ┌──────────────────────────────────────────────────┐    │
│  │ task_service.go: ListTasks CreateTask Duplicate.. │    │
│  │ meta_service.go:  ListProtocols ValidateTask...   │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  ┌─────────────── 运行生命周期 ─────────────────────┐    │
│  │                                                 │    │
│  │  StartRun                                       │    │
│  │    ├─ task.HydrateInput (解析模板变量)            │    │
│  │    ├─ new activeRun (初始化内存状态)              │    │
│  │    ├─ scheduler.Enqueue (入队 FIFO)              │    │
│  │    └─ bus.publishRunEvent(RunQueued)             │    │
│  │                                                 │    │
│  │  scheduler.loop() ──► dispatchQueuedRun         │    │
│  │    ├─ bus.publishRunEvent(RunStarted)            │    │
│  │    └─ switch mode:                               │    │
│  │       ├─ runStandard → RunRequestBatch → complete│    │
│  │       ├─ runTurbo    → engine.Run → complete     │    │
│  │       └─ runIntegrity → executor.Run → complete  │    │
│  │                                                 │    │
│  │  complete*/failRun/StopRun                       │    │
│  │    ├─ bus.publishRunEvent(Complete/Failed/Stop)  │    │
│  │    ├─ bus.closeRunEvents (关闭订阅通道)           │    │
│  │    ├─ persistFinalRun (落盘保存)                  │    │
│  │    └─ removeActiveRun (从内存移除)               │    │
│  └─────────────────────────────────────────────────┘    │
│                          │                               │
│  ┌──────────────────────────────────────────────────┐    │
│  │ GetRunState: 先查 activeRuns(内存), 再查 runStore │    │
│  │ SubscribeRunEvents: 通过 eventBus 订阅运行事件    │    │
│  │ progressTicker: 500ms 定时推送 EventProgressTick  │    │
│  │ Shutdown: cancel → scheduler.Shutdown → 超时兜底  │    │
│  └──────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────┘
```

---

## 二、组件关系图

```mermaid
graph TD
    subgraph 入口层
        TUI[TUI: model.go]
        Web[Web: api.go]
    end

    subgraph 接口层
        S[Server interface<br/>18 methods]
    end

    subgraph 核心实现
        SI[serverImpl]
    end

    subgraph 调度层
        SCHED[RunScheduler<br/>FIFO + semaphore(max=1)]
    end

    subgraph 运行层
        AR[activeRun<br/>state + ctx + runner]
        SST[snapshotState<br/>深拷贝]
        PT[progressTicker<br/>500ms定时]
    end

    subgraph 执行层
        STD[runStandard]
        TURBO[runTurbo]
        INTEG[runIntegrity]
    end

    subgraph 请求处理
        RQB[RunRequestBatch<br/>worker pool]
        RE[RequestExecutor<br/>client.Request]
        QCR[queuedCaseRunner<br/>integrity单请求]
        QLR[queuedLevelRunner<br/>turbo批量请求]
    end

    subgraph 聚合与事件
        RA[RunAggregator<br/>计数 + 持久化 + 发布]
        EB[eventBus<br/>pub/sub]
    end

    subgraph 持久化
        RS[RunStore<br/>run.json + result.json + requests.jsonl]
        JS[JSONStore<br/>泛型原子写入]
    end

    subgraph 外部服务
        MC[ModelClient<br/>HTTP → LLM API]
    end

    TUI --> S
    Web --> S
    S --> SI
    SI --> SCHED
    SI --> AR
    SI --> RS
    SI --> EB
    SCHED --> STD
    SCHED --> TURBO
    SCHED --> INTEG
    STD --> RQB
    TURBO --> TURBO_ENGINE[Turbo Engine]
    TURBO_ENGINE --> QLR
    INTEG --> INTEG_EXEC[Integrity Executor]
    INTEG_EXEC --> QCR
    QCR --> RQB
    QLR --> RQB
    RQB --> RE
    RE --> MC
    RQB --> RA
    RA --> AR
    RA --> RS
    RA --> EB
    RS --> JS
    AR --> SST
    AR --> PT
    PT --> EB
```

---

## 三、数据流

### 3.1 一次标准运行的完整时序

```
用户 → StartRun(taskID, input)
  │
  ├─ 1. task.HydrateInput → 解析 {{model}} 模板变量
  ├─ 2. new activeRun → 初始化 RunState
  ├─ 3. scheduler.Enqueue → FIFO 入队
  └─ 4. bus.publishRunEvent(RunQueued)

scheduler.loop() 取出队列项 → dispatchQueuedRun
  │
  ├─ 5. bus.publishRunEvent(RunStarted)
  └─ 6. runStandard:
       ├─ client.NewClient → 创建 HTTP 客户端
       ├─ newRunAggregator → 创建聚合器
       ├─ startProgressTicker → 启动 500ms 定时器
       └─ RunRequestBatch(concurrency=N) → 阻塞直到全部完成
            │
            ├─ N 个 worker goroutine 从 RequestQueue 取 Job
            ├─ RequestExecutor.Execute → client.Request → LLM API
            ├─ OnDone → RunAggregator.Complete:
            │   ├─ runStore.AppendRequest → atomic write JSONL
            │   ├─ 更新 activeRun.state (计数, TPS, TTFT, CacheRate, RPM, TPM)
            │   └─ bus.publishRunEvent(RequestDone)
            └─ RunAggregator.MarkQueued/Started/Skipped → 更新 RequestStates
  │
  ├─ 7. defer close(stopTick) → 停止定时器
  ├─ 8. standard.CalculateResult → 聚合统计
  └─ 9. completeStandardRun:
       ├─ 设置 Status=Completed, FinishedAt, ModeResult
       ├─ 计算最终 RPM/TPM
       ├─ bus.publishRunEvent(RunComplete)
       ├─ bus.closeRunEvents
       ├─ persistFinalRun:
       │   ├─ JSONStore.Save(run.json)   ← 元数据
       │   └─ JSONStore.Save(result.json) ← 结果
       └─ removeActiveRun → 从内存移除
```

### 3.2 持久化文件结构

```
~/.ait/
  tasks/
    {taskID}.json           ← JSONStore (task definition)
    views.json              ← JSONStore (任务列表视图, debounced)
  runs/
    {taskID}/
      {runID}/
        run.json            ← RunMetadata (JSONStore)
        result.json         ← RunResult (JSONStore, 含预计算汇总)
        requests.jsonl      ← 每行一个 RequestMetrics JSON
```

---

## 四、核心类型清单

### 4.1 状态类型

| 类型 | 位置 | 用途 |
|------|------|------|
| `RunState` | `server/types.go:57` | 运行完整快照，深拷贝后传递给 TUI |
| `RunStatus` | `server/types.go:47` | `queued/running/completed/failed/stopped` |
| `RequestState` | `server/types.go:50` | 单个请求的队列状态 |
| `Event` | `server/types.go:120` | `{RunID, Kind, Payload}` |
| `EventKind` | `server/types.go:114` | 17 种事件类型常量 |

### 4.2 运行数据

| 类型 | 位置 | 用途 |
|------|------|------|
| `activeRun` | `server/run_service.go:27` | 运行时内存状态（含锁、cancel、runner） |
| `RunAggregator` | `server/run_aggregator.go:13` | 请求生命周期钩子 + 计数 + 持久化 |
| `RequestJob` | `server/request_executor.go:21` | 一个模型请求的作业描述 |
| `RequestResult` | `server/request_executor.go:29` | 请求执行结果 |
| `RequestExecutor` | `server/request_executor.go:17` | 封装 ModelClient |

### 4.3 持久化类型

| 类型 | 位置 | 用途 |
|------|------|------|
| `RunStore` | `server/store/run.go:67` | 管理 runs/ 目录的读写 |
| `RunMetadata` | `server/store/run.go:17` | run.json 内容 |
| `RunResult` | `server/store/run.go:28` | result.json 内容 |
| `StoredRun` | `server/store/run.go:59` | Metadata + Result 组合 |
| `JSONStore[T]` | `server/store/store.go` | 泛型 JSON 原子写入 |

### 4.4 基础设施

| 类型 | 位置 | 用途 |
|------|------|------|
| `RunScheduler` | `server/scheduler.go:21` | FIFO 运行调度，maxRunning=1 |
| `eventBus` | `server/event_bus.go` | 发布/订阅，每 RunID 多个 subscriber |
| `RequestQueue` | `server/request_queue.go:13` | 请求级 worker pool |
| `Queue[T]` | `server/queue/queue.go` | 泛型 FIFO 队列 |

---

## 五、架构问题分析

### ? 问题 1: 三个 `complete*Run` 函数高度重复

**严重程度**: 中

`completeStandardRun`、`completeTurboRun`、`completeIntegrityRun` 三个函数结构几乎完全相同：

```
lock → 设置 Status/FinishedAt/ModeResult → 模式特定 ModeState 更新
→ 计算 RPM/TPM → snapshot → publish → closeEvents → persist → removeActive
```

**差异点**：
- `completeStandardRun`: 设置 AvgTPS/AvgTTFT/SuccessRate/CacheHitRate
- `completeTurboRun`: 更新 ModeState["levels"] + ModeState["current_level"]
- `completeIntegrityRun`: 可能将 Completed 覆盖为 Failed（语义异常）, 更新 ModeState["cases"] + ModeState["assertion_results"]

**建议**：提取 `completeRunTemplate(status ResultKind, setup func(snap *RunState))` 公共模板，差异逻辑用 `setup` 回调处理。

---

### ? 问题 2: `completeIntegrityRun` 中 `RunStatusCompleted` 被覆盖为 `RunStatusFailed`

**严重程度**: 高（语义混淆）

```go
if ar.state.Status != RunStatusStopped {
    ar.state.Status = RunStatusCompleted
    if result != nil && result.Status == "failed" {
        ar.state.Status = RunStatusFailed  // ← 覆盖
    }
}
```

- 函数的命名是 `completeIntegrityRun`，暗示"成功完成"
- 但内部却可能将状态改成 `Failed`
- 同时 `failRun` 也是设置 `RunStatusFailed`，两者语义重叠

**建议**：
- 将"运行完成但测试失败"和"运行过程异常失败"分开：
  - 过程异常 → `RunStatusFailed`（网络错误、超时等）
  - 测试不通过 → `RunStatusCompleted` + `IntegrityResult.Status = "failed"`

---

### ? 问题 3: `mode` 是裸字符串，缺少类型安全

**严重程度**: 低

`mode` 字段在 `RunState`、`RunMetadata`、`StoredRun.Summary` 等地方都是 `string`，到处硬编码 `"standard"`、`"turbo"`、`"integrity"`。

**建议**：定义 `type RunMode string` + 常量 `ModeStandard/ModeTurbo/ModeIntegrity`。

---

### ? 问题 4: `ModeState` 和 `ModeResult` 是 `any`，丢失类型安全

**严重程度**: 中

- `RunState.ModeState map[string]any` — key 没有统一常量，到处用字符串字面量
- `RunState.ModeResult any` — 三种具体类型 `*ReportData/*TurboResult/*IntegrityResult`

**建议**：
- 至少定义 `ModeState` 的 key 常量：`ModeStateKeySuite`, `ModeStateKeyLevels`, `ModeStateKeyCases` 等
- `snapshotState` 中的 `switch val := v.(type)` 已知类型列表需要和所有赋值点保持同步，容易遗漏

---

### ? 问题 5: `RunAggregator` 持有 `*serverImpl` 反向引用

**严重程度**: 低

```go
type RunAggregator struct {
    server   *serverImpl  // ← 反向引用
    active   *activeRun
    ...
}
```

`Complete()` 调用 `a.server.bus.publishRunEvent(...)`。虽然同 package 不会循环 import，但增加了心智负担。

**建议**：将 `*eventBus` 直接注入 `RunAggregator`，消除对 `serverImpl` 的依赖。

---

### ? 问题 6: `doneCount` 字段未使用

**严重程度**: 极低

`activeRun.doneCount` 在代码库中仅定义处出现，没有任何读取和使用。

```go
doneCount int   // 与 state.DoneReqs 保持同步，方便不加锁时计算
```

**建议**：直接删除。

---

### ? 问题 7: 全局同时运行数 `maxRunning` 硬编码为 `1`

**严重程度**: 低

`server.go:90`:
```go
scheduler := newRunScheduler(1, s.dispatchQueuedRun)
```

`newRunScheduler` 接收 `maxRunning` 参数，说明设计上考虑了可配置性，但未暴露。目前全局只能同时运行一个任务，如果 TUI 和 Web 同时启动运行，后者需排队。

**建议**：要么移除参数（明确单运行语义），要么暴露到配置层。

---

### ? 问题 8: `RunResult` 携带向后兼容的旧字段

**严重程度**: 低

```go
type RunResult struct {
    ModeResult       any  // 新字段 (泛型)
    StandardResult  *types.ReportData      // 旧字段
    TurboResult     *types.TurboResult     // 旧字段
    IntegrityResult *types.IntegrityResult // 旧字段
    ...
}
```

加载时有 3 处回退逻辑（`buildRunStateFromStoredRun`、`GenerateRunReport`、`StoredRun.Summary`）。

**建议**：如果有明确的迁移计划（数据迁移脚本或足够时间窗口），可以移除这些旧字段和对应回退代码。

---

### ? 问题 9: `runQueueItem` 全量复制大对象

**严重程度**: 极低

```go
type runQueueItem struct {
    TaskDef types.TaskDefinition
    Input   types.Input
    RunID   RunID
}
```

`TaskDefinition` 和 `Input` 可能较大（含 Request Body 等），在入队时会有一次完整拷贝。当前只有 1 个运行并发，实际影响小，但如果将来提高并发度，可考虑用指针。

---

### ? 问题 10: Context fallback 路径脆弱

**严重程度**: 低

多处代码存在 `ar.ctx` 为 nil 时的 fallback：

```go
ctx := ar.ctx
if ctx == nil {
    ctx = s.ctx
    if ctx == nil {
        ctx = context.Background()
    }
}
```

- 在 `runIntegrity` (line ~553) 和 `runTurbo` (line ~646) 各有一份
- 测试环境中 `newTestServer` 不设置 `ctx`，容易产生"测试过但生产有细微差异"的问题

**建议**：在 `newActiveRun` / `StartRun` 入口处保证 `ctx` 永不 nil，消除所有 fallback。

---

### ? 问题 11: `Lock → snapshotState → Unlock → Lock → Unlock` 双重加锁模式

**严重程度**: 低

多处代码存在以下模式：

```go
ar.mu.Lock()
...更新 state...
ar.mu.Unlock()

// 后续可能在其他 goroutine 中
ar.mu.Lock()
...再次更新 state...
ar.mu.Unlock()
```

这在 `completeIntegrityRun` 的 `OnCaseStarted` / `OnCaseDone` 回调中尤为明显。由于每次 `Lock/Unlock` 之间其他 goroutine 可能修改 state，在单次锁内应尽可能完成所有相关修改。

**建议**：尽量在一次 `Lock/Unlock` 中完成逻辑上原子的 state 修改；如当前设计确实需要分步，应添加注释说明原因。

---

## 六、优化建议（按优先级）

### P0 — 语义修复

1. **修复 `completeIntegrityRun` 的 `RunStatusFailed` 语义**  
   运行完成但测试不通过 → `RunStatusCompleted` + `IntegrityResult.Status = "failed"`，不要覆盖 `RunStatus`。

### P1 — 代码简化

2. **提取 `complete*Run` 公共模板**  
   三个函数合并为一个，用策略模式处理差异。

3. **删除 `doneCount`**  
   无任何引用，直接移除。

4. **删除 `RunResult` 旧字段**  
   如果确认迁移完成，移除 `StandardResult/TurboResult/IntegrityResult` 三个旧字段和所有回退逻辑。

### P2 — 类型安全

5. **引入 `RunMode` 类型常量**  
   替代所有 `"standard"` / `"turbo"` / `"integrity"` 硬编码。

6. **定义 `ModeState` key 常量**  
   替代 `"levels"`, `"cases"`, `"current_level"` 等字符串字面量。

7. **将 `*eventBus` 直接注入 `RunAggregator`**  
   消除 `RunAggregator` → `serverImpl` 反向引用。

### P3 — 健壮性

8. **保证 `ar.ctx` 永不 nil**  
   在创建入口处统一设置，移除所有 fallback 路径。

---

## 七、良好实践（值得保留）

| 实践 | 说明 |
|------|------|
| `snapshotState` 深拷贝 | 所有传递给 TUI/Web 的 `RunState` 都是独立副本，避免并发问题 |
| 非阻塞事件投递 | `select { case ch <- ev: default: }` 消费者慢时不阻塞生产者 |
| 两层 Context | Server 层 + 运行层，Shutdown 先 cancel 再等 scheduler 完成 |
| 预计算摘要 | `RunResult` 存储 `DoneReqs/SuccessRate/AvgTTFT/RPM/TPM`，任务列表不扫 JSONL |
| JSONStore 原子写入 | tmp + rename 模式保证不会读到半写文件 |
| LoadRequests 容忍损坏 | 跳过损坏 JSONL 行 + warn，不阻断整个加载 |
| JSONL Append + f.Sync | 确保写入真正落盘 |
| 泛型 Queue[T] | 复用队列逻辑 |
| 接口解耦 | `Server` 接口统一对外，`serverImpl` 未导出 |
