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
	// 基础统计
	CompletedCount  int                            // 已完成请求数
	FailedCount     int                            // 失败请求数
	
	// 时间指标
	TTFTs           []time.Duration                // 所有首个token响应时间 (Time to First Token)
	TotalTimes      []time.Duration                // 所有总耗时
	
	// 网络指标
	DNSTimes        []time.Duration                // 所有DNS解析时间
	ConnectTimes    []time.Duration                // 所有TCP连接时间
	TLSHandshakeTimes []time.Duration              // 所有TLS握手时间
	
	// 内容指标
	TokenCounts     []int                          // 所有 token 数量
	
	// 错误和可靠性指标
	TimeoutCount    int                            // 超时次数
	RetryCount      int                            // 重试次数
	
	// 测试控制
	StartTime       time.Time                      // 测试开始时间
	ElapsedTime     time.Duration                  // 已经过时间
}

// Result 性能测试结果
type Result struct {
	// 基础测试信息
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration
	TPS           float64

	// 时间性能指标
	TimeMetrics struct {
		AvgTTFT time.Duration // TTFT (Time to First Token) 指标
		MinTTFT time.Duration
		MaxTTFT time.Duration
		
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
	}

	// 内容指标
	ContentMetrics struct {
		AvgTokenCount int // Token 统计指标
		MinTokenCount int
		MaxTokenCount int
		TotalTokens   int
	}

	// 可靠性指标
	ReliabilityMetrics struct {
		ErrorRate    float64 // 错误率百分比
		TimeoutCount int     // 超时次数
		RetryCount   int     // 重试次数
		SuccessRate  float64 // 成功率百分比
	}
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
	var dnsTimes []time.Duration
	var connectTimes []time.Duration
	var tlsHandshakeTimes []time.Duration
	var tokenCounts []int
	var timeoutCount, retryCount int64
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
					CompletedCount:    int(atomic.LoadInt64(&completed)),
					FailedCount:       int(atomic.LoadInt64(&failed)),
					TTFTs:             make([]time.Duration, len(ttfts)),
					TotalTimes:        make([]time.Duration, len(totalTimes)),
					DNSTimes:          make([]time.Duration, len(dnsTimes)),
					ConnectTimes:      make([]time.Duration, len(connectTimes)),
					TLSHandshakeTimes: make([]time.Duration, len(tlsHandshakeTimes)),
					TokenCounts:       make([]int, len(tokenCounts)),
					TimeoutCount:      int(atomic.LoadInt64(&timeoutCount)),
					RetryCount:        int(atomic.LoadInt64(&retryCount)),
					StartTime:         start,
					ElapsedTime:       time.Since(start),
				}
				copy(stats.TTFTs, ttfts)
				copy(stats.TotalTimes, totalTimes)
				copy(stats.DNSTimes, dnsTimes)
				copy(stats.ConnectTimes, connectTimes)
				copy(stats.TLSHandshakeTimes, tlsHandshakeTimes)
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
			dnsTimes = append(dnsTimes, metrics.DNSTime)
			connectTimes = append(connectTimes, metrics.ConnectTime)
			tlsHandshakeTimes = append(tlsHandshakeTimes, metrics.TLSHandshakeTime)
			tokenCounts = append(tokenCounts, metrics.TokenCount)
			ttftsMutex.Unlock()
			
			// 更新错误计数
			if metrics.IsTimeout {
				atomic.AddInt64(&timeoutCount, 1)
			}
			if metrics.IsRetry {
				atomic.AddInt64(&retryCount, 1)
			}
			
			atomic.AddInt64(&completed, 1)
		}(i)
	}
	wg.Wait()
	close(stopProgress)
	elapsed := time.Since(start)

	// 最后一次进度更新
	ttftsMutex.Lock()
	finalStats := TestStats{
		CompletedCount:    int(atomic.LoadInt64(&completed)),
		FailedCount:       int(atomic.LoadInt64(&failed)),
		TTFTs:             make([]time.Duration, len(ttfts)),
		TotalTimes:        make([]time.Duration, len(totalTimes)),
		DNSTimes:          make([]time.Duration, len(dnsTimes)),
		ConnectTimes:      make([]time.Duration, len(connectTimes)),
		TLSHandshakeTimes: make([]time.Duration, len(tlsHandshakeTimes)),
		TokenCounts:       make([]int, len(tokenCounts)),
		TimeoutCount:      int(atomic.LoadInt64(&timeoutCount)),
		RetryCount:        int(atomic.LoadInt64(&retryCount)),
		StartTime:         start,
		ElapsedTime:       elapsed,
	}
	copy(finalStats.TTFTs, ttfts)
	copy(finalStats.TotalTimes, totalTimes)
	copy(finalStats.DNSTimes, dnsTimes)
	copy(finalStats.ConnectTimes, connectTimes)
	copy(finalStats.TLSHandshakeTimes, tlsHandshakeTimes)
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
	
	minDNSTime := firstResult.DNSTime
	maxDNSTime := firstResult.DNSTime
	minConnectTime := firstResult.ConnectTime
	maxConnectTime := firstResult.ConnectTime
	minTLSTime := firstResult.TLSHandshakeTime
	maxTLSTime := firstResult.TLSHandshakeTime

	// 累积统计
	var sumTTFT, sumTotalTime time.Duration
	var sumDNSTime, sumConnectTime, sumTLSTime time.Duration
	var sumTokens int
	var timeoutCount, retryCount int

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

		// 网络指标统计
		sumDNSTime += result.DNSTime
		if result.DNSTime < minDNSTime {
			minDNSTime = result.DNSTime
		}
		if result.DNSTime > maxDNSTime {
			maxDNSTime = result.DNSTime
		}
		
		sumConnectTime += result.ConnectTime
		if result.ConnectTime < minConnectTime {
			minConnectTime = result.ConnectTime
		}
		if result.ConnectTime > maxConnectTime {
			maxConnectTime = result.ConnectTime
		}
		
		sumTLSTime += result.TLSHandshakeTime
		if result.TLSHandshakeTime < minTLSTime {
			minTLSTime = result.TLSHandshakeTime
		}
		if result.TLSHandshakeTime > maxTLSTime {
			maxTLSTime = result.TLSHandshakeTime
		}

		// Token 统计
		sumTokens += result.TokenCount
		if result.TokenCount < minTokens {
			minTokens = result.TokenCount
		}
		if result.TokenCount > maxTokens {
			maxTokens = result.TokenCount
		}
		
		// 错误统计
		if result.IsTimeout {
			timeoutCount++
		}
		if result.IsRetry {
			retryCount++
		}
	}

	validCount := len(validResults)
	avgTTFT := sumTTFT / time.Duration(validCount)
	avgTotalTime := sumTotalTime / time.Duration(validCount)
	avgDNSTime := sumDNSTime / time.Duration(validCount)
	avgConnectTime := sumConnectTime / time.Duration(validCount)
	avgTLSTime := sumTLSTime / time.Duration(validCount)
	avgTokens := sumTokens / validCount
	tps := float64(r.config.Count) / totalTime.Seconds()
	
	// 计算错误率和成功率
	errorRate := float64(r.config.Count-validCount) / float64(r.config.Count) * 100
	successRate := float64(validCount) / float64(r.config.Count) * 100

	result := &Result{
		TotalRequests: r.config.Count,
		Concurrency:   r.config.Concurrency,
		TotalTime:     totalTime,
		IsStream:      r.config.Stream,
		TPS:           tps,
	}
	
	// 时间指标
	result.TimeMetrics.AvgTTFT = avgTTFT
	result.TimeMetrics.MinTTFT = minTTFT
	result.TimeMetrics.MaxTTFT = maxTTFT
	result.TimeMetrics.AvgTotalTime = avgTotalTime
	result.TimeMetrics.MinTotalTime = minTotalTime
	result.TimeMetrics.MaxTotalTime = maxTotalTime
	
	// 网络指标
	result.NetworkMetrics.AvgDNSTime = avgDNSTime
	result.NetworkMetrics.MinDNSTime = minDNSTime
	result.NetworkMetrics.MaxDNSTime = maxDNSTime
	result.NetworkMetrics.AvgConnectTime = avgConnectTime
	result.NetworkMetrics.MinConnectTime = minConnectTime
	result.NetworkMetrics.MaxConnectTime = maxConnectTime
	result.NetworkMetrics.AvgTLSHandshakeTime = avgTLSTime
	result.NetworkMetrics.MinTLSHandshakeTime = minTLSTime
	result.NetworkMetrics.MaxTLSHandshakeTime = maxTLSTime
	
	// 内容指标
	result.ContentMetrics.AvgTokenCount = avgTokens
	result.ContentMetrics.MinTokenCount = minTokens
	result.ContentMetrics.MaxTokenCount = maxTokens
	result.ContentMetrics.TotalTokens = sumTokens
	
	// 可靠性指标
	result.ReliabilityMetrics.ErrorRate = errorRate
	result.ReliabilityMetrics.TimeoutCount = timeoutCount
	result.ReliabilityMetrics.RetryCount = retryCount
	result.ReliabilityMetrics.SuccessRate = successRate

	return result
}
