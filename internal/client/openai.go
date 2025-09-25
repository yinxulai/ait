package client

import (
	"bufio"
	"bytes"
	"context"
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

// ChatCompletionMessage represents a message in the chat completion request
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamOptions represents stream options for chat completion
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ChatCompletionRequest represents the request payload for chat completion
type ChatCompletionRequest struct {
	Model         string                  `json:"model"`
	Messages      []ChatCompletionMessage `json:"messages"`
	Stream        bool                    `json:"stream,omitempty"`
	StreamOptions *StreamOptions          `json:"stream_options,omitempty"`
}

// ChatCompletionResponse represents the response from chat completion
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OpenAIErrorResponse represents OpenAI API error response
type OpenAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

// StreamResponseChunk 流式响应数据块
type StreamResponseChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			ReasoningContent *string `json:"reasoning_content,omitempty"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// OpenAIClient OpenAI 协议客户端
type OpenAIClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	Model      string
	Provider   string
	logger     *logger.Logger
}

// NewOpenAIClient 创建新的 OpenAI 客户端
func NewOpenAIClient(baseUrl, apiKey, model string) *OpenAIClient {
	return NewOpenAIClientWithTimeout(baseUrl, apiKey, model, 30*time.Second)
}

// NewOpenAIClientWithTimeout 创建新的带超时配置的 OpenAI 客户端
// 
// 重要配置说明：
// - DisableKeepAlives=true: 禁用 HTTP 连接复用，确保每个请求都建立新连接
//   这对于准确的性能测量至关重要，因为连接复用会跳过 DNS 解析和 TCP 连接建立时间，
//   导致测量结果不能反映真实的网络性能。在性能基准测试工具中，我们需要测量完整的
//   网络栈性能，包括 DNS 解析、TCP 连接建立、TLS 握手等。
// - DisableCompression=false: 启用压缩以节省带宽
func NewOpenAIClientWithTimeout(baseUrl, apiKey, model string, timeout time.Duration) *OpenAIClient {
	if baseUrl == "" {
		baseUrl = "https://api.openai.com"
	}
	
	// 禁用连接复用以确保每个请求都是独立的
	transport := &http.Transport{
		DisableKeepAlives: true,
		DisableCompression: false,
	}
	
	return &OpenAIClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		baseURL:  baseUrl,
		apiKey:   apiKey,
		Model:    model,
		Provider: "openai",
		logger:   nil,
	}
}

// SetLogger 设置日志记录器
func (c *OpenAIClient) SetLogger(l *logger.Logger) {
	c.logger = l
}

// Request 发送 OpenAI 协议请求（支持流式和非流式）
func (c *OpenAIClient) Request(prompt string, stream bool) (*ResponseMetrics, error) {
	// 记录请求开始日志
	if c.logger != nil && c.logger.IsEnabled() {
		c.logger.LogTestStart(c.Model, prompt, map[string]interface{}{
			"stream":     stream,
			"protocol":   c.Provider,
			"base_url":   c.baseURL,
		})
	}

	reqBody := ChatCompletionRequest{
		Model: c.Model,
		Messages: []ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: stream,
	}

	// 当启用流模式时，添加 stream_options 参数
	if stream {
		reqBody.StreamOptions = &StreamOptions{
			IncludeUsage: true,
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		// 记录错误日志
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "JSON encoding failed", err)
		}
		return nil, err
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonData))
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 记录请求日志
	if c.logger != nil && c.logger.IsEnabled() {
		headers := make(map[string]string)
		for k, v := range req.Header {
			if k == "Authorization" {
				headers[k] = "Bearer ***" // 隐藏敏感信息
			} else {
				headers[k] = strings.Join(v, ", ")
			}
		}
		
		c.logger.LogRequest(c.Model, logger.RequestData{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: headers,
			Body:    string(jsonData),
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

	if stream {
		// 流式请求
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
			
			// 尝试解析 OpenAI API 的错误响应
			var errorResp OpenAIErrorResponse
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

		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false
		var fullContent strings.Builder
		var completionTokens int
		var promptTokens int
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
				if data == "[DONE]" {
					break
				}
				
				// 记录流数据块
				if c.logger != nil && c.logger.IsEnabled() {
					streamChunks = append(streamChunks, data)
				}
				
				var chunk StreamResponseChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					continue // 跳过无法解析的行
				}
				
				if !gotFirst && len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta
					// 检查是否有 ReasoningContent 或 Content，任一不为空都算作第一个 token
					if delta.Content != "" || (delta.ReasoningContent != nil && *delta.ReasoningContent != "") {
						firstTokenTime = time.Since(t0)
						gotFirst = true
					}
				}
				
				// 累积内容
				if len(chunk.Choices) > 0 {
					fullContent.WriteString(chunk.Choices[0].Delta.Content)
				}
				
				// 获取 token 统计信息（通常在最后一个chunk中）
				if chunk.Usage != nil {
					promptTokens = chunk.Usage.PromptTokens
					completionTokens = chunk.Usage.CompletionTokens
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
				"prompt_tokens":      promptTokens,
				"completion_tokens":  completionTokens,
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
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			ErrorMessage:     "",
		}, nil
	} else {
		// 非流式请求
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

		if resp.StatusCode != http.StatusOK {
			responseData, _ := io.ReadAll(resp.Body)
			
			// 尝试解析 OpenAI API 的错误响应
			var errorResp OpenAIErrorResponse
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

		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			// 记录读取响应错误日志
			if c.logger != nil && c.logger.IsEnabled() {
				c.logger.Error(c.Model, "Failed to read response body", err)
			}
			return &ResponseMetrics{
				TimeToFirstToken: 0,
				TotalTime:        time.Since(t0),
				DNSTime:          dnsTime,
				ConnectTime:      connectTime,
				TLSHandshakeTime: tlsTime,
				TargetIP:         targetIP,
				CompletionTokens: 0,
				ErrorMessage:     fmt.Sprintf("Response body read error: %s", err.Error()),
			}, err
		}

		totalTime := time.Since(t0)
		
		// 检查空响应
		if len(responseData) == 0 {
			if c.logger != nil && c.logger.IsEnabled() {
				c.logger.Error(c.Model, "Empty response body", nil)
			}
			return &ResponseMetrics{
				TimeToFirstToken: 0,
				TotalTime:        totalTime,
				DNSTime:          dnsTime,
				ConnectTime:      connectTime,
				TLSHandshakeTime: tlsTime,
				TargetIP:         targetIP,
				CompletionTokens: 0,
				ErrorMessage:     "Empty response body",
			}, fmt.Errorf("empty response body")
		}
		
		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(responseData, &chatResp); err != nil {
			// 记录JSON解析错误日志
			if c.logger != nil && c.logger.IsEnabled() {
				c.logger.Error(c.Model, "Failed to parse response JSON", err)
			}
			return &ResponseMetrics{
				TimeToFirstToken: 0,
				TotalTime:        totalTime,
				DNSTime:          dnsTime,
				ConnectTime:      connectTime,
				TLSHandshakeTime: tlsTime,
				TargetIP:         targetIP,
				CompletionTokens: 0,
				ErrorMessage:     fmt.Sprintf("JSON parsing error: %s", err.Error()),
			}, err
		}

		return &ResponseMetrics{
			TimeToFirstToken: totalTime, // 非流式模式下，所有token一次性返回，TTFT等于总时间
			TotalTime:        totalTime,
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			ErrorMessage:     "",
		}, nil
	}
}

// GetProtocol 获取协议类型
func (c *OpenAIClient) GetProtocol() string {
	return c.Provider
}

// GetModel 获取模型名称
func (c *OpenAIClient) GetModel() string {
	return c.Model
}
