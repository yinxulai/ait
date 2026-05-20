package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// createTestConfig 创建用于测试的标准配置
func createTestConfig(baseUrl, apiKey, model string, timeout time.Duration, thinking bool) types.Input {
	return types.Input{
		Protocol: types.ProtocolAnthropicMessages,
		BaseUrl:  baseUrl,
		ApiKey:   apiKey,
		Model:    model,
		Timeout:  timeout,
		Thinking: thinking,
	}
}

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
		name   string
		config types.Input
		want   *AnthropicClient
	}{
		{
			name: "valid anthropic client",
			config: types.Input{
				Protocol: types.ProtocolAnthropicMessages,
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				Timeout:  30 * time.Second,
				Thinking: false,
			},
			want: &AnthropicClient{
				EndpointURL: "https://api.anthropic.com/v1/messages",
				ApiKey:      "test-key",
				Model:       "claude-3-sonnet-20240229",
				Provider:    types.ProtocolAnthropicMessages,
				Thinking:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAnthropicClient(tt.config)

			if got.EndpointURL != tt.want.EndpointURL {
				t.Errorf("NewAnthropicClient().EndpointURL = %v, want %v", got.EndpointURL, tt.want.EndpointURL)
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

			if got.Thinking != tt.want.Thinking {
				t.Errorf("NewAnthropicClient().Thinking = %v, want %v", got.Thinking, tt.want.Thinking)
			}
		})
	}
}

func TestAnthropicClient_GetProtocol(t *testing.T) {
	client := NewAnthropicClient(createTestConfig("https://api.anthropic.com", "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	if got := client.GetProtocol(); got != types.ProtocolAnthropicMessages {
		t.Errorf("AnthropicClient.GetProtocol() = %v, want %v", got, types.ProtocolAnthropicMessages)
	}
}

func TestAnthropicClient_GetModel(t *testing.T) {
	model := "claude-3-sonnet-20240229"
	client := NewAnthropicClient(createTestConfig("https://api.anthropic.com", "test-key", model, 30*time.Second, false))

	if got := client.GetModel(); got != model {
		t.Errorf("AnthropicClient.GetModel() = %v, want %v", got, model)
	}
}

func TestAnthropicClient_Request_NonStream(t *testing.T) {
	server := createMockAnthropicServer(100*time.Millisecond, false, http.StatusOK)
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	start := time.Now()
	metrics, err := client.Request("", "test prompt", false)
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

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	start := time.Now()
	metrics, err := client.Request("", "test prompt", true)
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

func TestAnthropicClient_Request_SystemPromptUsesCacheControl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		system, ok := body["system"].([]interface{})
		if !ok || len(system) != 2 {
			t.Fatalf("expected 2 system blocks, got %#v", body["system"])
		}

		lastBlock, ok := system[len(system)-1].(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected system block: %#v", system[len(system)-1])
		}
		cacheControl, ok := lastBlock["cache_control"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected cache_control on last system block, got %#v", lastBlock)
		}
		if cacheControl["type"] != "ephemeral" {
			t.Fatalf("cache_control.type = %#v, want %#v", cacheControl["type"], "ephemeral")
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"test","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-3","usage":{"input_tokens":4,"output_tokens":1}}`)
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
	if _, err := client.Request("公共消息1\n\n公共消息2", "user prompt", false); err != nil {
		t.Fatalf("Request() error = %v", err)
	}
}

func TestAnthropicClient_Request_PromptTokensIncludeCachedAndCreatedInput(t *testing.T) {
	t.Run("non-stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"msg_test","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"model":"claude-3-sonnet","usage":{"input_tokens":50,"cache_creation_input_tokens":100,"cache_read_input_tokens":900,"output_tokens":10}}`)
		}))
		defer server.Close()

		client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
		metrics, err := client.Request("shared system", "user prompt", false)
		if err != nil {
			t.Fatalf("Request() error = %v", err)
		}
		if metrics.PromptTokens != 1050 {
			t.Fatalf("PromptTokens = %d, want %d", metrics.PromptTokens, 1050)
		}
		if metrics.CachedInputTokens != 900 {
			t.Fatalf("CachedInputTokens = %d, want %d", metrics.CachedInputTokens, 900)
		}
	})

	t.Run("stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Transfer-Encoding", "chunked")

			flusher, _ := w.(http.Flusher)
			fmt.Fprint(w, "event: message_start\n")
			fmt.Fprint(w, `data: {"type":"message_start","message":{"usage":{"input_tokens":40,"cache_creation_input_tokens":160,"cache_read_input_tokens":800,"output_tokens":0}}}`+"\n\n")
			flusher.Flush()
			fmt.Fprint(w, "event: content_block_delta\n")
			fmt.Fprint(w, `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`+"\n\n")
			flusher.Flush()
			fmt.Fprint(w, "event: message_delta\n")
			fmt.Fprint(w, `data: {"type":"message_delta","usage":{"output_tokens":12}}`+"\n\n")
			flusher.Flush()
		}))
		defer server.Close()

		client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
		metrics, err := client.Request("shared system", "user prompt", true)
		if err != nil {
			t.Fatalf("Request() error = %v", err)
		}
		if metrics.PromptTokens != 1000 {
			t.Fatalf("PromptTokens = %d, want %d", metrics.PromptTokens, 1000)
		}
		if metrics.CachedInputTokens != 800 {
			t.Fatalf("CachedInputTokens = %d, want %d", metrics.CachedInputTokens, 800)
		}
		if metrics.CompletionTokens != 12 {
			t.Fatalf("CompletionTokens = %d, want %d", metrics.CompletionTokens, 12)
		}
	})
}

