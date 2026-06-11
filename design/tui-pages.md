# TUI 页面功能与布局设计

> 版本：v2.1 草案
> 日期：2026-06-11
> 目标：重新定义 AIT TUI 的所有页面功能、信息层级、布局规则和实现注意事项，作为后续重写页面的统一依据。

---

## 1. 设计目标

当前 TUI 的核心问题不是单个页面缺少修补，而是整体页面系统同时存在表格、边框、双栏、固定高度面板、复杂 Header、复杂 Hotkeys，导致渲染结果不可控。新版 TUI 必须先回到稳定基线：

1. **所有页面单栏纵向布局**：不再使用左右分栏、嵌套面板、表格布局。
2. **最终输出受控**：任何页面输出必须经过统一宽高裁剪，不能超过终端尺寸。
3. **信息优先级明确**：页面先展示当前用户最需要做的动作，再展示辅助信息。
4. **功能保留，视觉降级**：先保证任务、运行、历史、请求详情、配置编辑都能稳定使用，不追求复杂装饰。
5. **测试锁定布局**：所有页面都要有尺寸约束测试，避免后续再次出现“飞扬”。

---

## 2. 全局页面契约

### 2.1 页面结构

所有页面统一为三段：

```text
Header
Content
Hotkeys
```

推荐高度：

| 区域 | 高度 | 说明 |
|---|---:|---|
| Header | 2 行 | 当前页面标题、当前任务/运行上下文 |
| Content | 剩余高度 | 页面主体内容，必须单栏纵向排列 |
| Hotkeys | 1-2 行 | 当前页可用快捷键和全局提示 |

低高度终端下，Hotkeys 可以降级为 1 行；Header 不允许展开成 ASCII Art。

### 2.2 全局禁用项

TUI 页面渲染层禁止使用：

- `lipgloss/table`
- 左右分栏 `Split`
- 嵌套边框 Panel
- 应用外层边框
- ASCII Art Header
- 大面积背景色块
- 根据比例拆宽度的布局
- 页面内部自行拼出超过终端高度的完整块

### 2.3 全局允许组件

只保留以下轻量组件：

| 组件 | 用途 |
|---|---|
| Section 标题 | 分隔页面主要内容区 |
| Divider | 简短横线，长度受宽度约束 |
| Key-Value 行 | 展示配置、指标、状态 |
| Status 行 | 展示运行状态、加载状态、错误状态 |
| List 行 | 展示可选择列表项 |
| Progress 行 | 展示运行进度 |
| Preview 行 | 展示输入/输出摘要 |
| Empty 行 | 展示空状态 |

### 2.4 宽高约束

所有页面必须满足：

1. 输出行数 `<= terminal height`。
2. 任意一行显示宽度 `<= terminal width`。
3. 中文、emoji、ANSI 样式不能破坏宽度计算。
4. 长文本必须截断或受控折行。
5. 多行错误信息必须先归一化，不能直接插入页面结构。

### 2.5 标准线框

所有页面默认使用同一线框，页面差异只体现在 Content 的 section 顺序和字段取舍。

```text
┌ logical viewport width ─────────────────────────────────────┐
AIT · {page title}                                  {context}
{primary hotkeys for this page}

{section title}
{one-line summary}
{one-line details if needed}

{main list or main editable content}
> {selected row}
  {normal row}

{optional selected summary / error / hint}
[?] help · [F2] lang · [q] quit
└─────────────────────────────────────────────────────────────┘
```

注意：上面的框线只是文档示意，实际 TUI 不绘制外框。

### 2.6 三种高度档位

页面必须按高度档位降级，而不是每页自由发挥。

| 档位 | 终端高度 | Header | Hotkeys | Content 策略 |
|---|---:|---:|---:|---|
| Small | `12-19` | 2 | 1 | 只保留主状态和主列表 |
| Normal | `20-34` | 2 | 1-2 | 展示摘要、主列表、少量选中详情 |
| Large | `>=35` | 2 | 2 | 展示更多预览行，但仍保持单栏 |

Small 档位不能出现：多行说明、选中详情大块、二级指标展开、长 body 预览。

### 2.7 内容行类型

实现时建议统一为这些行类型，避免每页重复发明布局：

| 行类型 | 形式 | 说明 |
|---|---|---|
| `title` | `任务` | section 名称 |
| `summary` | `running · 32/100 · success=96.9%` | 一行摘要 |
| `kv` | `model=gpt-4.1 · stream=on` | 多个键值对 |
| `progress` | `████░░░░ 40%` | 受宽度约束 |
| `row` | `> #032 ok · total=820ms` | 可选择列表行 |
| `preview` | `raw={...}` | 受控预览 |
| `empty` | `暂无任务。按 n 创建。` | 空状态 |
| `error` | `error=429 rate limit` | 单行错误 |

---

## 3. 页面总览

| 页面 | 主要目标 | 主要动作 | 布局复杂度 |
|---|---|---|---|
| 任务列表 | 找到任务、查看最近状态、启动/新建任务 | 新建、运行、进入详情、删除、刷新 | 低 |
| 任务详情 | 查看任务配置、当前运行和历史记录 | 运行、编辑、查看历史、生成报告 | 中 |
| 标准运行页 | 观察标准压测进度和请求级结果 | 后台、停止、查看请求详情 | 中 |
| Turbo 运行页 | 观察并发爬坡和最佳并发 | 后台、停止、查看 level/请求 | 中 |
| 请求详情 | 查看单条请求的性能、输入、输出/错误 | 上下切换请求、返回 | 中 |
| 创建/编辑向导 | 创建或修改任务配置 | 编辑字段、切换步骤、保存 | 中 |
| 代理配置页 | 设置全局代理 | 输入、保存、清空 | 低 |
| 帮助页 | 查看快捷键和工作流说明 | 返回、滚动 | 低 |

### 3.1 ASCII 页面线框总览

以下 ASCII 只用于设计评审，帮助确认每页的区域和信息密度。实际 TUI 不绘制这些外框。

#### 任务列表

```text
+----------------------------------------------------------------------+
| AIT · 任务列表                                      tasks=12 running=2 |
| ↑/↓ 选择 · Enter 详情 · r 运行 · n 新建 · R 刷新                      |
|----------------------------------------------------------------------|
| 概览                                                                 |
| selected=3/12 · failed_recent=1 · source=summary                     |
|                                                                      |
| 任务                                                                 |
| > chat-prod · standard · responses · running 32/100                  |
|   model=gpt-4.1 · success=98.0% · ttft=320ms · tps=84.2              |
|   claude-cache · turbo · anthropic · done                            |
|   model=claude-3-5 · success=100% · best_c=12 · tps=142.8            |
+----------------------------------------------------------------------+
```

#### 任务详情

