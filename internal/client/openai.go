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
func (c *OpenAIClient) Request(prompt string, stream bool) (time.Duration, error) {
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
		return 0, err
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	t0 := time.Now()

	if stream {
		// 流式请求
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		firstTokenTime := time.Duration(0)
		gotFirst := false

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}
				if !gotFirst {
					firstTokenTime = time.Since(t0)
					gotFirst = true
				}
				// 继续读取但不处理内容
			}
		}

		if err := scanner.Err(); err != nil {
			return 0, err
		}

		return firstTokenTime, nil
	} else {
		// 非流式请求
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}

		// 读取响应但不解析
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		return time.Since(t0), nil
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
