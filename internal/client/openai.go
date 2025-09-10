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
)

// ChatCompletionMessage represents a message in the chat completion request
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents the request payload for chat completion
type ChatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []ChatCompletionMessage `json:"messages"`
	Stream   bool                    `json:"stream,omitempty"`
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
	}
}

// Request 发送 OpenAI 协议请求（支持流式和非流式）
func (c *OpenAIClient) Request(prompt string, stream bool) (*ResponseMetrics, error) {
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

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonData))
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false
		var fullContent strings.Builder
		var completionTokens int
		
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
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
					completionTokens = chunk.Usage.CompletionTokens
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
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		totalTime := time.Since(t0)
		
		var chatResp ChatCompletionResponse
		if err := json.Unmarshal(responseData, &chatResp); err != nil {
			return nil, err
		}

		return &ResponseMetrics{
			TimeToFirstToken: totalTime, // 非流式模式下，所有token一次性返回，TTFT等于总时间
			TotalTime:        totalTime,
			DNSTime:          dnsTime,
			ConnectTime:      connectTime,
			TLSHandshakeTime: tlsTime,
			TargetIP:         targetIP,
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
