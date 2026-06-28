package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// MainPage 是主布局页面（3-panel：任务列表 / 运行历史 / 统计+请求）。
type MainPage struct {
	app          *tview.Application
	srv          server.Server
	version      string
	header       *tview.TextView
	footer       *tview.TextView
	taskTable    *tview.Table
	historyTable *tview.Table
	statsPanel   *tview.Flex
	statsBox     *tview.Flex
	statsTable   *tview.Table
	progressText *tview.TextView
	requestTable *tview.Table
	activePanel  panelIndex
	root         *tview.Flex

	// 数据（用于 Enter 导航查找 + Refresh 重建）
	tasks          []types.TaskOverview
	runs           []types.TaskRunSummary
	requests       []types.RequestMetrics
	runRequests    map[int][]types.RequestMetrics // key = runs 切片索引
	totalReqs      int
	selectedTaskID string
	selectedRunID  string
}

// NewMainPage 创建主布局页面。
func NewMainPage(app *tview.Application, srv server.Server, version string) *MainPage {
	m := &MainPage{
		app:         app,
		srv:         srv,
		version:     version,
		runRequests: make(map[int][]types.RequestMetrics),
	}

	// Header
	m.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	m.header.SetText(fmt.Sprintf("◆ AIT  v%s", version))

	// Footer
	m.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	// Task Table (左 panel)
	m.taskTable = newSelectableTable(" Tasks ")
	m.taskTable.SetBorder(true).
		SetTitle(" Tasks ").
		SetTitleAlign(tview.AlignLeft)
	m.taskTable.SetSelectionChangedFunc(func(_, _ int) {
		m.updateFooter()
	})

	// History Table (中 panel)
	m.historyTable = newSelectableTable(" Run History ")
	m.historyTable.SetBorder(true).
		SetTitle(" Run History ").
		SetTitleAlign(tview.AlignLeft)
	m.historyTable.SetSelectionChangedFunc(func(_, _ int) {
		m.updateFooter()
	})

	// Stats Box (右 panel 上)
	m.statsTable = tview.NewTable().
		SetSelectable(false, false).
		SetSeparator(' ')
	m.progressText = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	m.statsBox = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.statsTable, 0, 1, false).
		AddItem(m.progressText, 1, 0, false)
	m.statsBox.SetBorder(true).
		SetTitle(" Statistics ").
		SetTitleAlign(tview.AlignLeft)

	// Request Table (右 panel 下)
	m.requestTable = newSelectableTable(" Requests ")
	m.requestTable.SetBorder(true).
		SetTitle(" Requests ").
		SetTitleAlign(tview.AlignLeft)
	m.requestTable.SetSelectionChangedFunc(func(_, _ int) {
		m.updateFooter()
	})

	// 右 panel 布局：statsText 上 + requestList 下
	m.statsPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.statsBox, 0, 4, false).
		AddItem(m.requestTable, 0, 6, true)

	// 三栏布局
	content := tview.NewFlex().
		AddItem(m.taskTable, 0, 2, false).
		AddItem(m.historyTable, 0, 2, false).
		AddItem(m.statsPanel, 0, 4, false)

	// 根布局：Header + Content + Footer
	m.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.header, 1, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(m.footer, 1, 0, false)

	m.activePanel = panelTasks
	m.updatePanelStyles()
	m.updateFooter()

	return m
}

// ─── Page 接口实现 ─────────────────────────────────────────────────────────────

func (m *MainPage) Name() string               { return "main" }
func (m *MainPage) Primitive() tview.Primitive { return m.root }
func (m *MainPage) OnActivate()                {}
func (m *MainPage) OnDeactivate()              {}

func (m *MainPage) FocusTarget() tview.Primitive {
	switch m.activePanel {
	case panelTasks:
		return m.taskTable
	case panelHistory:
		return m.historyTable
	default:
		return m.requestTable
	}
}

