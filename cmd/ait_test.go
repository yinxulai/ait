package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/display"
	"github.com/yinxulai/ait/internal/types"
)

// MockRunner 模拟 runner 以便测试 main 函数逻辑
type MockRunner struct {
	config types.Input
	result *types.ReportData
	err    error
}

func NewMockRunner(config types.Input) (*MockRunner, error) {
	if config.Model == "invalid-model" {
		return nil, fmt.Errorf("invalid model: %s", config.Model)
	}
	
	return &MockRunner{
		config: config,
		result: &types.ReportData{
			TotalRequests: config.Count,
			Concurrency:   config.Concurrency,
			IsStream:      config.Stream,
			TotalTime:     1500 * time.Millisecond,
			Metadata: struct {
				Timestamp string `json:"timestamp"`
				Protocol  string `json:"protocol"`
				Model     string `json:"model"`
				BaseUrl   string `json:"base_url"`
			}{
				Timestamp: time.Now().Format(time.RFC3339),
				Protocol:  config.Protocol,
				Model:     config.Model,
				BaseUrl:   config.BaseUrl,
			},
			TimeMetrics: struct {
				AvgTotalTime time.Duration `json:"avg_total_time"`
				MinTotalTime time.Duration `json:"min_total_time"`
				MaxTotalTime time.Duration `json:"max_total_time"`
			}{
				AvgTotalTime: 150 * time.Millisecond,
				MinTotalTime: 100 * time.Millisecond,
				MaxTotalTime: 200 * time.Millisecond,
			},
			ContentMetrics: struct {
				AvgTTFT       time.Duration `json:"avg_ttft"`
				MinTTFT       time.Duration `json:"min_ttft"`
				MaxTTFT       time.Duration `json:"max_ttft"`
				AvgTPOT       time.Duration `json:"avg_tpot"`
				MinTPOT       time.Duration `json:"min_tpot"`
				MaxTPOT       time.Duration `json:"max_tpot"`
				AvgTokenCount int           `json:"avg_token_count"`
				MinTokenCount int           `json:"min_token_count"`
				MaxTokenCount int           `json:"max_token_count"`
				AvgTPS        float64       `json:"avg_tps"`
				MinTPS        float64       `json:"min_tps"`
				MaxTPS        float64       `json:"max_tps"`
			}{
				AvgTTFT:       50 * time.Millisecond,
				MinTTFT:       30 * time.Millisecond,
				MaxTTFT:       70 * time.Millisecond,
				AvgTPOT:       25 * time.Millisecond,
				MinTPOT:       20 * time.Millisecond,
				MaxTPOT:       30 * time.Millisecond,
				AvgTokenCount: 100,
				MinTokenCount: 80,
				MaxTokenCount: 120,
				AvgTPS:        200.0,
				MinTPS:        150.0,
				MaxTPS:        250.0,
			},
			ReliabilityMetrics: struct {
				ErrorRate   float64 `json:"error_rate"`
				SuccessRate float64 `json:"success_rate"`
			}{
				ErrorRate:   0.0,
				SuccessRate: 100.0,
			},
		},
	}, nil
}

func (m *MockRunner) RunWithProgress(callback func(types.StatsData)) (*types.ReportData, error) {
	if m.err != nil {
		return nil, m.err
	}

	// 模拟进度回调
	for i := 0; i <= m.config.Count; i++ {
		callback(types.StatsData{
			CompletedCount: i,
			FailedCount:    0,
			ErrorMessages:  []string{},
		})
	}

	return m.result, nil
}

func (m *MockRunner) Run() (*types.ReportData, error) {
	return m.result, m.err
}

// MockDisplay 模拟 display 组件以便测试
type MockDisplay struct{}

func (md *MockDisplay) Init(total int) error {
	return nil
}

func (md *MockDisplay) Update(current int) error {
	return nil
}

func (md *MockDisplay) Finish() error {
	return nil
}

func (md *MockDisplay) ShowResults(data interface{}) error {
	return nil
}

func TestDetectProtocolFromEnv(t *testing.T) {
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
		expectedProtocol   string
	}{
		{
			name:             "OpenAI API key set",
			openaiKey:        "test-openai-key",
			openaiURL:        "",
			anthropicKey:     "",
			anthropicURL:     "",
			expectedProtocol: "openai",
		},
		{
			name:             "OpenAI base URL set",
			openaiKey:        "",
			openaiURL:        "https://api.openai.com",
			anthropicKey:     "",
			anthropicURL:     "",
			expectedProtocol: "openai",
		},
		{
			name:             "Anthropic API key set",
			openaiKey:        "",
			openaiURL:        "",
			anthropicKey:     "test-anthropic-key",
			anthropicURL:     "",
			expectedProtocol: "anthropic",
		},
		{
			name:             "Anthropic base URL set",
			openaiKey:        "",
			openaiURL:        "",
			anthropicKey:     "",
			anthropicURL:     "https://api.anthropic.com",
			expectedProtocol: "anthropic",
		},
		{
			name:             "Both providers set - OpenAI takes priority",
			openaiKey:        "test-openai-key",
			openaiURL:        "",
			anthropicKey:     "test-anthropic-key",
			anthropicURL:     "",
			expectedProtocol: "openai",
		},
		{
			name:             "No environment variables set - defaults to openai",
			openaiKey:        "",
			openaiURL:        "",
			anthropicKey:     "",
			anthropicURL:     "",
			expectedProtocol: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理所有相关环境变量
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

			got := detectProviderFromEnv()
			if got != tt.expectedProtocol {
				t.Errorf("detectProviderFromEnv() = %v, want %v", got, tt.expectedProtocol)
			}
		})
	}
}