```text
+----------------------------------------------------------------------+
| AIT · 任务详情                                      chat-prod · std    |
| r 运行 · s 停止 · e 编辑 · Enter 历史 · g 报告 · Esc 返回             |
|----------------------------------------------------------------------|
| 配置                                                                 |
| protocol=responses · model=gpt-4.1 · stream=on                       |
| endpoint=https://api.example.com/v1/responses                        |
| requests=100 · concurrency=8 · prompt=generated:100                  |
|                                                                      |
| 当前运行                                                             |
| running · 32/100 · success=96.9% · ttft=310ms · tps=82.4             |
| ████████░░░░░░░░░░ 32%                                               |
|                                                                      |
| 历史                                                                 |
| > 10:22 · done · 100/100 · success=99.0% · ttft=300ms                |
|   22:10 · failed · 42/100 · success=76.2% · error=429                |
+----------------------------------------------------------------------+
```

#### 标准运行页

```text
+----------------------------------------------------------------------+
| AIT · 运行                                         chat-prod · std    |
| b 后台 · s 停止 · Enter 请求详情 · ↑/↓ 选择 · Esc 返回               |
|----------------------------------------------------------------------|
| 状态                                                                 |
| running · run=abc123 · 32/100 · elapsed=28s                          |
| success=96.9% · failed=1 · ttft=310ms · tps=82.4 · cache=72.0%       |
| ████████░░░░░░░░░░ 32%                                               |
|                                                                      |
| 请求                                                                 |
| > #032 ok · total=820ms · ttft=310ms · tps=84.0 · tokens=982         |
|   #031 ok · total=790ms · ttft=295ms · tps=86.1 · tokens=1004        |
|   #030 failed · total=1.2s · error=429 rate limit                    |
+----------------------------------------------------------------------+
```

#### Turbo 运行页

```text
+----------------------------------------------------------------------+
| AIT · Turbo                                        cache-test · turbo |
| b 后台 · s 停止 · Enter 详情 · ↑/↓ 选择 · Esc 返回                   |
|----------------------------------------------------------------------|
| 状态                                                                 |
| running · current_c=16 · level=3/5 · requests=160/300                |
| best_c=12 · success=99.2% · ttft=420ms · tps=142.8 · stop=none       |
| ██████████░░░░░░░░ 53%                                               |
|                                                                      |
| Levels                                                               |
| > c=16 · running · 40/100 · success=95.0% · ttft=680ms · tps=150.2   |
|   c=12 · best · 100/100 · success=99.2% · ttft=420ms · tps=142.8     |
|   c=8 · done · 100/100 · success=100% · ttft=330ms · tps=110.5       |
+----------------------------------------------------------------------+
```

#### 请求详情

```text
+----------------------------------------------------------------------+
| AIT · 请求详情                                      chat-prod · #32   |
| ←/→ 切换请求 · Home/End 首尾 · Esc 返回                              |
|----------------------------------------------------------------------|
| 概览                                                                 |
| status=ok · total=820ms · ttft=310ms · tps=84.0 · tokens=982         |
| cache=hit · dns=12ms · tcp=28ms · tls=70ms · server=200              |
|                                                                      |
| 输入                                                                 |
| POST /v1/responses · stream=true · prompt=generated:100              |
| body={"model":"gpt-4.1","input":"..."}                           |
|                                                                      |
| 输出                                                                 |
| finish=stop · bytes=1842                                             |
| 这是响应内容的受控预览，最多占用剩余高度……                           |
+----------------------------------------------------------------------+
```

#### 创建/编辑向导

```text
+----------------------------------------------------------------------+
| AIT · 新建任务                                      step=1/3 基础     |
| ↑/↓ 字段 · Enter 编辑 · Tab 下一步 · Ctrl+S 保存                     |
|----------------------------------------------------------------------|
| 步骤                                                                 |
| [1 基础] -> 2 请求 -> 3 模式                                         |
|                                                                      |
| 字段                                                                 |
| > name       chat-prod                                               |
|   protocol   openai-responses                                        |
|   endpoint   https://api.example.com/v1/responses                    |
|   api_key    sk-••••••abcd                                           |
|   model      gpt-4.1                                                 |
|                                                                      |
| 说明                                                                 |
| 任务名称用于在列表和历史记录中识别配置。                             |
+----------------------------------------------------------------------+
```

#### 代理配置

```text
+----------------------------------------------------------------------+
| AIT · 代理配置                                      source=config     |
| Enter 编辑 · Ctrl+S 保存 · Ctrl+L 清空 · Esc 返回                    |
|----------------------------------------------------------------------|
| 代理                                                                 |
| current=http://127.0.0.1:7890                                        |
| input=http://127.0.0.1:7890                                          |
| env_proxy=detected · task_proxy_overrides=yes                        |
|                                                                      |
| 说明                                                                 |
| 任务 proxy_url > 全局代理 > 环境变量。清空后将回退到环境变量。       |
+----------------------------------------------------------------------+
```

#### 帮助页

```text
+----------------------------------------------------------------------+
| AIT · 帮助                                          scroll=0          |
| ↑/↓ 滚动 · PgUp/PgDn 翻页 · Esc 返回                                 |
|----------------------------------------------------------------------|
| 全局                                                                 |
| ? 帮助 · F2 语言 · q 退出 · Esc 返回                                 |
|                                                                      |
| 任务                                                                 |
| n 新建 · Enter 详情 · r 运行 · e 编辑 · d 删除 · R 刷新              |
|                                                                      |
| 运行                                                                 |
| b 后台 · s 停止 · Enter 请求详情 · ↑/↓ 选择请求                      |
|                                                                      |
| 工作流                                                               |
| 1. 新建或选择任务 -> 2. 启动运行 -> 3. 查看请求详情                  |
+----------------------------------------------------------------------+
```

---

## 4. 任务列表页

### 4.1 功能目标

任务列表是 TUI 首页。用户进入 `ait` 后应立即看到可复用任务和最近运行状态。

必须支持：

- 加载任务列表。
- 展示每个任务的核心配置和最近运行摘要。
- 选择任务。
- 新建任务。
- 启动任务。
- 进入任务详情。
- 删除或复制任务。
- 展示运行中任务状态。

### 4.2 信息优先级

1. 当前选中的任务。
2. 任务名、模式、协议、模型。
3. 最近运行状态、成功率、TTFT、TPS、最近时间。
4. 加载/错误/空状态。

### 4.3 布局设计

任务列表只做一件事：让用户快速选择任务。页面主体必须把纵向空间留给列表，不展示任务详情大块。

#### Normal 布局 `96x30`

