package report

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestJSONRenderer_GetFormat(t *testing.T) {
	renderer := &JSONRenderer{}
	expected := "json"
	
	if renderer.GetFormat() != expected {
		t.Errorf("GetFormat() = %v, want %v", renderer.GetFormat(), expected)
	}
}

func TestJSONRenderer_Render_EmptyData(t *testing.T) {
	renderer := &JSONRenderer{}
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
	
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Errorf("Failed to parse generated JSON: %v", err)
	}
	
	// 验证基本结构
	if result["report_type"] != "ait_benchmark_report" {
		t.Errorf("Expected report_type 'ait_benchmark_report', got %v", result["report_type"])
	}
	
	if result["total_models"] != float64(0) {
		t.Errorf("Expected total_models 0, got %v", result["total_models"])
	}
}

func TestJSONRenderer_Render_SingleModel(t *testing.T) {
	renderer := &JSONRenderer{}
	testData := []types.ReportData{createTestReportDataForJSON()}
	
	fileName, err := renderer.Render(testData)
	
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	
	if fileName == "" {
		t.Error("Render() returned empty filename")
	}
	
	// 验证文件名格式
	expectedPrefix := "ait-report-"
	expectedSuffix := ".json"
	if len(fileName) < len(expectedPrefix)+len(expectedSuffix) {
		t.Errorf("Filename too short: %s", fileName)
	}
	
	if fileName[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Filename should start with %s, got %s", expectedPrefix, fileName)
	}
	
	if fileName[len(fileName)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("Filename should end with %s, got %s", expectedSuffix, fileName)
	}
	
	// 验证文件存在
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
	
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Errorf("Failed to parse generated JSON: %v", err)
	}
	
	// 验证基本结构
	if result["report_type"] != "ait_benchmark_report" {
		t.Errorf("Expected report_type 'ait_benchmark_report', got %v", result["report_type"])
	}
	
	if result["total_models"] != float64(1) {
		t.Errorf("Expected total_models 1, got %v", result["total_models"])
	}
	
	// 验证模型数据存在
	models, ok := result["models"].([]interface{})
	if !ok {
		t.Error("Expected models to be an array")
	}
	
	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}
	
	// 验证时间戳格式
	timestamp, ok := result["timestamp"].(string)
	if !ok {
		t.Error("Expected timestamp to be a string")
	}
	
	if timestamp == "" {
		t.Error("Expected non-empty timestamp")
	}
	
	// 验证时间戳可以解析
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		t.Errorf("Invalid timestamp format: %s, error: %v", timestamp, err)
	}
}

func TestJSONRenderer_Render_MultipleModels(t *testing.T) {
	renderer := &JSONRenderer{}
	testData := []types.ReportData{
		createTestReportDataForJSON(),
		createTestReportDataForJSONWithModel("gpt-4"),
		createTestReportDataForJSONWithModel("claude-3"),
	}
	
	fileName, err := renderer.Render(testData)
	
	if err != nil {
		t.Errorf("Render() error = %v, want nil", err)
	}
	
	// 清理测试文件
	defer os.Remove(fileName)
	
	// 验证文件内容
	content, err := os.ReadFile(fileName)
	if err != nil {
		t.Errorf("Failed to read generated file: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(content, &result); err != nil {
		t.Errorf("Failed to parse generated JSON: %v", err)
	}
	
	// 验证模型数量
	if result["total_models"] != float64(3) {
		t.Errorf("Expected total_models 3, got %v", result["total_models"])
	}
	
	// 验证模型数据
	models, ok := result["models"].([]interface{})
	if !ok {
		t.Error("Expected models to be an array")
	}
	
	if len(models) != 3 {
		t.Errorf("Expected 3 models, got %d", len(models))
	}
}

func TestJSONRenderer_Render_FileCreationError(t *testing.T) {
	renderer := &JSONRenderer{}
	testData := []types.ReportData{createTestReportDataForJSON()}
	
	// 创建一个目录，使文件创建失败
	// 注意：这个测试在不同操作系统上可能有不同的行为
	// 我们需要一种更可靠的方式来测试文件创建错误
	// 但由于当前实现使用时间戳作为文件名，很难直接模拟这种情况
	// 所以我们只测试正常情况，留个测试框架在这里
	
	_, err := renderer.Render(testData)
	if err != nil {
		// 如果确实发生了错误，验证错误消息
		t.Logf("Expected file creation error occurred: %v", err)
	}
}

// 辅助函数：创建用于JSON测试的ReportData
func createTestReportDataForJSON() types.ReportData {
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
	data.ContentMetrics.AvgInputTokenCount = 50
	data.ContentMetrics.MinInputTokenCount = 40
	data.ContentMetrics.MaxInputTokenCount = 60
	data.ContentMetrics.AvgOutputTokenCount = 150
	data.ContentMetrics.MinOutputTokenCount = 100
	data.ContentMetrics.MaxOutputTokenCount = 200
	data.ContentMetrics.AvgTPS = 300.0
	data.ContentMetrics.MinTPS = 250.0
	data.ContentMetrics.MaxTPS = 350.0
	
	// 设置可靠性指标
	data.ReliabilityMetrics.SuccessRate = 95.0
	data.ReliabilityMetrics.ErrorRate = 5.0
	
	return data
}

func createTestReportDataForJSONWithModel(model string) types.ReportData {
	data := createTestReportDataForJSON()
	data.Metadata.Model = model
	return data
}
