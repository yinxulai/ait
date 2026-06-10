# Turbo 模式功能设计

> 版本：v0.1 设计草案  
> 日期：2026-06-10

---

## 目录

1. [概述](#1-概述)
2. [设计目标与非目标](#2-设计目标与非目标)
3. [任务配置](#3-任务配置)
4. [算法流程](#4-算法流程)
5. [停止条件](#5-停止条件)
6. [运行事件与状态](#6-运行事件与状态)
7. [结果模型与存储](#7-结果模型与存储)
8. [TUI 页面设计](#8-tui-页面设计)
9. [报告格式](#9-报告格式)
10. [推荐实现位置](#10-推荐实现位置)
11. [开发分期](#11-开发分期)

---

## 1. 概述

Turbo 模式是 AIT 除标准固定并发测试之外的承载能力探测模式。

它的核心目标是回答：

> 在当前模型、协议、接口地址、Prompt 与运行参数下，目标服务能够稳定支撑的最大并发是多少？

标准模式关注固定并发下的性能表现；Turbo 模式关注并发逐级提升过程中的稳定性边界。

典型输出包括：

- 最大稳定并发。
- 峰值 TPS。
- 每个并发级别的成功率、TTFT、TPS、总耗时和缓存命中率。
- 停止原因。
- 并发爬坡曲线。

---

## 2. 设计目标与非目标

### 2.1 目标

1. **自动探测稳定并发边界**

   从初始并发开始逐级提升，自动判断是否出现降级，并给出最大稳定并发。

2. **复用标准请求执行与指标采集能力**

   Turbo 模式不重新实现协议请求逻辑，底层复用标准模式的请求执行、网络指标、token 指标和错误采集。

3. **保存每级请求事实与最终爬坡结果**

   每个请求仍写入统一 `requests.jsonl`；每个并发级别和最终结论写入 `result.json`。

4. **提供可解释停止原因**

   用户需要知道停止是因为成功率下降、延迟超阈值、达到上限，还是手动标记。

5. **适配 TUI 与报告**

   TUI 需要展示当前级别、级别列表和爬坡结论；报告需要输出 JSON / CSV。

### 2.2 非目标

Turbo 模式当前阶段不做以下事情：

- 不做协议完整性判断；协议结构、错误格式和能力项验证属于接口完整性测试。
- 不做复杂自适应搜索；首版采用简单线性爬坡。
- 不做多模型批量对比；一个任务只绑定一个模型。
- 不把运行中的实时事件作为事实源；最终以 `requests.jsonl` 和 `result.json` 为准。

---

## 3. 任务配置

Turbo 模式使用统一任务模式字段：

```json
{
  "mode": "turbo",
  "protocol": "openai-responses",
  "endpoint": "https://api.openai.com/v1/responses",
  "model": "gpt-4o",
  "turbo": {
    "init_concurrency": 1,
    "max_concurrency": 50,
    "step": 2,
    "level_requests": 30,
    "min_success_rate": 0.9,
    "max_latency_ms": 10000
  }
}
```

字段说明：

| 字段 | 类型 | 说明 |
|------|------|------|
| `mode` | string | 固定为 `turbo` |
| `turbo.init_concurrency` | number | 初始并发数，默认 1 |
| `turbo.max_concurrency` | number | 最大探测并发数，默认 50 |
| `turbo.step` | number | 每级并发递增值，默认 2 |
| `turbo.level_requests` | number | 每级请求数，默认 30 |
| `turbo.min_success_rate` | number | 最低可接受成功率，默认 0.9 |
| `turbo.max_latency_ms` | number | 平均总耗时阈值，默认 10000 |

CLI 兼容层可以继续接受 `--turbo` 和 `--turbo-*` 参数，但进入任务模型后应归一化为 `mode = "turbo"` 与 `TurboConfig`。

---

## 4. 算法流程

```text
初始化
  concurrency = TurboConfig.InitConcurrency
  step        = TurboConfig.StepSize
  levelReqs   = TurboConfig.LevelRequests

循环
  1. 用当前 concurrency 执行 levelReqs 个请求
     - 复用标准 runner
     - 每条请求写入统一 requests.jsonl

  2. 聚合当前级别指标
     - successRate
     - avgTPS / peakTPS
     - avgTTFT
     - cacheHitRate
     - avgTotalTime

  3. 判断当前级别是否稳定
     - successRate >= MinSuccessRate
     - avgTotalTime <= MaxLatency
     - 无用户停止

  4. 记录 LevelResult

  5. 判断是否停止
     - 当前级别不稳定
     - concurrency >= MaxConcurrency
     - 用户手动停止或标记极限

  6. 如果继续
     - concurrency += step

结束
  - 最大稳定并发 = 最后一个 stable=true 的级别并发
  - 峰值 TPS = 所有稳定级别中的最大 avgTPS 或 peakTPS
  - 写入 result.json
```

缓存命中率定义为：

```text
cache_hit_rate = cached_input_tokens / max(input_tokens, 1)
```

协议字段映射：

- OpenAI Completions / Responses：读取 `usage.input_tokens_details.cached_tokens` 或等价字段。
- Anthropic Messages：读取 `usage.cache_read_input_tokens`。
- 若响应未提供缓存统计字段，则该指标记为 `N/A`，不参与停止条件判断。

---

## 5. 停止条件

### 5.1 StopReason

```go
type StopReason string

const (
    StopReasonLowSuccessRate  StopReason = "low_success_rate"
    StopReasonHighLatency     StopReason = "high_latency"
    StopReasonMaxConcurrency  StopReason = "max_concurrency"
    StopReasonManual          StopReason = "manual"
    StopReasonMarkedLimit     StopReason = "marked_limit"
    StopReasonRunError        StopReason = "run_error"
)
```

### 5.2 稳定级别判断

一个级别满足以下条件时视为稳定：

1. `success_rate >= min_success_rate`。
2. `avg_total_time <= max_latency`。
3. 当前级别没有发生运行级错误。
4. 用户没有手动停止。

示例：

```text
并发=8:  成功率=96%, avgTPS=245, avgTTFT=312ms, avg_total=1.51s → ✓ 稳定
并发=10: 成功率=84%                                             → ✗ 停止
最大稳定并发 = 8
```

### 5.3 手动标记极限

TUI 中 `[m]` 表示用户将当前并发标记为最大稳定边界并停止运行。

规则：

- 若当前级别已完成且稳定，最大稳定并发取当前级别。
- 若当前级别仍在运行，先停止后按已完成请求聚合当前级别；是否稳定由阈值判断。
- `stop_reason = marked_limit`。

---

## 6. 运行事件与状态

Turbo 模式复用统一运行事件模型，新增或使用以下事件：

```go
const (
    EventRequestDone  EventKind = "request_done"
    EventProgressTick EventKind = "progress_tick"
    EventLevelStarted EventKind = "level_started"
    EventLevelDone    EventKind = "level_done"
    EventRunComplete  EventKind = "run_complete"
    EventRunFailed    EventKind = "run_failed"
)
```

运行态建议包含：

```go
type TurboRunSnapshot struct {
    CurrentConcurrency int
    CurrentLevelIndex  int
    CurrentDone        int
    CurrentTotal       int
    Levels             []TurboLevelResult
    MaxStableSoFar     int
    StopReason         string
}
```

运行态只用于 TUI / MCP 实时展示，不作为最终事实源。最终结果以统一运行目录中的 `requests.jsonl` 和 `result.json` 为准。

---

## 7. 结果模型与存储

Turbo 模式遵循 [统一运行存储设计](storage.md)：

```text
~/.ait/runs/<task-id>/<run-id>/
  run.json
  requests.jsonl
  result.json
```

### 7.1 requests.jsonl

每个请求使用统一请求事实格式。Turbo 模式在公共字段基础上补充 `turbo` 上下文：

```json
{
  "kind": "request",
  "sequence": 42,
  "mode": "turbo",
  "turbo": {
    "level_index": 4,
    "concurrency": 8,
    "request_in_level": 12
  },
  "request": {},
  "response": {},
  "derived": {
    "usage": {},
    "metrics": {}
  },
  "error": null
}
```

### 7.2 result.json

`result.json` 使用统一外壳，并在 `turbo` 字段保存最终爬坡结果：

```json
{
  "mode": "turbo",
  "status": "completed",
  "summary": {
    "max_stable_concurrency": 8,
    "peak_tps": 245.3,
    "stop_reason": "low_success_rate"
  },
  "turbo": {
    "config": {},
    "levels": []
  }
}
```

---

## 8. TUI 页面设计

```text
╔══ AIT  Turbo 探测 ─ turbo-anthropic ───────────────────════╗
║  ◆ AIT   claude-3-7 · anthropic-messages · 1→50  步进+2    ║
╠══════════════════════════╦═════════════════════════════════╣
║  任务参数                 ║  当前级别实时指标 [并发 = 8]    ║
║                           ║                                 ║
║  协议  messages           ║  成功率    96.0%                ║
║  模型  claude-3-7         ║  TPS       245.3 tok/s          ║
║  爬坡  1→50  步进+2       ║  TTFT      312ms                ║
║  每级  30 请求            ║  Cache     44.0%                ║
║  停止  成功率 < 90%       ║  总耗时    1.51s                ║
║        延迟 > 10s         ║  已用      38.2s                ║
╠══════════════════════════╩═════════════════════════════════╣
║  进度  ████████░░  28/30  当前并发 8   总进度: 已完成 4/~25 级║
╠═════════════════════════════════════════════════════════════╣
║  级别列表                                                    ║
║  并发  成功率    TPS      TTFT    Cache   总耗时   结论      ║
║  ──────────────────────────────────────────────────────── ║
║     1  100.0%    31.2    89ms     0.0%    0.82s   ✓ 稳定   ║
║     2  100.0%    62.5    91ms    18.0%    0.84s   ✓ 稳定   ║
║     4   99.0%   121.3    98ms    26.0%    0.91s   ✓ 稳定   ║
║     6   98.0%   178.4   124ms    33.0%    1.08s   ✓ 稳定   ║
║  ▶  8   96.0%   245.3   312ms    44.0%    1.51s  🔄 进行中 ║
╠═════════════════════════════════════════════════════════════╣
║  [Enter] 查看该级别请求列表  [↑↓] 选择                      ║
╠═════════════════════════════════════════════════════════════╣
║  [s] 停止  [b] 后台运行  [m] 标记极限  [r] 提前报告  [q] 退出║
╚═════════════════════════════════════════════════════════════╝
```

说明：

- 主视图展示当前并发级别，而不是总请求列表。
- 级别列表是 Turbo 的核心视图。
- 选中已完成级别后，可进入该级别请求列表；请求列表复用标准模式请求详情能力。
- 完成后任务详情页展示最大稳定并发、峰值 TPS、停止原因和级别表。

---

## 9. 报告格式

### 9.1 JSON 报告

```json
{
  "turbo": {
    "max_stable_concurrency": 8,
    "peak_tps": 245.3,
    "stop_reason": "low_success_rate",
    "probe_duration": "52.3s",
    "protocol": "openai-responses",
    "endpoint_url": "https://api.openai.com/v1/responses",
    "config": {
      "init_concurrency": 1,
      "max_concurrency": 50,
      "step": 2,
      "level_requests": 30,
      "min_success_rate": 0.9,
      "max_latency": "10s"
    },
    "levels": [
      {
        "concurrency": 1,
        "total_requests": 30,
        "success_count": 30,
        "success_rate": 1.0,
        "avg_tps": 31.2,
        "avg_ttft": "89ms",
        "cache_hit_rate": 0.0,
        "avg_total_time": "0.82s",
        "stable": true
      }
    ]
  }
}
```

### 9.2 CSV 报告

```csv
protocol,concurrency,success_rate,avg_tps,peak_tps,avg_ttft,cache_hit_rate,avg_total_time,stable
openai-responses,1,1.00,31.2,38.5,89ms,0.00,0.82s,true
openai-responses,2,1.00,62.5,74.1,91ms,0.18,0.84s,true
openai-responses,4,0.99,121.3,145.2,98ms,0.26,0.91s,true
openai-responses,6,0.98,178.4,201.3,124ms,0.33,1.08s,true
openai-responses,8,0.96,245.3,280.1,312ms,0.44,1.51s,true
openai-responses,10,0.84,198.1,234.5,892ms,0.12,4.23s,false
```

---

## 10. 推荐实现位置

```text
internal/server/
  turbo/
    engine.go       # 爬坡调度
    strategy.go     # 稳定性判断和停止策略
    result.go       # TurboResult / LevelResult 聚合
    types.go        # 配置和运行态类型

  runner/
    runner.go       # 标准请求执行，被 turbo 复用

  report/
    turbo_json.go
    turbo_csv.go
```

TUI 侧建议新增或维护：

```text
internal/tui/pages/
  turbodash.go      # Turbo 运行页
  reqdetail.go      # 级别请求详情复用标准请求详情
```

---

## 11. 开发分期

### Phase 1：类型与配置

- 统一任务模式为 `mode = standard | turbo | integrity`。
- 定义 `TurboConfig`、`TurboLevelResult`、`TurboResult`。
- CLI 参数归一化为 `TurboConfig`。

### Phase 2：执行引擎

- 实现 `internal/server/turbo/engine.go`。
- 复用标准 runner 执行每个并发级别。
- 每条请求写入统一 `requests.jsonl`。

### Phase 3：策略与结果

- 实现稳定性判断。
- 实现停止原因。
- 聚合最大稳定并发和峰值 TPS。
- 写入 `result.json`。

### Phase 4：TUI 与报告

- 实现 Turbo 仪表盘。
- 支持级别列表与级别请求详情。
- 实现 JSON / CSV 报告。
- 在任务详情页展示 Turbo 结果摘要。
