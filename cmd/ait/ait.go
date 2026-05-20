package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/tui"
	"github.com/yinxulai/ait/internal/types"
)

// 版本信息，通过 ldflags 在构建时注入。
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// ── flags ────────────────────────────────────────────────────────────────
	versionFlag := flag.Bool("version", false, "显示版本信息")
	baseURL     := flag.String("baseUrl", "", "服务基础地址（可选，留空使用协议默认地址）")
	apiKey      := flag.String("apiKey", "", "API 密钥")
	model       := flag.String("model", "", "模型名称")
	protocol    := flag.String("protocol", "", "协议类型: openai / anthropic")
	promptText  := flag.String("prompt", "", "Prompt 文本（可选）")
	promptFile  := flag.String("prompt-file", "", "从文件读取 Prompt")
	promptLen   := flag.Int("prompt-length", 0, "生成指定长度的测试 Prompt（字符数）")
	stream      := flag.Bool("stream", true, "是否开启流式输出")
	thinking    := flag.Bool("thinking", false, "是否开启 Thinking 模式")
	concurrency := flag.Int("concurrency", 10, "并发数")
	count       := flag.Int("count", 100, "请求总数")
	timeout     := flag.Int("timeout", 300, "请求超时时间（秒）")
	turboFlag   := flag.Bool("turbo", false, "是否启用 Turbo 并发探测模式")
	flag.Parse()

	// ── 版本输出 ──────────────────────────────────────────────────────────────
	if *versionFlag {
		fmt.Printf("ait version %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// ── 创建 Server ───────────────────────────────────────────────────────────
	srv, err := server.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化 Server 失败: %v\n", err)
		os.Exit(1)
	}

	// ── 若提供了足够参数则预建任务并自动启动 ────────────────────────────────────
	if *model != "" {
		finalProtocol, finalBaseURL, finalAPIKey := resolveConfig(*protocol, *baseURL, *apiKey)
		if finalAPIKey == "" {
			fmt.Fprintln(os.Stderr, "错误: 缺少 API Key（-apiKey 或环境变量）")
			os.Exit(1)
		}

		inp := types.Input{
			Protocol:     finalProtocol,
			BaseUrl:      finalBaseURL,
			ApiKey:       finalAPIKey,
			Model:        *model,
			Stream:       *stream,
			Thinking:     *thinking,
			Concurrency:  *concurrency,
			Count:        *count,
			Turbo:        *turboFlag,
			Timeout:      time.Duration(*timeout) * time.Second,
		}

		// Prompt 配置
		switch {
		case *promptLen > 0:
			inp.PromptMode = "generated"
			inp.PromptLength = *promptLen
		case *promptFile != "":
			inp.PromptMode = "file"
			inp.PromptFile = *promptFile
		case *promptText != "":
			inp.PromptMode = "text"
			inp.PromptText = *promptText
		default:
			inp.PromptMode = "text"
			inp.PromptText = "你好，介绍一下你自己。"
		}

		taskName := fmt.Sprintf("%s@%s", *model, strings.TrimRight(finalBaseURL, "/"))
		_, err := srv.CreateTask(server.TaskConfig{Name: taskName, Input: inp})
		if err != nil {
			fmt.Fprintf(os.Stderr, "创建任务失败: %v\n", err)
			os.Exit(1)
		}
	}

	// ── 启动 TUI ──────────────────────────────────────────────────────────────
	tui.SetVersion(Version)
	if err := tui.Run(srv); err != nil {
		fmt.Fprintf(os.Stderr, "TUI 启动失败: %v\n", err)
		os.Exit(1)
	}
}

// resolveConfig 合并命令行参数与环境变量。
func resolveConfig(protocol, baseURL, apiKey string) (string, string, string) {
	if protocol == "" {
		if os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("OPENAI_BASE_URL") != "" {
			protocol = "openai"
		} else if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_BASE_URL") != "" {
			protocol = "anthropic"
		} else {
			protocol = "openai"
		}
	}
	if baseURL == "" {
		switch protocol {
		case "anthropic":
			baseURL = os.Getenv("ANTHROPIC_BASE_URL")
		default:
			baseURL = os.Getenv("OPENAI_BASE_URL")
		}
	}
	if apiKey == "" {
		switch protocol {
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		default:
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
	}
	return protocol, baseURL, apiKey
}
