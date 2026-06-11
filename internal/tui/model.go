// Package tui implements the interactive terminal UI for AIT.
// The TUI is built with BubbleTea; all server interactions go through Client.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/config"
	"github.com/yinxulai/ait/internal/server/types"
	"github.com/yinxulai/ait/internal/tui/pages"
)

// ─── 视图状态 ─────────────────────────────────────────────────────────────────

type viewState string

const (
	viewTaskList   viewState = "task-list"
	viewTaskDetail viewState = "task-detail"
	viewWizard     viewState = "wizard"
	viewDashboard  viewState = "dashboard"
	viewTurboDash  viewState = "turbo-dash"
	viewReqDetail  viewState = "req-detail"
	viewProxy      viewState = "proxy"
	viewHelp       viewState = "help"
)

// ─── 根 Model ─────────────────────────────────────────────────────────────────

// Model 是 BubbleTea 的根状态机。
// 所有 Server 交互均通过 Client 发出 tea.Cmd；Model 不直接 import runner/task/turbo 等下层包。
type Model struct {
	client *Client
	styles pages.Styles
	width  int
	height int
	view   viewState
	status string
	err    error

	// 页面局部状态（由 pages 包管理）
	taskList  *pages.TaskListState
	detail    *pages.TaskDetailState
	wizard    *pages.WizardState
	dash      *pages.DashboardState
	turboDash *pages.TurboDashState
	reqDetail *pages.ReqDetailState
	proxyConf *pages.ProxyConfigState
	help      *pages.HelpState
}

// NewModel 创建 Model。srv 不能为 nil。
func NewModel(srv server.Server) *Model {
	return &Model{
		client:   NewClient(srv),
		styles:   pages.NewStyles(),
		view:     viewTaskList,
		taskList: pages.NewTaskListState(),
	}
}

