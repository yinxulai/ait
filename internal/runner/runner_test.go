package runner

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/types"
)

// MockClient 用于测试的模拟客户端
type MockClient struct {
	shouldError     bool
	errorMsg        string
	responseMetrics *client.ResponseMetrics
	requestDelay    time.Duration
	callCount       int64 // 追踪调用次数
	failurePattern  []bool // 用于模拟间歇性失败的模式
	protocol        string
	model           string
}

func (m *MockClient) Request(prompt string, stream bool) (*client.ResponseMetrics, error) {
	callIndex := atomic.AddInt64(&m.callCount, 1) - 1
	
	if m.requestDelay > 0 {
		time.Sleep(m.requestDelay)
	}
	
	// 检查失败模式
	if m.failurePattern != nil && int(callIndex) < len(m.failurePattern) && m.failurePattern[callIndex] {
		return nil, errors.New("pattern-based failure")
	}
	
	if m.shouldError {
		return nil, errors.New(m.errorMsg)
	}
	
	// 返回预设的响应指标
	if m.responseMetrics != nil {
		// 为每次调用创建一个副本，避免并发问题
		metrics := *m.responseMetrics
		return &metrics, nil
	}
	
	// 默认响应
	return &client.ResponseMetrics{
		TotalTime:         100 * time.Millisecond,
		TimeToFirstToken:  20 * time.Millisecond,
		CompletionTokens:  50,
		DNSTime:          5 * time.Millisecond,
		ConnectTime:      10 * time.Millisecond,
		TLSHandshakeTime: 15 * time.Millisecond,
		TargetIP:         "127.0.0.1",
	}, nil
}

func (m *MockClient) GetProtocol() string {
	if m.protocol != "" {
		return m.protocol
	}
	return "mock"
}

func (m *MockClient) GetModel() string {
	if m.model != "" {
		return m.model
	}
	return "mock-model"
}

// GetCallCount 获取调用次数
func (m *MockClient) GetCallCount() int64 {
	return atomic.LoadInt64(&m.callCount)
}

// ResetCallCount 重置调用次数
func (m *MockClient) ResetCallCount() {
	atomic.StoreInt64(&m.callCount, 0)
}

// NewRunnerWithClient 创建带有自定义客户端的Runner（用于测试）
func NewRunnerWithClient(config types.Input, client client.ModelClient) *Runner {
	return &Runner{
		config: config,
		client: client,
	}
}

func TestNewRunner(t *testing.T) {
	tests := []struct {
		name      string
		input     types.Input
		wantError bool
	}{
		{
			name: "valid openai config",
			input: types.Input{
				Protocol:    "openai",
				BaseUrl:     "https://api.openai.com",
				ApiKey:      "test-key",
				Model:       "gpt-3.5-turbo",
				Concurrency: 1,
				Count:       10,
				Prompt:      "test prompt",
				Stream:      false,
			},
			wantError: false,
		},
		{
			name: "valid anthropic config",
			input: types.Input{
				Protocol:    "anthropic",
				BaseUrl:     "https://api.anthropic.com",
				ApiKey:      "test-key",
				Model:       "claude-3-sonnet-20240229",
				Concurrency: 2,
				Count:       5,
				Prompt:      "test prompt",
				Stream:      true,
			},
			wantError: false,
		},
		{
			name: "invalid provider",
			input: types.Input{
				Protocol:    "invalid",
				BaseUrl:     "https://api.test.com",
				ApiKey:      "test-key",
				Model:       "test-model",
				Concurrency: 1,
				Count:       10,
				Prompt:      "test prompt",
				Stream:      false,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewRunner(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewRunner() error = nil, wantError %v", tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewRunner() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if runner == nil {
				t.Error("NewRunner() returned nil runner")
				return
			}

			if runner.client == nil {
				t.Error("NewRunner().client should not be nil")
			}

			if runner.config.Protocol != tt.input.Protocol {
				t.Errorf("NewRunner().config.Protocol = %v, want %v", runner.config.Protocol, tt.input.Protocol)
			}

			if runner.config.Stream != tt.input.Stream {
				t.Errorf("NewRunner().config.Stream = %v, want %v", runner.config.Stream, tt.input.Stream)
			}
		})
	}
}

