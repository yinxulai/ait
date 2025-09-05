package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockServer 创建用于测试的模拟 HTTP 服务器
func createMockServer(responseDelay time.Duration, stream bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟延迟
		time.Sleep(responseDelay)

		w.Header().Set("Content-Type", "application/json")

		if stream {
			// 模拟流式响应
			w.Header().Set("Transfer-Encoding", "chunked")
			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}

			// 发送多个数据块
			for i := 0; i < 3; i++ {
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"chunk %d\"}}]}\n\n", i)
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			fmt.Fprint(w, "data: [DONE]\n\n")
		} else {
			// 模拟非流式响应
			response := `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-3.5-turbo",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hello! I'm a test response."
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 9,
					"completion_tokens": 12,
					"total_tokens": 21
				}
			}`
			fmt.Fprint(w, response)
		}
	}))
}

func TestOpenAIClient_Request_NonStream(t *testing.T) {
	server := createMockServer(100*time.Millisecond, false)
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "gpt-3.5-turbo")

	start := time.Now()
	metrics, err := client.Request("test prompt", false)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if metrics.TimeToFirstToken <= 0 {
		t.Errorf("Request() TimeToFirstToken should be > 0, got %v", metrics.TimeToFirstToken)
	}

	// 检查实际耗时是否合理（应该至少包含模拟的延迟）
	if elapsed < 100*time.Millisecond {
		t.Errorf("Request() actual time %v should be >= 100ms", elapsed)
	}
}

func TestOpenAIClient_Request_Stream(t *testing.T) {
	server := createMockServer(50*time.Millisecond, true)
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "gpt-3.5-turbo")

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
}

func TestOpenAIClient_Request_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewOpenAIClient(server.URL, "test-key", "gpt-3.5-turbo")

	_, err := client.Request("test prompt", false)

	if err == nil {
		t.Error("Request() should return error for server error")
	}
}

func TestOpenAIClient_Request_NetworkError(t *testing.T) {
	// 使用一个无效的地址来模拟网络错误
	client := NewOpenAIClient("http://invalid-host-that-does-not-exist.example", "test-key", "gpt-3.5-turbo")

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

func TestOpenAIClient_Request_StreamNetworkError(t *testing.T) {
	// 测试流式模式下的网络错误
	client := NewOpenAIClient("http://invalid-host-that-does-not-exist.example", "test-key", "gpt-3.5-turbo")

	metrics, err := client.Request("test prompt", true)

	// 应该返回错误
	if err == nil {
		t.Error("Request() should return error for network error in stream mode")
	}

	// 但应该返回包含错误信息的 metrics
	if metrics == nil {
		t.Error("Request() should return metrics even on network error in stream mode")
	}

	if metrics != nil {
		if !strings.Contains(metrics.ErrorMessage, "Network error:") {
			t.Errorf("Request() ErrorMessage should contain 'Network error:', got %v", metrics.ErrorMessage)
		}
	}
}

func TestOpenAIClient_Request_InvalidURL(t *testing.T) {
	// 使用一个格式错误的 URL
	client := NewOpenAIClient("://invalid-url", "test-key", "gpt-3.5-turbo")

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