// Run 启动 BubbleTea 全屏程序。是此包的主要外部入口。
func Run(srv server.Server) error {
	m := NewModel(srv)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// SetVersion 设置 AppHeader 中显示的版本字符串，应在 Run 之前调用。
func SetVersion(v string) { pages.SetAppVersion(v) }

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
		if m.taskList == nil {
			m.taskList = pages.NewTaskListState()
		}
		// 刷新前记录当前选中任务的 ID，刷新后保持光标指向同一任务。
		// 任务列表按 UpdatedAt 排序，编辑/复制任务后顺序会变化，若只靠下标
		// 定位会导致光标悄悄滑到别的任务上，进入错误任务的详情页。
		var prevID string
		if t, ok := m.taskList.CurrentTask(); ok {
			prevID = t.ID
		}
		m.taskList.Tasks = msg.Tasks
		if prevID != "" {
			for i, t := range msg.Tasks {
				if t.ID == prevID {
					m.taskList.Selected = i
					break
				}
			}
		}
		if m.taskList.Selected >= len(msg.Tasks) {
			m.taskList.Selected = max(len(msg.Tasks)-1, 0)
		}
		return m, nil

	// ── 任务保存完成（新建或更新） ──
	case TaskSavedMsg:
		m.status = fmt.Sprintf("任务 %q 已保存", msg.Task.Name)
		notRunning := (m.dash == nil || !m.dash.IsRunning()) && (m.turboDash == nil || !m.turboDash.IsRunning())
		if msg.AutoStart && notRunning {
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
		if m.detail != nil && m.detail.Task.ID == msg.TaskID {
			autoExpand := m.view == viewTaskDetail && len(msg.History) > 0
			m.detail = pages.UpdateTaskDetailHistory(m.detail, msg.History, autoExpand)
		}
		return m, nil

	// ── 运行启动 ──
	case RunStartedMsg:
		ch, cancel, firstCmd := m.client.SubscribeRunEventsCmd(msg.RunID)
		taskMode := m.getTaskMode(msg.TaskID)
		backNav := pages.NavAction{To: pages.NavTaskDetail, TaskID: msg.TaskID}
		if taskMode == "turbo" {
			m.turboDash = pages.NewTurboDashState(msg.RunID, msg.TaskID)
			m.turboDash.EventCh = ch
			m.turboDash.CancelFn = cancel
			m.turboDash.BackNav = backNav
			m.view = viewTurboDash
		} else {
			m.dash = pages.NewDashboardState(msg.RunID, msg.TaskID)
			m.dash.EventCh = ch
			m.dash.CancelFn = cancel
			m.dash.BackNav = backNav
			m.view = viewDashboard
		}
		if m.taskList != nil {
			m.taskList.ActiveRuns[msg.TaskID] = nil
		}
		m.status = ""
		return m, firstCmd

	// ── 停止请求已发送 ──
	case RunStopRequestedMsg:
		m.status = "已发送停止信号，等待当前请求收尾"
		return m, nil

	// ── Server 事件（来自运行中订阅） ──
	case ServerEventMsg:
		return m.handleServerEvent(msg)

	// ── 运行状态快照（重入仪表盘时 / 从历史导航时） ──
	case RunStateMsg:
		if msg.State == nil {
			return m, nil
		}
		if msg.FromHistory {
			backNav := pages.NavAction{To: pages.NavTaskDetail, TaskID: msg.State.TaskID}
			if msg.State.Mode == "turbo" {
				if m.turboDash == nil || m.turboDash.RunID != msg.State.RunID {
					m.turboDash = pages.NewTurboDashState(msg.State.RunID, msg.State.TaskID)
				}
				m.turboDash.RunState = msg.State
				m.turboDash.BackNav = backNav
				m.view = viewTurboDash
			} else {
				if m.dash == nil || m.dash.RunID != msg.State.RunID {
					m.dash = pages.NewDashboardState(msg.State.RunID, msg.State.TaskID)
				}
				m.dash.RunState = msg.State
				m.dash.BackNav = backNav
				m.view = viewDashboard
			}
			return m, nil
		}
		if m.dash != nil && m.dash.RunID == msg.State.RunID {
			m.dash.RunState = msg.State
		} else if m.turboDash != nil && m.turboDash.RunID == msg.State.RunID {
			m.turboDash.RunState = msg.State
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

	// ── 代理配置 ──
	case ProxyConfigLoadedMsg:
		m.proxyConf = pages.NewProxyConfigState(msg.ProxyURL)
		return m, nil

	case ProxyConfigSavedMsg:
		m.status = "代理配置已保存"
		return m, nil
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width < 4 || m.height < 4 {
		return "..."
	}
	innerW := m.width
	innerH := m.height

	// 状态/错误提示条占用一行
	var banner string
	if m.err != nil {
		banner = m.styles.ErrStyle.Width(innerW).Render(" ✗ " + m.err.Error())
		innerH--
	} else if m.status != "" {
		banner = m.styles.Ok.Width(innerW).Render(" ✓ " + m.status)
		innerH--
	}

	var content string
	switch m.view {
	case viewTaskList:
		content = pages.RenderTaskList(m.taskList, m.styles, innerW, innerH)
	case viewTaskDetail:
		content = pages.RenderTaskDetail(m.detail, m.styles, innerW, innerH)
	case viewWizard:
		content = pages.RenderWizard(m.wizard, m.styles, innerW, innerH)
	case viewDashboard:
		content = pages.RenderDashboard(m.dash, m.dashTaskName(), m.styles, innerW, innerH)
	case viewTurboDash:
		content = pages.RenderTurboDash(m.turboDash, m.turboDashTaskName(), m.styles, innerW, innerH)
	case viewReqDetail:
		content = pages.RenderReqDetail(m.reqDetail, m.reqDetailTaskName(), m.styles, innerW, innerH)
	case viewProxy:
		content = pages.RenderProxyConfig(m.proxyConf, m.styles, innerW, innerH)
	case viewHelp:
		content = pages.RenderHelp(m.help, m.styles, innerW, innerH)
	default:
		content = "未知视图"
	}

	if banner != "" {
		return banner + "\n" + content
	}
	return content
}

// ─── 键盘分发 ─────────────────────────────────────────────────────────────────

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 任意按键清除状态提示
	m.status = ""
	m.err = nil

	// ── 全局快捷键（所有页面层共享）──
	if msg.String() == "f2" {
		if i18n.Active() == i18n.ZH {
			i18n.SetLang(i18n.EN)
		} else {
			i18n.SetLang(i18n.ZH)
		}
		return m, saveLangConfigCmd(i18n.Active())
	}

	switch m.view {
	case viewTaskList:
		newState, cmd, nav := pages.HandleTaskListKey(m.taskList, msg, m.client)
		m.taskList = newState
		navCmd := m.handleNav(nav)
		return m, tea.Batch(cmd, navCmd)

	case viewTaskDetail:
		newState, cmd, nav := pages.HandleTaskDetailKey(m.detail, msg, m.client)
		m.detail = newState
		navCmd := m.handleNav(nav)
		return m, tea.Batch(cmd, navCmd)

	case viewWizard:
		newState, cmd, nav := pages.HandleWizardKey(m.wizard, msg, m.client)
		m.wizard = newState
		navCmd := m.handleNav(nav)
		return m, tea.Batch(cmd, navCmd)

	case viewDashboard:
		newState, cmd, nav := pages.HandleDashboardKey(m.dash, msg, m.client)
		m.dash = newState
		navCmd := m.handleNav(nav)
		return m, tea.Batch(cmd, navCmd)

	case viewTurboDash:
		newState, cmd, nav := pages.HandleTurboDashKey(m.turboDash, msg, m.client)
		m.turboDash = newState
		navCmd := m.handleNav(nav)
		return m, tea.Batch(cmd, navCmd)

	case viewReqDetail:
		newState, nav := pages.HandleReqDetailKey(m.reqDetail, msg)
		m.reqDetail = newState
		return m, m.handleNav(nav)

	case viewProxy:
		newState, cmd, nav := pages.HandleProxyConfigKey(m.proxyConf, msg, m.client)
		m.proxyConf = newState
		navCmd := m.handleNav(nav)
		return m, tea.Batch(cmd, navCmd)

	case viewHelp:
		newState, nav := pages.HandleHelpKey(m.help, msg)
		m.help = newState
		return m, m.handleNav(nav)
	}

	return m, nil
}

// ─── 导航处理 ─────────────────────────────────────────────────────────────────

func (m *Model) handleNav(nav pages.NavAction) tea.Cmd {
	switch nav.To {
	case pages.NavNone:
		return nil

	case pages.NavTaskList:
		m.view = viewTaskList
		return m.client.LoadTasksCmd()

	case pages.NavTaskDetail:
		backNav := m.taskDetailBackNav()
		task := m.findTask(nav.TaskID)
		if task != nil {
			if m.detail != nil && m.detail.Task.ID == task.ID {
				m.detail.Task = *task
			} else {
				m.detail = pages.NewTaskDetailState(*task)
			}
		} else {
			// 目标任务不在列表中（已删除或列表尚未加载），中止导航
			return nil
		}
		if m.detail != nil {
			m.detail.BackNav = backNav
		}
		// 若该任务有正在运行的实例，注入快照
		if m.detail != nil && m.taskList != nil {
			if rs, ok := m.taskList.ActiveRuns[m.detail.Task.ID]; ok && rs != nil {
				m.detail.ActiveRun = rs
			}
		}
		m.view = viewTaskDetail
		if m.detail != nil {
			return m.client.LoadTaskRunHistoryCmd(m.detail.Task.ID, 10)
		}
		return nil

	case pages.NavWizard:
		if nav.EditTask != nil {
			m.wizard = pages.NewWizardStateEdit(nav.EditTask)
		} else {
			m.wizard = pages.NewWizardState()
		}
		m.view = viewWizard
		return nil

	case pages.NavDashboard:
		if m.dash != nil {
			m.view = viewDashboard
		}
		return nil

	case pages.NavTurboDash:
		if m.turboDash != nil {
			m.view = viewTurboDash
		}
		return nil

	case pages.NavRunDetail:
		// 从历史记录进入某次运行的仪表盘
		return m.client.GetRunStateForHistoryCmd(nav.RunID, nav.Summary)

	case pages.NavReqDetail:
		reqs := m.collectRequests()
		s := pages.NewReqDetailState(m.currentRunID(), reqs, nav.ReqIndex)
		// 记录来源页面，用于 b/esc 返回
		if m.view == viewTurboDash {
			s.BackNav = pages.NavAction{To: pages.NavTurboDash}
		} else {
			s.BackNav = pages.NavAction{To: pages.NavDashboard}
		}
		m.reqDetail = s
		m.view = viewReqDetail
		return nil

	case pages.NavProxy:
		m.proxyConf = pages.NewProxyConfigState("")
		m.view = viewProxy
		return m.client.LoadProxyConfigCmd()

	case pages.NavHelp:
		m.help = pages.NewHelpState(pages.NavAction{To: m.currentNavTarget()})
		m.view = viewHelp
		return nil

	case pages.NavQuit:
		return tea.Quit
	}
	return nil
}

// currentNavTarget 返回当前视图对应的 NavTarget，用于帮助页的返回导航。
func (m *Model) currentNavTarget() pages.NavTarget {
	switch m.view {
	case viewTaskList:
		return pages.NavTaskList
	case viewTaskDetail:
		return pages.NavTaskDetail
	case viewWizard:
		return pages.NavWizard
	case viewDashboard:
		return pages.NavDashboard
	case viewTurboDash:
		return pages.NavTurboDash
	case viewReqDetail:
		return pages.NavReqDetail
	case viewProxy:
		return pages.NavProxy
	default:
		return pages.NavTaskList
	}
}

// ─── Server 事件处理 ──────────────────────────────────────────────────────────

// saveLangConfigCmd 将语言设置异步保存到配置文件（尽力而为）。
func saveLangConfigCmd(lang i18n.Lang) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			cfg = &config.Config{}
		}
		if lang == i18n.EN {
			cfg.Lang = "en"
		} else {
			cfg.Lang = "zh"
		}
		_ = cfg.Save()
		return nil
	}
}