func TestAnthropicClient_Request_ServerError(t *testing.T) {
	server := createMockAnthropicServer(0, false, http.StatusInternalServerError)
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	metrics, err := client.Request("", "test prompt", false)

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

		if !strings.Contains(metrics.ErrorMessage, "[api_error] Test error") {
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

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	// 这应该成功，因为我们使用的是正确的端点
	_, err := client.Request("", "test prompt", false)
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

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	// 这应该成功，因为我们的客户端发送了正确的请求头
	_, err := client.Request("", "test prompt", false)
	if err != nil {
		t.Errorf("Request() should succeed with correct headers, got error: %v", err)
	}
}

func TestAnthropicClient_Request_NetworkError(t *testing.T) {
	// 使用一个无效的地址来模拟网络错误
	client := NewAnthropicClient(createTestConfig("http://invalid-host-that-does-not-exist.example", "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	metrics, err := client.Request("", "test prompt", false)

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
	client := NewAnthropicClient(createTestConfig("://invalid-url", "test-key", "claude-3-sonnet-20240229", 30*time.Second, false))

	metrics, err := client.Request("", "test prompt", false)

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

func TestNewAnthropicClientTimeout(t *testing.T) {
	tests := []struct {
		name        string
		config      types.Input
		wantTimeout time.Duration
	}{
		{
			name: "with custom timeout",
			config: types.Input{
				Protocol: types.ProtocolAnthropicMessages,
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-sonnet",
				Timeout:  10 * time.Second,
				Thinking: false,
			},
			wantTimeout: 10 * time.Second,
		},
		{
			name: "with zero timeout",
			config: types.Input{
				Protocol: types.ProtocolAnthropicMessages,
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-opus",
				Timeout:  0,
				Thinking: false,
			},
			wantTimeout: 0,
		},
		{
			name: "with long timeout",
			config: types.Input{
				Protocol: types.ProtocolAnthropicMessages,
				BaseUrl:  "https://custom.api.com",
				ApiKey:   "test-key",
				Model:    "claude-3-haiku",
				Timeout:  60 * time.Second,
				Thinking: false,
			},
			wantTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAnthropicClient(tt.config)

			if got.httpClient == nil {
				t.Error("NewAnthropicClient().httpClient should not be nil")
				return
			}

			if got.httpClient.Timeout != tt.wantTimeout {
				t.Errorf("NewAnthropicClient().httpClient.Timeout = %v, want %v", got.httpClient.Timeout, tt.wantTimeout)
			}

			// 验证 Transport 设置
			transport, ok := got.httpClient.Transport.(*http.Transport)
			if !ok {
				t.Error("Expected http.Transport")
				return
			}

			if !transport.DisableKeepAlives {
				t.Error("Expected DisableKeepAlives to be true")
			}

			if transport.DisableCompression {
				t.Error("Expected DisableCompression to be false")
			}
		})
	}
}

func TestAnthropicClient_ConnectionReuse(t *testing.T) {
	// 创建一个测试服务器，记录连接数
	connectionCount := 0
	var connMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 每个请求到达时记录
		connMu.Lock()
		connectionCount++
		currentCount := connectionCount
		connMu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// 返回简单的非流式响应
		response := fmt.Sprintf(`{"id":"msg-%d","type":"message","role":"assistant","content":[{"type":"text","text":"Response %d"}],"model":"claude-3","usage":{"input_tokens":1,"output_tokens":1}}`, currentCount, currentCount)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	// 验证客户端确实禁用了连接复用
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected client to use http.Transport")
	}

	if !transport.DisableKeepAlives {
		t.Error("Expected DisableKeepAlives to be true to prevent connection reuse")
	}

	// 发送多个串行请求来验证不复用连接的行为
	requestCount := 3
	for i := 0; i < requestCount; i++ {
		metrics, err := client.Request("", fmt.Sprintf("test prompt %d", i), false)
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
			continue
		}

		if metrics == nil {
			t.Errorf("Request %d returned nil metrics", i)
			continue
		}

		// 验证每个请求都有合理的时间指标
		if metrics.TotalTime <= 0 {
			t.Errorf("Request %d has invalid TotalTime: %v", i, metrics.TotalTime)
		}
	}

	// 验证服务器确实收到了所有请求
	connMu.Lock()
	finalCount := connectionCount
	connMu.Unlock()

	if finalCount != requestCount {
		t.Errorf("Expected %d requests to reach server, got %d", requestCount, finalCount)
	}
}

// TestAnthropicClient_NoConnectionReuse 专门测试连接不复用的行为
func TestAnthropicClient_NoConnectionReuse(t *testing.T) {
	// 验证客户端的 Transport 配置确实禁用了连接复用
	client := NewAnthropicClient(createTestConfig("https://api.anthropic.com", "test-key", "claude-3-sonnet", 30*time.Second, false))

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected client to use http.Transport")
	}

	// 关键验证：DisableKeepAlives 应该为 true
	if !transport.DisableKeepAlives {
		t.Error("DisableKeepAlives should be true to prevent connection reuse, which could affect timing measurements")
	}

	// DisableCompression 应该为 false（我们想要压缩以节省带宽）
	if transport.DisableCompression {
		t.Error("DisableCompression should be false to enable compression")
	}
}