func TestLoadEnvForProtocol(t *testing.T) {
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
		protocol     string
		envVars      map[string]string
		expectedURL  string
		expectedKey  string
	}{
		{
			name:     "OpenAI protocol with environment variables",
			protocol: "openai",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://api.openai.com",
				"OPENAI_API_KEY":  "test-openai-key",
			},
			expectedURL: "https://api.openai.com",
			expectedKey: "test-openai-key",
		},
		{
			name:     "Anthropic protocol with environment variables",
			protocol: "anthropic",
			envVars: map[string]string{
				"ANTHROPIC_BASE_URL": "https://api.anthropic.com",
				"ANTHROPIC_API_KEY":  "test-anthropic-key",
			},
			expectedURL: "https://api.anthropic.com",
			expectedKey: "test-anthropic-key",
		},
		{
			name:        "OpenAI protocol without environment variables",
			protocol:    "openai",
			envVars:     map[string]string{},
			expectedURL: "",
			expectedKey: "",
		},
		{
			name:        "Anthropic protocol without environment variables",
			protocol:    "anthropic",
			envVars:     map[string]string{},
			expectedURL: "",
			expectedKey: "",
		},
		{
			name:        "Unknown protocol",
			protocol:    "unknown",
			envVars:     map[string]string{},
			expectedURL: "",
			expectedKey: "",
		},
		{
			name:     "Only OpenAI URL set",
			protocol: "openai",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://custom.openai.com",
			},
			expectedURL: "https://custom.openai.com",
			expectedKey: "",
		},
		{
			name:     "Only Anthropic key set",
			protocol: "anthropic",
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

			baseUrl, apiKey := loadEnvForProvider(tt.protocol)
			if baseUrl != tt.expectedURL {
				t.Errorf("loadEnvForProtocol(%v) baseUrl = %v, want %v", tt.protocol, baseUrl, tt.expectedURL)
			}
			if apiKey != tt.expectedKey {
				t.Errorf("loadEnvForProtocol(%v) apiKey = %v, want %v", tt.protocol, apiKey, tt.expectedKey)
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
	provider := flag.String("protocol", "", "协议类型: openai 或 anthropic")
	concurrency := flag.Int("concurrency", 3, "并发数")
	count := flag.Int("count", 10, "请求总数")
	prompt := flag.String("prompt", "你好，介绍一下你自己。", "测试用 prompt")
	stream := flag.Bool("stream", true, "是否开启流模式")
	reportFlag := flag.Bool("report", false, "是否生成报告文件")

	// 测试默认值
	if *provider != "" {
		t.Errorf("Expected default protocol '', got '%s'", *provider)
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

func TestParseModels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single model",
			input:    "gpt-3.5-turbo",
			expected: []string{"gpt-3.5-turbo"},
		},
		{
			name:     "Multiple models",
			input:    "gpt-3.5-turbo,gpt-4,claude-3",
			expected: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
		},
		{
			name:     "Models with spaces",
			input:    "gpt-3.5-turbo, gpt-4 , claude-3",
			expected: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
		},
		{
			name:     "Empty model in list",
			input:    "gpt-3.5-turbo,,gpt-4",
			expected: []string{"gpt-3.5-turbo", "", "gpt-4"},
		},
		{
			name:     "Single model with spaces",
			input:    " gpt-3.5-turbo ",
			expected: []string{"gpt-3.5-turbo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 main 函数中的模型解析逻辑
			modelList := strings.Split(tt.input, ",")
			for i, m := range modelList {
				modelList[i] = strings.TrimSpace(m)
			}

			if len(modelList) != len(tt.expected) {
				t.Errorf("Expected %d models, got %d", len(tt.expected), len(modelList))
				return
			}

			for i, expected := range tt.expected {
				if modelList[i] != expected {
					t.Errorf("Model[%d]: expected '%s', got '%s'", i, expected, modelList[i])
				}
			}
		})
	}
}

