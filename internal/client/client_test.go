package client

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		baseUrl   string
		apiKey    string
		model     string
		wantError bool
	}{
		{
			name:      "valid openai client",
			provider:  "openai",
			baseUrl:   "https://api.openai.com",
			apiKey:    "test-key",
			model:     "gpt-3.5-turbo",
			wantError: false,
		},
		{
			name:      "valid anthropic client",
			provider:  "anthropic",
			baseUrl:   "https://api.anthropic.com",
			apiKey:    "test-key",
			model:     "claude-3-sonnet-20240229",
			wantError: false,
		},
		{
			name:      "invalid provider",
			provider:  "invalid",
			baseUrl:   "https://api.test.com",
			apiKey:    "test-key",
			model:     "test-model",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.provider, tt.baseUrl, tt.apiKey, tt.model)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewClient() error = nil, wantError %v", tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClient() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if client == nil {
				t.Error("NewClient() returned nil client")
				return
			}

			if client.GetProtocol() != tt.provider {
				t.Errorf("NewClient().GetProtocol() = %v, want %v", client.GetProtocol(), tt.provider)
			}

			if client.GetModel() != tt.model {
				t.Errorf("NewClient().GetModel() = %v, want %v", client.GetModel(), tt.model)
			}
		})
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		baseUrl   string
		apiKey    string
		model     string
		timeout   time.Duration
		wantError bool
	}{
		{
			name:      "valid openai client with timeout",
			provider:  "openai",
			baseUrl:   "https://api.openai.com",
			apiKey:    "test-key",
			model:     "gpt-3.5-turbo",
			timeout:   10 * time.Second,
			wantError: false,
		},
		{
			name:      "valid anthropic client with timeout",
			provider:  "anthropic",
			baseUrl:   "https://api.anthropic.com",
			apiKey:    "test-key",
			model:     "claude-3-sonnet",
			timeout:   30 * time.Second,
			wantError: false,
		},
		{
			name:      "invalid provider with timeout",
			provider:  "invalid",
			baseUrl:   "https://api.test.com",
			apiKey:    "test-key",
			model:     "test-model",
			timeout:   5 * time.Second,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientWithTimeout(tt.provider, tt.baseUrl, tt.apiKey, tt.model, tt.timeout)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewClientWithTimeout() error = nil, wantError %v", tt.wantError)
				}
				return
			}

			if err != nil {
				t.Errorf("NewClientWithTimeout() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if client == nil {
				t.Error("NewClientWithTimeout() returned nil client")
				return
			}

			if client.GetProtocol() != tt.provider {
				t.Errorf("NewClientWithTimeout().GetProtocol() = %v, want %v", client.GetProtocol(), tt.provider)
			}

			if client.GetModel() != tt.model {
				t.Errorf("NewClientWithTimeout().GetModel() = %v, want %v", client.GetModel(), tt.model)
			}
		})
	}
}
