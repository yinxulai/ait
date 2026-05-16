// Package tui implements the interactive terminal UI for AIT.
// The TUI is built with BubbleTea; all server interactions go through Client.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
)

// ─── 视图状态 ─────────────────────────────────────────────────────────────────

type viewState string

const (
	viewTaskList   viewState = "task-list"
	viewTaskDetail viewState = "task-detail"
	viewWizard     viewState = "wizard"
	viewDashboard  viewState = "dashboard"
	viewReqDetail  viewState = "req-detail"
)

// ─── 根 Model ─────────────────────────────────────────────────────────────────

// Model 是 BubbleTea 的根状态机。
// 所有 Server 交互均通过 Client 发出 tea.Cmd；Model 不直接 import runner/task/turbo。
type Model struct {
	client  *Client
	styles  styles
	width   int
	height  int
	view    viewState
	status  string
	err     error

	// 页面局部状态
	taskList taskListState
	hist     *historyState    // 任务详情页的历史
	wizard   *wizardState     // nil = 向导未打开
	dash     *dashboardState  // nil = 无活跃运行
	reqDetail *reqDetailState // nil = 不在请求详情页
}

// NewModel 创建 Model。srv 不能为 nil。
func NewModel(srv server.Server) *Model {
	return &Model{
		client: NewClient(srv),
		styles: newStyles(),
		view:   viewTaskList,
		taskList: taskListState{selected: 0},
	}
}

// Run 启动 BubbleTea 全屏程序。是此包的主要外部入口。
func Run(srv server.Server) error {
	m := NewModel(srv)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ─── BubbleTea 接口 ───────────────────────────────────────────────────────────

func (m *Model) Init() tea.Cmd {
	return m.client.LoadTasksCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── 窗口尺寸 ──
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// ── 键盘 ──
	case tea.KeyMsg:
		return m.handleKey(msg)

	// ── 任务列表加载完成 ──
	case TasksLoadedMsg:
		m.taskList.tasks = msg.Tasks
		// 调整选中项不越界
		if m.taskList.selected >= len(msg.Tasks) {
			m.taskList.selected = max(len(msg.Tasks)-1, 0)
		}
		m.status = ""
		m.err = nil
		return m, nil

	// ── 任务保存完成（新建或更新） ──
	case TaskSavedMsg:
		m.status = fmt.Sprintf("任务 %q 已保存", msg.Task.Name)
		// 若 AutoStart 且无活跃运行，立刻发起运行
		if msg.AutoStart && (m.dash == nil || !m.dash.isRunning()) {
			return m, tea.Batch(
				m.client.LoadTasksCmd(),
				m.client.StartRunCmd(msg.Task.ID),
			)
		}
		return m, m.client.LoadTasksCmd()

	// ── 任务删除完成 ──
	case TaskDeletedMsg:
		m.status = "任务已删除"
		m.view = viewTaskList
		return m, m.client.LoadTasksCmd()

	// ── 历史加载完成 ──
	case HistoryLoadedMsg:
		m.hist = &historyState{taskID: msg.TaskID, history: msg.History}
		return m, nil

	// ── 运行启动 ──
	case RunStartedMsg:
		ch, cancel, firstCmd := m.client.SubscribeCmd(msg.RunID)
		m.dash = &dashboardState{
			runID:    msg.RunID,
			taskID:   msg.TaskID,
			eventCh:  ch,
			cancelFn: cancel,
			reqSel:   -1,
		}
		m.view = viewDashboard
		m.status = ""
		return m, firstCmd

	// ── Server 事件（来自运行中订阅） ──
	case ServerEventMsg:
		return m.handleServerEvent(msg)

	// ── 运行状态快照（重入仪表盘时） ──
	case RunStateMsg:
		if m.dash != nil && msg.State != nil && m.dash.runID == msg.State.RunID {
			m.dash.runState = msg.State
		}
		return m, nil

	// ── 报告生成完成 ──
	case ReportGeneratedMsg:
		m.status = fmt.Sprintf("报告已生成: %s", msg.Path)
		return m, nil

	// ── 错误 ──
	case ErrorMsg:
		m.err = msg.Err
		return m, nil
	}

	return m, nil
}

