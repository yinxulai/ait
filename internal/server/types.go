package server

import (
	"time"

	"github.com/yinxulai/ait/internal/server/types"
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
	RunID      RunID
	TaskID     string
	Status     RunStatus
	Mode       string // "standard" | "turbo" | "integrity"
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

	// 吞吐量指标（基于整体运行时长，最终稳定值）
	// RPM = 每分钟完成请求数；TPM = 每分钟输出 Token 数
	RPM float64
	TPM float64

	// 详细请求列表（按 index 排序）
	Requests []*types.RequestMetrics

	// 模式特定状态（运行时动态更新）
	// 不同模式可在此存储自定义状态，如：
	// - standard: 无额外状态
	// - turbo: {"levels": [...], "current_level": 3, "config": {...}}
	// - integrity: {"suite": {...}, "cases": [...], "current_case_id": "..."}
	ModeState map[string]any

	// 最终结果（运行结束后填充）
	// 根据 Mode 字段判断具体类型：
	// - standard: types.ReportData
	// - turbo: types.TurboResult
	// - integrity: types.IntegrityResult
	ModeResult any

	ErrorMsg string
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
	// EventIntegrityCaseStarted Integrity 模式下一个测试用例开始。
	EventIntegrityCaseStarted EventKind = "integrity_case_started"
	// EventIntegrityCaseDone Integrity 模式下一个测试用例完成。
	EventIntegrityCaseDone EventKind = "integrity_case_done"
	// EventAssertionResult Integrity 模式下断言完成。
	EventAssertionResult EventKind = "assertion_result"
	// EventRunComplete 运行正常结束。
	EventRunComplete EventKind = "run_complete"
	// EventRunFailed 运行异常中止。
	EventRunFailed EventKind = "run_failed"
)

// Event 是推送给 TUI 层的通知。Payload 类型随 Kind 不同：
//   - EventRequestDone            → *RunState（含最新请求结果的完整快照）
//   - EventProgressTick           → *RunState（定时聚合快照）
//   - EventLevelDone              → types.TurboLevelResult
//   - EventIntegrityCaseStarted   → *RunState
//   - EventIntegrityCaseDone      → *RunState
//   - EventAssertionResult        → []types.AssertionResult
//   - EventRunComplete            → *RunState（最终快照）
//   - EventRunFailed              → error
type Event struct {
	RunID   RunID
	Kind    EventKind
	Payload any
}
