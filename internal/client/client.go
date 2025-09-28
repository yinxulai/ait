package client

import (
	"fmt"
	"time"

	"github.com/yinxulai/ait/internal/logger"
	"github.com/yinxulai/ait/internal/types"
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
	PromptTokens     int           // 输入 token 数量
	ThinkingTokens   int           // 思考/推理 token 数量
	CompletionTokens int           // 输出 token 数量 (用于TPS计算)
	
	// 错误信息
	ErrorMessage     string        // 错误信息（如果有）
}

// ModelClient 定义统一的模型客户端接口
type ModelClient interface {
	Request(prompt string, stream bool) (*ResponseMetrics, error)
	GetProtocol() string
	GetModel() string
	SetLogger(logger *logger.Logger) // 设置日志记录器
}

// NewClient 根据配置创建客户端
func NewClient(config types.Input, logger *logger.Logger) (ModelClient, error) {
	switch config.Protocol {
	case "openai":
		client := NewOpenAIClient(config)
		client.SetLogger(logger)
		return client, nil
	case "anthropic":
		client := NewAnthropicClient(config)
		client.SetLogger(logger)
		return client, nil
	default:
		return nil, fmt.Errorf("不支持的 protocol 类型: %s", config.Protocol)
	}
}
