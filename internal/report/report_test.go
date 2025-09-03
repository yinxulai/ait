package report

import (
	"testing"
	"time"
)

func TestReporter_Generate(t *testing.T) {
	// 创建测试配置
	config := TestConfig{
		Provider:    "openai",
		BaseUrl:     "https://api.openai.com/v1",
		ApiKey:      "test-api-key",
		Model:       "gpt-3.5-turbo",
		Concurrency: 3,
		Count:       10,
		Stream:      true,
		Prompt:      "Hello, world!",
	}

	// 创建测试结果
	result := TestResult{
		TotalRequests: 10,
		Concurrency:   3,
		IsStream:      true,
		TotalTime:     time.Second * 30,
	}

	// 设置一些测试数据
	result.TimeMetrics.AvgTotalTime = time.Millisecond * 1500
	result.TimeMetrics.MinTotalTime = time.Millisecond * 1000
	result.TimeMetrics.MaxTotalTime = time.Millisecond * 2000

	result.NetworkMetrics.TargetIP = "192.168.1.1"
	result.NetworkMetrics.AvgDNSTime = time.Millisecond * 10
	result.NetworkMetrics.MinDNSTime = time.Millisecond * 5
	result.NetworkMetrics.MaxDNSTime = time.Millisecond * 15

	result.ContentMetrics.AvgTokenCount = 100
	result.ContentMetrics.MinTokenCount = 50
	result.ContentMetrics.MaxTokenCount = 150
	result.ContentMetrics.AvgTPS = 66.7

	result.ReliabilityMetrics.SuccessRate = 100.0
	result.ReliabilityMetrics.ErrorRate = 0.0

	// 创建报告生成器
	reporter := NewReporter(config, result)

	// 生成报告
	err := reporter.Generate()
	if err != nil {
		t.Fatalf("生成报告失败: %v", err)
	}

	// 清理测试文件
	// 注意：这里只是一个简单的清理，实际测试中可能需要更精确的文件名
	// 由于文件名包含时间戳，我们无法准确预测文件名，所以这里跳过清理
	t.Log("报告生成测试完成")
}
