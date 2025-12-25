package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/display"
	"github.com/yinxulai/ait/internal/prompt"
	"github.com/yinxulai/ait/internal/report"
	"github.com/yinxulai/ait/internal/runner"
	"github.com/yinxulai/ait/internal/types"
)

// 版本信息，通过 ldflags 在构建时注入
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func generateTaskID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)

	// 设置版本 (4) 和变体位
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

// readPromptFromStdin 从标准输入读取 prompt 内容
func readPromptFromStdin() (string, error) {
	// 检查是否有标准输入数据
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	// 如果没有管道输入，返回空字符串
	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}

	// 读取标准输入的所有内容
	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// resolvePrompt 解析最终的 prompt 内容
// 优先级：1. prompt-length 参数 > 2. prompt-file 参数 > 3. prompt 参数 > 4. 管道输入 > 5. 默认值
func resolvePrompt(promptLengthSpecified bool, promptLength int, promptSpecified bool, flagPrompt string, promptFileSpecified bool, flagPromptFile string) (*prompt.PromptSource, error) {
	// 1. 如果用户指定了 --prompt-length 参数，优先使用长度生成
	if promptLengthSpecified && promptLength > 0 {
		return prompt.LoadPromptByLength(promptLength)
	}

	// 2. 如果用户指定了 --prompt-file 参数，使用文件
	if promptFileSpecified {
		return prompt.LoadPromptsFromFile(flagPromptFile)
	}

	// 3. 如果用户明确指定了 --prompt 参数，则使用它
	if promptSpecified {
		return prompt.LoadPrompts(flagPrompt)
	}

	// 4. 检查是否有管道输入
	stdinPrompt, err := readPromptFromStdin()
	if err == nil && stdinPrompt != "" {
		return prompt.LoadPrompts(stdinPrompt)
	}

	// 5. 使用默认值
	return prompt.LoadPrompts(flagPrompt)
}

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

// validateRequiredParams 验证必需的参数
func validateRequiredParams(models, baseUrl, apiKey, protocol string) error {
	if models == "" {
		return fmt.Errorf("models 参数必填，请通过 -models 参数指定")
	}

	if baseUrl == "" || apiKey == "" {
		return fmt.Errorf("baseUrl 和 apikey 参数必填，对于 %s 协议，你也可以设置相应的环境变量", protocol)
	}

	return nil
}

// parseModelList 解析模型列表字符串
func parseModelList(models string) []string {
	if models == "" {
		return nil
	}

	modelList := strings.Split(models, ",")
	for i, m := range modelList {
		modelList[i] = strings.TrimSpace(m)
	}
	return modelList
}

// resolveConfigValues 解析并合并配置值
func resolveConfigValues(protocol, baseUrl, apiKey string) (string, string, string) {
	finalProtocol := protocol
	finalBaseUrl := baseUrl
	finalApiKey := apiKey

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

	return finalProtocol, finalBaseUrl, finalApiKey
}

// printErrorMessages 打印错误消息并提供环境变量设置建议
func printErrorMessages(protocol string) {
	fmt.Println("baseUrl 和 apikey 参数必填")
	fmt.Printf("对于 %s 协议，你也可以设置以下环境变量：\n", protocol)

	switch protocol {
	case "openai":
		fmt.Println("  OPENAI_BASE_URL - OpenAI API 基础 URL")
		fmt.Println("  OPENAI_API_KEY - OpenAI API 密钥")
	case "anthropic":
		fmt.Println("  ANTHROPIC_BASE_URL - Anthropic API 基础 URL")
		fmt.Println("  ANTHROPIC_API_KEY - Anthropic API 密钥")
	}
}

// createRunnerConfig 创建runner配置
func createRunnerConfig(protocol, baseUrl, apiKey, model string, promptSource *prompt.PromptSource, concurrency, count, timeout int, stream, report, log, thinking bool) types.Input {
	return types.Input{
		Protocol:     protocol,
		BaseUrl:      baseUrl,
		ApiKey:       apiKey,
		Model:        model,
		Concurrency:  concurrency,
		Count:        count,
		PromptSource: promptSource,
		Stream:       stream,
		Report:       report,
		Timeout:      time.Duration(timeout) * time.Second,
		Log:          log,
		Thinking:     thinking,
	}
}

