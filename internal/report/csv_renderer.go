package report

import (
"encoding/csv"
"fmt"
"os"
"strconv"
"time"
)

// CSVRenderer 统一的CSV格式渲染器
type CSVRenderer struct{}

// Render 渲染CSV报告
func (cr *CSVRenderer) Render(data []StandardReportData) (string, error) {
	var filename string
	timestamp := time.Now().Format("06-01-02-15-04-05")

	if len(data) == 1 {
		filename = fmt.Sprintf("ait-report-%s-%s.csv", data[0].Metadata.Model, timestamp)
	} else {
		filename = fmt.Sprintf("ait-report-multi-model-%s.csv", timestamp)
	}

	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"模型", "协议", "时间戳", "总请求数", "并发数", "流模式", "成功率", "错误率"}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("failed to write CSV headers: %v", err)
	}

	for _, modelData := range data {
		record := []string{
			modelData.Metadata.Model,
			modelData.Metadata.Protocol,
			modelData.Metadata.Timestamp,
			strconv.Itoa(modelData.Metadata.TotalRequest),
			strconv.Itoa(modelData.Metadata.Concurrency),
			strconv.FormatBool(modelData.Metadata.IsStream),
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