func (m *MainPage) HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyTab:
		m.nextPanel()
		m.app.SetFocus(m.FocusTarget())
		return nil
	case tcell.KeyBacktab:
		m.prevPanel()
		m.app.SetFocus(m.FocusTarget())
		return nil
	case tcell.KeyEnter:
		switch m.activePanel {
		case panelTasks:
			m.selectCurrentTask()
			m.activePanel = panelHistory
			m.updatePanelStyles()
			m.updateFooter()
			m.app.SetFocus(m.FocusTarget())
		case panelHistory:
			m.selectCurrentRun()
			m.activePanel = panelStats
			m.updatePanelStyles()
			m.updateFooter()
			m.app.SetFocus(m.FocusTarget())
		case panelStats:
			target := m.requestDetailPage()
			if target != nil {
				router.NavigateTo(target)
				m.app.SetFocus(target.FocusTarget())
			}
		}
		return nil
	case tcell.KeyEsc:
		m.app.Stop()
		return nil
	}

	// 操作按键
	switch event.Rune() {
	case 'r':
		if m.activePanel == panelTasks {
			idx := selectedDataIndex(m.taskTable)
			if idx >= 0 && idx < len(m.tasks) {
				go func() { _, _ = m.srv.StartRun(m.tasks[idx].ID) }()
			}
		}
		return nil
	case 's':
		if m.activePanel == panelHistory {
			idx := selectedDataIndex(m.historyTable)
			if idx >= 0 && idx < len(m.runs) && m.runs[idx].Status == "running" {
				go func() { _ = m.srv.StopRun(server.RunID(m.runs[idx].RunID)) }()
			}
		}
		return nil
	case 'R':
		switch m.activePanel {
		case panelTasks:
			go func() {
				tasks, _ := m.srv.ListTasks()
				m.app.QueueUpdateDraw(func() { m.RefreshTasks(tasks) })
			}()
		case panelHistory:
			go func() {
				idx := selectedDataIndex(m.taskTable)
				if idx >= 0 && idx < len(m.tasks) {
					runs, _ := m.srv.ListTaskRunHistory(m.tasks[idx].ID, 50)
					m.app.QueueUpdateDraw(func() { m.RefreshRuns(runs) })
				}
			}()
		}
		return nil
	}

	return event // ↑↓ 等透传给 tview Table
}

func (m *MainPage) selectCurrentTask() {
	idx := selectedDataIndex(m.taskTable)
	if idx < 0 || idx >= len(m.tasks) {
		return
	}
	taskID := m.tasks[idx].ID
	if taskID == "" {
		return
	}
	if taskID == m.selectedTaskID {
		m.selectCurrentRun()
		return
	}
	m.selectedTaskID = taskID
	m.selectedRunID = ""
	go func() {
		runs, _ := m.srv.ListTaskRunHistory(taskID, 50)
		m.app.QueueUpdateDraw(func() {
			if m.selectedTaskID == taskID {
				m.RefreshRuns(runs)
			}
		})
	}()
}

func (m *MainPage) selectCurrentRun() {
	idx := selectedDataIndex(m.historyTable)
	if idx < 0 || idx >= len(m.runs) {
		m.selectedRunID = ""
		m.requests = nil
		m.RebuildStatsAndRequests()
		return
	}

	m.selectedRunID = m.runs[idx].RunID
	if state, ok := m.srv.GetRunState(server.RunID(m.selectedRunID)); ok {
		m.requests = requestMetricsFromState(state)
	} else if reqs, ok := m.runRequests[idx]; ok {
		m.requests = reqs
	} else {
		m.requests = nil
	}
	m.RebuildStatsAndRequests()
}

func requestMetricsFromState(state *server.RunState) []types.RequestMetrics {
	if state == nil || len(state.Requests) == 0 {
		return nil
	}
	requests := make([]types.RequestMetrics, 0, len(state.Requests))
	for _, request := range state.Requests {
		if request != nil {
			requests = append(requests, *request)
		}
	}
	return requests
}

func (m *MainPage) requestDetailPage() Page {
	idx := selectedDataIndex(m.requestTable)
	if idx >= 0 && idx < len(m.requests) {
		return NewRequestDetailPage(&m.requests[idx])
	}
	return nil
}

