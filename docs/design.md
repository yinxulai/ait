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
6. [新增数据结构与接口](#6-新增数据结构与接口)
7. [开发计划](#7-开发计划)

---

## 1. 概述

本文档描述 AIT 工具 v2.0 的三个核心功能迭代：

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

- 每个任务只绑定一个模型，保存协议、完整接口地址、模型、Prompt、标准模式或 Turbo 参数
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
  │  client/        │  │  report/         │
  │  prompt/        │  │  logger/ ...     │
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
    ListTasks() ([]types.Task, error)
    GetTask(id string) (*types.Task, error)
    CreateTask(cfg types.TaskConfig) (*types.Task, error)
    UpdateTask(id string, cfg types.TaskConfig) error
    DeleteTask(id string) error
    CopyTask(id string) (*types.Task, error)

    // 运行管理
    StartRun(taskID string) (RunID, error)
    StopRun(runID RunID) error
    GetRunState(runID RunID) (*RunState, error)

    // 事件订阅（解耦 UI 刷新，替代直接回调）
    // 返回事件 channel、取消订阅函数、错误
    Subscribe(runID RunID) (<-chan Event, CancelFunc, error)

    // 历史 & 报告
    GetHistory(taskID string) ([]RunSummary, error)
    GenerateReport(runID RunID, format ReportFormat) (path string, err error)

    // 生命周期
    Shutdown() error
}

// Event 是 Server 推送给订阅者的运行事件
type Event struct {
    RunID   RunID
    Kind    EventKind  // RequestDone | ProgressTick | LevelDone | RunComplete | RunFailed
    Payload any        // *RequestMetrics | *ProgressSnapshot | *LevelResult | *RunResult
}

type CancelFunc func()
```

### 3.3 各层职责边界

| 层 | 包 | 职责 |
|----|-----|------|
| **入口层** | `cmd/ait` | flag 解析；创建 Server；启动 TUI Client |
| **Server 层** | `internal/server` | 暴露业务 API；编排下层；管理运行状态；分发 Event |
| **执行层** | `internal/runner` `internal/turbo` | 并发请求执行；回调推送指标；**不感知 UI** |
| **协议层** | `internal/client` | OpenAI / Anthropic HTTP 客户端；不感知上层 |
| **持久化层** | `internal/task` `internal/config` | 任务 / 历史 / 配置的 JSON 文件读写 |
| **渲染层** | `internal/report` | JSON / CSV / Turbo 报告渲染；纯函数，无副作用 |
| **工具层** | `internal/prompt` `internal/network` `internal/logger` `internal/upload` | 公共工具，无业务依赖 |
| **TUI Client** | `internal/tui` | BubbleTea 状态机；**只依赖 server.Server 接口**；渲染终端 UI |
| **Web Client** _(Future)_ | `internal/webui` | HTTP/WS 桥接；**只依赖 server.Server 接口**；提供 Web API |

### 3.4 目录结构

```
cmd/
  ait/
    main.go             ← 入口：创建 server.New() + 启动 TUI

internal/
  server/               ← SERVICE LAYER
    server.go           ← Server 接口 + 实现 (New / Shutdown)
    task.go             ← 任务 CRUD 方法实现
    run.go              ← 运行启动 / 停止 / 状态管理
    event.go            ← Event / EventKind / RunState 类型
    types.go            ← RunID / RunSummary / ReportFormat 等 Server 层类型

  tui/                  ← TUI CLIENT
    client.go           ← 持有 server.Server；提供 tea.Cmd 包装（异步调用 server）
    model.go            ← 根 BubbleTea Model + 全局状态机
    messages.go         ← 所有 tea.Msg 类型
    styles.go           ← lipgloss 样式常量
    pages/
      tasklist.go       ← 任务列表页渲染 + 按键处理
      taskdetail.go     ← 任务详情页
      wizard.go         ← 新建 / 编辑弹窗向导（overlay，覆盖任务列表）
      dashboard.go      ← 标准模式仪表盘
      turbodash.go      ← Turbo 仪表盘
      reqdetail.go      ← 请求详情页
      contextbar.go     ← Context Bar 组件（条件渲染）

  runner/
    runner.go           ← 并发请求执行（RunWithCallback / Stop）

  turbo/
    engine.go           ← 并发爬坡调度（Run / Stop）
    strategy.go         ← 步进 & 终止策略
    types.go            ← TurboResult / LevelResult

  client/
    client.go           ← AI 客户端接口定义
    openai.go           ← OpenAI Completions / Responses
    anthropic.go        ← Anthropic Messages

  store/                 ← 统一持久化层（独立包，可被多个模块引用）
    store.go            ← 泛型 JSON 文件读写基类（Load / Save / 文件锁）
    task.go             ← TaskStore：~/.ait/tasks.json CRUD
    history.go          ← HistoryStore：~/.ait/history/<task-id>.json 读写

  task/
    task.go             ← 任务纯业务逻辑（不持有文件 I/O）

  config/
    config.go           ← ~/.ait/config.json 全局配置（使用 store 基类）

  report/
    report.go
    csv_renderer.go
    json_renderer.go
    turbo_renderer.go   ← (新增) Turbo 爬坡报告

  types/
    types.go            ← 跨层共享领域类型（Task / TaskConfig / Input / ResponseMetrics ...）

  prompt/               ← 字符串 / 文件 / 长度 Prompt 生成
  network/              ← IP 工具
  logger/               ← 请求日志
  upload/               ← 匿名数据上传
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
    │               └─ cb 内部: eventBus.Publish(Event{RequestDone})
    │               └─ 返回 runID
    │
    ├─ client.SubscribeCmd(runID) → tea.Cmd
    │       └─ server.Subscribe(runID) → eventCh
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
    ├─ server.GetRunState(runID)  ← 恢复当前快照（已完成请求列表）
    └─ server.Subscribe(runID)   ← 重新订阅，接收后续事件
```

### 3.6 Web UI 接入路径（未来）

新增 `internal/webui/` 包，直接复用同一个 `server.Server` 实例，无需修改 Server 层或任何下层模块：

```go
// internal/webui/handler.go  （示意）
func (h *Handler) startRun(w http.ResponseWriter, r *http.Request) {
    runID, _ := h.server.StartRun(r.PathValue("taskID"))
    eventCh, cancel, _ := h.server.Subscribe(runID)
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
+   main.go                 ← 仅做：flag解析 + server.New() + tui启动

+ internal/store/            ← 新增：独立持久化层（store.go / task.go / history.go）
+ internal/server/          ← 新增：SERVICE LAYER（核心新增）
+   server.go / task.go / run.go / event.go / types.go

  internal/tui/             ← 已有，重构：
+   client.go               ← 新增：持有 server.Server，包装 tea.Cmd
    model.go                ← 改造：不再直接调用 runner/task，改调 server
+   pages/                  ← 新增：各页面拆分为独立文件

  internal/runner/
    runner.go               ← 扩展：增加 Stop() + RunWithCallback 稳定化

  internal/turbo/
+   engine.go / strategy.go / types.go  ← 新增

  internal/task/
+   task.go                  ← 任务纯业务逻辑（不持有文件 I/O）

  internal/report/
+   turbo_renderer.go       ← 新增

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
启动带完整参数 ─→ 创建任务 + 自动 StartRun ─→ Running / TurboRunning

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

         [请求详情]  在 Running/TurboRunning 请求列表中选中后
                 ┌─────────────┐
                 │RequestDetail│
                 │  请求详情页   │
                 └──────┬──────┘
           [b/Esc 返回] │
                        └──────→ Running / TurboRunning
```

**多任务并发规则：**

- 支持多个任务同时在后台运行，不限数量
- 启动第二个任务时弹出提示："当前已有 N 个任务正在运行，多任务并行可能影响网络指标（DNS/TCP/TLS），`[y]` 继续 `[n]` 取消"
- 任务列表中所有运行中任务都带 `◉` 标记和实时进度
- 任务完成后自动更新对应任务的状态和历史记录，无论当前处于哪个页面

**后台运行规则：**

- 在仪表盘（Running / TurboRunning）按 `[b]` 或 `[Esc]` 可返回任务列表，测试继续在后台执行
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
║  │  测试模式    ○ 标准模式    ● Turbo 模式               │  ║
║  │               [←→ 切换]                               │  ║
║  │  ── 标准模式参数 ───────────────────────────────      │  ║
║  │  并发数 [  5  ]   请求总数 [ 100 ]                    │  ║
║  │  超时时间 [ 30s ]   流式模式  [✓ 开启]                │  ║
║  │  ── Turbo 模式参数 ─────────────────────────────      │  ║
║  │  初始并发 [  1  ]   最大并发  [  50  ]                │  ║
║  │  步进值 [  2  ]   每级请求数 [  30  ]                 │  ║
║  │  停止条件  成功率低于 [ 90% ] 或 延迟 > [ 10s ]       │  ║
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
║  │  流式模式    开启                                     │  ║
║  │  Prompt      你好，介绍一下你自己。 (长度: 12)        │  ║
║  │                                                       │  ║
║  │  保存任务到 ~/.ait/tasks.json  [✓]                    │  ║
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

#### 页面 8：请求详情页

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
| `b` / `Esc` | 仪表盘（标准/Turbo） | **后台运行**，返回任务列表（测试继续进行） |
| `b` / `Esc` | 请求详情页 | 返回仪表盘 |
| `Enter` | 仪表盘请求列表 | 进入请求详情页 |
| `↑` / `↓` | 仪表盘请求列表 / 请求详情 | 选择请求 / 滚动内容 |
| `←` / `→` | 请求详情页 | 切换上/下一条请求 |
| `Tab` / `Shift+Tab` | 向导 | 在输入项间切换焦点 |
| `↑` / `↓` | 任务列表、向导 | 上下选择 |
| `←` / `→` | 向导模式选择 | 切换选项 |
| `Enter` | 向导 | 确认 / 下一步 / 保存 |
| `Esc` | 所有页 | 返回上一步 / 取消 |
| `s` | 仪表盘（标准/Turbo） | 停止测试 |
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
| 请求详情页 | `[b/Esc] 返回仪表盘  [←→] 上/下一条请求` |

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

### 5.1 算法流程

```
初始化
  concurrency = TurboConfig.InitConcurrency   // 默认: 1
  step        = TurboConfig.StepSize          // 默认: 2
  levelReqs   = TurboConfig.LevelRequests     // 默认: 30（每级执行的请求数）

循环
  ① 用当前 concurrency 执行 levelReqs 个请求
     → 并发调用已有 runner.Runner（复用所有指标收集逻辑）

  ② 采集该级别指标（LevelResult）
      successRate = 成功请求数 / levelReqs
      avgTPS      = mean(输出 TPS)
      avgTTFT     = mean(TTFT)
      cacheHitRate = cachedInputTokens / inputTokens
      avgTotalTime = mean(总耗时)

  ③ 判断终止条件（任意一条满足即终止）
     a. successRate < TurboConfig.MinSuccessRate   → 服务降级
     b. avgTotalTime > TurboConfig.MaxLatency       → 延迟过高
     c. concurrency >= TurboConfig.MaxConcurrency   → 达到探测上限
     d. 用户按下 [s] 或 [m]                         → 手动终止

  ④ 如果未终止
     记录 LevelResult 到 TurboResult.Levels
     concurrency += step
     继续循环

结束
  最大稳定并发 = 最后一个通过终止检查的并发数
  TurboResult.MaxStableConcurrency = 最后一个 ✓ 稳定级别的并发数
  TurboResult.PeakTPS = 所有 ✓ 稳定级别中的最大 avgTPS
  生成 TurboResult
```

其中缓存命中率定义为“缓存命中的输入 token / 总输入 token”。
- OpenAI Completions / Responses：基于 usage 中的 `cached_tokens`
- Anthropic Messages：基于 usage 中的 `cache_read_input_tokens`
- 若响应未提供缓存统计字段，则该指标记为 `N/A`，不参与阈值判断

### 5.2 停止条件详解

```
type StopReason int

const (
    StopReasonLowSuccessRate  StopReason = iota // 成功率低于阈值
    StopReasonHighLatency                        // 延迟超过阈值
    StopReasonMaxConcurrency                     // 达到最大并发上限
    StopReasonManual                             // 用户手动停止
    StopReasonDegraded                           // 综合降级判断
)
```

**降级判断示例：**

```
并发=8: 成功率=96%, avgTPS=245, avgTTFT=312ms, cache=44%  → ✓ 通过（成功率>90%，延迟<10s）
并发=10: 成功率=84%                             → ✗ 停止（成功率 84% < 阈值 90%）
最大稳定并发 = 8
```

### 5.3 CLI 参数

Turbo 模式通过 `--turbo` 标志启用，新增以下参数：

```bash
# 启用 Turbo 模式
ait --protocol=openai-responses --endpoint=https://api.openai.com/v1/responses --model=gpt-4o --turbo

# 完整 Turbo 参数
ait --protocol=openai-responses --endpoint=https://api.openai.com/v1/responses --model=gpt-4o --turbo \
  --turbo-init-concurrency=1 \   # 初始并发数（默认: 1）
  --turbo-max-concurrency=50 \   # 最大探测并发数（默认: 50）
  --turbo-step=2 \               # 每级步进值（默认: 2）
  --turbo-level-requests=30 \    # 每级执行的请求数（默认: 30）
  --turbo-min-success-rate=0.9 \ # 成功率低于此值停止（默认: 0.9）
  --turbo-max-latency=10s        # 延迟超过此值停止（默认: 10s）
```

**与现有参数的关系：**

- `--concurrency` 在 Turbo 模式下**被忽略**（Turbo 自己控制并发）
- `--count` 在 Turbo 模式下表示**每级**的请求数（等同于 `--turbo-level-requests`，优先级低于后者）
- `--protocol` 允许值为 `openai-completions`、`openai-responses`、`anthropic-messages`
- `--endpoint` 必须填写完整接口地址，例如 `https://api.openai.com/v1/responses`
- 其他参数（protocol、endpoint、apiKey、model、stream、timeout 等）正常生效
- 单个任务只接受一个 `--model`；如果要测试多个模型，应拆分成多个任务

**与任务管理的关系：**

- `--task=<id>` 直接加载已保存任务并进入详情页或直接运行
- 通过完整 CLI 参数启动时，AIT 自动创建任务并立即调用 `StartRun`，直接进入 Running / TurboRunning 仪表盘
- Turbo 运行完成后，其结果会自动追加到对应任务的历史记录中

### 5.4 Turbo 报告格式

**JSON 报告（新增字段）：**

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

**CSV 报告（新增 turbo 爬坡汇总表）：**

```
protocol,concurrency,success_rate,avg_tps,peak_tps,avg_ttft,cache_hit_rate,avg_total_time,stable
openai-responses,1,1.00,31.2,38.5,89ms,0.00,0.82s,true
openai-responses,2,1.00,62.5,74.1,91ms,0.18,0.84s,true
openai-responses,4,0.99,121.3,145.2,98ms,0.26,0.91s,true
openai-responses,6,0.98,178.4,201.3,124ms,0.33,1.08s,true
openai-responses,8,0.96,245.3,280.1,312ms,0.44,1.51s,true
openai-responses,10,0.84,198.1,234.5,892ms,0.12,4.23s,false
```

---

## 6. 新增数据结构与接口

### 6.1 TaskDefinition

```go
// internal/types/types.go 新增

// TaskDefinition 可重复执行的测试任务定义
type TaskDefinition struct {
  ID             string          `json:"id"`
  Name           string          `json:"name"`
  Input          Input           `json:"input"`
  CreatedAt      time.Time       `json:"created_at"`
  UpdatedAt      time.Time       `json:"updated_at"`
  LastRunAt      *time.Time      `json:"last_run_at,omitempty"`
  LastRunSummary *TaskRunSummary `json:"last_run_summary,omitempty"`
}
```

其中 `Input.Model` 为单个模型标识，不允许逗号分隔列表。
其中 `Input.Protocol` 允许值为 `openai-completions`、`openai-responses`、`anthropic-messages`。
其中 `Input.EndpointURL` 必须是完整接口地址，例如：
- `openai-completions` → `https://api.openai.com/v1/chat/completions`
- `openai-responses` → `https://api.openai.com/v1/responses`
- `anthropic-messages` → `https://api.anthropic.com/v1/messages`

### 6.2 TaskRunSummary

```go
// TaskRunSummary 单次任务运行后的摘要信息
type TaskRunSummary struct {
  RunID                 string        `json:"run_id"`
  TaskID                string        `json:"task_id"`
  Mode                  string        `json:"mode"`
  Status                string        `json:"status"`
  Protocol              string        `json:"protocol"`
  Model                 string        `json:"model"`
  StartedAt             time.Time     `json:"started_at"`
  FinishedAt            time.Time     `json:"finished_at"`
  SuccessRate           float64       `json:"success_rate"`
  AvgTTFT               time.Duration `json:"avg_ttft"`
  AvgTPS                float64       `json:"avg_tps"`
  CacheHitRate          float64       `json:"cache_hit_rate"`
  MaxStableConcurrency  int           `json:"max_stable_concurrency,omitempty"`
  ReportJSONPath        string        `json:"report_json_path,omitempty"`
  ReportCSVPath         string        `json:"report_csv_path,omitempty"`
  ErrorSummary          string        `json:"error_summary,omitempty"`
}
```

### 6.3 TurboConfig

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

  ### 6.4 Input 扩展

```go
// internal/types/types.go 扩展 Input

type Input struct {
    // ... 现有字段 ...

    Protocol string // openai-completions | openai-responses | anthropic-messages
    EndpointURL string // 完整接口地址，例如 https://api.openai.com/v1/responses

    // Turbo 模式
    Turbo       bool        // 是否启用 Turbo 模式
    TurboConfig TurboConfig // Turbo 配置（Turbo=true 时生效）
}
```

### 6.5 Server 层类型

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
    Mode         string       // "standard" | "turbo"
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
    // 聚合实时指标（标准模式 / Turbo 当前级别）
    AvgTPS       float64
    AvgTTFT      time.Duration
    SuccessRate  float64
    CacheHitRate float64
    // 最终结果（完成后填充）
    StandardResult *types.ReportData
    TurboResult    *types.TurboResult
    ErrorMsg       string
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
    EventRequestDone  EventKind = "request_done"   // 单条请求完成
    EventProgressTick EventKind = "progress_tick"  // 500ms 聚合快照
    EventLevelDone    EventKind = "level_done"      // Turbo 一级完成
    EventRunComplete  EventKind = "run_complete"    // 全部完成
    EventRunFailed    EventKind = "run_failed"      // 运行出错
)
```

### 6.6 TUI 消息类型

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

### 6.7 Runner 接口扩展

```go
// internal/runner/runner.go — Server 层内部使用，TUI 不直接调用

// RequestDoneCallback 每个请求完成后的回调（由 server/run.go 包装为 Event）
type RequestDoneCallback func(metrics *ResponseMetrics, index int, err error)

// RunWithCallback 运行测试，每个请求完成后调用 cb（线程安全）
func (r *Runner) RunWithCallback(cb RequestDoneCallback) (*types.ReportData, error)

// Stop 异步停止正在进行的测试
func (r *Runner) Stop()
```
```

### 6.8 任务与全局配置持久化

```go
// internal/config/config.go

type Config struct {
  SaveAPIKey         bool   `json:"save_api_key"`
  LastSelectedTaskID string `json:"last_selected_task_id,omitempty"`
  DefaultProtocol    string `json:"default_protocol,omitempty"` // openai-completions | openai-responses | anthropic-messages
}

func Load() (*Config, error)   // 从 ~/.ait/config.json 加载
func (c *Config) Save() error  // 保存到 ~/.ait/config.json

// internal/store/store.go
// 泛型基类，提供 JSON 文件安全读写

type JSONStore[T any] struct {
  path string
}

func NewJSONStore[T any](path string) *JSONStore[T]
func (s *JSONStore[T]) Load() (T, error)
func (s *JSONStore[T]) Save(v T) error  // 写入前加文件锁

// internal/store/task.go

type TaskStore struct {
  Tasks []types.TaskDefinition `json:"tasks"`
}

func LoadTasks() (*TaskStore, error)                   // 从 ~/.ait/tasks.json 加载
func (s *TaskStore) Save() error                       // 保存到 ~/.ait/tasks.json
func (s *TaskStore) Upsert(task types.TaskDefinition)  // 新建或更新任务
func (s *TaskStore) Delete(taskID string) error

// internal/store/history.go

func AppendRun(taskID string, run types.TaskRunSummary) error
func LoadHistory(taskID string, limit int) ([]types.TaskRunSummary, error)
```

---

## 7. 开发计划

> **约定：** 严格遵守 SC 分层原则——先建好 Server 接口，再实现 TUI Client；所有 UI 层代码仅 import `internal/server`，不直接 import `runner` / `task` / `report` 等下层包。

### Phase 1 — Server 层 + TUI 基础框架

**目标：** 建立 SC 架构骨架，跑通"创建任务 → 运行 → 看到进度 → 查看结果"主流程

**Step 1：Server 层（先行）**

- [ ] `internal/types/types.go`：补充 `Task`、`TaskConfig`、`TurboConfig`、`ReportData` 等领域类型
- [ ] `internal/store/store.go`：泛型 `JSONStore[T]` 基类（文件锁 + Load/Save）
- [ ] `internal/store/task.go`：`TaskStore`，`~/.ait/tasks.json` CRUD
- [ ] `internal/store/history.go`：`HistoryStore`，`~/.ait/history/<task-id>.json` 读写
- [ ] `internal/config/config.go`：`~/.ait/config.json` 全局配置（复用 `store.JSONStore`）
- [ ] `internal/runner/runner.go`：增加 `Stop()` + 稳定 `RunWithCallback`
- [ ] `internal/server/server.go`：定义 `Server` 接口 + `New()` 构造函数
- [ ] `internal/server/task.go`：实现任务 CRUD 方法（调用 task.Store）
- [ ] `internal/server/run.go`：实现 `StartRun / StopRun / GetRunState`（调用 runner）
- [ ] `internal/server/event.go`：`Event / EventKind / RunState / RunID` 类型 + 内部 eventBus
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
- [ ] 响应式布局（终端宽度自适应）
- [ ] `internal/display/` 退役，由 TUI 全面接管输出

---

### Phase 2 — Turbo 模式

**目标：** 将并发爬坡能力完整融入 SC 架构

- [ ] `internal/turbo/engine.go`：爬坡调度（`Run / Stop`）
- [ ] `internal/turbo/strategy.go`：步进 & 终止策略
- [ ] `internal/turbo/types.go`：`TurboResult / LevelResult`
- [ ] `internal/server/run.go`：扩展 `StartRun` 支持 Turbo 模式（发布 `EventLevelDone`）
- [ ] `internal/tui/pages/turbodash.go`：Turbo 仪表盘页（级别列表 + 当前级别指标）
- [ ] `internal/tui/pages/taskdetail.go`：扩展支持 Turbo 运行结果展开（爬坡表格 + ASCII 曲线）
- [ ] `internal/report/turbo_renderer.go`：Turbo CSV/JSON 报告渲染
- [ ] Turbo 运行历史写回任务摘要

---

### Phase 3 — 增强 & Web UI 接入准备

**目标：** 细节打磨，为 Web UI 预留接入点

- [ ] `internal/webui/`（骨架）：HTTP handler 接收请求 → 调用 `server.Server` → SSE/WS 推送 Event
- [ ] 多任务并发干扰提示完善
- [ ] 运行记录对比视图（同一任务不同 run）
- [ ] 任务详情页 `[c]` 复制最近运行摘要到剪贴板
- [ ] `ntcharts` 折线图替换 ASCII 折线图（Turbo 曲线）
- [ ] 终端尺寸变化自适应重绘
- [ ] 完善单元测试（TUI model 测试、server 集成测试、turbo strategy 测试）

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