```text
AIT · 任务列表                                      tasks=12 running=2
↑/↓ 选择 · Enter 详情 · r 运行 · n 新建 · R 刷新

概览
selected=3/12 · failed_recent=1 · source=summary

任务
> chat-prod · standard · openai-responses · running 32/100
  model=gpt-4.1 · success=98.0% · ttft=320ms · tps=84.2 · 2m ago
  claude-cache · turbo · anthropic-messages · done
  model=claude-3-5 · success=100% · best_c=12 · tps=142.8 · yesterday
```

#### Small 布局 `40x12`

```text
AIT · 任务列表                 12/2
↑/↓ · Enter · r · n

> chat-prod · standard · running 32/100
  claude-cache · turbo · done
  api-smoke · integrity · failed
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 概览 | 2 | 0 | Small 隐藏 |
| 列表标题 | 1 | 0 | Small 隐藏 |
| 任务列表 | 剩余 | 剩余 | 必须保证选中项可见 |
| 提示 | 0-1 | 0 | 只在有空间时显示 |

#### 行格式

主行：`{selector} {name} · {mode} · {protocol} · {status}`
副行：`  model={model} · success={rate} · ttft={duration} · tps={value} · {time}`

Small 只显示主行，且字段顺序固定为：任务名、模式、状态。

### 4.4 状态设计

| 状态 | 展示方式 |
|---|---|
| Loading | `正在加载任务...` |
| Empty | `暂无任务。按 n 创建第一个任务。` |
| Error | 单行错误摘要 + `按 R 重试` |
| Running | 任务主行显示 `running done/total` |
| Selected | 使用 `>` 标记，不使用整行反色大色块 |

### 4.5 实现注意事项

- 不读取请求级数据。
- 不使用表格。
- 每个任务最多 2 行。
- 列表高度由内容区剩余高度决定。
- `Selected` 必须始终在可视范围内。
- 长任务名、模型名、endpoint 必须截断。

---

## 5. 任务详情页

### 5.1 功能目标

任务详情是任务操作中心。它既展示任务配置，也展示当前运行和历史运行。

必须支持：

- 查看任务配置摘要。
- 查看活跃运行摘要。
- 查看历史运行列表。
- 选择历史运行。
- 进入历史运行详情或请求详情。
- 启动新运行。
- 编辑任务。
- 生成报告。

### 5.2 信息优先级

1. 任务名和模式。
2. 核心配置：协议、endpoint、模型、并发/请求数或 Turbo 参数。
3. 当前运行状态。
4. 历史运行列表。
5. 选中历史的摘要指标。

### 5.3 布局设计

任务详情是“配置摘要 + 当前运行 + 历史列表”。历史列表是主区域，配置只保留判断任务是否正确所需字段。

#### Normal 布局 `96x30`

```text
AIT · 任务详情                                      chat-prod · standard
r 运行 · s 停止 · e 编辑 · Enter 历史 · g 报告 · Esc 返回

配置
protocol=openai-responses · model=gpt-4.1 · stream=on
endpoint=https://api.example.com/v1/responses
requests=100 · concurrency=8 · prompt=generated:100

当前运行
running · 32/100 · success=96.9% · ttft=310ms · tps=82.4
████████░░░░░░░░░░ 32%

历史
> 2026-06-11 10:22 · done · 100/100 · success=99.0% · ttft=300ms · tps=83.1
  2026-06-10 22:10 · failed · 42/100 · success=76.2% · error=429 rate limit

选中
run=20260611102201 · duration=1m24s · cache=72.0% · rpm=1200
```

#### Small 布局 `40x12`

```text
AIT · 任务详情              chat-prod
r · e · Enter · Esc

config standard · gpt-4.1 · req=100
running 32/100 · success=96.9%

> 10:22 · done · 100/100 · 99.0%
  22:10 · failed · 42/100 · 76.2%
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 配置 | 4 | 1 | Small 压成一行 |
| 当前运行 | 3 | 1 | 无活跃运行时为 0 |
| 历史列表 | 剩余 | 剩余 | 页面主区域 |
| 选中摘要 | 2 | 0 | Small 隐藏 |

#### 行格式

配置行只显示任务识别和运行必要参数；endpoint 单独一行且截断。历史行固定：`{time} · {state} · {done}/{planned} · success={rate} · {ttft_or_error}`。

### 5.4 状态设计

| 状态 | 展示方式 |
|---|---|
| 无历史 | `暂无历史运行。按 r 启动任务。` |
| 有活跃运行 | 单独显示“当前运行”区块，历史列表仍可浏览 |
| 历史加载失败 | 单行错误，历史区显示降级提示 |
| 请求明细损坏 | 任务详情仍可打开，只显示 run/result 摘要 |

### 5.5 实现注意事项

- 页面不应加载全部请求明细。
- 历史列表只使用 run summary。
- 历史详情只按需加载请求级内容。
- 不使用左右配置/历史分栏。
- 历史摘要最多占 3-4 行，不能挤爆列表。
- `HistorySel` 必须始终可见。

---

## 6. 标准运行页

### 6.1 功能目标

标准运行页用于实时观察压测进度和请求级结果。

必须支持：

- 展示运行状态、进度、完成数。
- 展示核心指标：成功率、TTFT、TPS、RPM、TPM、缓存命中率。
- 展示请求列表。
- 选择请求。
- 进入请求详情。
- 停止运行。
- 后台运行并返回。

### 6.2 信息优先级

1. 是否仍在运行。
2. 完成进度。
3. 成功率和错误趋势。
4. 吞吐与延迟。
5. 最近/选中的请求详情入口。

### 6.3 布局设计

标准运行页只展示实时健康度和请求列表。不要在本页展开请求体、响应体或网络分解。

#### Normal 布局 `96x30`

```text
AIT · 运行                                      chat-prod · standard
b 后台 · s 停止 · Enter 请求详情 · ↑/↓ 选择 · Esc 返回

状态
running · run=abc123 · 32/100 · elapsed=28s
success=96.9% · failed=1 · ttft=310ms · tps=82.4 · rpm=1180 · cache=72.0%
████████░░░░░░░░░░ 32%

请求
> #032 ok · total=820ms · ttft=310ms · tps=84.0 · tokens=982
  #031 ok · total=790ms · ttft=295ms · tps=86.1 · tokens=1004
  #030 failed · total=1.2s · error=429 rate limit
```

#### Small 布局 `40x12`

```text
AIT · 运行                 32/100
b · s · Enter · Esc

running · success=96.9% · tps=82.4
████████░░ 32%

> #032 ok · 820ms · tps=84.0
  #031 ok · 790ms · tps=86.1
  #030 failed · 429 rate limit
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 状态标题 | 1 | 0 | Small 隐藏标题 |
| 状态摘要 | 2 | 1 | Small 只保留 success/tps |
| 进度条 | 1 | 1 | 固定 |
| 请求列表 | 剩余 | 剩余 | 至少 3 行 |

#### 行格式

成功请求：`{selector} #{n} ok · total={duration} · ttft={duration} · tps={value} · tokens={count}`
失败请求：`{selector} #{n} failed · total={duration} · error={summary}`

