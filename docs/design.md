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

- 首页展示任务列表，可直接选择已有任务再次执行
- 无需记忆所有参数，通过向导页创建或编辑任务
- 任务详情页集中展示配置摘要、最近结果与运行记录
- 测试运行时实时展示指标面板（成功率、TPS、TTFT、缓存命中率、并发状态）
- 请求日志滚动查看
- 结果页支持键盘操作（生成报告、返回任务详情、再次运行等）
- 支持通过 CLI 参数生成临时任务草稿并进入 TUI 继续操作

### 1.2 任务管理

新增“任务”作为一等对象，用于保存和复用测试配置：

- 每个任务只绑定一个模型，保存协议、完整接口地址、模型、Prompt、标准模式或 Turbo 参数
- 协议值细化为 `openai-completions`、`openai-responses`、`anthropic-messages`
- 首页以任务列表形式呈现，支持新建、编辑、删除、复制、搜索和直接运行
- 任务详情页展示最近一次运行摘要和最近运行记录
- 多模型回归通过多个任务组织，而不是在单个任务内批量执行
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

### 3.1 整体模块图

```
┌─────────────────────────────────────────────────────────────────┐
│  cmd/ait/main.go  ─  入口 & 模式路由                              │
│                                                                   │
│   ┌─ 无必填参数 ──→  TUI 任务中心（任务列表）                     │
│   └─ 有完整参数 ──→  TUI 任务详情（生成临时任务草稿）              │
└───────────────┬─────────────────────────────────────────────────┘
                │
    ┌───────────▼───────────────────────────────────────────────┐
    │              internal/tui/  (NEW)                          │
    │                                                            │
    │  model.go ─ BubbleTea 根模型 + 状态机                      │
    │  messages.go ─ 所有 Msg 类型定义                            │
    │  styles.go ─ lipgloss 样式常量                              │
    │                                                            │
    │  tasklist/   ─ 任务列表页                                   │
    │  taskdetail/ ─ 任务详情页                                   │
    │  wizard/     ─ 新建 / 编辑任务向导页                         │
    │  dashboard/  ─ 运行中仪表盘页                               │
    │  result/     ─ 结果展示页                                   │
    │  turbo/      ─ Turbo 仪表盘页                               │
    └───────────────────────────────────────────────────────────┘
                │ program.Send(msg)
                │ ↑ 从任意 goroutine 安全推送
    ┌───────────▼──────────────────────────────┐
    │  internal/runner/runner.go（已有，扩展）   │
    │  RunWithCallback(cb RequestDoneCallback)  │
    └──────────────────────────────────────────┘
                │
    ┌───────────▼──────────────────────────────┐
    │  internal/turbo/  (NEW)                  │
    │  Runner ─ 并发爬坡调度器                  │
    │  Strategy ─ 步进 & 终止策略               │
    └──────────────────────────────────────────┘
                │
    ┌───────────▼──────────────────────────────┐
    │  internal/task/  (NEW)                   │
    │  Store ─ 任务 CRUD / 搜索 / 排序          │
    │  History ─ 任务运行记录与最近结果摘要      │
    └──────────────────────────────────────────┘
          │
    ┌───────────▼──────────────────────────────┐
    │  internal/client/  (已有，不变)            │
    │  OpenAI Completions / Responses /          │
    │  Anthropic Messages HTTP 客户端            │
    └──────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────┐
    │  internal/config/ (NEW)                                  │
    │  从 ~/.ait/config.json 加载 / 保存全局偏好               │
    └─────────────────────────────────────────────────────────┘

    ┌─────────────────────────────────────────────────────────┐
    │  internal/report/ (已有，扩展)                            │
    │  新增 turbo_renderer.go ─ Turbo 爬坡报告                 │
    └─────────────────────────────────────────────────────────┘
```

### 3.2 目录结构变化

