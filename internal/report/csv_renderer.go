package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// CSVRenderer 统一的CSV格式渲染器
type CSVRenderer struct{}

// Render 渲染CSV报告
func (cr *CSVRenderer) Render(data []types.ReportData) (string, error) {
	timestamp := time.Now().Format("06-01-02-15-04-05")
	filename := fmt.Sprintf("ait-report-%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 完整的CSV头部，包含所有ReportData指标
	headers := []string{
		// 基础信息
		"模型", "协议", "时间戳", "基础URL", "总请求数", "并发数", "流模式", "总测试时间",
		// 时间性能指标
		"平均总耗时", "最小总耗时", "最大总耗时",
		// 网络性能指标
		"目标IP", "平均DNS时间", "最小DNS时间", "最大DNS时间",
		"平均连接时间", "最小连接时间", "最大连接时间",
		"平均TLS握手时间", "最小TLS握手时间", "最大TLS握手时间",
		// 服务性能指标
		"平均TTFT", "最小TTFT", "最大TTFT",
		"平均Token数", "最小Token数", "最大Token数",
		"平均TPS", "最小TPS", "最大TPS",
		// 可靠性指标
		"成功率", "错误率",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write CSV headers: %v", err)
	}

	for _, modelData := range data {
		// 处理TTFT字段，非流式模式显示为"-"
		avgTTFT := formatDurationForCSV(modelData.ContentMetrics.AvgTTFT, modelData.IsStream)
		minTTFT := formatDurationForCSV(modelData.ContentMetrics.MinTTFT, modelData.IsStream)
		maxTTFT := formatDurationForCSV(modelData.ContentMetrics.MaxTTFT, modelData.IsStream)

		record := []string{
			// 基础信息
			modelData.Metadata.Model,
			modelData.Metadata.Protocol,
			modelData.Metadata.Timestamp,
			modelData.Metadata.BaseUrl,
			strconv.Itoa(modelData.TotalRequests),
			strconv.Itoa(modelData.Concurrency),
			strconv.FormatBool(modelData.IsStream),
			modelData.TotalTime.String(),
			// 时间性能指标
			modelData.TimeMetrics.AvgTotalTime.String(),
			modelData.TimeMetrics.MinTotalTime.String(),
			modelData.TimeMetrics.MaxTotalTime.String(),
			// 网络性能指标
			modelData.NetworkMetrics.TargetIP,
			modelData.NetworkMetrics.AvgDNSTime.String(),
			modelData.NetworkMetrics.MinDNSTime.String(),
			modelData.NetworkMetrics.MaxDNSTime.String(),
			modelData.NetworkMetrics.AvgConnectTime.String(),
			modelData.NetworkMetrics.MinConnectTime.String(),
			modelData.NetworkMetrics.MaxConnectTime.String(),
			modelData.NetworkMetrics.AvgTLSHandshakeTime.String(),
			modelData.NetworkMetrics.MinTLSHandshakeTime.String(),
			modelData.NetworkMetrics.MaxTLSHandshakeTime.String(),
			// 服务性能指标
			avgTTFT,
			minTTFT,
			maxTTFT,
			strconv.Itoa(modelData.ContentMetrics.AvgTokenCount),
			strconv.Itoa(modelData.ContentMetrics.MinTokenCount),
			strconv.Itoa(modelData.ContentMetrics.MaxTokenCount),
			strconv.FormatFloat(modelData.ContentMetrics.AvgTPS, 'f', 2, 64),
			strconv.FormatFloat(modelData.ContentMetrics.MinTPS, 'f', 2, 64),
			strconv.FormatFloat(modelData.ContentMetrics.MaxTPS, 'f', 2, 64),
			// 可靠性指标
			strconv.FormatFloat(modelData.ReliabilityMetrics.SuccessRate, 'f', 2, 64),
			strconv.FormatFloat(modelData.ReliabilityMetrics.ErrorRate, 'f', 2, 64),
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("failed to write CSV record: %v", err)
		}
	}
	return filename, nil
}

func (cr *CSVRenderer) GetFormat() string {
	return "csv"
}

// formatDurationForCSV 格式化时间字段，非流式模式下的TTFT返回"-"
func formatDurationForCSV(duration time.Duration, isStream bool) string {
	if !isStream && (duration == 0) {
		return "-"
	}
	return duration.String()
}
