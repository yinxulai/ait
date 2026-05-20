package types

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	ProtocolOpenAICompletions = "openai-completions"
	ProtocolOpenAIResponses   = "openai-responses"
	ProtocolAnthropicMessages = "anthropic-messages"
)

func NormalizeProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "", "openai", ProtocolOpenAICompletions:
		return ProtocolOpenAICompletions
	case ProtocolOpenAIResponses:
		return ProtocolOpenAIResponses
	case "anthropic", ProtocolAnthropicMessages:
		return ProtocolAnthropicMessages
	default:
		return strings.TrimSpace(protocol)
	}
}

func DefaultEndpointURL(protocol string) string {
	switch NormalizeProtocol(protocol) {
	case ProtocolOpenAICompletions:
		return "https://api.openai.com/v1/chat/completions"
	case ProtocolOpenAIResponses:
		return "https://api.openai.com/v1/responses"
	case ProtocolAnthropicMessages:
		return "https://api.anthropic.com/v1/messages"
	default:
		return ""
	}
}

func ResolveEndpointURL(protocol, endpointURL, baseURL string) string {
	resolved := strings.TrimSpace(endpointURL)
	if resolved != "" {
		return resolved
	}

	resolved = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if resolved == "" {
		return DefaultEndpointURL(protocol)
	}

	switch NormalizeProtocol(protocol) {
	case ProtocolOpenAICompletions:
		if strings.HasSuffix(resolved, "/chat/completions") {
			return resolved
		}
		if strings.HasSuffix(resolved, "/v1") {
			return resolved + "/chat/completions"
		}
		return resolved + "/v1/chat/completions"
	case ProtocolOpenAIResponses:
		if strings.HasSuffix(resolved, "/responses") {
			return resolved
		}
		if strings.HasSuffix(resolved, "/v1") {
			return resolved + "/responses"
		}
		return resolved + "/v1/responses"
	case ProtocolAnthropicMessages:
		if strings.HasSuffix(resolved, "/v1/messages") {
			return resolved
		}
		return resolved + "/v1/messages"
	default:
		return resolved
	}
}

// PromptSource 需要前向声明，实际定义在 prompt 包中
type PromptSource interface {
	GetSystemContent() string
	GetRandomContent() string
	GetContentByIndex(index int) string
	Count() int
}

// Input 测试配置信息 - 统一的配置结构
type Input struct {
	Protocol     string        `json:"protocol"`
	EndpointURL  string        `json:"endpoint_url,omitempty"`
	BaseUrl      string        `json:"base_url,omitempty"`
	ProxyURL     string        `json:"proxy_url,omitempty"`
	ApiKey       string        `json:"api_key,omitempty"`
	Model        string        `json:"model"`
	Concurrency  int           `json:"concurrency,omitempty"`
	Count        int           `json:"count,omitempty"`
	Stream       bool          `json:"stream,omitempty"`
	Thinking     bool          `json:"thinking,omitempty"` // 是否开启 thinking 模式（仅支持 OpenAI 协议）
	Turbo        bool          `json:"turbo,omitempty"` // 是否启用 Turbo 模式
	TurboConfig  TurboConfig   `json:"turbo_config,omitempty"` // Turbo 模式配置
	PromptMode   string        `json:"prompt_mode,omitempty"`
	PromptText   string        `json:"prompt_text,omitempty"`
	PromptFile   string        `json:"prompt_file,omitempty"`
	PromptLength int           `json:"prompt_length,omitempty"`
	PromptSource PromptSource  `json:"-"` // 运行态字段，不直接持久化
	Report       bool          `json:"report,omitempty"` // 是否生成报告文件
	Timeout      time.Duration `json:"timeout,omitempty"` // 请求超时时间
	Log          bool          `json:"log,omitempty"` // 是否开启详细日志记录
}

func (i Input) NormalizedProtocol() string {
	return NormalizeProtocol(i.Protocol)
}

func (i Input) ResolvedEndpointURL() string {
	return ResolveEndpointURL(i.Protocol, i.EndpointURL, i.BaseUrl)
}