```diff
  cmd/
    ait/
-     ait.go                ← 原来全部逻辑
+     main.go               ← 入口：模式检测 + 任务中心路由
+     flags.go              ← 所有 flag 定义
+     run_tui.go            ← TUI 模式启动 / 临时任务草稿注入

  internal/
+   tui/
+     model.go              ← 根 BubbleTea Model + 状态机
+     messages.go           ← 所有 Msg 类型
+     styles.go             ← lipgloss 样式常量
+     tasklist/
+       model.go            ← 任务列表状态
+       view.go             ← 任务列表 UI 渲染
+     taskdetail/
+       model.go            ← 任务详情状态
+       view.go             ← 任务详情 UI 渲染
+     wizard/
+       model.go            ← 新建 / 编辑任务向导状态
+       view.go             ← 向导 UI 渲染
+     dashboard/
+       model.go            ← 运行仪表盘状态
+       view.go             ← 仪表盘 UI 渲染
+     result/
+       model.go            ← 结果页状态
+       view.go             ← 结果页 UI 渲染
+     turbo/
+       model.go            ← Turbo 仪表盘状态
+       view.go             ← Turbo 仪表盘 UI 渲染

+   turbo/
+     runner.go             ← 并发爬坡调度器
+     strategy.go           ← 步进 & 终止策略
+     result.go             ← TurboResult、LevelResult 类型

+   config/
+     config.go             ← ~/.ait/config.json 全局配置读写

+   task/
+     store.go              ← ~/.ait/tasks.json 读写 + CRUD
+     history.go            ← ~/.ait/history/<task-id>.json 读写

    runner/
      runner.go             ← 扩展：增加 Stop() 方法 + 每请求完成回调

    types/
      types.go              ← 扩展：增加 TurboConfig、TurboResult

    report/
      report.go             ← 已有
+     turbo_renderer.go     ← Turbo 爬坡报告渲染
```

### 3.3 关键设计原则

**原则 1：Runner 不感知 TUI**

Runner 通过回调函数推送进度，不直接依赖 BubbleTea。TUI 模式下由外层把 `program.Send(msg)` 包进回调：

```go
// runner 接口不变，增加每请求完成的细粒度回调
type RequestDoneCallback func(metrics *client.ResponseMetrics, index int, err error)

// TUI 模式下的回调实现
cb := func(m *client.ResponseMetrics, idx int, err error) {
    program.Send(tui.RequestDoneMsg{Metrics: m, Index: idx, Err: err})
}
runner.RunWithCallback(cb)
```

**原则 2：TUI 是纯状态机**

`tui.Model` 通过消息驱动状态转换，不直接调用任何 I/O 函数，所有副作用都封装在 `tea.Cmd` 中，方便测试。

**原则 3：任务是一等领域对象**

Runner 消费的是一次运行所需的 `Input`，但 UI 和持久化围绕 `TaskDefinition` 展开。列表、详情、编辑、重跑、历史记录都基于任务对象组织，而不是把一次性的 flag 输入直接暴露给用户。

**原则 4：一个任务只测一个模型**

任务是最小回归单元。`TaskDefinition` 只保存一个 `Model`，这样任务详情、运行记录、Turbo 极限和结果对比都能稳定映射到单一模型；若要覆盖多个模型，应创建多个任务分别执行。

### 3.4 任务生命周期

