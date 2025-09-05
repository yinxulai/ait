package display

import (
	"fmt"
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/yinxulai/ait/internal/types"
)

// Colors 定义终端颜色 - 导出供外部使用
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// Displayer 测试显示器
type Displayer struct {}

// New 创建新的测试显示器
func New() *Displayer {
	return &Displayer{}
}

// 将数据更新到终端上（刷新显示）
// 详细模式，展示所有 ReportData 的数据
func (td *Displayer) ShowSignalReport(data *types.ReportData) {
	fmt.Printf("\n=== AIT 开源测试工具结果报告 ===\n\n")
	
	// 单个综合表格
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("指标", "最小值", "平均值", "最大值", "单位")
	
	// 基础信息（这些只有单一值，只填最小值列）
	table.Append("🤖 模型", data.Metadata.Model, "", "", "-")
	table.Append("🔗 协议", data.Metadata.Protocol, "", "", "-")
	table.Append("🌐 基础URL", data.Metadata.BaseUrl, "", "", "-")
	table.Append("🌊 流式", strconv.FormatBool(data.IsStream), "", "", "-")
	table.Append("⚡ 并发数", strconv.Itoa(data.Concurrency), "", "", "个")
	table.Append("📊 总请求数", strconv.Itoa(data.TotalRequests), "", "", "个")
	table.Append("✅ 成功率", fmt.Sprintf("%.2f", data.ReliabilityMetrics.SuccessRate), "", "", "%")
	
	// 时间性能指标
	table.Append("🕐 总耗时", data.TimeMetrics.MinTotalTime.String(), data.TimeMetrics.AvgTotalTime.String(), data.TimeMetrics.MaxTotalTime.String(), "时间")
	
	// 网络性能指标
	table.Append("🔍 DNS时间", data.NetworkMetrics.MinDNSTime.String(), data.NetworkMetrics.AvgDNSTime.String(), data.NetworkMetrics.MaxDNSTime.String(), "时间")
	table.Append("🔒 TLS时间", data.NetworkMetrics.MinTLSHandshakeTime.String(), data.NetworkMetrics.AvgTLSHandshakeTime.String(), data.NetworkMetrics.MaxTLSHandshakeTime.String(), "时间")
	table.Append("🔌 TCP 连接时间", data.NetworkMetrics.MinConnectTime.String(), data.NetworkMetrics.AvgConnectTime.String(), data.NetworkMetrics.MaxConnectTime.String(), "时间")
	if data.NetworkMetrics.TargetIP != "" {
		table.Append("🎯 目标IP", data.NetworkMetrics.TargetIP, "", "", "-")
	}
	
	// 内容性能指标
	if data.IsStream {
		table.Append("⚡ TTFT", data.ContentMetrics.MinTTFT.String(), data.ContentMetrics.AvgTTFT.String(), data.ContentMetrics.MaxTTFT.String(), "时间")
	}
	table.Append("🎲 Token 数", strconv.Itoa(data.ContentMetrics.MinTokenCount), strconv.Itoa(data.ContentMetrics.AvgTokenCount), strconv.Itoa(data.ContentMetrics.MaxTokenCount), "个")
	table.Append("🚀 TPS", fmt.Sprintf("%.2f", data.ContentMetrics.MinTPS), fmt.Sprintf("%.2f", data.ContentMetrics.AvgTPS), fmt.Sprintf("%.2f", data.ContentMetrics.MaxTPS), "个/秒")
	
	table.Render()
	fmt.Println()
}

// 将数据更新到终端上（刷新显示）
// 概览模式，每行一个，展示主要数据（平均值）
func (td *Displayer) ShowMultiReport(data []*types.ReportData) {
	fmt.Printf("\n=== AIT 开源测试工具结果报告 ===\n\n")
	
	// 单个汇总表格，包含所有不同类型指标的平均值
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("🤖 模型", "🎯 目标IP", "📊 请求数", "⚡ 并发", "✅ 成功率",
		"🕐 平均总耗时", "⚡ 平均TTFT", "🚀 平均TPS", "🎲 平均Token数",
		"🔍 平均DNS时间", "🔌 平均 TCP 连接时间", "🔒 平均TLS时间")
	
	for _, report := range data {
		// TTFT 处理（流式模式才显示）
		ttftStr := "-"
		if report.IsStream {
			ttftStr = report.ContentMetrics.AvgTTFT.String()
		}
		
		table.Append(
			report.Metadata.Model,
			report.NetworkMetrics.TargetIP,
			strconv.Itoa(report.TotalRequests),
			strconv.Itoa(report.Concurrency),
			fmt.Sprintf("%.2f%%", report.ReliabilityMetrics.SuccessRate),
			report.TimeMetrics.AvgTotalTime.String(),
			ttftStr,
			fmt.Sprintf("%.2f", report.ContentMetrics.AvgTPS),
			strconv.Itoa(report.ContentMetrics.AvgTokenCount),
			report.NetworkMetrics.AvgDNSTime.String(),
			report.NetworkMetrics.AvgConnectTime.String(),
			report.NetworkMetrics.AvgTLSHandshakeTime.String(),
		)
	}
	
	table.Render()
	fmt.Println()
}
