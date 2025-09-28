package upload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/client"
	"github.com/yinxulai/ait/internal/network"
	"github.com/yinxulai/ait/internal/types"
)

// ReportUploadItem API接口所需的单个上传数据结构
type ReportUploadItem struct {
	TaskID                   string  `json:"taskId"`
	Thinking                 bool    `json:"thinking"` // 是否开启思考/推理内容
	ModelKey                 *string `json:"modelKey,omitempty"`
	Reporter                 string  `json:"reporter"`
	Protocol                 string  `json:"protocol"`
	Endpoint                 string  `json:"endpoint"`
	SourceIP                 string  `json:"sourceIP"`
	ServiceIP                string  `json:"serviceIP"`
	Successful               bool    `json:"successful"`
	ProviderKey              *string `json:"providerKey,omitempty"`
	ProviderModelKey         string  `json:"providerModelKey"`
	InputTokenCount          int     `json:"inputTokenCount"`
	OutputTokenCount         int     `json:"outputTokenCount"`
	OutputThinkingTokenCount int     `json:"outputThinkingTokenCount"` // 思考/推理内容输出长度（字符数）
	TotalTime                int64   `json:"totalTime"`                // 毫秒
	DNSLookupTime            int64   `json:"dnsLookupTime"`            // 毫秒
	TCPConnectTime           int64   `json:"tcpConnectTime"`           // 毫秒
	TLSHandshakeTime         int64   `json:"tlsHandshakeTime"`         // 毫秒
	PerOutputTokenTime       float64 `json:"perOutputTokenTime"`       // 毫秒
	FirstOutputTokenTime     int64   `json:"firstOutputTokenTime"`     // 毫秒
	ErrorMessage             string  `json:"errorMessage"`
}

// Uploader 上传器结构体
type Uploader struct {
	baseURL   string
	authToken string
	userAgent string
	client    *http.Client
}

var (
	// 这些变量会在构建时被 ldflags 替换
	UploadBaseURL   = "null"
	UploadAuthToken = "null"
	UploadUserAgent = "yinxulai/ait"
)

// New 创建新的上传器实例
func New() *Uploader {
	return &Uploader{
		baseURL:   UploadBaseURL,
		authToken: UploadAuthToken,
		userAgent: UploadUserAgent,
		client: &http.Client{
			Timeout: time.Second * 3,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

// isValidURL 检查给定的字符串是否是一个有效的URL
func (u *Uploader) isValidURL(urlStr string) bool {
	if urlStr == "" || urlStr == "null" {
		return false
	}

	// 解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// 检查协议必须是 http 或 https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// 检查必须有主机名
	if parsedURL.Host == "" {
		return false
	}

	return true
}

// convertResponseMetricsToUploadItem 将单个ResponseMetrics转换为上传格式
func (u *Uploader) convertResponseMetricsToUploadItem(taskID string, metrics *client.ResponseMetrics, input types.Input) ReportUploadItem {
	var errorMessage string
	successful := true

	// 检查是否有错误
	if metrics.ErrorMessage != "" {
		errorMessage = metrics.ErrorMessage
		successful = false
	}

	// 计算每个输出token的时间（毫秒）
	var perOutputTokenTime float64
	if metrics.CompletionTokens > 1 {
		// 总时间减去首token时间，然后除以剩余token数
		remainingTokens := metrics.CompletionTokens - 1
		remainingTime := metrics.TotalTime - metrics.TimeToFirstToken
		perOutputTokenTime = float64(remainingTime.Nanoseconds()) / 1e6 / float64(remainingTokens)
	}

	// 获取出口 IP 地址
	sourceIP := "--"
	if publicIP, err := network.GetPublicIPCached(); err == nil {
		sourceIP = publicIP
	}

	return ReportUploadItem{
		TaskID:                   taskID,
		Thinking:                 input.Thinking,
		ModelKey:                 nil, // 未知模型
		Reporter:                 u.userAgent,
		Protocol:                 strings.ToUpper(input.Protocol),
		Endpoint:                 input.BaseUrl,
		SourceIP:                 sourceIP,
		ServiceIP:                metrics.TargetIP,
		Successful:               successful,
		ProviderKey:              nil,                  // 未知提供商
		ProviderModelKey:         input.Model,          // 使用输入的模型名称
		InputTokenCount:          metrics.PromptTokens, // ResponseMetrics 中没有输入token数
		OutputTokenCount:         metrics.CompletionTokens,
		OutputThinkingTokenCount: metrics.ThinkingTokens,
		TotalTime:                metrics.TotalTime.Nanoseconds() / 1e6,        // 转换为毫秒
		DNSLookupTime:            metrics.DNSTime.Nanoseconds() / 1e6,          // 转换为毫秒
		TCPConnectTime:           metrics.ConnectTime.Nanoseconds() / 1e6,      // 转换为毫秒
		TLSHandshakeTime:         metrics.TLSHandshakeTime.Nanoseconds() / 1e6, // 转换为毫秒
		PerOutputTokenTime:       perOutputTokenTime,
		FirstOutputTokenTime:     metrics.TimeToFirstToken.Nanoseconds() / 1e6, // 转换为毫秒
		ErrorMessage:             errorMessage,
	}
}

// UploadReport 上传单个测试报告
func (u *Uploader) UploadReport(taskId string, metrics *client.ResponseMetrics, input types.Input) error {
	if !u.isValidURL(u.baseURL) || u.authToken == "null" {
		return nil
	}

	// 转换数据格式
	uploadItem := u.convertResponseMetricsToUploadItem(taskId, metrics, input)
	uploadItems := []ReportUploadItem{uploadItem} // API需要数组格式

	// 序列化为JSON
	jsonData, err := json.Marshal(uploadItems)
	if err != nil {
		return err
	}

	// 构造完整URL
	baseURL := u.baseURL
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	fullURL := baseURL + "/model/perf/report/upload"

	// 创建请求
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("User-Agent", u.userAgent)
	req.Header.Set("Content-Type", "application/json")

	// 添加鉴权header（如果提供了鉴权token）
	if u.authToken != "" && u.authToken != "null" {
		req.Header.Set("Authorization", "Bearer "+u.authToken)
	}

	// 发送请求（静默执行，不输出错误到终端）
	resp, err := u.client.Do(req)
	if err != nil {
		// 静默失败，只返回错误但不打印
		return err
	}

	defer resp.Body.Close()

	// 读取响应（即使不使用也要读取以释放连接）
	_, _ = io.ReadAll(resp.Body)

	// 检查状态码，但不打印错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("上传失败，状态码: %d", resp.StatusCode)
	}

	return nil
}