1. 用户从任务列表进入“新建任务”向导，保存后写入 `~/.ait/tasks.json`
2. 用户在任务详情页查看配置摘要、最近结果和最近运行记录
3. 用户从任务详情页启动标准模式或 Turbo 模式测试
4. 测试完成后写入 `~/.ait/history/<task-id>.json`，并回写任务的最近运行摘要
5. 用户后续可直接在任务列表或任务详情页再次运行，无需重新输入参数

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
启动带完整参数 ─→ 生成临时任务草稿 ───────→ TaskDetail

                 ┌─────────────┐
                 │  TaskList   │
                 │  任务列表页   │
                 └──────┬──────┘
               [a 新建] │  │ [Enter]
                        │  │
                        ▼  ▼
                 ┌─────────────┐
                 │   Wizard    │
                 │ 新建/编辑任务 │
                 └──────┬──────┘
                  [保存任务] │
                        ▼
                 ┌─────────────┐
                 │ TaskDetail  │
                 │  任务详情页   │
                 └──────┬──────┘
            [Enter / r] │  │ [e 编辑]
                       │  └──────────────┐
                       │                 │
         [标准模式]    ▼                 │
                 ┌─────────────┐         │
                 │   Running   │         │
                 │  标准运行中   │         │
                 └──────┬──────┘         │
              [完成/s] │                │
                        ▼                │
                 ┌─────────────┐         │
                 │ Completed   │         │
                 │  标准结果页   │         │
                 └──────┬──────┘         │
           [b 返回详情] │                │
                        └────────────────┘

         [Turbo 模式]   ▼
                 ┌─────────────┐
                 │TurboRunning │
                 │ Turbo 爬坡中 │
                 └──────┬──────┘
              [完成/s] │
                        ▼
                 ┌─────────────┐
                 │TurboCompleted│
                 │ Turbo 结果页 │
                 └──────┬──────┘
           [b 返回详情] │
                        └──────────────→ TaskDetail
```

### 4.3 页面设计

---

#### 页面 1：任务列表首页

```
╔══ AIT  任务中心 ─────────────────────────────────────────────══╗
║  已保存任务: 12   最近运行: 2026-05-16 09:42   [/] 搜索任务      ║
╠══════════════════════╦═══════════════════════════════════════════╣
║  任务列表              ║  快捷操作 / 最近摘要                      ║
║                        ║                                           ║
║ ▶ nightly-openai       ║  [a] 新建任务                            ║
║   标准模式 · gpt-4o    ║  [Enter] 查看详情                        ║
║   并发 10 / 请求 200   ║  [r] 直接运行选中任务                    ║
║   上次: 98.5% · 12m 前 ║  [e] 编辑  [d] 删除  [y] 复制任务        ║
║                        ║                                           ║
║   turbo-anthropic      ║  最近执行                                ║
║   Turbo · claude-3-7   ║  nightly-openai      ✓ 98.5%  245ms      ║
║   1→50 +2 / 每级 30    ║  turbo-anthropic     ★ 稳定并发 8        ║
║   上次: 峰值 TPS 245.3 ║  smoke-regression     ✗ timeout ×2       ║
║                        ║                                           ║
║   smoke-regression     ║  提示：支持按任务名、协议、模型、模式过滤║
║   标准模式 · gpt-4o-mini ║                                           ║
║   并发 2 / 请求 20     ║                                           ║
║   从未运行             ║                                           ║
╠══════════════════════╩═══════════════════════════════════════════╣
║  [↑↓] 选择   [Enter] 详情   [a] 新建   [r] 运行   [q] 退出        ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 2：任务详情页