// processModelExecution 处理单个模型的执行逻辑
func processModelExecution(taskID string, modelName string, config types.Input, displayer *display.Displayer, completedRequests, totalRequests int) (*types.ReportData, []string, error) {
	runnerInstance, err := runner.NewRunner(taskID, config)
	if err != nil {
		return nil, nil, fmt.Errorf("创建测试执行器失败: %v", err)
	}

	// 用于收集当前模型的错误信息
	var currentModelErrors []string

	// 执行测试，使用回调函数来更新显示
	result, err := runnerInstance.RunWithProgress(func(sd types.StatsData) {
		// 计算当前总完成数：之前模型的完成数 + 当前模型的完成数
		currentCompleted := completedRequests + sd.CompletedCount + sd.FailedCount

		// 计算百分比
		percent := float64(currentCompleted) / float64(totalRequests) * 100.0

		// 类型断言来调用UpdateProgress方法
		displayer.UpdateProgress(percent)

		// 保存最新的错误信息（覆盖之前的，确保获取最完整的错误列表）
		currentModelErrors = make([]string, len(sd.ErrorMessages))
		copy(currentModelErrors, sd.ErrorMessages)
	})
	if err != nil {
		return nil, nil, err
	}

	return result, currentModelErrors, nil
}

// collectErrorsWithContext 收集带有模型上下文的错误信息
func collectErrorsWithContext(modelName string, modelErrors []string) []string {
	var errors []string
	for _, errorMsg := range modelErrors {
		if errorMsg != "" {
			// 为错误信息添加模型上下文
			errorWithContext := fmt.Sprintf("[%s] %s", modelName, errorMsg)
			errors = append(errors, errorWithContext)
		}
	}
	return errors
}

// fillResultMetadata 填充结果元数据
func fillResultMetadata(results []*types.ReportData, modelList []string, baseUrl, protocol string) {
	for i, result := range results {
		result.Model = modelList[i]
		result.BaseUrl = baseUrl
		result.Protocol = protocol
		result.Timestamp = time.Now().Format(time.RFC3339)
	}
}

func convertErrorsToPointers(errors []string) []*string {
	errorPtrs := make([]*string, len(errors))
	for i := range errors {
		errorPtrs[i] = &errors[i]
	}
	return errorPtrs
}

// generateReportsIfEnabled 如果启用了报告功能，则生成报告
func generateReportsIfEnabled(reportFlag bool, results []*types.ReportData) error {
	if !reportFlag || len(results) == 0 {
		return nil
	}

	// 转换为 ReportData 切片
	reportDataList := make([]types.ReportData, len(results))
	for i, result := range results {
		reportDataList[i] = *result
	}

	// 使用 ReportManager 生成汇总报告
	manager := report.NewReportManager()
	filePaths, err := manager.GenerateReports(reportDataList, []string{"json", "csv"})
	if err != nil {
		return fmt.Errorf("生成汇总报告失败: %v", err)
	}

	fmt.Printf("\n汇总报告已生成:\n")
	for _, filePath := range filePaths {
		fmt.Printf("  - %s\n", filePath)
	}
	return nil
}

// executeModelsTestSuite 执行多个模型的测试套件
func executeModelsTestSuite(taskID string, modelList []string, finalProtocol, finalBaseUrl, finalApiKey string, promptSource *prompt.PromptSource, concurrency, count, timeout int, stream, reportFlag, log, thinking bool, displayer *display.Displayer) ([]*types.ReportData, []string, error) {
	// 用于收集所有错误信息
	var allErrors []string

	// 用于汇总所有模型的测试结果
	var allResults []*types.ReportData

	// 循环处理每个模型
	totalRequests := count * len(modelList)

	// 初始化总进度条
	displayer.InitProgress(totalRequests, fmt.Sprintf("🚀 测试进度 (%d 个模型)", len(modelList)))

	completedRequests := 0

	for _, modelName := range modelList {
		config := createRunnerConfig(finalProtocol, finalBaseUrl, finalApiKey, modelName, promptSource, concurrency, count, timeout, stream, reportFlag, log, thinking)

		result, currentModelErrors, err := processModelExecution(taskID, modelName, config, displayer, completedRequests, totalRequests)
		if err != nil {
			fmt.Printf("模型 %s 执行失败: %v\n", modelName, err)
			continue
		}

		// 处理当前模型的错误信息
		modelErrors := collectErrorsWithContext(modelName, currentModelErrors)
		allErrors = append(allErrors, modelErrors...)

		// 更新已完成的请求数（当前模型的所有请求都已完成）
		completedRequests += config.Count

		// 保存结果用于汇总
		allResults = append(allResults, result)
	}

	// 完成进度条
	displayer.FinishProgress()

	// 为所有结果填充模型名称元数据
	fillResultMetadata(allResults, modelList, finalBaseUrl, finalProtocol)

	return allResults, allErrors, nil
}

