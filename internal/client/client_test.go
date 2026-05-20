package client

import (
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name             string
		config           types.Input
		wantError        bool
		expectedProtocol string
		expectedEndpoint string
	}{
		{
			name: "valid openai completions client",
			config: types.Input{
				Protocol:    types.ProtocolOpenAICompletions,
				EndpointURL: "https://api.openai.com/v1/chat/completions",
				ApiKey:      "test-key",
				Model:       "gpt-4.1-mini",
				Timeout:     30 * time.Second,
			},
			wantError:        false,
			expectedProtocol: types.ProtocolOpenAICompletions,
			expectedEndpoint: "https://api.openai.com/v1/chat/completions",
		},
		{
			name: "valid openai responses client",
			config: types.Input{
				Protocol:    types.ProtocolOpenAIResponses,
				EndpointURL: "https://api.openai.com/v1/responses",
				ApiKey:      "test-key",
				Model:       "gpt-4.1-mini",
				Timeout:     30 * time.Second,
			},
			wantError:        false,
			expectedProtocol: types.ProtocolOpenAIResponses,
			expectedEndpoint: "https://api.openai.com/v1/responses",
		},
		{
			name: "valid anthropic messages client",
			config: types.Input{
				Protocol:    types.ProtocolAnthropicMessages,
				EndpointURL: "https://api.anthropic.com/v1/messages",
				ApiKey:      "test-key",
				Model:       "claude-3-7-sonnet-latest",
				Timeout:     30 * time.Second,
			},
			wantError:        false,
			expectedProtocol: types.ProtocolAnthropicMessages,
			expectedEndpoint: "https://api.anthropic.com/v1/messages",
		},
		{
			name: "legacy provider maps to explicit protocol and endpoint",
			config: types.Input{
				Protocol: "openai",
				BaseUrl:  "https://api.openai.com",
				ApiKey:   "test-key",
				Model:    "gpt-4.1-mini",
				Timeout:  30 * time.Second,
			},
			wantError:        false,
			expectedProtocol: types.ProtocolOpenAICompletions,
			expectedEndpoint: "https://api.openai.com/v1/chat/completions",
		},
		{
			name: "invalid provider",
			config: types.Input{
				Protocol:    "invalid",
				EndpointURL: "https://api.test.com/v1/anything",
				ApiKey:      "test-key",
				Model:       "test-model",
				Timeout:     30 * time.Second,
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

			if client.GetProtocol() != tt.expectedProtocol {
				t.Errorf("NewClient().GetProtocol() = %v, want %v", client.GetProtocol(), tt.expectedProtocol)
			}

			if client.GetModel() != tt.config.Model {
				t.Errorf("NewClient().GetModel() = %v, want %v", client.GetModel(), tt.config.Model)
			}

			switch typed := client.(type) {
			case *OpenAIClient:
				if typed.endpointURL != tt.expectedEndpoint {
					t.Errorf("NewClient() endpointURL = %v, want %v", typed.endpointURL, tt.expectedEndpoint)
				}
			case *AnthropicClient:
				if typed.EndpointURL != tt.expectedEndpoint {
					t.Errorf("NewClient() endpointURL = %v, want %v", typed.EndpointURL, tt.expectedEndpoint)
				}
			}
		})
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	tests := []struct {
		name             string
		config           types.Input
		wantError        bool
		expectedProtocol string
	}{
		{
			name: "valid openai completions client with timeout",
			config: types.Input{
				Protocol:    types.ProtocolOpenAICompletions,
				EndpointURL: "https://api.openai.com/v1/chat/completions",
				ApiKey:      "test-key",
				Model:       "gpt-4.1-mini",
				Timeout:     10 * time.Second,
			},
			wantError:        false,
			expectedProtocol: types.ProtocolOpenAICompletions,
		},
		{
			name: "valid anthropic client with timeout",
			config: types.Input{
				Protocol:    types.ProtocolAnthropicMessages,
				EndpointURL: "https://api.anthropic.com/v1/messages",
				ApiKey:      "test-key",
				Model:       "claude-3-sonnet",
				Timeout:     30 * time.Second,
			},
			wantError:        false,
			expectedProtocol: types.ProtocolAnthropicMessages,
		},
		{
			name: "invalid provider with timeout",
			config: types.Input{
				Protocol:    "invalid",
				EndpointURL: "https://api.test.com/v1/anything",
				ApiKey:      "test-key",
				Model:       "test-model",
				Timeout:     5 * time.Second,
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

			if client.GetProtocol() != tt.expectedProtocol {
				t.Errorf("NewClient().GetProtocol() = %v, want %v", client.GetProtocol(), tt.expectedProtocol)
			}

			if client.GetModel() != tt.config.Model {
				t.Errorf("NewClient().GetModel() = %v, want %v", client.GetModel(), tt.config.Model)
			}
		})
	}
}
