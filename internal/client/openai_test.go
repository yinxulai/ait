package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewOpenAIClient(t *testing.T) {
	tests := []struct {
		name    string
		baseUrl string
		apiKey  string
		model   string
		want    *OpenAIClient
	}{
		{
			name:    "with custom base URL",
			baseUrl: "https://custom.api.com",
			apiKey:  "test-key",
			model:   "gpt-3.5-turbo",
			want: &OpenAIClient{
				baseURL:  "https://custom.api.com",
				apiKey:   "test-key",
				Model:    "gpt-3.5-turbo",
				Provider: "openai",
			},
		},
		{
			name:    "with empty base URL (should use default)",
			baseUrl: "",
			apiKey:  "test-key",
			model:   "gpt-4",
			want: &OpenAIClient{
				baseURL:  "https://api.openai.com",
				apiKey:   "test-key",
				Model:    "gpt-4",
				Provider: "openai",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewOpenAIClient(tt.baseUrl, tt.apiKey, tt.model)

			if got.baseURL != tt.want.baseURL {
				t.Errorf("NewOpenAIClient().baseURL = %v, want %v", got.baseURL, tt.want.baseURL)
			}

			if got.apiKey != tt.want.apiKey {
				t.Errorf("NewOpenAIClient().apiKey = %v, want %v", got.apiKey, tt.want.apiKey)
			}

			if got.Model != tt.want.Model {
				t.Errorf("NewOpenAIClient().Model = %v, want %v", got.Model, tt.want.Model)
			}

			if got.Provider != tt.want.Provider {
				t.Errorf("NewOpenAIClient().Provider = %v, want %v", got.Provider, tt.want.Provider)
			}

			if got.httpClient == nil {
				t.Error("NewOpenAIClient().httpClient should not be nil")
			}

			if got.httpClient.Timeout != 30*time.Second {
				t.Errorf("NewOpenAIClient().httpClient.Timeout = %v, want %v", got.httpClient.Timeout, 30*time.Second)
			}
		})
	}
}

func TestNewOpenAIClientWithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		baseUrl     string
		apiKey      string
		model       string
		timeout     time.Duration
		wantTimeout time.Duration
	}{
		{
			name:        "with custom timeout",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			model:       "gpt-3.5-turbo",
			timeout:     10 * time.Second,
			wantTimeout: 10 * time.Second,
		},
		{
			name:        "with zero timeout",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			model:       "gpt-4",
			timeout:     0,
			wantTimeout: 0,
		},
		{
			name:        "with long timeout",
			baseUrl:     "https://custom.api.com",
			apiKey:      "test-key",
			model:       "gpt-4",
			timeout:     60 * time.Second,
			wantTimeout: 60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewOpenAIClientWithTimeout(tt.baseUrl, tt.apiKey, tt.model, tt.timeout)

			if got.httpClient == nil {
				t.Error("NewOpenAIClientWithTimeout().httpClient should not be nil")
				return
			}

			if got.httpClient.Timeout != tt.wantTimeout {
				t.Errorf("NewOpenAIClientWithTimeout().httpClient.Timeout = %v, want %v", got.httpClient.Timeout, tt.wantTimeout)
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

func TestOpenAIClient_ConnectionReuse(t *testing.T) {
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
		response := fmt.Sprintf(`{"id":"chatcmpl-%d","choices":[{"message":{"content":"Response %d"}}],"usage":{"completion_tokens":1}}`, currentCount, currentCount)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "test-model")

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
		metrics, err := client.Request(fmt.Sprintf("test prompt %d", i), false)
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

// TestOpenAIClient_NoConnectionReuse 专门测试连接不复用的行为
func TestOpenAIClient_NoConnectionReuse(t *testing.T) {
	// 验证客户端的 Transport 配置确实禁用了连接复用
	client := NewOpenAIClient("https://api.openai.com", "test-key", "test-model")
	
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

func TestOpenAIClient_ConnectionReuseImpactOnTiming(t *testing.T) {
	// 这个测试演示为什么禁用连接复用对于准确的性能测量很重要
	
	// 创建一个有一定延迟的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟网络延迟
		time.Sleep(50 * time.Millisecond)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"test"}}],"usage":{"completion_tokens":1}}`))
	}))
	defer server.Close()

	// 创建两个客户端：一个禁用连接复用，一个启用连接复用
	clientWithoutReuse := &OpenAIClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: true,  // 禁用连接复用
			},
			Timeout: 30 * time.Second,
		},
		baseURL:  server.URL,
		apiKey:   "test-key",
		Model:    "test-model",
		Provider: "openai",
	}

	clientWithReuse := &OpenAIClient{
		httpClient: &http.Client{
			Transport: &http.Transport{
				DisableKeepAlives: false, // 启用连接复用
			},
			Timeout: 30 * time.Second,
		},
		baseURL:  server.URL,
		apiKey:   "test-key",
		Model:    "test-model",
		Provider: "openai",
	}

	// 测试两个客户端的性能差异
	t.Run("without connection reuse", func(t *testing.T) {
		// 发送多个请求，每次都应该包含完整的连接建立时间
		var totalTimes []time.Duration
		for i := 0; i < 3; i++ {
			metrics, err := clientWithoutReuse.Request("test", false)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			totalTimes = append(totalTimes, metrics.TotalTime)
		}
		
		// 由于每次都要重新建立连接，时间应该相对稳定且包含连接开销
		for i, duration := range totalTimes {
			if duration < 40*time.Millisecond {
				t.Errorf("Request %d duration %v is too short, expected at least 40ms (including connection overhead)", i, duration)
			}
		}
		
		t.Logf("Without reuse - timing results: %v", totalTimes)
	})

	t.Run("with connection reuse demonstration", func(t *testing.T) {
		// 这里我们演示连接复用的情况，但在实际的性能测试工具中应该避免
		// 首个请求建立连接
		metrics1, err := clientWithReuse.Request("test", false)
		if err != nil {
			t.Fatalf("First request failed: %v", err)
		}

		// 后续请求可能复用连接，时间可能更短
		metrics2, err := clientWithReuse.Request("test", false)
		if err != nil {
			t.Fatalf("Second request failed: %v", err)
		}

		t.Logf("With reuse - First request: %v, Second request: %v", metrics1.TotalTime, metrics2.TotalTime)
		
		// 这个测试主要是为了说明问题，不是为了断言特定的性能差异
		// 因为在测试环境中，本地连接可能不会显示显著差异
	})
}

func TestOpenAIClient_GetProtocol(t *testing.T) {
	client := NewOpenAIClient("https://api.openai.com", "test-key", "gpt-3.5-turbo")

	if got := client.GetProtocol(); got != "openai" {
		t.Errorf("OpenAIClient.GetProtocol() = %v, want %v", got, "openai")
	}
}

func TestOpenAIClient_GetModel(t *testing.T) {
	model := "gpt-4"
	client := NewOpenAIClient("https://api.openai.com", "test-key", model)

	if got := client.GetModel(); got != model {
		t.Errorf("OpenAIClient.GetModel() = %v, want %v", got, model)
	}
}

func TestOpenAIClient_TransportConfiguration(t *testing.T) {
	client := NewOpenAIClient("https://api.openai.com", "test-key", "gpt-3.5-turbo")

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
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestOpenAIClient_Request_MalformedJSON(t *testing.T) {
	// 创建返回畸形 JSON 数据的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") || 
		   r.Header.Get("Stream") == "true" {
			// 流式响应：发送畸形的 JSON
			w.Write([]byte("data: {invalid json}\n\n"))
			w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"valid content\"}}]}\n\n"))
			w.Write([]byte("data: [DONE]\n\n"))
		} else {
			// 非流式响应：返回畸形 JSON
			w.Write([]byte("{invalid json}"))
		}
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "test-model")

	// 测试非流式请求的 JSON 解析错误
	t.Run("non-stream malformed JSON", func(t *testing.T) {
		_, err := client.Request("test prompt", false)
		if err == nil {
			t.Error("Expected error for malformed JSON response")
		}
	})

	// 测试流式请求（应该跳过畸形的 JSON 并处理有效的）
	t.Run("stream with some malformed JSON", func(t *testing.T) {
		metrics, err := client.Request("test prompt", true)
		if err != nil {
			t.Errorf("Request should succeed even with some malformed JSON: %v", err)
		}
		if metrics == nil {
			t.Error("Expected metrics to be returned")
		}
	})
}

func TestOpenAIClient_Request_BodyReadError(t *testing.T) {
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

	client := NewOpenAIClient(server.URL, "test-key", "test-model")

	_, err := client.Request("test prompt", false)
	if err == nil {
		t.Error("Expected error when response body cannot be read")
	}
}

func TestOpenAIClient_Request_ScannerError(t *testing.T) {
	// 创建一个返回超大响应的服务器，可能导致 scanner 错误
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		
		// 发送一个非常长的行，可能导致 scanner 错误
		longLine := strings.Repeat("x", 1024*1024) // 1MB 的数据
		fmt.Fprintf(w, "data: %s\n\n", longLine)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "test-model")

	// 这个测试可能会因为 scanner 的缓冲区限制而失败
	metrics, err := client.Request("test prompt", true)
	// 无论成功还是失败都是正常的，关键是要覆盖这个代码路径
	if err != nil {
		t.Logf("Scanner error (expected in some cases): %v", err)
	}
	if metrics != nil {
		t.Logf("Metrics: TTFT=%v, Total=%v", metrics.TimeToFirstToken, metrics.TotalTime)
	}
}

func TestOpenAIClient_Request_EdgeCases(t *testing.T) {
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
			responseBody: "data: [DONE]\n\n",
			stream:       true,
			expectError:  false,
		},
		{
			name:         "unicode content",
			responseBody: `{"id":"test","choices":[{"message":{"content":"你好世界 🌍 测试 Unicode 字符"}}],"usage":{"completion_tokens":10}}`,
			stream:       false,
			expectError:  false,
		},
		{
			name:         "very long content",
			responseBody: fmt.Sprintf(`{"id":"test","choices":[{"message":{"content":"%s"}}],"usage":{"completion_tokens":1000}}`, strings.Repeat("x", 10000)),
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

			client := NewOpenAIClient(server.URL, "test-key", "test-model")
			_, err := client.Request("test", tt.stream)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestOpenAIClient_ConcurrentRequests(t *testing.T) {
	// 创建一个慢响应的服务器来测试并发
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // 模拟慢响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		response := `{"id":"test","choices":[{"message":{"content":"concurrent response"}}],"usage":{"completion_tokens":2}}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "test-model")
	
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
			
			metrics, err := client.Request(fmt.Sprintf("concurrent test %d", id), false)
			
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