// ─── Panel 切换 ────────────────────────────────────────────────────────────────

func (m *MainPage) nextPanel() {
	m.activePanel = (m.activePanel + 1) % 3
	m.updatePanelStyles()
	m.updateFooter()
}

func (m *MainPage) prevPanel() {
	m.activePanel = (m.activePanel + 2) % 3 // +2 = -1 mod 3
	m.updatePanelStyles()
	m.updateFooter()
}

func (m *MainPage) updatePanelStyles() {
	activeColor := tcell.ColorYellow
	inactiveColor := tcell.ColorGray
	activeBg := tcell.ColorDarkBlue
	inactiveBg := tcell.ColorBlack

	for i, table := range []*tview.Table{m.taskTable, m.historyTable, m.requestTable} {
		if panelIndex(i) == m.activePanel {
			table.SetBorderColor(activeColor)
			table.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(activeBg))
		} else {
			table.SetBorderColor(inactiveColor)
			table.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(inactiveBg))
		}
	}
}

func (m *MainPage) updateFooter() {
	var txt string
	switch m.activePanel {
	case panelTasks:
		txt = "Tab=切换面板  ↑↓=移动  Enter=选择任务  r=运行  R=刷新  Esc=退出"
	case panelHistory:
		txt = "Tab=切换面板  ↑↓=移动  Enter=选择运行  s=停止  Esc=退出"
	case panelStats:
		txt = "Tab=切换面板  ↑↓=选择  Enter=请求详情  Esc=退出"
	}
	m.footer.SetText("[::b]" + txt + "[::-]")
}

// ─── 数据刷新 ──────────────────────────────────────────────────────────────────

// SetRunRequests 设置 run→requests 映射（静态数据）。
func (m *MainPage) SetRunRequests(rr map[int][]types.RequestMetrics) {
	m.runRequests = rr
}

// SetTotalReqs 设置总请求数（用于统计展示）。
func (m *MainPage) SetTotalReqs(n int) {
	m.totalReqs = n
}

// PopulateData 一次性注入初始 mock 数据。
func (m *MainPage) PopulateData(
	tasks []types.TaskOverview,
	runs []types.TaskRunSummary,
	reqs []types.RequestMetrics,
	runReqs map[int][]types.RequestMetrics,
	totalReqs int,
) {
	m.tasks = tasks
	m.runs = runs
	m.requests = reqs
	m.runRequests = runReqs
	m.totalReqs = totalReqs
	m.selectedTaskID = ""
	m.selectedRunID = ""

	m.RebuildTaskList()
	m.RebuildHistoryList()
	m.selectCurrentTask()
	m.selectCurrentRun()
}

// RebuildTaskList 清空并重建任务表。
func (m *MainPage) RebuildTaskList() {
	rows := make([][]string, 0, len(m.tasks))
	for _, t := range m.tasks {
		rows = append(rows, formatTaskRow(t))
	}
	setTableRows(m.taskTable, taskColumns(), rows)
}

// RebuildHistoryList 清空并重建历史表。
func (m *MainPage) RebuildHistoryList() {
	rows := make([][]string, 0, len(m.runs))
	for _, r := range m.runs {
		rows = append(rows, formatRunRow(r))
	}
	setTableRows(m.historyTable, runColumns(), rows)
}

// RebuildStatsAndRequests 重建统计文本和请求表。
func (m *MainPage) RebuildStatsAndRequests() {
	isCompleted := true
	for _, r := range m.runs {
		if r.Status == "running" {
			isCompleted = false
			break
		}
	}

	// 更新 stats 标题
	done := len(m.requests)
	if !isCompleted && m.totalReqs > 0 {
		pct := done * 100 / m.totalReqs
		m.statsBox.SetTitle(fmt.Sprintf(" Statistics [Running %d%%] ", pct))
	} else {
		m.statsBox.SetTitle(" Statistics [Completed 100%] ")
	}

	m.rebuildStatsTable(isCompleted)
	m.progressText.SetText(formatStatsProgress(m.requests, m.totalReqs, isCompleted, 0, 36))

	rows := make([][]string, 0, len(m.requests))
	for _, r := range m.requests {
		rows = append(rows, formatReqRow(r))
	}
	setTableRows(m.requestTable, requestColumns(), rows)
}