func (m *Model) handleServerEvent(msg ServerEventMsg) (tea.Model, tea.Cmd) {
	e := msg.Event

	isDash := m.dash != nil && m.dash.RunID == e.RunID
	isTurbo := m.turboDash != nil && m.turboDash.RunID == e.RunID

	if !isDash && !isTurbo {
		return m, nil
	}

	switch e.Kind {
	case server.EventProgressTick, server.EventRequestDone, server.EventLevelDone:
		if rs, ok := e.Payload.(*server.RunState); ok {
			if isDash {
				m.dash.RunState = rs
			} else {
				m.turboDash.RunState = rs
			}
			m.injectRunState(rs)
		}

	case server.EventRunComplete:
		if rs, ok := e.Payload.(*server.RunState); ok {
			if isDash {
				m.dash.RunState = rs
			} else {
				m.turboDash.RunState = rs
			}
		}
		taskID := m.currentRunTaskID(isDash)
		if m.taskList != nil {
			delete(m.taskList.ActiveRuns, taskID)
		}
		if m.detail != nil && m.detail.Task.ID == taskID {
			m.detail.ActiveRun = nil
		}
		// 在后台刷新任务列表和历史，不自动跳转页面；用户可按 b/Esc 返回
		return m, tea.Batch(
			m.client.LoadTasksCmd(),
			m.client.LoadTaskRunHistoryCmd(taskID, 10),
		)

	case server.EventRunFailed:
		var errorMsg string
		if rs, ok := e.Payload.(*server.RunState); ok {
			if isDash {
				m.dash.RunState = rs
			} else {
				m.turboDash.RunState = rs
			}
			errorMsg = rs.ErrorMsg
		}
		if errorMsg == "" {
			errorMsg = "运行异常终止"
		}
		m.err = fmt.Errorf("运行失败: %s", errorMsg)
		taskID := m.currentRunTaskID(isDash)
		if m.taskList != nil {
			delete(m.taskList.ActiveRuns, taskID)
		}
		if m.detail != nil && m.detail.Task.ID == taskID {
			m.detail.ActiveRun = nil
		}
		// 在后台刷新任务列表和历史，不自动跳转页面；用户可按 b/Esc 返回
		return m, tea.Batch(
			m.client.LoadTasksCmd(),
			m.client.LoadTaskRunHistoryCmd(taskID, 10),
		)
	case server.EventIntegrityCaseStarted:
		if rs, ok := e.Payload.(*server.RunState); ok {
			if m.dash != nil {
				m.dash.RunState = rs
			}
			m.injectRunState(rs)
		}

	case server.EventIntegrityCaseDone:
		if rs, ok := e.Payload.(*server.RunState); ok {
			if m.dash != nil {
				m.dash.RunState = rs
			}
			m.injectRunState(rs)
		}

	case server.EventAssertionResult:
		if assertions, ok := e.Payload.([]types.AssertionResult); ok {
			_ = assertions
		}
	}

	// 继续等待下一条事件
	var ch <-chan server.Event
	if isDash && m.dash.EventCh != nil {
		ch = m.dash.EventCh
	} else if isTurbo && m.turboDash.EventCh != nil {
		ch = m.turboDash.EventCh
	}
	if ch != nil {
		return m, WaitEventCmd(ch)
	}
	return m, nil
}

