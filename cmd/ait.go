package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yinxulai/ait/internal/display"
	"github.com/yinxulai/ait/internal/runner"
)

func main() {
	baseUrl := flag.String("baseUrl", "", "服务地址")
	apikey := flag.String("apikey", "", "API 密钥")
	model := flag.String("model", "", "模型名称")
	count := flag.Int("count", 10, "请求总数")
	provider := flag.String("provider", "openai", "协议类型: openai 或 anthropic")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", true, "是否开启流模式")
	concurrency := flag.Int("concurrency", 1, "并发数")
	flag.Parse()

	// 如果未指定参数，尝试从环境变量加载
	finalBaseUrl := *baseUrl
	finalApiKey := *apikey

	if finalBaseUrl == "" || finalApiKey == "" {
		switch *provider {
		case "openai":
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
		case "anthropic":
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

	// model 参数检查（只能通过命令行参数指定）
	if *model == "" {
		fmt.Println("model 参数必填，请通过 -model 参数指定")
		os.Exit(1)
	}

	// baseUrl 和 apikey 检查（可以通过环境变量获取）
	if finalBaseUrl == "" || finalApiKey == "" {
		fmt.Println("baseUrl 和 apikey 参数必填")
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

	config := runner.Config{
		Provider:    *provider,
		BaseUrl:     finalBaseUrl,
		ApiKey:      finalApiKey,
		Model:       *model,
		Concurrency: *concurrency,
		Count:       *count,
		Prompt:      *prompt,
		Stream:      *stream,
	}

	runnerInstance, err := runner.NewRunner(config)
	if err != nil {
		fmt.Printf("创建测试执行器失败: %v\n", err)
		os.Exit(1)
	}

	// 创建显示控制器
	displayConfig := display.TestConfig{
		Provider:    config.Provider,
		BaseUrl:     config.BaseUrl,
		ApiKey:      config.ApiKey,
		Model:       config.Model,
		Concurrency: config.Concurrency,
		Count:       config.Count,
		Stream:      config.Stream,
	}
	testDisplayer := display.NewTestDisplayer(displayConfig)

	// 显示测试开始信息
	testDisplayer.ShowTestStart()

	// 用于保存最后的统计信息
	var finalStats display.TestStats

	// 执行测试，使用回调函数来更新显示
	result, err := runnerInstance.RunWithProgress(func(stats runner.TestStats) {
		displayStats := display.TestStats{
			CompletedCount:      stats.CompletedCount,
			FailedCount:         stats.FailedCount,
			TTFTs:               stats.TTFTs,
			TotalTimes:          stats.TotalTimes,
			TokenCounts:         stats.TokenCounts,
			StartTime:           stats.StartTime,
			ElapsedTime:         stats.ElapsedTime,
			// 网络性能指标
			DNSTimes:            stats.DNSTimes,
			ConnectTimes:        stats.ConnectTimes,
			TLSHandshakeTimes:   stats.TLSHandshakeTimes,
			// 可靠性指标
			TimeoutCount:        stats.TimeoutCount,
			RetryCount:          stats.RetryCount,
		}
		finalStats = displayStats // 保存最后的统计信息
		testDisplayer.UpdateProgress(displayStats)
	})

	if err != nil {
		testDisplayer.ShowError(fmt.Sprintf("执行测试失败: %v", err))
		os.Exit(1)
	}

	// 显示测试完成
	testDisplayer.ShowTestComplete()

	// 显示测试摘要
	testDisplayer.ShowTestSummary(finalStats)

	// 转换结果并显示
	displayResult := &display.Result{
		TotalRequests: result.TotalRequests,
		Concurrency:   result.Concurrency,
		IsStream:      result.IsStream,
		TotalTime:     result.TotalTime,
		TPS:           result.TPS,
	}

	// 时间性能指标
	displayResult.TimeMetrics.AvgTTFT = result.TimeMetrics.AvgTTFT
	displayResult.TimeMetrics.MinTTFT = result.TimeMetrics.MinTTFT
	displayResult.TimeMetrics.MaxTTFT = result.TimeMetrics.MaxTTFT
	displayResult.TimeMetrics.AvgTotalTime = result.TimeMetrics.AvgTotalTime
	displayResult.TimeMetrics.MinTotalTime = result.TimeMetrics.MinTotalTime
	displayResult.TimeMetrics.MaxTotalTime = result.TimeMetrics.MaxTotalTime

	// 网络性能指标
	displayResult.NetworkMetrics.AvgDNSTime = result.NetworkMetrics.AvgDNSTime
	displayResult.NetworkMetrics.MinDNSTime = result.NetworkMetrics.MinDNSTime
	displayResult.NetworkMetrics.MaxDNSTime = result.NetworkMetrics.MaxDNSTime
	displayResult.NetworkMetrics.AvgConnectTime = result.NetworkMetrics.AvgConnectTime
	displayResult.NetworkMetrics.MinConnectTime = result.NetworkMetrics.MinConnectTime
	displayResult.NetworkMetrics.MaxConnectTime = result.NetworkMetrics.MaxConnectTime
	displayResult.NetworkMetrics.AvgTLSHandshakeTime = result.NetworkMetrics.AvgTLSHandshakeTime
	displayResult.NetworkMetrics.MinTLSHandshakeTime = result.NetworkMetrics.MinTLSHandshakeTime
	displayResult.NetworkMetrics.MaxTLSHandshakeTime = result.NetworkMetrics.MaxTLSHandshakeTime

	// 内容指标
	displayResult.ContentMetrics.AvgTokenCount = result.ContentMetrics.AvgTokenCount
	displayResult.ContentMetrics.MinTokenCount = result.ContentMetrics.MinTokenCount
	displayResult.ContentMetrics.MaxTokenCount = result.ContentMetrics.MaxTokenCount
	displayResult.ContentMetrics.TotalTokens = result.ContentMetrics.TotalTokens

	// 可靠性指标
	displayResult.ReliabilityMetrics.ErrorRate = result.ReliabilityMetrics.ErrorRate
	displayResult.ReliabilityMetrics.TimeoutCount = result.ReliabilityMetrics.TimeoutCount
	displayResult.ReliabilityMetrics.RetryCount = result.ReliabilityMetrics.RetryCount
	displayResult.ReliabilityMetrics.SuccessRate = result.ReliabilityMetrics.SuccessRate

	displayResult.PrintResult()
}
