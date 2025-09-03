package display

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReportData 报告数据结构
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
		AvgTokenCount int     `json:"avg_completion_tokens"`
		MinTokenCount int     `json:"min_completion_tokens"`
		MaxTokenCount int     `json:"max_completion_tokens"`
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

// GenerateReport 生成报告文件
func GenerateReport(result *Result, config TestConfig) error {
	// 生成文件名，格式：ait-report-{yymmdd-hhmmss}
	now := time.Now()
	filename := fmt.Sprintf("ait-report-%s.json", now.Format("20060102-150405"))
	
	// 获取当前工作目录
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}
	
	filePath := filepath.Join(pwd, filename)

	// 构建报告数据
	report := ReportData{}
	
	// 填充元数据
	report.Metadata.Timestamp = now.Format("2006-01-02 15:04:05")
	report.Metadata.Provider = config.Provider
	report.Metadata.Model = config.Model
	report.Metadata.BaseUrl = config.BaseUrl
	report.Metadata.Concurrency = result.Concurrency
	report.Metadata.TotalRequest = result.TotalRequests
	report.Metadata.IsStream = result.IsStream
	report.Metadata.TotalTime = result.TotalTime.String()

	// 填充时间性能指标
	report.TimeMetrics.AvgTotalTime = result.TimeMetrics.AvgTotalTime.String()
	report.TimeMetrics.MinTotalTime = result.TimeMetrics.MinTotalTime.String()
	report.TimeMetrics.MaxTotalTime = result.TimeMetrics.MaxTotalTime.String()

	// 填充网络性能指标
	report.NetworkMetrics.TargetIP = result.NetworkMetrics.TargetIP
	report.NetworkMetrics.AvgDNSTime = result.NetworkMetrics.AvgDNSTime.String()
	report.NetworkMetrics.MinDNSTime = result.NetworkMetrics.MinDNSTime.String()
	report.NetworkMetrics.MaxDNSTime = result.NetworkMetrics.MaxDNSTime.String()
	report.NetworkMetrics.AvgConnectTime = result.NetworkMetrics.AvgConnectTime.String()
	report.NetworkMetrics.MinConnectTime = result.NetworkMetrics.MinConnectTime.String()
	report.NetworkMetrics.MaxConnectTime = result.NetworkMetrics.MaxConnectTime.String()
	report.NetworkMetrics.AvgTLSHandshakeTime = result.NetworkMetrics.AvgTLSHandshakeTime.String()
	report.NetworkMetrics.MinTLSHandshakeTime = result.NetworkMetrics.MinTLSHandshakeTime.String()
	report.NetworkMetrics.MaxTLSHandshakeTime = result.NetworkMetrics.MaxTLSHandshakeTime.String()

	// 填充服务性能指标
	// 在非流式模式下，TTFT显示为"-"避免歧义
	if result.IsStream {
		report.ContentMetrics.AvgTTFT = result.ContentMetrics.AvgTTFT.String()
		report.ContentMetrics.MinTTFT = result.ContentMetrics.MinTTFT.String()
		report.ContentMetrics.MaxTTFT = result.ContentMetrics.MaxTTFT.String()
	} else {
		report.ContentMetrics.AvgTTFT = "-"
		report.ContentMetrics.MinTTFT = "-"
		report.ContentMetrics.MaxTTFT = "-"
	}
	report.ContentMetrics.AvgTokenCount = result.ContentMetrics.AvgTokenCount
	report.ContentMetrics.MinTokenCount = result.ContentMetrics.MinTokenCount
	report.ContentMetrics.MaxTokenCount = result.ContentMetrics.MaxTokenCount
	report.ContentMetrics.AvgTPS = result.ContentMetrics.AvgTPS
	report.ContentMetrics.MinTPS = result.ContentMetrics.MinTPS
	report.ContentMetrics.MaxTPS = result.ContentMetrics.MaxTPS

	// 填充可靠性指标
	report.ReliabilityMetrics.ErrorRate = result.ReliabilityMetrics.ErrorRate
	report.ReliabilityMetrics.SuccessRate = result.ReliabilityMetrics.SuccessRate

	// 序列化为 JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
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