```
╔══ AIT  任务详情 ─ nightly-openai ────────────────────────────══╗
║  任务 ID: task_01   更新: 2026-05-16 09:30   最近运行: 12m 前    ║
╠══════════════════════╦═══════════════════════════════════════════╣
║  配置摘要              ║  最近一次结果                            ║
║                        ║                                           ║
║  协议      openai-responses ║  状态        ✓ 完成                 ║
║  接口地址  https://api.openai.com/v1/responses                  ║
║  模型      gpt-4o      ║  avg TTFT    245ms                      ║
║  模式      标准模式     ║  avg TPS     124.3 tok/s               ║
║  并发      10          ║  缓存命中率   42.0%                     ║
║  请求数    200         ║  总耗时      20.4s                      ║
║  Prompt    你好，介绍一下你自己。║  报告        ait-report-...json ║
╠══════════════════════╩═══════════════════════════════════════════╣
║  最近运行记录                                                     ║
║  2026-05-16 09:30   ✓ 98.5%   TTFT 245ms   Cache 42%   20.4s    ║
║  2026-05-15 23:10   ✓ 99.0%   TTFT 231ms   Cache 38%   19.8s    ║
║  2026-05-15 21:42   ✗ timeout ×2     Cache 12%        31.2s     ║
╠══════════════════════════════════════════════════════════════════╣
║  [Enter] 运行   [e] 编辑   [h] 完整历史   [d] 删除   [b] 返回    ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 3：向导 - 新建任务（Step 1/3）

```
╔══════════════════════════════════════════════════════════════════╗
║  ██████╗ ██╗████████╗   AI 模型性能测试工具  v2.0              ║
║  ██╔══██╗██║╚══██╔══╝   https://github.com/yinxulai/ait       ║
║  ███████║██║   ██║                                              ║
║  ██╔══██║██║   ██║       向导  1/3 · 新建任务                  ║
║  ██║  ██║██║   ██║                                              ║
║  ╚═╝  ╚═╝╚═╝   ╚═╝                                              ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║    任务名称    nightly-openai                                    ║
║               ──────────────────────────────────────────        ║
║                                                                  ║
║    协议类型    > openai-responses                                ║
║               ○ openai-completions                              ║
║               ● openai-responses   ○ anthropic-messages         ║
║                                                                  ║
║    接口地址    https://api.openai.com/v1/responses              ║
║               ──────────────────────────────────────────        ║
║               提示：填写完整接口地址，而不是 base URL            ║
║                                                                  ║
║    API 密钥    sk-••••••••••••••••••••••••••••••                ║
║               ──────────────────────────────────────────        ║
║                                                                  ║
║    测试模型    gpt-4o                                            ║
║               ──────────────────────────────────────────        ║
║               提示：每个任务仅允许选择一个模型                   ║
║                                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║  [Tab] 下一项   [↑↓] 切换协议   [Enter] 下一步   [Esc] 返回    ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 4：向导 - 测试参数（Step 2/3）

```
╔══════════════════════════════════════════════════════════════════╗
║  AIT  v2.0                       向导  2/3 · 任务参数           ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║    测试模式    ○ 标准模式    ● Turbo 模式                        ║
║               [←→ 切换]                                         ║
║                                                                  ║
║  ── 标准模式参数 ──────────────────────────────────────────      ║
║    并发数      [  5  ]   请求总数   [ 100 ]                     ║
║    超时时间    [ 300s ]  流式模式   [✓ 开启]                     ║
║                                                                  ║
║  ── Turbo 模式参数 ────────────────────────────────────────      ║
║    初始并发    [  1  ]   最大并发   [  50  ]                    ║
║    步进值      [  2  ]   每级请求数 [  30  ]                    ║
║    停止条件    成功率低于 [ 90% ]  或  延迟超过 [ 10s ]          ║
║                                                                  ║
║  ── Prompt 配置 ───────────────────────────────────────────      ║
║    输入方式    ● 直接输入   ○ 文件   ○ 按长度生成               ║
║    内容        你好，介绍一下你自己。                            ║
║               ──────────────────────────────────────────        ║
║                                                                  ║
║    运行后记录    [✓ 保存运行记录到任务历史]                      ║
║                                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║  [Tab] 下一项   [←→] 切换模式   [Enter] 下一步   [Esc] 返回    ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 5：向导 - 确认（Step 3/3）

```
╔══════════════════════════════════════════════════════════════════╗
║  AIT  v2.0                       向导  3/3 · 保存任务           ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║    🆔  任务 ID     a3f2-8b1c-...                                 ║
║    🏷️  任务名称    nightly-openai                                ║
║    🔗  协议        openai-responses                              ║
║    🌐  接口地址    https://api.openai.com/v1/responses          ║
║    🔑  API 密钥    sk-****...****                               ║
║    🤖  测试模型    gpt-4o                                        ║
║    🚀  测试模式    Turbo 模式                                    ║
║    ⚡  并发爬坡    1 → 50  步进 +2  每级 30 请求                 ║
║    🛑  停止条件    成功率 < 90%  或  延迟 > 10s                  ║
║    🌊  流式模式    开启                                          ║
║    📝  Prompt      你好，介绍一下你自己。 (长度: 12)             ║
║                                                                  ║
║    💾  保存任务到 ~/.ait/tasks.json  [✓]                        ║
║    📝  创建后自动写入运行历史索引  [✓]                          ║
║                                                                  ║
║  ┌────────────────────────────────────────────────────────┐     ║
║  │  ▶  保存任务                                            │     ║
║  └────────────────────────────────────────────────────────┘     ║
║                                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║  [Enter] 保存任务   [r] 保存并运行   [Esc] 返回修改             ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 6：标准模式运行仪表盘

