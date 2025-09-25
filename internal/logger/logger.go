package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// generateLogFilePath 生成日志文件路径，格式：ait-25-09-22-17-00-27.log
func generateLogFilePath() string {
	now := time.Now()
	timestamp := now.Format("06-01-02-15-04-05") // yy-MM-dd-HH-mm-ss
	return fmt.Sprintf("ait-%s.log", timestamp)
}

// Logger 详细日志记录器
type Logger struct {
	enabled  bool
	filePath string
	file     *os.File
	logger   *log.Logger
}

// New 创建新的日志记录器
func New(enabled bool) *Logger {
	logger := &Logger{
		enabled: enabled,
	}

	if enabled {
		logger.filePath = generateLogFilePath()
		logger.init()
	}

	return logger
}

// init 初始化日志文件
func (l *Logger) init() {
	var err error
	l.file, err = os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("创建日志文件失败: %v\n", err)
		l.enabled = false
		return
	}

	l.logger = log.New(l.file, "", 0) // 不使用默认前缀，我们自定义格式
}

// Close 关闭日志文件
func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// LogLevel 日志级别
type LogLevel string

const (
	LevelInfo    LogLevel = "INFO"
	LevelRequest LogLevel = "REQUEST"
	LevelResponse LogLevel = "RESPONSE"
	LevelError   LogLevel = "ERROR"
	LevelDebug   LogLevel = "DEBUG"
)

// logEntry 日志条目结构
type logEntry struct {
	Timestamp string    `json:"timestamp"`
	Level     LogLevel  `json:"level"`
	Model     string    `json:"model,omitempty"`
	Message   string    `json:"message"`
	Details   any       `json:"details,omitempty"`
}

// writeLog 写入日志
func (l *Logger) writeLog(level LogLevel, model string, message string, details any) {
	if !l.enabled || l.logger == nil {
		return
	}

	entry := logEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05.000"),
		Level:     level,
		Model:     model,
		Message:   message,
		Details:   details,
	}

	// 将日志条目序列化为JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("日志序列化失败: %v\n", err)
		return
	}

	// 写入日志文件
	l.logger.Println(string(jsonData))
}

// Info 记录信息日志
func (l *Logger) Info(model, message string) {
	l.writeLog(LevelInfo, model, message, nil)
}

// Error 记录错误日志
func (l *Logger) Error(model, message string, err error) {
	details := map[string]interface{}{
		"error": err.Error(),
	}
	l.writeLog(LevelError, model, message, details)
}

// Debug 记录调试日志
func (l *Logger) Debug(model, message string, details any) {
	l.writeLog(LevelDebug, model, message, details)
}

// RequestData 请求数据结构
type RequestData struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	BodyEncoded string            `json:"body_encoded"` // 对特殊字符进行编码的body
}

// ResponseData 响应数据结构
type ResponseData struct {
	StatusCode     int               `json:"status_code"`
	Headers        map[string]string `json:"headers"`
	Body           string            `json:"body,omitempty"`
	BodyEncoded    string            `json:"body_encoded,omitempty"` // 对特殊字符进行编码的body
	Error          string            `json:"error,omitempty"`
	StreamChunks   []string          `json:"stream_chunks,omitempty"`   // 流式响应的数据块
	StreamEncoded  []string          `json:"stream_encoded,omitempty"`  // 编码后的流式数据块
}

// LogRequest 记录请求日志
func (l *Logger) LogRequest(model string, req RequestData) {
	// 对请求体进行编码处理
	req.BodyEncoded = encodeSpecialChars(req.Body)
	
	l.writeLog(LevelRequest, model, "HTTP Request", req)
}

// LogResponse 记录响应日志
func (l *Logger) LogResponse(model string, resp ResponseData) {
	// 对响应体进行编码处理
	if resp.Body != "" {
		resp.BodyEncoded = encodeSpecialChars(resp.Body)
	}
	
	// 对流式数据块进行编码
	if len(resp.StreamChunks) > 0 {
		resp.StreamEncoded = make([]string, len(resp.StreamChunks))
		for i, chunk := range resp.StreamChunks {
			resp.StreamEncoded[i] = encodeSpecialChars(chunk)
		}
	}
	
	l.writeLog(LevelResponse, model, "HTTP Response", resp)
}

// encodeSpecialChars 编码特殊字符，包括换行符等
func encodeSpecialChars(input string) string {
	// 使用Go的字符串转义
	// encoded := strings.ReplaceAll(input, "\n", "\\n")
	// encoded = strings.ReplaceAll(encoded, "\r", "\\r")
	// encoded = strings.ReplaceAll(encoded, "\t", "\\t")
	// encoded = strings.ReplaceAll(encoded, "\"", "\\\"")
	// encoded = strings.ReplaceAll(encoded, "\\", "\\\\")
	return input
}

// LogTestStart 记录测试开始
func (l *Logger) LogTestStart(model, prompt string, config map[string]interface{}) {
	details := map[string]interface{}{
		"prompt": prompt,
		"config": config,
	}
	l.writeLog(LevelInfo, model, "Test Started", details)
}

// LogTestEnd 记录测试结束
func (l *Logger) LogTestEnd(model string, stats map[string]interface{}) {
	l.writeLog(LevelInfo, model, "Test Completed", stats)
}

// IsEnabled 检查日志是否启用
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

// GetFilePath 获取日志文件路径
func (l *Logger) GetFilePath() string {
	return l.filePath
}
