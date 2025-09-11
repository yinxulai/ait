package client

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

// AnthropicResponse Anthropic 非流式响应结构
type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// AnthropicStreamChunk Anthropic 流式响应数据块
type AnthropicStreamChunk struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Thinking *string `json:"thinking,omitempty"`
		PartialJSON *string `json:"partial_json,omitempty"`
	} `json:"delta,omitempty"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

// AnthropicClient Anthropic 协议客户端
type AnthropicClient struct {
	BaseUrl    string
	ApiKey     string
	Model      string
	Provider   string
	httpClient *http.Client
}

// NewAnthropicClient 创建新的 Anthropic 客户端
func NewAnthropicClient(baseUrl, apiKey, model string) *AnthropicClient {
	return NewAnthropicClientWithTimeout(baseUrl, apiKey, model, 30*time.Second)
}

// NewAnthropicClientWithTimeout 创建新的带超时配置的 Anthropic 客户端
// NewAnthropicClientWithTimeout 创建新的带超时配置的 Anthropic 客户端
//
// 重要配置说明：
// - DisableKeepAlives=true: 禁用 HTTP 连接复用，确保每个请求都建立新连接
//   这对于准确的性能测量至关重要，因为连接复用会跳过 DNS 解析和 TCP 连接建立时间，
//   导致测量结果不能反映真实的网络性能。在性能基准测试工具中，我们需要测量完整的
//   网络栈性能，包括 DNS 解析、TCP 连接建立、TLS 握手等。
// - DisableCompression=false: 启用压缩以节省带宽
func NewAnthropicClientWithTimeout(baseUrl, apiKey, model string, timeout time.Duration) *AnthropicClient {
	// 禁用连接复用以确保每个请求都是独立的
	transport := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
	}

	return &AnthropicClient{
		BaseUrl:  baseUrl,
		ApiKey:   apiKey,
		Model:    model,
		Provider: "anthropic",
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

// Request 发送 Anthropic 协议请求（支持流式和非流式）
func (c *AnthropicClient) Request(prompt string, stream bool) (*ResponseMetrics, error) {
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
		// URL 格式错误或其他请求构建错误
		return &ResponseMetrics{
			TimeToFirstToken: 0,
			TotalTime:        0,
			DNSTime:          0,
			ConnectTime:      0,
			TLSHandshakeTime: 0,
			TargetIP:         "",
			CompletionTokens: 0,
			ErrorMessage:     fmt.Sprintf("Request creation error: %s", err.Error()),
		}, err
	}
	req.Header.Set("x-api-key", c.ApiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	// 网络指标收集
	var dnsStart, connectStart, tlsStart time.Time
	var dnsTime, connectTime, tlsTime time.Duration
	var targetIP string
	
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsTime = time.Since(dnsStart)
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			connectTime = time.Since(connectStart)
			if err == nil {
				// 提取 IP 地址（去除端口号）
				if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
					targetIP = host
				} else {
					targetIP = addr
				}
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsTime = time.Since(tlsStart)
		},
	}
	
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	t0 := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 网络错误（如地址错误、连接失败等）
		return &ResponseMetrics{
			TimeToFirstToken: 0,
			TotalTime:        time.Since(t0),
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
			CompletionTokens: 0,
			ErrorMessage:     fmt.Sprintf("Network error: %s", err.Error()),
		}, err
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		responseData, _ := io.ReadAll(resp.Body)
		return &ResponseMetrics{
			TimeToFirstToken: 0,
			TotalTime:        time.Since(t0),
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
			CompletionTokens: 0,
			ErrorMessage:     fmt.Sprintf("API request failed with status %d: %s", resp.StatusCode, string(responseData)),
		}, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	if stream {
		// 流式响应处理
		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false
		var fullContent strings.Builder
		var outputTokens int
		
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if strings.TrimSpace(data) == "" {
					continue
				}
				
				var chunk AnthropicStreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					continue // 跳过无法解析的行
				}
				
				if chunk.Type == "content_block_delta" {
					// 检查是否有任何形式的内容输出（包括 Text、Thinking 或 PartialJSON）
					hasContent := false
					if chunk.Delta.Text != "" {
						fullContent.WriteString(chunk.Delta.Text)
						hasContent = true
					}
					if chunk.Delta.Thinking != nil && *chunk.Delta.Thinking != "" {
						hasContent = true
					}
					if chunk.Delta.PartialJSON != nil && *chunk.Delta.PartialJSON != "" {
						hasContent = true
					}
					
					// 如果有任何内容输出且这是第一次，记录 TTFT 时间
					if hasContent && !gotFirst {
						firstTokenTime = time.Since(t0)
						gotFirst = true
					}
				}
				
				// 获取 token 统计信息
				if chunk.Usage != nil {
					outputTokens = chunk.Usage.OutputTokens
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		totalTime := time.Since(t0)
		
		return &ResponseMetrics{
			TimeToFirstToken: firstTokenTime,
			TotalTime:        totalTime,
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
			CompletionTokens: outputTokens,
			ErrorMessage:     "",
		}, nil
	} else {
		// 非流式响应处理
		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		totalTime := time.Since(t0)
		
		var anthropicResp AnthropicResponse
		if err := json.Unmarshal(responseData, &anthropicResp); err != nil {
			return nil, err
		}

		return &ResponseMetrics{
			TimeToFirstToken: totalTime, // 非流式模式下，所有token一次性返回，TTFT等于总时间
			TotalTime:        totalTime,
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			ErrorMessage:     "",
		}, nil
	}
}

// GetProtocol 获取协议类型
func (c *AnthropicClient) GetProtocol() string {
	return c.Provider
}

// GetModel 获取模型名称
func (c *AnthropicClient) GetModel() string {
	return c.Model
}
