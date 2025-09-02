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
	CompletedCount int
	FailedCount    int
	ResponseTimes  []time.Duration
	StartTime      time.Time
	ElapsedTime    time.Duration
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
	
	// 更新实时统计
	td.printRealTimeStats(stats)
}

// ShowTestComplete 显示测试完成
func (td *TestDisplayer) ShowTestComplete() {
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
	
	table.Append([]string{"测试时长", FormatDuration(elapsed)})
	table.Append([]string{"成功请求", fmt.Sprintf("%d", stats.CompletedCount)})
	table.Append([]string{"失败请求", fmt.Sprintf("%d", stats.FailedCount)})
	table.Append([]string{"成功率", fmt.Sprintf("%.1f%%", successRate)})
	
	if len(stats.ResponseTimes) > 0 {
		currentTPS := float64(stats.CompletedCount) / elapsed.Seconds()
		table.Append([]string{"平均TPS", fmt.Sprintf("%.2f", currentTPS)})
	}
	
	table.Render()
	fmt.Println()
}

// ShowError 显示错误信息
func (td *TestDisplayer) ShowError(message string) {
	td.errorColor.Printf("❌ %s\n", message)
}

// printRealTimeStats 打印实时统计信息
func (td *TestDisplayer) printRealTimeStats(stats TestStats) {
	if len(stats.ResponseTimes) == 0 {
		return
	}
	
	// 计算统计数据
	var sum time.Duration
	min := stats.ResponseTimes[0]
	max := stats.ResponseTimes[0]
	
	for _, d := range stats.ResponseTimes {
		sum += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}
	
	avg := sum / time.Duration(len(stats.ResponseTimes))
	currentTPS := float64(stats.CompletedCount) / stats.ElapsedTime.Seconds()
	
	// 显示实时统计
	metricName := "TTFT"
	if !td.config.Stream {
		metricName = "响应时间"
	}
	
	// 创建实时统计显示，使用中性的颜色
	statsLine := fmt.Sprintf("📊 %s | 完成: %d/%d | 失败: %d | %s: 平均 %s, 最小 %s, 最大 %s | TPS: %.2f",
		td.statsColor.Sprint("实时统计"),
		stats.CompletedCount, td.config.Count,
		stats.FailedCount,
		metricName,
		FormatDuration(avg), FormatDuration(min), FormatDuration(max),
		currentTPS)
	
	// 移动到进度条上方显示实时统计，然后回到原位置
	fmt.Printf("\033[A\033[2K%s\n\033[B", statsLine)
}

// Result 性能测试结果
type Result struct {
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration

	// 流式模式指标
	AvgTTFT time.Duration
	MinTTFT time.Duration
	MaxTTFT time.Duration

	// 非流式模式指标
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration

	TPS float64
}

// PrintResult 输出结果
func (r *Result) PrintResult() {
	titleColor := color.New(color.FgCyan, color.Bold)
	titleColor.Println("\n📊 测试结果")
	
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("指标", "最小值", "平均值", "最大值", "单位")
	
	table.Append([]string{"总请求数", "-", fmt.Sprintf("%d", r.TotalRequests), "-", "个"})
	table.Append([]string{"并发数", "-", fmt.Sprintf("%d", r.Concurrency), "-", "个"})
	table.Append([]string{"总耗时", "-", FormatDuration(r.TotalTime), "-", ""})

	if r.IsStream {
		table.Append([]string{"TTFT (首字节时间)",
			FormatDuration(r.MinTTFT),
			FormatDuration(r.AvgTTFT),
			FormatDuration(r.MaxTTFT), ""})
	} else {
		table.Append([]string{"响应时间",
			FormatDuration(r.MinResponseTime),
			FormatDuration(r.AvgResponseTime),
			FormatDuration(r.MaxResponseTime), ""})
	}

	table.Append([]string{"TPS", "-", FormatFloat(r.TPS, 2), "-", "req/s"})
	table.Render()

	// 显示模式提示
	fmt.Println()
	r.printModeInfo()
}

// printModeInfo 打印测试模式信息
func (r *Result) printModeInfo() {
	infoColor := color.New(color.FgBlue)
	
	if r.IsStream {
		infoColor.Println("💡 流式模式：可以准确测量 TTFT（首字节时间）")
	} else {
		infoColor.Println("ℹ️  非流式模式：测量总响应时间")
	}
	
	// 显示一些有用的指标说明
	fmt.Println("\n📖 指标说明：")
	if r.IsStream {
		infoColor.Println("  • TTFT: Time To First Token，首个令牌返回时间")
		infoColor.Println("  • 该指标反映模型开始生成响应的速度")
	} else {
		infoColor.Println("  • 响应时间: 完整请求-响应周期的时间")
		infoColor.Println("  • 该指标反映完整响应的总时间")
	}
	infoColor.Println("  • TPS: Transactions Per Second，每秒处理请求数")
	infoColor.Println("  • 并发数: 同时进行的请求数量")
}