### 6.4 状态设计

| 状态 | 展示方式 |
|---|---|
| Running | Header meta 显示 `running`，内容显示进度条 |
| Completed | Header meta 显示 `done`，底部提示可生成报告/返回详情 |
| Failed | 单行错误摘要，保留已完成请求列表 |
| Stopping | 状态行显示 `stopping...` |
| Empty Requests | `等待请求完成...` |

### 6.5 实现注意事项

- 不使用表格。
- 请求列表每条 1 行。
- 错误只显示单行摘要。
- 详情内容放到请求详情页，不在列表展开。
- 返回任务列表/详情不能取消订阅，除非用户明确停止。
- 请求列表 `ReqSel` 必须始终可见。

---

## 7. Turbo 运行页

### 7.1 功能目标

Turbo 运行页用于观察并发爬坡过程和最佳稳定承载点。

必须支持：

- 展示当前并发级别。
- 展示已完成 level。
- 展示当前最佳 level。
- 展示停止原因。
- 展示每个 level 的成功率、TTFT、TPS、RPM、缓存命中率。
- 停止、后台、查看详情。

### 7.2 信息优先级

1. 当前 Turbo 阶段。
2. 当前并发和进度。
3. 最佳并发。
4. Level 列表。
5. 停止原因或失败原因。

### 7.3 布局设计

Turbo 页按“当前爬坡状态 + 最佳结果 + level 列表”组织。Level 列表是主体，不展示单条请求明细。

#### Normal 布局 `96x30`

```text
AIT · Turbo                                        cache-test · turbo
b 后台 · s 停止 · Enter 详情 · ↑/↓ 选择 · Esc 返回

状态
running · current_c=16 · level=3/5 · requests=160/300
best_c=12 · success=99.2% · ttft=420ms · tps=142.8 · stop=none
██████████░░░░░░░░ 53%

Levels
> c=16 · running · 40/100 · success=95.0% · ttft=680ms · tps=150.2
  c=12 · best · 100/100 · success=99.2% · ttft=420ms · tps=142.8
  c=8 · done · 100/100 · success=100% · ttft=330ms · tps=110.5
```

#### Small 布局 `40x12`

```text
AIT · Turbo                 c=16
b · s · Enter · Esc

running · 160/300 · best=12
██████████░ 53%

> c=16 · running · 40/100 · 95.0%
  c=12 · best · 100/100 · 99.2%
  c=8 · done · 100/100 · 100%
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 状态摘要 | 2 | 1 | Small 保留 current/best |
| 进度条 | 1 | 1 | 固定 |
| Level 列表 | 剩余 | 剩余 | 当前 level 和 best 必须可见 |
| 停止原因 | 0-1 | 0-1 | 有 stop reason 时替换次要指标 |

#### 行格式

Level 行：`{selector} c={concurrency} · {state} · {done}/{planned} · success={rate} · ttft={duration} · tps={value}`。Small 行去掉 ttft/tps，只保留并发、状态、进度、成功率。

### 7.4 状态设计

| 状态 | 展示方式 |
|---|---|
| Running Level | 当前 level 显示 `running` |
| Best Level | level 行显示 `best` |
| Degraded | 显示停止原因，例如 `latency threshold exceeded` |
| Failed | 显示失败摘要，保留已完成 level |
| Empty | `等待第一个并发级别完成...` |

### 7.5 实现注意事项

- 不使用表格。
- Level 列表每条 1 行，必要时第二行展示补充指标。
- 当前 level、最佳 level 使用文本标签，不依赖颜色识别。
- 保留后台运行和订阅语义。
- 若需要请求级详情，应从选中 level 进入请求列表或请求详情，不在本页展开原始请求体。

---

## 8. 请求详情页

### 8.1 功能目标

请求详情页用于查看单条请求的完整观测结果。

必须支持：

- 查看请求序号、状态、协议、模型。
- 查看耗时拆解：DNS、TCP、TLS、TTFT、总耗时。
- 查看 token、TPS、缓存命中。
- 查看请求输入摘要。
- 查看响应输出摘要或错误摘要。
- 上一条/下一条请求切换。

### 8.2 信息优先级

1. 请求是否成功。
2. 总耗时、TTFT、TPS。
3. 错误摘要或响应摘要。
4. 输入摘要。
5. 网络拆解。

### 8.3 布局设计

请求详情页使用固定 section 顺序：概览、输入、输出。成功和失败必须共用同一套行预算。

#### Normal 布局 `96x30`

```text
AIT · 请求详情                                      chat-prod · #32/100
←/→ 切换请求 · Home/End 首尾 · Esc 返回

概览
status=ok · total=820ms · ttft=310ms · tps=84.0 · tokens=982
cache=hit · dns=12ms · tcp=28ms · tls=70ms · server=200

输入
POST /v1/responses · stream=true · prompt=generated:100
body={"model":"gpt-4.1","input":"..."}

输出
finish=stop · bytes=1842
这是响应内容的受控预览，最多占用剩余高度……
```

#### Small 布局 `40x12`

```text
AIT · 请求 #32/100             ok
←/→ · Esc

ok · total=820ms · ttft=310ms
POST /v1/responses · stream=true

输出
finish=stop · bytes=1842
这是响应内容预览……
```

#### 失败请求同形布局

```text
概览
status=failed · total=1.2s · ttft=- · tps=-
cache=miss · server=429

输入
POST /v1/responses · stream=true · prompt=generated:100
body={"model":"gpt-4.1","input":"..."}

输出
error=429 rate limit exceeded · retry_after=30s
raw={"error":{"message":"rate limit exceeded"...}}
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 概览 | 3 | 1 | Small 合并为一行 |
| 输入 | 3 | 1 | Small 不显示 body |
| 输出标题/元信息 | 2 | 1 | 固定 |
| 输出预览 | 剩余 | 剩余 | 至少 2 行 |

### 8.4 状态设计

| 状态 | 展示方式 |
|---|---|
| Success | 输出区展示响应摘要 |
| Failed | 输出区展示单行错误摘要和原始错误预览 |
| No Body | `无响应体` |
| Long Body | 只展示可见行，末尾显示 `... truncated` |
| Missing Request | `请求记录不存在或已损坏` |

### 8.5 实现注意事项

- 成功和失败路径总高度必须一致。
- 错误消息中的换行必须替换为空格或受控折行。
- 原始请求/响应不能无限展开。
- 页面不使用性能/网络双栏。
- 请求详情可以按需加载请求级数据，但失败必须降级显示，不阻断历史页。

---

## 9. 创建/编辑向导页

### 9.1 功能目标

Wizard 用于创建和编辑任务，不承担运行结果展示职责。

