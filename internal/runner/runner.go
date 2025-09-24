package runner

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/logger"
	"github.com/yinxulai/ait/internal/types"
	"github.com/yinxulai/ait/internal/upload"
)

// Runner 性能测试执行器
type Runner struct {
	taskID string
	input  types.Input
	upload *upload.Uploader
	client client.ModelClient
}

// NewRunner 创建新的性能测试执行器
func NewRunner(taskID string, config types.Input) (*Runner, error) {
	// 创建日志记录器（如果启用）
	var loggerInstance *logger.Logger
	if config.Log {
		loggerInstance = logger.New(config.Log)
	}

	client, err := client.NewClientWithTimeout(config.Protocol, config.BaseUrl, config.ApiKey, config.Model, config.Timeout, loggerInstance)
	if err != nil {
		return nil, err
	}
	
	return &Runner{
		taskID: taskID,
		client: client,
		input:  config,
		upload: upload.New(),
	}, nil
}

// Run 执行性能测试，返回结果数据
func (r *Runner) Run() (*types.ReportData, error) {
	var wg sync.WaitGroup
	results := make([]*client.ResponseMetrics, r.input.Count)
	start := time.Now()
	ch := make(chan int, r.input.Concurrency)

	completed := int64(0)
	failed := int64(0)

	for i := 0; i < r.input.Count; i++ {
		wg.Add(1)
		ch <- 1
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			// 获取当前请求使用的prompt
			currentPrompt := r.input.PromptSource.GetRandomContent()
			
			metrics, err := r.client.Request(currentPrompt, r.input.Stream)
			if err != nil {
				atomic.AddInt64(&failed, 1)
				return
			}

			results[idx] = metrics

			if metrics.ErrorMessage == "" && r.upload != nil {
				r.upload.UploadReport(r.taskID, metrics, r.input)
			}

			atomic.AddInt64(&completed, 1)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	// 计算并返回结果
	return r.calculateResult(results, elapsed), nil
}

// RunWithProgress 运行性能测试并实时显示进度
func (r *Runner) RunWithProgress(progressCallback func(types.StatsData)) (*types.ReportData, error) {
	var wg sync.WaitGroup
	results := make([]*client.ResponseMetrics, r.input.Count)
	start := time.Now()
	ch := make(chan int, r.input.Concurrency)

	completed := int64(0)
	failed := int64(0)
	var ttfts []time.Duration
	var totalTimes []time.Duration
	var dnsTimes []time.Duration
	var connectTimes []time.Duration
	var tlsHandshakeTimes []time.Duration
	var tokenCounts []int
	var errorMessages []string
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
				stats := types.StatsData{
					CompletedCount:    int(atomic.LoadInt64(&completed)),
					FailedCount:       int(atomic.LoadInt64(&failed)),
					TTFTs:             make([]time.Duration, len(ttfts)),
					TotalTimes:        make([]time.Duration, len(totalTimes)),
					DNSTimes:          make([]time.Duration, len(dnsTimes)),
					ConnectTimes:      make([]time.Duration, len(connectTimes)),
					TLSHandshakeTimes: make([]time.Duration, len(tlsHandshakeTimes)),
					TokenCounts:       make([]int, len(tokenCounts)),
					ErrorMessages:     make([]string, len(errorMessages)),
					StartTime:         start,
					ElapsedTime:       time.Since(start),
				}
				copy(stats.TTFTs, ttfts)
				copy(stats.TotalTimes, totalTimes)
				copy(stats.DNSTimes, dnsTimes)
				copy(stats.ConnectTimes, connectTimes)
				copy(stats.TLSHandshakeTimes, tlsHandshakeTimes)
				copy(stats.TokenCounts, tokenCounts)
				copy(stats.ErrorMessages, errorMessages)
				ttftsMutex.Unlock()

				progressCallback(stats)
			case <-stopProgress:
				return
			}
		}
	}()

	for i := 0; i < r.input.Count; i++ {
		wg.Add(1)
		ch <- 1
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			// 获取当前请求使用的prompt
			currentPrompt := r.input.PromptSource.GetRandomContent()
			
			metrics, err := r.client.Request(currentPrompt, r.input.Stream)
			if err != nil {
				ttftsMutex.Lock()
				errorMessages = append(errorMessages, err.Error())
				ttftsMutex.Unlock()
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
			tokenCounts = append(tokenCounts, metrics.CompletionTokens)
			ttftsMutex.Unlock()

			if metrics.ErrorMessage == "" && r.upload != nil {
				r.upload.UploadReport(r.taskID, metrics, r.input)
			}
			
			atomic.AddInt64(&completed, 1)
		}(i)
	}
	wg.Wait()
	close(stopProgress)
	elapsed := time.Since(start)

	// 最后一次进度更新
	ttftsMutex.Lock()
	finalStats := types.StatsData{
		CompletedCount:    int(atomic.LoadInt64(&completed)),
		FailedCount:       int(atomic.LoadInt64(&failed)),
		TTFTs:             make([]time.Duration, len(ttfts)),
		TotalTimes:        make([]time.Duration, len(totalTimes)),
		DNSTimes:          make([]time.Duration, len(dnsTimes)),
		ConnectTimes:      make([]time.Duration, len(connectTimes)),
		TLSHandshakeTimes: make([]time.Duration, len(tlsHandshakeTimes)),
		TokenCounts:       make([]int, len(tokenCounts)),
		ErrorMessages:     make([]string, len(errorMessages)),
		StartTime:         start,
		ElapsedTime:       elapsed,
	}
	copy(finalStats.TTFTs, ttfts)
	copy(finalStats.TotalTimes, totalTimes)
	copy(finalStats.DNSTimes, dnsTimes)
	copy(finalStats.ConnectTimes, connectTimes)
	copy(finalStats.TLSHandshakeTimes, tlsHandshakeTimes)
	copy(finalStats.TokenCounts, tokenCounts)
	copy(finalStats.ErrorMessages, errorMessages)
	ttftsMutex.Unlock()
	progressCallback(finalStats)

	// 计算并返回结果
	return r.calculateResult(results, elapsed), nil
}