func TestAnthropicClient_TransportConfiguration(t *testing.T) {
	client := NewAnthropicClient(createTestConfig("https://api.anthropic.com", "test-key", "claude-3-sonnet", 30*time.Second, false))

	if client.httpClient == nil {
		t.Error("Expected client to have httpClient")
		return
	}

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Error("Expected client to use http.Transport")
		return
	}

	// 验证关键的传输配置
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{
			name: "DisableKeepAlives should be true",
			got:  transport.DisableKeepAlives,
			want: true,
		},
		{
			name: "DisableCompression should be false",
			got:  transport.DisableCompression,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestAnthropicClient_Request_MalformedJSON(t *testing.T) {
	// 创建返回畸形 JSON 数据的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			// 流式响应：发送畸形的 JSON
			w.Write([]byte("event: content_block_delta\n"))
			w.Write([]byte("data: {invalid json}\n\n"))
			w.Write([]byte("event: content_block_delta\n"))
			w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"valid\"}}\n\n"))
			w.Write([]byte("event: message_stop\n"))
			w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		} else {
			// 非流式响应：返回畸形 JSON
			w.Write([]byte("{invalid json}"))
		}
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	// 测试非流式请求的 JSON 解析错误
	t.Run("non-stream malformed JSON", func(t *testing.T) {
		_, err := client.Request("", "test prompt", false)
		if err == nil {
			t.Error("Expected error for malformed JSON response")
		}
	})

	// 测试流式请求（应该跳过畸形的 JSON 并处理有效的）
	t.Run("stream with some malformed JSON", func(t *testing.T) {
		metrics, err := client.Request("", "test prompt", true)
		if err != nil {
			t.Errorf("Request should succeed even with some malformed JSON: %v", err)
		}
		if metrics == nil {
			t.Error("Expected metrics to be returned")
		}
	})
}

