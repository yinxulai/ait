package client

import (
	"fmt"
	"time"
)

// ResponseMetrics 响应指标数据
type ResponseMetrics struct {
	// 时间相关指标
	TimeToFirstToken time.Duration // 首个 token 的响应时间 (TTFT)
	TotalTime        time.Duration // 总耗时 (从请求开始到完全结束)
	
	// 网络连接指标
	DNSTime          time.Duration // DNS解析时间
	ConnectTime      time.Duration // TCP连接建立时间
	TLSHandshakeTime time.Duration // TLS握手时间
	TargetIP         string        // 目标服务器IP地址
	
	// 内容指标
	CompletionTokens int           // 输出 token 数量 (用于TPS计算)
	
	// 错误信息
	ErrorMessage     string        // 错误信息（如果有）
}

// ModelClient 定义统一的模型客户端接口
type ModelClient interface {
	Request(prompt string, stream bool) (*ResponseMetrics, error)
	GetProtocol() string
	GetModel() string
}

// NewClient 根据 protocol 类型创建客户端
func NewClient(protocol, baseUrl, apiKey, model string) (ModelClient, error) {
	return NewClientWithTimeout(protocol, baseUrl, apiKey, model, 30*time.Second)
}

// NewClientWithTimeout 根据 protocol 类型创建带超时配置的客户端
func NewClientWithTimeout(protocol, baseUrl, apiKey, model string, timeout time.Duration) (ModelClient, error) {
	switch protocol {
	case "openai":
		return NewOpenAIClientWithTimeout(baseUrl, apiKey, model, timeout), nil
	case "anthropic":
		return NewAnthropicClientWithTimeout(baseUrl, apiKey, model, timeout), nil
	default:
		return nil, fmt.Errorf("不支持的 protocol 类型: %s", protocol)
	}
}