// ─── 辅助方法 ─────────────────────────────────────────────────────────────────

func (m *Model) getTaskMode(taskID string) string {
	t := m.findTask(taskID)
	if t != nil && t.Input.Turbo {
		return "turbo"
	}
	return "standard"
}

func (m *Model) findTask(taskID string) *types.TaskDefinition {
	if m.taskList == nil {
		return nil
	}
	for i := range m.taskList.Tasks {
		if m.taskList.Tasks[i].ID == taskID {
			return &m.taskList.Tasks[i].TaskDefinition
		}
	}
	return nil
}

func (m *Model) injectRunState(rs *server.RunState) {
	if m.taskList == nil || rs == nil {
		return
	}
	if rs.Status == server.RunStatusRunning {
		m.taskList.ActiveRuns[rs.TaskID] = rs
	} else {
		delete(m.taskList.ActiveRuns, rs.TaskID)
	}
	// 如果详情页正在显示该任务，同步更新 ActiveRun
	if m.detail != nil && m.detail.Task.ID == rs.TaskID {
		if rs.Status == server.RunStatusRunning {
			m.detail.ActiveRun = rs
		} else {
			m.detail.ActiveRun = nil
		}
	}
}

func (m *Model) dashTaskName() string {
	if m.dash == nil {
		return "─"
	}
	t := m.findTask(m.dash.TaskID)
	if t != nil {
		return t.Name
	}
	return m.dash.TaskID
}

