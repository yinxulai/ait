package report

import (
	"fmt"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// ReportRenderer 报告渲染器接口
type ReportRenderer interface {
	Render(data []types.ReportData) (string, error)
	GetFormat() string
}

// ReportManager 统一的报告管理器
type ReportManager struct {
	renderers map[string]ReportRenderer
}

// NewReportManager 创建新的报告管理器
func NewReportManager() *ReportManager {
	manager := &ReportManager{
		renderers: make(map[string]ReportRenderer),
	}

	// 注册默认的渲染器
	manager.RegisterRenderer("json", &JSONRenderer{})
	manager.RegisterRenderer("csv", &CSVRenderer{})

	return manager
}

// RegisterRenderer 注册渲染器
func (rm *ReportManager) RegisterRenderer(format string, renderer ReportRenderer) {
	rm.renderers[format] = renderer
}

// GenerateReports 生成报告文件
func (rm *ReportManager) GenerateReports(data []types.ReportData, formats []string) ([]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data to generate reports")
	}

	var filePaths []string

	for _, format := range formats {
		renderer, exists := rm.renderers[format]
		if !exists {
			return nil, fmt.Errorf("unsupported format: %s", format)
		}

		filePath, err := renderer.Render(data)
		if err != nil {
			return nil, fmt.Errorf("failed to render %s: %v", format, err)
		}

		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}

// GenerateReport 便捷函数，用于生成单个报告
func GenerateReport(input *types.Input, reportData *types.ReportData, formats []string) ([]string, error) {
	manager := NewReportManager()

	// 填充元数据
	reportData.Metadata.Timestamp = time.Now().Format(time.RFC3339)
	reportData.Metadata.Protocol = input.Protocol
	reportData.Metadata.Model = input.Model
	reportData.Metadata.BaseUrl = input.BaseUrl

	return manager.GenerateReports([]types.ReportData{*reportData}, formats)
}

// Reporter 向后兼容的报告生成器
type Reporter struct {
	config *types.Input
	result *types.ReportData
}

// NewReporter 创建新的报告生成器
func NewReporter(config types.Input, result types.ReportData) *Reporter {
	return &Reporter{
		config: &config,
		result: &result,
	}
}

// Generate 生成报告文件
func (r *Reporter) Generate() error {
	_, err := GenerateReport(r.config, r.result, []string{"json", "csv"})
	return err
}