必须支持：

- 三步配置：基础信息、请求参数、模式参数。
- 字段编辑。
- 字段校验。
- 保存任务。
- 编辑已有任务时预填字段。

### 9.2 信息优先级

1. 当前步骤。
2. 当前字段。
3. 当前字段值。
4. 校验错误。
5. 下一步操作提示。

### 9.3 布局设计

Wizard 是表单，不是 dashboard。主体始终是字段列表；说明和错误是辅助区域。

#### Normal 布局 `96x30`

```text
AIT · 新建任务                                      step=1/3 基础
↑/↓ 字段 · Enter 编辑 · Tab 下一步 · Shift+Tab 上一步 · Ctrl+S 保存

步骤
[1 基础] -> 2 请求 -> 3 模式

字段
> name       chat-prod
  protocol   openai-responses
  endpoint   https://api.example.com/v1/responses
  api_key    sk-••••••abcd
  model      gpt-4.1

说明
任务名称用于在列表和历史记录中识别配置。
错误
-
```

#### Small 布局 `40x12`

```text
AIT · 新建任务              1/3
↑/↓ · Enter · Tab · Ctrl+S

[1 基础] -> 2 -> 3
> name      chat-prod
  protocol  openai-responses
  endpoint  https://api.example...
  model     gpt-4.1
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 步骤 | 2 | 1 | Small 简写 |
| 字段列表 | 剩余 | 剩余 | 至少 5 行 |
| 说明 | 2 | 0 | Small 隐藏 |
| 错误 | 1 | 1 | 有错误时优先显示 |

#### 字段行格式

普通：`{selector} {label_padded} {value}`
编辑：`> {label_padded} {input_view_with_cursor}`
敏感：`api_key sk-••••••abcd`

### 9.4 状态设计

| 状态 | 展示方式 |
|---|---|
| Editing | 当前字段行显示输入状态 |
| Invalid | 错误区显示单行错误 |
| Saving | 状态行显示 `saving...` |
| Saved | 返回任务详情或列表 |
| Cancel | 返回来源页面，不保存 |

### 9.5 实现注意事项

- 字段行固定高度，不能因输入过长撑开。
- 当前字段说明最多 2-3 行。
- 错误提示单行化。
- 不使用复杂居中布局。
- 不在 Wizard 中展示运行历史。

---

## 10. 代理配置页

### 10.1 功能目标

代理配置页用于设置应用默认代理。

必须支持：

- 查看当前代理。
- 输入新代理。
- 清空代理。
- 保存代理。
- 返回上一页。

### 10.2 布局设计

代理页是单字段设置页，不能做成弹窗或面板。主区域只展示当前值、输入值和优先级说明。

#### Normal 布局 `96x30`

```text
AIT · 代理配置                                      source=config
Enter 编辑 · Ctrl+S 保存 · Ctrl+L 清空 · Esc 返回

代理
current=http://127.0.0.1:7890
input=http://127.0.0.1:7890
env_proxy=detected · task_proxy_overrides=yes

说明
任务 proxy_url > 全局代理 > 环境变量。清空后将回退到环境变量。
```

#### Small 布局 `40x12`

```text
AIT · 代理配置
Enter · Ctrl+S · Ctrl+L · Esc

current=http://127.0.0.1:7890
input=http://127.0.0.1:7890

任务代理 > 全局代理 > 环境变量
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 代理值 | 4 | 2 | current/input 必显 |
| 说明 | 2 | 1 | Small 压缩为一行 |
| 错误 | 1 | 1 | 有错误时替换说明 |

### 10.3 实现注意事项

- 输入框单行显示，超长截断左侧或右侧，但内部值不丢失。
- 校验错误单行展示。
- 不使用 panel 边框。

---

## 11. 帮助页

### 11.1 功能目标

帮助页用于说明全局快捷键和当前 TUI 工作流。

必须支持：

- 展示全局快捷键。
- 展示页面快捷键分类。
- 展示推荐工作流。
- 滚动查看。
- 返回来源页。

### 11.2 布局设计

帮助页是可滚动文本，不随当前页面动态生成复杂内容。每个分组短句化，方便小窗口阅读。

#### Normal 布局 `96x30`

```text
AIT · 帮助                                          scroll=0
↑/↓ 滚动 · PgUp/PgDn 翻页 · Esc 返回

全局
? 帮助 · F2 语言 · q 退出 · Esc 返回

任务
n 新建 · Enter 详情 · r 运行 · e 编辑 · d 删除 · R 刷新

运行
b 后台 · s 停止 · Enter 请求详情 · ↑/↓ 选择请求

工作流
1. 新建或选择任务
2. 在任务详情启动运行
3. 在运行页观察指标
4. 在请求详情定位失败
```

#### Small 布局 `40x12`

```text
AIT · 帮助                   0%
↑/↓ · PgUp/PgDn · Esc

全局
? 帮助 · F2 语言 · q 退出

任务
n 新建 · Enter 详情 · r 运行
```

#### 区域预算

| 区域 | Normal | Small | 规则 |
|---|---:|---:|---|
| Header | 2 | 2 | 固定 |
| 帮助正文 | 剩余 | 剩余 | 按 scroll 切片 |
| 页脚 | 0-1 | 0 | 可选 |

### 11.3 实现注意事项

- 内容可以长，但必须按高度切片。
- 不使用复杂颜色块。
- 帮助文案保持短句。

---

## 12. 页面详细设计规格（实现版）

本节是实现时的主依据。前文给出方向，本节明确每个页面的用户任务、数据来源、内容区行预算、字段取舍、快捷键、状态、验收条件。

### 12.1 通用页面排布规则

所有页面内容区采用同一套行预算：

```text
Header  第 1 行：AIT · 页面名                         右侧：当前对象/状态
Header  第 2 行：当前页最重要的快捷键，不超过一行
Content 第 1..N 行：按页面定义的单栏 section
Footer  第 1 行：全局提示，例如 [?] 帮助 · [F2] 语言 · [q] 退出
```

内容区 section 只允许以下形式：

```text
SectionTitle
key=value · key=value · key=value
> selected list row
  normal list row
empty/error/loading line
```

禁止页面自行制造“卡片感”。视觉重点通过文字顺序、空行和 `>` 选择标记完成。

#### 12.1.1 行预算原则

页面必须先计算内容区高度 `contentH`，再按优先级分配：

| 类型 | 行数策略 |
|---|---|
| 顶部摘要 | 固定 2-5 行 |
| 进度/状态 | 固定 1-3 行 |
| 列表 | 占用剩余高度 |
| 选中详情 | 最多 3 行；空间不足时隐藏 |
| 说明文本 | 最多 2 行；空间不足时隐藏 |
| 错误文本 | 1 行；多行错误先归一化 |

若空间不足，删除低优先级 section，而不是压缩成乱行。

