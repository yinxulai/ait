package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	duration, err := client.Request("test prompt", false)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if duration <= 0 {
		t.Errorf("Request() duration should be > 0, got %v", duration)
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
	ttft, err := client.Request("test prompt", true)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("Request() error = %v", err)
	}

	if ttft <= 0 {
		t.Errorf("Request() TTFT should be > 0, got %v", ttft)
	}

	// TTFT 应该小于总耗时（因为我们在流中有多个块）
	if ttft > elapsed {
		t.Errorf("TTFT %v should be <= total elapsed time %v", ttft, elapsed)
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