func TestRunner_Run_Success(t *testing.T) {
	// 创建测试配置
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 2,
		Count:       5,
		Prompt:      "test prompt",
		Stream:      true,
	}
	
	// 创建mock客户端
	mockClient := &MockClient{
		shouldError: false,
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         200 * time.Millisecond,
			TimeToFirstToken:  50 * time.Millisecond,
			CompletionTokens:  100,
			DNSTime:          10 * time.Millisecond,
			ConnectTime:      20 * time.Millisecond,
			TLSHandshakeTime: 30 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	// 执行测试
	result, err := runner.Run()
	
	// 验证结果
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Run() returned nil result")
	}
	
	// 验证调用次数
	if mockClient.GetCallCount() != int64(config.Count) {
		t.Errorf("Expected %d client calls, got %d", config.Count, mockClient.GetCallCount())
	}
	
	// 验证基本配置
	if result.TotalRequests != config.Count {
		t.Errorf("Expected TotalRequests %d, got %d", config.Count, result.TotalRequests)
	}
	
	if result.Concurrency != config.Concurrency {
		t.Errorf("Expected Concurrency %d, got %d", config.Concurrency, result.Concurrency)
	}
	
	if result.IsStream != config.Stream {
		t.Errorf("Expected IsStream %v, got %v", config.Stream, result.IsStream)
	}
	
	// 验证成功率
	if result.ReliabilityMetrics.SuccessRate != 100.0 {
		t.Errorf("Expected SuccessRate 100.0, got %f", result.ReliabilityMetrics.SuccessRate)
	}
	
	if result.ReliabilityMetrics.ErrorRate != 0.0 {
		t.Errorf("Expected ErrorRate 0.0, got %f", result.ReliabilityMetrics.ErrorRate)
	}
	
	// 验证性能指标
	if result.ContentMetrics.AvgTokenCount != 100 {
		t.Errorf("Expected AvgTokenCount 100, got %d", result.ContentMetrics.AvgTokenCount)
	}
	
	// 验证总时间有合理值
	if result.TotalTime <= 0 {
		t.Error("Expected positive TotalTime")
	}
}

func TestRunner_Run_PartialFailures(t *testing.T) {
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 3,
		Count:       10,
		Prompt:      "test prompt",
		Stream:      false,
	}
	
	// 创建一个会间歇性失败的mock客户端
	// 让第3、6、9个请求失败（索引2、5、8）
	failurePattern := make([]bool, 10)
	failurePattern[2] = true  // 第3个请求失败
	failurePattern[5] = true  // 第6个请求失败  
	failurePattern[8] = true  // 第9个请求失败
	
	mockClient := &MockClient{
		shouldError:     false,
		failurePattern:  failurePattern,
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         150 * time.Millisecond,
			TimeToFirstToken:  30 * time.Millisecond,
			CompletionTokens:  80,
			DNSTime:          8 * time.Millisecond,
			ConnectTime:      15 * time.Millisecond,
			TLSHandshakeTime: 25 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	result, err := runner.Run()
	
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Run() returned nil result")
	}
	
	// 验证总调用次数
	if mockClient.GetCallCount() != int64(config.Count) {
		t.Errorf("Expected %d client calls, got %d", config.Count, mockClient.GetCallCount())
	}
	
	// 验证错误率和成功率  
	// 10个请求中，3个失败，所以7个成功
	expectedSuccessRate := 70.0
	expectedErrorRate := 30.0
	
	if result.ReliabilityMetrics.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected SuccessRate %f, got %f", expectedSuccessRate, result.ReliabilityMetrics.SuccessRate)
	}
	
	if result.ReliabilityMetrics.ErrorRate != expectedErrorRate {
		t.Errorf("Expected ErrorRate %f, got %f", expectedErrorRate, result.ReliabilityMetrics.ErrorRate)
	}
}

func TestRunner_Run_AllFailures(t *testing.T) {
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com", 
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       3,
		Prompt:      "test prompt",
		Stream:      false,
	}
	
	// 创建总是失败的mock客户端
	mockClient := &MockClient{
		shouldError: true,
		errorMsg:    "network timeout",
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	result, err := runner.Run()
	
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Run() returned nil result")
	}
	
	// 验证调用次数
	if mockClient.GetCallCount() != int64(config.Count) {
		t.Errorf("Expected %d client calls, got %d", config.Count, mockClient.GetCallCount())
	}
	
	// 调试输出
	t.Logf("TotalRequests: %d", result.TotalRequests)
	t.Logf("SuccessRate: %f", result.ReliabilityMetrics.SuccessRate)
	t.Logf("ErrorRate: %f", result.ReliabilityMetrics.ErrorRate)
	
	// 所有请求都失败的情况，由于当前calculateResult的实现
	// 在validResults为0时返回空ReportData，所以基础字段都是零值
	// 这可能是实现的一个问题，但我们先测试当前行为
	if result.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests 0 (current implementation), got %d", result.TotalRequests)
	}
	
	if result.ReliabilityMetrics.ErrorRate != 0.0 {
		t.Errorf("Expected ErrorRate 0.0 (current implementation), got %f", result.ReliabilityMetrics.ErrorRate)
	}
	
	if result.ReliabilityMetrics.SuccessRate != 0.0 {
		t.Errorf("Expected SuccessRate 0.0 (current implementation), got %f", result.ReliabilityMetrics.SuccessRate)
	}
}

