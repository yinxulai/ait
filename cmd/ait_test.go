package main

import (
	"flag"
	"os"
	"testing"
)

func TestDetectProviderFromEnv(t *testing.T) {
	// 保存原始环境变量
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	originalOpenAIURL := os.Getenv("OPENAI_BASE_URL")
	originalAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	originalAnthropicURL := os.Getenv("ANTHROPIC_BASE_URL")

	// 确保测试后恢复原始环境变量
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
		os.Setenv("OPENAI_BASE_URL", originalOpenAIURL)
		os.Setenv("ANTHROPIC_API_KEY", originalAnthropicKey)
		os.Setenv("ANTHROPIC_BASE_URL", originalAnthropicURL)
	}()

	tests := []struct {
		name                string
		openaiKey          string
		openaiURL          string
		anthropicKey       string
		anthropicURL       string
		expectedProvider   string
	}{
		{
			name:             "OpenAI API key set",
			openaiKey:        "test-openai-key",
			openaiURL:        "",
			anthropicKey:     "",
			anthropicURL:     "",
			expectedProvider: "openai",
		},
		{
			name:             "OpenAI base URL set",
			openaiKey:        "",
			openaiURL:        "https://api.openai.com",
			anthropicKey:     "",
			anthropicURL:     "",
			expectedProvider: "openai",
		},
		{
			name:             "Anthropic API key set",
			openaiKey:        "",
			openaiURL:        "",
			anthropicKey:     "test-anthropic-key",
			anthropicURL:     "",
			expectedProvider: "anthropic",
		},
		{
			name:             "Anthropic base URL set",
			openaiKey:        "",
			openaiURL:        "",
			anthropicKey:     "",
			anthropicURL:     "https://api.anthropic.com",
			expectedProvider: "anthropic",
		},
		{
			name:             "Both providers set - OpenAI takes priority",
			openaiKey:        "test-openai-key",
			openaiURL:        "",
			anthropicKey:     "test-anthropic-key",
			anthropicURL:     "",
			expectedProvider: "openai",
		},
		{
			name:             "No environment variables set - defaults to openai",
			openaiKey:        "",
			openaiURL:        "",
			anthropicKey:     "",
			anthropicURL:     "",
			expectedProvider: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有相关环境变量
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_BASE_URL")

			// 设置测试环境变量
			if tt.openaiKey != "" {
				os.Setenv("OPENAI_API_KEY", tt.openaiKey)
			}
			if tt.openaiURL != "" {
				os.Setenv("OPENAI_BASE_URL", tt.openaiURL)
			}
			if tt.anthropicKey != "" {
				os.Setenv("ANTHROPIC_API_KEY", tt.anthropicKey)
			}
			if tt.anthropicURL != "" {
				os.Setenv("ANTHROPIC_BASE_URL", tt.anthropicURL)
			}

			result := detectProviderFromEnv()
			if result != tt.expectedProvider {
				t.Errorf("detectProviderFromEnv() = %v, want %v", result, tt.expectedProvider)
			}
		})
	}
}

func TestLoadEnvForProvider(t *testing.T) {
	// 保存原始环境变量
	originalOpenAIKey := os.Getenv("OPENAI_API_KEY")
	originalOpenAIURL := os.Getenv("OPENAI_BASE_URL")
	originalAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	originalAnthropicURL := os.Getenv("ANTHROPIC_BASE_URL")

	// 确保测试后恢复原始环境变量
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalOpenAIKey)
		os.Setenv("OPENAI_BASE_URL", originalOpenAIURL)
		os.Setenv("ANTHROPIC_API_KEY", originalAnthropicKey)
		os.Setenv("ANTHROPIC_BASE_URL", originalAnthropicURL)
	}()

	tests := []struct {
		name         string
		provider     string
		envVars      map[string]string
		expectedURL  string
		expectedKey  string
	}{
		{
			name:     "OpenAI provider with environment variables",
			provider: "openai",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://api.openai.com",
				"OPENAI_API_KEY":  "test-openai-key",
			},
			expectedURL: "https://api.openai.com",
			expectedKey: "test-openai-key",
		},
		{
			name:     "Anthropic provider with environment variables",
			provider: "anthropic",
			envVars: map[string]string{
				"ANTHROPIC_BASE_URL": "https://api.anthropic.com",
				"ANTHROPIC_API_KEY":  "test-anthropic-key",
			},
			expectedURL: "https://api.anthropic.com",
			expectedKey: "test-anthropic-key",
		},
		{
			name:        "OpenAI provider without environment variables",
			provider:    "openai",
			envVars:     map[string]string{},
			expectedURL: "",
			expectedKey: "",
		},
		{
			name:        "Anthropic provider without environment variables",
			provider:    "anthropic",
			envVars:     map[string]string{},
			expectedURL: "",
			expectedKey: "",
		},
		{
			name:        "Unknown provider",
			provider:    "unknown",
			envVars:     map[string]string{},
			expectedURL: "",
			expectedKey: "",
		},
		{
			name:     "Only OpenAI URL set",
			provider: "openai",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://custom.openai.com",
			},
			expectedURL: "https://custom.openai.com",
			expectedKey: "",
		},
		{
			name:     "Only Anthropic key set",
			provider: "anthropic",
			envVars: map[string]string{
				"ANTHROPIC_API_KEY": "test-key-only",
			},
			expectedURL: "",
			expectedKey: "test-key-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有相关环境变量
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_BASE_URL")

			// 设置测试环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			baseUrl, apiKey := loadEnvForProvider(tt.provider)
			if baseUrl != tt.expectedURL {
				t.Errorf("loadEnvForProvider(%v) baseUrl = %v, want %v", tt.provider, baseUrl, tt.expectedURL)
			}
			if apiKey != tt.expectedKey {
				t.Errorf("loadEnvForProvider(%v) apiKey = %v, want %v", tt.provider, apiKey, tt.expectedKey)
			}
		})
	}
}

func TestFlagDefinitions(t *testing.T) {
	// 重置 flag 状态，避免冲突
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	// 模拟定义 flags（这部分通常在 main 中）
	baseUrl := flag.String("baseUrl", "", "服务地址")
	apikey := flag.String("apikey", "", "API 密钥")
	model := flag.String("model", "", "模型名称")
	provider := flag.String("provider", "", "协议类型: openai 或 anthropic")
	concurrency := flag.Int("concurrency", 3, "并发数")
	count := flag.Int("count", 10, "请求总数")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", true, "是否开启流模式")
	reportFlag := flag.Bool("report", false, "是否生成报告文件")

	// 测试默认值
	if *provider != "" {
		t.Errorf("Expected default provider '', got '%s'", *provider)
	}

	if *concurrency != 3 {
		t.Errorf("Expected default concurrency 3, got %d", *concurrency)
	}

	if *count != 10 {
		t.Errorf("Expected default count 10, got %d", *count)
	}

	if *stream != true {
		t.Errorf("Expected default stream true, got %t", *stream)
	}

	if *reportFlag != false {
		t.Errorf("Expected default report false, got %t", *reportFlag)
	}

	if *prompt != "你好，介绍一下你自己。" {
		t.Errorf("Expected default prompt '你好，介绍一下你自己。', got '%s'", *prompt)
	}

	// 测试 flag 是否正确定义
	if baseUrl == nil || apikey == nil || model == nil || prompt == nil {
		t.Error("Required flags should be defined")
	}
}
