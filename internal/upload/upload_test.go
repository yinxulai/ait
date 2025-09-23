package upload

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/types"
)

func TestNew(t *testing.T) {
	uploader := New()

	// 测试基本结构是否正确初始化
	if uploader == nil {
		t.Error("Expected New() to return non-nil Uploader")
		return
	}

	if uploader.client == nil {
		t.Error("Expected HTTP client to be initialized")
		return
	}

	if uploader.client.Timeout != time.Second*3 {
		t.Errorf("Expected client timeout to be 3 seconds, got %v", uploader.client.Timeout)
	}

	// 验证配置值是从全局变量中获取的
	if uploader.baseURL != UploadBaseURL {
		t.Errorf("Expected baseURL to match UploadBaseURL, got '%s', expected '%s'", uploader.baseURL, UploadBaseURL)
	}

	if uploader.authToken != UploadAuthToken {
		t.Errorf("Expected authToken to match UploadAuthToken, got '%s', expected '%s'", uploader.authToken, UploadAuthToken)
	}

	if uploader.userAgent != UploadUserAgent {
		t.Errorf("Expected userAgent to match UploadUserAgent, got '%s', expected '%s'", uploader.userAgent, UploadUserAgent)
	}

	// 测试 HTTP client 配置
	transport, ok := uploader.client.Transport.(*http.Transport)
	if !ok {
		t.Error("Expected HTTP transport to be *http.Transport")
	} else {
		if transport.MaxIdleConns != 10 {
			t.Errorf("Expected MaxIdleConns to be 10, got %d", transport.MaxIdleConns)
		}
		if transport.MaxIdleConnsPerHost != 5 {
			t.Errorf("Expected MaxIdleConnsPerHost to be 5, got %d", transport.MaxIdleConnsPerHost)
		}
		if transport.IdleConnTimeout != 30*time.Second {
			t.Errorf("Expected IdleConnTimeout to be 30s, got %v", transport.IdleConnTimeout)
		}
	}
}

