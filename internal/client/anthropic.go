package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicClient Anthropic 协议客户端
type AnthropicClient struct {
	BaseUrl  string
	ApiKey   string
	Model    string
	Provider string
}

// NewAnthropicClient 创建新的 Anthropic 客户端
func NewAnthropicClient(baseUrl, apiKey, model string) *AnthropicClient {
	return &AnthropicClient{
		BaseUrl:  baseUrl,
		ApiKey:   apiKey,
		Model:    model,
		Provider: "anthropic",
	}
}

// Request 发送 Anthropic 协议请求（支持流式和非流式）
func (c *AnthropicClient) Request(prompt string, stream bool) (time.Duration, error) {
	// Anthropic 使用不同的 API 格式
	reqBodyTemplate := `{
		"model": "%s",
		"max_tokens": 1024,
		"messages": [
			{
				"role": "user",
				"content": "%s"
			}
		],
		"stream": %t
	}`

	reqBody := []byte(fmt.Sprintf(reqBodyTemplate, c.Model, prompt, stream))

	req, err := http.NewRequest("POST", c.BaseUrl+"/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-api-key", c.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	t0 := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if stream {
		// 流式响应，读取第一个数据块来测量 TTFT
		buffer := make([]byte, 1024)
		_, err = resp.Body.Read(buffer)
		firstTokenTime := time.Since(t0)
		if err != nil && err != io.EOF {
			return 0, err
		}

		// 继续读取剩余数据
		_, _ = io.ReadAll(resp.Body)
		return firstTokenTime, nil
	} else {
		// 非流式响应，读取完整响应
		_, _ = io.ReadAll(resp.Body)
		return time.Since(t0), nil
	}
}

// GetProvider 获取协议类型
func (c *AnthropicClient) GetProvider() string {
	return c.Provider
}

// GetModel 获取模型名称
func (c *AnthropicClient) GetModel() string {
	return c.Model
}
