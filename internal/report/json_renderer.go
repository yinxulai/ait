package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// JSONRenderer 统一的JSON格式渲染器
type JSONRenderer struct{}

// Render 渲染JSON报告
// 统一处理单个或多个模型的数据
func (jr *JSONRenderer) Render(data []types.ReportData) (string, error) {
	timestamp := time.Now().Format("06-01-02-15-04-05")
	
	// 统一的报告结构
	content := map[string]interface{}{
		"report_type":  "ait_benchmark_report",
		"timestamp":    time.Now().Format(time.RFC3339),
		"total_models": len(data),
		"models":       data,
	}

	// 统一的文件名格式
	filename := fmt.Sprintf("ait-report-%s.json", timestamp)

	jsonData, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write JSON file: %v", err)
	}

	return filename, nil
}

// GetFormat 返回格式名称
func (jr *JSONRenderer) GetFormat() string {
	return "json"
}
