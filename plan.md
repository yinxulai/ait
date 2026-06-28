## Plan: tview TUI 基础布局与交互（第一版）

### TL;DR
用 tview v0.42.0 从零构建 TUI，以 **Page 接口** 为第一公民，Header + 2:2:4 分栏 Content + Footer，Tab 切换高亮边框，模拟数据严格对齐项目真实类型。

---

### 核心架构：Page 抽象 + PageRouter + Overlay

**设计原则**：每个页面是一个实现了 `Page` 接口的结构体，自包含布局构建、focus 目标、Enter 导航逻辑和**按键处理**。`HandleKey` 仅供页面拦截快捷键/命令键（Tab、Esc、Enter 等）；`HandleKey` 返回 `nil` 表示已消费，返回 `event` 则透传给 tview 原生路由——文本输入（InputField 等）由 tview 自动分发到焦点原语，无需 Page 层干预。PageRouter 管理导航栈和 overlay，按键由 `ActivePage()` 分发——不再需要全局层读取导航栈状态。

```
┌──────────────────────────────────────────────────┐
│  Page 接口 (page.go)                             │
│  ├─ Name() string                                │
│  ├─ Primitive() tview.Primitive                  │
│  ├─ FocusTarget() tview.Primitive               │
│  ├─ HandleKey(event, router) *tcell.EventKey  │
│  ├─ OnActivate()   — v1 空实现，预留              │
│  └─ OnDeactivate() — v1 空实现，预留              │
│                                                  │
│  实现者:                                         │
│  ├─ *MainPage       (主布局 3-panel + 导航键 + r/s/R 操作键) │
│  ├─ *TaskDetailPage (Esc→返回)                   │
│  ├─ *StandardDashboardPage (Esc→返回 / s=停止 / 列表原生滚动) │
│  ├─ *RequestDetailPage (Esc→返回)                │
│  └─ *WarningPage    (Esc→隐藏 overlay)           │
└──────────────────────────────────────────────────┘

PageRouter (router.go):
  NavigateTo(Page)  → push stack, hide/show
  GoBack()          → pop stack, show prev
  ShowOverlay(Page) → overlay 不干扰栈
  HideOverlay()     → 恢复栈顶
  ActivePage() Page → 优先返回 overlay，其次栈顶
  StackDepth() int

全局 keyhandler 精简为单行分派:
  cur := router.ActivePage(); return cur.HandleKey(event, router)
```

---

### 数据反应管道：channel → updateLoop → QueueUpdateDraw

