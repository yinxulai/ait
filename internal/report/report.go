package report

import (
	"fmt"
	"time"
)

// TestConfig 测试配置信息
type TestConfig struct {
	Protocol    string
	BaseUrl     string
	ApiKey      string
	Model       string
	Concurrency int
	Count       int
	Stream      bool
	Prompt      string
}

// TestResult 统一的测试结果数据结构
// 这个结构将被display和report模块共同使用，避免重复定义
type TestResult struct {
	// 基础测试信息
	TotalRequests int
	Concurrency   int
	IsStream      bool
	TotalTime     time.Duration

	// 时间性能指标
	TimeMetrics struct {
		AvgTotalTime time.Duration
		MinTotalTime time.Duration
		MaxTotalTime time.Duration
	}

	// 网络性能指标
	NetworkMetrics struct {
		AvgDNSTime          time.Duration
		MinDNSTime          time.Duration
		MaxDNSTime          time.Duration
		AvgConnectTime      time.Duration
		MinConnectTime      time.Duration
		MaxConnectTime      time.Duration
		AvgTLSHandshakeTime time.Duration
		MinTLSHandshakeTime time.Duration
		MaxTLSHandshakeTime time.Duration
		TargetIP            string
	}

	// 服务性能指标
	ContentMetrics struct {
		AvgTTFT       time.Duration
		MinTTFT       time.Duration
		MaxTTFT       time.Duration
		AvgTokenCount int
		MinTokenCount int
		MaxTokenCount int
		AvgTPS        float64
		MinTPS        float64
		MaxTPS        float64
	}

	// 可靠性指标
	ReliabilityMetrics struct {
		ErrorRate   float64
		SuccessRate float64
	}
}

// ReportData 报告数据结构，包含配置和结果
type ReportData struct {
	Config TestConfig
	Result TestResult
}

// StandardReportData 标准报告数据结构（基于 JSON 格式）
type StandardReportData struct {
	// 测试元数据
	Metadata struct {
		Timestamp    string `json:"timestamp"`
		Protocol     string `json:"protocol"`
		Model        string `json:"model"`
		BaseUrl      string `json:"base_url"`
		Concurrency  int    `json:"concurrency"`
		TotalRequest int    `json:"total_requests"`
		IsStream     bool   `json:"is_stream"`
		Prompt       string `json:"prompt"`
		TotalTime    string `json:"total_time"`
	} `json:"metadata"`

	// 时间性能指标
	TimeMetrics struct {
		AvgTotalTime string `json:"avg_total_time"`
		MinTotalTime string `json:"min_total_time"`
		MaxTotalTime string `json:"max_total_time"`
	} `json:"time_metrics"`

	// 网络性能指标
	NetworkMetrics struct {
		TargetIP            string `json:"target_ip"`
		AvgDNSTime          string `json:"avg_dns_time"`
		MinDNSTime          string `json:"min_dns_time"`
		MaxDNSTime          string `json:"max_dns_time"`
		AvgConnectTime      string `json:"avg_connect_time"`
		MinConnectTime      string `json:"min_connect_time"`
		MaxConnectTime      string `json:"max_connect_time"`
		AvgTLSHandshakeTime string `json:"avg_tls_handshake_time"`
		MinTLSHandshakeTime string `json:"min_tls_handshake_time"`
		MaxTLSHandshakeTime string `json:"max_tls_handshake_time"`
	} `json:"network_metrics"`

	// 服务性能指标
	ContentMetrics struct {
		AvgTTFT       string  `json:"avg_ttft"`
		MinTTFT       string  `json:"min_ttft"`
		MaxTTFT       string  `json:"max_ttft"`
		AvgTokenCount int     `json:"avg_token_count"`
		MinTokenCount int     `json:"min_token_count"`
		MaxTokenCount int     `json:"max_token_count"`
		AvgTPS        float64 `json:"avg_tps"`
		MinTPS        float64 `json:"min_tps"`
		MaxTPS        float64 `json:"max_tps"`
	} `json:"content_metrics"`

	// 可靠性指标
	ReliabilityMetrics struct {
		ErrorRate   float64 `json:"error_rate"`
		SuccessRate float64 `json:"success_rate"`
	} `json:"reliability_metrics"`
}

// ReportRenderer 报告渲染器接口
type ReportRenderer interface {
	Render(data []StandardReportData) (string, error)
	GetFormat() string
}

