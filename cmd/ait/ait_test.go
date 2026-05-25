package main

import (
	"os"
	"strings"
	"testing"
)

// ─── resolveConfig ────────────────────────────────────────────────────────────

// clearEnv 清除所有 provider 环境变量，返回 restore 函数。
func clearEnv(t *testing.T) func() {
	t.Helper()
	saved := map[string]string{}
	keys := []string{"OPENAI_API_KEY", "OPENAI_BASE_URL", "ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL"}
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	return func() {
		for k, v := range saved {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}
}

func TestResolveConfig_ProtocolInference(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantProt string
	}{
		{
			name:     "OpenAI API key → openai",
			envVars:  map[string]string{"OPENAI_API_KEY": "sk-test"},
			wantProt: "openai",
		},
		{
			name:     "OpenAI base URL → openai",
			envVars:  map[string]string{"OPENAI_BASE_URL": "https://api.openai.com"},
			wantProt: "openai",
		},
		{
			name:     "Anthropic key → anthropic",
			envVars:  map[string]string{"ANTHROPIC_API_KEY": "sk-ant"},
			wantProt: "anthropic",
		},
		{
			name:     "Anthropic URL → anthropic",
			envVars:  map[string]string{"ANTHROPIC_BASE_URL": "https://api.anthropic.com"},
			wantProt: "anthropic",
		},
		{
			name:     "Both set → openai wins",
			envVars:  map[string]string{"OPENAI_API_KEY": "sk-test", "ANTHROPIC_API_KEY": "sk-ant"},
			wantProt: "openai",
		},
		{
			name:     "No env vars → default openai",
			envVars:  map[string]string{},
			wantProt: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := clearEnv(t)
			defer restore()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			prot, _, _ := resolveConfig("", "", "")
			if prot != tt.wantProt {
				t.Errorf("protocol = %q, want %q", prot, tt.wantProt)
			}
		})
	}
}

func TestResolveConfig_KeyAndURL(t *testing.T) {
	tests := []struct {
		name      string
		protocol  string
		envVars   map[string]string
		wantURL   string
		wantKey   string
	}{
		{
			name:     "openai env vars resolved",
			protocol: "openai",
			envVars:  map[string]string{"OPENAI_BASE_URL": "https://api.openai.com", "OPENAI_API_KEY": "sk-openai"},
			wantURL:  "https://api.openai.com",
			wantKey:  "sk-openai",
		},
		{
			name:     "anthropic env vars resolved",
			protocol: "anthropic",
			envVars:  map[string]string{"ANTHROPIC_BASE_URL": "https://api.anthropic.com", "ANTHROPIC_API_KEY": "sk-ant"},
			wantURL:  "https://api.anthropic.com",
			wantKey:  "sk-ant",
		},
		{
			name:     "explicit args override env",
			protocol: "openai",
			envVars:  map[string]string{"OPENAI_BASE_URL": "https://env.url", "OPENAI_API_KEY": "env-key"},
			wantURL:  "https://explicit.url",
			wantKey:  "explicit-key",
		},
		{
			name:     "unknown protocol - no env",
			protocol: "other",
			envVars:  map[string]string{},
			wantURL:  "",
			wantKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := clearEnv(t)
			defer restore()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			var prot, url, key string
			if tt.name == "explicit args override env" {
				prot, url, key = resolveConfig(tt.protocol, "https://explicit.url", "explicit-key")
			} else {
				prot, url, key = resolveConfig(tt.protocol, "", "")
			}
			_ = prot
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

// ─── ParseModels (inline logic, no helper needed) ─────────────────────────────

func TestParseModels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single model", "gpt-3.5-turbo", []string{"gpt-3.5-turbo"}},
		{"multiple models", "gpt-3.5-turbo,gpt-4,claude-3", []string{"gpt-3.5-turbo", "gpt-4", "claude-3"}},
		{"models with spaces", "gpt-3.5-turbo, gpt-4 , claude-3", []string{"gpt-3.5-turbo", "gpt-4", "claude-3"}},
		{"single model with spaces", " gpt-3.5-turbo ", []string{"gpt-3.5-turbo"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(tt.input, ",")
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			if len(parts) != len(tt.expected) {
				t.Errorf("got %d models, want %d", len(parts), len(tt.expected))
				return
			}
			for i, want := range tt.expected {
				if parts[i] != want {
					t.Errorf("[%d] got %q, want %q", i, parts[i], want)
				}
			}
		})
	}
}
