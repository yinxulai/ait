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
	apiKey := flag.String("apiKey", "", "API 密钥")
	count := flag.Int("count", 10, "请求总数")
	models := flag.String("models", "", "模型名称，支持多个模型用,(逗号)分割")
	protocol := flag.String("protocol", "", "协议类型: openai 或 anthropic")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", true, "是否开启流模式")
	concurrency := flag.Int("concurrency", 3, "并发数")
	reportFlag := flag.Bool("report", false, "是否生成报告文件")
	timeout := flag.Int("timeout", 30, "请求超时时间(秒)")
	flag.Parse()

	// 自动推断 protocol 和加载环境变量
	finalProtocol := *protocol
	finalBaseUrl := *baseUrl
	finalApiKey := *apiKey

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
	if *models == "" {
		fmt.Println("model 参数必填，请通过 -model 参数指定")
		fmt.Println("支持多个模型，用逗号分割，例如：gpt-3.5-turbo,gpt-4")
		os.Exit(1)
	}

	// 解析多个模型
	modelList := strings.Split(*models, ",")
	for i, m := range modelList {
		modelList[i] = strings.TrimSpace(m)
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

	// 用于收集所有错误信息
	var allErrors []string

	// 用于汇总所有模型的测试结果
	var allResults []*types.ReportData

	displayer := display.New()

	// 显示欢迎信息
	displayer.ShowWelcome()

	displayer.ShowInput(&display.Input{
		Protocol:    finalProtocol,
		BaseUrl:     finalBaseUrl,
		ApiKey:      finalApiKey,
		Models:      modelList,
		Concurrency: *concurrency,
		Count:       *count,
		Stream:      *stream,
		Prompt:      *prompt,
		Report:      *reportFlag,
		Timeout:     *timeout,
	})

	// 循环处理每个模型
	totalRequests := *count * len(modelList)

	// 初始化总进度条
	displayer.InitProgress(totalRequests, fmt.Sprintf("🚀 测试进度 (%d 个模型)", len(modelList)))

	completedRequests := 0

	for _, modelName := range modelList {

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
			Timeout:     time.Duration(*timeout) * time.Second,
		}

		runnerInstance, err := runner.NewRunner(config)
		if err != nil {
			fmt.Printf("创建测试执行器失败: %v\n", err)
			continue
		}

		// 用于收集当前模型的错误信息
		var currentModelErrors []string

		// 执行测试，使用回调函数来更新显示
		result, err := runnerInstance.RunWithProgress(func(sd types.StatsData) {
			// 计算当前总完成数：之前模型的完成数 + 当前模型的完成数
			currentCompleted := completedRequests + sd.CompletedCount + sd.FailedCount

			// 计算百分比
			percent := float64(currentCompleted) / float64(totalRequests) * 100.0
			displayer.UpdateProgress(percent)

			// 保存最新的错误信息（覆盖之前的，确保获取最完整的错误列表）
			currentModelErrors = make([]string, len(sd.ErrorMessages))
			copy(currentModelErrors, sd.ErrorMessages)
		})
		if err != nil {
			panic(err)
		}

		// 处理当前模型的错误信息
		for _, errorMsg := range currentModelErrors {
			if errorMsg != "" {
				// 为错误信息添加模型上下文
				errorWithContext := fmt.Sprintf("[%s] %s", modelName, errorMsg)
				allErrors = append(allErrors, errorWithContext)
			}
		}

		// 更新已完成的请求数（当前模型的所有请求都已完成）
		completedRequests += config.Count

		// 保存结果用于汇总
		allResults = append(allResults, result)
	}

	// 完成进度条
	displayer.FinishProgress()

	// 为所有结果填充模型名称元数据
	for i, result := range allResults {
		result.Metadata.Model = modelList[i]
		result.Metadata.BaseUrl = finalBaseUrl
		result.Metadata.Protocol = finalProtocol
		result.Metadata.Timestamp = time.Now().Format(time.RFC3339)
	}

	// 显示错误报告（如果有错误的话）
	if len(allErrors) > 0 {
		// 将 []string 转换为 []*string
		errorPtrs := make([]*string, len(allErrors))
		for i := range allErrors {
			errorPtrs[i] = &allErrors[i]
		}
		displayer.ShowErrorsReport(errorPtrs)
	}

	if len(modelList) == 1 {
		displayer.ShowSignalReport(allResults[0])
	}

	if len(modelList) > 1 {
		displayer.ShowMultiReport(allResults)
	}

	// 如果启用了报告生成，则生成包含所有模型结果的汇总报告文件
	if *reportFlag && len(allResults) > 0 {
		// 转换为 ReportData 切片
		reportDataList := make([]types.ReportData, len(allResults))
		for i, result := range allResults {
			reportDataList[i] = *result
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