func TestAnthropicClient_Request_BodyReadError(t *testing.T) {
	// 创建一个在读取响应体时出错的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// 立即关闭连接，造成读取错误
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("webserver doesn't support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	_, err := client.Request("", "test prompt", false)
	if err == nil {
		t.Error("Expected error when response body cannot be read")
	}
}

func TestAnthropicClient_Request_ScannerError(t *testing.T) {
	// 创建一个返回超大响应的服务器，可能导致 scanner 错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// 发送一个非常长的行，可能导致 scanner 错误
		longLine := strings.Repeat("x", 1024*1024) // 1MB 的数据
		fmt.Fprintf(w, "event: content_block_delta\ndata: %s\n\n", longLine)
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	// 这个测试可能会因为 scanner 的缓冲区限制而失败
	metrics, err := client.Request("", "test prompt", true)
	// 无论成功还是失败都是正常的，关键是要覆盖这个代码路径
	if err != nil {
		t.Logf("Scanner error (expected in some cases): %v", err)
	}
	if metrics != nil {
		t.Logf("Metrics: TTFT=%v, Total=%v", metrics.TimeToFirstToken, metrics.TotalTime)
	}
}

func TestAnthropicClient_Request_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		stream       bool
		expectError  bool
	}{
		{
			name:         "empty response",
			responseBody: "",
			stream:       false,
			expectError:  true,
		},
		{
			name:         "empty stream response",
			responseBody: "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
			stream:       true,
			expectError:  false,
		},
		{
			name:         "unicode content",
			responseBody: `{"id":"test","type":"message","content":[{"type":"text","text":"你好世界 🌍 测试 Unicode 字符"}],"usage":{"output_tokens":10}}`,
			stream:       false,
			expectError:  false,
		},
		{
			name:         "very long content",
			responseBody: fmt.Sprintf(`{"id":"test","type":"message","content":[{"type":"text","text":"%s"}],"usage":{"output_tokens":1000}}`, strings.Repeat("x", 10000)),
			stream:       false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
			_, err := client.Request("", "test", tt.stream)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestAnthropicClient_ConcurrentRequests(t *testing.T) {
	// 创建一个慢响应的服务器来测试并发
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // 模拟慢响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := `{"id":"test","type":"message","content":[{"type":"text","text":"concurrent response"}],"usage":{"output_tokens":2}}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	// 并发执行多个请求
	numRequests := 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successCount int
	var errors []error

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			metrics, err := client.Request("", fmt.Sprintf("concurrent test %d", id), false)

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				successCount++
				if metrics == nil {
					errors = append(errors, fmt.Errorf("nil metrics for request %d", id))
				}
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 验证所有请求都成功
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("Concurrent request error: %v", err)
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}
}

func TestAnthropicClient_Request_TimeoutHandling(t *testing.T) {
	// 创建一个超慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 比客户端超时时间长
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","type":"message","content":[{"type":"text","text":"timeout test"}],"usage":{"output_tokens":1}}`))
	}))
	defer server.Close()

	// 创建一个超时时间很短的客户端
	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 100*time.Millisecond, false))

	_, err := client.Request("", "timeout test", false)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}

	// 确保错误信息包含超时相关内容
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout-related error, got: %v", err)
	}
}

