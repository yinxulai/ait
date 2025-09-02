package client

import (
	"fmt"
	"time"
)

// ResponseMetrics 响应指标数据
type ResponseMetrics struct {
	TimeToFirstToken time.Duration // 首个 token 的响应时间 (TTFT)
	TotalTime        time.Duration // 总耗时 (从请求开始到完全结束)
	TokenCount       int           // 返回的 token 总数
}

// ModelClient 定义统一的模型客户端接口
type ModelClient interface {
	Request(prompt string, stream bool) (*ResponseMetrics, error)
	GetProvider() string
	GetModel() string
}

// NewClient 根据 provider 类型创建客户端
func NewClient(provider, baseUrl, apiKey, model string) (ModelClient, error) {
	switch provider {
	case "openai":
		return NewOpenAIClient(baseUrl, apiKey, model), nil
	case "anthropic":
		return NewAnthropicClient(baseUrl, apiKey, model), nil
	default:
		return nil, fmt.Errorf("不支持的 provider 类型: %s", provider)
	}
}