```
Simulator(v1) / SubscribeRunEvents(v2)
        │
        ▼
  taskUpdates    <-chan []types.TaskOverview
  runUpdates     <-chan []types.TaskRunSummary
  requestUpdates <-chan []types.RequestMetrics
        │
        ▼
  startUpdateLoop() goroutine
        │  select { case data := <-ch: }
        ▼
  app.QueueUpdateDraw(func() { mainPage.Refresh*(data) })
        │
        ▼
  MainPage.RefreshTasks/RefreshRuns/RefreshRequests
        │  Clear + Rebuild + Rebind keyboard
        ▼
  tview.Draw()  ← 自动触发重绘

---

### 数据结构对齐（mock.go 将引用 `server/types`）

#### 左 Panel — 任务列表
数据来源：`server.types.TaskOverview`

渲染字段：`Name`(main)、`Input.RunMode()` + `LatestRun.SuccessRate` + `LatestRun.AvgTPS`(secondary)
每行 `AddItem(main, secondary, 0, nil)` 格式：
```
main:  "高并发压测-gpt4"
sec:   "standard  | SR 98.5%  | TPS 245"
main:  "缓存命中率测试"
sec:   "turbo     | SR 96.2%  | TPS 512"
main:  "新建任务-未运行"
sec:   "standard  | -"
```

> **注意**：`Input.RunMode()` 是方法不是字段，它能自动推导默认值：Turbo→"turbo"，Integrity→"integrity"，否则→"standard"。`LatestRun` 为 `nil` 时 secondary 显示 `"-"`。

#### 中 Panel — 历史记录
数据来源：`types.TaskRunSummary`（`RunID` 字段本身是 `string`，无需 `string(RunID)` 转换）

每行 `AddItem(main, secondary, 0, nil)` 格式：
```
main:  "run-a1b2c3"  （或显示简短任务名）
sec:   "turbo    | 06-28 14:22 | 45s  | SR 97.8% | TPS 342"
main:  "run-d4e5f6"
sec:   "standard | 06-28 13:15 | 120s | SR 99.1% | TPS 215"
```

#### 右 Panel 上部 — 统计

两种状态共用布局，差异在标题和进度指示：

**状态 A：实时运行中（n%）**
- 标题：`"Statistics & Requests [Running 67%]"`
- 统计文本首行新增进度条：
  ```
  ████████████░░░░░░░░  67% (16,750 / 25,000)  预计剩余: 18s
  ─────────────────────────────────────────────────
  总请求数: 25,000    成功率: 98.5%
  完成: 16,750  失败: 375
  平均 TPS: 245.3     缓存命中率: 45.2%
  P50 TTFT: 120ms     P99 TTFT: 850ms
  RPM: 3,420           TPM: 1,280,000
  ```
- 进度条用 Unicode block chars `█` + `░` 渲染，宽度自适应

**状态 B：已完成（100%）**
- 标题：`"Statistics & Requests [Completed 100%]"`
- 统计文本：
  ```
  ████████████████████  100% (25,000 / 25,000)  耗时: 2m45s
  ─────────────────────────────────────────────────
  总请求数: 25,000    成功率: 98.5%
  完成: 24,625  失败: 375
  平均 TPS: 245.3     缓存命中率: 45.2%
  P50 TTFT: 120ms     P99 TTFT: 850ms
  RPM: 3,420           TPM: 1,280,000
  ```

#### 右 Panel 下部 — 请求记录
数据来源：`types.RequestMetrics`

字段：`Index`、`Success`、`TotalTime`、`TTFT`、`TPS`、`PromptTokens`、`CompletionTokens`、`CachedTokens`、`CacheHitRate`、`DNSTime`、`ConnectTime`、`TLSTime`、`TargetIP`、`ErrorMessage`、`Level`

每行渲染格式：
```
#001 ✅  234ms  TTFT 45ms   TPS 512  | PT 1024  CT 256  CH 45%
#002 ❌  5.2s   TTFT -      TPS 0    | ERR: timeout
#003 ✅  189ms  TTFT 32ms   TPS 678  | PT 2048  CT 512  CH 72%
```

---

### Phase 1: Page 抽象 + 状态管理

**Step 1: `internal/tui/page.go` — Page 接口定义**
```go
// panelIndex 表示主布局中当前激活的 panel
type panelIndex int

const (
    panelTasks   panelIndex = iota // 左 panel — 任务列表
    panelHistory                   // 中 panel — 运行历史
    panelStats                     // 右 panel — 统计+请求
)

type Page interface {
    Name() string
    Primitive() tview.Primitive
    FocusTarget() tview.Primitive
    HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey
    OnActivate()
    OnDeactivate()
}
```

**Step 2: `internal/tui/tui.go` — 入口 + tuiState + 数据管道**

- 包级变量 `version string`
- `SetVersion(v string)` 存储版本
- 尺寸常量：`minTermWidth=120`，`minTermHeight=16`

```go
type tuiState struct {
    app         *tview.Application
    router      *PageRouter
    srv         server.Server      // 服务层接口，供各 Page 执行操作
    mainPage    *MainPage
    warningPage *WarningPage
    stopCh      chan struct{}      // 通知 updateLoop 退出
}
```
不再持有 mock 数据字段——数据所有权归 `MainPage`。不再持有 `HasOverlay`/`StackDepth` 检测——按键分派到 `ActivePage()` 自动解决上下文。

**数据管道**（同文件 `tui.go`）：

```go
// NewSimulator 返回三个只读 channel，v1 模拟数据推送
// v2 时替换为 server.SubscribeRunEvents() 获取实时事件流
func NewSimulator() (
    tasksCh   <-chan []types.TaskOverview,
    runsCh    <-chan []types.TaskRunSummary,
    reqsCh    <-chan []types.RequestMetrics,
)

