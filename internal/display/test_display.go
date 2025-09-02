package display

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
)

// TestDisplayer 测试显示控制器
type TestDisplayer struct {
	config      TestConfig
	progressBar *progressbar.ProgressBar
	startTime   time.Time

	// 颜色配置
	titleColor   *color.Color
	infoColor    *color.Color
	successColor *color.Color
	errorColor   *color.Color
	warningColor *color.Color
	statsColor   *color.Color
}

// TestConfig 测试显示配置
type TestConfig struct {
	Provider    string
	BaseUrl     string
	ApiKey      string
	Model       string
	Concurrency int
	Count       int
	Stream      bool
}

// TestStats 实时测试统计数据
type TestStats struct {
	// 基础统计
	CompletedCount int // 已完成请求数
	FailedCount    int // 失败请求数

	// 时间指标
	TTFTs      []time.Duration // 所有首个token响应时间 (Time to First Token)
	TotalTimes []time.Duration // 所有总耗时

	// 网络指标
	DNSTimes          []time.Duration // 所有DNS解析时间
	ConnectTimes      []time.Duration // 所有TCP连接时间
	TLSHandshakeTimes []time.Duration // 所有TLS握手时间

	// 服务性能指标
	TokenCounts []int // 所有 token 数量

	// 错误信息
	ErrorMessages []string // 所有错误信息

	// 测试控制
	StartTime   time.Time     // 测试开始时间
	ElapsedTime time.Duration // 已经过时间
}

// NewTestDisplayer 创建新的测试显示控制器
func NewTestDisplayer(config TestConfig) *TestDisplayer {
	return &TestDisplayer{
		config:       config,
		titleColor:   color.New(color.FgCyan, color.Bold),
		infoColor:    color.New(color.FgBlue),
		successColor: color.New(color.FgGreen, color.Bold),
		errorColor:   color.New(color.FgRed, color.Bold),
		warningColor: color.New(color.FgYellow, color.Bold),
		statsColor:   color.New(color.FgMagenta),
	}
}

