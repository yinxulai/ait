package display

import (
	"fmt"
	"time"
)

// TestDisplayer 测试显示控制器
type TestDisplayer struct {
	config     TestConfig
	progressBar *ProgressBar
	startTime   time.Time
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
		config: config,
	}
}

// ShowTestStart 显示测试开始信息
func (td *TestDisplayer) ShowTestStart() {
	PrintTitle("AI 模型性能测试")
	PrintInfo(fmt.Sprintf("Provider: %s", td.config.Provider))
	PrintInfo(fmt.Sprintf("BaseURL: %s", td.config.BaseUrl))
	
	// 隐藏 API Key 中间部分
	apiKeyDisplay := td.config.ApiKey
	if len(apiKeyDisplay) > 8 {
		start := apiKeyDisplay[:4]
		end := apiKeyDisplay[len(apiKeyDisplay)-4:]
		apiKeyDisplay = start + "**" + end
	}
	PrintInfo(fmt.Sprintf("ApiKey: %s", apiKeyDisplay))
	
	PrintInfo(fmt.Sprintf("Model: %s", td.config.Model))
	PrintInfo(fmt.Sprintf("并发数: %d", td.config.Concurrency))
	PrintInfo(fmt.Sprintf("总请求数: %d", td.config.Count))
	PrintInfo(fmt.Sprintf("流模式: %t", td.config.Stream))
	
	fmt.Println() // 为实时统计预留一行空间
	
	// 创建进度条
	td.progressBar = NewProgressBar(td.config.Count, "执行测试")
	td.startTime = time.Now()
}

// UpdateProgress 更新测试进度
func (td *TestDisplayer) UpdateProgress(stats TestStats) {
	// 更新进度条
	if td.progressBar != nil {
		td.progressBar.Update(stats.CompletedCount)
	}
	
	// 更新实时统计
	td.printRealTimeStats(stats)
}

// ShowTestComplete 显示测试完成
func (td *TestDisplayer) ShowTestComplete() {
	if td.progressBar != nil {
		td.progressBar.Finish()
	}
	PrintSuccess("测试完成！")
}

// ShowError 显示错误信息
func (td *TestDisplayer) ShowError(message string) {
	PrintError(message)
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
	
	// 移动到进度条上方显示实时统计
	fmt.Printf("\033[A\033[2K📊 实时统计 | 完成: %d/%d | 失败: %d | %s: 平均 %s, 最小 %s, 最大 %s | TPS: %.2f\n\033[B",
		stats.CompletedCount, td.config.Count, stats.FailedCount,
		metricName, FormatDuration(avg), FormatDuration(min), FormatDuration(max),
		currentTPS)
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
	PrintSection("测试结果")

	// 创建结果表格
	table := NewTable([]string{"指标", "最小值", "平均值", "最大值", "单位"})
	table.AddRow([]string{"总请求数", "-", fmt.Sprintf("%d", r.TotalRequests), "-", "个"})
	table.AddRow([]string{"并发数", "-", fmt.Sprintf("%d", r.Concurrency), "-", "个"})
	table.AddRow([]string{"总耗时", "-", FormatDuration(r.TotalTime), "-", ""})

	if r.IsStream {
		table.AddRow([]string{"TTFT (首字节时间)",
			FormatDuration(r.MinTTFT),
			FormatDuration(r.AvgTTFT),
			FormatDuration(r.MaxTTFT), ""})
	} else {
		table.AddRow([]string{"响应时间",
			FormatDuration(r.MinResponseTime),
			FormatDuration(r.AvgResponseTime),
			FormatDuration(r.MaxResponseTime), ""})
	}

	table.AddRow([]string{"TPS", "-", FormatFloat(r.TPS, 2), "-", "req/s"})

	table.Render()

	// 性能评级
	fmt.Println()
	r.printPerformanceRating()
}

// printPerformanceRating 打印性能评级
func (r *Result) printPerformanceRating() {
	PrintSection("性能评级")

	var avgMs float64
	var metricName string

	if r.IsStream {
		avgMs = float64(r.AvgTTFT.Nanoseconds()) / 1000000
		metricName = "TTFT"
	} else {
		avgMs = float64(r.AvgResponseTime.Nanoseconds()) / 1000000
		metricName = "响应时间"
	}

	var rating string
	var color string

	switch {
	case avgMs < 100:
		rating = "优秀 (< 100ms)"
		color = ColorGreen
	case avgMs < 300:
		rating = "良好 (100-300ms)"
		color = ColorYellow
	case avgMs < 1000:
		rating = "一般 (300ms-1s)"
		color = ColorYellow
	case avgMs < 3000:
		rating = "较慢 (1-3s)"
		color = ColorRed
	default:
		rating = "很慢 (> 3s)"
		color = ColorRed
	}

	fmt.Printf("%s%s: %s%s%s\n", color, metricName, ColorBold, rating, ColorReset)

	if r.TPS > 10 {
		PrintSuccess(fmt.Sprintf("吞吐量优秀: %.2f req/s", r.TPS))
	} else if r.TPS > 5 {
		PrintWarning(fmt.Sprintf("吞吐量良好: %.2f req/s", r.TPS))
	} else if r.TPS > 1 {
		PrintWarning(fmt.Sprintf("吞吐量一般: %.2f req/s", r.TPS))
	} else {
		PrintError(fmt.Sprintf("吞吐量较低: %.2f req/s", r.TPS))
	}

	if r.IsStream {
		PrintInfo("💡 流式模式可以准确测量 TTFT（首字节时间）")
	} else {
		PrintWarning("⚠️ 非流式模式只能测量总响应时间，无法获得 TTFT 指标")
	}
}
