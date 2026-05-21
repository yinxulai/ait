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
	"github.com/yinxulai/ait/internal/types"
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

// ThinkingOptions represents reasoning options for chat completion
type ThinkingOptions struct {
	Type string `json:"type"`
}

type ResponsesReasoningOptions struct {
	Effort string `json:"effort,omitempty"`
}

// CompletionTokensDetails represents detailed completion token usage breakdown
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
	ThinkingTokens  int `json:"thinking_tokens"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// ChatCompletionRequest represents the request payload for chat completion
type ChatCompletionRequest struct {
	Model         string                  `json:"model"`
	Messages      []ChatCompletionMessage `json:"messages"`
	Stream        bool                    `json:"stream,omitempty"`
	StreamOptions *StreamOptions          `json:"stream_options,omitempty"`
	Thinking      *ThinkingOptions        `json:"thinking,omitempty"`
}

type ResponsesAPIInputItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponsesAPIRequest struct {
	Model        string                     `json:"model"`
	Input        []ResponsesAPIInputItem    `json:"input"`
	Instructions string                     `json:"instructions,omitempty"`
	Store        bool                       `json:"store,omitempty"`
	Stream       bool                       `json:"stream,omitempty"`
	Reasoning    *ResponsesReasoningOptions `json:"reasoning,omitempty"`
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
		PromptTokens            int                      `json:"prompt_tokens"`
		CompletionTokens        int                      `json:"completion_tokens"`
		TotalTokens             int                      `json:"total_tokens"`
		PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
		CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	} `json:"usage"`
}

type ResponsesAPIResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	CreatedAt int64  `json:"created_at"`
	Model     string `json:"model"`
	Output    []struct {
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	} `json:"output"`
	Usage struct {
		InputTokens         int                      `json:"input_tokens"`
		OutputTokens        int                      `json:"output_tokens"`
		TotalTokens         int                      `json:"total_tokens"`
		InputTokensDetails  *PromptTokensDetails     `json:"input_tokens_details,omitempty"`
		OutputTokensDetails *CompletionTokensDetails `json:"output_tokens_details,omitempty"`
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
			ThinkingContent *string `json:"reasoning_content,omitempty"`
			Content         string  `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens            int                      `json:"prompt_tokens"`
		CompletionTokens        int                      `json:"completion_tokens"`
		TotalTokens             int                      `json:"total_tokens"`
		PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
		CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	} `json:"usage,omitempty"`
}

type ResponsesAPIStreamEvent struct {
	Type     string                `json:"type"`
	Delta    string                `json:"delta,omitempty"`
	Response *ResponsesAPIResponse `json:"response,omitempty"`
	Usage    *struct {
		InputTokens         int                      `json:"input_tokens"`
		OutputTokens        int                      `json:"output_tokens"`
		TotalTokens         int                      `json:"total_tokens"`
		InputTokensDetails  *PromptTokensDetails     `json:"input_tokens_details,omitempty"`
		OutputTokensDetails *CompletionTokensDetails `json:"output_tokens_details,omitempty"`
	} `json:"usage,omitempty"`
}

func extractThinkingTokens(details *CompletionTokensDetails) int {
	if details == nil {
		return 0
	}
	if details.ThinkingTokens > 0 {
		return details.ThinkingTokens
	}
	return details.ReasoningTokens
}

func extractCachedInputTokens(details *PromptTokensDetails) int {
	if details == nil {
		return 0
	}
	return details.CachedTokens
}

func (c *OpenAIClient) buildRequestBody(systemPrompt, userPrompt string, stream bool) ([]byte, error) {
	if c.Provider == types.ProtocolOpenAIResponses {
		reqBody := ResponsesAPIRequest{
			Model: c.Model,
			Input: []ResponsesAPIInputItem{
				{Role: "user", Content: userPrompt},
			},
			Instructions: systemPrompt,
			Store:        true,
			Stream:       stream,
		}
		if c.Thinking {
			reqBody.Reasoning = &ResponsesReasoningOptions{Effort: "medium"}
		}
		return json.Marshal(reqBody)
	}

	var messages []ChatCompletionMessage
	if systemPrompt != "" {
		messages = append(messages, ChatCompletionMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, ChatCompletionMessage{
		Role:    "user",
		Content: userPrompt,
	})

	reqBody := ChatCompletionRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   stream,
	}

	if stream {
		reqBody.StreamOptions = &StreamOptions{
			IncludeUsage: true,
		}
	}

	if c.Thinking {
		reqBody.Thinking = &ThinkingOptions{
			Type: "enabled",
		}
	}

	return json.Marshal(reqBody)
}