func (m *Model) View() string {
	switch m.view {
	case viewTaskList:
		return m.renderTaskList()
	case viewTaskDetail:
		return m.renderTaskDetail()
	case viewWizard:
		return m.renderWizard()
	case viewDashboard:
		return m.renderDashboard()
	case viewReqDetail:
		return m.renderReqDetail()
	}
	return "未知视图"
}

// ─── 键盘分发 ─────────────────────────────────────────────────────────────────

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case viewTaskList:
		return m.handleTaskListKey(msg)
	case viewTaskDetail:
		return m, m.handleTaskDetailKey(msg)
	case viewWizard:
		return m.handleWizardKey(msg)
	case viewDashboard:
		return m.handleDashboardKey(msg)
	case viewReqDetail:
		return m.handleReqDetailKey(msg)
	}
	return m, nil
}

// ─── Server 事件处理 ──────────────────────────────────────────────────────────

func (m *Model) handleServerEvent(msg ServerEventMsg) (tea.Model, tea.Cmd) {
	if m.dash == nil {
		return m, nil
	}
	e := msg.Event

	switch e.Kind {
	case server.EventProgressTick:
		if rs, ok := e.Payload.(*server.RunState); ok {
			m.dash.runState = rs
		}

	case server.EventRequestDone:
		if rs, ok := e.Payload.(*server.RunState); ok {
			m.dash.runState = rs
		}

	case server.EventLevelDone:
		if rs, ok := e.Payload.(*server.RunState); ok {
			m.dash.runState = rs
		}

	case server.EventRunComplete:
		if rs, ok := e.Payload.(*server.RunState); ok {
			m.dash.runState = rs
		}
		// 运行结束后保留 dash 供用户查阅，切换到详情页
		m.view = viewTaskDetail
		return m, tea.Batch(
			m.client.LoadTasksCmd(),
			m.client.LoadHistoryCmd(m.dash.taskID, 10),
		)

	case server.EventRunFailed:
		if rs, ok := e.Payload.(*server.RunState); ok {
			m.dash.runState = rs
		}
		m.err = fmt.Errorf("运行失败: %s", m.dash.runState.ErrorMsg)
		m.view = viewTaskDetail
		return m, tea.Batch(
			m.client.LoadTasksCmd(),
			m.client.LoadHistoryCmd(m.dash.taskID, 10),
		)
	}

	// 若 eventCh 还在，继续等待下一条事件
	if m.dash.eventCh != nil {
		return m, WaitEventCmd(m.dash.eventCh)
	}
	return m, nil
}

// ─── 共享渲染工具 ─────────────────────────────────────────────────────────────

// renderHeader 渲染顶部状态栏（全宽，左侧标题 + 右侧信息）。
func (m *Model) renderHeader(title, right string) string {
	w := m.width
	if w < 1 {
		w = 80
	}
	titleW := lipgloss.Width(title)
	rightW := lipgloss.Width(right)
	pad := w - titleW - rightW - 2
	if pad < 1 {
		pad = 1
	}
	line := " " + title + strings.Repeat(" ", pad) + right + " "
	// 截断
	if lipgloss.Width(line) > w {
		line = line[:w]
	}
	return m.styles.header.Width(w).Render(line)
}

// renderFooter 渲染底部状态栏（全宽）。
func (m *Model) renderFooter(parts ...string) string {
	w := m.width
	if w < 1 {
		w = 80
	}
	var visible []string
	for _, p := range parts {
		if p != "" {
			visible = append(visible, p)
		}
	}
	line := "  " + strings.Join(visible, "  │  ")
	return m.styles.footer.Width(w).Render(line)
}

// dualColumnLayout 将左右内容放入双列布局，高度限制为 maxH。
func (m *Model) dualColumnLayout(left, right string, leftW, rightW, maxH int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// 裁剪至 maxH
	if len(leftLines) > maxH {
		leftLines = leftLines[:maxH]
	}
	if len(rightLines) > maxH {
		rightLines = rightLines[:maxH]
	}
	// 补齐行数
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}

	var rows []string
	for i := 0; i < maxH; i++ {
		lLine := leftLines[i]
		rLine := rightLines[i]
		lW := lipgloss.Width(lLine)
		if lW < leftW {
			lLine += strings.Repeat(" ", leftW-lW)
		}
		rows = append(rows, lLine+"  "+rLine)
	}
	return strings.Join(rows, "\n")
}

// ─── 工具 ─────────────────────────────────────────────────────────────────────

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
