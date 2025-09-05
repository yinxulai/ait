package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/display"
	"github.com/yinxulai/ait/internal/report"
	"github.com/yinxulai/ait/internal/runner"
	"github.com/yinxulai/ait/internal/types"
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
	model := flag.String("model", "", "模型名称，支持多个模型用逗号分割")
	count := flag.Int("count", 10, "请求总数")
	protocol := flag.String("protocol", "", "协议类型: openai 或 anthropic")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", true, "是否开启流模式")
	concurrency := flag.Int("concurrency", 3, "并发数")
	reportFlag := flag.Bool("report", false, "是否生成报告文件")
	flag.Parse()

	// 自动推断 protocol 和加载环境变量
	finalProtocol := *protocol
	finalBaseUrl := *baseUrl
	finalApiKey := *apikey

	// 如果未指定 protocol，根据环境变量自动推断
	if finalProtocol == "" {
		finalProtocol = detectProviderFromEnv()
	}

	// 根据 protocol 加载对应的环境变量
	if finalBaseUrl == "" || finalApiKey == "" {
		envBaseUrl, envApiKey := loadEnvForProvider(finalProtocol)
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
		fmt.Println("支持多个模型，用逗号分割，例如：gpt-3.5-turbo,gpt-4")
		os.Exit(1)
	}

	// 解析多个模型
	models := strings.Split(*model, ",")
	for i, m := range models {
		models[i] = strings.TrimSpace(m)
	}

	// baseUrl 和 apikey 检查（可以通过环境变量获取）
	if finalBaseUrl == "" || finalApiKey == "" {
		fmt.Println("baseUrl 和 apikey 参数必填")
		fmt.Printf("对于 %s 协议，你也可以设置以下环境变量：\n", finalProtocol)

		switch finalProtocol {
		case "openai":
			fmt.Println("  OPENAI_BASE_URL - OpenAI API 基础 URL")
			fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥")
		case "anthropic":
			fmt.Println("  ANTHROPIC_BASE_URL - Anthropic API 基础 URL")
			fmt.Println("  ANTHROPIC_API_KEY - Anthropic API 密钥")
		}
		os.Exit(1)
	}

	// 用于汇总所有模型的测试结果
	var allResults []*types.ReportData

	// 循环处理每个模型
	for i, modelName := range models {
		fmt.Printf("\n=== 开始测试模型 [%d/%d]: %s ===\n", i+1, len(models), modelName)
		
		config := types.Input{
			Protocol:    finalProtocol,
			BaseUrl:     finalBaseUrl,
			ApiKey:      finalApiKey,
			Model:       modelName,
			Concurrency: *concurrency,
			Count:       *count,
			Prompt:      *prompt,
			Stream:      *stream,
			Report:      *reportFlag,
		}

		runnerInstance, err := runner.NewRunner(config)
		if err != nil {
			fmt.Printf("创建测试执行器失败: %v\n", err)
			continue
		}

		// 创建显示控制器
		testDisplayer := display.NewTestDisplayer(config)

		// 显示测试开始信息
		testDisplayer.ShowTestStart()

		// 用于保存最后的统计信息
		var finalStats types.StatsData

		// 执行测试，使用回调函数来更新显示
		result, err := runnerInstance.RunWithProgress(func(stats types.StatsData) {
			finalStats = stats // 保存最后的统计信息
			testDisplayer.UpdateProgress(stats)
		})

		if err != nil {
			testDisplayer.ShowError(fmt.Sprintf("执行测试失败: %v", err))
			continue
		}

		// 显示测试完成
		testDisplayer.ShowTestComplete()

		// 显示错误详情（如果有错误的话）
		testDisplayer.ShowErrorDetails(finalStats)

		// 直接使用 runner 的结果显示，无需转换
		display.PrintResult(result)

		// 保存结果用于汇总
		allResults = append(allResults, result)
	}

	// 显示汇总结果
	if len(allResults) > 1 {
		fmt.Printf("\n=== 所有模型测试结果汇总 ===\n")
		for i, result := range allResults {
			fmt.Printf("\n模型 %s 的测试结果:\n", models[i])
			fmt.Printf("  成功率: %.2f%%\n", result.ReliabilityMetrics.SuccessRate)
			fmt.Printf("  平均响应时间: %v\n", result.TimeMetrics.AvgTotalTime)
			fmt.Printf("  平均 TTFT: %v\n", result.ContentMetrics.AvgTTFT)
			fmt.Printf("  平均 TPS: %.2f\n", result.ContentMetrics.AvgTPS)
		}
	}

	// 如果启用了报告生成，则生成包含所有模型结果的汇总报告文件
	if *reportFlag && len(allResults) > 0 {
		// 为每个结果填充元数据
		reportDataList := make([]types.ReportData, len(allResults))
		for i, result := range allResults {
			reportData := *result
			reportData.Metadata.Timestamp = time.Now().Format(time.RFC3339)
			reportData.Metadata.Protocol = finalProtocol
			reportData.Metadata.Model = models[i]
			reportData.Metadata.BaseUrl = finalBaseUrl
			reportDataList[i] = reportData
		}

		// 使用 ReportManager 生成汇总报告
		manager := report.NewReportManager()
		filePaths, err := manager.GenerateReports(reportDataList, []string{"json", "csv"})
		if err != nil {
			fmt.Printf("生成汇总报告失败: %v\n", err)
		} else {
			fmt.Printf("\n汇总报告已生成:\n")
			for _, filePath := range filePaths {
				fmt.Printf("  - %s\n", filePath)
			}
		}
	}
}
