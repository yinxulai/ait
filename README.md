# AIT - AI 模型性能测试工具

[![test](https://github.com/yinxulai/ait/actions/workflows/test.yaml/badge.svg)](https://github.com/yinxulai/ait/actions/workflows/test.yaml)
[![codecov](https://codecov.io/gh/yinxulai/ait/graph/badge.svg?token=WO1ZIWNGJ8)](https://codecov.io/gh/yinxulai/ait)

一个强大的 CLI 工具，用于批量测试符合 OpenAI 协议和 Anthropic 协议的 AI 模型性能指标。默认启动交互式终端任务中心（TUI），也可通过 MCP 协议供 AI 客户端调用。

## ✨ 功能特性

- 🚀 **多协议支持**: 支持 OpenAI 和 Anthropic 协议
- 🖥️ **交互式 TUI**: 可视化创建、运行、管理测试任务
- 📊 **实时仪表盘**: 运行过程实时显示进度和指标
- 📄 **多格式报告**: 支持生成 JSON 和 CSV 格式的详细测试报告
- 🌐 **网络指标**: 包含 DNS、连接、TLS 握手等网络性能指标
- 🔄 **流式支持**: 默认支持流式响应，更真实的测试场景

## 🛠️ 安装

### Linux/macOS

```bash
curl -fsSL https://raw.githubusercontent.com/yinxulai/ait/main/scripts/install-ait.sh | bash
```

### Windows

```powershell
Invoke-WebRequest -Uri "https://github.com/yinxulai/ait/releases/latest/download/ait-windows-amd64.exe" -OutFile "ait.exe"
```

### 从源码编译

```bash
git clone https://github.com/yinxulai/ait.git
cd ait
make build
```

## 🚀 快速开始

### 启动交互式任务中心（TUI）

```bash
ait
```

在 TUI 中可完成：

- 创建和编辑测试任务
- 启动运行并实时查看仪表盘
- 查看历史记录和导出报告

### MCP 模式（供 AI 客户端调用）

```bash
ait --mcp
```

当前内置工具：

| 工具 | 描述 |
| ------ | ------ |
| `ait.list_tasks` | 列出所有任务 |
| `ait.create_task` | 创建新任务 |
| `ait.run_task` | 运行指定任务 |
| `ait.get_task_state` | 查询任务/运行状态 |

## 🔧 环境变量

### OpenAI 协议

```bash
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"
```

### Anthropic 协议

```bash
export ANTHROPIC_API_KEY="sk-ant-your-api-key"
export ANTHROPIC_BASE_URL="https://api.anthropic.com"
```

## ⚙️ MCP 客户端配置

AIT 可作为本地 MCP 服务器接入各种 AI 客户端。以下是常见的配置方式：

```json
{
  "mcpServers": {
    "ait": {
      "command": "ait",
      "args": ["--mcp"]
    }
  }
}
```

### 其他客户端

通用 stdio 方式：

```bash
ait --mcp
```

确保 `ait` 在您的 PATH 中。

## 📋 命令行参数

| 参数        | 描述                |
| ----------- | ------------------- |
| `--version` | 显示版本信息        |
| `--mcp`     | 以 MCP 服务模式启动 |

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