func TestInputConfig(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		baseUrl  string
		apiKey   string
		model    string
		config   types.Input
	}{
		{
			name:     "OpenAI configuration",
			protocol: "openai",
			baseUrl:  "https://api.openai.com",
			apiKey:   "test-openai-key",
			model:    "gpt-3.5-turbo",
			config: types.Input{
				Protocol:    "openai",
				BaseUrl:     "https://api.openai.com",
				ApiKey:      "test-openai-key",
				Model:       "gpt-3.5-turbo",
				Concurrency: 3,
				Count:       10,
				Prompt:      "你好，介绍一下你自己。",
				Stream:      true,
				Report:      false,
				Timeout:     30 * time.Second,
			},
		},
		{
			name:     "Anthropic configuration",
			protocol: "anthropic",
			baseUrl:  "https://api.anthropic.com",
			apiKey:   "test-anthropic-key",
			model:    "claude-3",
			config: types.Input{
				Protocol:    "anthropic",
				BaseUrl:     "https://api.anthropic.com",
				ApiKey:      "test-anthropic-key",
				Model:       "claude-3",
				Concurrency: 5,
				Count:       20,
				Prompt:      "Test prompt",
				Stream:      false,
				Report:      true,
				Timeout:     60 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟创建 Input 配置的过程
			config := types.Input{
				Protocol:    tt.protocol,
				BaseUrl:     tt.baseUrl,
				ApiKey:      tt.apiKey,
				Model:       tt.model,
				Concurrency: tt.config.Concurrency,
				Count:       tt.config.Count,
				Prompt:      tt.config.Prompt,
				Stream:      tt.config.Stream,
				Report:      tt.config.Report,
				Timeout:     tt.config.Timeout,
			}

			// 验证配置字段
			if config.Protocol != tt.config.Protocol {
				t.Errorf("Protocol: expected %s, got %s", tt.config.Protocol, config.Protocol)
			}
			if config.BaseUrl != tt.config.BaseUrl {
				t.Errorf("BaseUrl: expected %s, got %s", tt.config.BaseUrl, config.BaseUrl)
			}
			if config.ApiKey != tt.config.ApiKey {
				t.Errorf("ApiKey: expected %s, got %s", tt.config.ApiKey, config.ApiKey)
			}
			if config.Model != tt.config.Model {
				t.Errorf("Model: expected %s, got %s", tt.config.Model, config.Model)
			}
			if config.Concurrency != tt.config.Concurrency {
				t.Errorf("Concurrency: expected %d, got %d", tt.config.Concurrency, config.Concurrency)
			}
			if config.Count != tt.config.Count {
				t.Errorf("Count: expected %d, got %d", tt.config.Count, config.Count)
			}
			if config.Stream != tt.config.Stream {
				t.Errorf("Stream: expected %t, got %t", tt.config.Stream, config.Stream)
			}
			if config.Report != tt.config.Report {
				t.Errorf("Report: expected %t, got %t", tt.config.Report, config.Report)
			}
			if config.Timeout != tt.config.Timeout {
				t.Errorf("Timeout: expected %v, got %v", tt.config.Timeout, config.Timeout)
			}
		})
	}
}

func TestParameterValidation(t *testing.T) {
	tests := []struct {
		name        string
		models      string
		baseUrl     string
		apiKey      string
		shouldError bool
		errorDesc   string
	}{
		{
			name:        "Valid parameters",
			models:      "gpt-3.5-turbo",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			shouldError: false,
		},
		{
			name:        "Empty models",
			models:      "",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			shouldError: true,
			errorDesc:   "models parameter is required",
		},
		{
			name:        "Empty baseUrl",
			models:      "gpt-3.5-turbo",
			baseUrl:     "",
			apiKey:      "test-key",
			shouldError: true,
			errorDesc:   "baseUrl is required",
		},
		{
			name:        "Empty apiKey",
			models:      "gpt-3.5-turbo",
			baseUrl:     "https://api.openai.com",
			apiKey:      "",
			shouldError: true,
			errorDesc:   "apiKey is required",
		},
		{
			name:        "Both baseUrl and apiKey empty",
			models:      "gpt-3.5-turbo",
			baseUrl:     "",
			apiKey:      "",
			shouldError: true,
			errorDesc:   "both baseUrl and apiKey are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟主函数中的参数验证逻辑
			hasError := false
			
			// 检查 models 参数
			if tt.models == "" {
				hasError = true
			}
			
			// 检查 baseUrl 和 apiKey 参数
			if tt.baseUrl == "" || tt.apiKey == "" {
				hasError = true
			}

			if hasError != tt.shouldError {
				t.Errorf("Expected error: %t, got: %t. Description: %s", tt.shouldError, hasError, tt.errorDesc)
			}
		})
	}
}

func TestEnvironmentVariableOverride(t *testing.T) {
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
		name        string
		protocol    string
		cmdBaseUrl  string
		cmdApiKey   string
		envVars     map[string]string
		expectedUrl string
		expectedKey string
	}{
		{
			name:       "Command line takes priority over env vars",
			protocol:   "openai",
			cmdBaseUrl: "https://cmd.api.com",
			cmdApiKey:  "cmd-key",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://env.api.com",
				"OPENAI_API_KEY":  "env-key",
			},
			expectedUrl: "https://cmd.api.com",
			expectedKey: "cmd-key",
		},
		{
			name:       "Env vars used when command line empty",
			protocol:   "openai",
			cmdBaseUrl: "",
			cmdApiKey:  "",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://env.api.com",
				"OPENAI_API_KEY":  "env-key",
			},
			expectedUrl: "https://env.api.com",
			expectedKey: "env-key",
		},
		{
			name:       "Mixed: command line URL, env key",
			protocol:   "anthropic",
			cmdBaseUrl: "https://cmd.anthropic.com",
			cmdApiKey:  "",
			envVars: map[string]string{
				"ANTHROPIC_BASE_URL": "https://env.anthropic.com",
				"ANTHROPIC_API_KEY":  "env-anthropic-key",
			},
			expectedUrl: "https://cmd.anthropic.com",
			expectedKey: "env-anthropic-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有环境变量
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_BASE_URL")

			// 设置测试环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// 模拟 main 函数中的逻辑
			finalProtocol := tt.protocol
			finalBaseUrl := tt.cmdBaseUrl
			finalApiKey := tt.cmdApiKey

			// 根据 protocol 加载对应的环境变量
			if finalBaseUrl == "" || finalApiKey == "" {
				envBaseUrl, envApiKey := loadEnvForProvider(finalProtocol)
				if finalBaseUrl == "" {
					finalBaseUrl = envBaseUrl
				}
				if finalApiKey == "" {
					finalApiKey = envApiKey
				}
			}

			if finalBaseUrl != tt.expectedUrl {
				t.Errorf("Expected baseUrl %s, got %s", tt.expectedUrl, finalBaseUrl)
			}
			if finalApiKey != tt.expectedKey {
				t.Errorf("Expected apiKey %s, got %s", tt.expectedKey, finalApiKey)
			}
		})
	}
}

