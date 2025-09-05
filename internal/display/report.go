package display

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// GenerateReport 生成报告文件
// 这是一个简化的报告生成函数，用于向后兼容
func GenerateReport(config types.Input, result *types.ReportData) error {
	// 生成文件名，格式：ait-report-{model}-{yymmdd-hhmmss}
	now := time.Now()
	filename := fmt.Sprintf("ait-report-%s-%s.json", config.Model, now.Format("06-01-02-15-04-05"))
	
	// 获取当前工作目录
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}
	
	filePath := filepath.Join(pwd, filename)

	// 填充元数据
	result.Metadata.Timestamp = now.Format("2006-01-02 15:04:05")
	result.Metadata.Protocol = config.Protocol
	result.Metadata.Model = config.Model
	result.Metadata.BaseUrl = config.BaseUrl

	// 序列化为JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化报告数据失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(filePath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("写入报告文件失败: %v", err)
	}

	fmt.Printf("\n📄 报告已生成: %s\n", filePath)
	return nil
}