func (s *tuiState) startUpdateLoop(
    tasksCh <-chan []types.TaskOverview,
    runsCh  <-chan []types.TaskRunSummary,
    reqsCh  <-chan []types.RequestMetrics,
) {
    go func() {
        for {
            select {
            case tasks := <-tasksCh:
                s.app.QueueUpdateDraw(func() { s.mainPage.RefreshTasks(tasks) })
            case runs := <-runsCh:
                s.app.QueueUpdateDraw(func() { s.mainPage.RefreshRuns(runs) })
            case reqs := <-reqsCh:
                s.app.QueueUpdateDraw(func() { s.mainPage.RefreshRequests(reqs) })
            case <-s.stopCh:
                return
            }
        }
    }()
}

func (s *tuiState) stop() { close(s.stopCh) }
```

**`NewSimulator` 行为**（v1）：
- 构造时立即推入初始 mock 数据到三个 channel（buffer=1）
- 后续可扩展为定时推送更新（v1 不需要，初始数据足够）

**Step 3: `internal/tui/main_page.go` — MainPage（主布局 + 按键全控 + 数据刷新）**

结构体：
```go
type MainPage struct {
    app         *tview.Application  // 用于 SetFocus
    srv         server.Server       // 服务层接口，用于 StartRun/StopRun 等操作
    version     string
    header      *tview.TextView
    footer      *tview.TextView
    taskList    *tview.List
    historyList *tview.List
    statsPanel  *tview.Flex
    statsText   *tview.TextView
    requestList *tview.List
    activePanel panelIndex
    root        *tview.Flex
    // 数据（用于 Enter 导航查找 + Refresh 重建）
    tasks       []types.TaskOverview
    runs        []types.TaskRunSummary
    requests    []types.RequestMetrics
    runRequests map[int][]types.RequestMetrics  // key = runs 切片索引，v1 静态数据下稳定
    totalReqs   int
}
```

`NewMainPage(app *tview.Application, srv server.Server, version string) *MainPage`：构建全部 primitive + 布局。注意 `app` 参数：MainPage 需要 `app.SetFocus` 来切换 focus 目标；`srv` 参数：供操作按键（r=运行, s=停止）调用 Server API。

**实现 `Page` 接口**：
```go
func (m *MainPage) HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey {
    switch event.Key() {
    case tcell.KeyTab:
        m.nextPanel()
        m.app.SetFocus(m.FocusTarget())
        return nil
    case tcell.KeyBacktab:
        m.prevPanel()
        m.app.SetFocus(m.FocusTarget())
        return nil
    case tcell.KeyEnter:
        target := m.EnterAction()
        if target != nil {
            router.NavigateTo(target)
            m.app.SetFocus(target.FocusTarget())
        }
        return nil
    case tcell.KeyEsc:
        m.app.Stop()
        return nil
    }
    // 操作按键：根据当前 activePanel 分发
    switch event.Rune() {
    case 'r':                           // 运行任务
        if m.activePanel == panelTasks {
            idx := m.taskList.GetCurrentItem()
            if idx >= 0 {
                go func() { _, _ = m.srv.StartRun(m.tasks[idx].ID) }()
            }
        }
        return nil
    case 's':                           // 停止运行
        if m.activePanel == panelHistory {
            idx := m.historyList.GetCurrentItem()
            if idx >= 0 && m.runs[idx].Status == "running" {
                go func() { _ = m.srv.StopRun(server.RunID(m.runs[idx].RunID)) }()
            }
        }
        return nil
    case 'R':                           // 刷新列表
        switch m.activePanel {
        case panelTasks:
            go func() { tasks, _ := m.srv.ListTasks(); m.app.QueueUpdateDraw(func() { m.RefreshTasks(tasks) }) }()
        case panelHistory:
            go func() {
                if idx := m.taskList.GetCurrentItem(); idx >= 0 {
                    runs, _ := m.srv.ListTaskRunHistory(m.tasks[idx].ID, 50)
                    m.app.QueueUpdateDraw(func() { m.RefreshRuns(runs) })
                }
            }()
        }
        return nil
    }
    return event  // ↑↓ 等透传给 tview List
}
```

**数据刷新方法**（供 `startUpdateLoop` 调用）：
```go
func (m *MainPage) RefreshTasks(tasks []types.TaskOverview)   // 清空 taskList + AddItem 重建
func (m *MainPage) RefreshRuns(runs []types.TaskRunSummary)    // 清空 historyList + AddItem 重建
func (m *MainPage) RefreshRequests(reqs []types.RequestMetrics) // 更新 statsText（从 reqs 内部聚合统计）+ 清空 requestList 重建
```
- 重建后保留当前选中索引（若旧索引仍有效），否则选 0
- 重建后调 `m.updatePanelStyles()` 恢复边框/选中态样式
- statsText 的聚合值（TotalReqs/SuccessRate/AvgTPS 等）从 reqs 数据内部实时计算

Panel 切换方法：
- `(m *MainPage) nextPanel()` / `prevPanel()`：循环 `activePanel`，调 `updatePanelStyles()`
- `(m *MainPage) updatePanelStyles()`：
  - 边框切换：激活→黄加粗，非激活→灰色
  - 选中背景色同步：激活 panel→DarkBlue，非激活→Gray
  - Footer 更新：`m.updateFooter()`
- `(m *MainPage) updateFooter()`：根据 `activePanel` 设 footer 文本：
  - `panelTasks`：`"Tab=切换面板  ↑↓=选择  Enter=详情  r=运行  R=刷新  Esc=退出"`
  - `panelHistory`：`"Tab=切换面板  ↑↓=选择  Enter=仪表盘  s=停止  Esc=退出"`
  - `panelStats`：`"Tab=切换面板  ↑↓=选择  Enter=请求详情  Esc=退出"`

数据填充（初始 + Refresh 共用）：
- `(m *MainPage) PopulateData(...)`：外部注入 mock 数据，填充三个 List + statsText
- 内部调 `formatTaskLine`/`formatRunLine`/`formatReqLine`/`formatStatsWithProgress`

**Enter 导航**（自包含）：
- `(m *MainPage) EnterAction() Page`：
  - `panelTasks`：`idx := m.taskList.GetCurrentItem(); if idx >= 0 { return NewTaskDetailPage(&m.tasks[idx]) }`
  - `panelHistory`：`idx := m.historyList.GetCurrentItem(); if idx >= 0 { return NewRunDashboardPage(m.srv, &m.runs[idx], m.runRequests[idx]) }`
  - `panelStats`：`idx := m.requestList.GetCurrentItem(); if idx >= 0 { return NewRequestDetailPage(&m.requests[idx]) }`
  - 返回 `nil` 表示无操作

### Phase 2: 模拟数据

**Step 4: `internal/tui/mock.go`** *并行于 Phase 1*
- 所有 `format*` 函数保持独立（不依赖 `tuiState`）：`generateTaskOverviews()`, `generateRunSummaries()`, `generateRequestMetrics(n)`, `generateRunRequests(runIndex)`, `formatStatsWithProgress(...)`, `formatTaskLine(...)`, `formatRunLine(...)`, `formatReqLine(...)`, `renderProgressBar(...)`
- **移除** `(s *tuiState) populateMockData()` —— 数据注入改由 `MainPage.PopulateData(...)` 完成

---

### Phase 3: PageRouter（Page 栈 + Overlay）

**Step 5: `internal/tui/router.go`**

```go
type PageRouter struct {
    pages   *tview.Pages
    stack   []Page
    overlay Page
}

