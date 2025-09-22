package report

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestCSVRenderer_GetFormat(t *testing.T) {
	renderer := &CSVRenderer{}
	expected := "csv"
	
	if renderer.GetFormat() != expected {
		t.Errorf("GetFormat() = %v, want %v", renderer.GetFormat(), expected)
	}
}

func TestCSVRenderer_Render_EmptyData(t *testing.T) {
	renderer := &CSVRenderer{}
	var emptyData []types.ReportData
	
	fileName, err := renderer.Render(emptyData)
	
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	
	if fileName == "" {
		t.Error("Render() returned empty filename")
	}
	
	// 验证文件确实被创建
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Errorf("File %s was not created", fileName)
	}
	
	// 清理测试文件
	defer os.Remove(fileName)
	
	// 验证文件内容
	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read generated file: %v", err)
	}
	
	lines := strings.Split(string(content), "\n")
	// 应该至少有头部行
	if len(lines) < 1 {
		t.Error("CSV file should have at least header row")
	}
	
	// 验证头部存在
	headers := strings.Split(lines[0], ",")
	expectedHeaderCount := 35 // 基于CSV渲染器中的实际头部数量 (原32个 + 3个TPOT字段)
	if len(headers) != expectedHeaderCount {
		t.Errorf("Expected %d headers, got %d", expectedHeaderCount, len(headers))
	}
	
	// 验证关键头部字段
	headerStr := lines[0]
	expectedHeaders := []string{"模型", "协议", "时间戳", "总请求数", "并发数", "流模式"}
	for _, header := range expectedHeaders {
		if !strings.Contains(headerStr, header) {
			t.Errorf("Header should contain '%s'", header)
		}
	}
}

func TestCSVRenderer_Render_SingleModel(t *testing.T) {
	renderer := &CSVRenderer{}
	testData := []types.ReportData{createTestReportDataForCSV()}
	
	fileName, err := renderer.Render(testData)
	
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	
	if fileName == "" {
		t.Error("Render() returned empty filename")
	}
	
	// 验证文件名格式
	expectedPrefix := "ait-report-"
	expectedSuffix := ".csv"
	if len(fileName) < len(expectedPrefix)+len(expectedSuffix) {
		t.Errorf("Filename too short: %s", fileName)
	}
	
	if fileName[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Filename should start with %s, got %s", expectedPrefix, fileName)
	}
	
	if fileName[len(fileName)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("Filename should end with %s, got %s", expectedSuffix, fileName)
	}
	
	// 清理测试文件
	defer os.Remove(fileName)
	
	// 验证CSV内容
	file, err := os.Open(fileName)
	if err != nil {
		t.Errorf("Failed to open generated file: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("Failed to read CSV: %v", err)
	}
	
	// 应该有头部 + 1行数据
	if len(records) != 2 {
		t.Errorf("Expected 2 rows (header + 1 data), got %d", len(records))
	}
	
	// 验证头部
	headers := records[0]
	expectedHeaderCount := 35 // 原32个 + 3个TPOT字段
	if len(headers) != expectedHeaderCount {
		t.Errorf("Expected %d headers, got %d", expectedHeaderCount, len(headers))
	}
	
	// 验证数据行
	dataRow := records[1]
	if len(dataRow) != expectedHeaderCount {
		t.Errorf("Expected %d data fields, got %d", expectedHeaderCount, len(dataRow))
	}
	
	// 验证一些关键字段
	if dataRow[0] != "gpt-3.5-turbo" { // 模型
		t.Errorf("Expected model 'gpt-3.5-turbo', got '%s'", dataRow[0])
	}
	
	if dataRow[1] != "openai" { // 协议
		t.Errorf("Expected protocol 'openai', got '%s'", dataRow[1])
	}
	
	if dataRow[4] != "10" { // 总请求数
		t.Errorf("Expected total requests '10', got '%s'", dataRow[4])
	}
	
	if dataRow[5] != "2" { // 并发数
		t.Errorf("Expected concurrency '2', got '%s'", dataRow[5])
	}
	
	if dataRow[6] != "true" { // 流模式
		t.Errorf("Expected stream 'true', got '%s'", dataRow[6])
	}
}

func TestCSVRenderer_Render_MultipleModels(t *testing.T) {
	renderer := &CSVRenderer{}
	testData := []types.ReportData{
		createTestReportDataForCSV(),
		createTestReportDataForCSVWithModel("gpt-4"),
		createTestReportDataForCSVWithModel("claude-3"),
	}
	
	fileName, err := renderer.Render(testData)
	
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	
	// 清理测试文件
	defer os.Remove(fileName)
	
	// 验证CSV内容
	file, err := os.Open(fileName)
	if err != nil {
		t.Errorf("Failed to open generated file: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("Failed to read CSV: %v", err)
	}
	
	// 应该有头部 + 3行数据
	if len(records) != 4 {
		t.Errorf("Expected 4 rows (header + 3 data), got %d", len(records))
	}
	
	// 验证每个模型的数据
	expectedModels := []string{"gpt-3.5-turbo", "gpt-4", "claude-3"}
	for i, expectedModel := range expectedModels {
		dataRow := records[i+1] // +1 因为第0行是头部
		if dataRow[0] != expectedModel {
			t.Errorf("Expected model '%s' at row %d, got '%s'", expectedModel, i+1, dataRow[0])
		}
	}
}

func TestCSVRenderer_Render_StreamVsNonStream(t *testing.T) {
	renderer := &CSVRenderer{}
	
	// 创建流式数据
	streamData := createTestReportDataForCSV()
	streamData.IsStream = true
	
	// 创建非流式数据
	nonStreamData := createTestReportDataForCSV()
	nonStreamData.IsStream = false
	// 非流式模式下，TTFT应该为0
	nonStreamData.ContentMetrics.AvgTTFT = 0
	nonStreamData.ContentMetrics.MinTTFT = 0
	nonStreamData.ContentMetrics.MaxTTFT = 0
	
	testData := []types.ReportData{streamData, nonStreamData}
	
	fileName, err := renderer.Render(testData)
	
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	
	// 清理测试文件
	defer os.Remove(fileName)
	
	// 验证CSV内容
	file, err := os.Open(fileName)
	if err != nil {
		t.Errorf("Failed to open generated file: %v", err)
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Errorf("Failed to read CSV: %v", err)
	}
	
	// 验证流式数据行
	streamRow := records[1]
	if streamRow[6] != "true" { // 流模式
		t.Errorf("Expected stream 'true' for stream data, got '%s'", streamRow[6])
	}
	
	// 验证非流式数据行
	nonStreamRow := records[2]
	if nonStreamRow[6] != "false" { // 流模式
		t.Errorf("Expected stream 'false' for non-stream data, got '%s'", nonStreamRow[6])
	}
	
	// 验证非流式模式下TTFT字段应该是"-"
	// TTFT字段在CSV中是第21-23列 (平均、最小、最大TTFT)
	if nonStreamRow[21] != "-" { // 平均TTFT
		t.Errorf("Expected avg TTFT '-' for non-stream data, got '%s'", nonStreamRow[21])
	}
	if nonStreamRow[22] != "-" { // 最小TTFT
		t.Errorf("Expected min TTFT '-' for non-stream data, got '%s'", nonStreamRow[22])
	}
	if nonStreamRow[23] != "-" { // 最大TTFT
		t.Errorf("Expected max TTFT '-' for non-stream data, got '%s'", nonStreamRow[23])
	}
}

func TestFormatDurationForCSV(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		isStream bool
		expected string
	}{
		{
			name:     "stream mode with duration",
			duration: 100 * time.Millisecond,
			isStream: true,
			expected: "100ms",
		},
		{
			name:     "non-stream mode with zero duration",
			duration: 0,
			isStream: false,
			expected: "-",
		},
		{
			name:     "non-stream mode with non-zero duration",
			duration: 100 * time.Millisecond,
			isStream: false,
			expected: "100ms",
		},
		{
			name:     "stream mode with zero duration",
			duration: 0,
			isStream: true,
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDurationForCSV(tt.duration, tt.isStream)
			if result != tt.expected {
				t.Errorf("formatDurationForCSV(%v, %v) = %v, want %v", 
					tt.duration, tt.isStream, result, tt.expected)
			}
		})
	}
}