#### 12.1.2 字段显示规则

- 左侧字段名固定短标签，不用长中文句子。
- 一行最多 4 个键值对。
- 超长值只截断，不自动换到下一行。
- endpoint、model、run id、错误消息默认从右侧截断。
- body 预览允许多行，但行数必须由页面预算控制。

---

### 12.2 任务列表页详细规格

#### 用户任务

用户打开 TUI 后要回答三个问题：

1. 现在有哪些任务？
2. 哪些任务正在运行或最近失败？
3. 我下一步是运行、查看详情，还是新建任务？

#### 数据来源

| 内容 | 来源 | 是否允许读请求明细 |
|---|---|---|
| 任务基础信息 | `TaskListState.Tasks` / `types.TaskOverview` | 否 |
| 活跃运行摘要 | `TaskListState.ActiveRuns` 或运行订阅状态 | 否 |
| 最近运行摘要 | `TaskOverview.LatestRun` | 否 |
| 错误/加载状态 | `TaskListState` | 否 |

任务列表页永远不能读取 `requests.jsonl`。

#### 内容区结构

```text
概览
tasks=12 · running=2 · failed_recent=1 · selected=3/12

任务
> chat-prod · standard · openai-responses · running 32/100
  model=gpt-4.1 · success=98.0% · ttft=320ms · tps=84.2 · 2m ago
  claude-cache · turbo · anthropic-messages · done
  model=claude-3-5 · success=100% · best_c=12 · tps=142.8 · yesterday
```

#### 行预算

| 区块 | 行数 |
|---|---:|
| 概览标题 + 概览值 | 2 |
| 空行 | 1 |
| 列表标题 | 1 |
| 任务列表 | 剩余全部 |
| 底部提示 | 空间足够时 1 |

任务行策略：

- `contentH >= 20`：每个任务 2 行。
- `12 <= contentH < 20`：每个任务 1 行，只显示主行。
- `contentH < 12`：只显示列表，不显示概览。

#### 主行字段

```text
{selector} {name} · {mode} · {protocol} · {status}
```

字段说明：

| 字段 | 示例 | 规则 |
|---|---|---|
| selector | `>` / 空格 | 当前选择项 |
| name | `chat-prod` | 最重要，优先保留 |
| mode | `standard` / `turbo` / `integrity` | 必显 |
| protocol | `openai-responses` | 空间不足可截断 |
| status | `running 32/100` / `done` / `failed` | 必显 |

#### 副行字段

```text
  model={model} · success={rate} · ttft={duration} · tps={value} · {relative_time}
```

Turbo 任务可以将 `ttft` 替换为 `best_c`。

#### 快捷键

| 键 | 行为 |
|---|---|
| `↑/↓` | 移动选择 |
| `Enter` | 进入任务详情 |
| `r` | 运行选中任务 |
| `n` | 新建任务 |
| `e` | 编辑选中任务 |
| `d` | 删除选中任务，需要确认流程 |
| `R` | 重新加载任务 |
| `?` | 帮助 |

#### 状态验收

- 空任务：只显示 `暂无任务。按 n 创建第一个任务。`
- 加载中：只显示 `正在加载任务...`
- 加载失败：显示一行错误 + `按 R 重试`
- 运行中任务必须在不进入详情的情况下看到进度。
- 选择项滚动后必须仍可见。

---

### 12.3 任务详情页详细规格

#### 用户任务

任务详情页是任务的控制台。用户要完成：

1. 确认这个任务配置是否正确。
2. 启动或停止一次运行。
3. 查看最近历史结果。
4. 从历史进入某次运行或请求详情。
5. 编辑任务配置。

#### 数据来源

| 内容 | 来源 | 是否允许读请求明细 |
|---|---|---|
| 任务配置 | `TaskDetailState.Task` | 否 |
| 当前运行 | `TaskDetailState.ActiveRun` | 否 |
| 历史列表 | `TaskDetailState.History` / summary | 否 |
| 选中历史详情 | summary 优先；进入详情时按需加载 | 仅按需 |

#### 内容区结构

```text
配置
name=chat-prod · mode=standard · protocol=openai-responses
model=gpt-4.1 · endpoint=https://api.example.com/v1/responses
concurrency=8 · requests=100 · stream=on · prompt=generated:100

当前运行
running · 32/100 · success=96.9% · ttft=310ms · tps=82.4
████████░░░░░░░░░░ 32%

历史
> 2026-06-11 10:22 · done · 100/100 · success=99.0% · ttft=300ms · tps=83.1
  2026-06-10 22:10 · failed · 42/100 · success=76.2% · error=429 rate limit

选中
run=20260611102201 · duration=1m24s · cache=72.0% · rpm=1200
```

#### 行预算

| 区块 | 行数 |
|---|---:|
| 配置 | 4-5 |
| 当前运行 | 0 或 3 |
| 历史标题 | 1 |
| 历史列表 | 剩余高度 |
| 选中摘要 | 0-3 |

压缩规则：

1. 小窗口先隐藏“选中摘要”。
2. 再隐藏“当前运行”的进度条，只保留状态行。
3. 再将配置压缩为 2 行。
4. 永远保留历史列表至少 3 行。

#### 配置字段

Standard：

```text
name · mode · protocol
model · endpoint
concurrency · requests · timeout · stream
prompt
```

Turbo：

```text
name · mode · protocol
model · endpoint
init_c · max_c · step · level_requests
threshold_success · threshold_latency · prompt
```

Integrity：

```text
name · mode · protocol
model · endpoint
suite · cases · rule_files
```

#### 历史行字段

```text
{selector} {time} · {state} · {done}/{planned} · success={rate} · ttft={duration} · tps={value_or_error}
```

失败历史优先展示错误摘要而不是 TPS。

#### 快捷键

| 键 | 行为 |
|---|---|
| `r` | 启动运行 |
| `s` | 停止当前运行，只有活跃运行时显示 |
| `e` | 编辑任务 |
| `Enter` | 打开选中历史运行详情 |
| `g` | 生成报告 |
| `↑/↓` | 选择历史 |
| `Esc` | 返回任务列表 |

#### 状态验收

- 没有历史时必须告诉用户按 `r` 启动。
- 历史 summary 损坏时只影响该行，不影响整个页面。
- 请求 JSONL 损坏不能导致任务详情打不开。
- 活跃运行离开页面后仍能继续更新。

---

### 12.4 标准运行页详细规格

#### 用户任务

用户在标准运行页主要观察一次运行是否健康：

1. 请求是否还在推进？
2. 成功率是否下降？
3. TTFT/TPS/RPM 是否达标？
4. 哪些请求失败，需要查看详情？
5. 是否要后台运行或停止？

#### 数据来源

