package runner

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/logger"
	"github.com/yinxulai/ait/internal/server/types"
	"github.com/yinxulai/ait/internal/server/upload"
)

// Runner 性能测试执行器
type Runner struct {
	taskID   string
	input    types.Input
	upload   *upload.Uploader
	client   client.ModelClient
	stopCh   chan struct{}
	stopOnce sync.Once
}

type RequestDoneCallback func(metrics *client.ResponseMetrics, index int, err error)

// NewRunner 创建新的性能测试执行器
func NewRunner(taskID string, config types.Input) (*Runner, error) {
	// 创建日志记录器（如果启用）
	var loggerInstance *logger.Logger
	if config.Log {
		loggerInstance = logger.New(config.Log)
	}

	client, err := client.NewClient(config, loggerInstance)
	if err != nil {
		return nil, err
	}

	return &Runner{
		taskID: taskID,
		client: client,
		input:  config,
		upload: upload.New(),
		stopCh: make(chan struct{}),
	}, nil
}

func (r *Runner) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

func (r *Runner) acquireSlot(ch chan int) bool {
	select {
	case <-r.stopCh:
		return false
	case ch <- 1:
		return true
	}
}

func calculateCacheHitRate(metrics *client.ResponseMetrics) float64 {
	if metrics == nil || metrics.CachedInputTokens <= 0 {
		return 0
	}
	if metrics.PromptTokens <= 0 {
		return 0
	}
	return float64(metrics.CachedInputTokens) / float64(metrics.PromptTokens)
}