func (m *MainPage) rebuildStatsTable(isCompleted bool) {
	stats := calculateStats(m.requests, m.totalReqs, isCompleted, 0)
	rows := [][4]string{
		{"总请求数", fmt.Sprintf("%d", stats.TotalReqs), "成功率", fmt.Sprintf("%.1f%%", stats.SuccessRate)},
		{"完成", fmt.Sprintf("%d", stats.Done), "失败", fmt.Sprintf("%d", stats.Failed)},
		{"平均 TPS", fmt.Sprintf("%.1f", stats.AvgTPS), "缓存命中率", fmt.Sprintf("%.1f%%", stats.AvgCacheRate)},
		{"P50 TTFT", stats.P50TTFT.Truncate(time.Millisecond).String(), "P99 TTFT", stats.P99TTFT.Truncate(time.Millisecond).String()},
		{"RPM", fmt.Sprintf("%.0f", stats.RPM), "TPM", fmt.Sprintf("%.0f", stats.TPM)},
	}

	m.statsTable.Clear()
	for rowIdx, row := range rows {
		for colIdx, text := range row {
			cell := tview.NewTableCell(text).
				SetSelectable(false).
				SetExpansion(1).
				SetMaxWidth(24)
			if colIdx%2 == 0 {
				cell.SetTextColor(tcell.ColorGray)
			} else {
				cell.SetTextColor(tcell.ColorWhite)
			}
			m.statsTable.SetCell(rowIdx, colIdx, cell)
		}
	}
}

// RefreshTasks 清空并重建任务表（由 startUpdateLoop 调用）。
func (m *MainPage) RefreshTasks(tasks []types.TaskOverview) {
	oldIdx := selectedDataIndex(m.taskTable)
	oldTaskID := ""
	if oldIdx >= 0 && oldIdx < len(m.tasks) {
		oldTaskID = m.tasks[oldIdx].ID
	}
	m.tasks = tasks
	m.RebuildTaskList()
	selectDataIndex(m.taskTable, indexTaskByID(tasks, oldTaskID, oldIdx))
	m.selectedTaskID = ""
	m.selectCurrentTask()
	m.updatePanelStyles()
}

// RefreshRuns 清空并重建历史表（由 startUpdateLoop 调用）。
func (m *MainPage) RefreshRuns(runs []types.TaskRunSummary) {
	oldIdx := selectedDataIndex(m.historyTable)
	oldRunID := ""
	if oldIdx >= 0 && oldIdx < len(m.runs) {
		oldRunID = m.runs[oldIdx].RunID
	}
	m.runs = runs
	m.RebuildHistoryList()
	selectDataIndex(m.historyTable, indexRunByID(runs, oldRunID, oldIdx))
	m.selectCurrentRun()
	m.updatePanelStyles()
}

// RefreshRequests 重建统计和请求表（由 startUpdateLoop 调用）。
func (m *MainPage) RefreshRequests(reqs []types.RequestMetrics) {
	oldIdx := selectedDataIndex(m.requestTable)
	m.requests = reqs
	m.RebuildStatsAndRequests()
	selectDataIndex(m.requestTable, oldIdx)
	m.updatePanelStyles()
}

func indexTaskByID(tasks []types.TaskOverview, taskID string, fallback int) int {
	if taskID != "" {
		for idx, task := range tasks {
			if task.ID == taskID {
				return idx
			}
		}
	}
	if fallback >= 0 && fallback < len(tasks) {
		return fallback
	}
	return 0
}

func indexRunByID(runs []types.TaskRunSummary, runID string, fallback int) int {
	if runID != "" {
		for idx, run := range runs {
			if run.RunID == runID {
				return idx
			}
		}
	}
	if fallback >= 0 && fallback < len(runs) {
		return fallback
	}
	return 0
}
