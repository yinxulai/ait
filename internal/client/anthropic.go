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

	"github.com/yinxulai/ait/internal/logger"
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

// AnthropicErrorResponse Anthropic API 错误响应结构
type AnthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
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
	logger     *logger.Logger
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
		logger: nil,
	}
}

// SetLogger 设置日志记录器
func (c *AnthropicClient) SetLogger(l *logger.Logger) {
	c.logger = l
}

// Request 发送 Anthropic 协议请求（支持流式和非流式）
func (c *AnthropicClient) Request(prompt string, stream bool) (*ResponseMetrics, error) {
	// 记录请求开始日志
	if c.logger != nil && c.logger.IsEnabled() {
		c.logger.LogTestStart(c.Model, prompt, map[string]interface{}{
			"stream":     stream,
			"protocol":   c.Provider,
			"base_url":   c.BaseUrl,
		})
	}

	// 构造请求体结构，使用正确的 JSON 编码
	requestBody := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"stream": stream,
	}

	reqBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		// 记录错误日志
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "JSON encoding failed", err)
		}
		return &ResponseMetrics{
			TimeToFirstToken: 0,
			TotalTime:        0,
			DNSTime:          0,
			ConnectTime:      0,
			TLSHandshakeTime: 0,
			TargetIP:         "",
			CompletionTokens: 0,
			ErrorMessage:     fmt.Sprintf("JSON encoding error: %s", err.Error()),
		}, err
	}

	req, err := http.NewRequest("POST", c.BaseUrl+"/v1/messages", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		// 记录错误日志
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "Request creation failed", err)
		}
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

	// 记录请求日志
	if c.logger != nil && c.logger.IsEnabled() {
		headers := make(map[string]string)
		for k, v := range req.Header {
			if k == "x-api-key" {
				headers[k] = "***" // 隐藏敏感信息
			} else {
				headers[k] = strings.Join(v, ", ")
			}
		}
		
		c.logger.LogRequest(c.Model, logger.RequestData{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: headers,
			Body:    string(reqBodyBytes),
		})
	}

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
		// 记录网络错误日志
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "Network error occurred", err)
		}
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
		responseBody := string(responseData)
		
		// 记录HTTP错误响应日志
		if c.logger != nil && c.logger.IsEnabled() {
			headers := make(map[string]string)
			for k, v := range resp.Header {
				headers[k] = strings.Join(v, ", ")
			}
			
			c.logger.LogResponse(c.Model, logger.ResponseData{
				StatusCode: resp.StatusCode,
				Headers:    headers,
				Body:       responseBody,
				Error:      fmt.Sprintf("HTTP %d Error", resp.StatusCode),
			})
		}
		
		// 尝试解析 Anthropic API 的错误响应
		var errorResp AnthropicErrorResponse
		errorMessage := fmt.Sprintf("HTTP %d", resp.StatusCode)
		
		if err := json.Unmarshal(responseData, &errorResp); err == nil && errorResp.Error.Message != "" {
			// 成功解析错误响应，使用业务返回的详细错误信息
			errorMessage = fmt.Sprintf("[%s] %s", 
				errorResp.Error.Type, errorResp.Error.Message)
		}
		
		return &ResponseMetrics{
			TimeToFirstToken: 0,
			TotalTime:        time.Since(t0),
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
			CompletionTokens: 0,
			ErrorMessage:     errorMessage,
		}, fmt.Errorf(errorMessage)
	}

	if stream {
		// 流式响应处理
		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false
		var fullContent strings.Builder
		var outputTokens int
		var streamChunks []string // 用于记录所有流式数据块
		
		// 记录流式响应开始日志
		if c.logger != nil && c.logger.IsEnabled() {
			headers := make(map[string]string)
			for k, v := range resp.Header {
				headers[k] = strings.Join(v, ", ")
			}
			
			c.logger.Debug(c.Model, "Stream response started", map[string]interface{}{
				"status_code": resp.StatusCode,
				"headers":     headers,
			})
		}
		
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if strings.TrimSpace(data) == "" {
					continue
				}
				
				// 记录流数据块
				if c.logger != nil && c.logger.IsEnabled() {
					streamChunks = append(streamChunks, data)
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
			// 记录扫描错误日志
			if c.logger != nil && c.logger.IsEnabled() {
				c.logger.Error(c.Model, "Stream scanning failed", err)
			}
			return nil, err
		}

		totalTime := time.Since(t0)
		
		// 记录流式响应完成日志
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.LogResponse(c.Model, logger.ResponseData{
				StatusCode:   resp.StatusCode,
				StreamChunks: streamChunks,
			})
			
			c.logger.LogTestEnd(c.Model, map[string]interface{}{
				"total_time":         totalTime.String(),
				"time_to_first_token": firstTokenTime.String(),
				"output_tokens":      outputTokens,
				"full_content":       fullContent.String(),
			})
		}
		
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
			// 记录读取响应错误日志
			if c.logger != nil && c.logger.IsEnabled() {
				c.logger.Error(c.Model, "Failed to read response body", err)
			}
			return nil, err
		}

		totalTime := time.Since(t0)
		responseBody := string(responseData)
		
		// 记录响应日志
		if c.logger != nil && c.logger.IsEnabled() {
			headers := make(map[string]string)
			for k, v := range resp.Header {
				headers[k] = strings.Join(v, ", ")
			}
			
			c.logger.LogResponse(c.Model, logger.ResponseData{
				StatusCode: resp.StatusCode,
				Headers:    headers,
				Body:       responseBody,
			})
		}
		
		var anthropicResp AnthropicResponse
		if err := json.Unmarshal(responseData, &anthropicResp); err != nil {
			// 记录JSON解析错误日志
			if c.logger != nil && c.logger.IsEnabled() {
				c.logger.Error(c.Model, "Failed to parse response JSON", err)
			}
			return nil, err
		}

		// 记录测试完成日志
		if c.logger != nil && c.logger.IsEnabled() {
			var contentText string
			if len(anthropicResp.Content) > 0 {
				contentText = anthropicResp.Content[0].Text
			}
			
			c.logger.LogTestEnd(c.Model, map[string]interface{}{
				"total_time":     totalTime.String(),
				"output_tokens":  anthropicResp.Usage.OutputTokens,
				"input_tokens":   anthropicResp.Usage.InputTokens,
				"response_id":    anthropicResp.ID,
				"content_length": len(contentText),
			})
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