// StatsData 实时测试统计数据 - runner 内部使用的统计结构
// 用于在测试过程中实时收集和更新统计信息
type StatsData struct {
	// 基础统计
	CompletedCount int // 已完成请求数
	FailedCount    int // 失败请求数

	// 时间指标 - 原始数据收集
	TTFTs      []time.Duration // 所有首个token响应时间 (Time to First Token)
	TotalTimes []time.Duration // 所有总耗时

	// 网络指标 - 原始数据收集
	DNSTimes          []time.Duration // 所有DNS解析时间
	ConnectTimes      []time.Duration // 所有TCP连接时间
	TLSHandshakeTimes []time.Duration // 所有TLS握手时间

	// 服务性能指标 - 原始数据收集（与 ReportData 命名对齐）
	InputTokenCounts    []int // 所有 prompt/input token 数量
	CachedInputTokenCounts []int // 所有缓存命中的输入 token 数量
	OutputTokenCounts   []int // 所有 completion/output token 数量 (用于TPS计算)
	ThinkingTokenCounts []int // 所有思考/推理 token 数量
	CacheHitRates       []float64 // 所有请求的缓存命中率

	// 错误信息
	ErrorMessages []string // 所有错误信息

	// 测试控制
	StartTime   time.Time     // 测试开始时间
	ElapsedTime time.Duration // 已经过时间
}

// ReportData runner 返回的统一测试结果数据结构
// 包含经过统计分析后的最终结果，供 display 和 report 模块使用
// 支持 JSON 序列化用于报告生成
type ReportData struct {
	// 基础测试信息
	TotalRequests int           `json:"total_requests"` // 总请求数
	Concurrency   int           `json:"concurrency"`    // 并发数
	IsStream      bool          `json:"is_stream"`      // 是否为流式请求
	IsThinking    bool          `json:"is_thinking"`    // 是否启用思考模式
	TotalTime     time.Duration `json:"total_time"`     // 总测试时间

	// 扁平化的元数据信息
	Timestamp string `json:"timestamp"` // 测试时间戳
	Protocol  string `json:"protocol"`  // 协议类型
	Model     string `json:"model"`     // 模型名称
	EndpointURL string `json:"endpoint_url,omitempty"` // 完整接口地址
	BaseUrl   string `json:"base_url"`  // 基础URL

	// 时间性能指标 - 统计结果
	AvgTotalTime time.Duration `json:"avg_total_time"` // 平均总耗时
	MinTotalTime time.Duration `json:"min_total_time"` // 最小总耗时
	MaxTotalTime time.Duration `json:"max_total_time"` // 最大总耗时

	// 网络性能指标 - 统计结果
	AvgDNSTime          time.Duration `json:"avg_dns_time"`           // 平均DNS解析时间
	MinDNSTime          time.Duration `json:"min_dns_time"`           // 最小DNS解析时间
	MaxDNSTime          time.Duration `json:"max_dns_time"`           // 最大DNS解析时间
	AvgConnectTime      time.Duration `json:"avg_connect_time"`       // 平均TCP连接时间
	MinConnectTime      time.Duration `json:"min_connect_time"`       // 最小TCP连接时间
	MaxConnectTime      time.Duration `json:"max_connect_time"`       // 最大TCP连接时间
	AvgTLSHandshakeTime time.Duration `json:"avg_tls_handshake_time"` // 平均TLS握手时间
	MinTLSHandshakeTime time.Duration `json:"min_tls_handshake_time"` // 最小TLS握手时间
	MaxTLSHandshakeTime time.Duration `json:"max_tls_handshake_time"` // 最大TLS握手时间
	TargetIP            string        `json:"target_ip"`              // 目标IP地址

	// 服务性能指标 - 统计结果
	AvgTTFT             time.Duration `json:"avg_ttft"`               // 平均首个token响应时间
	MinTTFT             time.Duration `json:"min_ttft"`               // 最小首个token响应时间
	MaxTTFT             time.Duration `json:"max_ttft"`               // 最大首个token响应时间
	AvgTPOT             time.Duration `json:"avg_tpot"`               // 平均每个输出token的耗时（除首token外）
	MinTPOT             time.Duration `json:"min_tpot"`               // 最小每个输出token的耗时
	MaxTPOT             time.Duration `json:"max_tpot"`               // 最大每个输出token的耗时
	AvgInputTokenCount  int           `json:"avg_input_token_count"`  // 平均输入token数量
	MinInputTokenCount  int           `json:"min_input_token_count"`  // 最小输入token数量
	MaxInputTokenCount  int           `json:"max_input_token_count"`  // 最大输入token数量
	AvgCachedInputTokenCount int       `json:"avg_cached_input_token_count"` // 平均缓存命中的输入 token 数量
	MinCachedInputTokenCount int       `json:"min_cached_input_token_count"` // 最小缓存命中的输入 token 数量
	MaxCachedInputTokenCount int       `json:"max_cached_input_token_count"` // 最大缓存命中的输入 token 数量
	AvgOutputTokenCount int           `json:"avg_output_token_count"` // 平均输出token数量
	MinOutputTokenCount int           `json:"min_output_token_count"` // 最小输出token数量
	MaxOutputTokenCount int           `json:"max_output_token_count"` // 最大输出token数量
	AvgThinkingTokenCount int          `json:"avg_thinking_token_count"` // 平均思考token数量
	MinThinkingTokenCount int          `json:"min_thinking_token_count"` // 最小思考token数量
	MaxThinkingTokenCount int          `json:"max_thinking_token_count"` // 最大思考token数量
	AvgCacheHitRate     float64       `json:"avg_cache_hit_rate"`      // 平均缓存命中率
	MinCacheHitRate     float64       `json:"min_cache_hit_rate"`      // 最小缓存命中率
	MaxCacheHitRate     float64       `json:"max_cache_hit_rate"`      // 最大缓存命中率
	AvgTPS              float64       `json:"avg_tps"`                // 平均输出 TPS (仅输出 tokens per second)
	MinTPS              float64       `json:"min_tps"`                // 最小输出 TPS
	MaxTPS              float64       `json:"max_tps"`                // 最大输出 TPS

	// 吞吐量指标 - 统计结果
	AvgTotalThroughputTPS float64 `json:"avg_total_throughput_tps"` // 平均吞吐 TPS (输入+输出 tokens per second)
	MinTotalThroughputTPS float64 `json:"min_total_throughput_tps"` // 最小吞吐 TPS
	MaxTotalThroughputTPS float64 `json:"max_total_throughput_tps"` // 最大吞吐 TPS

	// 标准差指标 - 统计结果
	StdDevTotalTime        time.Duration `json:"stddev_total_time"`          // 总耗时标准差
	StdDevTTFT             time.Duration `json:"stddev_ttft"`                // TTFT 标准差
	StdDevTPOT             time.Duration `json:"stddev_tpot"`                // TPOT 标准差
	StdDevInputTokenCount  float64       `json:"stddev_input_token_count"`   // 输入 Token 数标准差
	StdDevCachedInputTokenCount float64  `json:"stddev_cached_input_token_count"` // 缓存命中输入 Token 数标准差
	StdDevOutputTokenCount float64       `json:"stddev_output_token_count"`  // 输出 Token 数标准差
	StdDevThinkingTokenCount float64     `json:"stddev_thinking_token_count"` // 思考 Token 数标准差
	StdDevCacheHitRate     float64       `json:"stddev_cache_hit_rate"`      // 缓存命中率标准差
	StdDevTPS              float64       `json:"stddev_tps"`                 // 输出 TPS 标准差
	StdDevTotalThroughputTPS float64     `json:"stddev_total_throughput_tps"` // 吞吐 TPS 标准差

	// 可靠性指标 - 统计结果
	ErrorRate   float64 `json:"error_rate"`   // 错误率 (%)
	SuccessRate float64 `json:"success_rate"` // 成功率 (%)
}

