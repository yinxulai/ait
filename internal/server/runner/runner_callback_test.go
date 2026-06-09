package runner

import (
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/types"
)

func TestRunner_RunWithCallback_InvokesCallbackForEachRequest(t *testing.T) {
	input := types.Input{
		Protocol:     types.ProtocolOpenAICompletions,
		BaseUrl:      "https://api.openai.com",
		ApiKey:       "test-key",
		Model:        "gpt-4.1-mini",
		Concurrency:  2,
		Count:        4,
		PromptSource: createTestPromptSource("test prompt"),
		Stream:       true,
	}

	mockClient := &MockClient{
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         200 * time.Millisecond,
			TimeToFirstToken:  50 * time.Millisecond,
			PromptTokens:      100,
			CachedInputTokens: 25,
			CompletionTokens:  100,
			ThinkingTokens:    20,
			DNSTime:           10 * time.Millisecond,
			ConnectTime:       20 * time.Millisecond,
			TLSHandshakeTime:  30 * time.Millisecond,
			TargetIP:          "8.8.8.8",
		},
	}

	runner := NewRunnerWithClient(input, mockClient)
	var callbackCount atomic.Int64

	result, err := runner.RunWithCallback(func(metrics *client.ResponseMetrics, index int, err error) {
		if err != nil {
			t.Errorf("callback received unexpected error: %v", err)
		}
		if metrics == nil {
			t.Errorf("callback metrics should not be nil for index %d", index)
			return
		}
		callbackCount.Add(1)
	})
	if err != nil {
		t.Fatalf("RunWithCallback() returned unexpected error: %v", err)
	}
	if callbackCount.Load() != int64(input.Count) {
		t.Fatalf("expected %d callbacks, got %d", input.Count, callbackCount.Load())
	}
	if result.AvgCachedInputTokenCount != 25 {
		t.Fatalf("expected AvgCachedInputTokenCount 25, got %d", result.AvgCachedInputTokenCount)
	}
	if result.AvgCacheHitRate != 0.25 {
		t.Fatalf("expected AvgCacheHitRate 0.25, got %f", result.AvgCacheHitRate)
	}
}

func TestRunner_Stop_StopsLaunchingNewRequests(t *testing.T) {
	input := types.Input{
		Protocol:     types.ProtocolOpenAICompletions,
		BaseUrl:      "https://api.openai.com",
		ApiKey:       "test-key",
		Model:        "gpt-4.1-mini",
		Concurrency:  1,
		Count:        20,
		PromptSource: createTestPromptSource("test prompt"),
		Stream:       false,
	}

	mockClient := &MockClient{
		requestDelay: 40 * time.Millisecond,
		responseMetrics: &client.ResponseMetrics{
			TotalTime:         40 * time.Millisecond,
			TimeToFirstToken:  40 * time.Millisecond,
			PromptTokens:      80,
			CachedInputTokens: 10,
			CompletionTokens:  30,
		},
	}

	runner := NewRunnerWithClient(input, mockClient)
	resultCh := make(chan *types.ReportData, 1)
	errCh := make(chan error, 1)
	var callbackCount atomic.Int64

	go func() {
		result, err := runner.RunWithCallback(func(metrics *client.ResponseMetrics, index int, err error) {
			if callbackCount.Add(1) == 1 {
				runner.Stop()
			}
		})
		resultCh <- result
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunWithCallback() returned unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunWithCallback() did not finish after Stop()")
	}

	result := <-resultCh
	callCount := mockClient.GetCallCount()
	if callCount >= int64(input.Count) {
		t.Fatalf("expected Stop() to stop before launching all requests, got %d calls", callCount)
	}
	if int64(result.TotalRequests) != callCount {
		t.Fatalf("expected TotalRequests %d to match launched calls, got %d", callCount, result.TotalRequests)
	}
	if callbackCount.Load() != callCount {
		t.Fatalf("expected callback count %d, got %d", callCount, callbackCount.Load())
	}
}

func TestRunner_CalculateResult_CacheHitRate(t *testing.T) {
	input := types.Input{
		Protocol:     types.ProtocolOpenAICompletions,
		BaseUrl:      "https://api.openai.com",
		ApiKey:       "test-key",
		Model:        "gpt-4.1-mini",
		Concurrency:  1,
		Count:        2,
		PromptSource: createTestPromptSource("test prompt"),
		Stream:       true,
	}

	runner := NewRunnerWithClient(input, &MockClient{})
	results := []*client.ResponseMetrics{
		{
			TotalTime:         2 * time.Second,
			TimeToFirstToken:  500 * time.Millisecond,
			PromptTokens:      100,
			CachedInputTokens: 50,
			CompletionTokens:  100,
			TargetIP:          "1.1.1.1",
		},
		{
			TotalTime:         1 * time.Second,
			TimeToFirstToken:  250 * time.Millisecond,
			PromptTokens:      80,
			CachedInputTokens: 20,
			CompletionTokens:  60,
			TargetIP:          "1.1.1.1",
		},
	}

	result := runner.calculateResult(results, 3*time.Second, 2)
	if result.AvgCachedInputTokenCount != 35 {
		t.Fatalf("expected AvgCachedInputTokenCount 35, got %d", result.AvgCachedInputTokenCount)
	}
	if result.MinCachedInputTokenCount != 20 || result.MaxCachedInputTokenCount != 50 {
		t.Fatalf("unexpected cached input token min/max: %d/%d", result.MinCachedInputTokenCount, result.MaxCachedInputTokenCount)
	}
	if math.Abs(result.AvgCacheHitRate-0.375) > 0.00001 {
		t.Fatalf("expected AvgCacheHitRate 0.375, got %f", result.AvgCacheHitRate)
	}
	if math.Abs(result.MinCacheHitRate-0.25) > 0.00001 || math.Abs(result.MaxCacheHitRate-0.5) > 0.00001 {
		t.Fatalf("unexpected cache hit rate min/max: %f/%f", result.MinCacheHitRate, result.MaxCacheHitRate)
	}
}