```
╔══ AIT  正在测试 ─ gpt-4o ────────────────────────────────────════╗
║  任务: nightly-openai  协议: openai-responses  并发: 5  请求: 100║
╠══════════════════════╦═══════════════════════════════════════════╣
║  进度                 ║  实时指标                                  ║
║                       ║                                           ║
║  完成  ████████░░  47 ║  成功率  ██████████████████░░░  98.0%    ║
║  失败  ░░░░░░░░░░   2 ║                                           ║
║  总计            100  ║  avg TPS      124.3 tok/s                ║
║                       ║  avg TTFT     245ms                      ║
║  ──────────────────── ║  缓存命中率   42.0%                      ║
║  已用时   12.4s       ║  avg 总耗时   1.24s                      ║
║  预计剩余  ~8.2s      ║  并发槽  [●●●●●]  5/5 活跃               ║
║                       ║                                           ║
╠══════════════════════╩═══════════════════════════════════════════╣
║  请求日志                                                  [l 展开] ║
║  ✓ #48  245ms  TTFT:82ms  cache:100%  128tok  12.3tok/s         ║
║  ✗ #47  timeout (30.0s)                                          ║
║  ✓ #46  312ms  TTFT:95ms  cache:25%    96tok   9.8tok/s         ║
║  ✓ #45  198ms  TTFT:71ms  cache:0%    145tok  14.2tok/s         ║
╠══════════════════════════════════════════════════════════════════╣
║  [p] 暂停   [s] 停止   [l] 切换日志详情   [r] 提前报告   [q] 退出 ║
╚══════════════════════════════════════════════════════════════════╝
```

#### 页面 7：Turbo 模式运行仪表盘