func TestModelListProcessing(t *testing.T) {
	tests := []struct {
		name           string
		modelsParam    string
		expectedCount  int
		expectedModels []string
		shouldError    bool
	}{
		{
			name:           "Valid single model",
			modelsParam:    "gpt-3.5-turbo",
			expectedCount:  1,
			expectedModels: []string{"gpt-3.5-turbo"},
			shouldError:    false,
		},
		{
			name:           "Valid multiple models",
			modelsParam:    "gpt-3.5-turbo,gpt-4,claude-3",
			expectedCount:  3,
			expectedModels: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
			shouldError:    false,
		},
		{
			name:           "Models with various spacing",
			modelsParam:    " gpt-3.5-turbo , gpt-4,  claude-3 ",
			expectedCount:  3,
			expectedModels: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
			shouldError:    false,
		},
		{
			name:        "Empty models parameter",
			modelsParam: "",
			shouldError: true,
		},
		{
			name:           "Models with empty entries",
			modelsParam:    "gpt-3.5-turbo,,gpt-4",
			expectedCount:  3,
			expectedModels: []string{"gpt-3.5-turbo", "", "gpt-4"},
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 main 函数中的模型处理逻辑
			if tt.modelsParam == "" && tt.shouldError {
				// 验证空参数应该导致错误
				return // 这在实际代码中会导致 os.Exit(1)
			}

			// 解析多个模型
			modelList := strings.Split(tt.modelsParam, ",")
			for i, m := range modelList {
				modelList[i] = strings.TrimSpace(m)
			}

			if !tt.shouldError {
				if len(modelList) != tt.expectedCount {
					t.Errorf("Expected %d models, got %d", tt.expectedCount, len(modelList))
				}

				for i, expectedModel := range tt.expectedModels {
					if i < len(modelList) && modelList[i] != expectedModel {
						t.Errorf("Model[%d]: expected '%s', got '%s'", i, expectedModel, modelList[i])
					}
				}
			}
		})
	}
}

func TestConfigCreation(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			protocol    string
			baseUrl     string
			apiKey      string
			model       string
			concurrency int
			count       int
			prompt      string
			stream      bool
			report      bool
			timeout     int
		}
		expected types.Input
	}{
		{
			name: "Complete OpenAI configuration",
			input: struct {
				protocol    string
				baseUrl     string
				apiKey      string
				model       string
				concurrency int
				count       int
				prompt      string
				stream      bool
				report      bool
				timeout     int
			}{
				protocol:    "openai",
				baseUrl:     "https://api.openai.com/v1",
				apiKey:      "sk-test123",
				model:       "gpt-3.5-turbo",
				concurrency: 5,
				count:       20,
				prompt:      "Hello, world!",
				stream:      true,
				report:      false,
				timeout:     30,
			},
			expected: types.Input{
				Protocol:    "openai",
				BaseUrl:     "https://api.openai.com/v1",
				ApiKey:      "sk-test123",
				Model:       "gpt-3.5-turbo",
				Concurrency: 5,
				Count:       20,
				Prompt:      "Hello, world!",
				Stream:      true,
				Report:      false,
				Timeout:     30 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 main 函数中创建配置的过程
			config := types.Input{
				Protocol:    tt.input.protocol,
				BaseUrl:     tt.input.baseUrl,
				ApiKey:      tt.input.apiKey,
				Model:       tt.input.model,
				Concurrency: tt.input.concurrency,
				Count:       tt.input.count,
				Prompt:      tt.input.prompt,
				Stream:      tt.input.stream,
				Report:      tt.input.report,
				Timeout:     time.Duration(tt.input.timeout) * time.Second,
			}

			// 验证所有字段
			if config.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol: expected %s, got %s", tt.expected.Protocol, config.Protocol)
			}
			if config.BaseUrl != tt.expected.BaseUrl {
				t.Errorf("BaseUrl: expected %s, got %s", tt.expected.BaseUrl, config.BaseUrl)
			}
			if config.Model != tt.expected.Model {
				t.Errorf("Model: expected %s, got %s", tt.expected.Model, config.Model)
			}
			if config.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout: expected %v, got %v", tt.expected.Timeout, config.Timeout)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		expected string
	}{
		{
			name:     "Empty protocol falls back to env detection",
			protocol: "",
			expected: "openai", // default when no env vars set
		},
		{
			name:     "Openai protocol",
			protocol: "openai",
			expected: "openai",
		},
		{
			name:     "Anthropic protocol",
			protocol: "anthropic",
			expected: "anthropic",
		},
		{
			name:     "Unknown protocol",
			protocol: "unknown",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除环境变量以确保一致的测试环境
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_BASE_URL")

			// 模拟main函数中的协议处理逻辑
			finalProtocol := tt.protocol
			if finalProtocol == "" {
				finalProtocol = detectProviderFromEnv()
			}

			if finalProtocol != tt.expected {
				t.Errorf("Expected protocol %s, got %s", tt.expected, finalProtocol)
			}
		})
	}
}

