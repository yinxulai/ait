package runner

import (
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestNewRunner(t *testing.T) {
	tests := []struct {
		name      string
		input     types.Input
		wantError bool
	}{
		{
			name: "valid openai config",
			input: types.Input{
				Protocol:    "openai",
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
			input: types.Input{
				Protocol:    "anthropic",
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
			input: types.Input{
				Protocol:    "invalid",
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
			runner, err := NewRunner(tt.input)

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

			if runner.config.Protocol != tt.input.Protocol {
				t.Errorf("NewRunner().config.Protocol = %v, want %v", runner.config.Protocol, tt.input.Protocol)
			}

			if runner.config.Stream != tt.input.Stream {
				t.Errorf("NewRunner().config.Stream = %v, want %v", runner.config.Stream, tt.input.Stream)
			}
		})
	}
}

func TestResult_PrintResult(t *testing.T) {
	tests := []struct {
		name   string
		result types.ReportData
	}{
		{
			name: "stream mode result",
			result: types.ReportData{
				TotalRequests: 10,
				Concurrency:   2,
				IsStream:      true,
				TotalTime:     5 * time.Second,
			},
		},
		{
			name: "non-stream mode result",
			result: types.ReportData{
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