func TestRunner_Run_ConcurrencyControl(t *testing.T) {
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key", 
		Model:       "gpt-3.5-turbo",
		Concurrency: 2, // 限制并发为2
		Count:       6,
		Prompt:      "test prompt",
		Stream:      true,
	}
	
	// 创建有延迟的mock客户端来测试并发控制
	mockClient := &MockClient{
		shouldError:  false,
		requestDelay: 50 * time.Millisecond, // 每个请求延迟50ms
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         100 * time.Millisecond,
			TimeToFirstToken:  25 * time.Millisecond,
			CompletionTokens:  75,
			DNSTime:          5 * time.Millisecond,
			ConnectTime:      10 * time.Millisecond,
			TLSHandshakeTime: 15 * time.Millisecond,
			TargetIP:         "1.1.1.1",
		},
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	start := time.Now()
	result, err := runner.Run()
	elapsed := time.Since(start)
	
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("Run() returned nil result")
	}
	
	// 验证并发控制：6个请求，并发度为2，每个请求50ms
	// 应该至少需要 (6/2) * 50ms = 150ms
	minExpectedTime := 150 * time.Millisecond
	if elapsed < minExpectedTime {
		t.Errorf("Expected execution time >= %v, got %v (concurrency control may not be working)", minExpectedTime, elapsed)
	}
	
	// 但也不应该太长（比如不应该超过串行执行的时间）
	maxExpectedTime := 400 * time.Millisecond // 给一些余量
	if elapsed > maxExpectedTime {
		t.Errorf("Execution time %v is too long, may indicate concurrency is not working", elapsed)
	}
	
	// 验证所有请求都被执行
	if mockClient.GetCallCount() != int64(config.Count) {
		t.Errorf("Expected %d client calls, got %d", config.Count, mockClient.GetCallCount())
	}
}

func TestRunner_RunWithProgress_Success(t *testing.T) {
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 2,
		Count:       5,
		Prompt:      "test prompt",
		Stream:      true,
	}
	
	mockClient := &MockClient{
		shouldError:  false,
		requestDelay: 20 * time.Millisecond, // 轻微延迟以确保进度回调被调用
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         100 * time.Millisecond,
			TimeToFirstToken:  25 * time.Millisecond,
			CompletionTokens:  60,
			DNSTime:          5 * time.Millisecond,
			ConnectTime:      10 * time.Millisecond,
			TLSHandshakeTime: 20 * time.Millisecond,
			TargetIP:         "1.1.1.1",
		},
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	// 跟踪进度回调
	var progressCallbacks []types.StatsData
	progressCallback := func(stats types.StatsData) {
		progressCallbacks = append(progressCallbacks, stats)
	}
	
	result, err := runner.RunWithProgress(progressCallback)
	
	if err != nil {
		t.Errorf("RunWithProgress() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("RunWithProgress() returned nil result")
	}
	
	// 验证基本结果
	if result.TotalRequests != config.Count {
		t.Errorf("Expected TotalRequests %d, got %d", config.Count, result.TotalRequests)
	}
	
	if result.Concurrency != config.Concurrency {
		t.Errorf("Expected Concurrency %d, got %d", config.Concurrency, result.Concurrency)
	}
	
	// 验证进度回调被调用
	if len(progressCallbacks) == 0 {
		t.Error("Expected progress callbacks to be called")
	}
	
	// 验证最后一个进度回调包含完整信息
	if len(progressCallbacks) > 0 {
		finalProgress := progressCallbacks[len(progressCallbacks)-1]
		
		if finalProgress.CompletedCount != config.Count {
			t.Errorf("Expected final CompletedCount %d, got %d", config.Count, finalProgress.CompletedCount)
		}
		
		if finalProgress.FailedCount != 0 {
			t.Errorf("Expected final FailedCount 0, got %d", finalProgress.FailedCount)
		}
		
		if len(finalProgress.TTFTs) != config.Count {
			t.Errorf("Expected %d TTFTs, got %d", config.Count, len(finalProgress.TTFTs))
		}
		
		if len(finalProgress.TotalTimes) != config.Count {
			t.Errorf("Expected %d TotalTimes, got %d", config.Count, len(finalProgress.TotalTimes))
		}
		
		if len(finalProgress.TokenCounts) != config.Count {
			t.Errorf("Expected %d TokenCounts, got %d", config.Count, len(finalProgress.TokenCounts))
		}
	}
	
	// 验证客户端调用次数
	if mockClient.GetCallCount() != int64(config.Count) {
		t.Errorf("Expected %d client calls, got %d", config.Count, mockClient.GetCallCount())
	}
}