func TestValidateParameters(t *testing.T) {
	tests := []struct {
		name        string
		models      string
		baseUrl     string
		apiKey      string
		expectExit  bool
		description string
	}{
		{
			name:        "All parameters valid",
			models:      "gpt-3.5-turbo",
			baseUrl:     "https://api.openai.com",
			apiKey:      "sk-test123",
			expectExit:  false,
			description: "Valid configuration should not exit",
		},
		{
			name:        "Empty models triggers exit",
			models:      "",
			baseUrl:     "https://api.openai.com",
			apiKey:      "sk-test123",
			expectExit:  true,
			description: "Empty models should trigger exit",
		},
		{
			name:        "Empty baseUrl triggers exit",
			models:      "gpt-3.5-turbo",
			baseUrl:     "",
			apiKey:      "sk-test123",
			expectExit:  true,
			description: "Empty baseUrl should trigger exit",
		},
		{
			name:        "Empty apiKey triggers exit",
			models:      "gpt-3.5-turbo",
			baseUrl:     "https://api.openai.com",
			apiKey:      "",
			expectExit:  true,
			description: "Empty apiKey should trigger exit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟参数验证逻辑
			shouldExit := false

			// models 参数检查
			if tt.models == "" {
				shouldExit = true
			}

			// baseUrl 和 apikey 检查
			if tt.baseUrl == "" || tt.apiKey == "" {
				shouldExit = true
			}

			if shouldExit != tt.expectExit {
				t.Errorf("%s: expected exit=%t, got exit=%t", tt.description, tt.expectExit, shouldExit)
			}
		})
	}
}

func TestTimeoutHandling(t *testing.T) {
	tests := []struct {
		name            string
		timeoutSeconds  int
		expectedTimeout time.Duration
	}{
		{
			name:            "Default timeout",
			timeoutSeconds:  30,
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "Custom timeout",
			timeoutSeconds:  60,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "Short timeout",
			timeoutSeconds:  5,
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "Zero timeout",
			timeoutSeconds:  0,
			expectedTimeout: 0 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟timeout转换逻辑
			timeout := time.Duration(tt.timeoutSeconds) * time.Second

			if timeout != tt.expectedTimeout {
				t.Errorf("Expected timeout %v, got %v", tt.expectedTimeout, timeout)
			}
		})
	}
}

