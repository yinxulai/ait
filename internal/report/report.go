package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TestConfig 测试配置信息
type TestConfig struct {
	Provider    string
	BaseUrl     string
	ApiKey      string
	Model       string
	Concurrency int
	Count       int
	Stream      bool
	Prompt      string
}

// TestResult 测试结果数据
type TestResult struct {
	// 基础测试信息
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration

	// 时间性能指标
	TimeMetrics struct {
		AvgTotalTime time.Duration
		MinTotalTime time.Duration
		MaxTotalTime time.Duration
	}

	// 网络性能指标
	NetworkMetrics struct {
		AvgDNSTime          time.Duration
		MinDNSTime          time.Duration
		MaxDNSTime          time.Duration
		AvgConnectTime      time.Duration
		MinConnectTime      time.Duration
		MaxConnectTime      time.Duration
		AvgTLSHandshakeTime time.Duration
		MinTLSHandshakeTime time.Duration
		MaxTLSHandshakeTime time.Duration
		TargetIP            string
	}

	// 服务性能指标
	ContentMetrics struct {
		AvgTTFT       time.Duration
		MinTTFT       time.Duration
		MaxTTFT       time.Duration
		AvgTokenCount int
		MinTokenCount int
		MaxTokenCount int
		AvgTPS        float64
		MinTPS        float64
		MaxTPS        float64
	}

	// 可靠性指标
	ReliabilityMetrics struct {
		ErrorRate   float64
		SuccessRate float64
	}
}

// ReportData JSON 报告数据结构
type ReportData struct {
	// 测试元数据
	Metadata struct {
		Timestamp    string `json:"timestamp"`
		Provider     string `json:"provider"`
		Model        string `json:"model"`
		BaseUrl      string `json:"base_url"`
		Concurrency  int    `json:"concurrency"`
		TotalRequest int    `json:"total_requests"`
		IsStream     bool   `json:"is_stream"`
		Prompt       string `json:"prompt"`
		TotalTime    string `json:"total_time"`
	} `json:"metadata"`

	// 时间性能指标
	TimeMetrics struct {
		AvgTotalTime string `json:"avg_total_time"`
		MinTotalTime string `json:"min_total_time"`
		MaxTotalTime string `json:"max_total_time"`
	} `json:"time_metrics"`

	// 网络性能指标
	NetworkMetrics struct {
		TargetIP            string `json:"target_ip"`
		AvgDNSTime          string `json:"avg_dns_time"`
		MinDNSTime          string `json:"min_dns_time"`
		MaxDNSTime          string `json:"max_dns_time"`
		AvgConnectTime      string `json:"avg_connect_time"`
		MinConnectTime      string `json:"min_connect_time"`
		MaxConnectTime      string `json:"max_connect_time"`
		AvgTLSHandshakeTime string `json:"avg_tls_handshake_time"`
		MinTLSHandshakeTime string `json:"min_tls_handshake_time"`
		MaxTLSHandshakeTime string `json:"max_tls_handshake_time"`
	} `json:"network_metrics"`

	// 服务性能指标
	ContentMetrics struct {
		AvgTTFT       string  `json:"avg_ttft"`
		MinTTFT       string  `json:"min_ttft"`
		MaxTTFT       string  `json:"max_ttft"`
		AvgTokenCount int     `json:"avg_token_count"`
		MinTokenCount int     `json:"min_token_count"`
		MaxTokenCount int     `json:"max_token_count"`
		AvgTPS        float64 `json:"avg_tps"`
		MinTPS        float64 `json:"min_tps"`
		MaxTPS        float64 `json:"max_tps"`
	} `json:"content_metrics"`

	// 可靠性指标
	ReliabilityMetrics struct {
		ErrorRate   float64 `json:"error_rate"`
		SuccessRate float64 `json:"success_rate"`
	} `json:"reliability_metrics"`
}

// Reporter 报告生成器
type Reporter struct {
	config TestConfig
	result TestResult
}

// NewReporter 创建新的报告生成器
func NewReporter(config TestConfig, result TestResult) *Reporter {
	return &Reporter{
		config: config,
		result: result,
	}
}