func (c *OpenAIClient) parseResponsesStream(resp *http.Response, t0 time.Time, dnsTime, connectTime, tlsTime time.Duration, targetIP string, requestBody []byte) (*ResponseMetrics, error) {
	scanner := bufio.NewScanner(resp.Body)
	firstTokenTime := time.Duration(0)
	gotFirst := false
	var completionTokens int
	var promptTokens int
	var cachedInputTokens int
	var thinkingTokens int
	var streamChunks []string
	var rawResponseBody strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		rawResponseBody.WriteString(line)
		rawResponseBody.WriteByte('\n')
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		if c.logger != nil && c.logger.IsEnabled() {
			streamChunks = append(streamChunks, data)
		}

		var event ResponsesAPIStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Delta != "" {
			if !gotFirst {
				firstTokenTime = time.Since(t0)
				gotFirst = true
			}
		}

		if event.Usage != nil {
			promptTokens = event.Usage.InputTokens
			completionTokens = event.Usage.OutputTokens
			cachedInputTokens = extractCachedInputTokens(event.Usage.InputTokensDetails)
			thinkingTokens = extractThinkingTokens(event.Usage.OutputTokensDetails)
		}

		if event.Response != nil {
			promptTokens = event.Response.Usage.InputTokens
			completionTokens = event.Response.Usage.OutputTokens
			cachedInputTokens = extractCachedInputTokens(event.Response.Usage.InputTokensDetails)
			thinkingTokens = extractThinkingTokens(event.Response.Usage.OutputTokensDetails)
		}
	}

	if err := scanner.Err(); err != nil {
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "Responses stream scanning failed", err)
		}
		return nil, err
	}

	totalTime := time.Since(t0)
	if c.logger != nil && c.logger.IsEnabled() {
		c.logger.LogResponse(c.Model, logger.ResponseData{
			StatusCode:   resp.StatusCode,
			StreamChunks: streamChunks,
		})
	}

	return &ResponseMetrics{
		TimeToFirstToken:  firstTokenTime,
		TotalTime:         totalTime,
		DNSTime:           dnsTime,
		ConnectTime:       connectTime,
		TLSHandshakeTime:  tlsTime,
		TargetIP:          targetIP,
		PromptTokens:      promptTokens,
		CachedInputTokens: cachedInputTokens,
		CompletionTokens:  completionTokens,
		ThinkingTokens:    thinkingTokens,
		RequestBody:       string(requestBody),
		ResponseBody:      rawResponseBody.String(),
		ErrorMessage:      "",
	}, nil
}

func (c *OpenAIClient) parseResponsesNonStream(responseData []byte, totalTime, dnsTime, connectTime, tlsTime time.Duration, targetIP string, requestBody []byte) (*ResponseMetrics, error) {
	var apiResp ResponsesAPIResponse
	if err := json.Unmarshal(responseData, &apiResp); err != nil {
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "Failed to parse responses API JSON", err)
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
		TimeToFirstToken:  totalTime,
		TotalTime:         totalTime,
		DNSTime:           dnsTime,
		ConnectTime:       connectTime,
		TLSHandshakeTime:  tlsTime,
		TargetIP:          targetIP,
		PromptTokens:      apiResp.Usage.InputTokens,
		CachedInputTokens: extractCachedInputTokens(apiResp.Usage.InputTokensDetails),
		CompletionTokens:  apiResp.Usage.OutputTokens,
		ThinkingTokens:    extractThinkingTokens(apiResp.Usage.OutputTokensDetails),
		RequestBody:       string(requestBody),
		ResponseBody:      string(responseData),
		ErrorMessage:      "",
	}, nil
}

// OpenAIClient OpenAI 协议客户端
type OpenAIClient struct {
	httpClient  *http.Client
	endpointURL string
	apiKey      string
	Model       string
	Provider    string
	Thinking    bool // 是否开启 thinking 模式
	logger      *logger.Logger
}

// NewOpenAIClient 根据配置创建 OpenAI 客户端
//
// 重要配置说明：
//   - DisableKeepAlives=true: 禁用 HTTP 连接复用，确保每个请求都建立新连接
//     这对于准确的性能测量至关重要，因为连接复用会跳过 DNS 解析和 TCP 连接建立时间，
//     导致测量结果不能反映真实的网络性能。在性能基准测试工具中，我们需要测量完整的
//     网络栈性能，包括 DNS 解析、TCP 连接建立、TLS 握手等。
//   - DisableCompression=false: 启用压缩以节省带宽
func NewOpenAIClient(config types.Input) *OpenAIClient {
	endpointURL := config.ResolvedEndpointURL()
	transport := newMeasuredTransport(config)

	return &OpenAIClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   config.Timeout,
		},
		endpointURL: endpointURL,
		apiKey:      config.ApiKey,
		Model:       config.Model,
		Provider:    config.NormalizedProtocol(),
		Thinking:    config.Thinking,
		logger:      nil,
	}
}

