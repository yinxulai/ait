package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// createMockAnthropicServer 创建用于测试 Anthropic API 的模拟 HTTP 服务器
func createMockAnthropicServer(responseDelay time.Duration, stream bool, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求路径
		if !strings.HasSuffix(r.URL.Path, "/v1/messages") {
			http.Error(w, "Invalid API endpoint", http.StatusNotFound)
			return
		}

		// 验证请求头
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		if r.Header.Get("anthropic-version") == "" {
			http.Error(w, "Missing anthropic-version header", http.StatusBadRequest)
			return
		}

		// 模拟延迟
		time.Sleep(responseDelay)

		// 如果指定了非200状态码，直接返回错误
		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			fmt.Fprintf(w, `{"error": {"type": "api_error", "message": "Test error"}}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if stream {
			// 模拟 Anthropic 流式响应
			w.Header().Set("Transfer-Encoding", "chunked")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}

			// 发送开始事件
			fmt.Fprint(w, "event: message_start\n")
			fmt.Fprint(w, `data: {"type": "message_start", "message": {"id": "msg_test", "type": "message", "role": "assistant", "content": [], "model": "claude-3-sonnet", "usage": {"input_tokens": 10, "output_tokens": 0}}}`+"\n\n")
			flusher.Flush()

			// 发送内容块
			for i := 0; i < 3; i++ {
				fmt.Fprint(w, "event: content_block_delta\n")
				fmt.Fprintf(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "chunk %d "}}`, i)
				fmt.Fprint(w, "\n\n")
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}

			// 发送结束事件
			fmt.Fprint(w, "event: message_delta\n")
			fmt.Fprint(w, `data: {"type": "message_delta", "delta": {"stop_reason": "end_turn", "stop_sequence": null}, "usage": {"output_tokens": 15}}`+"\n\n")
			flusher.Flush()
		} else {
			// 模拟 Anthropic 非流式响应
			response := `{
				"id": "msg_test123",
				"type": "message",
				"role": "assistant",
				"content": [
					{
						"type": "text",
						"text": "Hello! I'm Claude, an AI assistant created by Anthropic."
					}
				],
				"model": "claude-3-sonnet-20240229",
				"usage": {
					"input_tokens": 10,
					"output_tokens": 15
				}
			}`
			fmt.Fprint(w, response)
		}
	}))
}

func TestNewAnthropicClient(t *testing.T) {
	tests := []struct {
		name    string
		baseUrl string
		apiKey  string
		model   string
		want    *AnthropicClient
	}{
		{
			name:    "valid anthropic client",
			baseUrl: "https://api.anthropic.com",
			apiKey:  "test-key",
			model:   "claude-3-sonnet-20240229",
			want: &AnthropicClient{
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				Provider: "anthropic",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAnthropicClient(tt.baseUrl, tt.apiKey, tt.model)

			if got.BaseUrl != tt.want.BaseUrl {
				t.Errorf("NewAnthropicClient().BaseUrl = %v, want %v", got.BaseUrl, tt.want.BaseUrl)
			}

			if got.ApiKey != tt.want.ApiKey {
				t.Errorf("NewAnthropicClient().ApiKey = %v, want %v", got.ApiKey, tt.want.ApiKey)
			}

			if got.Model != tt.want.Model {
				t.Errorf("NewAnthropicClient().Model = %v, want %v", got.Model, tt.want.Model)
			}

			if got.Provider != tt.want.Provider {
				t.Errorf("NewAnthropicClient().Provider = %v, want %v", got.Provider, tt.want.Provider)
			}
		})
	}
}

func TestAnthropicClient_GetProvider(t *testing.T) {
	client := NewAnthropicClient("https://api.anthropic.com", "test-key", "claude-3-sonnet-20240229")

	if got := client.GetProvider(); got != "anthropic" {
		t.Errorf("AnthropicClient.GetProvider() = %v, want %v", got, "anthropic")
	}
}

func TestAnthropicClient_GetModel(t *testing.T) {
	model := "claude-3-sonnet-20240229"
	client := NewAnthropicClient("https://api.anthropic.com", "test-key", model)

	if got := client.GetModel(); got != model {
		t.Errorf("AnthropicClient.GetModel() = %v, want %v", got, model)
	}
}

func TestAnthropicClient_Request_NonStream(t *testing.T) {
	server := createMockAnthropicServer(100*time.Millisecond, false, http.StatusOK)
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-key", "claude-3-sonnet-20240229")

	start := time.Now()
	metrics, err := client.Request("test prompt", false)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TimeToFirstToken should be > 0, got %v", metrics.TimeToFirstToken)
	}

	if metrics.CompletionTokens != 15 {
		t.Errorf("Request() CompletionTokens = %v, want 15", metrics.CompletionTokens)
	}

	// 检查实际耗时是否合理（应该至少包含模拟的延迟）
	if elapsed < 100*time.Millisecond {
		t.Errorf("Request() actual time %v should be >= 100ms", elapsed)
	}

	// 验证网络指标是否被设置
	if metrics.DNSTime < 0 {
		t.Errorf("Request() DNSTime should be >= 0, got %v", metrics.DNSTime)
	}

	if metrics.TargetIP == "" {
		t.Error("Request() TargetIP should not be empty")
	}
}

