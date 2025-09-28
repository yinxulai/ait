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
		"模型", "协议", "时间戳", "基础URL", "总请求数", "并发数", "流模式", "思考模式", "总测试时间",
		// 时间性能指标
		"平均总耗时", "最小总耗时", "最大总耗时",
		// 网络性能指标
		"目标IP", "平均DNS时间", "最小DNS时间", "最大DNS时间",
		"平均连接时间", "最小连接时间", "最大连接时间",
		"平均TLS握手时间", "最小TLS握手时间", "最大TLS握手时间",
		// 服务性能指标
		"平均TTFT", "最小TTFT", "最大TTFT",
		"平均TPOT", "最小TPOT", "最大TPOT",
		"平均输入Token数", "最小输入Token数", "最大输入Token数",
		"平均输出Token数", "最小输出Token数", "最大输出Token数",
		"平均思考Token数", "最小思考Token数", "最大思考Token数",
		"平均TPS", "最小TPS", "最大TPS",
		// 可靠性指标
		"成功率", "错误率",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write CSV headers: %v", err)
	}

	for _, modelData := range data {
		// 处理TTFT和TPOT字段，非流式模式显示为"-"
		avgTTFT := formatDurationForCSV(modelData.AvgTTFT, modelData.IsStream)
		minTTFT := formatDurationForCSV(modelData.MinTTFT, modelData.IsStream)
		maxTTFT := formatDurationForCSV(modelData.MaxTTFT, modelData.IsStream)
		avgTPOT := formatDurationForCSV(modelData.AvgTPOT, modelData.IsStream)
		minTPOT := formatDurationForCSV(modelData.MinTPOT, modelData.IsStream)
		maxTPOT := formatDurationForCSV(modelData.MaxTPOT, modelData.IsStream)

		record := []string{
			// 基础信息
			modelData.Model,
			modelData.Protocol,
			modelData.Timestamp,
			modelData.BaseUrl,
			strconv.Itoa(modelData.TotalRequests),
			strconv.Itoa(modelData.Concurrency),
			strconv.FormatBool(modelData.IsStream),
			strconv.FormatBool(modelData.IsThinking),
			modelData.TotalTime.String(),
			// 时间性能指标
			modelData.AvgTotalTime.String(),
			modelData.MinTotalTime.String(),
			modelData.MaxTotalTime.String(),
			// 网络性能指标
			modelData.TargetIP,
			modelData.AvgDNSTime.String(),
			modelData.MinDNSTime.String(),
			modelData.MaxDNSTime.String(),
			modelData.AvgConnectTime.String(),
			modelData.MinConnectTime.String(),
			modelData.MaxConnectTime.String(),
			modelData.AvgTLSHandshakeTime.String(),
			modelData.MinTLSHandshakeTime.String(),
			modelData.MaxTLSHandshakeTime.String(),
			// 服务性能指标
			avgTTFT,
			minTTFT,
			maxTTFT,
			avgTPOT,
			minTPOT,
			maxTPOT,
			strconv.Itoa(modelData.AvgInputTokenCount),
			strconv.Itoa(modelData.MinInputTokenCount),
			strconv.Itoa(modelData.MaxInputTokenCount),
			strconv.Itoa(modelData.AvgOutputTokenCount),
			strconv.Itoa(modelData.MinOutputTokenCount),
			strconv.Itoa(modelData.MaxOutputTokenCount),
			strconv.Itoa(modelData.AvgThinkingTokenCount),
			strconv.Itoa(modelData.MinThinkingTokenCount),
			strconv.Itoa(modelData.MaxThinkingTokenCount),
			strconv.FormatFloat(modelData.AvgTPS, 'f', 2, 64),
			strconv.FormatFloat(modelData.MinTPS, 'f', 2, 64),
			strconv.FormatFloat(modelData.MaxTPS, 'f', 2, 64),
			// 可靠性指标
			strconv.FormatFloat(modelData.SuccessRate, 'f', 2, 64),
			strconv.FormatFloat(modelData.ErrorRate, 'f', 2, 64),
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
