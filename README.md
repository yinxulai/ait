# AIT - AI 模型性能测试工具

一个强大的 CLI 工具，用于批量测试符合 OpenAI 协议的模型的性能指标，支持 TTFT（首字节时间）和 TPS（吞吐量）等关键性能指标的测量。

## ✨ 功能特性

- 🚀 **多协议支持**: 支持 OpenAI 和 Anthropic 协议
- 📊 **实时进度条**: 测试过程可视化显示
- 🎨 **彩色输出**: 美观的终端界面
- 📋 **表格化结果**: 清晰的结果展示
- 🎯 **自动评级**: 基于响应时间的性能评级
- ⚡ **并发测试**: 支持自定义并发数压力测试
- 📈 **详细统计**: TTFT、TPS、最小/最大/平均响应时间

## 🛠️ 安装和使用

### 方式一：下载预编译二进制文件（推荐）

从 [Releases 页面](https://github.com/yinxulai/ait/releases) 下载适合您平台的预编译二进制文件：

```bash
# Linux (x64)
wget https://github.com/yinxulai/ait/releases/latest/download/ait-linux-amd64
chmod +x ait-linux-amd64
sudo mv ait-linux-amd64 /usr/local/bin/ait

# Linux (ARM64)
wget https://github.com/yinxulai/ait/releases/latest/download/ait-linux-arm64
chmod +x ait-linux-arm64
sudo mv ait-linux-arm64 /usr/local/bin/ait

# macOS (Intel)
wget https://github.com/yinxulai/ait/releases/latest/download/ait-darwin-amd64
chmod +x ait-darwin-amd64
sudo mv ait-darwin-amd64 /usr/local/bin/ait

# macOS (Apple Silicon)
wget https://github.com/yinxulai/ait/releases/latest/download/ait-darwin-arm64
chmod +x ait-darwin-arm64
sudo mv ait-darwin-arm64 /usr/local/bin/ait

# Windows (x64) - PowerShell
Invoke-WebRequest -Uri "https://github.com/yinxulai/ait/releases/latest/download/ait-windows-amd64.exe" -OutFile "ait.exe"
# 将 ait.exe 移动到您的 PATH 中

# Windows (ARM64) - PowerShell
Invoke-WebRequest -Uri "https://github.com/yinxulai/ait/releases/latest/download/ait-windows-arm64.exe" -OutFile "ait.exe"
# 将 ait.exe 移动到您的 PATH 中
```

### 方式二：从源码编译

```bash
# 克隆项目
git clone https://github.com/yinxulai/ait.git
cd ait

# 编译
make build

# 或者直接用 go build
go build -o bin/ait ./cmd/
```

## 🚀 快速开始

### OpenAI 协议测试

```bash
./bin/ait 
  --provider=openai 
  --baseUrl=https://api.openai.com 
  --apikey=sk-your-api-key 
  --model=gpt-3.5-turbo 
  --concurrency=3 
  --count=10
```

### Anthropic 协议测试

```bash
./bin/ait 
  --provider=anthropic 
  --baseUrl=https://api.anthropic.com 
  --apikey=sk-ant-your-api-key 
  --model=claude-3-haiku-20240307 
  --concurrency=2 
  --count=5
```

### 本地模型测试（如 Ollama）

```bash
./bin/ait 
  --provider=openai 
  --baseUrl=http://localhost:11434 
  --apikey=dummy 
  --model=llama2 
  --concurrency=1 
  --count=3
```

## 📋 命令行参数

| 参数 | 描述 | 默认值 | 必填 |
|------|------|--------|------|
| `--provider` | 协议类型 (openai/anthropic) | openai | ❌ |
| `--baseUrl` | 服务地址 | - | ✅ |
| `--apikey` | API 密钥 | - | ✅ |
| `--model` | 模型名称 | - | ✅ |
| `--concurrency` | 并发数 | 1 | ❌ |
| `--count` | 请求总数 | 10 | ❌ |
| `--prompt` | 测试提示语 | "你好，介绍一下你自己。" | ❌ |

## 📊 输出指标说明

- **TTFT (Time To First Token)**: 首字节时间，衡量模型开始响应的速度
- **TPS (Tokens Per Second)**: 每秒处理的请求数，衡量系统吞吐量
- **平均/最小/最大响应时间**: 请求的响应时间统计
- **性能评级**: 基于平均响应时间的自动评级
  - 优秀: < 100ms
  - 良好: 100-500ms  
  - 一般: 500ms-1s
  - 较慢: 1-3s
  - 很慢: > 3s

## 🎯 使用场景

- **模型性能基准测试**: 评估不同模型的响应速度
- **服务压力测试**: 测试服务在不同并发下的表现
- **API 接口验证**: 验证 OpenAI 兼容接口的正确性
- **性能监控**: 定期监控模型服务的性能表现
- **容量规划**: 为生产环境部署提供性能数据支持

## 🔧 开发和贡献

### 可用命令

```bash
make build          # 编译二进制文件
make test           # 运行测试
make clean          # 清理构建文件
make help           # 查看所有命令
```

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
