package report

import (
	"os"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// MockRenderer 用于测试的模拟渲染器
type MockRenderer struct {
	format      string
	shouldError bool
	fileName    string
}

func (m *MockRenderer) Render(data []types.ReportData) (string, error) {
	if m.shouldError {
		return "", &mockError{"mock render error"}
	}
	return m.fileName, nil
}

func (m *MockRenderer) GetFormat() string {
	return m.format
}

type mockError struct {
	text string
}

func (e *mockError) Error() string {
	return e.text
}

func TestNewReportManager(t *testing.T) {
	manager := NewReportManager()

	if manager == nil {
		t.Fatal("NewReportManager() returned nil")
	}

	if manager.renderers == nil {
		t.Fatal("NewReportManager().renderers is nil")
	}

	// 验证默认渲染器已注册
	if _, exists := manager.renderers["json"]; !exists {
		t.Error("JSON renderer not registered by default")
	}

	if _, exists := manager.renderers["csv"]; !exists {
		t.Error("CSV renderer not registered by default")
	}
}

func TestReportManager_RegisterRenderer(t *testing.T) {
	manager := NewReportManager()
	mockRenderer := &MockRenderer{format: "xml", fileName: "test.xml"}

	manager.RegisterRenderer("xml", mockRenderer)

	if _, exists := manager.renderers["xml"]; !exists {
		t.Error("Failed to register XML renderer")
	}

	// 验证注册的渲染器是正确的
	if manager.renderers["xml"] != mockRenderer {
		t.Error("Registered renderer is not the expected instance")
	}
}

func TestReportManager_GenerateReports_EmptyData(t *testing.T) {
	manager := NewReportManager()
	var emptyData []types.ReportData

	filePaths, err := manager.GenerateReports(emptyData, []string{"json"})

	if err == nil {
		t.Error("Expected error when generating reports with empty data")
	}

	if filePaths != nil {
		t.Error("Expected nil file paths when error occurs")
	}

	expectedError := "no data to generate reports"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestReportManager_GenerateReports_UnsupportedFormat(t *testing.T) {
	manager := NewReportManager()
	testData := []types.ReportData{createTestReportData()}

	filePaths, err := manager.GenerateReports(testData, []string{"unsupported"})

	if err == nil {
		t.Error("Expected error when using unsupported format")
	}

	if filePaths != nil {
		t.Error("Expected nil file paths when error occurs")
	}

	expectedError := "unsupported format: unsupported"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestReportManager_GenerateReports_RenderError(t *testing.T) {
	manager := NewReportManager()
	errorRenderer := &MockRenderer{format: "error", shouldError: true}
	manager.RegisterRenderer("error", errorRenderer)

	testData := []types.ReportData{createTestReportData()}

	filePaths, err := manager.GenerateReports(testData, []string{"error"})

	if err == nil {
		t.Error("Expected error when renderer fails")
	}

	if filePaths != nil {
		t.Error("Expected nil file paths when error occurs")
	}

	expectedError := "failed to render error: mock render error"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestReportManager_GenerateReports_Success(t *testing.T) {
	manager := NewReportManager()

	// 使用模拟渲染器以避免实际文件操作
	mockJSONRenderer := &MockRenderer{format: "json", fileName: "test.json"}
	mockCSVRenderer := &MockRenderer{format: "csv", fileName: "test.csv"}

	manager.RegisterRenderer("json", mockJSONRenderer)
	manager.RegisterRenderer("csv", mockCSVRenderer)

	testData := []types.ReportData{createTestReportData()}

	filePaths, err := manager.GenerateReports(testData, []string{"json", "csv"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(filePaths) != 2 {
		t.Errorf("Expected 2 file paths, got %d", len(filePaths))
	}

	expectedPaths := []string{"test.json", "test.csv"}
	for i, expected := range expectedPaths {
		if i >= len(filePaths) || filePaths[i] != expected {
			t.Errorf("Expected file path '%s' at index %d, got '%s'", expected, i, filePaths[i])
		}
	}
}

func TestReportManager_GenerateReports_MultipleFormats(t *testing.T) {
	manager := NewReportManager()

	mockRenderers := []*MockRenderer{
		{format: "json", fileName: "report.json"},
		{format: "csv", fileName: "report.csv"},
		{format: "xml", fileName: "report.xml"},
	}

	for _, renderer := range mockRenderers {
		manager.RegisterRenderer(renderer.format, renderer)
	}

	testData := []types.ReportData{createTestReportData()}
	formats := []string{"json", "csv", "xml"}

	filePaths, err := manager.GenerateReports(testData, formats)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(filePaths) != len(formats) {
		t.Errorf("Expected %d file paths, got %d", len(formats), len(filePaths))
	}

	for i, format := range formats {
		expectedPath := "report." + format
		if i >= len(filePaths) || filePaths[i] != expectedPath {
			t.Errorf("Expected file path '%s' for format '%s', got '%s'", expectedPath, format, filePaths[i])
		}
	}
}

func TestReportManager_GenerateReports_MultipleData(t *testing.T) {
	manager := NewReportManager()
	mockRenderer := &MockRenderer{format: "json", fileName: "multi-model.json"}
	manager.RegisterRenderer("json", mockRenderer)

	testData := []types.ReportData{
		createTestReportData(),
		createTestReportDataWithModel("gpt-4"),
	}

	filePaths, err := manager.GenerateReports(testData, []string{"json"})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(filePaths) != 1 {
		t.Errorf("Expected 1 file path, got %d", len(filePaths))
	}

	if filePaths[0] != "multi-model.json" {
		t.Errorf("Expected file path 'multi-model.json', got '%s'", filePaths[0])
	}
}

// 辅助函数：创建测试用的 ReportData
func createTestReportData() types.ReportData {
	data := types.ReportData{
		TotalRequests: 10,
		Concurrency:   2,
		IsStream:      true,
		IsThinking:    true,
		TotalTime:     5 * time.Second,
	}

	// 设置元数据 (已扁平化)
	data.Model = "gpt-3.5-turbo"
	data.Protocol = "openai"
	data.Timestamp = time.Now().Format(time.RFC3339)
	data.BaseUrl = "https://api.openai.com"

	// 设置时间指标
	data.AvgTotalTime = 500 * time.Millisecond
	data.MinTotalTime = 300 * time.Millisecond
	data.MaxTotalTime = 800 * time.Millisecond

	// 设置网络指标
	data.TargetIP = "8.8.8.8"
	data.AvgDNSTime = 10 * time.Millisecond
	data.MinDNSTime = 5 * time.Millisecond
	data.MaxDNSTime = 20 * time.Millisecond
	data.AvgConnectTime = 50 * time.Millisecond
	data.MinConnectTime = 30 * time.Millisecond
	data.MaxConnectTime = 80 * time.Millisecond
	data.AvgTLSHandshakeTime = 100 * time.Millisecond
	data.MinTLSHandshakeTime = 80 * time.Millisecond
	data.MaxTLSHandshakeTime = 150 * time.Millisecond

	// 设置内容指标
	data.AvgTTFT = 200 * time.Millisecond
	data.MinTTFT = 100 * time.Millisecond
	data.MaxTTFT = 300 * time.Millisecond
	data.AvgInputTokenCount = 50
	data.MinInputTokenCount = 40
	data.MaxInputTokenCount = 60
	data.AvgOutputTokenCount = 150
	data.MinOutputTokenCount = 100
	data.MaxOutputTokenCount = 200
	data.AvgThinkingTokenCount = 70
	data.MinThinkingTokenCount = 60
	data.MaxThinkingTokenCount = 80
	data.AvgTPS = 300.0
	data.MinTPS = 250.0
	data.MaxTPS = 350.0

	// 设置可靠性指标
	data.SuccessRate = 95.0
	data.ErrorRate = 5.0

	return data
}

func createTestReportDataWithModel(model string) types.ReportData {
	data := createTestReportData()
	data.Model = model
	return data
}

// TestCleanup 测试后清理临时文件
func TestMain(m *testing.M) {
	code := m.Run()

	// 清理可能生成的测试文件
	testFiles := []string{"test.json", "test.csv", "test.xml", "multi-model.json", "report.json", "report.csv", "report.xml"}
	for _, file := range testFiles {
		os.Remove(file)
	}

	os.Exit(code)
}
