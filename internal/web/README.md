# AIT Web 接口需求分析

当前 `internal/web` 只负责启动本地静态 SPA：`web.Run` 挂载 `/` 到前端产物，尚未注册任何 `/api` 接口。前端原型的数据来源是 `internal/web/source/mock.ts`，本文件根据当前 Web 页面反推接口需求，并对照 `internal/server.Server` 已有业务能力标记缺口。

## 页面与数据需求

Web 当前是任务中心结构：任务列表 → 任务详情 → 执行记录 → 单次请求详情。创建任务是 Dialog 分步表单，当前仍为 mock-only。

### 任务列表

需要展示：

- 任务 ID、名称、模式：`standard` / `turbo` / `integrity`
- 协议、模型、请求地址
- 并发数、请求总数、更新时间
- 最近一次执行摘要：状态、成功率、TTFT、TPS、缓存命中、错误摘要
- 任务搜索需要可检索名称、协议、模型等字段

### 任务详情

需要展示：

- 任务基础配置：名称、模式、协议、模型、endpoint、并发、请求总数
- 标准压测配置：Prompt、Prompt 来源、timeout、stream、thinking、report、log
- Turbo 配置：Prompt、初始并发、最大并发、递增步长、每级请求数、最低成功率、最大延迟
- 完整性校验配置：测试集、规则文件、fail fast、单 case timeout、case 列表
- 原始 `TaskConfig.Input` JSON，用于确认最终写入后端的配置

### 执行记录列表

需要展示指定任务的历史运行：

- run ID、状态、开始时间、结束时间/耗时
- 总请求数、成功数、失败数、成功率、错误率
- TTFT、TPOT、TPS、RPM、TPM、吞吐、缓存命中
- DNS、Connect、TLS、目标 IP 等网络指标摘要
- Turbo 的最大稳定并发、Integrity 的失败 case 摘要

### 单次运行详情

需要展示：

- 运行状态快照：queued/running/completed/failed/stopped
- 运行进度：总数、排队、运行中、完成、成功、失败、跳过
- 聚合指标：成功率、TTFT、TPS、RPM、TPM、缓存命中
- 模式状态：Turbo level 进度、Integrity case/assertion 进度
- 最终模式结果：standard report、turbo result、integrity result

### 单个请求详情

需要展示某次运行下的请求样本：

- index、状态、level、case ID
- 总耗时、TTFT、TPS
- prompt tokens、completion tokens、cached tokens、cache hit
- DNS、Connect、TLS、target IP
- 请求体、响应体、错误信息

### 创建任务

需要提交：

- 通用字段：`name`、`input.mode`、`input.protocol`、`input.endpoint_url`、`input.model`
- standard：`input.concurrency`、`input.count`、`input.prompt_*`、`input.timeout`、`input.stream`、`input.thinking`、`input.report`、`input.log`
- turbo：`input.mode=turbo`、`input.turbo=true`、`input.count`、`input.prompt_*`、`input.turbo_config`
- integrity：`input.mode=integrity`、`input.integrity.enabled=true`、`suite`、`rule_files`、`fail_fast`、`case_timeout_ms`

任务没有“标签”和“任务说明”这两个用户配置项；Web 不应提交这两个字段。

## 建议 HTTP API

建议所有接口挂到 `/api`，SPA fallback 继续处理非 `/api` 路由。

| 方法 | 路径 | Web 用途 | 后端业务能力 | 状态 |
| --- | --- | --- | --- | --- |
| `GET` | `/api/tasks` | 任务列表 + 最近执行摘要 | `Server.ListTasks()` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/tasks/{taskID}` | 任务详情/复制任务预填 | `Server.GetTask(id)` | 业务层已有，HTTP 缺失 |
| `POST` | `/api/tasks` | 创建任务 | `Server.CreateTask(TaskConfig)` | 业务层已有，HTTP 缺失 |
| `PUT` | `/api/tasks/{taskID}` | 更新任务 | `Server.UpdateTask(id, TaskConfig)` | 业务层已有，HTTP 缺失 |
| `DELETE` | `/api/tasks/{taskID}` | 删除任务 | `Server.DeleteTask(id)` | 业务层已有，HTTP 缺失 |
| `POST` | `/api/tasks/{taskID}/duplicate` | 快速复制任务 | `Server.DuplicateTask(id)` | 业务层已有，HTTP 缺失 |
| `POST` | `/api/tasks/{taskID}/runs` | 启动运行 | `Server.StartRun(taskID)` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/tasks/{taskID}/runs?limit=20` | 任务执行记录 | `Server.ListTaskRunHistory(taskID, limit)` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/runs/{runID}` | 运行状态/历史运行详情 | `Server.GetRunState(runID)` | 业务层已有，HTTP 缺失 |
| `POST` | `/api/runs/{runID}/stop` | 停止运行 | `Server.StopRun(runID)` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/runs/{runID}/events` | 运行实时更新 | `Server.SubscribeRunEvents(runID)` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/runs/{runID}/requests` | 请求明细表/曲线样本 | 可通过 `GetRunState(runID).Requests` 获得 | 业务层间接已有，HTTP 缺失 |
| `GET` | `/api/runs/{runID}/requests/{index}` | 单请求详情 | 可从 `RunState.Requests` 查找 | 业务层间接已有，HTTP 缺失 |
| `GET` | `/api/runs/{runID}/report?format=json|csv` | 下载报告 | `Server.GenerateRunReport(runID, format)` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/config` | 全局配置，如代理 | `Server.GetAppConfig()` | 业务层已有，HTTP 缺失 |
| `PUT` | `/api/config/proxy` | 更新代理 | `Server.UpdateProxyURL(proxyURL)` | 业务层已有，HTTP 缺失 |
| `GET` | `/api/integrity/suites` | 创建完整性任务时选择测试集 | `RulesManager` 内部有加载能力 | Web 需要，公开查询接口缺失 |
| `GET` | `/api/integrity/suites/{suiteID}` | 查看 suite/case/assertion 元数据 | `integrity.LoadSuiteWithManager` 可加载 | Web 需要，公开查询接口缺失 |
| `GET` | `/api/meta/protocols` | 创建任务协议选项、默认 endpoint | `types.NormalizeProtocol` / `DefaultEndpointURL` | 可静态实现，HTTP 缺失 |