// Run 执行性能测试，返回结果数据
func (r *Runner) Run() (*types.ReportData, error) {
	var wg sync.WaitGroup
	results := make([]*client.ResponseMetrics, r.input.Count)
	start := time.Now()
	ch := make(chan int, r.input.Concurrency)
	launchedCount := 0

	for i := 0; i < r.input.Count; i++ {
		if !r.acquireSlot(ch) {
			break
		}
		launchedCount++
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			// 获取当前请求使用的prompt
			var metrics *client.ResponseMetrics
			var err error
			if r.input.PromptMode == "raw" {
				rawBody := r.input.PromptSource.GetContentByIndex(idx)
				metrics, err = r.client.RawRequest(rawBody)
			} else {
				systemPrompt := r.input.PromptSource.GetSystemContent()
				userPrompt := r.input.PromptSource.GetContentByIndex(idx)
				metrics, err = r.client.Request(systemPrompt, userPrompt, r.input.Stream)
			}
			if err != nil {
				// 即使有错误，也尝试保存 metrics（如果有的话）
				if metrics != nil {
					results[idx] = metrics
				}
				return
			}

			results[idx] = metrics

			if metrics.ErrorMessage == "" && r.upload != nil {
				r.upload.UploadReport(r.taskID, metrics, r.input)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)

	// 计算并返回结果
	return r.calculateResult(results, elapsed, launchedCount), nil
}

func (r *Runner) RunWithCallback(cb RequestDoneCallback) (*types.ReportData, error) {
	var wg sync.WaitGroup
	results := make([]*client.ResponseMetrics, r.input.Count)
	start := time.Now()
	ch := make(chan int, r.input.Concurrency)
	launchedCount := 0

	for i := 0; i < r.input.Count; i++ {
		if !r.acquireSlot(ch) {
			break
		}
		launchedCount++
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			var metrics *client.ResponseMetrics
			var err error
			if r.input.PromptMode == "raw" {
				rawBody := r.input.PromptSource.GetContentByIndex(idx)
				metrics, err = r.client.RawRequest(rawBody)
			} else {
				systemPrompt := r.input.PromptSource.GetSystemContent()
				userPrompt := r.input.PromptSource.GetContentByIndex(idx)
				metrics, err = r.client.Request(systemPrompt, userPrompt, r.input.Stream)
			}
			if metrics != nil {
				results[idx] = metrics
			}

			if err == nil && metrics != nil && metrics.ErrorMessage == "" && r.upload != nil {
				r.upload.UploadReport(r.taskID, metrics, r.input)
			}

			if cb != nil {
				cb(metrics, idx, err)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	return r.calculateResult(results, elapsed, launchedCount), nil
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
	var outputTokenCounts []int
	var inputTokenCounts []int
	var cachedInputTokenCounts []int
	var thinkingTokenCounts []int
	var cacheHitRates []float64
	var errorMessages []string
	var ttftsMutex sync.Mutex
	launchedCount := 0

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
					CompletedCount:         int(atomic.LoadInt64(&completed)),
					FailedCount:            int(atomic.LoadInt64(&failed)),
					TTFTs:                  make([]time.Duration, len(ttfts)),
					TotalTimes:             make([]time.Duration, len(totalTimes)),
					DNSTimes:               make([]time.Duration, len(dnsTimes)),
					ConnectTimes:           make([]time.Duration, len(connectTimes)),
					TLSHandshakeTimes:      make([]time.Duration, len(tlsHandshakeTimes)),
					InputTokenCounts:       make([]int, len(inputTokenCounts)),
					CachedInputTokenCounts: make([]int, len(cachedInputTokenCounts)),
					OutputTokenCounts:      make([]int, len(outputTokenCounts)),
					ThinkingTokenCounts:    make([]int, len(thinkingTokenCounts)),
					CacheHitRates:          make([]float64, len(cacheHitRates)),
					ErrorMessages:          make([]string, len(errorMessages)),
					StartTime:              start,
					ElapsedTime:            time.Since(start),
				}
				copy(stats.TTFTs, ttfts)
				copy(stats.TotalTimes, totalTimes)
				copy(stats.DNSTimes, dnsTimes)
				copy(stats.ConnectTimes, connectTimes)
				copy(stats.TLSHandshakeTimes, tlsHandshakeTimes)
				copy(stats.InputTokenCounts, inputTokenCounts)
				copy(stats.CachedInputTokenCounts, cachedInputTokenCounts)
				copy(stats.OutputTokenCounts, outputTokenCounts)
				copy(stats.ThinkingTokenCounts, thinkingTokenCounts)
				copy(stats.CacheHitRates, cacheHitRates)
				copy(stats.ErrorMessages, errorMessages)
				ttftsMutex.Unlock()

				progressCallback(stats)
			case <-stopProgress:
				return
			}
		}
	}()

	for i := 0; i < r.input.Count; i++ {
		if !r.acquireSlot(ch) {
			break
		}
		launchedCount++
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() { <-ch }()

			// 获取当前请求使用的prompt
			var metrics *client.ResponseMetrics
			var err error
			if r.input.PromptMode == "raw" {
				rawBody := r.input.PromptSource.GetContentByIndex(idx)
				metrics, err = r.client.RawRequest(rawBody)
			} else {
				systemPrompt := r.input.PromptSource.GetSystemContent()
				userPrompt := r.input.PromptSource.GetContentByIndex(idx)
				metrics, err = r.client.Request(systemPrompt, userPrompt, r.input.Stream)
			}
			if err != nil {
				ttftsMutex.Lock()
				errorMessages = append(errorMessages, err.Error())
				ttftsMutex.Unlock()
				atomic.AddInt64(&failed, 1)
				// 即使有错误，也尝试保存 metrics（如果有的话）
				if metrics != nil {
					results[idx] = metrics
					// 仍然收集网络性能指标，即使请求失败
					ttftsMutex.Lock()
					ttfts = append(ttfts, metrics.TimeToFirstToken)
					totalTimes = append(totalTimes, metrics.TotalTime)
					dnsTimes = append(dnsTimes, metrics.DNSTime)
					connectTimes = append(connectTimes, metrics.ConnectTime)
					tlsHandshakeTimes = append(tlsHandshakeTimes, metrics.TLSHandshakeTime)
					outputTokenCounts = append(outputTokenCounts, metrics.CompletionTokens)
					inputTokenCounts = append(inputTokenCounts, metrics.PromptTokens)
					cachedInputTokenCounts = append(cachedInputTokenCounts, metrics.CachedInputTokens)
					thinkingTokenCounts = append(thinkingTokenCounts, metrics.ThinkingTokens)
					cacheHitRates = append(cacheHitRates, calculateCacheHitRate(metrics))
					ttftsMutex.Unlock()
				}
				return
			}

			results[idx] = metrics

			ttftsMutex.Lock()
			ttfts = append(ttfts, metrics.TimeToFirstToken)
			totalTimes = append(totalTimes, metrics.TotalTime)
			dnsTimes = append(dnsTimes, metrics.DNSTime)
			connectTimes = append(connectTimes, metrics.ConnectTime)
			tlsHandshakeTimes = append(tlsHandshakeTimes, metrics.TLSHandshakeTime)
			outputTokenCounts = append(outputTokenCounts, metrics.CompletionTokens)
			inputTokenCounts = append(inputTokenCounts, metrics.PromptTokens)
			cachedInputTokenCounts = append(cachedInputTokenCounts, metrics.CachedInputTokens)
			thinkingTokenCounts = append(thinkingTokenCounts, metrics.ThinkingTokens)
			cacheHitRates = append(cacheHitRates, calculateCacheHitRate(metrics))
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
		CompletedCount:         int(atomic.LoadInt64(&completed)),
		FailedCount:            int(atomic.LoadInt64(&failed)),
		TTFTs:                  make([]time.Duration, len(ttfts)),
		TotalTimes:             make([]time.Duration, len(totalTimes)),
		DNSTimes:               make([]time.Duration, len(dnsTimes)),
		ConnectTimes:           make([]time.Duration, len(connectTimes)),
		TLSHandshakeTimes:      make([]time.Duration, len(tlsHandshakeTimes)),
		InputTokenCounts:       make([]int, len(inputTokenCounts)),
		CachedInputTokenCounts: make([]int, len(cachedInputTokenCounts)),
		OutputTokenCounts:      make([]int, len(outputTokenCounts)),
		ThinkingTokenCounts:    make([]int, len(thinkingTokenCounts)),
		CacheHitRates:          make([]float64, len(cacheHitRates)),
		ErrorMessages:          make([]string, len(errorMessages)),
		StartTime:              start,
		ElapsedTime:            elapsed,
	}
	copy(finalStats.TTFTs, ttfts)
	copy(finalStats.TotalTimes, totalTimes)
	copy(finalStats.DNSTimes, dnsTimes)
	copy(finalStats.ConnectTimes, connectTimes)
	copy(finalStats.TLSHandshakeTimes, tlsHandshakeTimes)
	copy(finalStats.InputTokenCounts, inputTokenCounts)
	copy(finalStats.CachedInputTokenCounts, cachedInputTokenCounts)
	copy(finalStats.OutputTokenCounts, outputTokenCounts)
	copy(finalStats.ThinkingTokenCounts, thinkingTokenCounts)
	copy(finalStats.CacheHitRates, cacheHitRates)
	copy(finalStats.ErrorMessages, errorMessages)
	ttftsMutex.Unlock()
	progressCallback(finalStats)

	// 计算并返回结果
	return r.calculateResult(results, elapsed, launchedCount), nil
}