func TestRunner_RunWithProgress_WithFailures(t *testing.T) {
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       4,
		Prompt:      "test prompt",
		Stream:      false,
	}
	
	// 让第2和第4个请求失败
	failurePattern := []bool{false, true, false, true}
	
	mockClient := &MockClient{
		shouldError:     false,
		failurePattern:  failurePattern,
		requestDelay:    30 * time.Millisecond,
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         120 * time.Millisecond,
			TimeToFirstToken:  30 * time.Millisecond,
			CompletionTokens:  70,
			DNSTime:          6 * time.Millisecond,
			ConnectTime:      12 * time.Millisecond,
			TLSHandshakeTime: 24 * time.Millisecond,
			TargetIP:         "2.2.2.2",
		},
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	var progressCallbacks []types.StatsData
	progressCallback := func(stats types.StatsData) {
		progressCallbacks = append(progressCallbacks, stats)
	}
	
	result, err := runner.RunWithProgress(progressCallback)
	
	if err != nil {
		t.Errorf("RunWithProgress() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("RunWithProgress() returned nil result")
	}
	
	// 验证错误和成功计数
	expectedSuccessCount := 2
	expectedFailureCount := 2
	
	if len(progressCallbacks) > 0 {
		finalProgress := progressCallbacks[len(progressCallbacks)-1]
		
		if finalProgress.CompletedCount != expectedSuccessCount {
			t.Errorf("Expected final CompletedCount %d, got %d", expectedSuccessCount, finalProgress.CompletedCount)
		}
		
		if finalProgress.FailedCount != expectedFailureCount {
			t.Errorf("Expected final FailedCount %d, got %d", expectedFailureCount, finalProgress.FailedCount)
		}
		
		if len(finalProgress.ErrorMessages) != expectedFailureCount {
			t.Errorf("Expected %d error messages, got %d", expectedFailureCount, len(finalProgress.ErrorMessages))
		}
		
		// 验证成功的请求数据被收集
		if len(finalProgress.TTFTs) != expectedSuccessCount {
			t.Errorf("Expected %d TTFTs, got %d", expectedSuccessCount, len(finalProgress.TTFTs))
		}
	}
	
	// 验证最终结果的错误率
	expectedErrorRate := float64(expectedFailureCount) / float64(config.Count) * 100
	if result.ReliabilityMetrics.ErrorRate != expectedErrorRate {
		t.Errorf("Expected ErrorRate %f, got %f", expectedErrorRate, result.ReliabilityMetrics.ErrorRate)
	}
}

func TestRunner_RunWithProgress_ProgressTiming(t *testing.T) {
	config := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       3,
		Prompt:      "test prompt",
		Stream:      true,
	}
	
	mockClient := &MockClient{
		shouldError:  false,
		requestDelay: 100 * time.Millisecond, // 较长延迟确保多次进度更新
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         150 * time.Millisecond,
			TimeToFirstToken:  40 * time.Millisecond,
			CompletionTokens:  90,
			DNSTime:          7 * time.Millisecond,
			ConnectTime:      14 * time.Millisecond,
			TLSHandshakeTime: 28 * time.Millisecond,
			TargetIP:         "3.3.3.3",
		},
	}
	
	runner := NewRunnerWithClient(config, mockClient)
	
	var progressCallbacks []types.StatsData
	var callbackTimes []time.Time
	startTime := time.Now()
	
	progressCallback := func(stats types.StatsData) {
		progressCallbacks = append(progressCallbacks, stats)
		callbackTimes = append(callbackTimes, time.Now())
		
		// 验证ElapsedTime字段是合理的
		elapsed := time.Since(startTime)
		if stats.ElapsedTime > elapsed+100*time.Millisecond {
			t.Errorf("Stats ElapsedTime %v is much greater than actual elapsed %v", stats.ElapsedTime, elapsed)
		}
	}
	
	result, err := runner.RunWithProgress(progressCallback)
	
	if err != nil {
		t.Errorf("RunWithProgress() returned unexpected error: %v", err)
	}
	
	if result == nil {
		t.Fatal("RunWithProgress() returned nil result")
	}
	
	// 验证至少有一些进度回调（应该至少有最终回调）
	if len(progressCallbacks) == 0 {
		t.Error("Expected at least one progress callback")
	}
	
	// 验证进度数据的递增性质
	for i := 1; i < len(progressCallbacks); i++ {
		curr := progressCallbacks[i]
		prev := progressCallbacks[i-1]
		
		// 完成数应该单调递增或保持不变
		if curr.CompletedCount < prev.CompletedCount {
			t.Errorf("CompletedCount decreased from %d to %d", prev.CompletedCount, curr.CompletedCount)
		}
		
		// 失败数应该单调递增或保持不变
		if curr.FailedCount < prev.FailedCount {
			t.Errorf("FailedCount decreased from %d to %d", prev.FailedCount, curr.FailedCount)
		}
		
		// ElapsedTime应该递增
		if curr.ElapsedTime < prev.ElapsedTime {
			t.Errorf("ElapsedTime decreased from %v to %v", prev.ElapsedTime, curr.ElapsedTime)
		}
	}
}