```
╔══ AIT  Turbo 模式 ─ gpt-4o ──────────────────────────────────════╗
║  任务: turbo-anthropic   协议: anthropic-messages               ║
║  爬坡: 1→50  步进: +2/级  每级: 30 请求                          ║
╠══════════════════════╦═══════════════════════════════════════════╣
║  爬坡曲线 (TPS)        ║  当前级别  [并发 = 8]                     ║
║                        ║                                           ║
║  250┤         ╭──●     ║  成功率  █████████████████░░  96.0%     ║
║  200┤    ╭────╯         ║  TPS     245.3  tok/s                   ║
║  150┤ ╭──╯              ║  TTFT    312ms                          ║
║  100┤─╯                 ║  Cache   44.0%                          ║
║   50┤                   ║  总耗时  1.51s                          ║
║     └──┬──┬──┬──┬──→   ║  本级完成   28 / 30                     ║
║        1  2  4  6  8    ║  状态   🟢 稳定，继续探测...             ║
╠══════════════════════╩═══════════════════════════════════════════╣
║  并发  成功率    TPS      TTFT    Cache   总耗时   状态           ║
║  ──────────────────────────────────────────────────────────────  ║
║     1  100.0%   31.2     89ms    0.0%    0.82s   ✓ 稳定          ║
║     2  100.0%   62.5     91ms   18.0%    0.84s   ✓ 稳定          ║
║     4   99.0%  121.3     98ms   26.0%    0.91s   ✓ 稳定          ║
║     6   98.0%  178.4    124ms   33.0%    1.08s   ✓ 稳定          ║
║  ▶  8   96.0%  245.3    312ms   44.0%    1.51s   🔄 探测中       ║
╠══════════════════════════════════════════════════════════════════╣
║  [s] 停止   [m] 手动标记为极限   [r] 提前生成报告   [q] 退出       ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 8：标准模式结果页

```
╔══ AIT  测试完成 ─ gpt-4o ─────────────────────────────────────════╗
║  任务: nightly-openai   协议: openai-responses                  ║
║  耗时: 20.4s   成功率: 98.0%   总请求: 100                      ║
╠══════════════════════════════════════════════════════════════════╣
║  指标              最小值      平均值      标准差      最大值       ║
║  ──────────────────────────────────────────────────────────────  ║
║  总耗时              0.82s      1.24s      ±0.31s     3.12s       ║
║  TTFT                 71ms      245ms       ±89ms      812ms      ║
║  TPOT                 12ms       18ms        ±4ms       45ms      ║
║  输出 TPS             89.2      124.3       ±21.4      198.5      ║
║  吞吐 TPS            102.1      148.7       ±25.2      231.4      ║
║  缓存命中率           0.0%       42.0%      ±18.5%     100.0%     ║
║  输入 Token            42         64         ±12         98       ║
║  输出 Token            78        128         ±32        195       ║
║  DNS 时间            1.2ms      3.4ms                  12.1ms     ║
║  TCP 连接时间         2.1ms      4.8ms                   9.3ms    ║
║  TLS 握手时间         8.4ms     12.3ms                  28.7ms    ║
╠══════════════════════════════════════════════════════════════════╣
║  错误摘要 (2 个错误，占 2.0%)                                      ║
║  context deadline exceeded (timeout)   × 2                       ║
╠══════════════════════════════════════════════════════════════════╣
║  任务记录已更新：最近运行摘要 + 历史索引                          ║
╠══════════════════════════════════════════════════════════════════╣
║  [r] 生成报告   [c] 复制摘要   [b] 返回任务详情   [q] 退出         ║
╚══════════════════════════════════════════════════════════════════╝
```

---

#### 页面 9：Turbo 模式结果页

```
╔══ AIT  Turbo 完成 ─ gpt-4o ──────────────────────────────────════╗
║  任务: turbo-anthropic   协议: anthropic-messages              ║
║  🏆 最大稳定并发: 8   峰值 TPS: 245.3 tok/s   探测耗时: 52s       ║
╠══════════════════════════════════════════════════════════════════╣
║  TPS 爬坡曲线                          成功率曲线                  ║
║                                                                   ║
║  300┤             ╭─●最大稳定 245.3   100%┤████████████          ║
║  200┤        ╭────╯  ╲降级               ║        ████░░         ║
║  100┤   ╭────╯         ╲               95%┤            ░░░ ← 阈值 ║
║    0┤───╯               ●              90%└──────────────→       ║
║     └──┬──┬──┬──┬──┬──→                   1  2  4  6  8  10     ║
║        1  2  4  6  8  10                  并发数                  ║
║                                                                   ║
╠══════════════════════════════════════════════════════════════════╣
║  并发   成功率    TPS       TTFT     Cache   总耗时    结论        ║
║  ──────────────────────────────────────────────────────────────  ║
║     1   100.0%    31.2     89ms      0.0%    0.82s    ✓ 稳定     ║
║     2   100.0%    62.5     91ms     18.0%    0.84s    ✓ 稳定     ║
║     4    99.0%   121.3     98ms     26.0%    0.91s    ✓ 稳定     ║
║     6    98.0%   178.4    124ms     33.0%    1.08s    ✓ 稳定     ║
║  ★  8    96.0%   245.3    312ms     44.0%    1.51s    ✓ 最大稳定  ║
║    10    84.0%   198.1    892ms     12.0%    4.23s    ✗ 降级      ║
╠══════════════════════════════════════════════════════════════════╣
║  任务记录已更新：最近运行摘要 + 历史索引                          ║
╠══════════════════════════════════════════════════════════════════╣
║  [r] 生成报告   [d] 详细数据   [b] 返回任务详情   [q] 退出         ║
╚══════════════════════════════════════════════════════════════════╝
```

---

### 4.4 键盘交互规范

| 按键 | 适用页面 | 功能 |
|------|----------|------|
| `a` | 任务列表 | 新建任务 |
| `/` | 任务列表 | 搜索 / 过滤任务 |
| `Enter` | 任务列表 | 查看任务详情 |
| `r` | 任务列表 / 任务详情 | 直接运行当前任务 |
| `e` | 任务详情 | 编辑当前任务 |
| `d` | 任务详情 | 删除当前任务 |
| `y` | 任务列表 / 任务详情 | 复制当前任务 |
| `h` | 任务详情 | 打开完整运行历史 |
| `b` | 任务详情 / 结果页 | 返回上一级 |
| `Tab` / `Shift+Tab` | 向导 | 在输入项间切换焦点 |
| `↑` / `↓` | 任务列表、向导、结果表格 | 上下选择 |
| `←` / `→` | 向导模式选择 | 切换选项 |
| `Enter` | 向导 | 确认 / 下一步 / 保存 |
| `Esc` | 所有页 | 返回上一步 / 取消 |
| `p` | Running | 暂停/恢复 |
| `s` | Running / Turbo | 停止测试 |
| `l` | Dashboard | 切换日志详情展开/折叠 |
| `r` | Running / 结果页 | 生成报告文件 |
| `m` | Turbo Running | 手动标记当前并发为最大稳定并发并停止 |
| `c` | 结果页 | 复制摘要到剪贴板 |
| `q` / `Ctrl+C` | 所有页 | 退出程序 |

---

### 4.5 布局响应式策略

- 终端宽度 `< 80` 列：任务列表与任务详情折叠为单列，摘要面板移动到下方
- 终端宽度 `≥ 80` 列：任务列表、任务详情和运行页都采用双栏布局
- 终端高度不足时，历史记录区或日志区自动收缩，至少保留 3 行内容

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
- 通过完整 CLI 参数启动时，AIT 会先生成一个未保存的临时任务草稿，用户可选择保存后复用
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

现有指标结构也需要补充缓存命中率字段，用于 dashboard、结果页和报告渲染：

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

### 6.5 TUI 消息类型

```go
// internal/tui/messages.go

