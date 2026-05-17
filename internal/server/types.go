package server

import (
	"time"

	"github.com/yinxulai/ait/internal/types"
)

// RunID 唯一标识一次运行（全局唯一，格式 run_<unix_nano>）。
type RunID string

// CancelFunc 取消订阅用的函数，调用后关闭对应的事件通道。
type CancelFunc func()

// ReportFormat 报告文件格式。
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatCSV  ReportFormat = "csv"
)

// TaskConfig 新建/更新任务时提交的可变配置。
// ID、时间戳等元数据由 Server 自动管理。
type TaskConfig struct {
	Name  string
	Input types.Input
}

// RunStatus 运行的生命周期状态。
type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusStopped   RunStatus = "stopped"
)

// RunState 一次运行的完整快照，由 GetRunState 返回。
// 字段为只读快照，不持有锁，TUI 层可安全读取。
type RunState struct {
	RunID   RunID
	TaskID  string
	Status  RunStatus
	Mode    string // "standard" | "turbo"
	StartedAt  time.Time
	FinishedAt *time.Time

	// 进度计数
	TotalReqs   int
	DoneReqs    int
	SuccessReqs int
	FailedReqs  int

	// 聚合指标（实时更新）
	AvgTPS       float64
	AvgTTFT      time.Duration
	SuccessRate  float64
	CacheHitRate float64

	// 详细请求列表（按 index 排序）
	Requests []*RequestMetrics

	// Turbo 专用
	Levels       []types.TurboLevelResult
	CurrentLevel int

	// 最终结果（运行结束后填充）
	StandardResult *types.ReportData
	TurboResult    *types.TurboResult

	ErrorMsg string
}

// RequestMetrics 单次请求的详细指标，供请求列表页展示。
type RequestMetrics struct {
	Index            int
	Success          bool
	TotalTime        time.Duration
	TTFT             time.Duration
	TPS              float64
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
	CacheHitRate     float64
	DNSTime          time.Duration
	ConnectTime      time.Duration
	TLSTime          time.Duration
	TargetIP         string
	ErrorMessage     string
	// 原始请求/响应数据（供请求详情页展示和复制）
	RequestBody  string
	ResponseBody string
}

// EventKind 事件类型枚举。
type EventKind string

const (
	// EventRequestDone 单个请求完成（含成功/失败）。
	EventRequestDone EventKind = "request_done"
	// EventProgressTick 定时聚合快照（约 500ms 发一次）。
	EventProgressTick EventKind = "progress_tick"
	// EventLevelDone Turbo 模式下一个并发级别探测完成。
	EventLevelDone EventKind = "level_done"
	// EventRunComplete 运行正常结束。
	EventRunComplete EventKind = "run_complete"
	// EventRunFailed 运行异常中止。
	EventRunFailed EventKind = "run_failed"
)

// Event 是推送给 TUI 层的通知。Payload 类型随 Kind 不同：
//   - EventRequestDone  → *RunState（含最新请求结果的完整快照）
//   - EventProgressTick → *RunState（定时聚合快照）
//   - EventLevelDone    → types.TurboLevelResult
//   - EventRunComplete  → *RunState（最终快照）
//   - EventRunFailed    → error
type Event struct {
	RunID   RunID
	Kind    EventKind
	Payload any
}