type TaskDefinition struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Input     Input     `json:"input"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TaskOverview struct {
	TaskDefinition
	LatestRun *TaskRunSummary `json:"latest_run,omitempty"`
}

type TaskRunSummary struct {
	RunID                string        `json:"run_id"`
	TaskID               string        `json:"task_id"`
	Mode                 string        `json:"mode"`
	Status               string        `json:"status"`
	Protocol             string        `json:"protocol"`
	Model                string        `json:"model"`
	StartedAt            time.Time     `json:"started_at"`
	FinishedAt           time.Time     `json:"finished_at"`
	SuccessRate          float64       `json:"success_rate"`
	AvgTTFT              time.Duration `json:"avg_ttft"`
	AvgTPS               float64       `json:"avg_tps"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	MaxStableConcurrency int           `json:"max_stable_concurrency,omitempty"`
	ErrorSummary         string        `json:"error_summary,omitempty"`
}

type RequestMetrics struct {
	Index            int           `json:"index"`
	Success          bool          `json:"success"`
	TotalTime        time.Duration `json:"total_time"`
	TTFT             time.Duration `json:"ttft"`
	TPS              float64       `json:"tps"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	CachedTokens     int           `json:"cached_tokens"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	DNSTime          time.Duration `json:"dns_time"`
	ConnectTime      time.Duration `json:"connect_time"`
	TLSTime          time.Duration `json:"tls_time"`
	TargetIP         string        `json:"target_ip"`
	ErrorMessage     string        `json:"error_message,omitempty"`
	RequestBody      string        `json:"request_body,omitempty"`
	ResponseBody     string        `json:"response_body,omitempty"`
}