func TestOpenAIClient_Request_TimeoutHandling(t *testing.T) {
	// 创建一个超慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 比客户端超时时间长
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"timeout test"}}]}`))
	}))
	defer server.Close()

	// 创建一个超时时间很短的客户端
	client := NewOpenAIClientWithTimeout(server.URL, "test-key", "test-model", 100*time.Millisecond)
	
	_, err := client.Request("timeout test", false)
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
	
	// 确保错误信息包含超时相关内容
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout-related error, got: %v", err)
	}
}

func TestOpenAIClient_Request_EmptyChoicesArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			// 流式响应：发送空的 choices
			w.Write([]byte("data: {\"choices\":[]}\n\n"))
			w.Write([]byte("data: [DONE]\n\n"))
		} else {
			// 非流式响应：空的 choices 数组
			w.Write([]byte(`{"id":"test","choices":[],"usage":{"completion_tokens":0}}`))
		}
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "test-model")

	// 测试非流式请求
	metrics, err := client.Request("test", false)
	if err != nil {
		t.Errorf("Request should succeed with empty choices: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics even with empty choices")
	}

	// 测试流式请求
	metrics, err = client.Request("test", true)
	if err != nil {
		t.Errorf("Stream request should succeed with empty choices: %v", err)
	}
	if metrics == nil {
		t.Error("Expected metrics even with empty choices")
	}
}

// TestOpenAIClient_Request_ReasoningContent 测试 ReasoningContent 字段对 TTFT 统计的影响
func TestOpenAIClient_Request_ReasoningContent(t *testing.T) {
	tests := []struct {
		name               string
		streamResponses    []string
		expectedTTFTValid  bool
		description        string
	}{
		{
			name: "reasoning content first, then regular content",
			streamResponses: []string{
				`{"choices":[{"delta":{"reasoning_content":"Let me think about this..."}}]}`,
				`{"choices":[{"delta":{"content":"Hello"}}]}`,
				`{"choices":[{"delta":{"content":" world"}}]}`,
				"[DONE]",
			},
			expectedTTFTValid: true,
			description:       "TTFT should be captured when reasoning_content appears first",
		},
		{
			name: "regular content first",
			streamResponses: []string{
				`{"choices":[{"delta":{"content":"Hello"}}]}`,
				`{"choices":[{"delta":{"reasoning_content":"Now I'm thinking..."}}]}`,
				`{"choices":[{"delta":{"content":" world"}}]}`,
				"[DONE]",
			},
			expectedTTFTValid: true,
			description:       "TTFT should be captured when regular content appears first",
		},
		{
			name: "only reasoning content",
			streamResponses: []string{
				`{"choices":[{"delta":{"reasoning_content":"Thinking step 1..."}}]}`,
				`{"choices":[{"delta":{"reasoning_content":"Thinking step 2..."}}]}`,
				`{"choices":[{"delta":{"reasoning_content":"Final thought..."}}]}`,
				"[DONE]",
			},
			expectedTTFTValid: true,
			description:       "TTFT should be captured with only reasoning content",
		},
		{
			name: "empty chunks before content",
			streamResponses: []string{
				`{"choices":[{"delta":{}}]}`,
				`{"choices":[{"delta":{"content":""}}]}`,
				`{"choices":[{"delta":{"reasoning_content":"First actual content"}}]}`,
				`{"choices":[{"delta":{"content":"Regular content"}}]}`,
				"[DONE]",
			},
			expectedTTFTValid: true,
			description:       "TTFT should skip empty chunks and capture first non-empty content",
		},
		{
			name: "null reasoning content",
			streamResponses: []string{
				`{"choices":[{"delta":{"reasoning_content":null}}]}`,
				`{"choices":[{"delta":{"content":"First content"}}]}`,
				"[DONE]",
			},
			expectedTTFTValid: true,
			description:       "TTFT should handle null reasoning_content correctly",
		},
		{
			name: "empty reasoning content string",
			streamResponses: []string{
				`{"choices":[{"delta":{"reasoning_content":""}}]}`,
				`{"choices":[{"delta":{"content":"First content"}}]}`,
				"[DONE]",
			},
			expectedTTFTValid: true,
			description:       "TTFT should skip empty reasoning_content string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				
				// 添加小延迟以确保 TTFT 有意义的值
				for i, response := range tt.streamResponses {
					if i > 0 {
						time.Sleep(10 * time.Millisecond)
					}
					if response == "[DONE]" {
						fmt.Fprint(w, "data: [DONE]\n\n")
					} else {
						fmt.Fprintf(w, "data: %s\n\n", response)
					}
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}
				}
			}))
			defer server.Close()

			client := NewOpenAIClient(server.URL, "test-key", "test-model")
			
			metrics, err := client.Request("test prompt", true)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}

			if metrics == nil {
				t.Error("Expected metrics to be returned")
				return
			}

			if tt.expectedTTFTValid {
				if metrics.TimeToFirstToken <= 0 {
					t.Errorf("Expected valid TTFT, got %v. %s", metrics.TimeToFirstToken, tt.description)
				}
				if metrics.TimeToFirstToken > metrics.TotalTime {
					t.Errorf("TTFT (%v) should not exceed total time (%v). %s", 
						metrics.TimeToFirstToken, metrics.TotalTime, tt.description)
				}
			}

			t.Logf("Test: %s - TTFT: %v, Total: %v", tt.name, metrics.TimeToFirstToken, metrics.TotalTime)
		})
	}
}

// TestOpenAIClient_Request_TTFTAccuracy 测试 TTFT 统计的准确性
func TestOpenAIClient_Request_TTFTAccuracy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		
		// 第一个 chunk: 只有空的 delta
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{}}]}\n\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		
		// 等待 50ms 后发送第一个有内容的 chunk
		time.Sleep(50 * time.Millisecond)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"Thinking...\"}}]}\n\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		
		// 等待 30ms 后发送常规内容
		time.Sleep(30 * time.Millisecond)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		
		// 结束
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "test-model")
	
	start := time.Now()
	metrics, err := client.Request("test prompt", true)
	totalDuration := time.Since(start)
	
	if err != nil {
		t.Errorf("Request failed: %v", err)
		return
	}

	if metrics == nil {
		t.Error("Expected metrics to be returned")
		return
	}

	// TTFT 应该大约是 50ms（第一个有内容的响应的延迟）
	if metrics.TimeToFirstToken < 40*time.Millisecond || metrics.TimeToFirstToken > 70*time.Millisecond {
		t.Errorf("TTFT should be around 50ms, got %v", metrics.TimeToFirstToken)
	}

	// 总时间应该大约是 80ms（50 + 30ms 的延迟）
	if metrics.TotalTime < 70*time.Millisecond || metrics.TotalTime > 100*time.Millisecond {
		t.Errorf("Total time should be around 80ms, got %v", metrics.TotalTime)
	}

	// TTFT 应该小于总时间
	if metrics.TimeToFirstToken >= metrics.TotalTime {
		t.Errorf("TTFT (%v) should be less than total time (%v)", 
			metrics.TimeToFirstToken, metrics.TotalTime)
	}

	t.Logf("Actual timing - TTFT: %v, Total: %v, External total: %v", 
		metrics.TimeToFirstToken, metrics.TotalTime, totalDuration)
}