func NewPageRouter(pages *tview.Pages) *PageRouter
func (r *PageRouter) NavigateTo(page Page)
func (r *PageRouter) GoBack()
func (r *PageRouter) ShowOverlay(page Page)
func (r *PageRouter) HideOverlay()
func (r *PageRouter) ActivePage() Page   // overlay != nil ? overlay : stack top
func (r *PageRouter) StackDepth() int
```

**关键变更**：
- `stack` 从 `[]string` 变为 `[]Page` —— 导航状态与页面内容一体
- `GoBack()` 不再返回 `(string, bool)` —— 调用方通过 `ActivePage()` 获取返回后的页面
- **`ActivePage()` 是全局 keyhandler 的唯一依赖** —— 优先返回 overlay，其次栈顶
- Focus 不由 Router 管理 —— 由各 Page 的 `HandleKey` 自行调 `app.SetFocus()`

---

### Phase 4: 详情页（Page 实现者 + HandleKey）

**Step 6: `internal/tui/page_task_detail.go`**
- `TaskDetailPage` 结构体：`task`, `root *tview.Flex`, `tv *tview.TextView`
- `NewTaskDetailPage(task *types.TaskOverview) *TaskDetailPage`
- `Name()→"task_detail"`, `Primitive()→p.root`, `FocusTarget()→p.tv`
- `HandleKey`: `KeyEsc` → `router.GoBack()` + return nil；其余 `return event`（透传）
- `OnActivate/OnDeactivate` → 空
- 布局：Header + TextView(全量字段) + Footer。`p.tv.SetFocusable(true)`

**Step 7: `internal/tui/pages/page_standard_dashboard.go`**
- `StandardDashboardPage` 结构体：`srv server.Server`, `run`, `reqs`, `root *tview.Flex`, `rList *tview.List`
- `NewStandardDashboardPage(srv server.Server, run, reqs) *StandardDashboardPage`
- `Name()→"standard_dashboard"`, `Primitive()→p.root`, `FocusTarget()→p.rList`
- `HandleKey`: `KeyEsc` → `router.GoBack()` + return nil；**其余 `return event`**（↑↓/PgUp/PgDn 由 tview List 原生处理）；`s` 键 → 异步调 `p.srv.StopRun()`；`g` 键（v2）→ 生成报告
- `OnActivate/OnDeactivate` → 空（v2: OnActivate 调 `srv.SubscribeRunEvents()`，OnDeactivate 取消订阅）
- 布局：Header(`"Run Dashboard - {RunID[:8]} [Completed 100%]"`) + 内容（上半 statsText + 分隔线 + 下半 `p.rList`）+ Footer(`"s=停止  g=报告  ↑↓=选择  Esc=返回"`)
- `p.rList` 样式：`SetSelectedBackgroundColor(DarkBlue)` + `SetSelectedTextColor(White)`

**Step 8: `internal/tui/page_request_detail.go`**
- `RequestDetailPage` 结构体：`req`, `root *tview.Flex`, `tv *tview.TextView`
- `NewRequestDetailPage(req) *RequestDetailPage`
- `Name()→"request_detail"`, `Primitive()→p.root`, `FocusTarget()→p.tv`
- `HandleKey`: `KeyEsc` → `router.GoBack()` + return nil；其余 `return event`

**Step 9: `internal/tui/page_warning.go`**
- `WarningPage` 结构体：`root`, `header`, `footer`, `textView`
- `NewWarningPage(version) *WarningPage`
- `Name()→"warning"`, `Primitive()→p.root`, `FocusTarget()→p.textView`
- `HandleKey`: `KeyEsc` → `router.HideOverlay()` + return nil；其余 `return nil`（吃掉所有输入）
- `(p *WarningPage) UpdateText(w, h int)`：刷新尺寸提示文本

---

### Server API → TUI Page 映射

TUI 持有 `server.Server` 接口引用（与 web 层同等待遇）。以下映射约定各 Page 可调用的方法及触发方式：

| Server 方法 | 触发方式 | 页面 | 用途 |
|------------|---------|------|------|
| `ListTasks()` | `R` 刷新 / 管道刷新 | MainPage (左 panel) | 刷新任务列表 |
| `ListTaskRunHistory(taskID, limit)` | Enter 进入详情 / 管道刷新 | MainPage (中 panel) | 刷新运行历史 |
| `StartRun(taskID)` | `r` 按键 | MainPage (左 panel 选中) | 启动任务运行 |
| `StopRun(runID)` | `s` 按键 | MainPage (中 panel 选中) | 停止运行中的任务 |
| `GetRunState(runID)` | 进入仪表盘 / 管道刷新 | StandardDashboardPage | 获取运行实时状态 |
| `SubscribeRunEvents(runID)` | OnActivate（v2） | StandardDashboardPage | 订阅实时事件流 |
| `GetTask(id)` | Enter 进入详情（v2） | TaskDetailPage | 获取任务完整定义 |
| `GenerateRunReport(runID, format)` | `g` 按键（v2） | StandardDashboardPage | 生成运行报告 |
| `CreateTask(cfg)` | `n` 按键 → Wizard（v2） | WizardPage | 新建任务 |
| `UpdateTask(id, cfg)` | `e` 按键 → Wizard（v2） | WizardPage | 编辑任务 |
| `DeleteTask(id)` | `d` 按键（v2） | MainPage | 删除任务 |
| `DuplicateTask(id)` | `y` 按键（v2） | MainPage | 复制任务 |
| `ValidateTaskConfig(cfg)` | Wizard 保存前（v2） | WizardPage | 配置预验证 |
| `GetAppConfig()` | 配置页（v2） | ConfigPage | 获取全局配置 |
| `UpdateProxyURL(url)` | 配置页（v2） | ConfigPage | 更新代理 |
| `ListProtocols()` | 元数据刷新（v2） | MainPage | 协议列表 |
| `ListIntegritySuites(protocol)` | 元数据（v2） | IntegrityPage | 完整性套件 |
| `GetIntegritySuite(protocol, id)` | 元数据（v2） | IntegrityPage | 套件详情 |
| `Context()` | 全局 | tuiState | 上下文传递 |
| `Shutdown(timeout)` | `Esc` 退出时 | tuiState | 优雅关闭 |

**v1 实现范围**：`ListTasks`、`ListTaskRunHistory`、`StartRun`、`StopRun`（通过按键触发，与 Simulator 管道并行）+ `Context`/`Shutdown`（生命周期）。其余标记 v2。

**srv 传递方式**：
- `tuiState` 持有 `srv server.Server`
- `MainPage` 通过 `NewMainPage(app, srv, version)` 注入
- 详情页通过构造函数注入（`NewRunDashboardPage(srv, run, reqs)` 等）
- `PageRouter` 不持有 `srv`——各 Page 自管理

---

### Phase 5: 全局按键 + 尺寸检测

**Step 10: keybindings（合并到 `internal/tui/tui.go`）**

全局层不再读取导航栈——仅做分派 + 尺寸检测：

```go
func (s *tuiState) setupKeybindings() {
    s.app.SetInputCapture(s.globalKeyHandler)
    s.app.SetBeforeDrawFunc(s.beforeDrawCheck)
}