type TurboConfig struct {
	InitConcurrency int           `json:"init_concurrency"`
	MaxConcurrency  int           `json:"max_concurrency"`
	StepSize        int           `json:"step_size"`
	LevelRequests   int           `json:"level_requests"`
	MinSuccessRate  float64       `json:"min_success_rate"`
	MaxLatency      time.Duration `json:"max_latency"`
}

type TurboLevelResult struct {
	Concurrency   int           `json:"concurrency"`
	TotalRequests int           `json:"total_requests"`
	SuccessCount  int           `json:"success_count"`
	SuccessRate   float64       `json:"success_rate"`
	AvgTPS        float64       `json:"avg_tps"`
	PeakTPS       float64       `json:"peak_tps"`
	AvgTTFT       time.Duration `json:"avg_ttft"`
	CacheHitRate  float64       `json:"cache_hit_rate"`
	AvgTotalTime  time.Duration `json:"avg_total_time"`
	StdDevTPS     float64       `json:"stddev_tps"`
	Stable        bool          `json:"stable"`
	StopReason    string        `json:"stop_reason,omitempty"`
}

type TurboResult struct {
	Config               TurboConfig        `json:"config"`
	Levels               []TurboLevelResult `json:"levels"`
	MaxStableConcurrency int                `json:"max_stable_concurrency"`
	PeakTPS              float64            `json:"peak_tps"`
	StopReason           string             `json:"stop_reason"`
	ProbeDuration        time.Duration      `json:"probe_duration"`
	Model                string             `json:"model"`
	Protocol             string             `json:"protocol"`
	EndpointURL          string             `json:"endpoint_url"`
	Timestamp            string             `json:"timestamp"`
}

// MarshalJSON 自定义 JSON 序列化，将 time.Duration 转换为字符串
func (r *ReportData) MarshalJSON() ([]byte, error) {
	// 自定义序列化，所有 time.Duration 字段转为字符串
	type Alias ReportData
	return json.Marshal(&struct {
		*Alias
		TotalTime         string `json:"total_time"`
		AvgTotalTime      string `json:"avg_total_time"`
		MinTotalTime      string `json:"min_total_time"`
		MaxTotalTime      string `json:"max_total_time"`
		AvgDNSTime        string `json:"avg_dns_time"`
		MinDNSTime        string `json:"min_dns_time"`
		MaxDNSTime        string `json:"max_dns_time"`
		AvgConnectTime    string `json:"avg_connect_time"`
		MinConnectTime    string `json:"min_connect_time"`
		MaxConnectTime    string `json:"max_connect_time"`
		AvgTLSHandshakeTime string `json:"avg_tls_handshake_time"`
		MinTLSHandshakeTime string `json:"min_tls_handshake_time"`
		MaxTLSHandshakeTime string `json:"max_tls_handshake_time"`
		AvgTTFT           string `json:"avg_ttft"`
		MinTTFT           string `json:"min_ttft"`
		MaxTTFT           string `json:"max_ttft"`
		AvgTPOT           string `json:"avg_tpot"`
		MinTPOT           string `json:"min_tpot"`
		MaxTPOT           string `json:"max_tpot"`
		StdDevTotalTime   string `json:"stddev_total_time"`
		StdDevTTFT        string `json:"stddev_ttft"`
		StdDevTPOT        string `json:"stddev_tpot"`
	}{
		Alias:              (*Alias)(r),
		TotalTime:          r.TotalTime.String(),
		AvgTotalTime:       r.AvgTotalTime.String(),
		MinTotalTime:       r.MinTotalTime.String(),
		MaxTotalTime:       r.MaxTotalTime.String(),
		AvgDNSTime:         r.AvgDNSTime.String(),
		MinDNSTime:         r.MinDNSTime.String(),
		MaxDNSTime:         r.MaxDNSTime.String(),
		AvgConnectTime:     r.AvgConnectTime.String(),
		MinConnectTime:     r.MinConnectTime.String(),
		MaxConnectTime:     r.MaxConnectTime.String(),
		AvgTLSHandshakeTime: r.AvgTLSHandshakeTime.String(),
		MinTLSHandshakeTime: r.MinTLSHandshakeTime.String(),
		MaxTLSHandshakeTime: r.MaxTLSHandshakeTime.String(),
		AvgTTFT:            formatTTFT(r.AvgTTFT, r.IsStream),
		MinTTFT:            formatTTFT(r.MinTTFT, r.IsStream),
		MaxTTFT:            formatTTFT(r.MaxTTFT, r.IsStream),
		AvgTPOT:            formatTPOT(r.AvgTPOT, r.IsStream),
		MinTPOT:            formatTPOT(r.MinTPOT, r.IsStream),
		MaxTPOT:            formatTPOT(r.MaxTPOT, r.IsStream),
		StdDevTotalTime:    r.StdDevTotalTime.String(),
		StdDevTTFT:         formatTTFT(r.StdDevTTFT, r.IsStream),
		StdDevTPOT:         formatTPOT(r.StdDevTPOT, r.IsStream),
	})
}