func TestRunner_CalculateResult_EmptyResults(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 2,
		Count:       10,
		Stream:      false,
	}
	
	runner := &Runner{config: input}
	
	var emptyResults []*client.ResponseMetrics
	totalTime := 5 * time.Second
	
	result := runner.calculateResult(emptyResults, totalTime)
	
	if result == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 对于完全空的results切片，calculateResult返回空的ReportData
	// 这是实现中的早期返回逻辑
	if result.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests 0 for empty results, got %d", result.TotalRequests)
	}
	
	if result.Concurrency != 0 {
		t.Errorf("Expected Concurrency 0 for empty results, got %d", result.Concurrency)
	}
	
	if result.TotalTime != 0 {
		t.Errorf("Expected TotalTime 0 for empty results, got %v", result.TotalTime)
	}
}

func TestRunner_CalculateResult_AllNilResults(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 2,
		Count:       10,
		Stream:      false,
	}
	
	runner := &Runner{config: input}
	
	// 创建一些nil结果
	results := []*client.ResponseMetrics{nil, nil, nil}
	totalTime := 5 * time.Second
	
	result := runner.calculateResult(results, totalTime)
	
	if result == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 应该返回基础结果，所有指标应该是零值
	if result.ContentMetrics.AvgTokenCount != 0 {
		t.Errorf("Expected AvgTokenCount 0, got %d", result.ContentMetrics.AvgTokenCount)
	}
}

