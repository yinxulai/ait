package benchmark

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/display"
)

// Config 性能测试配置
type Config struct {
	Provider    string
	BaseUrl     string
	ApiKey      string
	Model       string
	Concurrency int
	Count       int
	Prompt      string
	Stream      bool // 是否使用流式请求
}

// Result 性能测试结果
type Result struct {
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration

	// 流式模式指标
	AvgTTFT       time.Duration
	MinTTFT       time.Duration
	MaxTTFT       time.Duration

	// 非流式模式指标
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration

	TPS             float64
}

// Runner 性能测试执行器
type Runner struct {
	client client.ModelClient
	config Config
}

// NewRunner 创建新的性能测试执行器
func NewRunner(config Config) (*Runner, error) {
	client, err := client.NewClient(config.Provider, config.BaseUrl, config.ApiKey, config.Model)
	if err != nil {
		return nil, err
	}
	return &Runner{
		client: client,
		config: config,
	}, nil
}

// Run 执行性能测试
func (r *Runner) Run() (*Result, error) {
	display.PrintTitle("AI 模型性能测试")
	display.PrintInfo(fmt.Sprintf("Provider: %s", r.config.Provider))
	display.PrintInfo(fmt.Sprintf("Model: %s", r.config.Model))
	display.PrintInfo(fmt.Sprintf("并发数: %d", r.config.Concurrency))
	display.PrintInfo(fmt.Sprintf("总请求数: %d", r.config.Count))
	display.PrintInfo(fmt.Sprintf("流模式: %t", r.config.Stream))
	
	var wg sync.WaitGroup
	results := make([]time.Duration, r.config.Count)
	start := time.Now()
	ch := make(chan int, r.config.Concurrency)
	
	// 创建进度条
	progressBar := display.NewProgressBar(r.config.Count, "执行测试")
	completed := int64(0)

	for i := 0; i < r.config.Count; i++ {
		wg.Add(1)
		ch <- 1
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			ttft, err := r.client.Request(r.config.Prompt, r.config.Stream)
			if err != nil {
				display.PrintError(fmt.Sprintf("请求失败 [%d]: %v", idx, err))
				return
			}
			results[idx] = ttft
			
			// 更新进度条
			current := atomic.AddInt64(&completed, 1)
			progressBar.Update(int(current))
		}(i)
	}
	wg.Wait()
	progressBar.Finish()
	elapsed := time.Since(start)

	display.PrintSuccess("测试完成！")
	
	// 统计结果
	result := r.calculateResult(results, elapsed)
	result.PrintResult()
	return result, nil
}

// calculateResult 计算性能统计结果
func (r *Runner) calculateResult(results []time.Duration, totalTime time.Duration) *Result {
	if len(results) == 0 {
		return &Result{}
	}

	var sum time.Duration
	min := results[0]
	max := results[0]
	validCount := 0

	for _, d := range results {
		if d > 0 { // 只统计成功的请求
			sum += d
			validCount++
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}
	}

	if validCount == 0 {
		return &Result{}
	}

	avg := sum / time.Duration(validCount)
	tps := float64(r.config.Count) / totalTime.Seconds()

	result := &Result{
		TotalRequests: r.config.Count,
		Concurrency:   r.config.Concurrency,
		TotalTime:     totalTime,
		IsStream:      r.config.Stream,
		TPS:           tps,
	}

	if r.config.Stream {
		result.AvgTTFT = avg
		result.MinTTFT = min
		result.MaxTTFT = max
	} else {
		result.AvgResponseTime = avg
		result.MinResponseTime = min
		result.MaxResponseTime = max
	}

	return result
}

// PrintResult 输出结果
func (r *Result) PrintResult() {
	display.PrintSection("测试结果")
	
	// 创建结果表格
	table := display.NewTable([]string{"指标", "最小值", "平均值", "最大值", "单位"})
	table.AddRow([]string{"总请求数", "-", fmt.Sprintf("%d", r.TotalRequests), "-", "个"})
	table.AddRow([]string{"并发数", "-", fmt.Sprintf("%d", r.Concurrency), "-", "个"})
	table.AddRow([]string{"总耗时", "-", display.FormatDuration(r.TotalTime), "-", ""})
	
	if r.IsStream {
		table.AddRow([]string{"TTFT (首字节时间)", 
			display.FormatDuration(r.MinTTFT), 
			display.FormatDuration(r.AvgTTFT), 
			display.FormatDuration(r.MaxTTFT), ""})
	} else {
		table.AddRow([]string{"响应时间", 
			display.FormatDuration(r.MinResponseTime), 
			display.FormatDuration(r.AvgResponseTime), 
			display.FormatDuration(r.MaxResponseTime), ""})
	}
	
	table.AddRow([]string{"TPS", "-", display.FormatFloat(r.TPS, 2), "-", "req/s"})
	
	table.Render()
	
	// 性能评级
	fmt.Println()
	r.printPerformanceRating()
}

// printPerformanceRating 打印性能评级
func (r *Result) printPerformanceRating() {
	display.PrintSection("性能评级")
	
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
		color = display.ColorGreen
	case avgMs < 300:
		rating = "良好 (100-300ms)"
		color = display.ColorYellow
	case avgMs < 1000:
		rating = "一般 (300ms-1s)"
		color = display.ColorYellow
	case avgMs < 3000:
		rating = "较慢 (1-3s)"
		color = display.ColorRed
	default:
		rating = "很慢 (> 3s)"
		color = display.ColorRed
	}
	
	fmt.Printf("%s%s: %s%s%s\n", color, metricName, display.ColorBold, rating, display.ColorReset)
	
	if r.TPS > 10 {
		display.PrintSuccess(fmt.Sprintf("吞吐量优秀: %.2f req/s", r.TPS))
	} else if r.TPS > 5 {
		display.PrintWarning(fmt.Sprintf("吞吐量良好: %.2f req/s", r.TPS))
	} else if r.TPS > 1 {
		display.PrintWarning(fmt.Sprintf("吞吐量一般: %.2f req/s", r.TPS))
	} else {
		display.PrintError(fmt.Sprintf("吞吐量较低: %.2f req/s", r.TPS))
	}
	
	if r.IsStream {
		display.PrintInfo("💡 流式模式可以准确测量 TTFT（首字节时间）")
	} else {
		display.PrintWarning("⚠️ 非流式模式只能测量总响应时间，无法获得 TTFT 指标")
	}
}