// SetLogger 设置日志记录器
func (c *OpenAIClient) SetLogger(l *logger.Logger) {
	c.logger = l
}

// Request 发送 OpenAI 协议请求（支持流式和非流式）
func (c *OpenAIClient) Request(systemPrompt, userPrompt string, stream bool) (*ResponseMetrics, error) {
	// 记录请求开始日志
	if c.logger != nil && c.logger.IsEnabled() {
		c.logger.LogTestStart(c.Model, userPrompt, map[string]interface{}{
			"stream":       stream,
			"protocol":     c.Provider,
			"endpoint_url": c.endpointURL,
		})
	}

	jsonData, err := c.buildRequestBody(systemPrompt, userPrompt, stream)
	if err != nil {
		// 记录错误日志
		if c.logger != nil && c.logger.IsEnabled() {
			c.logger.Error(c.Model, "JSON encoding failed", err)
		}
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", c.endpointURL, bytes.NewBuffer(jsonData))
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
			RequestBody:      string(jsonData),
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
				RequestBody:      string(jsonData),
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
				RequestBody:      string(jsonData),
				ResponseBody:     responseBody,
				ErrorMessage:     errorMessage,
			}, fmt.Errorf(errorMessage)
		}

		if c.Provider == types.ProtocolOpenAIResponses {
			return c.parseResponsesStream(resp, t0, dnsTime, connectTime, tlsTime, targetIP, jsonData)
		}

		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false
		var fullContent strings.Builder
		var completionTokens int
		var promptTokens int
		var cachedInputTokens int
		var thinkingTokens int
		var streamChunks []string // 用于记录所有流式数据块
		var rawResponseLines strings.Builder

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
			rawResponseLines.WriteString(line)
			rawResponseLines.WriteByte('\n')
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
					// 检查是否有 ThinkingContent 或 Content，任一不为空都算作第一个 token
					if delta.Content != "" || (delta.ThinkingContent != nil && *delta.ThinkingContent != "") {
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
					cachedInputTokens = extractCachedInputTokens(chunk.Usage.PromptTokensDetails)
					thinkingTokens = extractThinkingTokens(chunk.Usage.CompletionTokensDetails)
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
				"total_time":          totalTime.String(),
				"time_to_first_token": firstTokenTime.String(),
				"prompt_tokens":       promptTokens,
				"cached_input_tokens": cachedInputTokens,
				"completion_tokens":   completionTokens,
				"thinking_tokens":     thinkingTokens,
				"full_content":        fullContent.String(),
			})
		}

		return &ResponseMetrics{
			TimeToFirstToken:  firstTokenTime,
			TotalTime:         totalTime,
			DNSTime:           dnsTime,
			ConnectTime:       connectTime,
			TLSHandshakeTime:  tlsTime,
			TargetIP:          targetIP,
			PromptTokens:      promptTokens,
			CachedInputTokens: cachedInputTokens,
			CompletionTokens:  completionTokens,
			ThinkingTokens:    thinkingTokens,
			RequestBody:       string(jsonData),
			ResponseBody:      rawResponseLines.String(),
			ErrorMessage:      "",
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
				CompletionTokens: 0,			RequestBody:      string(jsonData),				ErrorMessage:     fmt.Sprintf("Network error: %s", err.Error()),
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
				RequestBody:      string(jsonData),
				ResponseBody:     string(responseData),
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
				RequestBody:      string(jsonData),
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

		if c.Provider == types.ProtocolOpenAIResponses {
			return c.parseResponsesNonStream(responseData, totalTime, dnsTime, connectTime, tlsTime, targetIP, jsonData)
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

		thinkingTokens := extractThinkingTokens(chatResp.Usage.CompletionTokensDetails)

		return &ResponseMetrics{
			TimeToFirstToken:  totalTime, // 非流式模式下，所有token一次性返回，TTFT等于总时间
			TotalTime:         totalTime,
			DNSTime:           dnsTime,
			ConnectTime:       connectTime,
			TLSHandshakeTime:  tlsTime,
			TargetIP:          targetIP,
			PromptTokens:      chatResp.Usage.PromptTokens,
			CachedInputTokens: extractCachedInputTokens(chatResp.Usage.PromptTokensDetails),
			CompletionTokens:  chatResp.Usage.CompletionTokens,
			ThinkingTokens:    thinkingTokens,
			RequestBody:       string(jsonData),
			ResponseBody:      string(responseData),
			ErrorMessage:      "",
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
