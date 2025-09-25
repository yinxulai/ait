package client

import (
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    types.Input
		wantError bool
	}{
		{
			name: "valid openai client",
			config: types.Input{
				Protocol: "openai",
				BaseUrl:  "https://api.openai.com",
				ApiKey:   "test-key",
				Model:    "gpt-3.5-turbo",
				Timeout:  30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "valid anthropic client",
			config: types.Input{
				Protocol: "anthropic",
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				Timeout:  30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "invalid provider",
			config: types.Input{
				Protocol: "invalid",
				BaseUrl:  "https://api.test.com",
				ApiKey:   "test-key",
				Model:    "test-model",
				Timeout:  30 * time.Second,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config, nil)

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

			if client.GetProtocol() != tt.config.Protocol {
				t.Errorf("NewClient().GetProtocol() = %v, want %v", client.GetProtocol(), tt.config.Protocol)
			}

			if client.GetModel() != tt.config.Model {
				t.Errorf("NewClient().GetModel() = %v, want %v", client.GetModel(), tt.config.Model)
			}
		})
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	tests := []struct {
		name      string
		config    types.Input
		wantError bool
	}{
		{
			name: "valid openai client with timeout",
			config: types.Input{
				Protocol: "openai",
				BaseUrl:  "https://api.openai.com",
				ApiKey:   "test-key",
				Model:    "gpt-3.5-turbo",
				Timeout:  10 * time.Second,
			},
			wantError: false,
		},
		{
			name: "valid anthropic client with timeout",
			config: types.Input{
				Protocol: "anthropic",
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-sonnet",
				Timeout:  30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "invalid provider with timeout",
			config: types.Input{
				Protocol: "invalid",
				BaseUrl:  "https://api.test.com",
				ApiKey:   "test-key",
				Model:    "test-model",
				Timeout:  5 * time.Second,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config, nil)

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

			if client.GetProtocol() != tt.config.Protocol {
				t.Errorf("NewClient().GetProtocol() = %v, want %v", client.GetProtocol(), tt.config.Protocol)
			}

			if client.GetModel() != tt.config.Model {
				t.Errorf("NewClient().GetModel() = %v, want %v", client.GetModel(), tt.config.Model)
			}
		})
	}
}
