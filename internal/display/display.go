package display

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
	"github.com/yinxulai/ait/internal/types"
)

// Colors 定义终端颜色 - 导出供外部使用
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

type Input struct {
	Protocol    string
	BaseUrl     string
	ApiKey      string
	Models      []string // 多个模型列表
	Concurrency int
	Count       int
	Stream      bool
	Prompt      string
	Report      bool // 是否生成报告文件
	Timeout     int  // 请求超时时间(秒)
}

// Displayer 测试显示器
type Displayer struct {
	progressBar *progressbar.ProgressBar
	mu          sync.Mutex
}

// New 创建新的测试显示器
func New() *Displayer {
	return &Displayer{}
}

func (td *Displayer) ShowWelcome() {
	fmt.Printf("\n")
	// AIT ASCII 字符画
	fmt.Printf("%s%s", ColorBold, ColorCyan)
	fmt.Printf("    █████╗  ██╗ ████████╗\n")
	fmt.Printf("   ██╔══██╗ ██║ ╚══██╔══╝\n")
	fmt.Printf("   ███████║ ██║    ██║   \n")
	fmt.Printf("   ██╔══██║ ██║    ██║   \n")
	fmt.Printf("   ██║  ██║ ██║    ██║   \n")
	fmt.Printf("   ╚═╝  ╚═╝ ╚═╝    ╚═╝   \n")
	fmt.Printf("%s", ColorReset)
	fmt.Printf("\n")
	fmt.Printf("🚀 %s%sAI 模型性能测试工具%s\n", ColorBold, ColorCyan, ColorReset)
	fmt.Printf("   %s一个强大的 CLI 工具，用于测试 AI 模型的性能指标%s\n", ColorWhite, ColorReset)
	fmt.Printf("   %s🌐 项目地址: https://github.com/yinxulai/ait%s\n", ColorBlue, ColorReset)
	fmt.Printf("\n")
	fmt.Printf("✨ %s功能特性:%s\n", ColorBold, ColorReset)
	fmt.Printf("   🎯 多模型批量测试  ⚡ 并发压力测试  📊 实时进度显示\n")
	fmt.Printf("   🌐 网络性能分析  📈 详细统计报告  🎨 美观界面输出\n")
	fmt.Printf("\n")
}

func (td *Displayer) ShowInput(data *Input) {
	// 创建配置信息表格
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithEastAsian(false),
	)
	table.Header("配置项", "值", "说明")

	// 基础配置
	table.Append("🔗 协议", data.Protocol, "API 协议类型")
	table.Append("🌐 服务地址", data.BaseUrl, "API 基础 URL")
	table.Append("🔑 API 密钥", maskApiKey(data.ApiKey), "API 访问密钥（已隐藏）")

	// 模型配置
	modelsStr := ""
	if len(data.Models) > 0 {
		for i, model := range data.Models {
			if i > 0 {
				modelsStr += ", "
			}
			modelsStr += model
		}
	}
	table.Append("🤖 测试模型", modelsStr, "待测试的模型列表")

	// 测试参数
	table.Append("📊 请求总数", strconv.Itoa(data.Count), "每个模型的请求数量")
	table.Append("⚡ 并发数", strconv.Itoa(data.Concurrency), "同时发送的请求数")
	table.Append("⏱️ 超时时间", strconv.Itoa(data.Timeout)+"秒", "每个请求的超时时间")
	table.Append("🌊 流式模式", strconv.FormatBool(data.Stream), "是否启用流式响应")
	table.Append("📝 测试提示词", truncatePrompt(data.Prompt), "用于测试的提示内容")
	table.Append("📄 生成报告", strconv.FormatBool(data.Report), "是否生成测试报告文件")

	table.Render()
}

// InitProgress 初始化进度条
func (td *Displayer) InitProgress(total int, description string) {
	td.mu.Lock()
	defer td.mu.Unlock()

	td.progressBar = progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(50),
		progressbar.OptionThrottle(100), // 限制更新频率
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetRenderBlankState(true),
	)
}