func main() {
	taskID := generateTaskID()
	versionFlag := flag.Bool("version", false, "显示版本信息")
	baseUrl := flag.String("baseUrl", "", "服务地址")
	apiKey := flag.String("apiKey", "", "API 密钥")
	count := flag.Int("count", 10, "请求总数")
	model := flag.String("model", "", "模型名称（单个模型）")
	models := flag.String("models", "", "模型名称，支持多个模型用,(逗号)分割")
	protocol := flag.String("protocol", "", "协议类型: openai 或 anthropic")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt 内容。未指定时支持管道输入")
	promptFile := flag.String("prompt-file", "", "从文件读取 prompt。支持单文件路径或通配符 (如: prompts/*.txt)")
	promptLength := flag.Int("prompt-length", 0, "生成指定长度的测试 prompt（字符数）。优先级高于其他 prompt 参数")
	stream := flag.Bool("stream", true, "是否开启流模式")
	concurrency := flag.Int("concurrency", 3, "并发数")
	reportFlag := flag.Bool("report", false, "是否生成报告文件")
	timeout := flag.Int("timeout", 300, "请求超时时间(秒)")
	logFlag := flag.Bool("log", false, "是否开启详细日志记录")
	thinking := flag.Bool("thinking", false, "是否开启 thinking 模式")
	flag.Parse()

	// 如果指定了 --version，显示版本信息后退出
	if *versionFlag {
		fmt.Printf("ait version %s\n", Version)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// 合并 --model 和 --models 参数
	finalModels := *models
	if *model != "" {
		if finalModels != "" {
			fmt.Println("错误：不能同时使用 --model 和 --models 参数")
			os.Exit(1)
		}
		finalModels = *model
	}

	// 解析和验证配置
	finalProtocol, finalBaseUrl, finalApiKey := resolveConfigValues(*protocol, *baseUrl, *apiKey)

	// 验证必需参数
	if err := validateRequiredParams(finalModels, finalBaseUrl, finalApiKey, finalProtocol); err != nil {
		if finalModels == "" {
			fmt.Println("model/models 参数必填，请通过 --model 或 --models 参数指定")
			fmt.Println("--model: 指定单个模型，例如：--model gpt-3.5-turbo")
			fmt.Println("--models: 支持多个模型，用逗号分割，例如：--models gpt-3.5-turbo,gpt-4")
		} else {
			printErrorMessages(finalProtocol)
		}
		os.Exit(1)
	}

	// 解析模型列表
	modelList := parseModelList(finalModels)

	// 检查用户是否明确指定了 --prompt、--prompt-file 和 --prompt-length 参数
	promptSpecified := false
	promptFileSpecified := false
	promptLengthSpecified := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "prompt" {
			promptSpecified = true
		}
		if f.Name == "prompt-file" {
			promptFileSpecified = true
		}
		if f.Name == "prompt-length" {
			promptLengthSpecified = true
		}
	})

	// 解析最终的 prompt，优先级：prompt-length > prompt-file > prompt > 管道输入 > 默认值
	promptSource, err := resolvePrompt(promptLengthSpecified, *promptLength, promptSpecified, *prompt, promptFileSpecified, *promptFile)
	if err != nil {
		fmt.Printf("解析 prompt 失败: %v\n", err)
		os.Exit(1)
	}

	displayer := display.New()

	// 显示欢迎信息
	displayer.ShowWelcome(Version)

	displayer.ShowInput(&display.Input{
		TaskId:               taskID,
		Protocol:             finalProtocol,
		BaseUrl:              finalBaseUrl,
		ApiKey:               finalApiKey,
		Models:               modelList,
		Concurrency:          *concurrency,
		Count:                *count,
		Stream:               *stream,
		Thinking:             *thinking,
		PromptText:           promptSource.DisplayText,
		PromptShouldTruncate: promptSource.ShouldTruncate,
		IsFile:               promptSource.IsFile,
		Report:               *reportFlag,
		Timeout:              *timeout,
	})

	// 执行多个模型的测试套件
	allResults, allErrors, err := executeModelsTestSuite(
		taskID, modelList, finalProtocol, finalBaseUrl, finalApiKey, promptSource,
		*concurrency, *count, *timeout, *stream, *reportFlag, *logFlag, *thinking, displayer,
	)
	if err != nil {
		fmt.Printf("执行测试套件失败: %v\n", err)
		os.Exit(1)
	}

	// 显示错误报告（如果有错误的话）
	if len(allErrors) > 0 {
		errorPtrs := convertErrorsToPointers(allErrors)
		displayer.ShowErrorsReport(errorPtrs)
	}

	// 根据模型数量显示相应的报告
	if len(modelList) == 1 {
		displayer.ShowSignalReport(allResults[0])
	}

	if len(modelList) > 1 {
		displayer.ShowMultiReport(allResults)
	}

	// 生成报告文件（如果启用）
	if err := generateReportsIfEnabled(*reportFlag, allResults); err != nil {
		fmt.Printf("报告生成失败: %v\n", err)
	}
}