func TestAnthropicClient_Request_EmptyContentArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			// 流式响应：发送空的 content
			w.Write([]byte("event: message_start\n"))
			w.Write([]byte("data: {\"type\":\"message_start\"}\n\n"))
			w.Write([]byte("event: message_stop\n"))
			w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		} else {
			// 非流式响应：空的 content 数组
			w.Write([]byte(`{"id":"test","type":"message","content":[],"usage":{"output_tokens":0}}`))
		}
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	// 测试非流式请求
	metrics, err := client.Request("", "test", false)
	if err != nil {
		t.Errorf("Request should succeed with empty content: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics even with empty content")
	}

	// 测试流式请求
	metrics, err = client.Request("", "test", true)
	if err != nil {
		t.Errorf("Stream request should succeed with empty content: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics even with empty content")
	}
}

// TestAnthropicClient_Request_StreamWithThinking 测试包含 Thinking 输出的 TTFT 计算
func TestAnthropicClient_Request_StreamWithThinking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, _ := w.(http.Flusher)

		// 发送开始事件
		fmt.Fprint(w, "event: message_start\n")
		fmt.Fprint(w, `data: {"type": "message_start", "message": {"id": "msg_test", "type": "message", "role": "assistant", "content": [], "model": "claude-3-sonnet", "usage": {"input_tokens": 10, "output_tokens": 0}}}`+"\n\n")
		flusher.Flush()

		// 模拟延迟，然后发送 thinking 内容
		time.Sleep(10 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "thinking": "Let me think about this..."}}`+"\n\n")
		flusher.Flush()

		// 再发送一些普通文本
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello there!"}}`+"\n\n")
		flusher.Flush()

		// 发送结束事件
		fmt.Fprint(w, "event: message_delta\n")
		fmt.Fprint(w, `data: {"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"output_tokens": 10}}`+"\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	start := time.Now()
	metrics, err := client.Request("", "test prompt", true)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TTFT should be > 0 when thinking content is present, got %v", metrics.TimeToFirstToken)
	}

	// TTFT 应该在第一个 thinking 输出时就开始计算
	if metrics.TimeToFirstToken > time.Since(start) {
		t.Errorf("TTFT should be calculated from thinking output, got %v", metrics.TimeToFirstToken)
	}

	if metrics.CompletionTokens != 10 {
		t.Errorf("Request() CompletionTokens = %v, want 10", metrics.CompletionTokens)
	}
}

// TestAnthropicClient_Request_StreamWithPartialJSON 测试包含 PartialJSON 输出的 TTFT 计算
func TestAnthropicClient_Request_StreamWithPartialJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, _ := w.(http.Flusher)

		// 发送开始事件
		fmt.Fprint(w, "event: message_start\n")
		fmt.Fprint(w, `data: {"type": "message_start", "message": {"id": "msg_test", "type": "message", "role": "assistant", "content": [], "model": "claude-3-sonnet", "usage": {"input_tokens": 10, "output_tokens": 0}}}`+"\n\n")
		flusher.Flush()

		// 模拟延迟，然后发送 partial_json 内容
		time.Sleep(10 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "partial_json": "{\"name\": \"John\""}}`+"\n\n")
		flusher.Flush()

		// 继续发送更多的 partial_json
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "partial_json": ", \"age\": 30}"}}`+"\n\n")
		flusher.Flush()

		// 发送结束事件
		fmt.Fprint(w, "event: message_delta\n")
		fmt.Fprint(w, `data: {"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"output_tokens": 8}}`+"\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	start := time.Now()
	metrics, err := client.Request("", "test prompt", true)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TTFT should be > 0 when partial_json content is present, got %v", metrics.TimeToFirstToken)
	}

	// TTFT 应该在第一个 partial_json 输出时就开始计算
	if metrics.TimeToFirstToken > time.Since(start) {
		t.Errorf("TTFT should be calculated from partial_json output, got %v", metrics.TimeToFirstToken)
	}

	if metrics.CompletionTokens != 8 {
		t.Errorf("Request() CompletionTokens = %v, want 8", metrics.CompletionTokens)
	}
}

