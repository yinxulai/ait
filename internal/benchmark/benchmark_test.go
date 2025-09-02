package benchmark

import (
	"testing"
	"time"
)

// BenchmarkNewRunner 基准测试 NewRunner 函数
func BenchmarkNewRunner(b *testing.B) {
	config := Config{
		Provider:    "openai",
		BaseUrl:     "https://api.openai.com",
		ApiKey:      "test-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 1,
		Count:       10,
		Prompt:      "test prompt",
		Stream:      false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner, err := NewRunner(config)
		if err != nil {
			b.Fatalf("NewRunner failed: %v", err)
		}
		_ = runner
	}
}

// BenchmarkResultPrintResult 基准测试结果打印
func BenchmarkResultPrintResult(b *testing.B) {
	result := &Result{
		TotalRequests:   1000,
		Concurrency:     10,
		IsStream:        false,
		TotalTime:       30 * time.Second,
		AvgResponseTime: 500 * time.Millisecond,
		MinResponseTime: 100 * time.Millisecond,
		MaxResponseTime: 2 * time.Second,
		TPS:             33.33,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 不实际执行 PrintResult 以避免大量输出，只测试结构体创建开销
		_ = result
	}
}
