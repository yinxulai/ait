package report

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONRenderer 统一的JSON格式渲染器
type JSONRenderer struct{}

// Render 渲染JSON报告
// 支持单个或多个模型的数据
func (jr *JSONRenderer) Render(data []StandardReportData) (string, error) {
	var content interface{}
	var filename string
	timestamp := time.Now().Format("06-01-02-15-04-05")

	if len(data) == 1 {
		// 单模型报告
		content = data[0]
		filename = fmt.Sprintf("ait-report-%s-%s.json", data[0].Metadata.Model, timestamp)
	} else {
		// 多模型报告
		content = map[string]interface{}{
			"report_type": "multi_model_comparison",
			"timestamp":   time.Now().Format(time.RFC3339),
			"total_models": len(data),
			"models":      data,
		}
		filename = fmt.Sprintf("ait-report-multi-model-%s.json", timestamp)
	}

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
