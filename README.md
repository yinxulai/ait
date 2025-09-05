# AIT - AI 模型性能测试工具

一个强大的 CLI 工具，用于批量测试符合 OpenAI 协议的模型的性能指标，支持 TTFT（首字节时间）和 TPS（吞吐量）等关键性能指标的测量。

## ✨ 功能特性

- 🚀 **多协议支持**: 支持 OpenAI 和 Anthropic 协议
- 🎯 **多模型测试**: 支持同时测试多个模型，用逗号分割模型名称
- 📊 **实时进度条**: 测试过程可视化显示
- 🎨 **彩色输出**: 美观的终端界面
- 📋 **表格化结果**: 清晰的结果展示
- ⚡ **并发测试**: 支持自定义并发数压力测试
- 📈 **详细统计**: TTFT、TPS、最小/最大/平均响应时间
- 📄 **多格式报告**: 支持生成 JSON 和 CSV 格式的详细测试报告
- 🌐 **网络指标**: 包含 DNS、连接、TLS 握手等网络性能指标

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
  --protocol=openai 
  --baseUrl=https://api.openai.com 
  --apikey=sk-your-api-key 
  --model=gpt-3.5-turbo 
  --concurrency=3 
  --count=10
  --report
```

### Anthropic 协议测试

```bash
./bin/ait 
  --protocol=anthropic 
  --baseUrl=https://api.anthropic.com 
  --apikey=sk-ant-your-api-key 
  --model=claude-3-haiku-20240307 
  --concurrency=2 
  --count=5
  --report
```

### 多模型比较测试

```bash
# 同时测试多个 OpenAI 模型
./bin/ait 
  --protocol=openai 
  --baseUrl=https://api.openai.com/v1 
  --apikey=sk-your-api-key 
  --model="gpt-3.5-turbo,gpt-4,gpt-4-turbo" 
  --concurrency=3 
  --count=10
  --report

# 多模型测试会为每个模型生成独立的 JSON 和 CSV 报告
# 同时还会生成一个综合比较的 CSV 报告方便对比分析
```

### 本地模型测试（如 Ollama）

```bash
./bin/ait 
  --protocol=openai 
  --baseUrl=http://localhost:11434 
  --apikey=dummy 
  --model=llama2 
  --concurrency=1 
  --count=3
```

## 🔧 环境变量支持

为了简化使用，AIT 支持通过环境变量自动配置 API 密钥和服务地址：

### OpenAI 协议

```bash
export OPENAI_API_KEY="sk-your-api-key"
export OPENAI_BASE_URL="https://api.openai.com/v1"

# 简化使用，protocol 会自动推断为 openai
./bin/ait --model=gpt-3.5-turbo --count=10 --report
```

### Anthropic 协议

```bash
export ANTHROPIC_API_KEY="sk-ant-your-api-key"
export ANTHROPIC_BASE_URL="https://api.anthropic.com"

# 简化使用，protocol 会自动推断为 anthropic
./bin/ait --model=claude-3-haiku-20240307 --count=5 --report
```

## 📋 命令行参数

| 参数            | 描述                                                          | 默认值                    | 必填 |
|:---------------|:-------------------------------------------------------------|:--------------------------|:----:|
| `--protocol`   | 协议类型 (`openai`/`anthropic`)                               | `openai`                  |  ❌  |
| `--baseUrl`    | 服务地址<br/>支持环境变量：`OPENAI_BASE_URL` 或 `ANTHROPIC_BASE_URL` | -                         |  ✅  |
| `--apikey`     | API 密钥<br/>支持环境变量：`OPENAI_API_KEY` 或 `ANTHROPIC_API_KEY`  | -                         |  ✅  |
| `--model`      | 模型名称，支持多个模型用逗号分割<br/>如：`gpt-4,claude-3-sonnet`     | -                         |  ✅  |
| `--concurrency`| 并发数                                                        | `1`                       |  ❌  |
| `--count`      | 请求总数                                                       | `10`                      |  ❌  |
| `--prompt`     | 测试提示语                                                     | `"你好，介绍一下你自己。"`     |  ❌  |
| `--report`     | 是否生成报告文件（同时生成 JSON 和 CSV）                           | `false`                   |  ❌  |

## 📊 输出指标说明

### 终端输出指标

- **TTFT (Time To First Token)**: 首字节时间，衡量模型开始响应的速度
- **TPS (Tokens Per Second)**: 每秒处理的请求数，衡量系统吞吐量
- **平均/最小/最大响应时间**: 请求的响应时间统计
- **网络性能指标**: DNS 解析、TCP 连接、TLS 握手时间

### 报告文件生成

当使用 `--report` 参数时，将在当前目录生成多种格式的报告文件：

#### 单模型测试

- **JSON 报告**: `ait-report-{模型名}-{时间戳}.json` - 详细的结构化数据
- **CSV 报告**: `ait-report-{模型名}-{时间戳}.csv` - 表格格式，便于导入 Excel 分析

#### 多模型测试

- **每个模型的独立报告**: JSON 和 CSV 格式各一份
- **综合比较报告**: `ait-comparison-{时间戳}.csv` - 包含所有模型的比较数据

#### 报告内容包含

- **metadata**: 测试元数据（时间戳、配置信息等）
- **time_metrics**: 时间性能指标（平均、最小、最大响应时间）
- **network_metrics**: 网络性能指标（DNS、连接、TLS 时间，目标 IP）
- **content_metrics**: 服务性能指标（TTFT、Token 统计、TPS 等）
- **reliability_metrics**: 可靠性指标（成功率、错误率）

## 🎯 使用场景

- **模型性能基准测试**: 评估不同模型的响应速度和质量
- **多模型比较测试**: 同时测试多个模型并生成比较报告
- **服务压力测试**: 测试服务在不同并发下的表现
- **API 接口验证**: 验证 OpenAI 兼容接口的正确性
- **性能监控**: 定期监控模型服务的性能表现
- **容量规划**: 为生产环境部署提供性能数据支持
- **自动化测试**: 结合 CI/CD 流程进行自动化性能测试
- **性能报告**: 生成详细的 JSON 和 CSV 报告用于数据分析和存档

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