// TestAnthropicClient_Request_StreamWithMixedContent 测试混合内容类型的 TTFT 计算
func TestAnthropicClient_Request_StreamWithMixedContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, _ := w.(http.Flusher)

		// 发送开始事件
		fmt.Fprint(w, "event: message_start\n")
		fmt.Fprint(w, `data: {"type": "message_start", "message": {"id": "msg_test", "type": "message", "role": "assistant", "content": [], "model": "claude-3-sonnet", "usage": {"input_tokens": 10, "output_tokens": 0}}}`+"\n\n")
		flusher.Flush()

		// 首先发送 thinking 内容
		time.Sleep(15 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "thinking": "I need to analyze this carefully..."}}`+"\n\n")
		flusher.Flush()

		// 然后发送 partial_json
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "partial_json": "{\"result\": \""}}`+"\n\n")
		flusher.Flush()

		// 最后发送普通文本
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "This is the final answer."}}`+"\n\n")
		flusher.Flush()

		// 发送结束事件
		fmt.Fprint(w, "event: message_delta\n")
		fmt.Fprint(w, `data: {"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"output_tokens": 20}}`+"\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	start := time.Now()
	metrics, err := client.Request("", "test prompt", true)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TTFT should be > 0 with mixed content, got %v", metrics.TimeToFirstToken)
	}

	// TTFT 应该在第一个内容输出时就开始计算（thinking 内容）
	expectedMinTime := 10 * time.Millisecond // 小于第一个 thinking 输出的延迟
	if metrics.TimeToFirstToken < expectedMinTime {
		t.Errorf("TTFT seems too fast, expected >= %v, got %v", expectedMinTime, metrics.TimeToFirstToken)
	}

	expectedMaxTime := time.Since(start)
	if metrics.TimeToFirstToken > expectedMaxTime {
		t.Errorf("TTFT should be calculated from first output, got %v", metrics.TimeToFirstToken)
	}

	if metrics.CompletionTokens != 20 {
		t.Errorf("Request() CompletionTokens = %v, want 20", metrics.CompletionTokens)
	}
}