func TestValidateRequiredParams(t *testing.T) {
	tests := []struct {
		name        string
		models      string
		baseUrl     string
		apiKey      string
		protocol    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "All params valid",
			models:      "gpt-3.5-turbo",
			baseUrl:     "https://api.openai.com",
			apiKey:      "sk-test123",
			protocol:    "openai",
			expectError: false,
		},
		{
			name:        "Empty models",
			models:      "",
			baseUrl:     "https://api.openai.com",
			apiKey:      "sk-test123", 
			protocol:    "openai",
			expectError: true,
			errorMsg:    "models 参数必填",
		},
		{
			name:        "Empty baseUrl",
			models:      "gpt-3.5-turbo",
			baseUrl:     "",
			apiKey:      "sk-test123",
			protocol:    "openai",
			expectError: true,
			errorMsg:    "baseUrl 和 apikey 参数必填",
		},
		{
			name:        "Empty apiKey",
			models:      "gpt-3.5-turbo",
			baseUrl:     "https://api.openai.com",
			apiKey:      "",
			protocol:    "openai",
			expectError: true,
			errorMsg:    "baseUrl 和 apikey 参数必填",
		},
		{
			name:        "Both baseUrl and apiKey empty",
			models:      "gpt-3.5-turbo",
			baseUrl:     "",
			apiKey:      "",
			protocol:    "anthropic",
			expectError: true,
			errorMsg:    "baseUrl 和 apikey 参数必填",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequiredParams(tt.models, tt.baseUrl, tt.apiKey, tt.protocol)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestParseModelList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Single model",
			input:    "gpt-3.5-turbo",
			expected: []string{"gpt-3.5-turbo"},
		},
		{
			name:     "Multiple models",
			input:    "gpt-3.5-turbo,gpt-4,claude-3",
			expected: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
		},
		{
			name:     "Models with spaces",
			input:    " gpt-3.5-turbo , gpt-4 , claude-3 ",
			expected: []string{"gpt-3.5-turbo", "gpt-4", "claude-3"},
		},
		{
			name:     "Models with empty entries",
			input:    "gpt-3.5-turbo,,gpt-4",
			expected: []string{"gpt-3.5-turbo", "", "gpt-4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseModelList(tt.input)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d models, got %d", len(tt.expected), len(result))
				return
			}
			
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Model[%d]: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestResolveConfigValues(t *testing.T) {
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
		name               string
		inputProtocol      string
		inputBaseUrl       string
		inputApiKey        string
		envVars            map[string]string
		expectedProtocol   string
		expectedBaseUrl    string
		expectedApiKey     string
	}{
		{
			name:             "All command line params provided",
			inputProtocol:    "openai",
			inputBaseUrl:     "https://cmd.api.com",
			inputApiKey:      "cmd-key",
			envVars:          map[string]string{},
			expectedProtocol: "openai",
			expectedBaseUrl:  "https://cmd.api.com",
			expectedApiKey:   "cmd-key",
		},
		{
			name:             "Empty protocol, auto-detect from env",
			inputProtocol:    "",
			inputBaseUrl:     "https://cmd.api.com",
			inputApiKey:      "cmd-key",
			envVars: map[string]string{
				"OPENAI_API_KEY": "env-key",
			},
			expectedProtocol: "openai",
			expectedBaseUrl:  "https://cmd.api.com", 
			expectedApiKey:   "cmd-key",
		},
		{
			name:             "Missing baseUrl, get from env",
			inputProtocol:    "openai",
			inputBaseUrl:     "",
			inputApiKey:      "cmd-key",
			envVars: map[string]string{
				"OPENAI_BASE_URL": "https://env.api.com",
			},
			expectedProtocol: "openai",
			expectedBaseUrl:  "https://env.api.com",
			expectedApiKey:   "cmd-key",
		},
		{
			name:             "Missing apiKey, get from env",
			inputProtocol:    "anthropic",
			inputBaseUrl:     "https://cmd.api.com",
			inputApiKey:      "",
			envVars: map[string]string{
				"ANTHROPIC_API_KEY": "env-key",
			},
			expectedProtocol: "anthropic",
			expectedBaseUrl:  "https://cmd.api.com",
			expectedApiKey:   "env-key",
		},
		{
			name:             "All from env vars",
			inputProtocol:    "",
			inputBaseUrl:     "",
			inputApiKey:      "",
			envVars: map[string]string{
				"ANTHROPIC_BASE_URL": "https://env.anthropic.com",
				"ANTHROPIC_API_KEY":  "env-ant-key",
			},
			expectedProtocol: "anthropic",
			expectedBaseUrl:  "https://env.anthropic.com",
			expectedApiKey:   "env-ant-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清除所有环境变量
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_BASE_URL")
			os.Unsetenv("ANTHROPIC_API_KEY")
			os.Unsetenv("ANTHROPIC_BASE_URL")

			// 设置测试环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			protocol, baseUrl, apiKey := resolveConfigValues(tt.inputProtocol, tt.inputBaseUrl, tt.inputApiKey)

			if protocol != tt.expectedProtocol {
				t.Errorf("Protocol: expected %s, got %s", tt.expectedProtocol, protocol)
			}
			if baseUrl != tt.expectedBaseUrl {
				t.Errorf("BaseUrl: expected %s, got %s", tt.expectedBaseUrl, baseUrl)
			}
			if apiKey != tt.expectedApiKey {
				t.Errorf("ApiKey: expected %s, got %s", tt.expectedApiKey, apiKey)
			}
		})
	}
}

func TestPrintErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		expected []string
	}{
		{
			name:     "OpenAI protocol",
			protocol: "openai",
			expected: []string{
				"  OPENAI_BASE_URL - OpenAI API 基础 URL",
				"  OPENAI_API_KEY - OpenAI API 密钥",
			},
		},
		{
			name:     "Anthropic protocol",
			protocol: "anthropic", 
			expected: []string{
				"  ANTHROPIC_BASE_URL - Anthropic API 基础 URL",
				"  ANTHROPIC_API_KEY - Anthropic API 密钥",
			},
		},
		{
			name:     "Unknown protocol",
			protocol: "unknown",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于printErrorMessages函数直接输出到stdout，我们只能测试它不会panic
			// 在实际应用中，可以考虑将其重构为返回字符串的函数
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("printErrorMessages panicked: %v", r)
				}
			}()
			
			printErrorMessages(tt.protocol)
		})
	}
}

