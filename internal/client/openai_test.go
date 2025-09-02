package client

import (
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

func TestOpenAIClient_GetProvider(t *testing.T) {
	client := NewOpenAIClient("https://api.openai.com", "test-key", "gpt-3.5-turbo")
	
	if got := client.GetProvider(); got != "openai" {
		t.Errorf("OpenAIClient.GetProvider() = %v, want %v", got, "openai")
	}
}

func TestOpenAIClient_GetModel(t *testing.T) {
	model := "gpt-4"
	client := NewOpenAIClient("https://api.openai.com", "test-key", model)
	
	if got := client.GetModel(); got != model {
		t.Errorf("OpenAIClient.GetModel() = %v, want %v", got, model)
	}
}