func (td *Displayer) UpdateProgress(percent float64) {
	td.mu.Lock()
	defer td.mu.Unlock()

	if td.progressBar != nil {
		// 计算当前进度值（基于进度条的最大值）
		current := int(percent * float64(td.progressBar.GetMax()) / 100.0)
		td.progressBar.Set(current)
	}
}

// FinishProgress 完成进度条
func (td *Displayer) FinishProgress() {
	td.mu.Lock()
	defer td.mu.Unlock()

	if td.progressBar != nil {
		td.progressBar.Finish()
		fmt.Println() // 添加一个空行
		td.progressBar = nil
	}
}

func (td *Displayer) ShowErrorsReport(errors []*string) {
	if len(errors) == 0 {
		return
	}

	// 统计错误信息和出现次数
	errorCounts := make(map[string]int)
	totalErrors := 0

	for _, errorPtr := range errors {
		if errorPtr != nil {
			errorMsg := *errorPtr
			errorCounts[errorMsg]++
			totalErrors++
		}
	}

	if totalErrors == 0 {
		return
	}

	fmt.Printf("%s%s❌ 错误信息报告%s\n", ColorBold, ColorRed, ColorReset)
	fmt.Printf("   %s检测到 %d 个错误（%d 种不同类型）%s\n\n", ColorYellow, totalErrors, len(errorCounts), ColorReset)

	// 创建错误信息表格
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithEastAsian(false),
	)

	table.Header("序号", "错误详情", "出现次数")

	// 添加错误信息到表格
	index := 1
	for errorMsg, count := range errorCounts {
		// 如果错误信息太长，进行适当的截断和格式化
		displayMsg := errorMsg
		if len(displayMsg) > 100 {
			displayMsg = displayMsg[:97] + "..."
		}
		table.Append(fmt.Sprintf("%d", index), displayMsg, fmt.Sprintf("%d", count))
		index++
	}

	table.Render()
	fmt.Println()
}

// 将数据更新到终端上（刷新显示）
// 详细模式，展示所有 ReportData 的数据
func (td *Displayer) ShowSignalReport(data *types.ReportData) {
	// 单个综合表格
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithEastAsian(false),
	)

	table.Header("指标", "最小值", "平均值", "最大值", "单位", "采样方式说明")

	// 基础信息（这些只有单一值，只填最小值列）
	table.Append("🔗 协议", data.Metadata.Protocol, "", "", "-", "配置信息")
	table.Append("🤖 模型", data.Metadata.Model, "", "", "-", "配置信息")
	table.Append("🌐 URL", data.Metadata.BaseUrl, "", "", "-", "配置信息")
	table.Append("🌊 流式", strconv.FormatBool(data.IsStream), "", "", "-", "配置信息")
	table.Append("⚡ 并发数", strconv.Itoa(data.Concurrency), "", "", "个", "配置信息")
	table.Append("📊 总请求数", strconv.Itoa(data.TotalRequests), "", "", "个", "完成的请求总数")
	table.Append("✅ 成功率", fmt.Sprintf("%.2f", data.ReliabilityMetrics.SuccessRate), "", "", "%", "成功请求占比")

	// 时间性能指标
	table.Append("🕐 总耗时", data.TimeMetrics.MinTotalTime.String(), data.TimeMetrics.AvgTotalTime.String(), data.TimeMetrics.MaxTotalTime.String(), "时间", "单个请求从发起到完全结束的时间")

	if data.NetworkMetrics.TargetIP != "" {
		table.Append("🎯 目标 IP", data.NetworkMetrics.TargetIP, "", "", "-", "DNS 解析后的实际连接 IP")
	}

	// 网络性能指标
	table.Append("🔍 DNS 时间", data.NetworkMetrics.MinDNSTime.String(), data.NetworkMetrics.AvgDNSTime.String(), data.NetworkMetrics.MaxDNSTime.String(), "时间", "域名解析耗时 (httptrace)")
	table.Append("🔒 TLS 时间", data.NetworkMetrics.MinTLSHandshakeTime.String(), data.NetworkMetrics.AvgTLSHandshakeTime.String(), data.NetworkMetrics.MaxTLSHandshakeTime.String(), "时间", "TLS 握手耗时 (httptrace)")
	table.Append("🔌 TCP 连接时间", data.NetworkMetrics.MinConnectTime.String(), data.NetworkMetrics.AvgConnectTime.String(), data.NetworkMetrics.MaxConnectTime.String(), "时间", "TCP 连接建立耗时 (httptrace)")

	// 内容性能指标
	if data.IsStream {
		table.Append("⚡ TTFT", data.ContentMetrics.MinTTFT.String(), data.ContentMetrics.AvgTTFT.String(), data.ContentMetrics.MaxTTFT.String(), "时间", "首个 token 响应时间 (含请求发送+网络+服务器处理)")
	}

	table.Append("🎲 Token 数", strconv.Itoa(data.ContentMetrics.MinTokenCount), strconv.Itoa(data.ContentMetrics.AvgTokenCount), strconv.Itoa(data.ContentMetrics.MaxTokenCount), "个", "API 返回的 completion tokens")
	table.Append("🚀 TPS", fmt.Sprintf("%.2f", data.ContentMetrics.MinTPS), fmt.Sprintf("%.2f", data.ContentMetrics.AvgTPS), fmt.Sprintf("%.2f", data.ContentMetrics.MaxTPS), "个/秒", "tokens/总耗时 计算得出")

	table.Render()
	fmt.Println()
}