| 内容 | 来源 |
|---|---|
| 运行状态 | `DashboardState.RunState` |
| 请求列表 | `RunState.Requests` 已加载部分 |
| 订阅状态 | `DashboardState.EventCh` / `CancelFn` |
| 任务名 | Model 传入 `taskName` |

#### 内容区结构

```text
状态
running · run=abc123 · 32/100 · elapsed=28s
success=96.9% · failed=1 · ttft=310ms · tps=82.4 · rpm=1180 · cache=72.0%
████████░░░░░░░░░░ 32%

请求
> #032 ok · total=820ms · ttft=310ms · tps=84.0 · tokens=982
  #031 ok · total=790ms · ttft=295ms · tps=86.1 · tokens=1004
  #030 failed · total=1.2s · error=429 rate limit
```

#### 行预算

| 区块 | 行数 |
|---|---:|
| 状态标题 | 1 |
| 状态行 | 1 |
| 指标行 | 1 |
| 进度条 | 1 |
| 空行 | 1 |
| 请求标题 | 1 |
| 请求列表 | 剩余全部 |

小窗口压缩：

- 隐藏 run id。
- 隐藏 cache/RPM/TPM，只保留 success/ttft/tps。
- 请求列表保留至少 4 行。

#### 请求行字段

成功：

```text
{selector} #{n} ok · total={duration} · ttft={duration} · tps={value} · tokens={in+out}
```

失败：

```text
{selector} #{n} failed · total={duration} · error={summary}
```

等待/进行中：

```text
{selector} #{n} pending · started={relative_time}
```

#### 快捷键

| 键 | 行为 |
|---|---|
| `b` | 后台运行并返回来源页 |
| `s` | 停止运行 |
| `Enter` | 查看选中请求详情 |
| `↑/↓` | 选择请求 |
| `PgUp/PgDn` | 翻页 |
| `Esc` | 返回任务详情，不停止运行 |

#### 状态验收

- 没有请求时显示 `等待请求完成...`。
- 停止中显示 `stopping...`。
- 完成后状态变 `done`，仍显示请求列表。
- 失败运行保留已完成请求，不用错误页替换整个页面。

---

### 12.5 Turbo 运行页详细规格

#### 用户任务

Turbo 页回答：

1. 当前爬坡到哪个并发？
2. 哪个并发目前最好？
3. 是否已经触发停止条件？
4. 每个 level 的成功率、延迟和吞吐如何？

#### 数据来源

| 内容 | 来源 |
|---|---|
| 当前 Turbo 状态 | `TurboDashState.RunState` |
| level 结果 | `RunState.TurboLevels` 或结果聚合 |
| 当前请求 | `RunState.Requests` |
| 停止原因 | Turbo result / run error |

#### 内容区结构

```text
状态
running · current_c=16 · level=3/5 · requests=160/300
best_c=12 · success=99.2% · ttft=420ms · tps=142.8 · stop=none
██████████░░░░░░░░ 53%

Levels
> c=16 · running · 40/100 · success=95.0% · ttft=680ms · tps=150.2
  c=12 · best · 100/100 · success=99.2% · ttft=420ms · tps=142.8
  c=8  · done · 100/100 · success=100% · ttft=330ms · tps=110.5
```

#### 行预算

与标准运行页一致，但列表标题为 `Levels`。

#### Level 行字段

```text
{selector} c={concurrency} · {state} · {done}/{planned} · success={rate} · ttft={duration} · tps={value}
```

可选第二行，仅在宽高足够时展示：

```text
  rpm={rpm} · tpm={tpm} · cache={rate} · p95={duration}
```

#### 快捷键

| 键 | 行为 |
|---|---|
| `b` | 后台运行 |
| `s` | 停止运行 |
| `Enter` | 查看选中 level 或请求详情 |
| `↑/↓` | 选择 level |
| `Esc` | 返回任务详情 |

#### 状态验收

- 当前 level 必须可见。
- best level 必须有文本标签 `best`，不能只靠颜色。
- 触发降级时显示 stop reason。
- 没有 level 时显示 `等待第一个并发级别完成...`。

---

### 12.6 请求详情页详细规格

#### 用户任务

请求详情页用于定位单次请求问题：

1. 请求成功还是失败？
2. 慢在哪里：DNS/TCP/TLS/TTFT/总耗时？
3. 输入是什么？
4. 服务返回了什么？
5. 是否能快速切换到前后请求对比？

#### 数据来源

| 内容 | 来源 |
|---|---|
| 请求指标 | `ReqDetailState.Requests[ReqIndex]` |
| 请求序号 | `ReqDetailState.Index` |
| 来源页面 | `ReqDetailState.Back` |
| 任务名 | Model 传入 `taskName` |

#### 内容区结构

```text
概览
#32/100 · status=ok · total=820ms · ttft=310ms · tps=84.0 · tokens=982
cache=hit · dns=12ms · tcp=28ms · tls=70ms · server=200

输入
method=POST · path=/v1/responses · stream=true
prompt=generated:100 · body={"model":"gpt-4.1",...}

输出
finish=stop · bytes=1842
这是响应内容的受控预览，最多占用剩余高度……
```

失败输出：

```text
输出
error=429 rate limit exceeded · total=1.2s
raw={"error":{"message":"rate limit exceeded","type":"rate_limit"...}}
```

#### 行预算

| 区块 | 行数 |
|---|---:|
| 概览 | 3 |
| 输入 | 2-4 |
| 输出标题 + 元信息 | 2 |
| 输出预览 | 剩余全部 |

小窗口压缩：

1. 概览压缩为 2 行。
2. 输入压缩为 1 行。
3. 输出预览至少保留 3 行。

#### 文本规则

- `ErrorMessage` 先执行 `NormalizeInlineText`。
- request/response body 先按行切片，再逐行截断。
- 原始 JSON 不做漂亮格式化；漂亮格式化会增加高度不确定性。
- 成功和失败都走同一个行预算函数。

#### 快捷键

| 键 | 行为 |
|---|---|
| `←/h` | 上一条请求 |
| `→/l` | 下一条请求 |
| `Esc` | 返回来源页 |
| `Home/End` | 第一条/最后一条，可选 |

#### 状态验收

- 请求缺失时显示 `请求记录不存在或已损坏`。
- 成功和失败渲染高度一致。
- 多行错误不会增加页面结构高度。
- 超长 body 不超高、不超宽。

---

### 12.7 创建/编辑向导页详细规格

#### 用户任务

Wizard 只负责配置任务：

1. 输入任务基础信息。
2. 配置请求协议和模型。
3. 配置测试模式参数。
4. 保存或取消。

#### 数据来源

| 内容 | 来源 |
|---|---|
| 表单字段 | `WizardState` |
| 编辑任务 | `WizardState.EditTask` |
| 校验错误 | `WizardState.Err` / validation result |
| 保存状态 | create/update command result |

