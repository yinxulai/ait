package types

import (
	"encoding/json"
	"time"
)

// PromptSource 需要前向声明，实际定义在 prompt 包中
type PromptSource interface {
	GetRandomContent() string
	GetContentByIndex(index int) string
	Count() int
}

// Input 测试配置信息 - 统一的配置结构
type Input struct {
	Protocol     string
	BaseUrl      string
	ApiKey       string
	Model        string // 多个模型列表
	Concurrency  int
	Count        int
	Stream       bool
	PromptSource PromptSource  // 改为使用PromptSource而不是简单字符串
	Report       bool          // 是否生成报告文件
	Timeout      time.Duration // 请求超时时间
	Log          bool          // 是否开启详细日志记录
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

	// 服务性能指标 - 原始数据收集
	TokenCounts []int // 所有 completion token 数量 (用于TPS计算)

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
	TotalTime     time.Duration `json:"total_time"`     // 总测试时间

	// 元数据信息 - 用于报告
	Metadata struct {
		Timestamp string `json:"timestamp"`  // 测试时间戳
		Protocol  string `json:"protocol"`   // 协议类型
		Model     string `json:"model"`      // 模型名称
		BaseUrl   string `json:"base_url"`   // 基础URL
	} `json:"metadata"`

	// 时间性能指标 - 统计结果
	TimeMetrics struct {
		AvgTotalTime time.Duration `json:"avg_total_time"` // 平均总耗时
		MinTotalTime time.Duration `json:"min_total_time"` // 最小总耗时
		MaxTotalTime time.Duration `json:"max_total_time"` // 最大总耗时
	} `json:"time_metrics"`

	// 网络性能指标 - 统计结果
	NetworkMetrics struct {
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
	} `json:"network_metrics"`

	// 服务性能指标 - 统计结果
	ContentMetrics struct {
		AvgTTFT       time.Duration `json:"avg_ttft"`        // 平均首个token响应时间
		MinTTFT       time.Duration `json:"min_ttft"`        // 最小首个token响应时间
		MaxTTFT       time.Duration `json:"max_ttft"`        // 最大首个token响应时间
		AvgTPOT       time.Duration `json:"avg_tpot"`        // 平均每个输出token的耗时（除首token外）
		MinTPOT       time.Duration `json:"min_tpot"`        // 最小每个输出token的耗时
		MaxTPOT       time.Duration `json:"max_tpot"`        // 最大每个输出token的耗时
		AvgTokenCount int           `json:"avg_token_count"` // 平均token数量
		MinTokenCount int           `json:"min_token_count"` // 最小token数量
		MaxTokenCount int           `json:"max_token_count"` // 最大token数量
		AvgTPS        float64       `json:"avg_tps"`         // 平均每秒token数 (Tokens Per Second)
		MinTPS        float64       `json:"min_tps"`         // 最小每秒token数
		MaxTPS        float64       `json:"max_tps"`         // 最大每秒token数
	} `json:"content_metrics"`

	// 可靠性指标 - 统计结果
	ReliabilityMetrics struct {
		ErrorRate   float64 `json:"error_rate"`   // 错误率 (%)
		SuccessRate float64 `json:"success_rate"` // 成功率 (%)
	} `json:"reliability_metrics"`
}