func (m *Model) turboDashTaskName() string {
	if m.turboDash == nil {
		return "─"
	}
	t := m.findTask(m.turboDash.TaskID)
	if t != nil {
		return t.Name
	}
	return m.turboDash.TaskID
}

func (m *Model) reqDetailTaskName() string {
	// 根据 reqDetail 的来源视图确定任务名，避免两个面板均有状态时取错
	if m.reqDetail != nil && m.reqDetail.BackNav.To == pages.NavTurboDash {
		if m.turboDash != nil {
			if t := m.findTask(m.turboDash.TaskID); t != nil {
				return t.Name
			}
		}
	}
	if m.dash != nil {
		if t := m.findTask(m.dash.TaskID); t != nil {
			return t.Name
		}
	}
	if m.turboDash != nil {
		if t := m.findTask(m.turboDash.TaskID); t != nil {
			return t.Name
		}
	}
	return "─"
}

func (m *Model) currentRunID() server.RunID {
	if m.dash != nil {
		return m.dash.RunID
	}
	if m.turboDash != nil {
		return m.turboDash.RunID
	}
	return ""
}

func (m *Model) currentRunTaskID(isDash bool) string {
	if isDash && m.dash != nil {
		return m.dash.TaskID
	}
	if m.turboDash != nil {
		return m.turboDash.TaskID
	}
	return ""
}

func (m *Model) taskDetailBackNav() pages.NavAction {
	return pages.NavAction{To: pages.NavTaskList}
}

func (m *Model) collectRequests() []*types.RequestMetrics {
	// 优先使用当前活跃视图的数据，避免两个面板均有 RunState 时取错
	switch m.view {
	case viewTurboDash:
		if m.turboDash != nil && m.turboDash.RunState != nil {
			return m.turboDash.RunState.Requests
		}
	case viewDashboard:
		if m.dash != nil && m.dash.RunState != nil {
			return m.dash.RunState.Requests
		}
	}
	return nil
}

// ─── 工具 ─────────────────────────────────────────────────────────────────────

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
