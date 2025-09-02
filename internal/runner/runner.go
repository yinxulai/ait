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
	CompletedCount  int                            // 已完成请求数
	FailedCount     int                            // 失败请求数
	TTFTs           []time.Duration                // 所有首个token响应时间 (Time to First Token)
	TotalTimes      []time.Duration                // 所有总耗时
	TokenCounts     []int                          // 所有 token 数量
	StartTime       time.Time                      // 测试开始时间
	ElapsedTime     time.Duration                  // 已经过时间
}

// Result 性能测试结果
type Result struct {
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration

	// TTFT (Time to First Token) 指标
	AvgTTFT time.Duration
	MinTTFT time.Duration
	MaxTTFT time.Duration

	// 总耗时指标
	AvgTotalTime time.Duration
	MinTotalTime time.Duration
	MaxTotalTime time.Duration

	// Token 统计指标
	AvgTokenCount int
	MinTokenCount int
	MaxTokenCount int
	TotalTokens   int

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
	results := make([]*client.ResponseMetrics, r.config.Count)
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

			metrics, err := r.client.Request(r.config.Prompt, r.config.Stream)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			results[idx] = metrics
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
	results := make([]*client.ResponseMetrics, r.config.Count)
	start := time.Now()
	ch := make(chan int, r.config.Concurrency)

	completed := int64(0)
	failed := int64(0)
	var ttfts []time.Duration
	var totalTimes []time.Duration
	var tokenCounts []int
	var ttftsMutex sync.Mutex

	// 启动进度更新 goroutine
	stopProgress := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				ttftsMutex.Lock()
				stats := TestStats{
					CompletedCount:  int(atomic.LoadInt64(&completed)),
					FailedCount:     int(atomic.LoadInt64(&failed)),
					TTFTs:           make([]time.Duration, len(ttfts)),
					TotalTimes:      make([]time.Duration, len(totalTimes)),
					TokenCounts:     make([]int, len(tokenCounts)),
					StartTime:       start,
					ElapsedTime:     time.Since(start),
				}
				copy(stats.TTFTs, ttfts)
				copy(stats.TotalTimes, totalTimes)
				copy(stats.TokenCounts, tokenCounts)
				ttftsMutex.Unlock()
				
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

			metrics, err := r.client.Request(r.config.Prompt, r.config.Stream)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}
			
			results[idx] = metrics
			
			ttftsMutex.Lock()
			ttfts = append(ttfts, metrics.TimeToFirstToken)
			totalTimes = append(totalTimes, metrics.TotalTime)
			tokenCounts = append(tokenCounts, metrics.TokenCount)
			ttftsMutex.Unlock()
			
			atomic.AddInt64(&completed, 1)
		}(i)
	}
	wg.Wait()
	close(stopProgress)
	elapsed := time.Since(start)

	// 最后一次进度更新
	ttftsMutex.Lock()
	finalStats := TestStats{
		CompletedCount:  int(atomic.LoadInt64(&completed)),
		FailedCount:     int(atomic.LoadInt64(&failed)),
		TTFTs:           make([]time.Duration, len(ttfts)),
		TotalTimes:      make([]time.Duration, len(totalTimes)),
		TokenCounts:     make([]int, len(tokenCounts)),
		StartTime:       start,
		ElapsedTime:     elapsed,
	}
	copy(finalStats.TTFTs, ttfts)
	copy(finalStats.TotalTimes, totalTimes)
	copy(finalStats.TokenCounts, tokenCounts)
	ttftsMutex.Unlock()
	progressCallback(finalStats)

	// 计算并返回结果
	return r.calculateResult(results, elapsed), nil
}

// calculateResult 计算性能统计结果
func (r *Runner) calculateResult(results []*client.ResponseMetrics, totalTime time.Duration) *Result {
	if len(results) == 0 {
		return &Result{}
	}

	validResults := make([]*client.ResponseMetrics, 0)
	for _, result := range results {
		if result != nil {
			validResults = append(validResults, result)
		}
	}

	if len(validResults) == 0 {
		return &Result{}
	}

	// 初始化最小值和最大值
	firstResult := validResults[0]
	minTTFT := firstResult.TimeToFirstToken
	maxTTFT := firstResult.TimeToFirstToken
	minTotalTime := firstResult.TotalTime
	maxTotalTime := firstResult.TotalTime
	minTokens := firstResult.TokenCount
	maxTokens := firstResult.TokenCount

	// 累积统计
	var sumTTFT, sumTotalTime time.Duration
	var sumTokens int

	for _, result := range validResults {
		// TTFT 统计
		sumTTFT += result.TimeToFirstToken
		if result.TimeToFirstToken < minTTFT {
			minTTFT = result.TimeToFirstToken
		}
		if result.TimeToFirstToken > maxTTFT {
			maxTTFT = result.TimeToFirstToken
		}

		// 总时间统计
		sumTotalTime += result.TotalTime
		if result.TotalTime < minTotalTime {
			minTotalTime = result.TotalTime
		}
		if result.TotalTime > maxTotalTime {
			maxTotalTime = result.TotalTime
		}

		// Token 统计
		sumTokens += result.TokenCount
		if result.TokenCount < minTokens {
			minTokens = result.TokenCount
		}
		if result.TokenCount > maxTokens {
			maxTokens = result.TokenCount
		}
	}

	validCount := len(validResults)
	avgTTFT := sumTTFT / time.Duration(validCount)
	avgTotalTime := sumTotalTime / time.Duration(validCount)
	avgTokens := sumTokens / validCount
	tps := float64(r.config.Count) / totalTime.Seconds()

	result := &Result{
		TotalRequests: r.config.Count,
		Concurrency:   r.config.Concurrency,
		TotalTime:     totalTime,
		IsStream:      r.config.Stream,
		TPS:           tps,
		AvgTTFT:       avgTTFT,
		MinTTFT:       minTTFT,
		MaxTTFT:       maxTTFT,
		AvgTotalTime:  avgTotalTime,
		MinTotalTime:  minTotalTime,
		MaxTotalTime:  maxTotalTime,
		AvgTokenCount: avgTokens,
		MinTokenCount: minTokens,
		MaxTokenCount: maxTokens,
		TotalTokens:   sumTokens,
	}

	return result
}
