# Modes 目录说明

此目录包含 AIT 的所有运行模式实现，各模式彼此独立。

## 目录结构

```
modes/
├── standard/      # 标准性能测试模式
├── turbo/         # 并发爬坡测试模式
└── integrity/     # 接口完整性测试模式
```

## 模式说明

### standard - 标准性能测试
- **入口**: `runner.go` - `NewRunner()`
- **职责**: 以固定并发度执行 N 个请求，收集性能指标
- **依赖**: client、logger、types、upload
- **使用场景**: 基准测试、性能对比、压力测试

### turbo - 并发爬坡测试
- **入口**: `engine.go` - `New()`
- **职责**: 自动探测最大稳定并发度，逐级递增并发直到失败
- **依赖**: standard（复用 Runner）、types
- **使用场景**: 寻找最佳并发配置、容量规划

### integrity - 接口完整性测试
- **入口**: `executor.go` - `NewExecutor()`
- **职责**: 执行声明式断言，验证接口响应结构和行为
- **依赖**: standard（复用 Runner）、client、task、types
- **子模块**:
  - `suite.go` - 内置套件与规则文件加载
  - `observation.go` - 将响应转为断言观测模型
  - `assertion/` - 断言评估引擎
- **使用场景**: 冒烟测试、回归测试、兼容性验证

## 添加新模式

1. 在 `modes/` 下创建新目录，如 `modes/benchmark/`
2. 实现模式核心逻辑（参考现有模式）
3. 在 `internal/server/types/types.go` 中定义相关领域类型
4. 在 `internal/server/run_service.go` 中添加模式分支
5. 更新 `Input.RunMode()` 支持新模式标识

## 设计原则

- **独立性**: 各模式包互不依赖，只依赖基础设施层和类型层
- **复用性**: 优先复用 `standard.Runner` 和基础设施层组件
- **扩展性**: 新增模式无需修改现有模式代码
- **清晰性**: 每个模式职责单一，目录名即模式名
