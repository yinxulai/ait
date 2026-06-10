# 接口完整性测试功能设计

> 版本：v0.2 设计草案  
> 日期：2026-06-10

---

## 目录

1. [概述](#1-概述)
2. [设计目标与非目标](#2-设计目标与非目标)
3. [核心概念](#3-核心概念)
4. [断言机制设计](#4-断言机制设计)
5. [测试套件与测试用例模型](#5-测试套件与测试用例模型)
6. [任务配置设计](#6-任务配置设计)
7. [执行流程](#7-执行流程)
8. [运行事件与状态](#8-运行事件与状态)
9. [结果模型与存储](#9-结果模型与存储)
10. [TUI 页面设计](#10-tui-页面设计)
11. [MCP 输出设计](#11-mcp-输出设计)
12. [推荐实现位置](#12-推荐实现位置)
13. [内置测试套件规划](#13-内置测试套件规划)
14. [开发分期](#14-开发分期)

---

## 1. 概述

接口完整性测试是 AIT 除“标准性能测试”和“Turbo 承载能力测试”之外的第三种运行模式。

它的核心目标不是测吞吐极限，而是回答一个更基础的问题：

> 目标服务是否真的完整兼容所声明的接口协议，并且是否具备任务所要求的功能能力？

标准模式关注批量请求下的性能表现；Turbo 模式关注服务的最大稳定承载能力；接口完整性测试关注协议、结构、行为、错误格式和能力项是否符合预期。

典型检测内容包括：

- 协议路径是否可用。
- 请求参数是否被正确接受。
- 响应结构是否符合 OpenAI / Anthropic 兼容协议。
- 流式响应事件顺序是否稳定。
- 错误响应格式是否可解析。
- 是否支持 JSON 输出、工具调用、思考字段等能力。
- 是否返回 token usage、cached tokens、finish reason 等关键字段。
- 基础延迟、首包时间等是否超过最低可用阈值。

---

## 2. 设计目标与非目标

### 2.1 目标

1. **提供独立测试模式**

   接口完整性测试应作为独立运行模式存在，而不是标准模式的一个开关。它有自己的执行节奏、结果模型和页面展示。

2. **以测试套件组织能力验证**

   使用 Suite / Case / Assertion 组织测试：

   - Suite：面向一个协议或能力集合。
   - Case：一项具体接口能力验证。
   - Assertion：判断该用例是否通过的断言。

3. **支持自定义测试规则**

   自定义测试规则是接口完整性测试的核心能力。第一版必须允许用户通过规则文件补充或覆盖测试用例中的断言，用于验证服务端私有字段、兼容性差异和业务要求。

4. **内置声明式断言机制**

   完整性测试不引入脚本语言，而是使用 `path / op / value` 形式的声明式断言描述预期行为。自定义规则也使用同一套声明式断言模型。

5. **保留断言所需的服务端返回结构**

   第一版采集层应保留 HTTP 状态、响应头、响应体、错误体、流式事件与解析错误，满足协议结构检查、规则判断和失败排查需求。

6. **提供适合回归的结论**

   结果应直接告诉用户：哪些能力通过、哪些失败、哪些仅警告、哪些被跳过，以及失败原因是什么。

7. **适配 TUI 与 MCP**

   TUI 需要有完整性测试专属运行页和结果页；MCP 需要返回适合 AI 客户端判断兼容性的结构化摘要。

### 2.2 非目标

接口完整性测试当前阶段不做以下事情：

- 不做极限吞吐探测；极限承载属于 Turbo 模式。
- 不替代标准性能测试；性能指标只作为辅助信息。
- 不执行任意脚本，不引入脚本沙箱。
- 不支持自定义请求编排脚本；自定义能力首版限定为声明式规则文件。
- 不把外部规则文件全文复制进运行目录，只记录规则文件来源摘要和执行结果。
- 不保存运行中的页面视图或中间快照。
- 不做跨供应商协议自动转换；只检测目标服务是否符合任务声明的协议。

---

## 3. 核心概念

### 3.1 Run Mode

运行模式。

AIT 的任务运行模式统一由 `mode` 字段表达，不再使用布尔开关表示模式。

```json
{
  "mode": "standard | turbo | integrity"
}
```

接口完整性测试对应：

```json
{
  "mode": "integrity"
}
```

### 3.2 Suite

测试套件，一组面向同一协议或能力集合的测试用例。

示例：

- `openai-completions-smoke`
- `openai-completions-full`
- `openai-responses-smoke`
- `openai-responses-full`
- `anthropic-messages-smoke`
- `anthropic-messages-full`

Suite 负责说明：

- 适用协议。
- 覆盖能力。
- 包含哪些测试用例。
- 哪些用例是必需项，哪些是可选能力项。

### 3.3 Case / Test Case

测试用例，一项具体接口能力验证。

一个 Case 通常包含：

1. 请求定义。
2. 执行条件。
3. 断言列表。
4. 失败等级。
5. 重试和超时策略。

示例：

- `basic-response-shape`：基础响应结构检测。
- `stream-sse-format`：流式 SSE 格式检测。
- `error-response-shape`：错误响应结构检测。
- `token-usage-fields`：token usage 字段检测。
- `json-output-capability`：JSON 输出能力检测。

### 3.4 Assertion

断言，用于判断请求、响应、指标或聚合结果是否符合预期。

断言使用声明式表达式，例如：

```json
{
  "path": "response.status_code",
  "op": "between",
  "value": [200, 299]
}
```

### 3.5 Capability

能力项，用来描述服务支持的功能范围。

建议初始能力项：

| 能力项 | 含义 |
| ------ | ---- |
| `basic_request` | 基础请求响应 |
| `streaming` | 流式响应 |
| `error_shape` | 错误结构 |
| `usage` | token usage 字段 |
| `cache_usage` | cached tokens / cache hit 字段 |
| `json_output` | JSON 输出能力 |
| `tool_calling` | 工具调用能力 |
| `thinking` | 思考字段或 reasoning 字段 |
| `performance_baseline` | 基础可用性能阈值 |

---

## 4. 断言机制设计

接口完整性测试内置一套声明式断言机制，用于描述“如何判断一个测试用例是否通过”。

可以理解为：

```text
接口完整性测试
  └─ Suite 测试套件
      └─ Case 测试用例
          ├─ Request 请求定义
          └─ Assertions 断言列表
              └─ path / op / value 表达式求值
```

断言机制不是独立测试模式。它是完整性测试的判定层，也可以在未来被标准模式和 Turbo 模式复用：

| 模式 | 是否使用断言 | 断言作用 |
| ---- | ------------ | -------- |
| 标准模式 | 可选 | 对请求和聚合性能结果做额外校验 |
| Turbo 模式 | 可选 | 对每个并发级别和最终承载结果做阈值校验 |
| 接口完整性测试 | 默认使用 | 作为每个测试用例是否通过的核心判定依据 |

### 4.1 Assertion 模型

每条断言建议包含：

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `id` | string | 可选，稳定断言 ID |
| `name` | string | 可选，展示名称 |
| `phase` | string | 可选，执行阶段；缺省由所在 Case 推断 |
| `level` | string | 失败等级：`error` / `warn` |
| `path` | string | 观测字段路径 |
| `op` | string | 操作符 |
| `value` | any | 期望值 |
| `message` | string | 失败说明 |

示例：

```json
{
  "id": "responses.output.exists",
  "level": "error",
  "path": "response.body.output",
  "op": "exists",
  "message": "Responses 响应体必须包含 output 字段。"
}
```

### 4.2 断言阶段

断言按阶段执行，避免把所有校验都耦合在请求完成后。

| 阶段 | 触发时机 | 典型用途 |
| ---- | -------- | -------- |
| `preflight` | 请求发出前 | 检查任务配置、协议、模型、Prompt、参数组合 |
| `request` | 单个请求完成后 | 检查 HTTP 状态、响应字段、错误结构、TTFT、Token 字段 |
| `stream` | 流式响应解析过程中或完成后 | 检查 SSE 事件顺序、首包、done 标记、增量字段 |
| `integrity_case` | 单个完整性测试用例结束后 | 判断该用例是否整体通过 |
| `aggregate` | 全部用例完成后 | 检查整体成功率、能力覆盖、基础延迟 |

### 4.3 支持的操作符

首版仅支持声明式表达式，不支持脚本。

| 操作符 | 含义 | 示例 |
| ------ | ---- | ---- |
| `exists` | 字段存在 | `response.body.id exists` |
| `not_exists` | 字段不存在 | `response.error not_exists` |
| `eq` | 等于 | `response.status_code eq 200` |
| `neq` | 不等于 | `error_message neq ""` |
| `in` | 在集合中 | `finish_reason in ["stop", "length"]` |
| `contains` | 字符串或数组包含 | `response.text contains "OK"` |
| `matches` | 正则匹配 | `response.text matches "^\\{.*\\}$"` |
| `gt` / `gte` | 大于 / 大于等于 | `metrics.success_rate gte 0.99` |
| `lt` / `lte` | 小于 / 小于等于 | `metrics.ttft.avg_ms lte 1000` |
| `between` | 区间 | `response.status_code between [200, 299]` |

### 4.4 Path 语法

`path` 采用受限 JSON 路径语法，首版只支持确定性访问，不支持脚本和任意表达式。

| 语法 | 示例 | 说明 |
| ---- | ---- | ---- |
| 点路径 | `response.body.id` | 访问对象字段 |
| 数组索引 | `response.body.output[0]` | 访问数组元素 |
| 嵌套路径 | `response.body.output[0].content[0].text` | 对象和数组组合 |

语义约束：

- 字段不存在与字段存在但值为 `null` 必须区分。
- `exists` 表示路径可解析到任意值，包括 `null`。
- `not_exists` 表示路径无法解析。
- `response.body` 是解析后的 JSON 对象；无法解析时记录 `response.parse_error`。
- 首版不支持通配符、过滤器和自定义函数。

### 4.5 可观测字段模型

断言不直接访问内部 Go 结构，而是访问稳定的观测字段模型。

观测字段模型应分为两层：

1. **响应事实层**：尽可能完整保留服务端返回内容，用于协议兼容性检查、错误排查和后续内容检查。
2. **派生字段层**：从响应事实中提取标准化指标和摘要字段，用于常见断言和页面展示。

响应事实层不应因为当前断言暂时用不到某些字段就提前丢弃信息。协议适配器可以生成 `response.text`、`metrics.ttft_ms` 等派生字段，但不能替代响应事实。

#### preflight 字段

- `task.protocol`
- `task.model`
- `task.endpoint`
- `task.stream`
- `task.concurrency`
- `task.count`
- `task.prompt.mode`
- `task.prompt.length`

#### request 字段

响应字段：

- `request.index`
- `request.method`
- `request.url`
- `request.headers`
- `request.body`
- `response.status_code`
- `response.headers`
- `response.body`
- `response.error_body`
- `response.parse_error`

派生字段：

- `case.id`
- `case.category`
- `case.capability`
- `response.text`
- `response.error`
- `metrics.total_ms`
- `metrics.ttft_ms`
- `metrics.tps`
- `metrics.input_tokens`
- `metrics.output_tokens`
- `metrics.cached_tokens`
- `network.dns_ms`
- `network.connect_ms`
- `network.tls_ms`
- `network.target_ip`

#### stream 字段

流式响应字段：

- `stream.events`
- `stream.parse_errors`
- `stream.done_event_seen`
- `stream.error_event`

派生字段：

- `stream.first_event_ms`
- `stream.event_count`
- `stream.text_delta_count`
- `stream.accumulated_text`

#### integrity_case 字段

- `case.id`
- `case.required`
- `case.assertions.total`
- `case.assertions.passed`
- `case.assertions.failed`
- `case.assertions.warned`
- `case.duration_ms`
- `case.status`

#### aggregate 字段

- `integrity.total_cases`
- `integrity.passed_cases`
- `integrity.failed_cases`
- `integrity.warned_cases`
- `integrity.skipped_cases`
- `integrity.required_failed_cases`
- `integrity.capabilities.total`
- `integrity.capabilities.passed`
- `metrics.total_time.avg_ms`
- `metrics.ttft.avg_ms`
- `metrics.ttft.p95_ms`

### 4.6 自定义测试规则

自定义测试规则是第一版必须保留的核心能力，但首版只支持声明式规则文件，不支持脚本。

规则文件用于向内置 Suite 追加或覆盖断言。它不负责生成新的请求流程，也不负责定义复杂执行编排。

规则来源按优先级从低到高合并：

1. 内置 Suite 中的默认断言。
2. 任务配置中显式附加的规则文件。

同一个 `assertion id` 多次出现时，规则文件中的断言覆盖内置断言，并在结果中记录来源摘要。

规则文件示例：

```json
{
  "version": "ait.integrity.rules/v1",
  "suite": "openai-responses-smoke",
  "assertions": [
    {
      "id": "custom.response.service_tier",
      "case_id": "basic-response-shape",
      "level": "error",
      "path": "response.body.service_tier",
      "op": "exists",
      "message": "响应体必须包含 service_tier 字段。"
    }
  ]
}
```

---

## 5. 测试套件与测试用例模型

### 5.1 Suite 模型

建议字段：

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `version` | string | 套件格式版本 |
| `id` | string | 稳定套件 ID |
| `name` | string | 展示名称 |
| `description` | string | 套件说明 |
| `protocols` | string[] | 适用协议 |
| `capabilities` | string[] | 覆盖能力项 |
| `cases` | Case[] | 测试用例列表 |

### 5.2 Case 模型

建议字段：

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `id` | string | 稳定用例 ID |
| `name` | string | 展示名称 |
| `description` | string | 用例说明 |
| `category` | string | 用例分类 |
| `capability` | string | 对应能力项 |
| `required` | bool | 是否为必需用例 |
| `request` | object | 请求模板 |
| `assertions` | Assertion[] | 断言列表 |
| `timeout_ms` | number | 单用例超时 |
| `retry` | number | 失败重试次数 |
| `skip_when` | Assertion[] | 跳过条件，复用断言表达式；全部命中时跳过 |

### 5.3 Category 建议值

| 分类 | 用途 |
| ---- | ---- |
| `protocol` | 协议基础结构 |
| `stream` | 流式响应行为 |
| `error` | 错误响应格式 |
| `capability` | 功能能力验证 |
| `usage` | token / cache usage 字段 |
| `performance_baseline` | 基础可用性能阈值 |

### 5.4 Suite 示例

```json
{
  "version": "ait.integrity/v1",
  "id": "openai-responses-smoke",
  "name": "OpenAI Responses Smoke 完整性测试",
  "description": "验证目标服务是否具备 OpenAI Responses 协议的基础兼容性。",
  "protocols": ["openai-responses"],
  "capabilities": [
    "basic_request",
    "streaming",
    "usage",
    "error_shape"
  ],
  "cases": [
    {
      "id": "basic-response-shape",
      "name": "基础响应结构",
      "category": "protocol",
      "capability": "basic_request",
      "required": true,
      "request": {
        "stream": false,
        "prompt": "Reply with a short greeting."
      },
      "assertions": [
        {
          "path": "response.status_code",
          "op": "between",
          "value": [200, 299],
          "message": "HTTP 状态码必须为 2xx。"
        },
        {
          "path": "response.body.id",
          "op": "exists",
          "message": "响应体必须包含 id 字段。"
        },
        {
          "path": "response.body.output",
          "op": "exists",
          "message": "Responses 响应体必须包含 output 字段。"
        }
      ]
    },
    {
      "id": "stream-sse-format",
      "name": "流式 SSE 格式",
      "category": "stream",
      "capability": "streaming",
      "required": true,
      "request": {
        "stream": true,
        "prompt": "Count from 1 to 3."
      },
      "assertions": [
        {
          "path": "stream.first_event_ms",
          "op": "lte",
          "value": 3000,
          "message": "流式响应首事件应在 3 秒内返回。"
        },
        {
          "path": "stream.done_event_seen",
          "op": "eq",
          "value": true,
          "message": "流式响应必须包含结束事件。"
        }
      ]
    },
    {
      "id": "token-usage-fields",
      "name": "Token Usage 字段",
      "category": "usage",
      "capability": "usage",
      "required": false,
      "request": {
        "stream": false,
        "prompt": "Return one sentence."
      },
      "assertions": [
        {
          "path": "response.body.usage.input_tokens",
          "op": "exists",
          "message": "响应应包含 input_tokens。"
        },
        {
          "path": "response.body.usage.output_tokens",
          "op": "exists",
          "message": "响应应包含 output_tokens。"
        }
      ]
    }
  ]
}
```

---

## 6. 任务配置设计

任务配置统一使用 `mode` 表达运行模式。接口完整性测试使用 `mode = integrity`，并增加 `integrity` 区块。

示例：

```json
{
  "mode": "integrity",
  "protocol": "openai-responses",
  "endpoint": "https://api.example.com/v1/responses",
  "model": "example-model",
  "integrity": {
    "suite": "openai-responses-smoke",
    "fail_fast": false,
    "case_timeout_ms": 30000,
    "rule_files": [
      "~/.ait/rules/openai-responses-extra.json"
    ]
  }
}
```

字段说明：

| 字段 | 类型 | 说明 |
| ---- | ---- | ---- |
| `mode` | string | 运行模式：`standard` / `turbo` / `integrity` |
| `integrity.suite` | string | 要执行的测试套件 ID；第一版建议一次只运行一个 suite |
| `integrity.fail_fast` | bool | 必需用例失败后是否立即停止 |
| `integrity.case_timeout_ms` | number | 单用例默认超时 |
| `integrity.rule_files` | string[] | 自定义测试规则文件，用于补充或覆盖内置断言 |

`integrity` 区块同时决定执行哪个 suite，以及是否加载自定义测试规则。

---

## 7. 执行流程

```text
StartRun(taskID)
  ├─ 加载任务定义
  ├─ 识别 mode = integrity
  ├─ 加载 integrity 配置
  ├─ 加载内置 Suite
  ├─ 加载自定义测试规则文件
  ├─ 合并并编译断言
  ├─ 执行基础配置校验
  ├─ for each Case:
  │    ├─ 发布 IntegrityCaseStarted
  │    ├─ 基于任务配置和 Case request 生成实际请求
  │    ├─ 调用协议客户端发送请求
  │    ├─ 保留 HTTP 状态、响应头、响应体、错误体、流式事件和解析错误
  │    ├─ 从响应中派生文本、usage、TTFT、状态摘要等字段
  │    ├─ 执行 request / stream / integrity_case 断言
  │    ├─ 追加请求事实和断言结果到 requests.jsonl
  │    ├─ 生成 CaseResult
  │    ├─ 发布 AssertionResult / IntegrityCaseDone
  │    └─ required 失败且 fail_fast=true → 中止后续 Case
  ├─ 聚合 SuiteResult
  ├─ 写入 result.json
  ├─ 渲染报告
  └─ 发布 IntegrityRunComplete
```

### 7.1 用例执行规则

第一版优先保证执行模型简单：

1. 协议不匹配则运行前失败。
2. Suite 中的 Case 默认全部执行。
3. 自定义规则只追加或覆盖断言，不改变 Case 执行顺序。
4. 规则文件加载或编译失败：视为运行前失败，不发起请求。

### 7.2 fail_fast 规则

建议规则：

- Case 失败，且 `fail_fast = true`：立即停止后续 Case。
- Case 失败，且 `fail_fast = false`：继续执行，但最终状态为 `failed`。
- 规则文件加载或编译失败：视为运行前失败，不发起请求。

---

## 8. 运行事件与状态

### 8.1 新增事件类型

建议扩展 Server 事件：

```go
const (
    EventIntegrityCaseStarted EventKind = "IntegrityCaseStarted"
    EventIntegrityCaseDone    EventKind = "IntegrityCaseDone"
    EventAssertionResult      EventKind = "AssertionResult"
    EventIntegrityRunComplete EventKind = "IntegrityRunComplete"
)
```

### 8.2 运行态字段

`RunState` 建议增加完整性测试相关快照：

```go
type IntegrityRunSnapshot struct {
    SuiteID       string
    SuiteName     string
    TotalCases    int
    CurrentIndex  int
    CurrentCaseID string
    Passed        int
    Failed        int
    Warned        int
    Skipped       int
    Status        IntegrityStatus
    Failures      []IntegrityFailureSummary
}
```

运行态只存在内存中，用于 TUI/MCP 实时展示；最终落盘以 `requests.jsonl` 和 `result.json` 为准。

实时事件仅用于 UI 进度展示。事件总线允许为避免阻塞而丢弃实时事件，因此事件不能作为最终事实源。

---

## 9. 结果模型与存储

完整性测试遵循“请求级事实源 + 最终结论”的存储分工：

- `requests.jsonl`：保存每个 Case 的请求、响应、解析结果、派生指标和断言结果，是后续内容检查与离线排查的事实源。
- `result.json`：保存本次运行的最终完整性结论、能力覆盖、Case 状态和失败摘要。

第一版只落地这两个核心文件，先保证测试执行、断言判定和结果回溯闭环。

### 9.1 requests.jsonl 记录

每个 Case 至少追加一条请求级事实记录。

```json
{
  "kind": "integrity_case",
  "case_id": "basic-response-shape",
  "status": "passed",
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
    "usage": {},
    "metrics": {
      "total_ms": 820,
      "ttft_ms": 0
    }
  },
  "assertions": []
}
```

### 9.2 流式事件格式

流式事件直接记录在 `requests.jsonl` 的 `response.stream_events` 中。

```json
{
  "seq": 1,
  "time_ms": 120,
  "event": "response.output_text.delta",
  "parsed": {},
  "parse_error": null
}
```

### 9.3 result.json 结构

完整性测试结果写入单次运行的 `result.json`，建议新增 `integrity` 区块。

```json
{
  "integrity": {
    "status": "failed",
    "suite": {
      "id": "openai-responses-smoke",
      "version": "ait.integrity/v1",
      "name": "OpenAI Responses Smoke 完整性测试"
    },
    "summary": {
      "total": 12,
      "passed": 9,
      "failed": 2,
      "warned": 1,
      "skipped": 0
    },
    "capabilities": [
      {
        "name": "basic_request",
        "status": "passed",
        "passed": 2,
        "failed": 0,
        "warned": 0,
        "skipped": 0
      }
    ],
    "cases": [
      {
        "id": "basic-response-shape",
        "status": "passed",
        "required": true,
        "capability": "basic_request",
        "request_ref": "requests.jsonl#basic-response-shape",
        "assertions": {
          "passed": 3,
          "failed": 0,
          "warned": 0,
          "skipped": 0
        }
      }
    ],
    "failures": [
      {
        "case_id": "stream-sse-format",
        "assertion_id": "stream.done_event_seen",
        "message": "流式响应必须包含结束事件。",
        "request_ref": "requests.jsonl#stream-sse-format"
      }
    ]
  }
}
```

### 9.4 状态枚举

建议完整性运行总状态：

| 状态 | 含义 |
| ---- | ---- |
| `passed` | 必需用例和可选用例均通过 |
| `passed_with_warnings` | 必需用例通过，但存在警告或可选能力失败 |
| `failed` | 至少一个必需用例失败 |
| `skipped` | 没有可执行用例或全部被跳过 |
| `error` | 套件加载、断言编译、运行环境错误 |

Case 状态由断言结果、必需性和执行错误共同决定：

| 条件 | Case 状态 | Suite 状态影响 |
| ---- | --------- | -------------- |
| required Case 中 `error` 断言失败 | `failed` | 整体 `failed` |
| optional Case 中 `error` 断言失败 | `warning` | 整体最多 `passed_with_warnings` |
| 任意 Case 中 `warn` 断言失败 | `warned` | 整体最多 `passed_with_warnings` |
| Case 超时 | `failed` 或 `warning` | 取决于 `required` |
| Case 被跳过 | `skipped` | 不导致失败，但计入覆盖统计 |

### 9.5 存储原则

遵循 [存储设计](storage.md) 的原则：

- 只保存最终业务结果。
- 不保存运行中的 UI 快照。
- 不保存可从请求明细和结果重算的查询视图。
- 不复制外部规则文件全文，只保存规则文件路径、版本、来源摘要和执行结果。
- `requests.jsonl` 是请求级事实源，保存请求、响应、解析结果、派生字段和断言结果。
- `result.json` 是最终结论文件，保存 Suite / Capability / Case 的最终状态、计数和失败摘要。
- 第一版优先保证结构完整和实现简单，只实现完整性测试闭环必需的数据。
- 对服务端返回结构采用“响应事实优先、派生补充”的保存策略：HTTP 状态、响应头、响应体、错误体、流式事件应尽可能保留；标准化文本、usage、指标和摘要字段作为派生结果保存。

---

## 10. TUI 页面设计

完整性测试需要单独页面，不应直接复用标准性能测试仪表盘作为主视图。

### 10.1 运行中页面

主线是 Case 进度。

```text
╔══ AIT  接口完整性测试 ─ openai-responses-smoke ─══════════╗
║  协议 openai-responses   模型 example-model                ║
║  进度  5/12   ✓ 4   ✗ 1   ! 0   - 0   当前 stream-sse-format ║
╠══════════════════════════════════════════════════════════════╣
║  状态  分类       用例                         耗时   摘要    ║
║  ✓     protocol   basic-response-shape        820ms  通过    ║
║  ✗     stream     stream-sse-format           3.2s   未见 done║
║  …     usage      token-usage-fields          --     待执行  ║
╠══════════════════════════════════════════════════════════════╣
║  [Enter] 查看用例详情  [s] 停止  [b] 后台  [q] 退出          ║
╚══════════════════════════════════════════════════════════════╝
```

### 10.2 用例详情页

```text
╔══ 用例详情 ─ stream-sse-format ─────────────────────────══╗
║  分类 stream   能力 streaming   必需 是   状态 失败          ║
╠══════════════════════════════════════════════════════════════╣
║  请求摘要                                                   ║
║  stream=true   prompt="Count from 1 to 3."                  ║
╠══════════════════════════════════════════════════════════════╣
║  断言结果                                                   ║
║  ✓ response.status_code between [200,299]                   ║
║  ✗ stream.done_event_seen eq true                           ║
║    期望 true，实际 false                                    ║
╠══════════════════════════════════════════════════════════════╣
║  [r] 查看请求详情  [b] 返回                                  ║
╚══════════════════════════════════════════════════════════════╝
```

### 10.3 任务详情结果页

完整性测试完成后，任务详情页应以完整性结论为主。

```text
╔══ AIT  任务详情 ─ responses-integrity-smoke ───────────════╗
║  最近运行  接口完整性测试  ✗ FAILED                         ║
╠══════════════════════════════════════════════════════════════╣
║  结论                                                        ║
║  必需能力失败：2   可选能力警告：1   通过：9/12              ║
╠══════════════════════════════════════════════════════════════╣
║  能力覆盖                                                    ║
║  basic_request       ✓ 2/2                                   ║
║  streaming           ✗ 1/2                                   ║
║  usage               ! 1/2                                   ║
║  error_shape         ✓ 1/1                                   ║
╠══════════════════════════════════════════════════════════════╣
║  失败用例                                                    ║
║  ✗ stream-sse-format      未观察到 done 事件                 ║
║  ✗ error-response-shape   error.message 字段缺失             ║
╠══════════════════════════════════════════════════════════════╣
║  [Enter] 查看失败详情  [j/k] 移动  [e] 编辑  [r] 再次运行     ║
╚══════════════════════════════════════════════════════════════╝
```

辅助性能指标可以在结果页底部折叠展示，例如总耗时、平均 TTFT、状态码分布，但不作为主结果。

---

## 11. MCP 输出设计

MCP 工具返回任务状态时，应包含完整性摘要，便于 AI 客户端判断目标服务是否兼容。

示例：

```json
{
  "task_id": "task_01",
  "run_id": "run_01",
  "mode": "integrity",
  "status": "completed",
  "integrity": {
    "status": "failed",
    "suite_id": "openai-responses-smoke",
    "summary": {
      "total": 12,
      "passed": 9,
      "failed": 2,
      "warned": 1,
      "skipped": 0
    },
    "failures": [
      {
        "case_id": "stream-sse-format",
        "message": "未观察到 done 事件。"
      }
    ]
  }
}
```

后续可新增 MCP 工具：

| 工具 | 描述 |
| ---- | ---- |
| `ait.list_integrity_suites` | 列出可用完整性测试套件 |
| `ait.preview_integrity_suite` | 预览某个测试套件包含的用例 |
| `ait.validate_rule_file` | 校验自定义测试规则文件格式与表达式 |
| `ait.get_integrity_case_detail` | 获取单个用例详情和断言结果 |

MCP 默认状态输出只返回摘要和失败原因；用例详情按需查询。

---

## 12. 推荐实现位置

```text
internal/server/
  integrity/
    engine.go        # 完整性测试执行引擎
    suite.go         # Suite 加载与过滤
    case.go          # Case 定义与执行编排
    result.go        # IntegrityResult / CaseResult
    builtin.go       # 内置测试套件注册

  assertion/
    loader.go        # 断言加载
    compiler.go      # 断言编译
    evaluator.go     # 断言求值
    fields.go        # 观测字段模型映射

  report/
    integrity_json.go
    integrity_csv.go
```

TUI 侧建议新增：

```text
internal/tui/pages/
  integritydash.go       # 完整性测试运行页
  integrityresult.go     # 完整性测试结果区块
  integritycase.go       # 用例详情页
```

Server 层负责完整性测试编排，TUI/MCP 只依赖 `server.Server` 接口，不直接访问 `internal/server/integrity` 或 `internal/server/assertion`。

---

## 13. 内置测试套件规划

内置套件按协议和覆盖深度命名：

| Suite ID | 覆盖目标 |
| -------- | -------- |
| `openai-completions-smoke` | Chat Completions 基础请求、基础响应结构、错误响应结构 |
| `openai-completions-full` | 在 smoke 基础上增加流式 chunk、finish reason、usage、边界错误格式 |
| `openai-responses-smoke` | Responses 基础请求、output 数组、text 输出提取、错误响应结构 |
| `openai-responses-full` | 在 smoke 基础上增加流式事件顺序、usage、cached tokens、复杂 content item |
| `anthropic-messages-smoke` | Messages 基础请求、content block、stop_reason、错误响应结构 |
| `anthropic-messages-full` | 在 smoke 基础上增加 streaming event、usage、tool use、边界错误格式 |

后续可加入能力型附加套件：

- JSON 输出能力套件。
- 工具调用能力套件。
- Thinking / Reasoning 字段套件。
- Cache usage 套件。

---

## 14. 开发分期

### Phase 1：类型与配置模型

- 统一任务运行模式为 `mode = standard | turbo | integrity`。
- 定义 Suite / Case / Assertion / Result 类型。
- 定义请求事实记录和 result 结论模型。
- 支持 `preflight`、`request`、`stream`、`integrity_case`、`aggregate` 等断言阶段。

### Phase 2：响应采集与请求事实存储

- 保留 HTTP 状态、headers、body、error body、parse error、stream events。
- 从响应中派生 text、usage、TTFT、状态摘要等字段。
- 写入 `requests.jsonl`。
- 明确实时事件不可作为事实源。

### Phase 3：断言与自定义规则引擎

- 实现内置断言与自定义规则文件的加载、合并、编译和离线校验。
- 实现受限 path 解析、操作符求值和 missing/null 区分。
- 增加 `ValidateRuleFile` / `PreviewIntegritySuite` 能力。

### Phase 4：完整性执行引擎

- 新增 `internal/server/integrity/engine.go`。
- 接入 `StartRun` 模式分发。
- 执行 Suite / Case、生成派生字段、执行断言、聚合状态。
- 写入 `result.json`。

### Phase 5：TUI / MCP / 报告

- 新增完整性运行页和用例详情页。
- 任务详情页支持完整性结果区块。
- MCP 状态输出加入完整性摘要。
- MCP 支持规则文件校验能力。
- JSON / CSV 报告输出完整性结果。

### Phase 6：内置 Suite 完善

- 完成 OpenAI / Anthropic smoke 套件。
- 完成 OpenAI / Anthropic full 套件。
- 增加可选能力套件。
