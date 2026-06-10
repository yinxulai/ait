# 统一运行存储设计

> 版本：v0.2 设计草案  
> 日期：2026-06-10

---

## 目录

1. [概述](#1-概述)
2. [设计目标与非目标](#2-设计目标与非目标)
3. [目录布局](#3-目录布局)
4. [核心数据模型](#4-核心数据模型)
5. [统一请求事实记录](#5-统一请求事实记录)
6. [统一结果文件](#6-统一结果文件)
7. [写入模型](#7-写入模型)
8. [读取模型](#8-读取模型)
9. [不持久化的内容](#9-不持久化的内容)
10. [Repository 建议](#10-repository-建议)
11. [代价与取舍](#11-代价与取舍)

---

## 1. 概述

本文定义 AIT 下一版统一运行存储模型。

存储设计不应该只服务某一个功能。标准模式、Turbo 模式、接口完整性测试都应共享同一套底层运行事实模型：

- 统一任务定义。
- 统一运行目录。
- 统一请求事实记录 `requests.jsonl`。
- 统一最终业务结果 `result.json`。

各模式可以在统一基础结构上增加自己的上下文字段，但不能各自发明独立的请求记录格式或运行目录结构。

---

## 2. 设计目标与非目标

### 2.1 目标

1. **统一底层事实模型**

   标准模式、Turbo 模式、接口完整性测试都使用同一种请求事实记录结构，便于后续查询、报告、回放和对比。

2. **只保存业务本体和最终业务结果**

   存储层只保存配置、任务、运行元数据、请求事实和最终结果。不保存 UI 视图、不保存中间态快照、不保存可稳定重算的冗余副本。

3. **保留足够的服务端返回结构**

   请求事实记录应尽可能保留 HTTP 状态、响应头、响应体、错误体、流式事件、解析错误和派生指标，满足性能分析、协议兼容性检查和后续内容检查。

4. **模式扩展不破坏公共格式**

   Turbo 的并发级别、完整性测试的 Case / Assertion 都放在模式上下文中，不改变公共请求事实结构。

5. **最终结果单文件收敛**

   每次运行只有一份 `result.json` 表达最终业务结论。任务列表、历史列表、报告摘要都从 `run.json`、`requests.jsonl`、`result.json` 读取或聚合。

### 2.2 非目标

当前阶段不做以下事情：

- 不维护任务列表缓存。
- 不维护历史索引。
- 不维护 active runs / recent runs 视图文件。
- 不保存运行中的 UI 快照。
- 不保存独立 lifecycle 事件流作为事实源。
- 不保存报告产物清单。
- 不为查询速度维护投影表或二级索引。

---

## 3. 目录布局

推荐目录布局：

```text
~/.ait/
  config.json
  tasks/
    <task-id>.json
  runs/
    <task-id>/
      <run-id>/
        run.json
        requests.jsonl
        result.json
```

说明：

- `config.json` 保存全局配置。
- `tasks/<task-id>.json` 保存任务定义。
- `runs/<task-id>/<run-id>/run.json` 保存最小运行元数据。
- `requests.jsonl` 保存请求级事实，是所有模式共享的底层事实源。
- `result.json` 保存本次运行最终业务结论。

不额外引入 `views/`、`snapshot/`、`history/`、`artifacts/` 等目录。

---

## 4. 核心数据模型

### 4.1 config.json

只保存全局配置，例如：

- 默认协议。
- 上次选中的任务。
- 是否保存 API Key。
- 默认输出目录。

不保存任何运行态信息。

### 4.2 tasks/<task-id>.json

每个任务一个文件，任务是可重复执行的测试配置本体。

建议结构：

```json
{
  "name": "nightly-openai",
  "input": {
    "mode": "standard",
    "protocol": "openai-responses",
    "endpoint": "https://api.openai.com/v1/responses",
    "model": "gpt-4o"
  },
  "created_at": "2026-06-10T10:00:00Z",
  "updated_at": "2026-06-10T10:00:00Z"
}
```

说明：

- `task_id` 由文件名表达，不在文件体内重复存储。
- `input` 是任务配置本体，包含标准模式、Turbo 模式或完整性测试模式所需参数。
- 任务文件不保存 `last_run_at`、`last_run_summary`、`history`、`report_path` 等运行摘要。

### 4.3 runs/<task-id>/<run-id>/run.json

`run.json` 保存最小运行元数据，用来描述这次运行属于谁、何时开始、何时结束、最终状态是什么。

建议结构：

```json
{
  "mode": "integrity",
  "protocol": "openai-responses",
  "endpoint": "https://api.example.com/v1/responses",
  "model": "example-model",
  "status": "completed",
  "started_at": "2026-06-10T10:00:00Z",
  "finished_at": "2026-06-10T10:01:12Z",
  "ait_version": "v0.2.0"
}
```

说明：

- `task_id` 和 `run_id` 由目录路径表达，不在 `run.json` 中重复存储。
- `run.json` 不承担请求明细和最终业务结果存储。

---

## 5. 统一请求事实记录

`requests.jsonl` 是请求级业务事实源。每行一条请求事实记录。

### 5.1 公共结构

```json
{
  "kind": "request",
  "sequence": 1,
  "mode": "standard",
  "request": {
    "method": "POST",
    "url": "https://api.example.com/v1/responses",
    "headers": {},
    "body": {}
  },
  "response": {
    "status_code": 200,
    "headers": {},
    "body": {},
    "error_body": null,
    "stream_events": [],
    "parse_error": null
  },
  "derived": {
    "text": "Hello.",
    "usage": {
      "input_tokens": 10,
      "output_tokens": 8,
      "cached_tokens": 0
    },
    "metrics": {
      "total_ms": 820,
      "ttft_ms": 120,
      "tps": 18.4,
      "cache_hit_rate": 0
    },
    "network": {
      "dns_ms": 1.2,
      "connect_ms": 2.1,
      "tls_ms": 8.4,
      "target_ip": "203.0.113.10"
    }
  },
  "error": null,
  "contexts": {}
}
```

### 5.2 字段分层

| 字段 | 含义 |
|------|------|
| `kind` | 记录类型，首版主要为 `request`，完整性测试 Case 也落为请求事实 |
| `sequence` | 本次运行中的顺序号 |
| `mode` | `standard` / `turbo` / `integrity` |
| `request` | 实际发出的 HTTP 请求事实 |
| `response` | 服务端响应事实，尽可能保留结构 |
| `derived` | 从请求和响应中派生出的标准字段 |
| `error` | 本次请求级错误摘要 |
| `contexts` | 模式上下文，存放 Turbo / Integrity 等模式特有字段 |

### 5.3 标准模式上下文

标准模式不需要额外上下文，必要时可以记录请求批次信息：

```json
{
  "contexts": {
    "standard": {
      "request_index": 1
    }
  }
}
```

### 5.4 Turbo 模式上下文

Turbo 模式在 `contexts.turbo` 中记录并发级别：

```json
{
  "contexts": {
    "turbo": {
      "level_index": 4,
      "concurrency": 8,
      "request_in_level": 12
    }
  }
}
```

### 5.5 完整性测试上下文

完整性测试在 `contexts.integrity` 中记录 Case、Capability 和断言结果：

```json
{
  "contexts": {
    "integrity": {
      "suite": "openai-responses-smoke",
      "case_id": "basic-response-shape",
      "capability": "basic_request",
      "assertions": [
        {
          "id": "responses.output.exists",
          "level": "error",
          "path": "response.body.output",
          "op": "exists",
          "passed": true,
          "source": "builtin"
        }
      ]
    }
  }
}
```

### 5.6 流式响应事件

流式响应事件直接保存在 `response.stream_events` 中：

```json
{
  "seq": 1,
  "time_ms": 120,
  "event": "response.output_text.delta",
  "parsed": {},
  "parse_error": null
}
```

首版不引入额外原始响应目录。需要做协议结构检查和失败排查的内容，应尽量进入 `response` 和 `derived`。

---

## 6. 统一结果文件

`result.json` 是本次运行最终业务结果。它使用统一外壳，并按模式写入模式结果区块。

### 6.1 公共结构

```json
{
  "mode": "integrity",
  "status": "completed",
  "summary": {},
  "standard": null,
  "turbo": null,
  "integrity": {}
}
```

字段说明：

| 字段 | 含义 |
|------|------|
| `mode` | 本次运行模式 |
| `status` | `completed` / `failed` / `stopped` |
| `summary` | 面向列表和摘要展示的最终结论，不是独立视图缓存 |
| `standard` | 标准模式最终结果 |
| `turbo` | Turbo 模式最终结果 |
| `integrity` | 接口完整性测试最终结果 |

### 6.2 标准模式结果

标准模式最终结果保存固定并发测试结论，例如：

- 请求总数。
- 成功率。
- TTFT / TPS / 总耗时统计。
- token 与 cache 统计。
- 错误摘要。

### 6.3 Turbo 模式结果

Turbo 模式最终结果保存：

- 配置。
- 每个并发级别结果。
- 最大稳定并发。
- 峰值 TPS。
- 停止原因。

### 6.4 完整性测试结果

完整性测试最终结果保存：

- Suite 信息。
- Case 统计。
- Capability 覆盖。
- 断言统计。
- 失败摘要。
- 自定义规则来源。

---

## 7. 写入模型

### 7.1 创建或更新任务

只写任务文件：

```text
tasks/<task-id>.json
```

不回填运行摘要。

### 7.2 启动运行

启动运行时：

1. 创建 `runs/<task-id>/<run-id>/`。
2. 写入 `run.json`，状态为 `running`。

不写列表视图，不写历史索引。

### 7.3 运行中

每完成一个请求或完整性 Case：

1. 追加一条请求事实到 `requests.jsonl`。
2. 内存运行态更新，用于 TUI / MCP 实时展示。

不写运行中快照，不写任务摘要。

### 7.4 运行结束

运行结束时：

1. 更新 `run.json` 的最终状态和结束时间。
2. 写入 `result.json`。

到此结束，不做额外索引更新。

---

## 8. 读取模型

### 8.1 任务列表

通过扫描 `tasks/` 目录得到任务定义。

如需展示上次运行摘要：

1. 扫描 `runs/<task-id>/`。
2. 找到最近 run。
3. 读取其 `run.json` 和 `result.json`。
4. 现场构造列表摘要。

### 8.2 任务历史

通过扫描 `runs/<task-id>/` 下的 run 目录得到。

历史卡片需要的字段来自：

- `run.json`
- `result.json`

### 8.3 单次运行详情

单次运行详情只读取：

1. `run.json`
2. `requests.jsonl`
3. `result.json`

没有 fallback 文件，没有额外索引文件。

---

## 9. 不持久化的内容

以下内容不进入持久化层：

1. 任务列表缓存。
2. 历史列表索引。
3. active runs / recent runs 视图。
4. last run summary 之类的任务回填摘要。
5. 运行中的快照文件。
6. lifecycle 事件流。
7. Turbo 级别事件日志。
8. 完整性测试实时事件日志。
9. 报告产物清单。
10. schema 文件。
11. task meta、notes、tags、owner 这类当前非核心业务字段。

这些内容要么属于界面视图，要么属于运行时控制信息，要么可以从底层业务数据重新计算。

报告是导出物，不属于核心业务数据。默认按需生成，输出到用户指定路径，不纳入主存储目录。

---

## 10. Repository 建议

下一版 `internal/server/store` 可以收敛成最小集合：

```text
internal/server/store/
  fs.go                # 原子写、JSON/JSONL、目录工具
  config_repo.go       # config.json
  task_repo.go         # tasks/<task-id>.json
  run_repo.go          # run.json / result.json
  request_log.go       # requests.jsonl append/read
```

不再需要：

- view_repo
- projector
- artifact_repo
- rebuild
- history store
- task summary store

---

## 11. 代价与取舍

这个方案的代价很明确：

1. 任务列表和任务历史读取时需要扫描目录。
2. 运行中的跨进程恢复能力会变弱。
3. 某些 UI 页面首次读取会比预存视图慢。

但换来的好处更符合目标：

1. 存储模型极简。
2. 不再维护多份互相覆盖的摘要。
3. 标准模式、Turbo 模式、完整性测试共享底层事实模型。
4. 后续报告、对比、回放、内容检查都可以基于同一份请求事实扩展。

一句话总结：

> 下一版只存配置、任务、运行元数据、请求事实和最终结果；其他一律运行时聚合。
