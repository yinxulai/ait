# AIT 规则系统完整指南

本文档是 AIT 规则系统的完整参考，包含内置规则和本地规则的使用和管理。

## 目录

- [概述](#概述)
- [快速开始](#快速开始)
- [规则类型](#规则类型)
  - [内置规则](#内置规则)
  - [本地规则](#本地规则)
- [规则格式](#规则格式)
- [使用指南](#使用指南)
- [版本管理](#版本管理)
- [缓存机制](#缓存机制)
- [故障排查](#故障排查)
- [最佳实践](#最佳实践)
- [API 参考](#api-参考)

---

## 概述

AIT 提供了灵活的规则加载系统，支持两种规则来源：

| 类型 | 说明 | 位置 | 更新方式 |
|-----|------|------|---------|
| **内置规则** | 从 AIT 仓库自动下载 | `~/.ait/rules/` | 后台检查版本更新 |
| **本地规则** | 项目本地文件 | 用户指定路径 | 手动管理 |

### 规则优先级

当使用内置规则管理器时，规则按以下顺序合并：

1. 内置规则（自动加载）
2. 用户规则（覆盖同 ID 的内置规则）

---

## 快速开始

### 1. 使用内置规则（推荐）

最简单的方式，直接引用规则套件 ID：

```yaml
tasks:
  - name: openai-test
    protocol: openai-completions
    suite: openai-completions-smoke  # 自动使用内置规则
    cases:
      - id: test-1
        model: gpt-4
        messages:
          - role: user
            content: "Hello"
```

### 2. 使用本地规则

使用项目中的规则文件：

```yaml
tasks:
  - name: project-test
    protocol: openai-completions
    suite: ./rules/project-rules.json
    cases:
      - id: test-1
        model: gpt-4
```

---

## 规则类型

### 内置规则

#### 功能特性

- ✅ **自动下载**：首次运行时从 GitHub 下载最新规则
- ✅ **本地缓存**：缓存在 `~/.ait/rules/`
- ✅ **智能更新**：后台检查版本号，有新版本自动更新
- ✅ **版本兼容**：根据 AIT 版本自动选择兼容规则
- ✅ **多更新源**：stable/latest/dev 三种更新源
- ✅ **离线容错**：网络失败时使用已有缓存

#### 可用的内置规则

| 规则 ID | 协议 | 套件 | 说明 |
|--------|------|------|------|
| `openai-completions` | openai-completions | openai-completions-smoke | OpenAI Completions API 基础检查 |
| `openai-responses` | openai-responses | openai-responses-smoke | OpenAI Responses API 基础检查 |
| `anthropic-messages` | anthropic-messages | anthropic-messages-smoke | Anthropic Messages API 基础检查 |
| `common-latency` | * | * | 通用延迟检查 |

#### 存储位置

```
~/.ait/rules/
├── index.json              # 规则索引
└── integrity/              # 规则文件
    ├── openai-completions.json
    ├── openai-responses.json
    ├── anthropic-messages.json
    └── common-latency.json
```

#### 更新源

根据 AIT 版本自动选择：

- **Stable**（`v1.0.0` 等）：`https://raw.githubusercontent.com/yinxulai/ait/v{version}/data/index.json`
- **Latest**（`dev`）：`https://raw.githubusercontent.com/yinxulai/ait/main/data/index.json`
- **Dev**：`https://raw.githubusercontent.com/yinxulai/ait/dev/data/index.json`

#### 加载流程

```
启动
  ↓
检查本地缓存
  ↓
缓存存在且兼容？
  ├─ 是 → 使用缓存 + 后台检查版本更新
  └─ 否 → 从网络下载
           ↓
       下载成功？
         ├─ 是 → 保存到缓存
         └─ 否 → 使用已有缓存（如果存在）
                   ↓
                 有缓存？
                   ├─ 是 → 带警告继续
                   └─ 否 → 返回错误
```

**后台更新机制**：
- 对比本地和远程的版本号 + 更新时间
- 有新版本才下载更新
- 无新版本继续使用本地缓存
- 更新失败不影响程序运行

---

### 本地规则

#### 使用本地文件

直接指定相对或绝对路径：

```yaml
tasks:
  - name: local-test
    protocol: openai-completions
    suite: ./rules/custom.json
    # 或绝对路径
    # suite: /path/to/rules/custom.json
```

#### 推荐的项目结构

```
project/
├── rules/                  # 规则目录
│   ├── base.json          # 基础规则
│   ├── performance.json   # 性能规则
│   └── custom.json        # 自定义规则
├── tasks/                  # 任务配置
│   └── test.yaml
└── README.md
```

---

## 规则格式

所有规则文件使用统一的 JSON 格式：

```json
{
  "version": "ait.integrity.rules/v1",
  "suite": "test-suite-id",
  "assertions": [
    {
      "id": "unique-assertion-id",
      "case_id": "*",
      "level": "critical",
      "path": "$.response.status",
      "op": "eq",
      "value": 200,
      "message": "状态码应为 200"
    }
  ]
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| `version` | string | ✅ | 规则版本，固定为 `ait.integrity.rules/v1` |
| `suite` | string | ✅ | 规则套件 ID，与任务配置匹配 |
| `assertions` | array | ✅ | 断言列表 |

### 断言字段

| 字段 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| `id` | string | ✅ | 断言唯一标识 |
| `case_id` | string | ✅ | 测试用例 ID，`*` 匹配所有 |
| `level` | string | ✅ | `critical`, `error`, `warning`, `info` |
| `path` | string | ✅ | JSONPath 表达式 |
| `op` | string | ✅ | 操作符（见下表） |
| `value` | any | ❌ | 比较值（部分操作符不需要） |
| `message` | string | ✅ | 错误消息 |

### 支持的操作符

| 操作符 | 说明 | 示例 |
|-------|------|------|
| `eq` | 等于 | `"value": 200` |
| `ne` | 不等于 | `"value": 404` |
| `gt` | 大于 | `"value": 0` |
| `gte` | 大于等于 | `"value": 100` |
| `lt` | 小于 | `"value": 1000` |
| `lte` | 小于等于 | `"value": 5000` |
| `contains` | 包含 | `"value": "error"` |
| `not_contains` | 不包含 | `"value": "success"` |
| `regex` | 正则匹配 | `"value": "^[0-9]+$"` |
| `exists` | 字段存在 | 无需 value |
| `not_exists` | 字段不存在 | 无需 value |

### 断言级别

- **critical**：关键错误，测试立即失败
- **error**：错误，记录但继续
- **warning**：警告，不影响测试结果
- **info**：信息，仅记录

---

## 使用指南

### 场景 1：完全使用内置规则

适合快速开始和标准 API 测试：

```yaml
tasks:
  - name: openai-basic
    protocol: openai-completions
    suite: openai-completions-smoke
    cases:
      - id: test-1
        model: gpt-4
        messages:
          - role: user
            content: "Hello"
```

### 场景 2：内置规则 + 自定义断言

在内置规则基础上添加项目特定检查：

```yaml
tasks:
  - name: openai-extended
    protocol: openai-completions
    suite: openai-completions-smoke
    cases:
      - id: test-1
        model: gpt-4
        messages:
          - role: user
            content: "Hello"
    assertions:
      # 自定义断言会与内置规则合并
      - id: check-model
        case_id: "*"
        level: error
        path: "$.response.model"
        op: "eq"
        value: "gpt-4"
        message: "必须使用 GPT-4 模型"
      
      - id: check-tokens
        case_id: "*"
        level: warning
        path: "$.usage.total_tokens"
        op: "lt"
        value: 1000
        message: "token 使用应小于 1000"
```

### 场景 3：多协议测试

测试多个 API 协议：

```yaml
tasks:
  - name: openai-completions-test
    protocol: openai-completions
    suite: openai-completions-smoke
    cases:
      - id: test-1
        model: gpt-4

  - name: openai-responses-test
    protocol: openai-responses
    suite: openai-responses-smoke
    cases:
      - id: test-2
        model: gpt-4

  - name: anthropic-test
    protocol: anthropic-messages
    suite: anthropic-messages-smoke
    cases:
      - id: test-3
        model: claude-3-opus-20240229
```

---

## 版本管理

### 规则索引格式

`index.json` 包含规则集的元数据和版本信息：

```json
{
  "version": "1.0.0",
  "repository": "https://github.com/yinxulai/ait",
  "description": "AIT 内置规则集",
  "compatibility": {
    "min_version": "0.1.0",
    "max_version": "99.99.99"
  },
  "rules": {
    "openai-completions": {
      "suite": "openai-completions-smoke",
      "protocol": "openai-completions",
      "file": "integrity/openai-completions.json",
      "description": "OpenAI Completions API 基础规则"
    }
  },
  "update_sources": {
    "stable": "https://raw.githubusercontent.com/yinxulai/ait/v{version}/data/index.json",
    "latest": "https://raw.githubusercontent.com/yinxulai/ait/main/data/index.json",
    "dev": "https://raw.githubusercontent.com/yinxulai/ait/dev/data/index.json"
  },
  "last_updated": "2026-06-11T00:00:00Z"
}
```

### 版本匹配逻辑

1. **开发版本**（`dev` 或空）：
   - 使用 `latest` 或 `dev` 更新源
   - 总是兼容所有规则版本

2. **正式版本**（`v1.0.0`）：
   - 使用 `stable` 更新源
   - 必须匹配 `compatibility` 范围
   - 不兼容时返回错误

---

## 缓存机制

### 内置规则缓存

- **位置**：`~/.ait/rules/`
- **更新策略**：
  - 首次启动：必须从网络下载
  - 启动时：使用本地缓存，后台检查版本
  - 版本对比：比较版本号和更新时间
  - 有新版本：后台自动下载更新
  - 无新版本：继续使用本地缓存
  - 网络失败：使用已有缓存（带提示）

### 缓存管理

**查看缓存**
```bash
# 内置规则缓存
ls -lah ~/.ait/rules/
```

**清除缓存**
```bash
# 清除所有缓存
rm -rf ~/.ait/rules/
```

---

## 故障排查

### 首次启动失败

**错误信息**
```
failed to load rules: no cache available and network update failed
```

**解决方法**
1. 检查网络连接
2. 测试 GitHub 访问：
   ```bash
   curl -I https://raw.githubusercontent.com/yinxulai/ait/main/data/index.json
   ```
3. 检查代理设置
4. 检查防火墙规则

### 规则更新失败

**错误信息**
```
warning: failed to update builtin rules: <error>
warning: using expired cache due to network failure
```

**解决方法**
- 程序会继续使用缓存运行
- 检查网络连接后重新启动

### 版本不兼容

**错误信息**
```
downloaded rules version incompatible with current version X.Y.Z
```

**解决方法**
- 升级 AIT 到最新版本
- 或使用兼容的规则版本

### 规则文件格式错误

**错误信息**
```
failed to parse rule file: invalid character...
```

**解决方法**
1. 验证 JSON 格式：`jq empty rules.json`
2. 检查规则文件结构
3. 确认 `version` 字段正确

---

## 最佳实践

### 1. 规则组织

**推荐结构**
```
project/
├── rules/
│   ├── base/               # 基础规则
│   │   ├── http.json      # HTTP 基础检查
│   │   └── latency.json   # 延迟检查
│   ├── protocol/           # 协议特定规则
│   │   ├── openai.json
│   │   └── anthropic.json
│   └── custom/             # 自定义规则
│       └── business.json   # 业务逻辑检查
└── tasks/
    └── test.yaml
```

### 2. 规则复用

**使用规则合并**
```yaml
tasks:
  - name: comprehensive-test
    protocol: openai-completions
    suite: openai-completions-smoke  # 内置规则
    assertions:
      # 添加业务特定检查
      - id: check-response-format
        case_id: "*"
        level: error
        path: "$.choices[0].message.content"
        op: "regex"
        value: "^\\{.*\\}$"
        message: "响应必须是 JSON 格式"
```

### 3. 文档维护

**为规则提供文档**
```
rules/
├── openai-completions.json
├── openai-completions.md      # 规则说明
└── README.md                   # 总览
```

### 4. CI/CD 集成

**在 CI 中使用锁定版本**
```yaml
# .github/workflows/api-tests.yml
name: API Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Install AIT
        run: curl -sSL https://...install-ait.sh | bash
          
      - name: Run Tests
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: ait run ci-tests.yaml
```

```yaml
# ci-tests.yaml
tasks:
  - name: ci-test
    protocol: openai-completions
    # 使用内置规则（自动匹配版本）
    suite: openai-completions-smoke
```

### 5. 网络环境

**确保网络访问**
- 服务器/容器能访问 GitHub（加载内置规则需要）
- 配置代理（如需要）
- 首次部署时预先下载规则

### 6. 错误处理

**准备本地备份**
```yaml
tasks:
  - name: production-test
    protocol: openai-completions
    # 使用本地规则文件作为备份
    suite: ./rules/backup.json
```

---

## API 参考

### LoadSuite

简单加载（不使用内置规则管理器）：

```go
import "github.com/yinxulai/ait/internal/server/modes/integrity"

suite, err := integrity.LoadSuite("./rules/custom.json")
if err != nil {
    log.Fatal(err)
}
```

### LoadSuiteWithManager

使用内置规则管理器加载（推荐）：

```go
import (
    "github.com/yinxulai/ait/internal/server/modes/integrity"
)

// 创建规则管理器
manager, err := integrity.NewRulesManager("v1.0.0")
if err != nil {
    log.Fatal(err)
}

// 初始化
ctx := context.Background()
if err := manager.Initialize(ctx); err != nil {
    log.Fatal(err)
}

// 加载规则
suite, err := integrity.LoadSuiteWithManager(input, manager)
if err != nil {
    log.Fatal(err)
}
```

---

## 附录

### 完整示例

参见 [examples/rules-examples.md](../examples/rules-examples.md)

### 规则模板

```json
{
  "version": "ait.integrity.rules/v1",
  "suite": "my-custom-suite",
  "assertions": [
    {
      "id": "check-status",
      "case_id": "*",
      "level": "critical",
      "path": "$.status_code",
      "op": "eq",
      "value": 200,
      "message": "HTTP 状态码应为 200"
    },
    {
      "id": "check-response-time",
      "case_id": "*",
      "level": "warning",
      "path": "$.response_time_ms",
      "op": "lt",
      "value": 5000,
      "message": "响应时间应小于 5 秒"
    },
    {
      "id": "check-content",
      "case_id": "*",
      "level": "error",
      "path": "$.response.content",
      "op": "exists",
      "message": "响应内容必须存在"
    }
  ]
}
```

### 相关文档

- [完整性测试](integrity.md)
- [任务配置](../docs/task-configuration.md)
- [协议支持](../docs/protocols.md)

---

*最后更新：2026-06-11*