// calculateResult 计算性能统计结果
func (r *Runner) calculateResult(results []*client.ResponseMetrics, totalTime time.Duration) *types.ReportData {
	if len(results) == 0 {
		return &types.ReportData{}
	}

	validResults := make([]*client.ResponseMetrics, 0)
	for _, result := range results {
		if result != nil && result.CompletionTokens > 0 {
			validResults = append(validResults, result)
		}
	}

	if len(validResults) == 0 {
		return &types.ReportData{}
	}

	// 初始化最小值和最大值
	firstResult := validResults[0]
	minTTFT := firstResult.TimeToFirstToken
	maxTTFT := firstResult.TimeToFirstToken
	minTotalTime := firstResult.TotalTime
	maxTotalTime := firstResult.TotalTime
	minTokens := firstResult.CompletionTokens
	maxTokens := firstResult.CompletionTokens

	minDNSTime := firstResult.DNSTime
	maxDNSTime := firstResult.DNSTime
	minConnectTime := firstResult.ConnectTime
	maxConnectTime := firstResult.ConnectTime
	minTLSTime := firstResult.TLSHandshakeTime
	maxTLSTime := firstResult.TLSHandshakeTime

	// 计算第一个结果的 TPS 和 TPOT
	var firstTPS float64
	if firstResult.TotalTime.Seconds() > 0 {
		firstTPS = float64(firstResult.CompletionTokens) / firstResult.TotalTime.Seconds()
	}
	minTPS := firstTPS
	maxTPS := firstTPS

	// 计算第一个结果的 TPOT (Time Per Output Token)
	var firstTPOT time.Duration
	if firstResult.CompletionTokens > 1 {
		// TPOT = (总耗时 - 首token耗时) / (总token数 - 1)
		remainingTime := firstResult.TotalTime - firstResult.TimeToFirstToken
		firstTPOT = remainingTime / time.Duration(firstResult.CompletionTokens-1)
	}
	minTPOT := firstTPOT
	maxTPOT := firstTPOT

	// 获取目标IP（使用第一个有效结果的IP）
	var targetIP string
	for _, result := range validResults {
		if result.TargetIP != "" {
			targetIP = result.TargetIP
			break
		}
	}

	// 累积统计
	var sumTTFT, sumTotalTime time.Duration
	var sumDNSTime, sumConnectTime, sumTLSTime time.Duration
	var sumTokens int
	var sumTPOT time.Duration

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

		// TPOT 统计
		var tpot time.Duration
		if result.CompletionTokens > 1 {
			// TPOT = (总耗时 - 首token耗时) / (总token数 - 1)
			remainingTime := result.TotalTime - result.TimeToFirstToken
			tpot = remainingTime / time.Duration(result.CompletionTokens-1)
			sumTPOT += tpot

			if tpot < minTPOT || minTPOT == 0 {
				minTPOT = tpot
			}
			if tpot > maxTPOT {
				maxTPOT = tpot
			}
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
		sumTokens += result.CompletionTokens
		if result.CompletionTokens < minTokens {
			minTokens = result.CompletionTokens
		}
		if result.CompletionTokens > maxTokens {
			maxTokens = result.CompletionTokens
		}

		// TPS 统计
		var tps float64
		if result.TotalTime.Seconds() > 0 {
			tps = float64(result.CompletionTokens) / result.TotalTime.Seconds()
		}
		if tps < minTPS {
			minTPS = tps
		}
		if tps > maxTPS {
			maxTPS = tps
		}
	}

	validCount := len(validResults)

	// 计算错误率和成功率
	errorRate := float64(r.input.Count-validCount) / float64(r.input.Count) * 100
	successRate := float64(validCount) / float64(r.input.Count) * 100

	// 如果没有有效结果，返回基础结果
	if validCount == 0 {
		result := &types.ReportData{
			TotalRequests: r.input.Count,
			Concurrency:   r.input.Concurrency,
			TotalTime:     totalTime,
			IsStream:      r.input.Stream,
		}

		// 可靠性指标
		result.ReliabilityMetrics.ErrorRate = errorRate
		result.ReliabilityMetrics.SuccessRate = successRate

		return result
	}

	// 计算各项指标的平均值
	// 注意：时间指标可以直接用总和除以数量来计算平均值，因为时间是可加性的
	avgTTFT := sumTTFT / time.Duration(validCount)
	avgTotalTime := sumTotalTime / time.Duration(validCount)
	avgDNSTime := sumDNSTime / time.Duration(validCount)
	avgConnectTime := sumConnectTime / time.Duration(validCount)
	avgTLSTime := sumTLSTime / time.Duration(validCount)

	// 计算TPOT平均值 - 只对有效的TPOT计算结果求平均
	var avgTPOT time.Duration
	validTPOTCount := 0
	for _, result := range validResults {
		if result.CompletionTokens > 1 {
			validTPOTCount++
		}
	}
	if validTPOTCount > 0 {
		avgTPOT = sumTPOT / time.Duration(validTPOTCount)
	}

	// Token数量也可以直接用总和除以数量，因为数量是可加性的
	avgTokens := sumTokens / validCount

	// TPS是比率指标，需要特殊处理：
	// 错误方式：float64(sumTokens) / sumTotalTime.Seconds() - 这相当于计算总体批处理的TPS
	// 正确方式：先计算每个请求的TPS，然后求算术平均值 - 这反映单个请求的平均性能
	var sumTPS float64
	for _, result := range validResults {
		if result.TotalTime.Seconds() > 0 {
			tps := float64(result.CompletionTokens) / result.TotalTime.Seconds()
			sumTPS += tps
		}
	}
	avgTPS := sumTPS / float64(validCount)

	result := &types.ReportData{
		TotalRequests: r.input.Count,
		Concurrency:   r.input.Concurrency,
		TotalTime:     totalTime,
		IsStream:      r.input.Stream,
	}

	// 时间指标
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
	result.NetworkMetrics.TargetIP = targetIP

	// 服务性能指标
	result.ContentMetrics.AvgTTFT = avgTTFT
	result.ContentMetrics.MinTTFT = minTTFT
	result.ContentMetrics.MaxTTFT = maxTTFT
	result.ContentMetrics.AvgTPOT = avgTPOT
	result.ContentMetrics.MinTPOT = minTPOT
	result.ContentMetrics.MaxTPOT = maxTPOT
	result.ContentMetrics.AvgTokenCount = avgTokens
	result.ContentMetrics.MinTokenCount = minTokens
	result.ContentMetrics.MaxTokenCount = maxTokens
	result.ContentMetrics.AvgTPS = avgTPS
	result.ContentMetrics.MinTPS = minTPS
	result.ContentMetrics.MaxTPS = maxTPS

	// 可靠性指标
	result.ReliabilityMetrics.ErrorRate = errorRate
	result.ReliabilityMetrics.SuccessRate = successRate

	return result
}
