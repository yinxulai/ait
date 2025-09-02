package client

import (
	"testing"
)

func TestNewAnthropicClient(t *testing.T) {
	tests := []struct {
		name    string
		baseUrl string
		apiKey  string
		model   string
		want    *AnthropicClient
	}{
		{
			name:    "valid anthropic client",
			baseUrl: "https://api.anthropic.com",
			apiKey:  "test-key",
			model:   "claude-3-sonnet-20240229",
			want: &AnthropicClient{
				BaseUrl:  "https://api.anthropic.com",
				ApiKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				Provider: "anthropic",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAnthropicClient(tt.baseUrl, tt.apiKey, tt.model)
			
			if got.BaseUrl != tt.want.BaseUrl {
				t.Errorf("NewAnthropicClient().BaseUrl = %v, want %v", got.BaseUrl, tt.want.BaseUrl)
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
		})
	}
}

func TestAnthropicClient_GetProvider(t *testing.T) {
	client := NewAnthropicClient("https://api.anthropic.com", "test-key", "claude-3-sonnet-20240229")
	
	if got := client.GetProvider(); got != "anthropic" {
		t.Errorf("AnthropicClient.GetProvider() = %v, want %v", got, "anthropic")
	}
}

func TestAnthropicClient_GetModel(t *testing.T) {
	model := "claude-3-sonnet-20240229"
	client := NewAnthropicClient("https://api.anthropic.com", "test-key", model)
	
	if got := client.GetModel(); got != model {
		t.Errorf("AnthropicClient.GetModel() = %v, want %v", got, model)
	}
}