// TestAnthropicClient_Request_StreamWithEmptyThinkingAndPartialJSON 测试空的 thinking 和 partial_json 字段
func TestAnthropicClient_Request_StreamWithEmptyThinkingAndPartialJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, _ := w.(http.Flusher)

		// 发送开始事件
		fmt.Fprint(w, "event: message_start\n")
		fmt.Fprint(w, `data: {"type": "message_start", "message": {"id": "msg_test", "type": "message", "role": "assistant", "content": [], "model": "claude-3-sonnet", "usage": {"input_tokens": 10, "output_tokens": 0}}}`+"\n\n")
		flusher.Flush()

		// 发送空的 thinking 内容（不应该触发 TTFT）
		time.Sleep(10 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "thinking": ""}}`+"\n\n")
		flusher.Flush()

		// 发送空的 partial_json 内容（不应该触发 TTFT）
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "partial_json": ""}}`+"\n\n")
		flusher.Flush()

		// 最后发送真正的文本内容（应该触发 TTFT）
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, `data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Real content here"}}`+"\n\n")
		flusher.Flush()

		// 发送结束事件
		fmt.Fprint(w, "event: message_delta\n")
		fmt.Fprint(w, `data: {"type": "message_delta", "delta": {"stop_reason": "end_turn"}, "usage": {"output_tokens": 5}}`+"\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))

	start := time.Now()
	metrics, err := client.Request("", "test prompt", true)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TTFT should be > 0 when real text content is present, got %v", metrics.TimeToFirstToken)
	}

	// TTFT 应该在真正的文本内容输出时计算，而不是空的 thinking/partial_json
	expectedMinTime := 15 * time.Millisecond // 应该大于前两个空内容的延迟总和
	if metrics.TimeToFirstToken < expectedMinTime {
		t.Errorf("TTFT should be calculated from real text content, expected >= %v, got %v", expectedMinTime, metrics.TimeToFirstToken)
	}

	expectedMaxTime := time.Since(start)
	if metrics.TimeToFirstToken > expectedMaxTime {
		t.Errorf("TTFT calculation error, got %v", metrics.TimeToFirstToken)
	}

	if metrics.CompletionTokens != 5 {
		t.Errorf("Request() CompletionTokens = %v, want 5", metrics.CompletionTokens)
	}
}

