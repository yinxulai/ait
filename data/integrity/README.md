# Integrity Rules 规则文件格式

## 新格式（推荐）⭐

每个规则文件包含**完整的测试用例**，包括输入（request）和多个输出检查（assertions）。

### 基本结构

```json
{
  "version": "ait.integrity.rules/v1",
  "suite": "openai-completions-smoke",
  "description": "规则描述",
  "cases": [
    {
      "id": "test-case-1",
      "name": "测试用例名称",
      "description": "详细描述",
      "category": "protocol",
      "capability": "basic_request",
      "required": true,
      "request": {
        "prompt": "Say hello",
        "stream": false
      },
      "timeout_ms": 30000,
      "assertions": [
        {
          "id": "check-1",
          "level": "error",
          "path": "response.status_code",
          "op": "eq",
          "value": 200,
          "message": "状态码必须是 200"
        },
        {
          "id": "check-2",
          "level": "warn",
          "path": "response.body.id",
          "op": "exists",
          "message": "响应应包含 id"
        }
      ]
    }
  ]
}
```

### 核心改进

#### ✅ 1. **输入和输出分离明确**

```json
{
  "request": {          // 输入：发送什么请求
    "prompt": "...",
    "stream": false
  },
  "assertions": [       // 输出：检查什么响应
    { "path": "...", "op": "...", "value": ... }
  ]
}
```

#### ✅ 2. **一个输入 → 多个输出检查**

```json
{
  "id": "basic-response",
  "request": {
    "prompt": "Say hello",
    "stream": false
  },
  "assertions": [
    { "id": "status", "path": "response.status_code", ... },
    { "id": "id", "path": "response.body.id", ... },
    { "id": "choices", "path": "response.body.choices", ... },
    { "id": "content", "path": "response.body.choices[0].message.content", ... },
    { "id": "usage", "path": "response.body.usage", ... }
  ]
}
```

#### ✅ 3. **不同输入测试不同场景**

```json
{
  "cases": [
    {
      "id": "non-streaming",
      "request": { "prompt": "Say hello", "stream": false },
      "assertions": [
        { "path": "response.body.choices[0].message.content", ... }
      ]
    },
    {
      "id": "streaming",
      "request": { "prompt": "Count 1 to 10", "stream": true },
      "assertions": [
        { "path": "response.headers.content-type", "op": "contains", "value": "text/event-stream" },
        { "path": "response.body.chunks[0].choices[0].delta", ... }
      ]
    }
  ]
}
```

## 旧格式（兼容）

为了向后兼容，仍然支持只包含 assertions 的格式：

```json
{
  "version": "ait.integrity.rules/v1",
  "suite": "openai-completions-smoke",
  "assertions": [
    {
      "id": "check-1",
      "case_id": "existing-case-id",
      "level": "error",
      "path": "response.status_code",
      "op": "eq",
      "value": 200,
      "message": "..."
    }
  ]
}
```

**限制**：
- ❌ 必须引用已存在的 `case_id`
- ❌ 不能定义测试输入（request）
- ❌ 只能添加 assertions 到现有 case

## 对比示例

### 旧格式 ❌

```json
{
  "assertions": [
    {
      "id": "http_status",
      "case_id": "basic-response-shape",  // 引用代码中的 case
      "path": "response.status_code",
      "op": "between",
      "value": [200, 299]
    }
  ]
}
```

**问题**：
- 不知道输入是什么
- case 定义在代码中，不在规则文件中
- 无法定义自己的完整测试用例

### 新格式 ✅

```json
{
  "cases": [
    {
      "id": "basic-response-shape",
      "request": {                        // 明确定义输入
        "prompt": "Reply with hello",
        "stream": false
      },
      "assertions": [                     // 多个输出检查
        {
          "id": "http_status",
          "path": "response.status_code",
          "op": "between",
          "value": [200, 299]
        },
        {
          "id": "has_content",
          "path": "response.body.choices[0].message.content",
          "op": "exists"
        }
      ]
    }
  ]
}
```

**优点**：
- ✅ 输入输出完整定义
- ✅ 自包含的测试用例
- ✅ 一个输入可以有多个检查
- ✅ 可以定义多个不同的测试场景

## 合并行为

当加载多个规则文件时：

### 1. 新 case 直接添加

```json
// file1.json
{ "cases": [{ "id": "case-a", ... }] }

// file2.json
{ "cases": [{ "id": "case-b", ... }] }

// 结果：包含 case-a 和 case-b
```

### 2. 相同 ID 的 case 会合并

```json
// file1.json
{
  "cases": [{
    "id": "test-1",
    "request": { "prompt": "Hello" },
    "assertions": [{ "id": "check-1", ... }]
  }]
}

// file2.json
{
  "cases": [{
    "id": "test-1",
    "assertions": [{ "id": "check-2", ... }]
  }]
}

// 结果：test-1 包含 check-1 和 check-2
```

### 3. 相同 ID 的 assertion 会被替换

```json
// file1.json (内置规则)
{ "assertions": [{ "id": "check-1", "level": "warn", ... }] }

// file2.json (用户规则)
{ "assertions": [{ "id": "check-1", "level": "error", ... }] }

// 结果：check-1 使用 "error" 级别（用户规则优先）
```

## Assertion 字段说明

### 必填字段

- `id`: 唯一标识符
- `path`: JSON 路径（如 `response.body.choices[0].message.content`）
- `op`: 操作符（`eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `contains`, `matches`, `exists`, `between`）
- `level`: 严重级别（`error`, `warn`, `info`）

### 可选字段

- `value`: 比较值（某些操作符需要）
- `message`: 错误消息
- `case_id`: 所属 case ID（在 cases 中会自动设置）

## 最佳实践

### ✅ 推荐

```json
{
  "cases": [
    {
      "id": "descriptive-name",
      "name": "易读的中文名称",
      "description": "详细说明这个测试的目的",
      "request": {
        "prompt": "Specific test prompt",
        "stream": false
      },
      "assertions": [
        {
          "id": "short-id",
          "level": "error",
          "path": "response.body.specific_field",
          "op": "exists",
          "message": "清晰的错误消息"
        }
      ]
    }
  ]
}
```

### ❌ 避免

```json
{
  "assertions": [
    {
      "case_id": "unknown-case",  // 不清楚这个 case 在哪里定义
      "path": "response.body",     // 过于宽泛
      "op": "exists"               // 没有 message
    }
  ]
}
```

## 参考

- [openai-completions.json](./openai-completions.json) - OpenAI 协议完整示例
- [design/rules.md](../../design/rules.md) - 完整规则系统文档