func TestUploader_isValidURL(t *testing.T) {
	uploader := &Uploader{}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid https URL",
			url:      "https://api.example.com",
			expected: true,
		},
		{
			name:     "valid http URL",
			url:      "http://api.example.com",
			expected: true,
		},
		{
			name:     "valid URL with path",
			url:      "https://api.example.com/v1/upload",
			expected: true,
		},
		{
			name:     "valid URL with port",
			url:      "https://api.example.com:8080",
			expected: true,
		},
		{
			name:     "empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "null string",
			url:      "null",
			expected: false,
		},
		{
			name:     "invalid scheme - ftp",
			url:      "ftp://example.com",
			expected: false,
		},
		{
			name:     "no scheme",
			url:      "api.example.com",
			expected: false,
		},
		{
			name:     "no host",
			url:      "https://",
			expected: false,
		},
		{
			name:     "malformed URL",
			url:      "https://[invalid-host",
			expected: false,
		},
		{
			name:     "file scheme",
			url:      "file:///path/to/file",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uploader.isValidURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidURL(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestUploader_convertResponseMetricsToUploadItem(t *testing.T) {
	uploader := &Uploader{
		userAgent: "test-agent",
	}

	tests := []struct {
		name     string
		taskID   string
		metrics  *client.ResponseMetrics
		input    types.Input
		expected ReportUploadItem
	}{
		{
			name:   "successful response",
			taskID: "task-123",
			metrics: &client.ResponseMetrics{
				PromptTokens:      100,
				CompletionTokens:  50,
				TotalTime:         time.Millisecond * 1500,
				DNSTime:           time.Millisecond * 50,
				ConnectTime:       time.Millisecond * 100,
				TLSHandshakeTime:  time.Millisecond * 200,
				TimeToFirstToken:  time.Millisecond * 800,
				TargetIP:          "1.2.3.4",
				ErrorMessage:      "",
			},
			input: types.Input{
				Protocol: "openai",
				BaseUrl:  "https://api.example.com",
				Model:    "gpt-3.5-turbo",
			},
			expected: ReportUploadItem{
				TaskID:               "task-123",
				ModelKey:             nil,
				Reporter:             "test-agent",
				Protocol:             "OPENAI",
				Endpoint:             "https://api.example.com",
				SourceIP:             "", // 将在后面动态设置
				ServiceIP:            "1.2.3.4",
				Successful:           true,
				ProviderKey:          nil,
				ProviderModelKey:     "gpt-3.5-turbo",
				InputTokenCount:      100,
				OutputTokenCount:     50,
				TotalTime:            1500,
				DNSLookupTime:        50,
				TCPConnectTime:       100,
				TLSHandshakeTime:     200,
				PerOutputTokenTime:   14.285714285714286, // (1500-800)/(50-1) = 700/49
				FirstOutputTokenTime: 800,
				ErrorMessage:         "",
			},
		},
		{
			name:   "failed response with error",
			taskID: "task-456",
			metrics: &client.ResponseMetrics{
				PromptTokens:      0,
				CompletionTokens:  0,
				TotalTime:         time.Millisecond * 5000,
				DNSTime:           time.Millisecond * 100,
				ConnectTime:       time.Millisecond * 0,
				TLSHandshakeTime:  time.Millisecond * 0,
				TimeToFirstToken:  time.Millisecond * 0,
				TargetIP:          "",
				ErrorMessage:      "Connection timeout",
			},
			input: types.Input{
				Protocol: "anthropic",
				BaseUrl:  "https://api.example.com",
				Model:    "claude-3-sonnet",
			},
			expected: ReportUploadItem{
				TaskID:               "task-456",
				ModelKey:             nil,
				Reporter:             "test-agent",
				Protocol:             "ANTHROPIC",
				Endpoint:             "https://api.example.com",
				SourceIP:             "", // 将在后面动态设置
				ServiceIP:            "",
				Successful:           false,
				ProviderKey:          nil,
				ProviderModelKey:     "claude-3-sonnet",
				InputTokenCount:      0,
				OutputTokenCount:     0,
				TotalTime:            5000,
				DNSLookupTime:        100,
				TCPConnectTime:       0,
				TLSHandshakeTime:     0,
				PerOutputTokenTime:   0,
				FirstOutputTokenTime: 0,
				ErrorMessage:         "Connection timeout",
			},
		},
		{
			name:   "single completion token",
			taskID: "task-789",
			metrics: &client.ResponseMetrics{
				PromptTokens:      10,
				CompletionTokens:  1,
				TotalTime:         time.Millisecond * 1000,
				DNSTime:           time.Millisecond * 10,
				ConnectTime:       time.Millisecond * 20,
				TLSHandshakeTime:  time.Millisecond * 30,
				TimeToFirstToken:  time.Millisecond * 900,
				TargetIP:          "5.6.7.8",
				ErrorMessage:      "",
			},
			input: types.Input{
				Protocol: "local",
				BaseUrl:  "http://localhost:8080",
				Model:    "local-model",
			},
			expected: ReportUploadItem{
				TaskID:               "task-789",
				ModelKey:             nil,
				Reporter:             "test-agent",
				Protocol:             "LOCAL",
				Endpoint:             "http://localhost:8080",
				SourceIP:             "", // 将在后面动态设置
				ServiceIP:            "5.6.7.8",
				Successful:           true,
				ProviderKey:          nil,
				ProviderModelKey:     "local-model",
				InputTokenCount:      10,
				OutputTokenCount:     1,
				TotalTime:            1000,
				DNSLookupTime:        10,
				TCPConnectTime:       20,
				TLSHandshakeTime:     30,
				PerOutputTokenTime:   0, // 只有一个token，不计算每token时间
				FirstOutputTokenTime: 900,
				ErrorMessage:         "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uploader.convertResponseMetricsToUploadItem(tt.taskID, tt.metrics, tt.input)

			// 由于 SourceIP 是动态获取的，我们需要相应地设置期望值
			tt.expected.SourceIP = result.SourceIP

			// 比较各个字段
			if result.TaskID != tt.expected.TaskID {
				t.Errorf("TaskID: got %q, expected %q", result.TaskID, tt.expected.TaskID)
			}
			if (result.ModelKey == nil) != (tt.expected.ModelKey == nil) {
				t.Errorf("ModelKey nil mismatch: got %v, expected %v", result.ModelKey == nil, tt.expected.ModelKey == nil)
			} else if result.ModelKey != nil && tt.expected.ModelKey != nil && *result.ModelKey != *tt.expected.ModelKey {
				t.Errorf("ModelKey: got %q, expected %q", *result.ModelKey, *tt.expected.ModelKey)
			}
			if result.Reporter != tt.expected.Reporter {
				t.Errorf("Reporter: got %q, expected %q", result.Reporter, tt.expected.Reporter)
			}
			if result.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol: got %q, expected %q", result.Protocol, tt.expected.Protocol)
			}
			if result.Endpoint != tt.expected.Endpoint {
				t.Errorf("Endpoint: got %q, expected %q", result.Endpoint, tt.expected.Endpoint)
			}
			if result.SourceIP != tt.expected.SourceIP {
				t.Errorf("SourceIP: got %q, expected %q", result.SourceIP, tt.expected.SourceIP)
			}
			if result.ServiceIP != tt.expected.ServiceIP {
				t.Errorf("ServiceIP: got %q, expected %q", result.ServiceIP, tt.expected.ServiceIP)
			}
			if result.Successful != tt.expected.Successful {
				t.Errorf("Successful: got %v, expected %v", result.Successful, tt.expected.Successful)
			}
			if (result.ProviderKey == nil) != (tt.expected.ProviderKey == nil) {
				t.Errorf("ProviderKey nil mismatch: got %v, expected %v", result.ProviderKey == nil, tt.expected.ProviderKey == nil)
			} else if result.ProviderKey != nil && tt.expected.ProviderKey != nil && *result.ProviderKey != *tt.expected.ProviderKey {
				t.Errorf("ProviderKey: got %q, expected %q", *result.ProviderKey, *tt.expected.ProviderKey)
			}
			if result.ProviderModelKey != tt.expected.ProviderModelKey {
				t.Errorf("ProviderModelKey: got %q, expected %q", result.ProviderModelKey, tt.expected.ProviderModelKey)
			}
			if result.InputTokenCount != tt.expected.InputTokenCount {
				t.Errorf("InputTokenCount: got %d, expected %d", result.InputTokenCount, tt.expected.InputTokenCount)
			}
			if result.OutputTokenCount != tt.expected.OutputTokenCount {
				t.Errorf("OutputTokenCount: got %d, expected %d", result.OutputTokenCount, tt.expected.OutputTokenCount)
			}
			if result.TotalTime != tt.expected.TotalTime {
				t.Errorf("TotalTime: got %d, expected %d", result.TotalTime, tt.expected.TotalTime)
			}
			if result.DNSLookupTime != tt.expected.DNSLookupTime {
				t.Errorf("DNSLookupTime: got %d, expected %d", result.DNSLookupTime, tt.expected.DNSLookupTime)
			}
			if result.TCPConnectTime != tt.expected.TCPConnectTime {
				t.Errorf("TCPConnectTime: got %d, expected %d", result.TCPConnectTime, tt.expected.TCPConnectTime)
			}
			if result.TLSHandshakeTime != tt.expected.TLSHandshakeTime {
				t.Errorf("TLSHandshakeTime: got %d, expected %d", result.TLSHandshakeTime, tt.expected.TLSHandshakeTime)
			}
			if result.PerOutputTokenTime != tt.expected.PerOutputTokenTime {
				t.Errorf("PerOutputTokenTime: got %f, expected %f", result.PerOutputTokenTime, tt.expected.PerOutputTokenTime)
			}
			if result.FirstOutputTokenTime != tt.expected.FirstOutputTokenTime {
				t.Errorf("FirstOutputTokenTime: got %d, expected %d", result.FirstOutputTokenTime, tt.expected.FirstOutputTokenTime)
			}
			if result.ErrorMessage != tt.expected.ErrorMessage {
				t.Errorf("ErrorMessage: got %q, expected %q", result.ErrorMessage, tt.expected.ErrorMessage)
			}
		})
	}
}

func TestUploader_UploadReport(t *testing.T) {
	tests := []struct {
		name           string
		baseURL        string
		authToken      string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
	}{
		{
			name:      "invalid base URL - should return nil",
			baseURL:   "null",
			authToken: "test-token",
			expectedError: false,
		},
		{
			name:      "null auth token - should return nil", 
			baseURL:   "https://api.example.com",
			authToken: "null",
			expectedError: false,
		},
		{
			name:      "successful upload",
			baseURL:   "", // 将在测试中设置为test server URL
			authToken: "test-token",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// 验证请求方法
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				
				// 验证URL路径
				expectedPath := "/model/perf/report/upload"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				
				// 验证Content-Type
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}
				
				// 验证Authorization header
				expectedAuth := "Bearer test-token"
				if r.Header.Get("Authorization") != expectedAuth {
					t.Errorf("Expected Authorization %s, got %s", expectedAuth, r.Header.Get("Authorization"))
				}
				
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success": true}`))
			},
			expectedError: false,
		},
		{
			name:      "server error response",
			baseURL:   "", // 将在测试中设置为test server URL
			authToken: "test-token",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			expectedError: true,
		},
		{
			name:      "client error response",
			baseURL:   "", // 将在测试中设置为test server URL
			authToken: "test-token",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "bad request"}`))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 如果需要mock server，创建它
			var server *httptest.Server
			if tt.serverResponse != nil {
				server = httptest.NewServer(http.HandlerFunc(tt.serverResponse))
				defer server.Close()
				tt.baseURL = server.URL
			}

			uploader := &Uploader{
				baseURL:   tt.baseURL,
				authToken: tt.authToken,
				userAgent: "test-agent",
				client:    &http.Client{Timeout: time.Second * 3},
			}

			// 准备测试数据
			taskID := "test-task-123"
			metrics := &client.ResponseMetrics{
				PromptTokens:      100,
				CompletionTokens:  50,
				TotalTime:         time.Millisecond * 1500,
				DNSTime:           time.Millisecond * 50,
				ConnectTime:       time.Millisecond * 100,
				TLSHandshakeTime:  time.Millisecond * 200,
				TimeToFirstToken:  time.Millisecond * 800,
				TargetIP:          "1.2.3.4",
				ErrorMessage:      "",
			}
			input := types.Input{
				Protocol: "openai",
				BaseUrl:  "https://api.example.com",
				Model:    "gpt-3.5-turbo",
			}

			// 执行测试
			err := uploader.UploadReport(taskID, metrics, input)

			// 验证结果
			if tt.expectedError && err == nil {
				t.Error("Expected an error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestUploader_UploadReport_NetworkError(t *testing.T) {
	// 测试网络错误情况
	uploader := &Uploader{
		baseURL:   "http://nonexistent-server-12345.com",
		authToken: "test-token",
		userAgent: "test-agent",
		client:    &http.Client{Timeout: time.Millisecond * 100}, // 设置很短的超时
	}

	taskID := "test-task-123"
	metrics := &client.ResponseMetrics{
		PromptTokens:      100,
		CompletionTokens:  50,
		TotalTime:         time.Millisecond * 1500,
		DNSTime:           time.Millisecond * 50,
		ConnectTime:       time.Millisecond * 100,
		TLSHandshakeTime:  time.Millisecond * 200,
		TimeToFirstToken:  time.Millisecond * 800,
		TargetIP:          "1.2.3.4",
		ErrorMessage:      "",
	}
	input := types.Input{
		Protocol: "openai",
		BaseUrl:  "https://api.example.com",
		Model:    "gpt-3.5-turbo",
	}

	err := uploader.UploadReport(taskID, metrics, input)

	if err == nil {
		t.Error("Expected network error but got nil")
	}

	if err != nil && !contains(err.Error(), "context deadline exceeded") && !contains(err.Error(), "HTTP请求失败") && !contains(err.Error(), "no such host") {
		t.Errorf("Expected network or timeout error message, got: %v", err)
	}
}

// contains 检查字符串是否包含子字符串的辅助函数
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