// Generate 生成报告文件
func (r *Reporter) Generate() error {
	// 生成文件名，格式：ait-report-{yy-mm-dd-hh-mm-ss}
	now := time.Now()
	filename := fmt.Sprintf("ait-report-%s.json", now.Format("06-01-02-15-04-05"))
	
	// 获取当前工作目录
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}
	
	filePath := filepath.Join(pwd, filename)

	// 构建报告数据
	reportData := r.buildReportData(now)

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(reportData, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("写入报告文件失败: %v", err)
	}

	fmt.Printf("\n📄 报告已生成: %s\n", filePath)
	return nil
}

// buildReportData 构建报告数据
func (r *Reporter) buildReportData(timestamp time.Time) ReportData {
	report := ReportData{}
	
	// 填充元数据
	report.Metadata.Timestamp = timestamp.Format("2006-01-02 15:04:05")
	report.Metadata.Provider = r.config.Provider
	report.Metadata.Model = r.config.Model
	report.Metadata.BaseUrl = r.config.BaseUrl
	report.Metadata.Concurrency = r.result.Concurrency
	report.Metadata.TotalRequest = r.result.TotalRequests
	report.Metadata.IsStream = r.result.IsStream
	report.Metadata.Prompt = r.config.Prompt
	report.Metadata.TotalTime = r.result.TotalTime.String()

	// 填充时间性能指标
	report.TimeMetrics.AvgTotalTime = r.result.TimeMetrics.AvgTotalTime.String()
	report.TimeMetrics.MinTotalTime = r.result.TimeMetrics.MinTotalTime.String()
	report.TimeMetrics.MaxTotalTime = r.result.TimeMetrics.MaxTotalTime.String()

	// 填充网络性能指标
	report.NetworkMetrics.TargetIP = r.result.NetworkMetrics.TargetIP
	report.NetworkMetrics.AvgDNSTime = r.result.NetworkMetrics.AvgDNSTime.String()
	report.NetworkMetrics.MinDNSTime = r.result.NetworkMetrics.MinDNSTime.String()
	report.NetworkMetrics.MaxDNSTime = r.result.NetworkMetrics.MaxDNSTime.String()
	report.NetworkMetrics.AvgConnectTime = r.result.NetworkMetrics.AvgConnectTime.String()
	report.NetworkMetrics.MinConnectTime = r.result.NetworkMetrics.MinConnectTime.String()
	report.NetworkMetrics.MaxConnectTime = r.result.NetworkMetrics.MaxConnectTime.String()
	report.NetworkMetrics.AvgTLSHandshakeTime = r.result.NetworkMetrics.AvgTLSHandshakeTime.String()
	report.NetworkMetrics.MinTLSHandshakeTime = r.result.NetworkMetrics.MinTLSHandshakeTime.String()
	report.NetworkMetrics.MaxTLSHandshakeTime = r.result.NetworkMetrics.MaxTLSHandshakeTime.String()

	// 填充服务性能指标
	report.ContentMetrics.AvgTTFT = r.result.ContentMetrics.AvgTTFT.String()
	report.ContentMetrics.MinTTFT = r.result.ContentMetrics.MinTTFT.String()
	report.ContentMetrics.MaxTTFT = r.result.ContentMetrics.MaxTTFT.String()
	report.ContentMetrics.AvgTokenCount = r.result.ContentMetrics.AvgTokenCount
	report.ContentMetrics.MinTokenCount = r.result.ContentMetrics.MinTokenCount
	report.ContentMetrics.MaxTokenCount = r.result.ContentMetrics.MaxTokenCount
	report.ContentMetrics.AvgTPS = r.result.ContentMetrics.AvgTPS
	report.ContentMetrics.MinTPS = r.result.ContentMetrics.MinTPS
	report.ContentMetrics.MaxTPS = r.result.ContentMetrics.MaxTPS

	// 填充可靠性指标
	report.ReliabilityMetrics.ErrorRate = r.result.ReliabilityMetrics.ErrorRate
	report.ReliabilityMetrics.SuccessRate = r.result.ReliabilityMetrics.SuccessRate

	return report
}