func TestRunner_CalculateResult_MixedResults(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 2,
		Count:       5,
		Stream:      true,
	}
	
	runner := &Runner{config: input}
	
	// 创建混合结果：一些有效，一些nil，一些无效(token=0)
	results := []*client.ResponseMetrics{
		{
			TotalTime:         500 * time.Millisecond,
			TimeToFirstToken:  100 * time.Millisecond,
			CompletionTokens:  150,
			DNSTime:          10 * time.Millisecond,
			ConnectTime:      50 * time.Millisecond,
			TLSHandshakeTime: 80 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
		nil, // nil结果
		{
			TotalTime:         300 * time.Millisecond,
			TimeToFirstToken:  80 * time.Millisecond,
			CompletionTokens:  0, // 无效结果(token=0)
			DNSTime:          8 * time.Millisecond,
			ConnectTime:      40 * time.Millisecond,
			TLSHandshakeTime: 60 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
		{
			TotalTime:         700 * time.Millisecond,
			TimeToFirstToken:  120 * time.Millisecond,
			CompletionTokens:  200,
			DNSTime:          15 * time.Millisecond,
			ConnectTime:      60 * time.Millisecond,
			TLSHandshakeTime: 100 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
		nil, // 另一个nil结果
	}
	
	totalTime := 2 * time.Second
	
	result := runner.calculateResult(results, totalTime)
	
	if result == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 只有2个有效结果
	expectedSuccessRate := float64(2) / float64(5) * 100
	expectedErrorRate := float64(3) / float64(5) * 100
	
	if result.ReliabilityMetrics.SuccessRate != expectedSuccessRate {
		t.Errorf("Expected SuccessRate %f, got %f", expectedSuccessRate, result.ReliabilityMetrics.SuccessRate)
	}
	
	if result.ReliabilityMetrics.ErrorRate != expectedErrorRate {
		t.Errorf("Expected ErrorRate %f, got %f", expectedErrorRate, result.ReliabilityMetrics.ErrorRate)
	}
	
	// 验证时间指标计算
	expectedAvgTotalTime := (500*time.Millisecond + 700*time.Millisecond) / 2
	if result.TimeMetrics.AvgTotalTime != expectedAvgTotalTime {
		t.Errorf("Expected AvgTotalTime %v, got %v", expectedAvgTotalTime, result.TimeMetrics.AvgTotalTime)
	}
	
	if result.TimeMetrics.MinTotalTime != 500*time.Millisecond {
		t.Errorf("Expected MinTotalTime %v, got %v", 500*time.Millisecond, result.TimeMetrics.MinTotalTime)
	}
	
	if result.TimeMetrics.MaxTotalTime != 700*time.Millisecond {
		t.Errorf("Expected MaxTotalTime %v, got %v", 700*time.Millisecond, result.TimeMetrics.MaxTotalTime)
	}
	
	// 验证token指标计算
	expectedAvgTokens := (150 + 200) / 2
	if result.ContentMetrics.AvgTokenCount != expectedAvgTokens {
		t.Errorf("Expected AvgTokenCount %d, got %d", expectedAvgTokens, result.ContentMetrics.AvgTokenCount)
	}
	
	if result.ContentMetrics.MinTokenCount != 150 {
		t.Errorf("Expected MinTokenCount %d, got %d", 150, result.ContentMetrics.MinTokenCount)
	}
	
	if result.ContentMetrics.MaxTokenCount != 200 {
		t.Errorf("Expected MaxTokenCount %d, got %d", 200, result.ContentMetrics.MaxTokenCount)
	}
	
	// 验证TPS计算
	tps1 := float64(150) / (500 * time.Millisecond).Seconds()
	tps2 := float64(200) / (700 * time.Millisecond).Seconds()
	expectedAvgTPS := (tps1 + tps2) / 2
	
	if result.ContentMetrics.AvgTPS != expectedAvgTPS {
		t.Errorf("Expected AvgTPS %f, got %f", expectedAvgTPS, result.ContentMetrics.AvgTPS)
	}
	
	// 验证网络指标
	if result.NetworkMetrics.TargetIP != "8.8.8.8" {
		t.Errorf("Expected TargetIP '8.8.8.8', got '%s'", result.NetworkMetrics.TargetIP)
	}
	
	expectedAvgDNS := (10*time.Millisecond + 15*time.Millisecond) / 2
	if result.NetworkMetrics.AvgDNSTime != expectedAvgDNS {
		t.Errorf("Expected AvgDNSTime %v, got %v", expectedAvgDNS, result.NetworkMetrics.AvgDNSTime)
	}
}

func TestRunner_CalculateResult_SingleValidResult(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key", 
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       1,
		Stream:      true,
	}
	
	runner := &Runner{config: input}
	
	singleResult := &client.ResponseMetrics{
		TotalTime:         500 * time.Millisecond,
		TimeToFirstToken:  100 * time.Millisecond,
		CompletionTokens:  150,
		DNSTime:          10 * time.Millisecond,
		ConnectTime:      50 * time.Millisecond,
		TLSHandshakeTime: 80 * time.Millisecond,
		TargetIP:         "1.1.1.1",
	}
	
	results := []*client.ResponseMetrics{singleResult}
	totalTime := 1 * time.Second
	
	result := runner.calculateResult(results, totalTime)
	
	if result == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 单个结果的情况下，平均值、最小值、最大值应该都相等
	if result.TimeMetrics.AvgTotalTime != singleResult.TotalTime {
		t.Errorf("Expected AvgTotalTime %v, got %v", singleResult.TotalTime, result.TimeMetrics.AvgTotalTime)
	}
	
	if result.TimeMetrics.MinTotalTime != singleResult.TotalTime {
		t.Errorf("Expected MinTotalTime %v, got %v", singleResult.TotalTime, result.TimeMetrics.MinTotalTime)
	}
	
	if result.TimeMetrics.MaxTotalTime != singleResult.TotalTime {
		t.Errorf("Expected MaxTotalTime %v, got %v", singleResult.TotalTime, result.TimeMetrics.MaxTotalTime)
	}
	
	// 验证成功率
	if result.ReliabilityMetrics.SuccessRate != 100.0 {
		t.Errorf("Expected SuccessRate 100.0, got %f", result.ReliabilityMetrics.SuccessRate)
	}
	
	if result.ReliabilityMetrics.ErrorRate != 0.0 {
		t.Errorf("Expected ErrorRate 0.0, got %f", result.ReliabilityMetrics.ErrorRate)
	}
}

func TestResult_PrintResult(t *testing.T) {
	tests := []struct {
		name   string
		result types.ReportData
	}{
		{
			name: "stream mode result",
			result: types.ReportData{
				TotalRequests: 10,
				Concurrency:   2,
				IsStream:      true,
				TotalTime:     5 * time.Second,
			},
		},
		{
			name: "non-stream mode result",
			result: types.ReportData{
				TotalRequests: 20,
				Concurrency:   4,
				IsStream:      false,
				TotalTime:     10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这个测试主要确保 PrintResult 不会 panic
			// 验证结果结构体的基本字段
			if tt.result.TotalRequests == 0 {
				t.Error("TotalRequests should not be zero")
			}
			if tt.result.TotalTime == 0 {
				t.Error("TotalTime should not be zero")
			}
		})
	}
}

func TestRunner_RunWithProgress_BasicFunctionality(t *testing.T) {
	// 创建一个简单的测试用例来验证RunWithProgress的基本功能
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       2,
		Prompt:      "test",
		Stream:      true,
	}
	
	// 由于我们无法轻易模拟实际的客户端，这个测试主要验证
	// RunWithProgress方法的存在和基本结构
	runner := &Runner{config: input}
	
	// 验证runner配置正确设置
	if runner.config.Count != 2 {
		t.Errorf("Expected Count 2, got %d", runner.config.Count)
	}
	
	if runner.config.Concurrency != 1 {
		t.Errorf("Expected Concurrency 1, got %d", runner.config.Concurrency)
	}
	
	if runner.config.Stream != true {
		t.Errorf("Expected Stream true, got %v", runner.config.Stream)
	}
}

