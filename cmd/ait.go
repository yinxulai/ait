package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yinxulai/ait/internal/benchmark"
)

func main() {
	baseUrl := flag.String("baseUrl", "", "服务地址")
	apikey := flag.String("apikey", "", "API 密钥")
	model := flag.String("model", "", "模型名称")
	count := flag.Int("count", 10, "请求总数")
	provider := flag.String("provider", "openai", "协议类型: openai 或 anthropic")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", false, "是否开启流模式")
	concurrency := flag.Int("concurrency", 1, "并发数")
	flag.Parse()

	// 如果未指定参数，尝试从环境变量加载
	finalBaseUrl := *baseUrl
	finalApiKey := *apikey

	if finalBaseUrl == "" || finalApiKey == "" {
		if *provider == "openai" {
			if finalBaseUrl == "" {
				if envBaseUrl := os.Getenv("OPENAI_BASE_URL"); envBaseUrl != "" {
					finalBaseUrl = envBaseUrl
				}
			}
			if finalApiKey == "" {
				if envApiKey := os.Getenv("OPENAI_API_KEY"); envApiKey != "" {
					finalApiKey = envApiKey
				}
			}
		} else if *provider == "anthropic" {
			if finalBaseUrl == "" {
				if envBaseUrl := os.Getenv("ANTHROPIC_BASE_URL"); envBaseUrl != "" {
					finalBaseUrl = envBaseUrl
				}
			}
			if finalApiKey == "" {
				if envApiKey := os.Getenv("ANTHROPIC_API_KEY"); envApiKey != "" {
					finalApiKey = envApiKey
				}
			}
		}
	}

	if finalBaseUrl == "" || finalApiKey == "" || *model == "" {
		fmt.Println("baseUrl、apikey、model 参数必填")
		fmt.Printf("对于 %s 协议，你也可以设置以下环境变量：\n", *provider)
		if *provider == "openai" {
			fmt.Println("  OPENAI_BASE_URL - OpenAI API 基础 URL")
			fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥")
		} else if *provider == "anthropic" {
			fmt.Println("  ANTHROPIC_BASE_URL - Anthropic API 基础 URL")
			fmt.Println("  ANTHROPIC_API_KEY - Anthropic API 密钥")
		}
		os.Exit(1)
	}

	config := benchmark.Config{
		Provider:    *provider,
		BaseUrl:     finalBaseUrl,
		ApiKey:      finalApiKey,
		Model:       *model,
		Concurrency: *concurrency,
		Count:       *count,
		Prompt:      *prompt,
		Stream:      *stream,
	}

	runner, err := benchmark.NewRunner(config)
	if err != nil {
		fmt.Printf("创建测试执行器失败: %v\n", err)
		os.Exit(1)
	}

	result, err := runner.Run()
	if err != nil {
		fmt.Printf("执行测试失败: %v\n", err)
		os.Exit(1)
	}

	result.PrintResult()
}
