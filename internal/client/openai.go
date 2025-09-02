package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	if baseUrl == "" {
		baseUrl = "https://api.openai.com"
	}
	return &OpenAIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseUrl,
		apiKey:     apiKey,
		Model:      model,
		Provider:   "openai",
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
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	t0 := time.Now()

	if stream {
		// 流式请求
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false
		var fullContent strings.Builder
		var totalTokens int
		
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
				
				if !gotFirst && len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					firstTokenTime = time.Since(t0)
					gotFirst = true
				}
				
				// 累积内容
				if len(chunk.Choices) > 0 {
					fullContent.WriteString(chunk.Choices[0].Delta.Content)
				}
				
				// 获取 token 统计信息（通常在最后一个chunk中）
				if chunk.Usage != nil {
					totalTokens = chunk.Usage.TotalTokens
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
			TokenCount:       totalTokens,
		}, nil
	} else {
		// 非流式请求
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
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
			TimeToFirstToken: totalTime, // 非流式模式下，首个token时间就是总时间
			TotalTime:        totalTime,
			TokenCount:       chatResp.Usage.TotalTokens,
		}, nil
	}
}

// GetProvider 获取协议类型
func (c *OpenAIClient) GetProvider() string {
	return c.Provider
}

// GetModel 获取模型名称
func (c *OpenAIClient) GetModel() string {
	return c.Model
}
