package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// Client 持有 server.Server，为 TUI 层提供 tea.Cmd 包装的异步调用。
// TUI Model 通过 Client 与 Server 交互，不直接 import runner/task/turbo 等下层包。
type Client struct {
	srv server.Server
}

// NewClient 创建 Client 实例。
func NewClient(srv server.Server) *Client {
	return &Client{srv: srv}
}

// ─── 任务 CRUD ────────────────────────────────────────────────────────────────

// LoadTasksCmd 异步加载任务列表。
func (c *Client) LoadTasksCmd() tea.Cmd {
	return func() tea.Msg {
		tasks, err := c.srv.ListTasks()
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("加载任务失败: %w", err)}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

// CreateTaskCmd 异步新建任务，autoStart 表示成功后是否自动触发运行。
func (c *Client) CreateTaskCmd(cfg server.TaskConfig, autoStart bool) tea.Cmd {
	return func() tea.Msg {
		task, err := c.srv.CreateTask(cfg)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("创建任务失败: %w", err)}
		}
		return TaskSavedMsg{Task: task, AutoStart: autoStart}
	}
}

// UpdateTaskCmd 异步更新任务。
func (c *Client) UpdateTaskCmd(id string, cfg server.TaskConfig) tea.Cmd {
	return func() tea.Msg {
		task, err := c.srv.UpdateTask(id, cfg)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("更新任务失败: %w", err)}
		}
		return TaskSavedMsg{Task: task, AutoStart: false}
	}
}

// DeleteTaskCmd 异步删除任务。
func (c *Client) DeleteTaskCmd(id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.srv.DeleteTask(id); err != nil {
			return ErrorMsg{Err: fmt.Errorf("删除任务失败: %w", err)}
		}
		return TaskDeletedMsg{TaskID: id}
	}
}

// CopyTaskCmd 异步复制任务。
func (c *Client) CopyTaskCmd(id string) tea.Cmd {
	return func() tea.Msg {
		task, err := c.srv.CopyTask(id)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("复制任务失败: %w", err)}
		}
		return TaskSavedMsg{Task: task, AutoStart: false}
	}
}

// ─── 运行管理 ─────────────────────────────────────────────────────────────────

// StartRunCmd 异步启动运行。
func (c *Client) StartRunCmd(taskID string) tea.Cmd {
	return func() tea.Msg {
		runID, err := c.srv.StartRun(taskID)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("启动运行失败: %w", err)}
		}
		return RunStartedMsg{RunID: runID, TaskID: taskID}
	}
}

// StopRunCmd 异步停止运行（fire-and-forget，忽略错误）。
func (c *Client) StopRunCmd(runID server.RunID) tea.Cmd {
	return func() tea.Msg {
		_ = c.srv.StopRun(runID)
		return nil
	}
}

// SubscribeCmd 订阅 runID 的事件流，返回用于首次等待的 Cmd 和 CancelFunc。
// 调用方应将 ch 存储在 dashboardState 中，每次收到 ServerEventMsg 后
// 再次调用 WaitEventCmd(ch) 继续监听。
func (c *Client) SubscribeCmd(runID server.RunID) (<-chan server.Event, server.CancelFunc, tea.Cmd) {
	ch, cancel := c.srv.Subscribe(runID)
	return ch, cancel, WaitEventCmd(ch)
}

// WaitEventCmd 等待事件通道的下一条事件。
// 通道关闭时返回 nil（Update 中检测 nil 即可停止循环）。
func WaitEventCmd(ch <-chan server.Event) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return ServerEventMsg{Event: event}
	}
}

// ─── 历史 & 报告 ──────────────────────────────────────────────────────────────

// LoadHistoryCmd 异步加载指定任务的运行历史，limit<=0 表示不限条数。
func (c *Client) LoadHistoryCmd(taskID string, limit int) tea.Cmd {
	return func() tea.Msg {
		history, err := c.srv.GetHistory(taskID, limit)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("加载历史失败: %w", err)}
		}
		return HistoryLoadedMsg{TaskID: taskID, History: history}
	}
}

// GetRunStateCmd 异步获取运行状态快照（后台模式重入仪表盘时使用）。
func (c *Client) GetRunStateCmd(runID server.RunID) tea.Cmd {
	return func() tea.Msg {
		state, ok := c.srv.GetRunState(runID)
		if !ok {
			return nil
		}
		return RunStateMsg{State: state}
	}
}

// GetRunStateForHistoryCmd 从历史记录导航时异步加载运行状态快照。
// 若磁盘文件不存在，则用 summary 摘要数据构造最小化 RunState 作为回退。
func (c *Client) GetRunStateForHistoryCmd(runID server.RunID, summary *types.TaskRunSummary) tea.Cmd {
	return func() tea.Msg {
		state, ok := c.srv.GetRunState(runID)
		if !ok {
			if summary != nil {
				state = summaryToRunState(summary)
			} else {
				return ErrorMsg{Err: fmt.Errorf("该次运行数据不在内存中，请重新运行")}
			}
		}
		return RunStateMsg{State: state, FromHistory: true}
	}
}

// summaryToRunState 用 TaskRunSummary 摘要数据构造最小化 RunState，供无磁盘快照时回退展示。
func summaryToRunState(s *types.TaskRunSummary) *server.RunState {
	status := server.RunStatusCompleted
	switch s.Status {
	case string(server.RunStatusRunning):
		status = server.RunStatusRunning
	case string(server.RunStatusFailed):
		status = server.RunStatusFailed
	case string(server.RunStatusStopped):
		status = server.RunStatusStopped
	}
	var finished *time.Time
	if !s.FinishedAt.IsZero() {
		finishedAt := s.FinishedAt
		finished = &finishedAt
	}
	return &server.RunState{
		RunID:        server.RunID(s.RunID),
		TaskID:       s.TaskID,
		Status:       status,
		Mode:         s.Mode,
		StartedAt:    s.StartedAt,
		FinishedAt:   finished,
		AvgTPS:       s.AvgTPS,
		AvgTTFT:      s.AvgTTFT,
		SuccessRate:  s.SuccessRate,
		CacheHitRate: s.CacheHitRate,
		ErrorMsg:     s.ErrorSummary,
	}
}

// GenerateReportCmd 异步生成报告文件。
func (c *Client) GenerateReportCmd(runID server.RunID, format server.ReportFormat) tea.Cmd {
	return func() tea.Msg {
		path, err := c.srv.GenerateReport(runID, format)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("生成报告失败: %w", err)}
		}
		return ReportGeneratedMsg{RunID: runID, Path: path}
	}
}