// 将数据更新到终端上（刷新显示）
// 概览模式，每行一个，展示主要数据（平均值）
func (td *Displayer) ShowMultiReport(data []*types.ReportData) {
	// 单个汇总表格，包含所有不同类型指标的平均值
	table := tablewriter.NewTable(
		os.Stdout,
		tablewriter.WithEastAsian(false),
	)

	table.Header("🤖 模型", "🎯 目标 IP", "📊 请求数", "⚡ 并发", "✅ 成功率",
		"🕐 平均总耗时", "⚡ 平均 TTFT", "🚀 平均 TPS", "🎲 平均 Token 数",
		"🔍 平均 DNS 时间", "🔌 平均 TCP 连接时间", "🔒 平均 TLS 时间")

	for _, report := range data {
		// TTFT 处理（流式模式才显示）
		ttftStr := "-"
		if report.IsStream {
			ttftStr = report.ContentMetrics.AvgTTFT.String()
		}

		table.Append(
			report.Metadata.Model,
			report.NetworkMetrics.TargetIP,
			strconv.Itoa(report.TotalRequests),
			strconv.Itoa(report.Concurrency),
			fmt.Sprintf("%.2f%%", report.ReliabilityMetrics.SuccessRate),
			report.TimeMetrics.AvgTotalTime.String(),
			ttftStr,
			fmt.Sprintf("%.2f", report.ContentMetrics.AvgTPS),
			strconv.Itoa(report.ContentMetrics.AvgTokenCount),
			report.NetworkMetrics.AvgDNSTime.String(),
			report.NetworkMetrics.AvgConnectTime.String(),
			report.NetworkMetrics.AvgTLSHandshakeTime.String(),
		)
	}

	table.Render()
	fmt.Println()
}

// maskApiKey 隐藏 API 密钥的敏感部分
func maskApiKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

// truncatePrompt 截断过长的提示词并显示长度信息
func truncatePrompt(prompt string) string {
	runes := []rune(prompt)
	charCount := len(runes)
	if charCount <= 50 {
		return fmt.Sprintf("%s (长度: %d)", prompt, charCount)
	}
	return fmt.Sprintf("%s... (长度: %d)", string(runes[:47]), charCount)
}