// 辅助函数：创建用于CSV测试的ReportData
func createTestReportDataForCSV() types.ReportData {
	data := types.ReportData{
		TotalRequests: 10,
		Concurrency:   2,
		IsStream:      true,
		TotalTime:     5 * time.Second,
	}
	
	// 设置元数据
	data.Metadata.Model = "gpt-3.5-turbo"
	data.Metadata.Protocol = "openai"
	data.Metadata.Timestamp = time.Now().Format(time.RFC3339)
	data.Metadata.BaseUrl = "https://api.openai.com"
	
	// 设置时间指标
	data.TimeMetrics.AvgTotalTime = 500 * time.Millisecond
	data.TimeMetrics.MinTotalTime = 300 * time.Millisecond
	data.TimeMetrics.MaxTotalTime = 800 * time.Millisecond
	
	// 设置网络指标
	data.NetworkMetrics.TargetIP = "8.8.8.8"
	data.NetworkMetrics.AvgDNSTime = 10 * time.Millisecond
	data.NetworkMetrics.MinDNSTime = 5 * time.Millisecond
	data.NetworkMetrics.MaxDNSTime = 20 * time.Millisecond
	data.NetworkMetrics.AvgConnectTime = 50 * time.Millisecond
	data.NetworkMetrics.MinConnectTime = 30 * time.Millisecond
	data.NetworkMetrics.MaxConnectTime = 80 * time.Millisecond
	data.NetworkMetrics.AvgTLSHandshakeTime = 100 * time.Millisecond
	data.NetworkMetrics.MinTLSHandshakeTime = 80 * time.Millisecond
	data.NetworkMetrics.MaxTLSHandshakeTime = 150 * time.Millisecond
	
	// 设置内容指标
	data.ContentMetrics.AvgTTFT = 200 * time.Millisecond
	data.ContentMetrics.MinTTFT = 100 * time.Millisecond
	data.ContentMetrics.MaxTTFT = 300 * time.Millisecond
	data.ContentMetrics.AvgTokenCount = 150
	data.ContentMetrics.MinTokenCount = 100
	data.ContentMetrics.MaxTokenCount = 200
	data.ContentMetrics.AvgTPS = 300.0
	data.ContentMetrics.MinTPS = 250.0
	data.ContentMetrics.MaxTPS = 350.0
	
	// 设置可靠性指标
	data.ReliabilityMetrics.SuccessRate = 95.0
	data.ReliabilityMetrics.ErrorRate = 5.0
	
	return data
}

func createTestReportDataForCSVWithModel(model string) types.ReportData {
	data := createTestReportDataForCSV()
	data.Metadata.Model = model
	return data
}