## 数据模型映射

### 任务

后端已有：

- `types.TaskDefinition`
- `types.TaskOverview`
- `types.Input`

Web 不需要单独的 `description/tags` 输入。任务列表中如果需要辅助展示，可从 `Input.RunMode()`、协议、模型、最近执行摘要派生。

### 执行摘要

后端已有：

- `types.TaskRunSummary`
- `store.StoredRun.Summary()`
- `RunStore.ListSummariesByTask()`

当前摘要字段能覆盖任务列表和执行记录的大部分需求。Web 还想展示的 `failed/success/count/errorRate/duration/min/max/stddev/network summary` 等字段，部分只存在于 `RunState.Requests` 或 `ReportData`，建议在 API DTO 中统一派生，避免前端直接理解多种后端结果结构。

### 运行详情

后端已有：

- `server.RunState`
- `types.RequestMetrics`
- `ModeState`
- `ModeResult`

`GetRunState` 已支持：先读内存 active run，找不到则从磁盘恢复历史运行，并尽量补充 `requests.jsonl`。

### 实时事件

后端已有：

- `server.Event`
- `SubscribeRunEvents(runID)`
- `EventRunQueued` / `EventRunStarted` / `EventRequestDone` / `EventProgressTick` / `EventLevelDone` / `EventIntegrityCaseDone` 等

Web 建议用 SSE：`GET /api/runs/{runID}/events`，事件 payload 直接使用 `Event` 的 JSON DTO。

## 缺失清单

### 1. Web HTTP API 层缺失

`internal/web/web.go` 目前只注册了：

- `/` → SPA 静态资源

缺少：

- API router
- JSON 编解码与错误响应
- 路径参数解析
- SSE writer
- `server.New()` 生命周期管理
- API 与 SPA fallback 的路由隔离

### 2. Web DTO/适配层缺失

不建议前端直接消费内部 Go 结构，原因：

- `time.Duration` JSON 是纳秒数字，不适合 UI 直接展示
- `ModeResult` / `ModeState` 是 `any`，跨 JSON 后类型不稳定
- Web 需要中文标签、格式化耗时、派生状态、请求详情合并字段

建议新增 `internal/web/api` 或 `internal/web/adapter`：

- `TaskDTO`
- `TaskRunDTO`
- `RunStateDTO`
- `RequestDTO`
- `CreateTaskRequest`
- `UpdateTaskRequest`
- `ErrorResponse`

### 3. 请求明细独立接口缺失

业务层能通过 `GetRunState(runID).Requests` 拿到请求，但没有直接面向 Web 的分页/筛选接口。

建议：

- `GET /api/runs/{runID}/requests?offset=0&limit=100&status=failed`
- `GET /api/runs/{runID}/requests/{index}`

这样请求量大时不必一次性把所有请求体/响应体传给前端。

### 4. Integrity suite 查询接口缺失

创建完整性任务需要知道可用测试集和 case。当前运行时可加载 suite，但没有 Web 查询接口。

建议：

- 列出内置/已下载 suite
- 查看 suite 下 case、capability、assertions、required、timeout
- 可选：触发规则更新/刷新

### 5. 任务创建校验接口缺失

业务层 `CreateTask` 会持久化任务，但 Web 创建前最好能做结构校验与默认值归一化。

建议二选一：

- `POST /api/tasks/validate`
- 或在 `POST /api/tasks` 返回结构化字段错误

至少需要检查：

- `name` 非空
- `protocol/model/endpoint` 合法
- standard/turbo 必须有 Prompt
- integrity 必须有 suite/rule files
- duration 字符串可解析
- turbo 并发参数关系合法

## 建议优先级

1. **第一批：任务与运行只读接口**
   - `GET /api/tasks`
   - `GET /api/tasks/{taskID}`
   - `GET /api/tasks/{taskID}/runs`
   - `GET /api/runs/{runID}`
   - `GET /api/runs/{runID}/requests`

2. **第二批：任务生命周期接口**
   - `POST /api/tasks`
   - `PUT /api/tasks/{taskID}`
   - `DELETE /api/tasks/{taskID}`
   - `POST /api/tasks/{taskID}/duplicate`

3. **第三批：运行控制与实时更新**
   - `POST /api/tasks/{taskID}/runs`
   - `POST /api/runs/{runID}/stop`
   - `GET /api/runs/{runID}/events`

4. **第四批：辅助元数据**
   - `GET /api/meta/protocols`
   - `GET /api/integrity/suites`
   - `GET /api/integrity/suites/{suiteID}`
   - `GET /api/config`
   - `PUT /api/config/proxy`

## 结论

AIT 的 server 业务层已经覆盖了大多数 Web 所需能力：任务 CRUD、复制、运行启动/停止、历史记录、运行状态、事件订阅、报告生成、全局代理配置。当前真正缺的是 `internal/web` 下的 HTTP API 层、Web DTO 适配层，以及少量面向创建表单的元数据/校验接口。