// MarshalJSON 自定义 JSON 序列化，将 time.Duration 转换为字符串
func (r *ReportData) MarshalJSON() ([]byte, error) {
	type Alias ReportData
	return json.Marshal(&struct {
		*Alias
		TotalTime string `json:"total_time"`
		TimeMetrics struct {
			AvgTotalTime string `json:"avg_total_time"`
			MinTotalTime string `json:"min_total_time"`
			MaxTotalTime string `json:"max_total_time"`
		} `json:"time_metrics"`
		NetworkMetrics struct {
			AvgDNSTime          string `json:"avg_dns_time"`
			MinDNSTime          string `json:"min_dns_time"`
			MaxDNSTime          string `json:"max_dns_time"`
			AvgConnectTime      string `json:"avg_connect_time"`
			MinConnectTime      string `json:"min_connect_time"`
			MaxConnectTime      string `json:"max_connect_time"`
			AvgTLSHandshakeTime string `json:"avg_tls_handshake_time"`
			MinTLSHandshakeTime string `json:"min_tls_handshake_time"`
			MaxTLSHandshakeTime string `json:"max_tls_handshake_time"`
			TargetIP            string `json:"target_ip"`
		} `json:"network_metrics"`
		ContentMetrics struct {
			AvgTTFT       string  `json:"avg_ttft"`
			MinTTFT       string  `json:"min_ttft"`
			MaxTTFT       string  `json:"max_ttft"`
			AvgTPOT       string  `json:"avg_tpot"`
			MinTPOT       string  `json:"min_tpot"`
			MaxTPOT       string  `json:"max_tpot"`
			AvgTokenCount int     `json:"avg_token_count"`
			MinTokenCount int     `json:"min_token_count"`
			MaxTokenCount int     `json:"max_token_count"`
			AvgTPS        float64 `json:"avg_tps"`
			MinTPS        float64 `json:"min_tps"`
			MaxTPS        float64 `json:"max_tps"`
		} `json:"content_metrics"`
	}{
		Alias:     (*Alias)(r),
		TotalTime: r.TotalTime.String(),
		TimeMetrics: struct {
			AvgTotalTime string `json:"avg_total_time"`
			MinTotalTime string `json:"min_total_time"`
			MaxTotalTime string `json:"max_total_time"`
		}{
			AvgTotalTime: r.TimeMetrics.AvgTotalTime.String(),
			MinTotalTime: r.TimeMetrics.MinTotalTime.String(),
			MaxTotalTime: r.TimeMetrics.MaxTotalTime.String(),
		},
		NetworkMetrics: struct {
			AvgDNSTime          string `json:"avg_dns_time"`
			MinDNSTime          string `json:"min_dns_time"`
			MaxDNSTime          string `json:"max_dns_time"`
			AvgConnectTime      string `json:"avg_connect_time"`
			MinConnectTime      string `json:"min_connect_time"`
			MaxConnectTime      string `json:"max_connect_time"`
			AvgTLSHandshakeTime string `json:"avg_tls_handshake_time"`
			MinTLSHandshakeTime string `json:"min_tls_handshake_time"`
			MaxTLSHandshakeTime string `json:"max_tls_handshake_time"`
			TargetIP            string `json:"target_ip"`
		}{
			AvgDNSTime:          r.NetworkMetrics.AvgDNSTime.String(),
			MinDNSTime:          r.NetworkMetrics.MinDNSTime.String(),
			MaxDNSTime:          r.NetworkMetrics.MaxDNSTime.String(),
			AvgConnectTime:      r.NetworkMetrics.AvgConnectTime.String(),
			MinConnectTime:      r.NetworkMetrics.MinConnectTime.String(),
			MaxConnectTime:      r.NetworkMetrics.MaxConnectTime.String(),
			AvgTLSHandshakeTime: r.NetworkMetrics.AvgTLSHandshakeTime.String(),
			MinTLSHandshakeTime: r.NetworkMetrics.MinTLSHandshakeTime.String(),
			MaxTLSHandshakeTime: r.NetworkMetrics.MaxTLSHandshakeTime.String(),
			TargetIP:            r.NetworkMetrics.TargetIP,
		},
		ContentMetrics: struct {
			AvgTTFT       string  `json:"avg_ttft"`
			MinTTFT       string  `json:"min_ttft"`
			MaxTTFT       string  `json:"max_ttft"`
			AvgTPOT       string  `json:"avg_tpot"`
			MinTPOT       string  `json:"min_tpot"`
			MaxTPOT       string  `json:"max_tpot"`
			AvgTokenCount int     `json:"avg_token_count"`
			MinTokenCount int     `json:"min_token_count"`
			MaxTokenCount int     `json:"max_token_count"`
			AvgTPS        float64 `json:"avg_tps"`
			MinTPS        float64 `json:"min_tps"`
			MaxTPS        float64 `json:"max_tps"`
		}{
			AvgTTFT:       formatTTFT(r.ContentMetrics.AvgTTFT, r.IsStream),
			MinTTFT:       formatTTFT(r.ContentMetrics.MinTTFT, r.IsStream),
			MaxTTFT:       formatTTFT(r.ContentMetrics.MaxTTFT, r.IsStream),
			AvgTPOT:       formatTPOT(r.ContentMetrics.AvgTPOT, r.IsStream),
			MinTPOT:       formatTPOT(r.ContentMetrics.MinTPOT, r.IsStream),
			MaxTPOT:       formatTPOT(r.ContentMetrics.MaxTPOT, r.IsStream),
			AvgTokenCount: r.ContentMetrics.AvgTokenCount,
			MinTokenCount: r.ContentMetrics.MinTokenCount,
			MaxTokenCount: r.ContentMetrics.MaxTokenCount,
			AvgTPS:        r.ContentMetrics.AvgTPS,
			MinTPS:        r.ContentMetrics.MinTPS,
			MaxTPS:        r.ContentMetrics.MaxTPS,
		},
	})
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
