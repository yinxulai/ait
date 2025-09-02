package runner

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/yinxulai/ait/internal/client"
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

// TestStats 实时测试统计数据
type TestStats struct {
	CompletedCount int           // 已完成请求数
	FailedCount    int           // 失败请求数
	ResponseTimes  []time.Duration // 所有响应时间
	StartTime      time.Time     // 测试开始时间
	ElapsedTime    time.Duration // 已经过时间
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

// Run 执行性能测试，返回结果数据
func (r *Runner) Run() (*Result, error) {
	var wg sync.WaitGroup
	results := make([]time.Duration, r.config.Count)
	start := time.Now()
	ch := make(chan int, r.config.Concurrency)

	completed := int64(0)
	failed := int64(0)

	for i := 0; i < r.config.Count; i++ {
		wg.Add(1)
		ch <- 1
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			ttft, err := r.client.Request(r.config.Prompt, r.config.Stream)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			results[idx] = ttft
			atomic.AddInt64(&completed, 1)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	// 计算并返回结果
	return r.calculateResult(results, elapsed), nil
}

// RunWithProgress 执行性能测试，通过回调函数提供进度更新
func (r *Runner) RunWithProgress(progressCallback func(TestStats)) (*Result, error) {
	var wg sync.WaitGroup
	results := make([]time.Duration, r.config.Count)
	start := time.Now()
	ch := make(chan int, r.config.Concurrency)

	completed := int64(0)
	failed := int64(0)
	var responseTimes []time.Duration
	var responseTimesMutex sync.Mutex

	// 启动进度更新 goroutine
	stopProgress := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				responseTimesMutex.Lock()
				stats := TestStats{
					CompletedCount: int(atomic.LoadInt64(&completed)),
					FailedCount:    int(atomic.LoadInt64(&failed)),
					ResponseTimes:  make([]time.Duration, len(responseTimes)),
					StartTime:      start,
					ElapsedTime:    time.Since(start),
				}
				copy(stats.ResponseTimes, responseTimes)
				responseTimesMutex.Unlock()
				
				progressCallback(stats)
			case <-stopProgress:
				return
			}
		}
	}()

	for i := 0; i < r.config.Count; i++ {
		wg.Add(1)
		ch <- 1
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			ttft, err := r.client.Request(r.config.Prompt, r.config.Stream)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			
			results[idx] = ttft
			
			responseTimesMutex.Lock()
			responseTimes = append(responseTimes, ttft)
			responseTimesMutex.Unlock()
			
			atomic.AddInt64(&completed, 1)
		}(i)
	}
	wg.Wait()
	close(stopProgress)
	elapsed := time.Since(start)

	// 最后一次进度更新
	responseTimesMutex.Lock()
	finalStats := TestStats{
		CompletedCount: int(atomic.LoadInt64(&completed)),
		FailedCount:    int(atomic.LoadInt64(&failed)),
		ResponseTimes:  make([]time.Duration, len(responseTimes)),
		StartTime:      start,
		ElapsedTime:    elapsed,
	}
	copy(finalStats.ResponseTimes, responseTimes)
	responseTimesMutex.Unlock()
	progressCallback(finalStats)

	// 计算并返回结果
	return r.calculateResult(results, elapsed), nil
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