// TasksLoadedMsg 任务列表加载完成
type TasksLoadedMsg struct {
  Tasks []types.TaskDefinition
}

// TaskSavedMsg 任务保存完成
type TaskSavedMsg struct {
  Task types.TaskDefinition
}

// TaskHistoryLoadedMsg 任务运行记录加载完成
type TaskHistoryLoadedMsg struct {
  TaskID  string
  History []types.TaskRunSummary
}

// RequestDoneMsg 单个请求完成
type RequestDoneMsg struct {
    Metrics *client.ResponseMetrics
    Index   int
    Err     error
}

// AllRequestsDoneMsg 所有请求完成
type AllRequestsDoneMsg struct {
    Result *types.ReportData
    Errors []string
}

// TurboLevelStartMsg Turbo 新一级开始
type TurboLevelStartMsg struct {
    Concurrency int
    LevelIndex  int
}

// TurboLevelDoneMsg Turbo 一级完成
type TurboLevelDoneMsg struct {
    Level      types.TurboLevelResult
    LevelIndex int
}

// TurboDoneMsg Turbo 全部完成
type TurboDoneMsg struct {
    Result *types.TurboResult
}

// ProgressTickMsg 定时刷新实时指标
type ProgressTickMsg struct {
    Stats types.StatsData
}

// ErrorMsg 运行时错误
type ErrorMsg struct {
    Err error
}
```

### 6.6 Runner 接口扩展

```go
// internal/runner/runner.go 新增

// RequestDoneCallback 每个请求完成后的回调（细粒度，供 TUI 使用）
type RequestDoneCallback func(metrics *client.ResponseMetrics, index int, err error)

// RunWithCallback 运行测试，每个请求完成后调用 cb（线程安全）
// 同时保留原有的 RunWithProgress，供 Legacy 模式使用
func (r *Runner) RunWithCallback(cb RequestDoneCallback) (*types.ReportData, error)

// Stop 异步停止正在进行的测试
func (r *Runner) Stop()
```

### 6.7 任务与全局配置持久化

```go
// internal/config/config.go