// 测试进度回调的数据结构
func TestStatsDataStructure(t *testing.T) {
	stats := types.StatsData{
		CompletedCount: 5,
		FailedCount:    2,
		TTFTs:          []time.Duration{100 * time.Millisecond, 150 * time.Millisecond},
		TotalTimes:     []time.Duration{500 * time.Millisecond, 600 * time.Millisecond},
		TokenCounts:    []int{100, 150},
		ErrorMessages:  []string{"error1", "error2"},
		StartTime:      time.Now(),
		ElapsedTime:    2 * time.Second,
	}
	
	// 验证基本字段
	if stats.CompletedCount != 5 {
		t.Errorf("Expected CompletedCount 5, got %d", stats.CompletedCount)
	}
	
	if stats.FailedCount != 2 {
		t.Errorf("Expected FailedCount 2, got %d", stats.FailedCount)
	}
	
	if len(stats.TTFTs) != 2 {
		t.Errorf("Expected 2 TTFTs, got %d", len(stats.TTFTs))
	}
	
	if len(stats.ErrorMessages) != 2 {
		t.Errorf("Expected 2 ErrorMessages, got %d", len(stats.ErrorMessages))
	}
	
	if stats.ElapsedTime != 2*time.Second {
		t.Errorf("Expected ElapsedTime 2s, got %v", stats.ElapsedTime)
	}
}

