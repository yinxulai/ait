package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yinxulai/ait/internal/display"
	"github.com/yinxulai/ait/internal/report"
	"github.com/yinxulai/ait/internal/runner"
)

// detectProviderFromEnv 根据环境变量自动检测 provider
func detectProviderFromEnv() string {
	// 优先检查 OpenAI 环境变量
	if os.Getenv("OPENAI_API_KEY") != "" || os.Getenv("OPENAI_BASE_URL") != "" {
		return "openai"
	}
	// 其次检查 Anthropic 环境变量
	if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_BASE_URL") != "" {
		return "anthropic"
	}
	// 默认返回 openai
	return "openai"
}

// loadEnvForProvider 根据 provider 加载对应的环境变量
func loadEnvForProvider(provider string) (baseUrl, apiKey string) {
	switch provider {
	case "openai":
		return os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		return os.Getenv("ANTHROPIC_BASE_URL"), os.Getenv("ANTHROPIC_API_KEY")
	default:
		return "", ""
	}
}

func main() {
	baseUrl := flag.String("baseUrl", "", "服务地址")
	apikey := flag.String("apikey", "", "API 密钥")
	model := flag.String("model", "", "模型名称")
	count := flag.Int("count", 10, "请求总数")
	provider := flag.String("provider", "", "协议类型: openai 或 anthropic")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", true, "是否开启流模式")
	concurrency := flag.Int("concurrency", 3, "并发数")
	reportFlag := flag.Bool("report", false, "是否生成报告文件")
	flag.Parse()

	// 自动推断 provider 和加载环境变量
	finalProvider := *provider
	finalBaseUrl := *baseUrl
	finalApiKey := *apikey

	// 如果未指定 provider，根据环境变量自动推断
	if finalProvider == "" {
		finalProvider = detectProviderFromEnv()
	}

	// 根据 provider 加载对应的环境变量
	if finalBaseUrl == "" || finalApiKey == "" {
		envBaseUrl, envApiKey := loadEnvForProvider(finalProvider)
		if finalBaseUrl == "" {
			finalBaseUrl = envBaseUrl
		}
		if finalApiKey == "" {
			finalApiKey = envApiKey
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
		fmt.Printf("对于 %s 协议，你也可以设置以下环境变量：\n", finalProvider)

		switch finalProvider {
		case "openai":
			fmt.Println("  OPENAI_BASE_URL - OpenAI API 基础 URL")
			fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥")
		case "anthropic":
			fmt.Println("  ANTHROPIC_BASE_URL - Anthropic API 基础 URL")
			fmt.Println("  ANTHROPIC_API_KEY - Anthropic API 密钥")
		}
		os.Exit(1)
	}

	config := runner.Config{
		Provider:    finalProvider,
		BaseUrl:     finalBaseUrl,
		ApiKey:      finalApiKey,
		Model:       *model,
		Concurrency: *concurrency,
		Count:       *count,
		Prompt:      *prompt,
		Stream:      *stream,
		Report:      *reportFlag,
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
			CompletedCount: stats.CompletedCount,
			FailedCount:    stats.FailedCount,
			TTFTs:          stats.TTFTs,
			TotalTimes:     stats.TotalTimes,
			TokenCounts:    stats.TokenCounts,
			ErrorMessages:  stats.ErrorMessages,
			StartTime:      stats.StartTime,
			ElapsedTime:    stats.ElapsedTime,
			// 网络性能指标
			DNSTimes:          stats.DNSTimes,
			ConnectTimes:      stats.ConnectTimes,
			TLSHandshakeTimes: stats.TLSHandshakeTimes,
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

	// 显示错误详情（如果有错误的话）
	testDisplayer.ShowErrorDetails(finalStats)

	// 转换结果并显示
	displayResult := &display.Result{
		TotalRequests: result.TotalRequests,
		Concurrency:   result.Concurrency,
		IsStream:      result.IsStream,
		TotalTime:     result.TotalTime,
	}

	// 时间性能指标
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
	displayResult.NetworkMetrics.TargetIP = result.NetworkMetrics.TargetIP

	// 服务性能指标
	displayResult.ContentMetrics.AvgTTFT = result.ContentMetrics.AvgTTFT
	displayResult.ContentMetrics.MinTTFT = result.ContentMetrics.MinTTFT
	displayResult.ContentMetrics.MaxTTFT = result.ContentMetrics.MaxTTFT
	displayResult.ContentMetrics.AvgTokenCount = result.ContentMetrics.AvgTokenCount
	displayResult.ContentMetrics.MinTokenCount = result.ContentMetrics.MinTokenCount
	displayResult.ContentMetrics.MaxTokenCount = result.ContentMetrics.MaxTokenCount

	// 可靠性指标
	displayResult.ReliabilityMetrics.ErrorRate = result.ReliabilityMetrics.ErrorRate
	displayResult.ReliabilityMetrics.SuccessRate = result.ReliabilityMetrics.SuccessRate

	displayResult.PrintResult()

	// 如果启用了报告生成，则生成报告文件
	if config.Report {
		// 构建报告配置
		reportConfig := report.TestConfig{
			Provider:    config.Provider,
			BaseUrl:     config.BaseUrl,
			ApiKey:      config.ApiKey,
			Model:       config.Model,
			Concurrency: config.Concurrency,
			Count:       config.Count,
			Stream:      config.Stream,
			Prompt:      config.Prompt,
		}

		// 构建报告结果数据
		reportResult := report.TestResult{
			TotalRequests: result.TotalRequests,
			Concurrency:   result.Concurrency,
			IsStream:      result.IsStream,
			TotalTime:     result.TotalTime,
		}

		// 复制时间性能指标
		reportResult.TimeMetrics.AvgTotalTime = result.TimeMetrics.AvgTotalTime
		reportResult.TimeMetrics.MinTotalTime = result.TimeMetrics.MinTotalTime
		reportResult.TimeMetrics.MaxTotalTime = result.TimeMetrics.MaxTotalTime

		// 复制网络性能指标
		reportResult.NetworkMetrics.AvgDNSTime = result.NetworkMetrics.AvgDNSTime
		reportResult.NetworkMetrics.MinDNSTime = result.NetworkMetrics.MinDNSTime
		reportResult.NetworkMetrics.MaxDNSTime = result.NetworkMetrics.MaxDNSTime
		reportResult.NetworkMetrics.AvgConnectTime = result.NetworkMetrics.AvgConnectTime
		reportResult.NetworkMetrics.MinConnectTime = result.NetworkMetrics.MinConnectTime
		reportResult.NetworkMetrics.MaxConnectTime = result.NetworkMetrics.MaxConnectTime
		reportResult.NetworkMetrics.AvgTLSHandshakeTime = result.NetworkMetrics.AvgTLSHandshakeTime
		reportResult.NetworkMetrics.MinTLSHandshakeTime = result.NetworkMetrics.MinTLSHandshakeTime
		reportResult.NetworkMetrics.MaxTLSHandshakeTime = result.NetworkMetrics.MaxTLSHandshakeTime
		reportResult.NetworkMetrics.TargetIP = result.NetworkMetrics.TargetIP

		// 复制服务性能指标
		reportResult.ContentMetrics.AvgTTFT = result.ContentMetrics.AvgTTFT
		reportResult.ContentMetrics.MinTTFT = result.ContentMetrics.MinTTFT
		reportResult.ContentMetrics.MaxTTFT = result.ContentMetrics.MaxTTFT
		reportResult.ContentMetrics.AvgTokenCount = result.ContentMetrics.AvgTokenCount
		reportResult.ContentMetrics.MinTokenCount = result.ContentMetrics.MinTokenCount
		reportResult.ContentMetrics.MaxTokenCount = result.ContentMetrics.MaxTokenCount
		reportResult.ContentMetrics.AvgTPS = result.ContentMetrics.AvgTPS
		reportResult.ContentMetrics.MinTPS = result.ContentMetrics.MinTPS
		reportResult.ContentMetrics.MaxTPS = result.ContentMetrics.MaxTPS

		// 复制可靠性指标
		reportResult.ReliabilityMetrics.ErrorRate = result.ReliabilityMetrics.ErrorRate
		reportResult.ReliabilityMetrics.SuccessRate = result.ReliabilityMetrics.SuccessRate

		// 生成报告
		reporter := report.NewReporter(reportConfig, reportResult)
		if err := reporter.Generate(); err != nil {
			fmt.Printf("生成报告失败: %v\n", err)
		}
	}
}
