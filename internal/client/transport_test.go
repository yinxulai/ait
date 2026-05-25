package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yinxulai/ait/internal/types"
)

func TestNewMeasuredTransport_ExplicitProxy(t *testing.T) {
	transport := newMeasuredTransport(types.Input{ProxyURL: "http://proxy.example:8080"})
	if transport.Proxy == nil {
		t.Fatal("Proxy should be configured when proxy_url is provided")
	}

	proxy, err := transport.Proxy(httptest.NewRequest(http.MethodGet, "https://api.example.com", nil))
	if err != nil {
		t.Fatalf("Proxy returned error: %v", err)
	}
	if proxy == nil || proxy.String() != "http://proxy.example:8080" {
		t.Fatalf("proxy = %v, want http://proxy.example:8080", proxy)
	}
}

func TestNewMeasuredTransport_InvalidProxy(t *testing.T) {
	transport := newMeasuredTransport(types.Input{ProxyURL: "://bad proxy"})
	if transport.Proxy == nil {
		t.Fatal("Proxy callback should be set for invalid proxy_url")
	}

	if _, err := transport.Proxy(httptest.NewRequest(http.MethodGet, "https://api.example.com", nil)); err == nil {
		t.Fatal("expected invalid proxy_url to return an error")
	}
}

func TestNewClients_UseConfiguredProxy(t *testing.T) {
	constructors := []struct {
		name      string
		transport func() *http.Transport
	}{
		{
			name: "openai",
			transport: func() *http.Transport {
				client := NewOpenAIClient(types.Input{
					Protocol: types.ProtocolOpenAICompletions,
					ProxyURL: "http://proxy.example:8080",
				})
				transport, _ := client.httpClient.Transport.(*http.Transport)
				return transport
			},
		},
		{
			name: "anthropic",
			transport: func() *http.Transport {
				client := NewAnthropicClient(types.Input{
					Protocol: types.ProtocolAnthropicMessages,
					ProxyURL: "http://proxy.example:8080",
				})
				transport, _ := client.httpClient.Transport.(*http.Transport)
				return transport
			},
		},
	}

	for _, tt := range constructors {
		t.Run(tt.name, func(t *testing.T) {
			transport := tt.transport()
			if transport == nil {
				t.Fatal("expected http.Transport")
			}
			proxy, err := transport.Proxy(httptest.NewRequest(http.MethodGet, "https://api.example.com", nil))
			if err != nil {
				t.Fatalf("Proxy returned error: %v", err)
			}
			if proxy == nil || proxy.String() != "http://proxy.example:8080" {
				t.Fatalf("proxy = %v, want http://proxy.example:8080", proxy)
			}
		})
	}
}
