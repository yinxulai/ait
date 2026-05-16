package tui

import (
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// TasksLoadedMsg 任务列表加载完成（初始化或刷新后）。
type TasksLoadedMsg struct {
	Tasks []types.TaskDefinition
}

// TaskSavedMsg 新建或更新任务完成。
type TaskSavedMsg struct {
	Task      types.TaskDefinition
	AutoStart bool // 是否自动启动运行（由调用方决定）
}

// TaskDeletedMsg 任务删除完成。
type TaskDeletedMsg struct {
	TaskID string
}

// HistoryLoadedMsg 任务历史记录加载完成。
type HistoryLoadedMsg struct {
	TaskID  string
	History []types.TaskRunSummary
}

// RunStartedMsg 运行成功启动，携带 RunID 供后续订阅事件使用。
type RunStartedMsg struct {
	RunID  server.RunID
	TaskID string
}

// ServerEventMsg 封装从 server.Subscribe 获取的事件，由 waitEventCmd 产生。
type ServerEventMsg struct {
	Event server.Event
}

// RunStateMsg server.GetRunState 的轮询结果（用于后台运行恢复仪表盘）。
type RunStateMsg struct {
	State *server.RunState
}

// ReportGeneratedMsg 报告文件生成完成。
type ReportGeneratedMsg struct {
	RunID server.RunID
	Path  string
}

// ErrorMsg 通用异步错误，显示在状态栏。
type ErrorMsg struct {
	Err error
}
