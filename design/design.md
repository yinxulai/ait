# AIT 功能设计与架构文档

> 版本：v2.0 设计草案  
> 日期：2026-05-16

---

## 目录

1. [概述](#1-概述)
2. [现有架构分析](#2-现有架构分析)
3. [新架构设计](#3-新架构设计)
4. [交互式 TUI 设计](#4-交互式-tui-设计)
5. [Turbo 模式设计](#5-turbo-模式设计)
6. [接口完整性测试模式设计](#6-接口完整性测试模式设计)
7. [新增数据结构与接口](#7-新增数据结构与接口)
8. [开发计划](#8-开发计划)

相关设计：

- [统一运行存储设计](storage.md)
- [Turbo 模式功能设计](turbo.md)
- [接口完整性测试功能设计](integrity.md)

---

## 1. 概述

本文档描述 AIT 工具 v2.0 的四个核心功能迭代：

### 1.1 交互式 TUI

将现有"执行完即退出"的单次命令行模式升级为**全屏交互式终端界面**，并以任务管理作为主入口：

- `ait` 不带参数直接进入 TUI 任务中心；带完整参数时自动创建任务并立即启动，直接进入运行仪表盘（v2.0 不再提供独立的 `tpg` 子命令）
- 首页展示任务列表，可直接选择已有任务再次执行，运行中任务用 `◉` 实时标注进度
- 无需记忆所有参数，通过弹窗向导创建或编辑任务
- 任务详情页集中展示配置摘要和最近运行记录
- 运行仪表盘：任务配置概览、实时指标汇总、进度、请求列表（每行一条请求含完整指标）
- 请求列表中可选中单条，按 `[Enter]` 进入请求详情页，展示原始输入/输出及全量性能数据
- 支持多个任务同时在后台运行；任务间（尤其是网络指标）可能存在干扰，启动时给出提示
- 仪表盘按 `[b]` 后台运行并返回任务列表；运行结束后自动跳转回任务详情页
- 任务详情页兼做结果页：最近一次运行指标高亮展开，支持生成报告和复制摘要

### 1.2 任务管理

新增“任务”作为一等对象，用于保存和复用测试配置：

- 每个任务只绑定一个模型，保存协议、完整接口地址、模型、Prompt、测试模式和对应模式参数
- 测试模式统一为 `standard`、`turbo`、`integrity`
- 协议值细化为 `openai-completions`、`openai-responses`、`anthropic-messages`
- 首页以任务列表形式呈现，支持新建、编辑、删除、复制和直接运行
- 任务详情页展示配置摘要和最近运行记录
- 多模型回归通过多个任务组织，而不是在单个任务内批量执行
- 支持同时运行多个任务；系统会提示任务间存在潜在干扰，尤其是 DNS/TCP/TLS 等网络指标
- 每次运行都会沉淀为任务记录，便于后续直接选择再次测试或对比回归结果

### 1.3 Turbo 模式

一种新的测试模式，用于**探测服务的最大稳定承载能力**：

- 从初始并发数出发，按步进值逐级提升并发
- 每个并发级别执行固定数量请求，采集该级别的完整指标
- 当成功率或延迟超过阈值时判定服务出现降级，自动停止
- 输出"并发爬坡曲线"报告，直观展示吞吐量、延迟与缓存命中率随并发变化的趋势

### 1.4 接口完整性测试

第三种测试模式，用于**验证上游服务接口是否符合目标协议和业务要求**：

- 与标准模式、Turbo 模式并列，统一使用 `mode = "integrity"`
- 通过内置 Suite 执行协议级 Case，验证响应结构、错误格式、流式行为、usage、关键字段和能力项
- 每个 Case 通过声明式 Assertion 给出通过、警告或失败结论，不以性能指标作为主要目标
- 自定义测试规则是第一版核心能力：用户可通过 `integrity.rule_files` 追加或覆盖内置断言
- 首版自定义规则只支持声明式规则文件，不支持脚本和自定义请求编排
- 底层运行记录必须尽可能保留服务端返回结构，便于后续继续扩展内容检查能力

---

## 2. 现有架构分析

### 2.1 模块职责

```
cmd/ait/ait.go          ← 入口：flag 解析、参数校验、编排多模型执行
internal/
  display/display.go   ← 输出层：欢迎信息、进度条、结果表格
  runner/runner.go     ← 执行层：并发请求、指标收集、进度回调
  client/              ← 协议层：OpenAI / Anthropic HTTP 客户端
  types/types.go       ← 共享类型：Input、StatsData、ReportData
  report/              ← 报告层：JSON / CSV 渲染
  prompt/              ← 输入层：字符串/文件/长度生成
  config/              ← 空（待实现）
  network/             ← IP 工具
  upload/              ← 匿名数据上传
  logger/              ← 请求日志记录
```

### 2.2 当前执行链

```
main()
 ├─ flag 解析 + 参数校验
 ├─ display.ShowWelcome()
 ├─ display.ShowInput()
 ├─ for each model:
 │    runner.RunWithProgress(progressCallback)
 │      └─ progressCallback → display.UpdateProgress()
 ├─ display.FinishProgress()
 ├─ display.ShowErrorsReport()
 ├─ display.ShowSignalReport() 或 ShowMultiReport()
 └─ report.GenerateReports()
```

### 2.3 现有问题

| 问题 | 影响 |
|------|------|
| `progressCallback` 仅能更新一个全局进度条，无法展示实时指标 | 测试中只能看到"完成了多少个"，看不到当前 TPS/TTFT 等关键指标 |
| 多模型串行执行，前一个模型结束后进度条重置 | 体验割裂 |
| 没有中途中断的能力 | 一旦启动只能 Ctrl+C 强杀 |
| 并发数固定，无法动态探测极限 | 需要手动二分多次执行 |
| 协议抽象过粗，只区分 `openai` / `anthropic` | 无法准确对应 OpenAI Completions、OpenAI Responses、Anthropic Messages 的差异 |
| 测试配置无法保存为任务 | 每次测试都要重复输入参数，难以形成稳定回归基线 |
| 结果展示完即退出，无法二次查阅 | 对比分析需要回滚终端 |

---

## 3. 新架构设计

### 3.1 SC 分层概览

整体采用 **Server-Client** 架构：所有业务能力集中在 Server 层，TUI 和未来的 Web UI 作为 Client 通过统一接口调用。当前是同进程函数调用（无网络开销），但清晰的接口边界使未来接入 Web UI 时无需改动任何业务逻辑。

```
┌──────────────────────────────────────────────────────────────┐
│  cmd/ait/main.go  ─  入口                                    │
│   解析 flag → 创建 Server → 启动 TUI Client                  │
└──────────────────────┬───────────────────────────────────────┘
                       │
   ╔═══════════════════▼════════════════════════════════════╗
   ║          internal/server/   ─  SERVICE LAYER           ║
   ║                                                         ║
   ║  对外暴露 Server 接口（任务管理 / 运行管理 / 事件订阅）  ║
   ║  编排下层执行/持久化/报告模块，不感知任何 UI 细节       ║
   ╚═══════╤══════════════════╤══════════════════════════════╝
           │ uses             │ uses
  ┌────────▼────────┐  ┌──────▼───────────┐
  │  执行层          │  │  持久化 & 工具层   │
  │  runner/        │  │  task/           │
  │  turbo/         │  │  config/         │
  │  integrity/     │  │  report/         │
  │  assertion/     │  │  logger/ ...     │
  │  client/        │  │  prompt/         │
  └─────────────────┘  └──────────────────┘

   CLIENT LAYER（调用 Server 接口，不直接依赖下层）：
   ┌─────────────────────────────────────────────────────┐
   │  internal/tui/   ─  TUI Client（当前）               │
   │  BubbleTea 状态机 + 页面渲染                         │
   └─────────────────────────────────────────────────────┘
   ┌─────────────────────────────────────────────────────┐
   │  internal/webui/  ─  Web UI Bridge（未来）           │
   │  HTTP / WebSocket 桥接，前端通过浏览器访问            │
   └─────────────────────────────────────────────────────┘
```

### 3.2 Server 接口定义

Server 层对外暴露一个统一的 Go 接口，TUI 和 Web UI 只依赖该接口，不直接依赖任何下层包：

```go
// internal/server/server.go

// Server 是 AIT 的核心服务接口。
// TUI、Web UI 等所有前端仅依赖此接口，不直接依赖 runner/task 等底层包。
type Server interface {
    // 任务管理
    ListTasks() ([]types.TaskOverview, error)
    GetTask(id string) (types.TaskDefinition, error)
    CreateTask(cfg TaskConfig) (types.TaskDefinition, error)
    UpdateTask(id string, cfg TaskConfig) (types.TaskDefinition, error)
    DeleteTask(id string) error
    DuplicateTask(id string) (types.TaskDefinition, error)

    // 运行管理
    StartRun(taskID string) (RunID, error)
    StopRun(runID RunID) error
    GetRunState(runID RunID) (*RunState, bool)
    SubscribeRunEvents(runID RunID) (<-chan Event, CancelFunc)
    ListTaskRunHistory(taskID string, limit int) ([]types.TaskRunSummary, error)
    GenerateRunReport(runID RunID, format ReportFormat) (string, error)

    // 完整性测试辅助能力
    ListIntegritySuites() ([]types.IntegritySuiteOverview, error)
    PreviewIntegritySuite(suiteID string, ruleFiles []string) (*types.IntegritySuitePreview, error)
    ValidateRuleFile(path string) (*types.RuleFileValidationResult, error)

    // 应用配置
    GetAppConfig() (*config.Config, error)
    UpdateProxyURL(proxyURL string) error
}

// Event 是 Server 推送给订阅者的运行事件
type Event struct {
    RunID   RunID
    Kind    EventKind  // RequestDone | ProgressTick | LevelDone | IntegrityCaseDone | AssertionResult | RunComplete | RunFailed
    Payload any        // *RequestMetrics | *ProgressSnapshot | *LevelResult | *IntegrityCaseResult | *AssertionResult | *RunResult
}

type CancelFunc func()
```

### 3.3 各层职责边界

| 层 | 包 | 职责 |
|----|-----|------|
| **入口层** | `cmd/ait` | flag 解析；创建 Server；按模式启动 TUI / MCP Client |
| **Server 层** | `internal/server` | 暴露业务 API；编排本层子模块；管理运行状态；分发 Event |
| **执行子模块** | `internal/server/runner` `internal/server/turbo` `internal/server/integrity` | 标准压测、并发爬坡、接口完整性测试执行；回调推送指标或 Case 结果；**不感知 UI** |
| **断言子模块** | `internal/server/assertion` | 声明式规则文件加载、合并、编译、path 解析和断言求值；不发起请求 |
| **协议子模块** | `internal/server/client` | OpenAI / Anthropic HTTP 客户端；尽可能保留服务端返回结构供完整性测试使用；不感知上层 |
| **持久化子模块** | `internal/server/store` `internal/server/config` | `store` 是下一版唯一持久化实现；`config` 仅负责应用目录与路径解析 |
| **渲染子模块** | `internal/server/report` | JSON / CSV / Turbo / Integrity 报告渲染；纯函数，无副作用 |
| **工具子模块** | `internal/server/prompt` `internal/server/network` `internal/server/logger` `internal/server/upload` | Server 内部公共工具，无 UI 依赖 |
| **TUI Client** | `internal/tui` | BubbleTea 状态机；**只依赖 server.Server 接口**；渲染终端 UI |
| **MCP Client** | `internal/mcp` | MCP 协议适配层；**只依赖 server.Server 接口**；不直接访问 Server 子模块 |
| **Web Client** _(Future)_ | `internal/webui` | HTTP/WS 桥接；**只依赖 server.Server 接口**；提供 Web API |

> 存储设计请优先参考 [统一运行存储设计](storage.md)。该文档描述目标存储架构，不以当前实现与兼容层为约束。目标源码目录采用“父模块聚合”原则：属于 `server` 的执行、协议、存储、报告、工具等子模块都放在 `internal/server/` 下；其他父模块也按同样方式收纳自己的子模块，避免 `internal/` 顶层平铺过多业务细节。

### 3.4 目录结构

```
cmd/
  ait/
    main.go                 ← 入口：创建 server.New()；按 flag 启动 TUI / MCP

internal/
  server/                   ← SERVICE LAYER（业务父模块）
    server.go               ← Server 接口 + 构造函数
    task_service.go         ← 任务管理 facade 方法实现
    run_service.go          ← 运行管理 facade 方法实现
    event_bus.go            ← 内部事件总线实现
    types.go                ← RunID / RunState / Event / ReportFormat 等 Server 层类型

    runner/                 ← Server 子模块：标准压测执行
      runner.go             ← 并发请求执行（RunWithCallback / Stop）

    turbo/                  ← Server 子模块：Turbo 并发爬坡
      engine.go             ← 并发爬坡调度（Run / Stop）
      strategy.go           ← 步进 & 终止策略
      types.go              ← TurboResult / LevelResult

    integrity/              ← Server 子模块：接口完整性测试
      engine.go             ← Suite / Case 执行调度（Run / Stop）
      suite.go              ← Suite 定义与内置 Suite 注册
      case.go               ← Case 请求构造与执行
      result.go             ← IntegrityResult / CaseResult 聚合
      builtin.go            ← 内置 smoke / full Suite

    assertion/              ← Server 子模块：声明式断言与规则文件
      loader.go             ← 规则文件加载与版本校验
      compiler.go           ← 内置断言和规则文件合并、编译
      evaluator.go          ← op 求值与 AssertionResult 生成
      fields.go             ← 受限 path 解析，区分 missing / null

    client/                 ← Server 子模块：上游 AI 协议客户端
      client.go             ← AI 客户端接口定义
      openai.go             ← OpenAI Completions / Responses
      anthropic.go          ← Anthropic Messages

    store/                  ← Server 子模块：统一持久化层
      store.go              ← 泛型 JSON 文件读写基类（Load / Save / 文件锁）
      task.go               ← TaskStore：~/.ait/tasks/<task-id>.json CRUD
      run.go                ← RunStore：~/.ait/runs/<task-id>/<run-id>/ 读写

    task/                   ← Server 子模块：任务纯业务逻辑
      task.go               ← 任务校验 / 复制 / 默认值归一化

    config/                 ← Server 子模块：应用目录与全局配置
      config.go             ← ~/.ait/config.json 全局配置（使用 store 基类）

    report/                 ← Server 子模块：报告渲染
      report.go
      csv_renderer.go
      json_renderer.go
      turbo_renderer.go     ← Turbo 爬坡报告

    prompt/                 ← Server 子模块：字符串 / 文件 / 长度 Prompt 生成
    network/                ← Server 子模块：IP / DNS / 网络辅助
    logger/                 ← Server 子模块：请求日志
    upload/                 ← Server 子模块：匿名数据上传
    types/                  ← Server 子模块：共享领域类型（Input / RequestMetrics ...）

  tui/                      ← TUI CLIENT（客户端父模块）
    client.go               ← 持有 server.Server；提供 tea.Cmd 包装（异步调用 server）
    model.go                ← 根 BubbleTea Model + 全局状态机
    messages.go             ← 所有 tea.Msg 类型
    styles.go               ← lipgloss 样式常量
    pages/                  ← TUI 子模块：页面组件
      tasklist.go           ← 任务列表页渲染 + 按键处理
      taskdetail.go         ← 任务详情页
      wizard.go             ← 新建 / 编辑弹窗向导（overlay，覆盖任务列表）
      dashboard.go          ← 标准模式仪表盘
      turbodash.go          ← Turbo 仪表盘
      integritydash.go      ← 接口完整性测试仪表盘
      integritydetail.go    ← Case 详情与断言详情页
      reqdetail.go          ← 请求详情页
      contextbar.go         ← Context Bar 组件（条件渲染）

  mcp/                      ← MCP CLIENT（客户端父模块）
    server.go               ← MCP 协议适配与工具注册；仅调用 server.Server 接口

  webui/                    ← WEB CLIENT（未来客户端父模块）
    handler.go              ← HTTP / WebSocket / SSE 桥接
    static/                 ← Web 静态资源或前端构建产物
```

### 3.5 TUI Client 与 Server 交互示意

```
TUI Model (tea.Update)      server.Server            底层执行模块
───────────────────         ─────────────            ────────────
[用户按 r 运行任务]
    │
    ├─ client.StartRunCmd(taskID)
    │       └─ server.StartRun(taskID)
    │               └─ 创建 RunState
    │               └─ go runner.RunWithCallback(cb)  ─→  runner/
    │               └─ cb 内部: eventBus.publishRunEvent(Event{RequestDone})
    │               └─ 返回 runID
    │
    ├─ client.SubscribeRunEventsCmd(runID) → tea.Cmd
    │       └─ server.SubscribeRunEvents(runID) → eventCh
    │       └─ tea.Cmd: 持续从 eventCh 读事件 → tea.Msg
    │
[Event: RequestDone]
    → RequestDoneMsg → Update() → 追加请求行 → View()

[Event: RunComplete]
    → RunCompleteMsg → Update() → 切换到任务详情页（最近运行展开）

[用户按 b 后台运行]
    │ cancelFunc()   ← 停止接收事件，但运行仍在 Server 继续
    │ 返回任务列表
    │ 任务列表中◉ 标记：定时 server.GetRunState(runID) 轮询刷新进度

[用户重新进入仪表盘]
    ├─ server.GetRunState(runID)       ← 恢复当前快照（已完成请求列表）
    └─ server.SubscribeRunEvents(runID) ← 重新订阅，接收后续事件
```

### 3.6 Web UI 接入路径（未来）

新增 `internal/webui/` 包，直接复用同一个 `server.Server` 实例，无需修改 Server 层或任何下层模块：

```go
// internal/webui/handler.go  （示意）
func (h *Handler) startRun(w http.ResponseWriter, r *http.Request) {
    runID, _ := h.server.StartRun(r.PathValue("taskID"))
    eventCh, cancel := h.server.SubscribeRunEvents(runID)
    defer cancel()
    // 通过 SSE 或 WebSocket 将 Event 推给浏览器
    for event := range eventCh {
        writeSSE(w, event)
    }
}
```

### 3.7 关键设计原则

**原则 1：TUI / Web UI 只依赖 server.Server 接口，不直接 import runner / task / report 等包。**

**原则 2：Server 层只依赖执行层和持久化层，不 import 任何 UI 包（tui / webui）。**

**原则 3：执行层（runner / turbo）通过回调/channel 推送进度，不感知 UI，不 import tea 或 http。**

**原则 4：TUI Model 是纯状态机。** 所有副作用（调用 server、读文件）封装在 `tea.Cmd` 中，`Update()` 只做状态转换，方便单元测试。

**原则 5：一个任务只测一个模型。** 任务是最小回归单元，多模型对比通过创建多个任务实现。

### 3.8 迁移自现有架构的变化摘要

```diff
  cmd/ait/
-   ait.go                  ← 含全部逻辑
+   main.go                 ← 仅做：flag解析 + server.New() + tui/mcp 启动

+ internal/server/          ← SERVICE LAYER（核心新增，聚合所有业务子模块）
+   server.go / task.go / run.go / event.go / types.go
+   runner/                 ← 从 internal/runner 迁入
+   turbo/                  ← 从 internal/turbo 迁入
+   integrity/              ← 新增：接口完整性测试执行
+   assertion/              ← 新增：声明式断言与规则文件
+   client/                 ← 从 internal/client 迁入并补充结构化响应保留
+   store/                  ← 从 internal/store 迁入
+   task/                   ← 从 internal/task 迁入
+   config/                 ← 从 internal/config 迁入
+   report/                 ← 从 internal/report 迁入
+   prompt/                 ← 从 internal/prompt 迁入
+   network/                ← 从 internal/network 迁入
+   logger/                 ← 从 internal/logger 迁入
+   upload/                 ← 从 internal/upload 迁入
+   types/                  ← 从 internal/types 迁入

+ internal/mcp/             ← MCP CLIENT（新增）
+   server.go               ← MCP 协议适配，仅依赖 server.Server 接口

  internal/tui/             ← TUI CLIENT（已有，结构不变）

- internal/display/         ← 废弃（功能被 tui/ 和 server/ 替代）
- cmd/tpg/                  ← 废弃（功能合并进 tui 向导）
```

---

## 4. 交互式 TUI 设计

### 4.1 技术选型

| 库 | 用途 |
|----|------|
| `charm.land/bubbletea/v2` | 主框架：消息驱动状态机，从任意 goroutine 安全推送消息 |
| `github.com/charmbracelet/bubbles` | 预制组件：`textinput`、`list`、`spinner`、`viewport`、`progress`、`table` |
| `github.com/charmbracelet/lipgloss` | 样式 & 布局：边框、颜色、多栏弹性布局 |
| `github.com/NimbleMarkets/ntcharts` | Turbo 爬坡折线图 |

### 4.2 TUI 状态机

```
启动无参数 ───────────────────────────────→ TaskList
启动带完整参数 ─→ 创建任务 + 自动 StartRun ─→ Running / TurboRunning / IntegrityRunning

                 ┌─────────────┐
                 │  TaskList   │
                 │  任务列表页   │
                 └──────┬──────┘
    [a 新建/e 编辑]     │  │ [Enter]
          ╔══════▼══╗   │  │  ← Wizard 弹窗 overlay（不切换页面）
          ║  Wizard  ║  │  │
          ║ 弹窗向导  ║  │  │
          ╚══════╤══╝  │  │
           [保存] │     │  │
                 └─────┘  │
                          ▼
                 ┌─────────────┐
                 │ TaskDetail  │
                 │  任务详情页   │
                 └──────┬──────┘
            [Enter / r] │  │ [e 编辑 → 弹窗]
                       │  └──────────────┐
                       │                 │
         [标准模式]    ▼                 │
                 ┌─────────────┐         │
                 │   Running   │         │
                 │  标准运行中   │         │
                 └──────┬──┬───┘         │
      [完成/s 停止]    │  │ [b/Esc 后台] │
                        │  └──→ TaskList │
                        ▼     （◉ 标记）  │
                        └────────────────┘
                （完成/停止后直接返回 TaskDetail，最近运行展开）

         [Turbo 模式]   ▼
                 ┌─────────────┐
                 │TurboRunning │
                 │ Turbo 爬坡中 │
                 └──────┬──┬───┘
      [完成/s 停止]    │  │ [b/Esc 后台]
                        │  └──→ TaskList
                        ▼     （◉ 标记）
                        └──────────────→ TaskDetail
                （完成/停止后直接返回 TaskDetail，最近运行展开）

         [完整性测试] ▼
                 ┌────────────────┐
                 │IntegrityRunning│
                 │ Suite / Case 执行│
                 └──────┬──┬──────┘
      [完成/s 停止]    │  │ [b/Esc 后台]
                        │  └──→ TaskList
                        ▼     （◉ 标记）
                        └──────────────→ TaskDetail
                （完成/停止后直接返回 TaskDetail，最近运行展开）

         [请求详情]  在 Running/TurboRunning 请求列表中选中后
                 ┌─────────────┐
                 │RequestDetail│
                 │  请求详情页   │
                 └──────┬──────┘
           [b/Esc 返回] │
                        └──────→ Running / TurboRunning

         [Case 详情]  在 IntegrityRunning Case 列表中选中后
                 ┌───────────────┐
                 │IntegrityDetail│
                 │Case / Assertion│
                 └──────┬────────┘
           [b/Esc 返回] │
                        └──────→ IntegrityRunning
```

**多任务并发规则：**

- 支持多个任务同时在后台运行，不限数量
- 启动第二个任务时弹出提示："当前已有 N 个任务正在运行，多任务并行可能影响网络指标（DNS/TCP/TLS），`[y]` 继续 `[n]` 取消"
- 任务列表中所有运行中任务都带 `◉` 标记和实时进度
- 任务完成后自动更新对应任务的状态和历史记录，无论当前处于哪个页面

**后台运行规则：**

- 在仪表盘（Running / TurboRunning / IntegrityRunning）按 `[b]` 或 `[Esc]` 可返回任务列表，测试继续在后台执行
- 任务列表中正在运行的任务行首显示 `◉` 标记，对其按 `[Enter]` 可随时重新进入仪表盘

### 4.3 页面设计

---

#### 页面 1：任务列表首页

```
╔══ AIT  任务中心 ──────────────────────────────────────══════╗
║  ◆ AIT   已保存任务: 3   最近运行: 2026-05-16 09:42           ║
╠══════════════════════════════════════════════════════════════╣
║  任务名称                      模式     协议          上次结果║
║  ──────────────────────────────────────────────────────────║
║ ▶ ◉ nightly-openai             标准    responses   ✓ 98.5% ║
║     gpt-4o  并发10  请求200   ◉ 47/100  成功率 98.0%        ║
║                                                             ║
║   turbo-anthropic              Turbo   messages   ★ 并发8  ║
║     claude-3-7  1→50  步进+2  上次: 峰值 TPS 245.3          ║
║                                                             ║
║   integrity-responses          完整性   responses   ✓ 12/12 ║
║     Suite openai-responses-smoke  规则 1 个  警告 2          ║
║                                                             ║
║   smoke-regression             标准    completions  从未运行 ║
║     gpt-4o-mini  并发2  请求20                              ║
║                                                             ║
╠══════════════════════════════════════════════════════════════╣
║  [Enter] 详情/仪表盘  [a] 新建  [e] 编辑  [d] 删除  [r] 运行 ║  ← context bar
╠══════════════════════════════════════════════════════════════╣
║  [↑↓] 选择  [y] 复制  [q] 退出   ◆ AIT  v0.1               ║
╚══════════════════════════════════════════════════════════════╝
```

> **说明：** 任务名前的 `◉` 表示该任务正在后台运行，子行显示实时进度。Context bar（倒数第三行）根据当前选中任务动态调整可用操作；若选中的是运行中任务，`[Enter]` 进入仪表盘而非详情页。

---

#### 页面 2：任务详情页

*（兼做结果页——运行结束后自动跳转至此，最近一次运行展开展示）*

```
╔══ AIT  任务详情 ─ nightly-openai ──────────────────────════╗
║  ◆ AIT   任务 ID: task_01   更新: 2026-05-16 09:30   刚刚    ║
╠══════════════════════════════════════════════════════════════╣
║  配置摘要                                                     ║
║  协议  openai-responses    接口  https://api.openai.com/...  ║
║  模型  gpt-4o   模式  标准模式   并发  10   请求  200          ║
║  超时  30s   流式  开启   Prompt  你好，介绍一下你自己。        ║
╠══════════════════════════════════════════════════════════════╣
║  最近运行 ▼ 2026-05-16 09:30  ✓ 完成  100 请求  耗时 20.4s   ║  ← 展开行
║  ──────────────────────────────────────────────────────────  ║
║  指标          最小值    平均值    标准差    最大值             ║
║  总耗时         0.82s    1.24s    ±0.31s    3.12s             ║
║  TTFT           71ms    245ms     ±89ms     812ms             ║
║  输出 TPS       89.2    124.3     ±21.4     198.5             ║
║  缓存命中率     0.0%     42.0%   ±18.5%    100.0%             ║
║  输入 Token       42       64      ±12        98             ║
║  输出 Token       78      128      ±32       195             ║
║  错误  context deadline exceeded (timeout) × 2              ║
╠══════════════════════════════════════════════════════════════╣
║  历史运行记录                                                  ║
║   时间               模式   成功率    TTFT     TPS    Cache   ║
║  ──────────────────────────────────────────────────────────  ║
║  2026-05-15 23:10    标准   99.0%   231ms   128.1   38.0%    ║
║  2026-05-15 21:42    标准   87.0%   timeout ×2      12.0%    ║
╠══════════════════════════════════════════════════════════════╣
║  [Enter/r] 运行  [r] 生成报告  [c] 复制摘要  [e] 编辑         ║  ← context bar
╠══════════════════════════════════════════════════════════════╣
║  [b/Esc] 返回列表   ◆ AIT  v0.1                              ║
╚══════════════════════════════════════════════════════════════╝
```

> **说明：** 运行结束（或 `[s]` 停止）后自动跳转至此，最近一次运行结果默认展开；历史记录折叠为摘要行。`[r] 生成报告` 和 `[c] 复制摘要` 仅在有运行记录时出现在 Context bar。

---

#### 页面 3：弹窗向导 — 新建任务（Step 1/3 · 基本信息）

*(overlay 覆盖任务列表，背景列表只读；编辑任务时同样覆盖任务详情页)*

```
╔══ AIT  任务中心 ─────────────────────────────════╗
║  ◆ AIT   (列表背景，只读暗化)               ║
║  ┌────────────── 新建任务  1/3 · 基本信息 ──────────────┐  ║
║  │                                                       │  ║
║  │  任务名称    ________________________                 │  ║
║  │               nightly-openai                         │  ║
║  │                                                       │  ║
║  │  协议类型    ● openai-responses                       │  ║
║  │               ○ openai-completions                   │  ║
║  │               ○ anthropic-messages                   │  ║
║  │                                                       │  ║
║  │  接口地址    ________________________                 │  ║
║  │               https://api.openai.com/...             │  ║
║  │               提示：填写完整接口地址，而非 base URL   │  ║
║  │                                                       │  ║
║  │  API 密钥    ________________________                 │  ║
║  │               sk-••••••••••••••••                    │  ║
║  │                                                       │  ║
║  │  测试模型    ________________________                 │  ║
║  │               gpt-4o                                 │  ║
║  │               提示：每个任务仅允许选择一个模型         │  ║
║  │                                                       │  ║
║  ├───────────────────────────────────────────────────────┤  ║
║  │  [Tab] 下一项  [↑↓] 切换协议  [Enter] 下一步  [Esc] 取消 │  ║
║  └───────────────────────────────────────────────────────┘  ║
╚══════════════════════════════════════════════════════════════╝
```

---

#### 页面 4：弹窗向导 — 新建任务（Step 2/3 · 测试参数）

```
╔══ AIT  任务中心 ─────────────────────────────════╗
║  ◆ AIT   (列表背景，只读暗化)               ║
║  ┌────────────── 新建任务  2/3 · 测试参数 ──────────────┐  ║
║  │                                                       │  ║
║  │  测试模式    ○ 标准模式    ● Turbo 模式    ○ 完整性测试 │  ║
║  │               [←→ 切换]                               │  ║
║  │  ── 标准模式参数 ───────────────────────────────      │  ║
║  │  并发数 [  5  ]   请求总数 [ 100 ]                    │  ║
║  │  超时时间 [ 30s ]   流式模式  [✓ 开启]                │  ║
║  │  ── Turbo 模式参数 ─────────────────────────────      │  ║
║  │  初始并发 [  1  ]   最大并发  [  50  ]                │  ║
║  │  步进值 [  2  ]   每级请求数 [  30  ]                 │  ║
║  │  停止条件  成功率低于 [ 90% ] 或 延迟 > [ 10s ]       │  ║
║  │  ── 完整性测试参数 ─────────────────────────────      │  ║
║  │  Suite [ openai-responses-smoke      ]                │  ║
║  │  规则文件 [ ~/.ait/rules/responses-extra.json ]       │  ║
║  │  失败策略 [ 全部执行 ]   单 Case 超时 [ 30s ]         │  ║
║  │  ── Prompt 配置 ────────────────────────────────      │  ║
║  │  输入方式  ● 直接输入   ○ 文件   ○ 按长度生成         │  ║
║  │  内容      你好，介绍一下你自己。                     │  ║
║  │               ─────────────────────────────────       │  ║
║  │                                                       │  ║
║  ├───────────────────────────────────────────────────────┤  ║
║  │  [Tab] 下一项  [←→] 切换模式  [Enter] 下一步  [Esc] 返回 │  ║
║  └───────────────────────────────────────────────────────┘  ║
╚══════════════════════════════════════════════════════════════╝
```

---

#### 页面 5：弹窗向导 — 新建任务（Step 3/3 · 确认保存）

```
╔══ AIT  任务中心 ─────────────────────────────════╗
║  ◆ AIT   (列表背景，只读暗化)               ║
║  ┌────────────── 新建任务  3/3 · 确认保存 ──────────────┐  ║
║  │                                                       │  ║
║  │  任务名称    nightly-openai                           │  ║
║  │  协议        openai-responses                         │  ║
║  │  接口地址    https://api.openai.com/...               │  ║
║  │  API 密钥    sk-****...****                           │  ║
║  │  测试模型    gpt-4o                                   │  ║
║  │  测试模式    Turbo 模式                               │  ║
║  │  并发爬坡    1 → 50  步进 +2  每级 30 请求            │  ║
║  │  停止条件    成功率 < 90%  或  延迟 > 10s             │  ║
║  │  完整性      Suite openai-responses-smoke  规则 1 个  │  ║
║  │  流式模式    开启                                     │  ║
║  │  Prompt      你好，介绍一下你自己。 (长度: 12)        │  ║
║  │                                                       │  ║
║  │  保存任务到 ~/.ait/tasks/<task-id>.json  [✓]          │  ║
║  │                                                       │  ║
║  │  ▶  保存任务                                          │  ║
║  │                                                       │  ║
║  ├───────────────────────────────────────────────────────┤  ║
║  │  [Enter] 保存任务   [r] 保存并运行   [Esc] 返回修改   │  ║
║  └───────────────────────────────────────────────────────┘  ║
╚══════════════════════════════════════════════════════════════╝
```

> **说明：** 弹窗向导不切换页面，以 overlay 方式覆盖当前页面（任务列表或任务详情）。新建 / 编辑同一套 overlay，编辑时内容预填。
>
> **自动运行规则：** `[Enter]` 保存任务时，若当前**没有任何运行中的任务**，自动调用 `StartRun` 并进入仪表盘；若已有任务运行，则仅保存并返回任务列表（不弹干扰提示）。`[r]` 保存并运行时**无论是否有其他任务运行，始终启动**（与多任务并发规则一致，启动前弹干扰风险提示）。

---

#### 页面 6：标准模式运行仪表盘

```
╔══ AIT  正在测试 ─ nightly-openai ──────────────────────════╗
║  ◆ AIT   gpt-4o · openai-responses · 并发: 5 · 请求: 100   ║
╠══════════════════════════╦═════════════════════════════════╣
║  任务参数                 ║  实时指标                        ║
║                           ║                                 ║
║  协议  responses          ║  成功率     98.0%               ║
║  模型  gpt-4o             ║  avg TPS    124.3 tok/s         ║
║  并发  5   请求  100      ║  avg TTFT   245ms               ║
║  超时  30s   流式  开启   ║  缓存命中   42.0%               ║
║                           ║  avg 总耗时  1.24s              ║
║                           ║  成功: 45   失败: 2             ║
╠══════════════════════════╩═════════════════════════════════╣
║  进度  ████████░░  47 / 100   已用: 12.4s  剩余: ~8.2s      ║
╠═════════════════════════════════════════════════════════════╣
║  请求列表                                                    ║
║   #     状态   总耗时   TTFT     Cache    输出Token   TPS    ║
║  ──────────────────────────────────────────────────────── ║
║ ▶ #48   ✓      245ms   82ms     100%     128tok    12.3/s   ║
║   #47   ✗      timeout(30.0s)                              ║
║   #46   ✓      312ms   95ms      25%      96tok     9.8/s   ║
║   #45   ✓      198ms   71ms       0%     145tok    14.2/s   ║
║   #44   ✓      271ms  103ms      50%     112tok    11.1/s   ║
╠═════════════════════════════════════════════════════════════╣
║  [Enter] 查看请求详情  [↑↓] 选择请求  [s] 停止               ║  ← context bar
╠═════════════════════════════════════════════════════════════╣
║  [s] 停止  [b] 后台运行  [r] 提前报告  [q] 退出              ║
╚═════════════════════════════════════════════════════════════╝
```

> **说明：** 上方左右分栏：左侧展示任务参数，右侧展示实时指标（任务完成后变为结果指标）；进度条独立一行；请求列表按完成时间倒序滚动（最新在上方），可用 `[↑↓]` 选中一行，`[Enter]` 进入请求详情页。Context bar 根据是否有选中请求动态显示可用操作。

---

#### 页面 7：Turbo 模式运行仪表盘

```
╔══ AIT  Turbo 探测 ─ turbo-anthropic ───────────────────════╗
║  ◆ AIT   claude-3-7 · anthropic-messages · 1→50  步进+2    ║
╠══════════════════════════╦═════════════════════════════════╣
║  任务参数                 ║  当前级别实时指标 [并发 = 8]    ║
║                           ║                                 ║
║  协议  messages           ║  成功率    96.0%                ║
║  模型  claude-3-7         ║  TPS       245.3 tok/s         ║
║  爬坡  1→50  步进+2       ║  TTFT      312ms               ║
║  每级  30 请求            ║  Cache     44.0%               ║
║  停止  成功率 < 90%       ║  总耗时    1.51s               ║
║        延迟 > 10s         ║  已用      38.2s               ║
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
║  ▶  8   96.0%   245.3   312ms    44.0%    1.51s  🔄 进行中  ║
╠═════════════════════════════════════════════════════════════╣
║  [Enter] 查看该级别请求列表  [↑↓] 选择                      ║  ← context bar
╠═════════════════════════════════════════════════════════════╣
║  [s] 停止  [b] 后台运行  [m] 标记极限  [r] 提前报告  [q] 退出║
╚═════════════════════════════════════════════════════════════╝
```

> **说明：** 上方左右分栏：左侧展示任务参数，右侧展示当前级别实时指标（并发标号显示在标题）；进度条同时显示当前级别进度和总体级别进度；级别列表可用 `[↑↓]` 选中已完成的级别，`[Enter]` 查看该级别的请求列表（复用标准仪表盘的请求列表布局，状态为只读）。

---

#### 页面 8：接口完整性测试仪表盘

```
╔══ AIT  接口完整性测试 ─ integrity-responses ─────────────════╗
║  ◆ AIT   gpt-4o · openai-responses · openai-responses-smoke ║
╠══════════════════════════╦═════════════════════════════════╣
║  测试配置                 ║  当前结论                        ║
║                           ║                                 ║
║  Suite  smoke             ║  通过       10 / 12              ║
║  规则   1 个文件           ║  警告       2                    ║
║  Case   12                ║  失败       0                    ║
║  超时   30s / Case        ║  结论       ✓ 通过，有警告         ║
╠══════════════════════════╩═════════════════════════════════╣
║  进度  █████████░  11 / 12   当前: stream-basic-shape       ║
╠═════════════════════════════════════════════════════════════╣
║  Case 列表                                                   ║
║   Case ID                    能力项          断言       结论 ║
║  ──────────────────────────────────────────────────────── ║
║   basic-response-shape        response       12/12      ✓    ║
║   usage-shape                 usage           5/6      ⚠    ║
║ ▶ stream-basic-shape          streaming       8/8      🔄    ║
║   error-format                error           4/4      ✓    ║
╠═════════════════════════════════════════════════════════════╣
║  [Enter] 查看 Case 详情  [↑↓] 选择  [s] 停止                 ║
╠═════════════════════════════════════════════════════════════╣
║  [s] 停止  [b] 后台运行  [r] 提前报告  [q] 退出              ║
╚═════════════════════════════════════════════════════════════╝
```

> **说明：** 完整性测试仪表盘关注 Suite、Case、Assertion 和 Capability 覆盖，不展示 TPS/TTFT 等性能指标作为主视图。Case 详情页展示请求、响应结构、派生指标和断言结果；运行完成后跳转任务详情页，最近运行展开为完整性结论摘要。

---

#### 页面 9：请求详情页

```
╔══ AIT  请求详情 - nightly-openai  #48 ─────────────────════╗
║  ◆ AIT   任务: nightly-openai  请求 #48 / 100   ✓ 成功      ║
╠══════════════════════════╦═════════════════════════════════╣
║  性能指标                 ║  网络指标                        ║
║                           ║                                 ║
║  状态    ✓ 成功           ║  DNS       1.2ms                ║
║  总耗时  245ms            ║  TCP 连接  2.1ms                ║
║  TTFT    82ms             ║  TLS 握手  8.4ms                ║
║  输出TPS 12.3 tok/s       ║                                 ║
║  输入Token   64           ║                                 ║
║  输出Token  128           ║                                 ║
║  缓存命中  100%           ║                                 ║
╠══════════════════════════╩═════════════════════════════════╣
║  输入 (Prompt)                                               ║
║  ──────────────────────────────────────────────────────── ║
║  你好，介绍一下你自己。                                      ║
╠═════════════════════════════════════════════════════════════╣
║  输出 (Response)                                             ║
║  ──────────────────────────────────────────────────────── ║
║  你好！我是 Claude，一个由 Anthropic 开发的 AI 助手。我可以  ║
║  帮助你解答问题、分析文本、编写代码等各种任务。请告诉我你     ║
║  需要什么帮助！                                              ║
║  （↑↓ 滚动查看完整内容）                                     ║
╠═════════════════════════════════════════════════════════════╣
║  [b/Esc] 返回仪表盘  [↑↓] 滚动  [←→] 上/下一条请求          ║
╚═════════════════════════════════════════════════════════════╝
```

> **说明：** 支持 `[←→]` 切换前后请求，无需每次返回仪表盘。输入/输出区域均可独立滚动，内容较长时显示滚动提示。

---

### 4.4 键盘交互规范

| 按键 | 适用页面 | 功能 |
|------|----------|------|
| `a` | 任务列表 | 新建任务 |
| `Enter` | 任务列表（普通任务） | 查看任务详情 |
| `Enter` | 任务列表（运行中任务） | 重新进入仪表盘 |
| `r` | 任务列表 / 任务详情 | 运行当前任务（有其他任务运行时提示干扰风险） |
| `e` | 任务列表 / 任务详情 | 编辑当前任务 |
| `d` | 任务列表 / 任务详情 | 删除当前任务 |
| `y` | 任务列表 / 任务详情 | 复制当前任务 |
| `b` / `Esc` | 任务详情 | 返回任务列表 |
| `b` / `Esc` | 仪表盘（标准/Turbo/完整性） | **后台运行**，返回任务列表（测试继续进行） |
| `b` / `Esc` | 请求详情页 / Case 详情页 | 返回仪表盘 |
| `Enter` | 仪表盘请求列表 | 进入请求详情页 |
| `Enter` | 完整性仪表盘 Case 列表 | 进入 Case 详情页 |
| `↑` / `↓` | 仪表盘请求列表 / Case 列表 / 详情页 | 选择条目 / 滚动内容 |
| `←` / `→` | 请求详情页 | 切换上/下一条请求 |
| `Tab` / `Shift+Tab` | 向导 | 在输入项间切换焦点 |
| `↑` / `↓` | 任务列表、向导 | 上下选择 |
| `←` / `→` | 向导模式选择 | 切换选项 |
| `Enter` | 向导 | 确认 / 下一步 / 保存 |
| `Esc` | 所有页 | 返回上一步 / 取消 |
| `s` | 仪表盘（标准/Turbo/完整性） | 停止测试 |
| `r` | 仪表盘 / 任务详情 | 生成报告文件 |
| `m` | Turbo 仪表盘 | 手动标记当前并发为最大稳定并发并停止 |
| `c` | 任务详情（有运行记录时） | 复制最近运行摘要到剪贴板 |
| `q` / `Ctrl+C` | 所有页 | 退出程序 |

---

### 4.5 Context Bar 规范

**Context Bar** 是紧贴 Footer 上方的一行动态提示区域，**仅当当前页面/选中项有可用操作时才显示**；若无选中项或无可执行操作，该行完全不渲染（不占空间）。

**格式：**

```
[key] 操作描述  [key] 操作描述  [key] 操作描述
```

**各页面 Context Bar 内容：**

| 页面 / 场景 | Context Bar 示例 |
|------------|------------------|
| 任务列表（选中普通任务） | `[Enter] 查看详情  [r] 运行  [e] 编辑  [d] 删除  [y] 复制` |
| 任务列表（选中运行中任务） | `[Enter] 进入仪表盘  [s] 停止  [y] 复制` |
| 任务列表（无任务） | 不显示 |
| 任务详情（无运行记录） | `[Enter/r] 运行  [e] 编辑  [y] 复制  [d] 删除` |
| 任务详情（有运行记录） | `[r] 生成报告  [c] 复制摘要  [Enter/r] 再次运行  [e] 编辑` |
| 仪表盘（未选中请求） | `[s] 停止  [b] 后台运行  [r] 提前报告` |
| 仪表盘（选中请求） | `[Enter] 查看请求详情  [↑↓] 选择请求  [s] 停止` |
| Turbo 仪表盘（未选中级别） | `[s] 停止  [b] 后台运行  [m] 标记极限` |
| Turbo 仪表盘（选中已完成级别） | `[Enter] 查看该级别请求列表  [↑↓] 选择  [s] 停止` |
| 完整性仪表盘（未选中 Case） | `[s] 停止  [b] 后台运行  [r] 提前报告` |
| 完整性仪表盘（选中 Case） | `[Enter] 查看 Case 详情  [↑↓] 选择  [s] 停止` |
| 请求详情页 | `[b/Esc] 返回仪表盘  [←→] 上/下一条请求` |
| Case 详情页 | `[b/Esc] 返回完整性仪表盘  [←→] 上/下一个 Case` |

**规则：**
- Context Bar 使用与 Footer 相同的暗色调，但前景色略亮（用于区分层级）
- 仅展示**当前状态下可执行**的操作（例如：仅在请求列表选中行时才显示 `[Enter] 查看详情`）
- Context Bar 不替代 Footer——Footer 始终展示全局快捷键（`[q]` 退出等）

---

### 4.6 布局响应式策略

- 终端宽度 `< 80` 列：所有页面折叠为单列；仪表盘将指标区与进度条叠放为两行
- 终端宽度 `≥ 80` 列：仪表盘实时指标全宽展示，进度条独立一行；任务列表与任务详情均为全宽单列
- 终端高度不足时，请求列表区域自动收缩（最少保留 3 行）；输入/输出内容区优先滚动而非截断
- Context Bar 若无内容则不渲染，不影响其他区域高度分配
- 仪表盘页面进度条独立占一行，位于实时指标区域与请求列表之间

---

## 5. Turbo 模式设计

Turbo 模式是 AIT 的承载能力探测模式，用于在同一任务配置下逐级提升并发，自动找到最大稳定并发边界。

完整设计见 [Turbo 模式功能设计](turbo.md)。总设计只保留关键约束：

- `mode = "turbo"`，与 `standard`、`integrity` 并列。
- 从 `init_concurrency` 开始线性爬坡，直到成功率低于阈值、平均延迟超过阈值、达到最大并发或用户手动停止。
- 每个并发级别都复用标准请求执行和指标采集能力。
- 每条请求写入统一 `requests.jsonl`，并在 `contexts.turbo` 中记录并发级别、级别序号和级别内请求序号。
- 最终爬坡结论写入 `result.json` 的 `turbo` 区块，包括最大稳定并发、峰值 TPS、停止原因和级别结果。
- TUI 主视图展示当前级别、级别列表和爬坡结论；选中级别后可查看该级别请求明细。

---

## 6. 接口完整性测试模式设计

接口完整性测试是 AIT 的协议与业务行为验证模式。它不以性能压测为主要目标，而是确认目标服务在指定协议下是否具备可用、稳定、可解释的接口行为。

完整设计见 [接口完整性测试功能设计](integrity.md)。总设计只保留关键约束：

- `mode = "integrity"`，与 `standard`、`turbo` 并列。
- 核心模型统一为 `Suite / Case / Assertion / Capability`。
- 内置 Suite 负责协议级 Case；自定义测试规则通过 `integrity.rule_files` 追加或覆盖断言。
- 自定义测试规则是第一版核心能力，首版只支持声明式规则文件，不支持脚本和自定义请求编排。
- 执行层必须尽可能保留服务端返回结构，供断言、失败排查和后续内容检查使用。
- 每个 Case 对应的请求事实写入统一 `requests.jsonl`，并在 `contexts.integrity` 中记录 Suite、Case、Capability 和断言结果。
- 最终完整性结论写入 `result.json` 的 `integrity` 区块，包括 Case 统计、Capability 覆盖、断言统计和失败摘要。
- MCP 适配层只通过 `server.Server` 暴露 Suite 预览、规则校验和 Case 详情等能力。

---

## 7. 新增数据结构与接口

### 7.1 TaskDefinition

```go
// internal/types/types.go 新增

// TaskDefinition 可重复执行的测试任务定义
type TaskDefinition struct {
  ID        string    `json:"id"`
  Name      string    `json:"name"`
  Input     Input     `json:"input"`
  CreatedAt time.Time `json:"created_at"`
  UpdatedAt time.Time `json:"updated_at"`
}
```

任务文件只保存配置本体，不回填最近运行摘要；任务列表和任务详情中的最近运行信息由 `runs/<task-id>/` 下的 `run.json` 与 `result.json` 现场读取或聚合。

其中 `Input.Model` 为单个模型标识，不允许逗号分隔列表。
其中 `Input.Protocol` 允许值为 `openai-completions`、`openai-responses`、`anthropic-messages`。
其中 `Input.EndpointURL` 必须是完整接口地址，例如：
- `openai-completions` → `https://api.openai.com/v1/chat/completions`
- `openai-responses` → `https://api.openai.com/v1/responses`
- `anthropic-messages` → `https://api.anthropic.com/v1/messages`

### 7.2 TaskRunSummary

```go
// TaskRunSummary 单次任务运行后的摘要信息
type TaskRunSummary struct {
  RunID                 string        `json:"run_id"`
  TaskID                string        `json:"task_id"`
  Mode                  string        `json:"mode"` // standard | turbo | integrity
  Status                string        `json:"status"`
  Protocol              string        `json:"protocol"`
  Model                 string        `json:"model"`
  StartedAt             time.Time     `json:"started_at"`
  FinishedAt            time.Time     `json:"finished_at"`
  SuccessRate           float64       `json:"success_rate,omitempty"`
  AvgTTFT               time.Duration `json:"avg_ttft,omitempty"`
  AvgTPS                float64       `json:"avg_tps,omitempty"`
  CacheHitRate          float64       `json:"cache_hit_rate,omitempty"`
  MaxStableConcurrency  int           `json:"max_stable_concurrency,omitempty"`
  IntegrityConclusion   string        `json:"integrity_conclusion,omitempty"`
  IntegrityPassedCases  int           `json:"integrity_passed_cases,omitempty"`
  IntegrityFailedCases  int           `json:"integrity_failed_cases,omitempty"`
  IntegrityWarnCount    int           `json:"integrity_warn_count,omitempty"`
  ReportJSONPath        string        `json:"report_json_path,omitempty"`
  ReportCSVPath         string        `json:"report_csv_path,omitempty"`
  ErrorSummary          string        `json:"error_summary,omitempty"`
}
```

### 7.3 TurboConfig

```go
// internal/types/types.go 新增

// TurboConfig Turbo 模式的配置参数
type TurboConfig struct {
    InitConcurrency int           // 初始并发数，默认 1
    MaxConcurrency  int           // 最大探测并发数，默认 50
    StepSize        int           // 每级步进值，默认 2
    LevelRequests   int           // 每级执行请求数，默认 30
    MinSuccessRate  float64       // 停止阈值：成功率，默认 0.9
    MaxLatency      time.Duration // 停止阈值：平均延迟，默认 10s
}

// TurboLevelResult 单个并发级别的测试结果
type TurboLevelResult struct {
    Concurrency   int           `json:"concurrency"`
    TotalRequests int           `json:"total_requests"`
    SuccessCount  int           `json:"success_count"`
    SuccessRate   float64       `json:"success_rate"`
    AvgTPS        float64       `json:"avg_tps"`
    PeakTPS       float64       `json:"peak_tps"`
    AvgTTFT       time.Duration `json:"avg_ttft"`
    CacheHitRate  float64       `json:"cache_hit_rate"`
    AvgTotalTime  time.Duration `json:"avg_total_time"`
    StdDevTPS     float64       `json:"stddev_tps"`
    Stable        bool          `json:"stable"`
    StopReason    string        `json:"stop_reason,omitempty"`
}

// TurboResult Turbo 模式的最终结果
type TurboResult struct {
    Config                TurboConfig        `json:"config"`
    Levels                []TurboLevelResult `json:"levels"`
    MaxStableConcurrency  int                `json:"max_stable_concurrency"`
    PeakTPS               float64            `json:"peak_tps"`
    StopReason            string             `json:"stop_reason"`
    ProbeDuration         time.Duration      `json:"probe_duration"`
    Model                 string             `json:"model"`
    Protocol              string             `json:"protocol"`
    EndpointURL           string             `json:"endpoint_url"`
    Timestamp             string             `json:"timestamp"`
}
```

现有指标结构也需要补充缓存命中率字段，用于 dashboard、任务详情页和报告渲染：

```go
type ResponseMetrics struct {
  CachedInputTokens int     // 当前请求命中的输入缓存 token 数
  CacheHitRate      float64 // CachedInputTokens / max(PromptTokens, 1)
}

type StatsData struct {
  CacheHitRates []float64 // 所有请求的缓存命中率
}

type ReportData struct {
  AvgCacheHitRate    float64 `json:"avg_cache_hit_rate"`
  MinCacheHitRate    float64 `json:"min_cache_hit_rate"`
  MaxCacheHitRate    float64 `json:"max_cache_hit_rate"`
  StdDevCacheHitRate float64 `json:"stddev_cache_hit_rate"`
}
```

### 7.4 Input 扩展

```go
// internal/types/types.go 扩展 Input

type Input struct {
    // ... 现有字段 ...

    Mode        string // standard | turbo | integrity
    Protocol    string // openai-completions | openai-responses | anthropic-messages
    EndpointURL string // 完整接口地址，例如 https://api.openai.com/v1/responses

    // 标准模式
    Concurrency int
    Count       int

    // Turbo 模式
    TurboConfig TurboConfig // Mode == "turbo" 时生效

    // 接口完整性测试模式
    Integrity IntegrityConfig // Mode == "integrity" 时生效
}

// IntegrityConfig 接口完整性测试配置
type IntegrityConfig struct {
    Suite         string   `json:"suite"`           // 例如 openai-responses-smoke
    FailFast      bool     `json:"fail_fast"`
    CaseTimeoutMS int      `json:"case_timeout_ms"`
    RuleFiles     []string `json:"rule_files"`      // 自定义测试规则文件
}
```

### 7.5 IntegrityConfig 与结果类型

```go
// AssertionResult 单条断言结果
type AssertionResult struct {
    ID      string `json:"id"`
    Name    string `json:"name,omitempty"`
    CaseID  string `json:"case_id"`
    Phase   string `json:"phase,omitempty"`
    Level   string `json:"level"`   // error | warn | info
    Path    string `json:"path,omitempty"`
    Op      string `json:"op"`
    Passed  bool   `json:"passed"`
    Message string `json:"message,omitempty"`
    Source  string `json:"source"`  // builtin | rule_file
}

// IntegrityCaseResult 单个 Case 的执行结果
type IntegrityCaseResult struct {
    CaseID      string            `json:"case_id"`
    Name        string            `json:"name"`
    Capability  string            `json:"capability"`
    Status      string            `json:"status"` // passed | warned | failed | skipped
    Assertions  []AssertionResult `json:"assertions"`
    RequestRef  string            `json:"request_ref"`
    ErrorSummary string           `json:"error_summary,omitempty"`
}

// IntegrityResult 完整性测试最终结果
type IntegrityResult struct {
    Suite           string                `json:"suite"`
    Protocol        string                `json:"protocol"`
    EndpointURL     string                `json:"endpoint_url"`
    Model           string                `json:"model"`
    Conclusion      string                `json:"conclusion"` // passed | warned | failed
    TotalCases      int                   `json:"total_cases"`
    PassedCases     int                   `json:"passed_cases"`
    WarnedCases     int                   `json:"warned_cases"`
    FailedCases     int                   `json:"failed_cases"`
    Capabilities    map[string]string     `json:"capabilities"`
    Cases           []IntegrityCaseResult `json:"cases"`
    RuleFileSources []string              `json:"rule_file_sources,omitempty"`
}
```

### 7.6 Server 层类型

```go
// internal/server/types.go

// RunID 是一次运行的唯一标识
type RunID string

// ReportFormat 报告格式
type ReportFormat string

const (
    ReportFormatJSON ReportFormat = "json"
    ReportFormatCSV  ReportFormat = "csv"
)

// RunStatus 运行状态枚举
type RunStatus string

const (
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusStopped   RunStatus = "stopped"
)

// RunState 一次运行的当前状态快照（用于 GetRunState / 后台轮询）
type RunState struct {
    RunID        RunID
    TaskID       string
    Status       RunStatus
    Mode         string       // "standard" | "turbo" | "integrity"
    StartedAt    time.Time
    FinishedAt   *time.Time
    // 标准模式
    TotalReqs    int
    DoneReqs     int
    SuccessReqs  int
    FailedReqs   int
    Requests     []*RequestMetrics // 已完成请求列表（供重新进入仪表盘恢复）
    // Turbo 模式
    Levels       []types.TurboLevelResult
    CurrentLevel int
    // 完整性测试模式
    IntegritySuite       string
    IntegrityCases       []types.IntegrityCaseResult
    CurrentIntegrityCase string
    AssertionResults     []types.AssertionResult
    // 聚合实时指标（标准模式 / Turbo 当前级别）
    AvgTPS       float64
    AvgTTFT      time.Duration
    SuccessRate  float64
    CacheHitRate float64
    // 最终结果（完成后填充）
    StandardResult  *types.ReportData
    TurboResult     *types.TurboResult
    IntegrityResult *types.IntegrityResult
    ErrorMsg        string
}

// RequestMetrics 单条请求的指标快照（供请求详情页展示）
type RequestMetrics struct {
    Index            int
    Success          bool
    TotalTime        time.Duration
    TTFT             time.Duration
    TPS              float64
    PromptTokens     int
    CompletionTokens int
    CachedTokens     int
    CacheHitRate     float64
    DNSTime          time.Duration
    ConnectTime      time.Duration
    TLSTime          time.Duration
    TargetIP         string
    ErrorMessage     string
    PromptText       string   // 原始输入（供请求详情页展示）
    ResponseText     string   // 原始输出
}

// RunSummary 历史记录列表项
type RunSummary struct {
    RunID               RunID
    StartedAt           time.Time
    Status              RunStatus
    Mode                string
    SuccessRate         float64
    AvgTTFT             time.Duration
    AvgTPS              float64
    CacheHitRate        float64
    MaxStableConcurrency int
    ReportPath          string
}

// EventKind 事件类型枚举
type EventKind string

const (
    EventRequestDone          EventKind = "request_done"           // 单条请求完成
    EventProgressTick         EventKind = "progress_tick"          // 500ms 聚合快照
    EventLevelDone            EventKind = "level_done"             // Turbo 一级完成
    EventIntegrityCaseStarted EventKind = "integrity_case_started" // 完整性 Case 开始
    EventIntegrityCaseDone    EventKind = "integrity_case_done"    // 完整性 Case 完成
    EventAssertionResult      EventKind = "assertion_result"       // 单条断言完成
    EventRunComplete          EventKind = "run_complete"           // 全部完成
    EventRunFailed            EventKind = "run_failed"             // 运行出错
)
```

### 7.7 TUI 消息类型

TUI 层的 `tea.Msg` 类型由 `tui/client.go` 包装 `server.Server` 调用后产生，不直接暴露 server 内部类型：

```go
// internal/tui/messages.go

// TasksLoadedMsg 任务列表加载完成
type TasksLoadedMsg struct {
    Tasks []types.Task
}

// TaskSavedMsg 任务保存完成
type TaskSavedMsg struct {
    Task types.Task
}

// HistoryLoadedMsg 运行历史加载完成
type HistoryLoadedMsg struct {
    TaskID  string
    History []server.RunSummary
}

// RunStartedMsg 运行启动成功，获得 RunID
type RunStartedMsg struct {
    RunID  server.RunID
    TaskID string
}

// ServerEventMsg 从 server.Subscribe 接收到的事件（统一包装）
type ServerEventMsg struct {
    Event server.Event
}

// RunStateMsg server.GetRunState 的轮询结果（后台模式重新进入仪表盘时使用）
type RunStateMsg struct {
    State *server.RunState
}

// ReportGeneratedMsg 报告生成完成
type ReportGeneratedMsg struct {
    Path string
}

// ErrorMsg 操作出错
type ErrorMsg struct {
    Err error
}
```

### 7.8 Runner 接口扩展

```go
// internal/server/runner/runner.go — Server 层内部使用，TUI 不直接调用

// RequestDoneCallback 每个请求完成后的回调（由 server/run_service.go 包装为 Event）
type RequestDoneCallback func(metrics *ResponseMetrics, index int, err error)

// RunWithCallback 运行测试，每个请求完成后调用 cb（线程安全）
func (r *Runner) RunWithCallback(cb RequestDoneCallback) (*types.ReportData, error)

// Stop 异步停止正在进行的测试
func (r *Runner) Stop()
```

### 7.9 统一存储接口

目标存储结构以 [统一运行存储设计](storage.md) 为准。总设计只约定最小 Repository 能力：

```go
// internal/server/store/config_repo.go
func LoadConfig() (*config.Config, error)    // ~/.ait/config.json
func SaveConfig(cfg *config.Config) error

// internal/server/store/task_repo.go
func ListTasks() ([]types.TaskDefinition, error)       // 扫描 ~/.ait/tasks/
func LoadTask(taskID string) (*types.TaskDefinition, error)
func SaveTask(task types.TaskDefinition) error          // ~/.ait/tasks/<task-id>.json
func DeleteTask(taskID string) error

// internal/server/store/run_repo.go
func CreateRun(taskID string, meta types.RunMeta) (server.RunID, error)
func UpdateRunMeta(taskID string, runID server.RunID, meta types.RunMeta) error
func SaveRunResult(taskID string, runID server.RunID, result types.RunResult) error
func LoadRun(taskID string, runID server.RunID) (*types.RunMeta, *types.RunResult, error)
func ListRuns(taskID string, limit int) ([]types.RunSummary, error) // 扫描 runs/<task-id>/

// internal/server/store/request_log.go
func AppendRequestFact(taskID string, runID server.RunID, fact types.RequestFact) error
func ReadRequestFacts(taskID string, runID server.RunID) ([]types.RequestFact, error)
```

约束：

- 任务定义存入 `~/.ait/tasks/<task-id>.json`，不再使用单文件聚合任务清单。
- 运行历史来自 `~/.ait/runs/<task-id>/<run-id>/`，不再维护独立历史文件或历史索引。
- 请求级事实统一写入 `requests.jsonl`。
- 最终业务结论统一写入 `result.json`。
- 运行事件只服务实时 UI，不作为最终事实源。

---

## 8. 开发计划

> **约定：** 严格遵守 SC 分层原则——先建好 Server 接口，再实现 TUI Client；所有 UI 层代码仅 import `internal/server`，不直接 import `runner` / `task` / `report` 等下层包。

### Phase 1 — Server 层 + TUI 基础框架

**目标：** 建立 SC 架构骨架，跑通"创建任务 → 运行 → 看到进度 → 查看结果"主流程

**Step 1：Server 层（先行）**

- [ ] `internal/server/types/types.go`：补充 `Task`、`TaskConfig`、`TurboConfig`、`ReportData` 等领域类型
- [ ] `internal/server/store/fs.go`：统一文件读写、JSON/JSONL、目录工具与文件锁
- [ ] `internal/server/store/task_repo.go`：`~/.ait/tasks/<task-id>.json` CRUD
- [ ] `internal/server/store/run_repo.go`：`~/.ait/runs/<task-id>/<run-id>/run.json` 与 `result.json` 读写
- [ ] `internal/server/store/request_log.go`：统一 `requests.jsonl` append/read
- [ ] `internal/server/config/config.go`：`~/.ait/config.json` 全局配置（复用 store 基础能力）
- [ ] `internal/server/runner/runner.go`：增加 `Stop()` + 稳定 `RunWithCallback`
- [ ] `internal/server/server.go`：定义 `Server` 接口 + `New()` 构造函数
- [ ] `internal/server/task_service.go`：实现任务管理 facade 方法（调用 task.Store）
- [ ] `internal/server/run_service.go`：实现 `StartRun / StopRun / GetRunState`（调用 runner）
- [ ] `internal/server/event_bus.go`：实现内部 eventBus
- [ ] `internal/server/types.go`：定义 `Event / EventKind / RunState / RunID` 等 Server 层类型
- [ ] Server 单元测试：任务 CRUD、运行状态机、事件分发

**Step 2：TUI Client（依赖 Server 接口完成后）**

- [ ] `internal/tui/client.go`：持有 `server.Server`，封装 `tea.Cmd` 异步调用
- [ ] `internal/tui/model.go`：根 BubbleTea Model + 全局状态机（只依赖 `client.go`）
- [ ] `internal/tui/messages.go`：所有 `tea.Msg` 类型
- [ ] `internal/tui/styles.go`：lipgloss 样式常量
- [ ] `internal/tui/pages/contextbar.go`：Context Bar 组件（条件渲染）
- [ ] `internal/tui/pages/tasklist.go`：任务列表页（含 ◉ 运行状态展示）
- [ ] `internal/tui/pages/taskdetail.go`：任务详情页
- [ ] `internal/tui/pages/wizard.go`：三步弹窗向导（overlay）
- [ ] `internal/tui/pages/dashboard.go`：标准模式仪表盘（请求列表 + 实时指标）
- [ ] `internal/tui/pages/reqdetail.go`：请求详情页（含原始输入/输出）
- [ ] `cmd/ait/main.go`：`server.New()` → 启动 TUI
- [ ] 协议枚举：`openai-completions`、`openai-responses`、`anthropic-messages`
- [ ] 统一任务模式：`standard`、`turbo`、`integrity`
- [ ] 响应式布局（终端宽度自适应）
- [ ] `internal/display/` 退役，由 TUI 全面接管输出

---

### Phase 2 — Turbo 模式

**目标：** 将并发爬坡能力完整融入 SC 架构，详见 [Turbo 模式功能设计](turbo.md)

- [ ] `internal/server/turbo/engine.go`：爬坡调度（`Run / Stop`）
- [ ] `internal/server/turbo/strategy.go`：步进 & 终止策略
- [ ] `internal/server/turbo/types.go`：`TurboResult / LevelResult`
- [ ] `internal/server/run_service.go`：扩展 `StartRun` 支持 Turbo 模式（发布 `EventLevelDone`）
- [ ] `internal/server/store/request_log.go`：按统一 `requests.jsonl` 记录每级请求事实
- [ ] `internal/tui/pages/turbodash.go`：Turbo 仪表盘页（级别列表 + 当前级别指标）
- [ ] `internal/tui/pages/taskdetail.go`：从 `result.json` 展示 Turbo 运行结果（爬坡表格 + ASCII 曲线）
- [ ] `internal/server/report/turbo_renderer.go`：Turbo CSV/JSON 报告渲染

---

### Phase 3 — 接口完整性测试模式

**目标：** 实现内置 Suite + 自定义测试规则文件 + Case 结果页的 MVP 主流程

- [ ] `internal/server/integrity/engine.go`：Suite / Case 执行调度（`Run / Stop`）
- [ ] `internal/server/integrity/suite.go`：Suite 定义、Case 定义和内置 Suite 注册
- [ ] `internal/server/integrity/builtin.go`：内置 `smoke` / `full` Suite
- [ ] `internal/server/integrity/result.go`：`IntegrityResult / IntegrityCaseResult` 聚合
- [ ] `internal/server/assertion/loader.go`：声明式规则文件加载与版本校验
- [ ] `internal/server/assertion/compiler.go`：内置断言和 `integrity.rule_files` 合并、编译
- [ ] `internal/server/assertion/evaluator.go`：断言求值，支持 `exists / eq / contains / matches / gt` 等操作符
- [ ] `internal/server/assertion/fields.go`：受限 path 解析，区分 missing / null
- [ ] `internal/server/run_service.go`：扩展 `StartRun` 支持 `mode = "integrity"`，发布 Case 与 Assertion 事件
- [ ] `internal/server/store/request_log.go`：按统一 `requests.jsonl` 保存 Case 请求事实与断言上下文
- [ ] `internal/server/store/run_repo.go`：按统一 `result.json` 保存完整性测试最终业务结论
- [ ] `internal/tui/pages/integritydash.go`：完整性测试仪表盘（Suite / Case / 结论）
- [ ] `internal/tui/pages/integritydetail.go`：Case 详情与断言详情页
- [ ] `internal/server/report`：完整性测试 JSON 报告渲染
- [ ] `internal/mcp/server.go`：通过 `server.Server` 暴露 `ait.list_integrity_suites`、`ait.preview_integrity_suite`、`ait.validate_rule_file`、`ait.get_integrity_case_detail`
- [ ] 完整性测试单元测试：规则文件校验、断言求值、Suite 执行、结果落盘

---

### Phase 4 — 增强 & Web UI 接入准备

**目标：** 细节打磨，为 Web UI 预留接入点

- [ ] `internal/webui/`（骨架）：HTTP handler 接收请求 → 调用 `server.Server` → SSE/WS 推送 Event
- [ ] 多任务并发干扰提示完善
- [ ] 运行记录对比视图（同一任务不同 run）
- [ ] 任务详情页 `[c]` 复制最近运行摘要到剪贴板
- [ ] `ntcharts` 折线图替换 ASCII 折线图（Turbo 曲线）
- [ ] 终端尺寸变化自适应重绘
- [ ] 完善单元测试（TUI model 测试、server 集成测试、turbo strategy 测试、integrity/assertion 测试）

---

### 依赖变更汇总

```diff
  # go.mod 新增
+ charm.land/bubbletea/v2              # TUI 主框架
+ github.com/charmbracelet/bubbles     # 预制 UI 组件
+ github.com/charmbracelet/lipgloss    # 样式与布局
+ github.com/NimbleMarkets/ntcharts    # Phase 3 图表（可选）

  # go.mod 移除
- github.com/schollz/progressbar/v3    # 由 bubbles/progress 替代
- github.com/olekukonko/tablewriter    # 由 bubbles/table + lipgloss 替代
```
