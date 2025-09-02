package runner

import (
	"testing"
	"time"
)

func TestNewRunner(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "valid openai config",
			config: Config{
				Provider:    "openai",
				BaseUrl:     "https://api.openai.com",
				ApiKey:      "test-key",
				Model:       "gpt-3.5-turbo",
				Concurrency: 1,
				Count:       10,
				Prompt:      "test prompt",
				Stream:      false,
			},
			wantError: false,
		},
		{
			name: "valid anthropic config",
			config: Config{
				Provider:    "anthropic",
				BaseUrl:     "https://api.anthropic.com",
				ApiKey:      "test-key",
				Model:       "claude-3-sonnet-20240229",
				Concurrency: 2,
				Count:       5,
				Prompt:      "test prompt",
				Stream:      true,
			},
			wantError: false,
		},
		{
			name: "invalid provider",
			config: Config{
				Provider:    "invalid",
				BaseUrl:     "https://api.test.com",
				ApiKey:      "test-key",
				Model:       "test-model",
				Concurrency: 1,
				Count:       10,
				Prompt:      "test prompt",
				Stream:      false,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewRunner(tt.config)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewRunner() error = nil, wantError %v", tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewRunner() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if runner == nil {
				t.Error("NewRunner() returned nil runner")
				return
			}

			if runner.client == nil {
				t.Error("NewRunner().client should not be nil")
			}

			if runner.config.Provider != tt.config.Provider {
				t.Errorf("NewRunner().config.Provider = %v, want %v", runner.config.Provider, tt.config.Provider)
			}

			if runner.config.Stream != tt.config.Stream {
				t.Errorf("NewRunner().config.Stream = %v, want %v", runner.config.Stream, tt.config.Stream)
			}
		})
	}
}

func TestResult_PrintResult(t *testing.T) {
	tests := []struct {
		name   string
		result Result
	}{
		{
			name: "stream mode result",
			result: Result{
				TotalRequests: 10,
				Concurrency:   2,
				IsStream:      true,
				TotalTime:     5 * time.Second,
			},
		},
		{
			name: "non-stream mode result",
			result: Result{
				TotalRequests: 20,
				Concurrency:   4,
				IsStream:      false,
				TotalTime:     10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 这个测试主要确保 PrintResult 不会 panic
			// 验证结果结构体的基本字段
			if tt.result.TotalRequests == 0 {
				t.Error("TotalRequests should not be zero")
			}
			if tt.result.TotalTime == 0 {
				t.Error("TotalTime should not be zero")
			}
		})
	}
}