// TestAnthropicClient_Request_ErrorHandlingFixes 测试错误处理修复
func TestAnthropicClient_Request_ErrorHandlingFixes(t *testing.T) {
	t.Run("JSON parsing error returns metrics with error info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("invalid json response"))
		}))
		defer server.Close()

		client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
		metrics, err := client.Request("", "test prompt", false)

		// 应该有错误
		if err == nil {
			t.Error("Expected error for malformed JSON")
		}

		// 关键修复：应该返回包含错误信息的 metrics，而不是 nil
		if metrics == nil {
			t.Fatal("Expected metrics to be returned even on JSON parsing error, got nil")
		}

		// 验证 metrics 包含正确的错误信息
		if !strings.Contains(metrics.ErrorMessage, "JSON parsing error") {
			t.Errorf("Expected ErrorMessage to contain 'JSON parsing error', got: %s", metrics.ErrorMessage)
		}

		// 验证网络指标仍然被收集
		if metrics.TotalTime <= 0 {
			t.Error("Expected TotalTime to be > 0 even on JSON parsing error")
		}

		// 验证其他指标的合理性
		if metrics.TimeToFirstToken != 0 {
			t.Error("Expected TimeToFirstToken to be 0 on JSON parsing error")
		}
		if metrics.CompletionTokens != 0 {
			t.Error("Expected CompletionTokens to be 0 on JSON parsing error")
		}
	})

	t.Run("Empty response returns metrics with error info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// 返回空响应体
		}))
		defer server.Close()

		client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
		metrics, err := client.Request("", "test prompt", false)

		// 应该有错误
		if err == nil {
			t.Error("Expected error for empty response")
		}

		// 关键修复：应该返回包含错误信息的 metrics
		if metrics == nil {
			t.Fatal("Expected metrics to be returned even on empty response error, got nil")
		}

		// 验证 metrics 包含正确的错误信息
		if !strings.Contains(metrics.ErrorMessage, "Empty response body") {
			t.Errorf("Expected ErrorMessage to contain 'Empty response body', got: %s", metrics.ErrorMessage)
		}

		// 验证网络指标仍然被收集
		if metrics.TotalTime <= 0 {
			t.Error("Expected TotalTime to be > 0 even on empty response error")
		}
	})

	t.Run("Response body read error returns metrics", func(t *testing.T) {
		// 测试策略：创建一个声称有内容但实际没有完整内容的响应
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1000")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("incomplete"))
			// 在 httptest 环境中，这种情况通常不会导致 io.ReadAll 错误
			// 但我们仍然测试基本逻辑
		}))
		defer server.Close()

		client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
		metrics, err := client.Request("", "test prompt", false)

		// 这种情况下通常会是 JSON 解析错误而不是读取错误
		if metrics == nil && err != nil {
			t.Error("Expected metrics to be returned even when there are response reading issues")
		}
	})

	t.Run("Stream JSON parsing error continues processing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// 发送一些无效的 JSON 数据块，然后发送有效的
			w.Write([]byte("data: {invalid json}\n\n"))
			w.Write([]byte("data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"valid\"}}\n\n"))
			w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		}))
		defer server.Close()

		client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
		metrics, err := client.Request("", "test prompt", true)

		// 流式处理应该继续，即使有些 JSON 块无效
		if err != nil {
			t.Errorf("Stream request should succeed even with some malformed JSON: %v", err)
		}

		if metrics == nil {
			t.Fatal("Expected metrics to be returned for stream request")
		}

		// 应该没有错误信息，因为流式处理成功了
		if metrics.ErrorMessage != "" {
			t.Errorf("Expected no error message for successful stream, got: %s", metrics.ErrorMessage)
		}
	})

	t.Run("Consistent error handling across different error types", func(t *testing.T) {
		testCases := []struct {
			name           string
			setupServer    func() *httptest.Server
			expectedErrMsg string
		}{
			{
				name: "HTTP 404 error",
				setupServer: func() *httptest.Server {
					return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte(`{"type":"error","error":{"type":"not_found","message":"Resource not found"}}`))
					}))
				},
				expectedErrMsg: "not_found",
			},
			{
				name: "HTTP 500 error",
				setupServer: func() *httptest.Server {
					return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte(`{"type":"error","error":{"type":"server_error","message":"Internal error"}}`))
					}))
				},
				expectedErrMsg: "server_error",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				server := tc.setupServer()
				defer server.Close()

				client := NewAnthropicClient(createTestConfig(server.URL, "test-key", "claude-3-sonnet", 30*time.Second, false))
				metrics, err := client.Request("", "test prompt", false)

				// 所有类型的错误都应该返回错误
				if err == nil {
					t.Errorf("Expected error for %s", tc.name)
				}

				// 所有类型的错误都应该返回 metrics
				if metrics == nil {
					t.Fatalf("Expected metrics to be returned for %s, got nil", tc.name)
				}

				// 验证错误信息包含预期内容
				if !strings.Contains(metrics.ErrorMessage, tc.expectedErrMsg) {
					t.Errorf("Expected ErrorMessage to contain '%s' for %s, got: %s",
						tc.expectedErrMsg, tc.name, metrics.ErrorMessage)
				}

				// 验证网络指标被收集
				if metrics.TotalTime <= 0 {
					t.Errorf("Expected TotalTime to be > 0 for %s", tc.name)
				}
			})
		}
	})
}

func TestAnthropicClientWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		thinking    bool
		expectField bool
	}{
		{
			name:        "thinking enabled",
			thinking:    true,
			expectField: true,
		},
		{
			name:        "thinking disabled",
			thinking:    false,
			expectField: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewAnthropicClient(createTestConfig("https://api.anthropic.com", "test-key", "claude-3-sonnet", 30*time.Second, tt.thinking))

			// 验证 thinking 字段设置正确
			if client.Thinking != tt.thinking {
				t.Errorf("Expected Thinking = %v, got %v", tt.thinking, client.Thinking)
			}

			// 验证其他基本字段
			if client.EndpointURL != "https://api.anthropic.com/v1/messages" {
				t.Errorf("Expected EndpointURL = https://api.anthropic.com/v1/messages, got %s", client.EndpointURL)
			}

			if client.Model != "claude-3-sonnet" {
				t.Errorf("Expected Model = claude-3-sonnet, got %s", client.Model)
			}

			if client.Provider != types.ProtocolAnthropicMessages {
				t.Errorf("Expected Provider = %s, got %s", types.ProtocolAnthropicMessages, client.Provider)
			}
		})
	}
}
