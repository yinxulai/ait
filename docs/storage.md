# 存储设计（精简目标方案）

本文定义下一版存储架构的目标约束：

- 只存业务数据。
- 不存查询视图。
- 不存中间态快照。
- 不存可以稳定从底层业务数据重新聚合出来的数据副本。

如果某份数据既不是业务本体，也不是最终必须落库的业务结果，就不进入持久化层。

## 1. 只存什么

下一版只持久化四类数据：

1. 配置。
2. 任务。
3. 请求明细。
4. 测试结果指标。

除此之外，默认都不存。

## 2. 明确不存什么

以下内容不进入持久化层：

1. 任务列表缓存。
2. 历史列表索引。
3. active runs 视图。
4. recent runs 视图。
5. last run summary 之类的冗余摘要。
6. 运行中的快照文件。
7. lifecycle 事件流。
8. Turbo 级别事件日志。
9. 报告产物清单。
10. schema 文件。
11. task meta、notes、tags、owner 这类当前非核心业务字段。

这些内容要么属于界面视图，要么属于运行时控制信息，要么可以从底层业务数据重新计算。

## 3. 设计原则

### 3.1 单份业务事实

同一类业务信息只保存一份：

- 任务定义只保存在任务文件里。
- 请求事实只保存在请求日志里。
- 最终测试指标只保存在结果文件里。

### 3.2 不为查询速度存冗余副本

任务列表、任务历史、最近运行、当前运行，都通过扫描任务文件和运行目录实时构造。

本方案优先保证存储模型干净，而不是用额外索引换读取速度。

### 3.3 运行态不落盘

运行中的中间进度、实时聚合值、仪表盘卡片数据都只存在内存。

落盘只发生在两种时机：

1. 请求完成，追加请求事实。
2. 运行结束，写最终指标结果。

### 3.4 结果允许保留一份最终聚合

虽然最终指标可以从请求明细重新计算，但“最终测试指标”本身属于业务结果，因此允许保留一份最终结果文件。

约束是：

- 只保留一份最终结果。
- 不再额外保留面向不同页面的摘要副本。

## 4. 目录布局

推荐目录布局如下：

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

这就是完整持久化集合，不再额外引入 views、artifacts、meta、snapshot 等目录。

## 5. 核心数据模型

### 5.1 config.json

只保存全局配置，例如：

- 默认协议
- 上次选中的任务
- 是否保存 API Key

不保存任何运行态信息。

### 5.2 tasks/<task-id>.json

每个任务只对应一个文件。

建议字段：

- task_id
- name
- input
- created_at
- updated_at

这里的 input 就是任务配置本体：

- 协议
- endpoint/base_url
- model
- concurrency
- count
- stream
- thinking
- turbo 配置
- prompt 配置
- timeout

不再在任务文件里嵌入：

- last_run_at
- last_run_summary
- history
- report path

因为这些都不是任务定义本身。

### 5.3 runs/<task-id>/<run-id>/run.json

run.json 只保存最小运行元数据，用来描述这次运行属于谁、何时开始、何时结束、最终状态是什么。

建议字段：

- run_id
- task_id
- mode
- protocol
- model
- status
- started_at
- finished_at

它不承担指标存储，不承担请求明细存储，也不承担列表视图存储。

### 5.4 runs/<task-id>/<run-id>/requests.jsonl

这是请求级业务事实源。

每行一条请求记录，建议至少包含：

- request_index
- success
- total_time
- ttft
- tps
- input_tokens
- output_tokens
- cached_tokens
- cache_hit_rate
- dns_time
- connect_time
- tls_time
- target_ip
- error_message
- request_body
- response_body

所有请求详情、离线分析、结果复算，都以这里为准。

### 5.5 runs/<task-id>/<run-id>/result.json

这是单次运行的最终业务结果。

建议保存：

- 总请求数
- 成功率 / 错误率
- TTFT / TPS / 总耗时等最终聚合指标
- 输入输出 token 统计
- Turbo 模式的最终级别结果
- 最终错误摘要（如有）

这个文件只在运行结束后写一次。

它是最终对外展示的“测试结果”，而不是运行中的中间态快照。

## 6. 为什么不再存其他内容

### 6.1 不存任务历史索引

任务历史本质上可以通过扫描 `runs/<task-id>/` 下的 run 目录得到。

因此不再单独维护：

- `history/<task-id>.json`
- task-runs 视图文件

### 6.2 不存任务列表摘要

任务列表可以通过扫描 `tasks/` 目录构造。

因此不再单独维护：

- `tasks.json`
- task-list view
- last_run_summary 类冗余字段

### 6.3 不存运行中快照

运行中快照本质上是 UI 关心的中间态，不是必须持久化的业务数据。

因此不再存：

- snapshot.json
- active-runs.json
- recent-runs.json

如果进程退出，运行中状态丢失是可接受的；真正的业务结果以已落盘的请求记录和最终 result 为准。

### 6.4 不存导出产物清单

报告是导出物，不属于核心业务数据。

下一版建议：

- 报告按需生成。
- 默认不纳入主存储目录。
- 如果用户要导出，就直接输出到指定路径。

因此不再存：

- artifacts.json
- report 路径索引
- report availability 标志

### 6.5 不存额外生命周期日志

状态变更、停止原因、页面卡片状态等，如果不是最终 run.json 必须字段，就不单独落日志。

因此不再存：

- lifecycle.jsonl
- 中间状态事件流

## 7. 写入模型

### 7.1 创建任务

只写一个任务文件：

1. 写 `tasks/<task-id>.json`

### 7.2 更新任务

只覆盖任务文件：

1. 覆盖 `tasks/<task-id>.json`

### 7.3 启动运行

启动运行时只写最小运行元数据：

1. 创建 `runs/<task-id>/<run-id>/`
2. 写 `run.json`，状态为 running

不写摘要，不写视图，不写任务回填字段。

### 7.4 运行中

每完成一个请求：

1. 追加一条到 `requests.jsonl`

不写快照，不写任务历史，不写任务摘要。

### 7.5 运行结束

运行结束时：

1. 更新 `run.json` 的最终状态和结束时间
2. 写 `result.json`

到此结束，不做额外索引更新。

## 8. 读取模型

### 8.1 任务列表

通过扫描 `tasks/` 目录得到。

### 8.2 任务历史

通过扫描 `runs/<task-id>/` 下的所有 run 目录得到。

历史卡片需要的展示字段，来自：

- `run.json`
- `result.json`

### 8.3 单次运行详情

单次运行详情只读三类文件：

1. `run.json`
2. `requests.jsonl`
3. `result.json`

没有 fallback 文件，没有额外索引文件。

## 9. Repository 建议

下一版 `internal/store` 可以收敛成最小集合：

```text
internal/store/
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

## 10. 代价与取舍

这个方案的代价很明确：

1. 任务列表和任务历史读取时需要扫描目录。
2. 运行中的跨进程恢复能力会变弱。
3. 某些 UI 页面首次读取会比“预存视图”慢。

但换来的好处更符合目标：

1. 存储模型极简。
2. 不再维护多份互相覆盖的摘要。
3. 高低频写路径都更清晰。
4. 数据结构更接近业务本体。

## 11. 一句话总结

下一版只存：

- 配置
- 任务
- 请求明细
- 最终测试指标

其他一律不存；需要展示时，从这四类业务数据实时聚合。