func TestCreateRunnerConfig(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		baseUrl     string
		apiKey      string
		model       string
		prompt      string
		concurrency int
		count       int
		timeout     int
		stream      bool
		report      bool
		expected    types.Input
	}{
		{
			name:        "Complete OpenAI config",
			protocol:    "openai",
			baseUrl:     "https://api.openai.com",
			apiKey:      "sk-test123",
			model:       "gpt-3.5-turbo",
			prompt:      "Hello world",
			concurrency: 5,
			count:       20,
			timeout:     30,
			stream:      true,
			report:      false,
			expected: types.Input{
				Protocol:    "openai",
				BaseUrl:     "https://api.openai.com",
				ApiKey:      "sk-test123",
				Model:       "gpt-3.5-turbo",
				Prompt:      "Hello world",
				Concurrency: 5,
				Count:       20,
				Timeout:     30 * time.Second,
				Stream:      true,
				Report:      false,
			},
		},
		{
			name:        "Anthropic config with defaults",
			protocol:    "anthropic",
			baseUrl:     "https://api.anthropic.com",
			apiKey:      "ant-test456",
			model:       "claude-3",
			prompt:      "Test prompt",
			concurrency: 3,
			count:       10,
			timeout:     60,
			stream:      false,
			report:      true,
			expected: types.Input{
				Protocol:    "anthropic",
				BaseUrl:     "https://api.anthropic.com",
				ApiKey:      "ant-test456",
				Model:       "claude-3",
				Prompt:      "Test prompt",
				Concurrency: 3,
				Count:       10,
				Timeout:     60 * time.Second,
				Stream:      false,
				Report:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createRunnerConfig(
				tt.protocol, tt.baseUrl, tt.apiKey, tt.model, tt.prompt,
				tt.concurrency, tt.count, tt.timeout, tt.stream, tt.report,
			)

			if result.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol: expected %s, got %s", tt.expected.Protocol, result.Protocol)
			}
			if result.BaseUrl != tt.expected.BaseUrl {
				t.Errorf("BaseUrl: expected %s, got %s", tt.expected.BaseUrl, result.BaseUrl)
			}
			if result.ApiKey != tt.expected.ApiKey {
				t.Errorf("ApiKey: expected %s, got %s", tt.expected.ApiKey, result.ApiKey)
			}
			if result.Model != tt.expected.Model {
				t.Errorf("Model: expected %s, got %s", tt.expected.Model, result.Model)
			}
			if result.Prompt != tt.expected.Prompt {
				t.Errorf("Prompt: expected %s, got %s", tt.expected.Prompt, result.Prompt)
			}
			if result.Concurrency != tt.expected.Concurrency {
				t.Errorf("Concurrency: expected %d, got %d", tt.expected.Concurrency, result.Concurrency)
			}
			if result.Count != tt.expected.Count {
				t.Errorf("Count: expected %d, got %d", tt.expected.Count, result.Count)
			}
			if result.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout: expected %v, got %v", tt.expected.Timeout, result.Timeout)
			}
			if result.Stream != tt.expected.Stream {
				t.Errorf("Stream: expected %t, got %t", tt.expected.Stream, result.Stream)
			}
			if result.Report != tt.expected.Report {
				t.Errorf("Report: expected %t, got %t", tt.expected.Report, result.Report)
			}
		})
	}
}

func TestCollectErrorsWithContext(t *testing.T) {
	tests := []struct {
		name        string
		modelName   string
		modelErrors []string
		expected    []string
	}{
		{
			name:        "No errors",
			modelName:   "gpt-3.5-turbo",
			modelErrors: []string{},
			expected:    []string{},
		},
		{
			name:        "Single error",
			modelName:   "gpt-4",
			modelErrors: []string{"Connection timeout"},
			expected:    []string{"[gpt-4] Connection timeout"},
		},
		{
			name:        "Multiple errors",
			modelName:   "claude-3",
			modelErrors: []string{"Rate limit exceeded", "API key invalid"},
			expected:    []string{"[claude-3] Rate limit exceeded", "[claude-3] API key invalid"},
		},
		{
			name:        "Mixed errors with empty strings",
			modelName:   "gpt-3.5-turbo",
			modelErrors: []string{"Error 1", "", "Error 2"},
			expected:    []string{"[gpt-3.5-turbo] Error 1", "[gpt-3.5-turbo] Error 2"},
		},
		{
			name:        "Only empty errors",
			modelName:   "test-model",
			modelErrors: []string{"", "", ""},
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectErrorsWithContext(tt.modelName, tt.modelErrors)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d errors, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Error[%d]: expected '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestFillResultMetadata(t *testing.T) {
	// 创建测试数据
	modelList := []string{"gpt-3.5-turbo", "gpt-4"}
	baseUrl := "https://api.openai.com"
	protocol := "openai"

	results := []*types.ReportData{
		{
			TotalRequests: 10,
			Concurrency:   3,
		},
		{
			TotalRequests: 20,
			Concurrency:   5,
		},
	}

	// 调用被测试函数
	fillResultMetadata(results, modelList, baseUrl, protocol)

	// 验证结果
	for i, result := range results {
		if result.Metadata.Model != modelList[i] {
			t.Errorf("Result[%d] Model: expected %s, got %s", i, modelList[i], result.Metadata.Model)
		}
		if result.Metadata.BaseUrl != baseUrl {
			t.Errorf("Result[%d] BaseUrl: expected %s, got %s", i, baseUrl, result.Metadata.BaseUrl)
		}
		if result.Metadata.Protocol != protocol {
			t.Errorf("Result[%d] Protocol: expected %s, got %s", i, protocol, result.Metadata.Protocol)
		}
		if result.Metadata.Timestamp == "" {
			t.Errorf("Result[%d] Timestamp should not be empty", i)
		}
		
		// 验证时间戳格式是否为RFC3339
		if _, err := time.Parse(time.RFC3339, result.Metadata.Timestamp); err != nil {
			t.Errorf("Result[%d] Timestamp format invalid: %v", i, err)
		}
	}
}

func TestConvertErrorsToPointers(t *testing.T) {
	tests := []struct {
		name     string
		errors   []string
		expected int
	}{
		{
			name:     "Empty slice",
			errors:   []string{},
			expected: 0,
		},
		{
			name:     "Single error",
			errors:   []string{"Error 1"},
			expected: 1,
		},
		{
			name:     "Multiple errors",
			errors:   []string{"Error 1", "Error 2", "Error 3"},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertErrorsToPointers(tt.errors)

			if len(result) != tt.expected {
				t.Errorf("Expected %d pointers, got %d", tt.expected, len(result))
				return
			}

			// 验证指针指向正确的值
			for i, errorPtr := range result {
				if errorPtr == nil {
					t.Errorf("Pointer[%d] should not be nil", i)
					continue
				}
				if *errorPtr != tt.errors[i] {
					t.Errorf("Pointer[%d]: expected '%s', got '%s'", i, tt.errors[i], *errorPtr)
				}
			}
		})
	}
}

func TestGenerateReportsIfEnabled(t *testing.T) {
	tests := []struct {
		name       string
		reportFlag bool
		results    []*types.ReportData
		expectCall bool
	}{
		{
			name:       "Report disabled",
			reportFlag: false,
			results:    []*types.ReportData{{}},
			expectCall: false,
		},
		{
			name:       "No results",
			reportFlag: true,
			results:    []*types.ReportData{},
			expectCall: false,
		},
		{
			name:       "Report enabled with results",
			reportFlag: true,
			results: []*types.ReportData{
				{
					TotalRequests: 10,
					Concurrency:   3,
				},
			},
			expectCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 由于这个函数调用外部依赖（report.NewReportManager），
			// 在实际测试中可能需要依赖注入或模拟
			// 这里我们只测试函数不会panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("generateReportsIfEnabled panicked: %v", r)
				}
			}()

			err := generateReportsIfEnabled(tt.reportFlag, tt.results)
			
			// 如果不应该调用，检查是否返回nil error（因为是空操作）
			if !tt.expectCall && err != nil {
				t.Errorf("Expected no error when report disabled or no results, got: %v", err)
			}
		})
	}
}