func TestAnthropicClient_Request_Stream(t *testing.T) {
	server := createMockAnthropicServer(50*time.Millisecond, true, http.StatusOK)
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-key", "claude-3-sonnet-20240229")

	start := time.Now()
	metrics, err := client.Request("test prompt", true)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TTFT should be > 0, got %v", metrics.TimeToFirstToken)
	}

	// TTFT 应该小于总耗时（因为我们在流中有多个块）
	if metrics.TimeToFirstToken > elapsed {
		t.Errorf("TTFT %v should be <= total elapsed time %v", metrics.TimeToFirstToken, elapsed)
	}

	if metrics.CompletionTokens != 15 {
		t.Errorf("Request() CompletionTokens = %v, want 15", metrics.CompletionTokens)
	}

	// 验证网络指标是否被设置
	if metrics.DNSTime < 0 {
		t.Errorf("Request() DNSTime should be >= 0, got %v", metrics.DNSTime)
	}
}

func TestAnthropicClient_Request_ServerError(t *testing.T) {
	server := createMockAnthropicServer(0, false, http.StatusInternalServerError)
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-key", "claude-3-sonnet-20240229")

	metrics, err := client.Request("test prompt", false)

	if err == nil {
		t.Error("Request() should return error for server error")
	}

	if metrics == nil {
		t.Error("Request() should return metrics even on error")
	}

	if metrics != nil {
		if metrics.CompletionTokens != 0 {
			t.Errorf("Request() CompletionTokens should be 0 on error, got %v", metrics.CompletionTokens)
		}

		if !strings.Contains(metrics.ErrorMessage, "API request failed with status 500") {
			t.Errorf("Request() ErrorMessage should contain status code, got %v", metrics.ErrorMessage)
		}

		if metrics.TotalTime <= 0 {
			t.Errorf("Request() TotalTime should be > 0 even on error, got %v", metrics.TotalTime)
		}
	}
}

func TestAnthropicClient_Request_InvalidEndpoint(t *testing.T) {
	// 创建一个服务器，只接受正确的端点路径
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 如果请求的不是正确的端点，返回 404
		if !strings.HasSuffix(r.URL.Path, "/v1/messages") {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"error": {"type": "not_found_error", "message": "Invalid endpoint"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": "test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "test"}], "model": "claude-3", "usage": {"input_tokens": 1, "output_tokens": 1}}`)
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-key", "claude-3-sonnet-20240229")

	// 这应该成功，因为我们使用的是正确的端点
	_, err := client.Request("test prompt", false)
	if err != nil {
		t.Errorf("Request() should succeed with correct endpoint, got error: %v", err)
	}
}

func TestAnthropicClient_Request_MissingHeaders(t *testing.T) {
	// 创建一个严格检查请求头的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error": {"type": "authentication_error", "message": "Missing API key"}}`)
			return
		}
		if r.Header.Get("anthropic-version") == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": {"type": "invalid_request_error", "message": "Missing anthropic-version header"}}`)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error": {"type": "invalid_request_error", "message": "Invalid content type"}}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id": "test", "type": "message", "role": "assistant", "content": [{"type": "text", "text": "test"}], "model": "claude-3", "usage": {"input_tokens": 1, "output_tokens": 1}}`)
	}))
	defer server.Close()

	client := NewAnthropicClient(server.URL, "test-key", "claude-3-sonnet-20240229")

	// 这应该成功，因为我们的客户端发送了正确的请求头
	_, err := client.Request("test prompt", false)
	if err != nil {
		t.Errorf("Request() should succeed with correct headers, got error: %v", err)
	}
}

func TestAnthropicClient_Request_NetworkError(t *testing.T) {
	// 使用一个无效的地址来模拟网络错误
	client := NewAnthropicClient("http://invalid-host-that-does-not-exist.example", "test-key", "claude-3-sonnet-20240229")

	metrics, err := client.Request("test prompt", false)

	// 应该返回错误
	if err == nil {
		t.Error("Request() should return error for network error")
	}

	// 但应该返回包含错误信息的 metrics
	if metrics == nil {
		t.Error("Request() should return metrics even on network error")
	}

	if metrics != nil {
		if metrics.CompletionTokens != 0 {
			t.Errorf("Request() CompletionTokens should be 0 on network error, got %v", metrics.CompletionTokens)
		}

		if !strings.Contains(metrics.ErrorMessage, "Network error:") {
			t.Errorf("Request() ErrorMessage should contain 'Network error:', got %v", metrics.ErrorMessage)
		}

		if metrics.TotalTime <= 0 {
			t.Errorf("Request() TotalTime should be > 0 even on network error, got %v", metrics.TotalTime)
		}
	}
}

func TestAnthropicClient_Request_InvalidURL(t *testing.T) {
	// 使用一个格式错误的 URL
	client := NewAnthropicClient("://invalid-url", "test-key", "claude-3-sonnet-20240229")

	metrics, err := client.Request("test prompt", false)

	// 应该返回错误
	if err == nil {
		t.Error("Request() should return error for invalid URL")
	}

	// 但应该返回包含错误信息的 metrics
	if metrics == nil {
		t.Error("Request() should return metrics even on invalid URL error")
	}

	if metrics != nil {
		if !strings.Contains(metrics.ErrorMessage, "Request creation error:") {
			t.Errorf("Request() ErrorMessage should contain 'Request creation error:', got %v", metrics.ErrorMessage)
		}
	}
}