#### 步骤设计

| Step | 名称 | 字段 |
|---:|---|---|
| 1 | 基础 | name, protocol, endpoint, api key, model |
| 2 | 请求 | prompt mode, prompt text/file/length, stream, timeout, proxy |
| 3 | 模式 | standard/turbo/integrity 对应参数 |

#### 内容区结构

```text
步骤
[1 基础] -> 2 请求 -> 3 模式

字段
> 名称      chat-prod
  协议      openai-responses
  Endpoint  https://api.example.com/v1/responses
  Model     gpt-4.1

说明
任务名称用于在列表和历史记录中识别配置。

错误
Endpoint 不能为空
```

#### 行预算

| 区块 | 行数 |
|---|---:|
| 步骤条 | 2 |
| 字段标题 | 1 |
| 字段列表 | 剩余高度 - 说明/错误 |
| 说明 | 0-3 |
| 错误 | 0-1 |

字段列表至少保留 5 行。空间不足时隐藏说明，保留错误。

#### 字段行规则

普通状态：

```text
{selector} {label_padded} {value}
```

编辑状态：

```text
> {label_padded} {input_cursor_view}
```

敏感值：

```text
API Key   sk-••••••abcd
```

#### 快捷键

| 键 | 行为 |
|---|---|
| `↑/↓` | 选择字段 |
| `Enter` | 编辑/确认字段 |
| `Tab` | 下一步 |
| `Shift+Tab` | 上一步 |
| `Ctrl+S` | 保存 |
| `Esc` | 取消编辑或返回来源页 |

#### 状态验收

- 编辑已有任务时字段必须完整预填。
- 输入超长不会撑开字段行。
- 校验错误只占一行。
- Step 3 根据 mode 切换字段，不显示无关参数。

---

### 12.8 代理配置页详细规格

#### 用户任务

用户要查看、设置或清空全局代理。

#### 内容区结构

```text
代理
current=http://127.0.0.1:7890
input=http://127.0.0.1:7890
source=config · env_proxy=detected

说明
任务中的 proxy_url 优先于全局代理；全局代理优先于环境变量。
```

#### 快捷键

| 键 | 行为 |
|---|---|
| `Enter` | 编辑输入 |
| `Ctrl+S` | 保存 |
| `Ctrl+L` | 清空 |
| `Esc` | 返回来源页 |

#### 状态验收

- 保存中显示 `saving...`。
- 保存失败显示一行错误。
- 清空不需要用户手动删除长字符串。
- 超长代理 URL 不超宽。

---

### 12.9 帮助页详细规格

#### 用户任务

用户需要快速查到当前可用按键和推荐操作路径。

#### 内容结构

```text
全局
? 帮助 · F2 语言 · q 退出 · Esc 返回

任务
n 新建 · Enter 详情 · r 运行 · e 编辑 · d 删除

运行
b 后台 · s 停止 · Enter 请求详情

工作流
1. 新建或选择任务
2. 在任务详情启动运行
3. 在运行页观察指标
4. 在请求详情定位失败
```

#### 快捷键

| 键 | 行为 |
|---|---|
| `↑/↓` | 滚动 |
| `PgUp/PgDn` | 翻页 |
| `Esc` | 返回来源页 |

#### 状态验收

- 帮助内容可以滚动，但任意时刻输出不超高。
- 不按页面做复杂动态帮助；保持短句。

---

## 13. 页面间导航设计

```text
TaskList
  ├─ n/e -> Wizard
  ├─ Enter -> TaskDetail
  └─ r -> Dashboard 或 TurboDash

TaskDetail
  ├─ r -> Dashboard 或 TurboDash
  ├─ e -> Wizard
  ├─ Enter(history) -> Run/History detail 或 ReqDetail
  └─ Esc -> TaskList

Dashboard / TurboDash
  ├─ Enter(request/level) -> ReqDetail
  ├─ b -> TaskList 或 TaskDetail，保持后台订阅
  ├─ s -> StopRun
  └─ Esc -> TaskDetail

ReqDetail
  ├─ ←/→ -> 前后请求
  └─ Esc -> 来源页

Wizard / Proxy / Help
  └─ Esc -> 来源页
```

导航原则：

1. 返回路径必须明确，不允许循环迷路。
2. 从运行页离开不等于停止运行。
3. 停止运行必须是显式动作。
4. 历史详情不能因为请求明细损坏而无法打开。

---

## 14. 实现分层注意事项

### 14.1 页面渲染层

- 页面 Render 函数只负责把当前状态转成文本行。
- 页面不得直接访问磁盘。
- 页面不得做请求级大数据加载。
- 页面不得依赖具体终端颜色表达语义，文本标签必须完整。

### 14.2 状态层

- 选择索引必须 clamp 到合法范围。
- 滚动 offset 必须保证 selected 可见。
- loading、error、empty 是一等状态，不能用空字符串糊弄。
- 运行订阅生命周期由导航/Model 层管理，页面只展示状态。

### 14.3 文本处理

- 所有外部文本进入页面前都应经过：trim、换行归一化、宽度截断。
- CJK 宽度必须使用现有 shared 文本工具或 lipgloss 宽度计算。
- 错误消息默认单行，只有输出预览区允许受控多行。

### 14.4 测试要求

每个页面至少覆盖：

1. 正常状态。
2. 空状态。
3. 错误状态。
4. 小窗口 `40x12`。
5. 常规窗口 `96x30`。
6. 长文本。
7. 中文内容。
8. 输出不超宽、不超高。

---

## 15. 重写顺序

推荐顺序：

1. `layout.go`、`helpers.go`：建立最终输出约束和基础组件。
2. `tasklist.go`：首页先稳定。
3. `taskdetail.go`：任务中心稳定。
4. `dashboard.go`：标准运行页稳定。
5. `turbodash.go`：Turbo 运行页稳定。
6. `reqdetail.go`：请求详情稳定。
7. `wizard.go`：配置编辑稳定。
8. `proxy.go`、`help.go`：收尾统一。
9. `styles.go`：清理未使用复杂样式。
10. `*_test.go`：补齐尺寸和导航测试。

---

## 16. 验收标准

实现完成后必须满足：

1. 全仓 TUI 页面无 `lipgloss/table`。
2. 全仓 TUI 页面无左右 `Split` 布局。
3. 所有页面输出不超过终端宽高。
4. 所有页面在 `40x12` 下可读且不乱。
5. 任务列表不加载请求级明细。
6. 任务详情不因请求 JSONL 损坏失败。
7. Dashboard/TurboDash 可后台返回且不停止运行。
8. 请求详情成功/失败高度一致。
9. 中文和长错误信息不会撑乱布局。
10. `go test ./internal/tui/... -count=1` 和 `go test ./... -count=1` 通过。