// calculateResult 计算性能统计结果
func (r *Runner) calculateResult(results []*client.ResponseMetrics, totalTime time.Duration, totalRequests ...int) *types.ReportData {
	requestCount := r.input.Count
	if len(totalRequests) > 0 {
		requestCount = totalRequests[0]
	}
	if requestCount <= 0 || len(results) == 0 {
		return &types.ReportData{}
	}

	allResults := make([]*client.ResponseMetrics, 0)
	successResults := make([]*client.ResponseMetrics, 0)
	for _, result := range results {
		if result == nil {
			continue
		}
		allResults = append(allResults, result)
		if result.ErrorMessage == "" && result.CompletionTokens > 0 {
			successResults = append(successResults, result)
		}
	}
	if len(allResults) == 0 {
		return &types.ReportData{}
	}

	validResults := successResults
	if len(validResults) == 0 {
		for _, result := range allResults {
			if result.TotalTime > 0 {
				validResults = append(validResults, result)
			}
		}
		if len(validResults) == 0 {
			return &types.ReportData{}
		}
	}

	firstResult := validResults[0]
	minTTFT := firstResult.TimeToFirstToken
	maxTTFT := firstResult.TimeToFirstToken
	minTotalTime := firstResult.TotalTime
	maxTotalTime := firstResult.TotalTime
	minOutputTokens := firstResult.CompletionTokens
	maxOutputTokens := firstResult.CompletionTokens
	minInputTokens := firstResult.PromptTokens
	maxInputTokens := firstResult.PromptTokens
	minCachedInputTokens := firstResult.CachedInputTokens
	maxCachedInputTokens := firstResult.CachedInputTokens
	minThinkingTokens := firstResult.ThinkingTokens
	maxThinkingTokens := firstResult.ThinkingTokens
	minCacheHitRate := calculateCacheHitRate(firstResult)
	maxCacheHitRate := minCacheHitRate

	minDNSTime := firstResult.DNSTime
	maxDNSTime := firstResult.DNSTime
	minConnectTime := firstResult.ConnectTime
	maxConnectTime := firstResult.ConnectTime
	minTLSTime := firstResult.TLSHandshakeTime
	maxTLSTime := firstResult.TLSHandshakeTime

	var firstTPS float64
	if firstResult.TotalTime.Seconds() > 0 {
		firstTPS = float64(firstResult.CompletionTokens) / firstResult.TotalTime.Seconds()
	}
	minTPS := firstTPS
	maxTPS := firstTPS

	var firstTotalThroughputTPS float64
	if firstResult.TotalTime.Seconds() > 0 {
		totalTokens := firstResult.PromptTokens + firstResult.CompletionTokens
		firstTotalThroughputTPS = float64(totalTokens) / firstResult.TotalTime.Seconds()
	}
	minTotalThroughputTPS := firstTotalThroughputTPS
	maxTotalThroughputTPS := firstTotalThroughputTPS

	var firstTPOT time.Duration
	if firstResult.CompletionTokens > 1 {
		remainingTime := firstResult.TotalTime - firstResult.TimeToFirstToken
		firstTPOT = remainingTime / time.Duration(firstResult.CompletionTokens-1)
	}
	minTPOT := firstTPOT
	maxTPOT := firstTPOT

	var targetIP string
	for _, result := range validResults {
		if result.TargetIP != "" {
			targetIP = result.TargetIP
			break
		}
	}

	var sumTTFT, sumTotalTime time.Duration
	var sumDNSTime, sumConnectTime, sumTLSTime time.Duration
	var sumOutputTokens, sumInputTokens, sumCachedInputTokens int
	var sumThinkingTokens int
	var sumTPOT time.Duration
	var sumCacheHitRate, sumTotalThroughputTPS float64

	for _, result := range validResults {
		sumTTFT += result.TimeToFirstToken
		if result.TimeToFirstToken < minTTFT {
			minTTFT = result.TimeToFirstToken
		}
		if result.TimeToFirstToken > maxTTFT {
			maxTTFT = result.TimeToFirstToken
		}

		sumTotalTime += result.TotalTime
		if result.TotalTime < minTotalTime {
			minTotalTime = result.TotalTime
		}
		if result.TotalTime > maxTotalTime {
			maxTotalTime = result.TotalTime
		}

		var tpot time.Duration
		if result.CompletionTokens > 1 {
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

		sumOutputTokens += result.CompletionTokens
		if result.CompletionTokens < minOutputTokens {
			minOutputTokens = result.CompletionTokens
		}
		if result.CompletionTokens > maxOutputTokens {
			maxOutputTokens = result.CompletionTokens
		}

		sumInputTokens += result.PromptTokens
		if result.PromptTokens < minInputTokens {
			minInputTokens = result.PromptTokens
		}
		if result.PromptTokens > maxInputTokens {
			maxInputTokens = result.PromptTokens
		}

		sumCachedInputTokens += result.CachedInputTokens
		if result.CachedInputTokens < minCachedInputTokens {
			minCachedInputTokens = result.CachedInputTokens
		}
		if result.CachedInputTokens > maxCachedInputTokens {
			maxCachedInputTokens = result.CachedInputTokens
		}

		sumThinkingTokens += result.ThinkingTokens
		if result.ThinkingTokens < minThinkingTokens {
			minThinkingTokens = result.ThinkingTokens
		}
		if result.ThinkingTokens > maxThinkingTokens {
			maxThinkingTokens = result.ThinkingTokens
		}

		cacheHitRate := calculateCacheHitRate(result)
		sumCacheHitRate += cacheHitRate
		if cacheHitRate < minCacheHitRate {
			minCacheHitRate = cacheHitRate
		}
		if cacheHitRate > maxCacheHitRate {
			maxCacheHitRate = cacheHitRate
		}

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

		var totalThroughputTPS float64
		if result.TotalTime.Seconds() > 0 {
			totalTokens := result.PromptTokens + result.CompletionTokens
			totalThroughputTPS = float64(totalTokens) / result.TotalTime.Seconds()
			sumTotalThroughputTPS += totalThroughputTPS
		}
		if totalThroughputTPS < minTotalThroughputTPS {
			minTotalThroughputTPS = totalThroughputTPS
		}
		if totalThroughputTPS > maxTotalThroughputTPS {
			maxTotalThroughputTPS = totalThroughputTPS
		}
	}

	// successCount 基于真正成功的请求（有输出 token 且无错误）
	// validCount 可能是 successCount 的 fallback 集，仅用于计算平均指标，不参与成功率
	successCount := len(successResults)
	validCount := len(validResults)
	errorRate := float64(requestCount-successCount) / float64(requestCount) * 100
	successRate := float64(successCount) / float64(requestCount) * 100
	resolvedEndpoint := r.input.ResolvedEndpointURL()

	if validCount == 0 {
		return &types.ReportData{
			TotalRequests: requestCount,
			Concurrency:   r.input.Concurrency,
			TotalTime:     totalTime,
			IsStream:      r.input.Stream,
			IsThinking:    r.input.Thinking,
			Protocol:      r.input.NormalizedProtocol(),
			EndpointURL:   resolvedEndpoint,
			BaseUrl:       resolvedEndpoint,
			ErrorRate:     errorRate,
			SuccessRate:   successRate,
		}
	}

	avgTTFT := sumTTFT / time.Duration(validCount)
	avgTotalTime := sumTotalTime / time.Duration(validCount)
	avgDNSTime := sumDNSTime / time.Duration(validCount)
	avgConnectTime := sumConnectTime / time.Duration(validCount)
	avgTLSTime := sumTLSTime / time.Duration(validCount)

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

	avgOutputTokens := sumOutputTokens / validCount
	avgInputTokens := sumInputTokens / validCount
	avgCachedInputTokens := sumCachedInputTokens / validCount
	avgThinkingTokens := sumThinkingTokens / validCount
	avgCacheHitRate := sumCacheHitRate / float64(validCount)

	var sumTPS float64
	for _, result := range validResults {
		if result.TotalTime.Seconds() > 0 {
			sumTPS += float64(result.CompletionTokens) / result.TotalTime.Seconds()
		}
	}
	avgTPS := sumTPS / float64(validCount)
	avgTotalThroughputTPS := sumTotalThroughputTPS / float64(validCount)

	var varianceSumTotalTime, varianceSumTTFT, varianceSumTPOT float64
	var varianceSumInputTokens, varianceSumCachedInputTokens, varianceSumOutputTokens, varianceSumThinkingTokens float64
	var varianceSumCacheHitRate, varianceSumTPS, varianceSumTotalThroughputTPS float64

	for _, result := range validResults {
		diffTotalTime := float64(result.TotalTime - avgTotalTime)
		varianceSumTotalTime += diffTotalTime * diffTotalTime

		diffTTFT := float64(result.TimeToFirstToken - avgTTFT)
		varianceSumTTFT += diffTTFT * diffTTFT

		diffInputTokens := float64(result.PromptTokens - avgInputTokens)
		varianceSumInputTokens += diffInputTokens * diffInputTokens

		diffCachedInputTokens := float64(result.CachedInputTokens - avgCachedInputTokens)
		varianceSumCachedInputTokens += diffCachedInputTokens * diffCachedInputTokens

		diffOutputTokens := float64(result.CompletionTokens - avgOutputTokens)
		varianceSumOutputTokens += diffOutputTokens * diffOutputTokens

		diffThinkingTokens := float64(result.ThinkingTokens - avgThinkingTokens)
		varianceSumThinkingTokens += diffThinkingTokens * diffThinkingTokens

		diffCacheHitRate := calculateCacheHitRate(result) - avgCacheHitRate
		varianceSumCacheHitRate += diffCacheHitRate * diffCacheHitRate

		var tps float64
		if result.TotalTime.Seconds() > 0 {
			tps = float64(result.CompletionTokens) / result.TotalTime.Seconds()
		}
		diffTPS := tps - avgTPS
		varianceSumTPS += diffTPS * diffTPS

		var totalThroughputTPS float64
		if result.TotalTime.Seconds() > 0 {
			totalTokens := result.PromptTokens + result.CompletionTokens
			totalThroughputTPS = float64(totalTokens) / result.TotalTime.Seconds()
		}
		diffTotalThroughputTPS := totalThroughputTPS - avgTotalThroughputTPS
		varianceSumTotalThroughputTPS += diffTotalThroughputTPS * diffTotalThroughputTPS
	}

	for _, result := range validResults {
		if result.CompletionTokens > 1 {
			remainingTime := result.TotalTime - result.TimeToFirstToken
			tpot := remainingTime / time.Duration(result.CompletionTokens-1)
			diffTPOT := float64(tpot - avgTPOT)
			varianceSumTPOT += diffTPOT * diffTPOT
		}
	}

	stdDevTotalTime := time.Duration(math.Sqrt(varianceSumTotalTime / float64(validCount)))
	stdDevTTFT := time.Duration(math.Sqrt(varianceSumTTFT / float64(validCount)))
	stdDevTPOT := time.Duration(0)
	if validTPOTCount > 0 {
		stdDevTPOT = time.Duration(math.Sqrt(varianceSumTPOT / float64(validTPOTCount)))
	}
	stdDevInputTokenCount := math.Sqrt(varianceSumInputTokens / float64(validCount))
	stdDevCachedInputTokenCount := math.Sqrt(varianceSumCachedInputTokens / float64(validCount))
	stdDevOutputTokenCount := math.Sqrt(varianceSumOutputTokens / float64(validCount))
	stdDevThinkingTokenCount := math.Sqrt(varianceSumThinkingTokens / float64(validCount))
	stdDevCacheHitRate := math.Sqrt(varianceSumCacheHitRate / float64(validCount))
	stdDevTPS := math.Sqrt(varianceSumTPS / float64(validCount))
	stdDevTotalThroughputTPS := math.Sqrt(varianceSumTotalThroughputTPS / float64(validCount))

	var rpm, tpm float64
	if totalTime.Minutes() > 0 {
		rpm = float64(successCount) / totalTime.Minutes()
		tpm = float64(sumOutputTokens) / totalTime.Minutes()
	}

	return &types.ReportData{
		TotalRequests:               requestCount,
		Concurrency:                 r.input.Concurrency,
		TotalTime:                   totalTime,
		IsStream:                    r.input.Stream,
		IsThinking:                  r.input.Thinking,
		Protocol:                    r.input.NormalizedProtocol(),
		EndpointURL:                 resolvedEndpoint,
		BaseUrl:                     resolvedEndpoint,
		AvgTotalTime:                avgTotalTime,
		MinTotalTime:                minTotalTime,
		MaxTotalTime:                maxTotalTime,
		AvgDNSTime:                  avgDNSTime,
		MinDNSTime:                  minDNSTime,
		MaxDNSTime:                  maxDNSTime,
		AvgConnectTime:              avgConnectTime,
		MinConnectTime:              minConnectTime,
		MaxConnectTime:              maxConnectTime,
		AvgTLSHandshakeTime:         avgTLSTime,
		MinTLSHandshakeTime:         minTLSTime,
		MaxTLSHandshakeTime:         maxTLSTime,
		TargetIP:                    targetIP,
		AvgTTFT:                     avgTTFT,
		MinTTFT:                     minTTFT,
		MaxTTFT:                     maxTTFT,
		AvgTPOT:                     avgTPOT,
		MinTPOT:                     minTPOT,
		MaxTPOT:                     maxTPOT,
		AvgInputTokenCount:          avgInputTokens,
		MinInputTokenCount:          minInputTokens,
		MaxInputTokenCount:          maxInputTokens,
		AvgCachedInputTokenCount:    avgCachedInputTokens,
		MinCachedInputTokenCount:    minCachedInputTokens,
		MaxCachedInputTokenCount:    maxCachedInputTokens,
		AvgOutputTokenCount:         avgOutputTokens,
		MinOutputTokenCount:         minOutputTokens,
		MaxOutputTokenCount:         maxOutputTokens,
		AvgThinkingTokenCount:       avgThinkingTokens,
		MinThinkingTokenCount:       minThinkingTokens,
		MaxThinkingTokenCount:       maxThinkingTokens,
		AvgCacheHitRate:             avgCacheHitRate,
		MinCacheHitRate:             minCacheHitRate,
		MaxCacheHitRate:             maxCacheHitRate,
		AvgTPS:                      avgTPS,
		MinTPS:                      minTPS,
		MaxTPS:                      maxTPS,
		AvgTotalThroughputTPS:       avgTotalThroughputTPS,
		MinTotalThroughputTPS:       minTotalThroughputTPS,
		MaxTotalThroughputTPS:       maxTotalThroughputTPS,
		RPM:                         rpm,
		TPM:                         tpm,
		StdDevTotalTime:             stdDevTotalTime,
		StdDevTTFT:                  stdDevTTFT,
		StdDevTPOT:                  stdDevTPOT,
		StdDevInputTokenCount:       stdDevInputTokenCount,
		StdDevCachedInputTokenCount: stdDevCachedInputTokenCount,
		StdDevOutputTokenCount:      stdDevOutputTokenCount,
		StdDevThinkingTokenCount:    stdDevThinkingTokenCount,
		StdDevCacheHitRate:          stdDevCacheHitRate,
		StdDevTPS:                   stdDevTPS,
		StdDevTotalThroughputTPS:    stdDevTotalThroughputTPS,
		ErrorRate:                   errorRate,
		SuccessRate:                 successRate,
	}
}