// UnmarshalJSON 自定义 JSON 反序列化，将字符串形式的 Duration 还原为 time.Duration。
// 与 MarshalJSON 配对使用，确保持久化后的数据能正确加载。
func (r *ReportData) UnmarshalJSON(data []byte) error {
	type Alias ReportData
	aux := &struct {
		*Alias
		TotalTime           string `json:"total_time"`
		AvgTotalTime        string `json:"avg_total_time"`
		MinTotalTime        string `json:"min_total_time"`
		MaxTotalTime        string `json:"max_total_time"`
		AvgDNSTime          string `json:"avg_dns_time"`
		MinDNSTime          string `json:"min_dns_time"`
		MaxDNSTime          string `json:"max_dns_time"`
		AvgConnectTime      string `json:"avg_connect_time"`
		MinConnectTime      string `json:"min_connect_time"`
		MaxConnectTime      string `json:"max_connect_time"`
		AvgTLSHandshakeTime string `json:"avg_tls_handshake_time"`
		MinTLSHandshakeTime string `json:"min_tls_handshake_time"`
		MaxTLSHandshakeTime string `json:"max_tls_handshake_time"`
		AvgTTFT             string `json:"avg_ttft"`
		MinTTFT             string `json:"min_ttft"`
		MaxTTFT             string `json:"max_ttft"`
		AvgTPOT             string `json:"avg_tpot"`
		MinTPOT             string `json:"min_tpot"`
		MaxTPOT             string `json:"max_tpot"`
		StdDevTotalTime     string `json:"stddev_total_time"`
		StdDevTTFT          string `json:"stddev_ttft"`
		StdDevTPOT          string `json:"stddev_tpot"`
	}{Alias: (*Alias)(r)}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	parseDur := func(s string) time.Duration {
		if s == "" || s == "-" {
			return 0
		}
		d, _ := time.ParseDuration(s)
		return d
	}

	r.TotalTime = parseDur(aux.TotalTime)
	r.AvgTotalTime = parseDur(aux.AvgTotalTime)
	r.MinTotalTime = parseDur(aux.MinTotalTime)
	r.MaxTotalTime = parseDur(aux.MaxTotalTime)
	r.AvgDNSTime = parseDur(aux.AvgDNSTime)
	r.MinDNSTime = parseDur(aux.MinDNSTime)
	r.MaxDNSTime = parseDur(aux.MaxDNSTime)
	r.AvgConnectTime = parseDur(aux.AvgConnectTime)
	r.MinConnectTime = parseDur(aux.MinConnectTime)
	r.MaxConnectTime = parseDur(aux.MaxConnectTime)
	r.AvgTLSHandshakeTime = parseDur(aux.AvgTLSHandshakeTime)
	r.MinTLSHandshakeTime = parseDur(aux.MinTLSHandshakeTime)
	r.MaxTLSHandshakeTime = parseDur(aux.MaxTLSHandshakeTime)
	r.AvgTTFT = parseDur(aux.AvgTTFT)
	r.MinTTFT = parseDur(aux.MinTTFT)
	r.MaxTTFT = parseDur(aux.MaxTTFT)
	r.AvgTPOT = parseDur(aux.AvgTPOT)
	r.MinTPOT = parseDur(aux.MinTPOT)
	r.MaxTPOT = parseDur(aux.MaxTPOT)
	r.StdDevTotalTime = parseDur(aux.StdDevTotalTime)
	r.StdDevTTFT = parseDur(aux.StdDevTTFT)
	r.StdDevTPOT = parseDur(aux.StdDevTPOT)
	return nil
}

// formatTTFT 格式化 TTFT 字段，非流式模式返回 "-"
func formatTTFT(duration time.Duration, isStream bool) string {
	if !isStream {
		return "-"
	}
	return duration.String()
}

// formatTPOT 格式化 TPOT 字段，非流式模式返回 "-"
func formatTPOT(duration time.Duration, isStream bool) string {
	if !isStream {
		return "-"
	}
	return duration.String()
}