type Config struct {
  SaveAPIKey         bool   `json:"save_api_key"`
  LastSelectedTaskID string `json:"last_selected_task_id,omitempty"`
  DefaultProtocol    string `json:"default_protocol,omitempty"` // openai-completions | openai-responses | anthropic-messages
}

func Load() (*Config, error)   // 从 ~/.ait/config.json 加载
func (c *Config) Save() error  // 保存到 ~/.ait/config.json

// internal/task/store.go

type TaskStore struct {
  Tasks []types.TaskDefinition `json:"tasks"`
}

func LoadTasks() (*TaskStore, error)                  // 从 ~/.ait/tasks.json 加载
func (s *TaskStore) Save() error                      // 保存到 ~/.ait/tasks.json
func (s *TaskStore) Upsert(task types.TaskDefinition) // 新建或更新任务
func (s *TaskStore) Delete(taskID string) error

// internal/task/history.go

func AppendRun(taskID string, run types.TaskRunSummary) error
func LoadHistory(taskID string, limit int) ([]types.TaskRunSummary, error)
```

---

## 7. 开发计划

### Phase 1 — 任务中心与 TUI 基础框架（优先）

**目标：** 先建立任务管理主流程，再用 BubbleTea 替换现有的进度条 + 静态表格输出

**任务清单：**

- [ ] 引入依赖：`charm.land/bubbletea/v2`、`bubbles`、`lipgloss`
- [ ] 实现 `internal/tui/` 基础骨架（model、messages、styles）
- [ ] 实现任务列表页（tasklist）：选择 / 搜索 / 删除 / 复制
- [ ] 实现任务详情页（taskdetail）：配置摘要 + 最近记录 + 直接运行
- [ ] 实现向导页（wizard）：三步创建 / 编辑任务
- [ ] 实现仪表盘页（dashboard）：进度 + 实时指标双栏
- [ ] 实现结果页（result）：完整指标表格 + 键盘操作
- [ ] 协议枚举细化：`openai-completions`、`openai-responses`、`anthropic-messages`
- [ ] 扩展指标采集与渲染：缓存命中率（dashboard / result / report）
- [ ] `cmd/ait/main.go` 模式检测路由（无参数 → 任务列表，有参数 → 临时任务草稿）
- [ ] 实现任务持久化（`tasks.json` + `history/*.json`）
- [ ] Runner 增加 `Stop()` 方法和 `RunWithCallback` 接口
- [ ] 全局配置持久化（默认协议、最后选择任务、密钥保存策略）
- [ ] 结果页回写任务最近运行摘要
- [ ] 响应式布局（终端宽度自适应）
- [ ] `internal/display/` 模块退役，由 TUI 全面接管输出

---

### Phase 2 — Turbo 模式

**目标：** 将并发爬坡能力完整融入任务体系

**任务清单：**

- [ ] 实现 `internal/turbo/runner.go`：封装爬坡调度逻辑
- [ ] 实现 `internal/turbo/strategy.go`：步进 & 终止策略
- [ ] `types.TurboConfig`、`TurboLevelResult`、`TurboResult` 数据结构
- [ ] TUI Turbo 仪表盘页（折线图 + 爬坡表格）
- [ ] `internal/report/turbo_renderer.go`：Turbo CSV/JSON 报告
- [ ] Turbo 结果写回任务最近摘要和运行记录
- [ ] 新增 CLI 参数：`--turbo`、`--turbo-*` 系列

---

### Phase 3 — 增强

**目标：** 细节打磨与扩展

**任务清单：**

- [ ] 多任务 Turbo 对比（并排爬坡曲线）
- [ ] 任务收藏和快速筛选视图
- [ ] 任务复制、模板化创建和批量导入
- [ ] 运行记录对比视图（同一任务不同 run 对比）
- [ ] 结果页 `c` 键复制摘要到剪贴板
- [ ] `ntcharts` 折线图替换 ASCII 折线图
- [ ] 终端尺寸变化自适应重绘
- [ ] 完善单元测试（TUI model 测试、turbo strategy 测试）

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