// ShowTestStart 显示测试开始信息
func (td *TestDisplayer) ShowTestStart() {
	// 清屏
	fmt.Print("\033[H\033[2J")

	// 显示标题
	td.printTitle("🚀 AI 模型性能测试")
	fmt.Println()

	// 显示配置信息
	td.printConfigTable()
	fmt.Println()

	// 显示准备提示
	td.infoColor.Println("⏳ 准备开始测试...")
	time.Sleep(1 * time.Second)

	// 创建进度条
	td.progressBar = progressbar.NewOptions(td.config.Count,
		progressbar.OptionSetDescription("🔥 执行测试中"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionSetWidth(50),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("req"),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	)

	td.startTime = time.Now()
	fmt.Println() // 为实时统计预留空间
}

// printTitle 打印美化的标题
func (td *TestDisplayer) printTitle(title string) {
	width := 80
	padding := (width - len(title) - 2) / 2
	border := strings.Repeat("═", width)

	td.titleColor.Println(border)
	td.titleColor.Printf("║%s%s%s║\n",
		strings.Repeat(" ", padding),
		title,
		strings.Repeat(" ", width-padding-len(title)-2))
	td.titleColor.Println(border)
}

// printConfigTable 打印配置信息表格
func (td *TestDisplayer) printConfigTable() {
	// 隐藏 API Key 中间部分
	apiKeyDisplay := td.config.ApiKey
	if len(apiKeyDisplay) > 8 {
		start := apiKeyDisplay[:4]
		end := apiKeyDisplay[len(apiKeyDisplay)-4:]
		apiKeyDisplay = start + "****" + end
	}

	streamMode := "❌ 关闭"
	if td.config.Stream {
		streamMode = "✅ 开启"
	}

	// 使用 tablewriter 创建配置表格
	fmt.Println("📋 测试配置：")

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("配置项", "值")

	// 添加数据行
	table.Append([]string{"Provider", td.config.Provider})
	table.Append([]string{"BaseURL", td.truncateString(td.config.BaseUrl, 40)})
	table.Append([]string{"ApiKey", apiKeyDisplay})
	table.Append([]string{"Model", td.config.Model})
	table.Append([]string{"并发数", fmt.Sprintf("%d", td.config.Concurrency)})
	table.Append([]string{"总请求数", fmt.Sprintf("%d", td.config.Count)})
	table.Append([]string{"流模式", streamMode})

	table.Render()
}

// truncateString 截断字符串以适应表格宽度
func (td *TestDisplayer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// UpdateProgress 更新测试进度
func (td *TestDisplayer) UpdateProgress(stats TestStats) {
	// 更新进度条
	if td.progressBar != nil {
		td.progressBar.Set(stats.CompletedCount)
	}
}

// ShowTestComplete 显示测试完成
func (td *TestDisplayer) ShowTestComplete() {
	// 完成进度条
	if td.progressBar != nil {
		td.progressBar.Finish()
	}
	fmt.Println()
	td.successColor.Println("🎉 测试完成！")
	fmt.Println()
}

// ShowTestSummary 显示测试摘要（在最终结果之前）
func (td *TestDisplayer) ShowTestSummary(stats TestStats) {
	titleColor := color.New(color.FgCyan, color.Bold)
	titleColor.Println("📋 测试摘要")

	elapsed := stats.ElapsedTime
	successRate := float64(stats.CompletedCount) / float64(td.config.Count) * 100

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("指标", "值")

	table.Append([]string{"⏱️  测试时长", FormatDuration(elapsed)})
	table.Append([]string{"✅ 成功请求", fmt.Sprintf("%d", stats.CompletedCount)})
	table.Append([]string{"❌ 失败请求", fmt.Sprintf("%d", stats.FailedCount)})
	table.Append([]string{"📊 成功率", fmt.Sprintf("%.1f%%", successRate)})

	if len(stats.TTFTs) > 0 {
		// 计算 Token-based TPS
		var currentTPS float64
		if len(stats.TokenCounts) > 0 {
			totalTokens := 0
			for _, count := range stats.TokenCounts {
				totalTokens += count
			}
			currentTPS = float64(totalTokens) / elapsed.Seconds()
		} else {
			currentTPS = 0
		}
		table.Append([]string{"🚀 平均TPS", fmt.Sprintf("%.2f", currentTPS)})
	}

	table.Render()
	
	// 如果有错误，显示错误信息
	if len(stats.ErrorMessages) > 0 {
		fmt.Println()
		td.errorColor.Println("🚨 错误详情：")
		
		// 统计错误类型
		errorCounts := make(map[string]int)
		for _, errMsg := range stats.ErrorMessages {
			errorCounts[errMsg]++
		}
		
		// 显示错误统计
		errorTable := tablewriter.NewWriter(os.Stdout)
		errorTable.Header("错误信息", "出现次数")
		
		for errMsg, count := range errorCounts {
			// 截断过长的错误信息
			displayMsg := errMsg
			if len(displayMsg) > 60 {
				displayMsg = displayMsg[:57] + "..."
			}
			errorTable.Append([]string{displayMsg, fmt.Sprintf("%d", count)})
		}
		
		errorTable.Render()
	}
	
	fmt.Println()
}

// ShowError 显示错误信息
func (td *TestDisplayer) ShowError(message string) {
	td.errorColor.Printf("❌ %s\n", message)
}

// ShowErrorDetails 显示错误详情
func (td *TestDisplayer) ShowErrorDetails(stats TestStats) {
	// 如果有错误，显示错误信息
	if len(stats.ErrorMessages) > 0 {
		fmt.Println()
		td.errorColor.Println("🚨 错误详情：")
		
		// 统计错误类型
		errorCounts := make(map[string]int)
		for _, errMsg := range stats.ErrorMessages {
			errorCounts[errMsg]++
		}
		
		// 显示错误统计
		errorTable := tablewriter.NewWriter(os.Stdout)
		errorTable.Header("错误信息", "出现次数")
		
		for errMsg, count := range errorCounts {
			// 截断过长的错误信息
			displayMsg := errMsg
			if len(displayMsg) > 60 {
				displayMsg = displayMsg[:57] + "..."
			}
			errorTable.Append([]string{displayMsg, fmt.Sprintf("%d", count)})
		}
		
		errorTable.Render()
		fmt.Println()
	}
}

// Result 性能测试结果
type Result struct {
	// 基础测试信息
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration

	// 时间性能指标
	TimeMetrics struct {
		AvgTotalTime time.Duration // 总耗时指标
		MinTotalTime time.Duration
		MaxTotalTime time.Duration
	}

	// 网络性能指标
	NetworkMetrics struct {
		AvgDNSTime time.Duration // DNS解析时间指标
		MinDNSTime time.Duration
		MaxDNSTime time.Duration

		AvgConnectTime time.Duration // TCP连接时间指标
		MinConnectTime time.Duration
		MaxConnectTime time.Duration

		AvgTLSHandshakeTime time.Duration // TLS握手时间指标
		MinTLSHandshakeTime time.Duration
		MaxTLSHandshakeTime time.Duration
		
		TargetIP string // 目标服务器IP地址
	}

	// 服务性能指标
	ContentMetrics struct {
		AvgTTFT time.Duration // TTFT (Time to First Token) 指标
		MinTTFT time.Duration
		MaxTTFT time.Duration
		
		AvgTokenCount int // Token 统计指标
		MinTokenCount int
		MaxTokenCount int
		
		AvgTPS float64 // TPS (Tokens Per Second) 指标
		MinTPS float64
		MaxTPS float64
	}

	// 可靠性指标
	ReliabilityMetrics struct {
		ErrorRate    float64 // 错误率百分比
		SuccessRate  float64 // 成功率百分比
	}
}

// PrintResult 输出结果
func (r *Result) PrintResult() {
	titleColor := color.New(color.FgCyan, color.Bold)
	titleColor.Println("\n📊 测试结果")

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("指标", "最小值", "平均值", "最大值", "单位")

	table.Append([]string{"🎯 目标服务器 IP", "-", r.NetworkMetrics.TargetIP, "-", ""})
	table.Append([]string{"📊 总请求数", "-", fmt.Sprintf("%d", r.TotalRequests), "-", "个"})

	// 添加总耗时指标
	table.Append([]string{"⌛ 请求耗时",
		FormatDuration(r.TimeMetrics.MinTotalTime),
		FormatDuration(r.TimeMetrics.AvgTotalTime),
		FormatDuration(r.TimeMetrics.MaxTotalTime), ""})

	// 添加网络性能指标分组
	table.Append([]string{"🌐 DNS 解析时间",
		FormatDuration(r.NetworkMetrics.MinDNSTime),
		FormatDuration(r.NetworkMetrics.AvgDNSTime),
		FormatDuration(r.NetworkMetrics.MaxDNSTime), ""})

	table.Append([]string{"🔗 TCP 连接时间",
		FormatDuration(r.NetworkMetrics.MinConnectTime),
		FormatDuration(r.NetworkMetrics.AvgConnectTime),
		FormatDuration(r.NetworkMetrics.MaxConnectTime), ""})

	table.Append([]string{"🔒 TLS 握手时间",
		FormatDuration(r.NetworkMetrics.MinTLSHandshakeTime),
		FormatDuration(r.NetworkMetrics.AvgTLSHandshakeTime),
		FormatDuration(r.NetworkMetrics.MaxTLSHandshakeTime), ""})


	// 添加服务性能指标
	table.Append([]string{"🔤 Token 数量",
		fmt.Sprintf("%d", r.ContentMetrics.MinTokenCount),
		fmt.Sprintf("%d", r.ContentMetrics.AvgTokenCount),
		fmt.Sprintf("%d", r.ContentMetrics.MaxTokenCount), "个"})
	table.Append([]string{"🚀 TTFT (首个Token)",
		FormatDuration(r.ContentMetrics.MinTTFT),
		FormatDuration(r.ContentMetrics.AvgTTFT),
		FormatDuration(r.ContentMetrics.MaxTTFT), ""})
	table.Append([]string{"🚀 TPS(每秒 Token)",
		FormatFloat(r.ContentMetrics.MinTPS, 2),
		FormatFloat(r.ContentMetrics.AvgTPS, 2),
		FormatFloat(r.ContentMetrics.MaxTPS, 2), "tokens/s"})

	// 添加可靠性指标
	table.Append([]string{"✅ 成功率", "-", FormatFloat(r.ReliabilityMetrics.SuccessRate, 2), "-", "%"})
	table.Append([]string{"❌ 错误率", "-", FormatFloat(r.ReliabilityMetrics.ErrorRate, 2), "-", "%"})

	table.Render()

	// 显示模式提示
	fmt.Println()
	r.printModeInfo()
}

// printModeInfo 打印测试模式信息
func (r *Result) printModeInfo() {
	infoColor := color.New(color.FgBlue)

	if r.IsStream {
		infoColor.Println("💡 流式模式：可以准确测量 TTFT（首个令牌时间）和流式响应特性")
	} else {
		infoColor.Println("ℹ️  非流式模式：测量完整响应时间和批量处理性能")
	}

	// 显示详细的指标说明
	fmt.Println("\n📖 指标说明：")
	
	// 基础测试信息
	infoColor.Println("【基础信息】")
	infoColor.Println("  • 目标服务器 IP: 实际连接的服务器IP地址")
	infoColor.Println("  • 总请求数: 测试执行的请求总数量")
	infoColor.Println("  • 并发数: 同时进行的并发请求数量")
	
	// 时间性能指标
	infoColor.Println("\n【时间性能指标】")
	infoColor.Println("  • 请求耗时: 从发起请求到接收完整响应的总时间")
	if r.IsStream {
		infoColor.Println("  • TTFT: Time To First Token，首个令牌返回时间")
		infoColor.Println("    - 反映模型开始生成响应的速度，流式模式下的关键指标")
	} else {
		infoColor.Println("  • 响应时间: 完整请求-响应周期的时间")
		infoColor.Println("    - 非流式模式下测量完整响应的总时间")
	}
	
	// 网络性能指标
	infoColor.Println("\n【网络性能指标】")
	infoColor.Println("  • DNS 解析时间: 域名解析为IP地址所需时间")
	infoColor.Println("  • TCP 连接时间: 建立TCP连接所需时间")
	infoColor.Println("  • TLS 握手时间: 完成TLS/SSL握手所需时间")
	infoColor.Println("    - 这些指标帮助分析网络层面的性能瓶颈")
	
	// 服务性能指标
	infoColor.Println("\n【服务性能指标】")
	infoColor.Println("  • Token 数量: API 返回的 token 总数（输入+输出）")
	infoColor.Println("  • TPS: Tokens Per Second，每秒处理的令牌数")
	infoColor.Println("    - 衡量AI模型实际处理能力的核心指标")
	
	// 可靠性指标
	infoColor.Println("\n【可靠性指标】")
	infoColor.Println("  • 成功率: 成功完成的请求占总请求的百分比")
	infoColor.Println("  • 错误率: 失败请求占总请求的百分比")
	infoColor.Println("    - 评估服务稳定性和可靠性的重要指标")
}