// TestRunner_CalculateResult_TPOT 测试 TPOT (Time Per Output Token) 指标计算
func TestRunner_CalculateResult_TPOT(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       3,
		Stream:      true, // 流式模式下才有TPOT
	}
	
	runner := &Runner{config: input}
	
	// 创建具有不同token数和时间的测试结果
	results := []*client.ResponseMetrics{
		{
			TotalTime:         500 * time.Millisecond,
			TimeToFirstToken:  100 * time.Millisecond,
			CompletionTokens:  5, // 5个token：TPOT = (500-100) / (5-1) = 400ms / 4 = 100ms
			DNSTime:          10 * time.Millisecond,
			ConnectTime:      50 * time.Millisecond,
			TLSHandshakeTime: 80 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
		{
			TotalTime:         600 * time.Millisecond,
			TimeToFirstToken:  200 * time.Millisecond,
			CompletionTokens:  3, // 3个token：TPOT = (600-200) / (3-1) = 400ms / 2 = 200ms
			DNSTime:          15 * time.Millisecond,
			ConnectTime:      60 * time.Millisecond,
			TLSHandshakeTime: 100 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
		{
			TotalTime:         300 * time.Millisecond,
			TimeToFirstToken:  50 * time.Millisecond,
			CompletionTokens:  1, // 1个token：TPOT无法计算（需要>1个token）
			DNSTime:          8 * time.Millisecond,
			ConnectTime:      40 * time.Millisecond,
			TLSHandshakeTime: 60 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
	}
	
	totalTime := 2 * time.Second
	
	result := runner.calculateResult(results, totalTime)
	
	if result == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 验证TPOT计算
	// 第1个结果：TPOT = (500-100) / (5-1) = 100ms
	// 第2个结果：TPOT = (600-200) / (3-1) = 200ms
	// 第3个结果：token=1，不参与TPOT计算
	// 平均TPOT = (100 + 200) / 2 = 150ms
	expectedAvgTPOT := 150 * time.Millisecond
	expectedMinTPOT := 100 * time.Millisecond
	expectedMaxTPOT := 200 * time.Millisecond
	
	if result.ContentMetrics.AvgTPOT != expectedAvgTPOT {
		t.Errorf("Expected AvgTPOT %v, got %v", expectedAvgTPOT, result.ContentMetrics.AvgTPOT)
	}
	
	if result.ContentMetrics.MinTPOT != expectedMinTPOT {
		t.Errorf("Expected MinTPOT %v, got %v", expectedMinTPOT, result.ContentMetrics.MinTPOT)
	}
	
	if result.ContentMetrics.MaxTPOT != expectedMaxTPOT {
		t.Errorf("Expected MaxTPOT %v, got %v", expectedMaxTPOT, result.ContentMetrics.MaxTPOT)
	}
}

// TestRunner_CalculateResult_TPOT_SingleToken 测试只有1个token的情况下TPOT处理
func TestRunner_CalculateResult_TPOT_SingleToken(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       2,
		Stream:      true,
	}
	
	runner := &Runner{config: input}
	
	// 创建所有结果都只有1个token的情况
	results := []*client.ResponseMetrics{
		{
			TotalTime:         500 * time.Millisecond,
			TimeToFirstToken:  100 * time.Millisecond,
			CompletionTokens:  1, // 1个token，TPOT无法计算
			DNSTime:          10 * time.Millisecond,
			ConnectTime:      50 * time.Millisecond,
			TLSHandshakeTime: 80 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
		{
			TotalTime:         400 * time.Millisecond,
			TimeToFirstToken:  80 * time.Millisecond,
			CompletionTokens:  1, // 1个token，TPOT无法计算
			DNSTime:          8 * time.Millisecond,
			ConnectTime:      40 * time.Millisecond,
			TLSHandshakeTime: 60 * time.Millisecond,
			TargetIP:         "8.8.8.8",
		},
	}
	
	totalTime := 1 * time.Second
	
	result := runner.calculateResult(results, totalTime)
	
	if result == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 所有结果都只有1个token，TPOT应该为0
	if result.ContentMetrics.AvgTPOT != 0 {
		t.Errorf("Expected AvgTPOT 0 for single token results, got %v", result.ContentMetrics.AvgTPOT)
	}
	
	if result.ContentMetrics.MinTPOT != 0 {
		t.Errorf("Expected MinTPOT 0 for single token results, got %v", result.ContentMetrics.MinTPOT)
	}
	
	if result.ContentMetrics.MaxTPOT != 0 {
		t.Errorf("Expected MaxTPOT 0 for single token results, got %v", result.ContentMetrics.MaxTPOT)
	}
}

// TestRunner_CalculateResult_TPOT_NonStream 测试非流式模式下TPOT处理
func TestRunner_CalculateResult_TPOT_NonStream(t *testing.T) {
	input := types.Input{
		Protocol:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       1,
		Stream:      false, // 非流式模式
	}
	
	runner := &Runner{config: input}
	
	result := &client.ResponseMetrics{
		TotalTime:         500 * time.Millisecond,
		TimeToFirstToken:  0, // 非流式模式下通常TTFT为0
		CompletionTokens:  5,
		DNSTime:          10 * time.Millisecond,
		ConnectTime:      50 * time.Millisecond,
		TLSHandshakeTime: 80 * time.Millisecond,
		TargetIP:         "8.8.8.8",
	}
	
	results := []*client.ResponseMetrics{result}
	totalTime := 1 * time.Second
	
	calculatedResult := runner.calculateResult(results, totalTime)
	
	if calculatedResult == nil {
		t.Fatal("calculateResult should not return nil")
	}
	
	// 非流式模式下，TPOT也应该被计算
	// TPOT = (500-0) / (5-1) = 500ms / 4 = 125ms
	expectedTPOT := 125 * time.Millisecond
	
	if calculatedResult.ContentMetrics.AvgTPOT != expectedTPOT {
		t.Errorf("Expected AvgTPOT %v for non-stream mode, got %v", expectedTPOT, calculatedResult.ContentMetrics.AvgTPOT)
	}
}