// ReportManager 统一的报告管理器
// 支持处理任意数量的模型数据，不再区分单模型和多模型
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
// data 参数可以包含一个或多个模型的数据
func (rm *ReportManager) GenerateReports(data []StandardReportData, formats []string) ([]string, error) {
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

// convertToStandardData 将 ReportData 转换为 StandardReportData
func convertToStandardData(data *ReportData) StandardReportData {
	var standardData StandardReportData
	
	// 设置元数据
	standardData.Metadata.Timestamp = time.Now().Format(time.RFC3339)
	standardData.Metadata.Protocol = data.Config.Protocol
	standardData.Metadata.Model = data.Config.Model
	standardData.Metadata.BaseUrl = data.Config.BaseUrl
	standardData.Metadata.Concurrency = data.Config.Concurrency
	standardData.Metadata.TotalRequest = data.Result.TotalRequests
	standardData.Metadata.IsStream = data.Config.Stream
	standardData.Metadata.Prompt = data.Config.Prompt
	standardData.Metadata.TotalTime = data.Result.TotalTime.String()

	// 时间性能指标
	standardData.TimeMetrics.AvgTotalTime = data.Result.TimeMetrics.AvgTotalTime.String()
	standardData.TimeMetrics.MinTotalTime = data.Result.TimeMetrics.MinTotalTime.String()
	standardData.TimeMetrics.MaxTotalTime = data.Result.TimeMetrics.MaxTotalTime.String()

	// 网络性能指标
	standardData.NetworkMetrics.TargetIP = data.Result.NetworkMetrics.TargetIP
	standardData.NetworkMetrics.AvgDNSTime = data.Result.NetworkMetrics.AvgDNSTime.String()
	standardData.NetworkMetrics.MinDNSTime = data.Result.NetworkMetrics.MinDNSTime.String()
	standardData.NetworkMetrics.MaxDNSTime = data.Result.NetworkMetrics.MaxDNSTime.String()
	standardData.NetworkMetrics.AvgConnectTime = data.Result.NetworkMetrics.AvgConnectTime.String()
	standardData.NetworkMetrics.MinConnectTime = data.Result.NetworkMetrics.MinConnectTime.String()
	standardData.NetworkMetrics.MaxConnectTime = data.Result.NetworkMetrics.MaxConnectTime.String()
	standardData.NetworkMetrics.AvgTLSHandshakeTime = data.Result.NetworkMetrics.AvgTLSHandshakeTime.String()
	standardData.NetworkMetrics.MinTLSHandshakeTime = data.Result.NetworkMetrics.MinTLSHandshakeTime.String()
	standardData.NetworkMetrics.MaxTLSHandshakeTime = data.Result.NetworkMetrics.MaxTLSHandshakeTime.String()

	// 服务性能指标
	standardData.ContentMetrics.AvgTTFT = data.Result.ContentMetrics.AvgTTFT.String()
	standardData.ContentMetrics.MinTTFT = data.Result.ContentMetrics.MinTTFT.String()
	standardData.ContentMetrics.MaxTTFT = data.Result.ContentMetrics.MaxTTFT.String()
	standardData.ContentMetrics.AvgTokenCount = data.Result.ContentMetrics.AvgTokenCount
	standardData.ContentMetrics.MinTokenCount = data.Result.ContentMetrics.MinTokenCount
	standardData.ContentMetrics.MaxTokenCount = data.Result.ContentMetrics.MaxTokenCount
	standardData.ContentMetrics.AvgTPS = data.Result.ContentMetrics.AvgTPS
	standardData.ContentMetrics.MinTPS = data.Result.ContentMetrics.MinTPS
	standardData.ContentMetrics.MaxTPS = data.Result.ContentMetrics.MaxTPS

	// 可靠性指标
	standardData.ReliabilityMetrics.ErrorRate = data.Result.ReliabilityMetrics.ErrorRate
	standardData.ReliabilityMetrics.SuccessRate = data.Result.ReliabilityMetrics.SuccessRate

	return standardData
}

// GenerateReport 生成报告的便捷函数
// 自动处理单个或多个模型的数据
func GenerateReport(reportDataList []*ReportData, formats []string) ([]string, error) {
	if len(reportDataList) == 0 {
		return nil, fmt.Errorf("no report data provided")
	}

	// 转换为标准数据格式
	var standardDataList []StandardReportData
	for _, data := range reportDataList {
		standardData := convertToStandardData(data)
		standardDataList = append(standardDataList, standardData)
	}

	// 创建报告管理器并生成报告
	manager := NewReportManager()
	filePaths, err := manager.GenerateReports(standardDataList, formats)
	if err != nil {
		return nil, err
	}

	// 打印生成的报告信息
	if len(filePaths) > 0 {
		if len(reportDataList) == 1 {
			fmt.Printf("\n📄 报告已生成 (模型: %s):\n", reportDataList[0].Config.Model)
		} else {
			fmt.Printf("\n📊 多模型比较报告已生成 (%d个模型):\n", len(reportDataList))
		}
		
		for _, path := range filePaths {
			fmt.Printf("  %s\n", path)
		}
	}

	return filePaths, nil
}

// Reporter 向后兼容的报告生成器
type Reporter struct {
	config TestConfig
	result TestResult
}

// NewReporter 创建新的报告生成器（向后兼容）
func NewReporter(config TestConfig, result TestResult) *Reporter {
	return &Reporter{
		config: config,
		result: result,
	}
}

// Generate 生成报告文件（向后兼容）
func (r *Reporter) Generate() error {
	data := &ReportData{
		Config: r.config,
		Result: r.result,
	}
	
	_, err := GenerateReport([]*ReportData{data}, []string{"json", "csv"})
	return err
}