func TestProcessModelExecution(t *testing.T) {
	tests := []struct {
		name               string
		modelName          string
		config             types.Input
		displayer          *display.Displayer
		completedRequests  int
		totalRequests      int
		expectedResult     bool
	}{
		{
			name:      "Successful execution",
			modelName: "gpt-3.5-turbo",
			config: types.Input{
				Prompt:      "test prompt",
				Protocol:    "openai",
				BaseUrl:     "https://api.openai.com",
				ApiKey:      "test-key",
				Timeout:     30,
				Count:       1,
				Concurrency: 1,
			},
			displayer:         display.New(),
			completedRequests: 0,
			totalRequests:     1,
			expectedResult:    true,
		},
		{
			name:      "Another model execution",
			modelName: "gpt-4",
			config: types.Input{
				Prompt:      "test prompt",
				Protocol:    "openai",
				BaseUrl:     "https://api.openai.com",
				ApiKey:      "test-key",
				Timeout:     30,
				Count:       1,
				Concurrency: 1,
			},
			displayer:         display.New(),
			completedRequests: 0,
			totalRequests:     1,
			expectedResult:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置模型配置
			tt.config.Model = tt.modelName
			
			result, errorMessages, err := processModelExecution(
				tt.modelName, 
				tt.config, 
				tt.displayer, 
				tt.completedRequests, 
				tt.totalRequests,
			)

			// 检查基本返回值
			if tt.expectedResult {
				if result == nil && err != nil {
					t.Errorf("Expected successful result, but got error: %v", err)
				}
			}

			// 验证 errorMessages 不是 nil
			if errorMessages == nil {
				t.Errorf("errorMessages should not be nil")
			}

			// 确保没有 panic，这是最基本的要求
		})
	}
}

func TestExecuteModelsTestSuite(t *testing.T) {
	tests := []struct {
		name         string
		modelList    []string
		protocol     string
		baseUrl      string
		apiKey       string
		prompt       string
		concurrency  int
		count        int
		timeout      int
		stream       bool
		reportFlag   bool
		displayer    *display.Displayer
		expectedLen  int
		expectError  bool
	}{
		{
			name:        "Single model execution",
			modelList:   []string{"gpt-3.5-turbo"},
			protocol:    "openai",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			prompt:      "test prompt",
			concurrency: 1,
			count:       1,
			timeout:     30,
			stream:      false,
			reportFlag:  false,
			displayer:   display.New(),
			expectedLen: 1,
			expectError: false,
		},
		{
			name:        "Multiple models execution",
			modelList:   []string{"gpt-3.5-turbo", "gpt-4"},
			protocol:    "openai",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			prompt:      "test prompt",
			concurrency: 1,
			count:       1,
			timeout:     30,
			stream:      false,
			reportFlag:  false,
			displayer:   display.New(),
			expectedLen: 2,
			expectError: false,
		},
		{
			name:        "Empty models list",
			modelList:   []string{},
			protocol:    "openai",
			baseUrl:     "https://api.openai.com",
			apiKey:      "test-key",
			prompt:      "test prompt",
			concurrency: 1,
			count:       1,
			timeout:     30,
			stream:      false,
			reportFlag:  false,
			displayer:   display.New(),
			expectedLen: 0,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, errorMessages, err := executeModelsTestSuite(
				tt.modelList,
				tt.protocol,
				tt.baseUrl,
				tt.apiKey,
				tt.prompt,
				tt.concurrency,
				tt.count,
				tt.timeout,
				tt.stream,
				tt.reportFlag,
				tt.displayer,
			)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if len(results) != tt.expectedLen {
				t.Errorf("Expected %d results, got %d", tt.expectedLen, len(results))
			}

			// All tests should complete without panicking
			_ = errorMessages // Use the variable to avoid unused variable warning
		})
	}
}