// globalKeyHandler: 单行分派，无栈判断
func (s *tuiState) globalKeyHandler(event *tcell.EventKey) *tcell.EventKey {
    cur := s.router.ActivePage()
    if cur != nil {
        return cur.HandleKey(event, s.router)
    }
    return event
}
```

**`beforeDrawCheck`**：
```go
func (s *tuiState) beforeDrawCheck(screen tcell.Screen) bool {
    w, h := screen.Size()
    tooSmall := w < minTermWidth || h < minTermHeight
    overlayActive := s.router.ActivePage() != nil && s.router.ActivePage().Name() == "warning"

    if tooSmall && !overlayActive {
        s.warningPage.UpdateText(w, h)
        go func() {
            s.app.QueueUpdateDraw(func() {
                s.router.ShowOverlay(s.warningPage)
                s.app.SetFocus(s.warningPage.FocusTarget())
            })
        }()
    } else if !tooSmall && overlayActive {
        go func() {
            s.app.QueueUpdateDraw(func() {
                s.router.HideOverlay()
                cur := s.router.ActivePage()
                if cur != nil { s.app.SetFocus(cur.FocusTarget()) }
            })
        }()
    }
    return true   // 允许 tview 正常渲染
}
```

**关键简化**：
- `globalKeyHandler` 从 30 行缩减到 6 行
- 不再 import `HasOverlay`/`StackDepth`/`isOnMainPage` 等检测
- 每个 Page 自行定义自己关心的快捷键——MainPage 处理 Tab/Enter/Esc 导航 + `r`(运行)/`s`(停止)/`R`(刷新) 操作键，详情页只处理 Esc，WarningPage 吃掉所有输入
- WarningPage 的 Esc → `HideOverlay()` 恢复栈顶，完美保存子页面上下文

---

### Phase 6: Run() 串联（`tui.go` 终版）

**Step 11: `internal/tui/tui.go`**

```go
func Run(srv server.Server) error {
    app := tview.NewApplication()
    pages := tview.NewPages()
    router := NewPageRouter(pages)

    // 1. 创建 MainPage（初始空壳，注入 srv 供操作按键使用）
    mp := NewMainPage(app, srv, version)

    // 2. 创建 Simulator（channel 管道）
    tasksCh, runsCh, reqsCh := NewSimulator()
    // v2: 用 srv.SubscribeRunEvents(runID) 订阅活跃 run 的事件流

    // 3. 首次数据注入（Simulator 构造时已推入初始数据到 channel）
    state := &tuiState{
        app:         app,
        router:      router,
        srv:         srv,
        mainPage:    mp,
        warningPage: NewWarningPage(version),
        stopCh:      make(chan struct{}),
    }

    // 4. 启动更新循环（先启动，确保能消费初始数据）
    state.startUpdateLoop(tasksCh, runsCh, reqsCh)

    // 5. SetRoot + 导航到主页
    app.SetRoot(pages, true)
    router.NavigateTo(mp)
    app.SetFocus(mp.FocusTarget())

    // 6. 注册按键 + 尺寸检测
    state.setupKeybindings()

    // 7. 运行
    if err := app.Run(); err != nil {
        state.stop()
        _ = srv.Shutdown(5 * time.Second)
        return fmt.Errorf("tui: %w", err)
    }
    state.stop()
    _ = srv.Shutdown(5 * time.Second)
    return nil
}
```

**数据流时序说明**：
1. `NewSimulator()` 在返回前推入初始 mock 数据到三个 channel（buffer=1）
2. `startUpdateLoop` 启动 goroutine → 消费初始数据 → `QueueUpdateDraw` → `MainPage.PopulateData`
3. 用户看到 UI 渲染完毕。后续 channel 有新数据时自动触发 Refresh

---

### 文件清单（终版）

| 文件 | 操作 | 内容 |
|------|------|------|
| `internal/tui/page.go` | **新建** | Page 接口 + panelIndex 类型 |
| `internal/tui/main_page.go` | **新建** | MainPage（3-panel + HandleKey + Refresh） |
| `internal/tui/router.go` | **新建** | PageRouter（`[]Page` 栈 + overlay + ActivePage） |
| `internal/tui/tui.go` | **新建** | SetVersion + Run() + tuiState + NewSimulator + startUpdateLoop + globalKeyHandler(6行分派) + beforeDrawCheck |
| `internal/tui/mock.go` | **新建** | 模拟数据生成 + format* 格式化函数 |
| `internal/tui/pages/page_task_detail.go` | **新建** | TaskDetailPage（Esc→GoBack） |
| `internal/tui/pages/page_standard_dashboard.go` | **新建** | StandardDashboardPage（Esc→GoBack，列表原生滚动） |
| `internal/tui/pages/page_request_detail.go` | **新建** | RequestDetailPage（Esc→GoBack） |
| `internal/tui/pages/page_warning.go` | **新建** | WarningPage（Esc→HideOverlay，吃掉输入） |

**删除的旧文件**：`state.go`, `layout.go`, `panels.go`, `pages.go`, `keybindings.go`

**不变**：`cmd/ait/ait.go`

---

### Verification

1. `go build ./cmd/ait/` 编译通过，无 import cycle
2. 启动后 128×24 终端可见 3-panel + Header + Footer（数据由 Simulator 通过 channel 注入）
3. Tab/Shift-Tab 循环切换 panel（由 MainPage.HandleKey 处理），边框灰↔黄加粗，选中背景灰↔蓝
4. Enter → TaskDetailPage / RunDashboardPage / RequestDetailPage 各页面正确渲染
5. RunDashboard 内请求列表可独立 ↑↓ 滚动（tview List 原生处理，HandleKey 透传）
6. Esc → 返回主布局（TaskDetailPage.HandleKey 调 router.GoBack）
7. 终端 <120 宽 → WarningPage overlay（beforeDrawCheck 检测）
8. WarningPage 中 Esc → HideOverlay → 恢复栈顶（含子页面上下文）
9. `GetCurrentItem() >= 0` 守卫空列表 Enter 无 crash
10. `Simulator` 推入数据后 `Refresh*` 正确重建 List 内容
11. `r` 键在左 panel 选中任务时触发 `srv.StartRun()`（goroutine 异步，不阻塞 UI）
12. `s` 键在中 panel 选中 running 状态运行时触发 `srv.StopRun()`
13. `R` 键调用 `srv.ListTasks()`/`ListTaskRunHistory()` 刷新当前 panel 数据（goroutine 异步）
14. `NewMainPage(app, srv, version)` 正确注入 srv 引用，编译时类型检查通过

### Decisions

- **Page 级 HandleKey**：每个 Page 自行定义按键，全局层仅做 `ActivePage().HandleKey()` 分派。新增页面无需修改全局 handler。`HandleKey` 仅拦截快捷键/命令键，文本输入（InputField）由 tview 原生路由处理——返回 `nil` 表示已消费，返回 `event` 表示透传
- **ActivePage 优先 overlay**：`router.ActivePage()` 返回 overlay（若存在），确保 beforeDrawCheck 和 globalKeyHandler 无需额外判断 overlay 状态
- **Focus 由 HandleKey 内部管理**：`NavigateTo`/`GoBack` 后，调用方（各 Page 的 HandleKey）负责 `app.SetFocus`
- **OnActivate/OnDeactivate 预留空实现**：v1 不触发副作用，为未来刷新/订阅/取消订阅预留。v2 中 `RunDashboardPage.OnActivate` 可调用 `srv.SubscribeRunEvents(runID)` 订阅实时事件流，`OnDeactivate` 调 `CancelFunc` 取消
- **EnterAction 返回 Page 而非执行导航**：MainPage 不持有 router 引用，解耦——router 由 HandleKey 统一使用
- **Overlay 统一机制**：Warning 走 `ShowOverlay/HideOverlay`，任何页面都可作为 overlay
- **数据反应管道**：`Simulator → chan → startUpdateLoop → QueueUpdateDraw → Page.Refresh*`。v2 替换为 `srv.SubscribeRunEvents(runID)` 实时事件流即可
- **Refresh 方法保持选中索引**：重建 List 后恢复旧选中位置（若仍有效），避免用户交互中断
- **字段安全不变**：`RunMode()`、`FinishedAt.Sub(StartedAt)`、`string(RunID)`、`totalReqs == 0` 守卫、`GetCurrentItem() >= 0` 守卫、`LatestRun == nil` 守卫
- **Server 接口值传递**：`tui.Run(srv server.Server)` 使用接口值而非指针。TUI 与 web 层享有同等的 Server 方法访问权，v1 实现 `StartRun`/`StopRun`/`ListTasks` 等核心操作，其余方法 v2 补齐
- **srv 通过构造函数注入**：`NewMainPage(app, srv, version)` 注入，详情页同理。`PageRouter` 不持有 srv——各 Page 自管理
- **操作按键用 goroutine 异步**：`StartRun`/`StopRun` 等阻塞操作在 goroutine 中执行，不阻塞 tview 事件循环

### 扩展性

- **新页面**：实现 `Page` 接口 + `HandleKey` → `NavigateTo(newPage)`，router/keybindings 零修改
- **新快捷键**：在对应 Page 的 `HandleKey` 中添加 case 分支，不影响其他页面
- **生命周期**：`OnActivate` 中加数据刷新/订阅，`OnDeactivate` 中取消订阅
- **Overlay**：任何页面可作 modal/confirm，Router 无需改动
- **翻页**：MainPage 预留 `pageSize`/`currentPage`，PgUp/PgDn 在 HandleKey 增量添加
- **真实数据接入**：替换 `NewSimulator()` 为 `srv.SubscribeRunEvents(runID)` 事件流，Pipe 层和 Page 层不变
