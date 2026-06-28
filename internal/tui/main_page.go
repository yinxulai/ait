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
	app         *tview.Application
	srv         server.Server
	version     string
	header      *tview.TextView
	footer      *tview.TextView
	taskList    *tview.Table
	historyList *tview.Table
	statsPanel  *tview.Flex
	statsText   *tview.TextView
	requestList *tview.Table
	activePanel panelIndex
	root        *tview.Flex

	// 数据（用于 Enter 导航查找 + Refresh 重建）
	tasks       []types.TaskOverview
	runs        []types.TaskRunSummary
	requests    []types.RequestMetrics
	runRequests map[int][]types.RequestMetrics // key = runs 切片索引
	totalReqs   int
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
	m.taskList = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	m.taskList.SetBorder(true).
		SetTitle(" Tasks ").
		SetTitleAlign(tview.AlignLeft)
	m.taskList.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))
	m.taskList.SetSelectionChangedFunc(func(_ int, _ int) {
		m.updateFooter()
	})

	// History Table (中 panel)
	m.historyList = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	m.historyList.SetBorder(true).
		SetTitle(" Run History ").
		SetTitleAlign(tview.AlignLeft)
	m.historyList.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))
	m.historyList.SetSelectionChangedFunc(func(_ int, _ int) {
		m.updateFooter()
	})

	// Stats Text (右 panel 上)
	m.statsText = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	m.statsText.SetBorder(true).
		SetTitle(" Statistics ").
		SetTitleAlign(tview.AlignLeft)

	// Request Table (右 panel 下)
	m.requestList = tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)
	m.requestList.SetBorder(true).
		SetTitle(" Requests ").
		SetTitleAlign(tview.AlignLeft)
	m.requestList.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite))
	m.requestList.SetSelectionChangedFunc(func(_ int, _ int) {
		m.updateFooter()
	})

	// 右 panel 布局：statsText 上 + requestList 下
	m.statsPanel = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.statsText, 0, 4, false).
		AddItem(m.requestList, 0, 6, true)

	// 三栏布局
	content := tview.NewFlex().
		AddItem(m.taskList, 0, 2, false).
		AddItem(m.historyList, 0, 2, false).
		AddItem(m.statsPanel, 0, 4, false)

	// 根布局：Header + Content + Footer
	m.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(m.header, 1, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(m.footer, 1, 0, false)
	m.root.SetBorder(true).SetTitle(" AIT Main ")

	m.activePanel = panelTasks
	m.updatePanelStyles()
	m.updateFooter()

	return m
}

// ─── Page 接口实现 ─────────────────────────────────────────────────────────────

func (m *MainPage) Name() string                   { return "main" }
func (m *MainPage) Primitive() tview.Primitive      { return m.root }
func (m *MainPage) OnActivate()                     {}
func (m *MainPage) OnDeactivate()                   {}

func (m *MainPage) FocusTarget() tview.Primitive {
	switch m.activePanel {
	case panelTasks:
		return m.taskList
	case panelHistory:
		return m.historyList
	default:
		return m.requestList
	}
}

func (m *MainPage) selectedTaskIndex() int {
	row, _ := m.taskList.GetSelection()
	if row <= 0 || row > m.taskList.GetRowCount()-1 {
		return -1
	}
	return row - 1
}

func (m *MainPage) selectedHistoryIndex() int {
	row, _ := m.historyList.GetSelection()
	if row <= 0 || row > m.historyList.GetRowCount()-1 {
		return -1
	}
	return row - 1
}

func (m *MainPage) selectedRequestIndex() int {
	row, _ := m.requestList.GetSelection()
	if row <= 0 || row > m.requestList.GetRowCount()-1 {
		return -1
	}
	return row - 1
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
		target := m.EnterAction()
		if target != nil {
			router.NavigateTo(target)
			m.app.SetFocus(target.FocusTarget())
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
			idx := m.selectedTaskIndex()
			if idx >= 0 && idx < len(m.tasks) {
				go func() { _, _ = m.srv.StartRun(m.tasks[idx].ID) }()
			}
		}
		return nil
	case 's':
		if m.activePanel == panelHistory {
			idx := m.selectedHistoryIndex()
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
				idx := m.selectedTaskIndex()
				if idx >= 0 && idx < len(m.tasks) {
					runs, _ := m.srv.ListTaskRunHistory(m.tasks[idx].ID, 50)
					m.app.QueueUpdateDraw(func() { m.RefreshRuns(runs) })
				}
			}()
		}
		return nil
	}

	return event // ↑↓ 等透传给 tview List
}

// ─── Enter 导航 ────────────────────────────────────────────────────────────────

// EnterAction 根据当前 activePanel 和选中项返回目标页面，nil 表示无操作。
func (m *MainPage) EnterAction() Page {
	switch m.activePanel {
	case panelTasks:
		idx := m.selectedTaskIndex()
		if idx >= 0 && idx < len(m.tasks) {
			return NewTaskDetailPage(&m.tasks[idx])
		}
	case panelHistory:
		idx := m.selectedHistoryIndex()
		if idx >= 0 && idx < len(m.runs) {
			reqs, ok := m.runRequests[idx]
			if !ok {
				reqs = nil
			}
			return NewStandardDashboardPage(m.srv, &m.runs[idx], reqs, m.totalReqs)
		}
	case panelStats:
		idx := m.selectedRequestIndex()
		if idx >= 0 && idx < len(m.requests) {
			return NewRequestDetailPage(&m.requests[idx])
		}
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

	for i, table := range []*tview.Table{m.taskList, m.historyList, m.requestList} {
		if panelIndex(i) == m.activePanel {
			table.SetBorderColor(activeColor)
		} else {
			table.SetBorderColor(inactiveColor)
		}
	}
}

func (m *MainPage) updateFooter() {
	var txt string
	switch m.activePanel {
	case panelTasks:
		txt = "Tab=切换面板  ↑↓=选择  Enter=详情  r=运行  R=刷新  Esc=退出"
	case panelHistory:
		txt = "Tab=切换面板  ↑↓=选择  Enter=仪表盘  s=停止  Esc=退出"
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

	m.RebuildTaskList()
	m.RebuildHistoryList()
	m.RebuildStatsAndRequests()
}

// RebuildTaskList 清空并重建任务表格。
func (m *MainPage) RebuildTaskList() {
	m.taskList.Clear()
	m.taskList.SetCell(0, 0, tview.NewTableCell("Task").SetAlign(tview.AlignLeft).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.taskList.SetCell(0, 1, tview.NewTableCell("Mode").SetAlign(tview.AlignCenter).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.taskList.SetCell(0, 2, tview.NewTableCell("Status").SetAlign(tview.AlignCenter).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.taskList.SetCell(0, 3, tview.NewTableCell("SR").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.taskList.SetCell(0, 4, tview.NewTableCell("TPS").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	for i, t := range m.tasks {
		row := i + 1
		status := "idle"
		srText := "-"
		tpsText := "-"
		mode := t.Input.RunMode()
		if t.LatestRun != nil {
			status = t.LatestRun.Status
			srText = fmt.Sprintf("%.1f%%", t.LatestRun.SuccessRate*100)
			tpsText = fmt.Sprintf("%.0f", t.LatestRun.AvgTPS)
		}
		m.taskList.SetCell(row, 0, tview.NewTableCell(t.Name).SetAlign(tview.AlignLeft))
		m.taskList.SetCell(row, 1, tview.NewTableCell(mode).SetAlign(tview.AlignCenter))
		m.taskList.SetCell(row, 2, tview.NewTableCell(status).SetAlign(tview.AlignCenter))
		m.taskList.SetCell(row, 3, tview.NewTableCell(srText).SetAlign(tview.AlignRight))
		m.taskList.SetCell(row, 4, tview.NewTableCell(tpsText).SetAlign(tview.AlignRight))
	}
	if m.taskList.GetRowCount() > 1 {
		m.taskList.Select(1, 0)
	}
}

// RebuildHistoryList 清空并重建历史列表。
func (m *MainPage) RebuildHistoryList() {
	m.historyList.Clear()
	m.historyList.SetCell(0, 0, tview.NewTableCell("RunID").SetAlign(tview.AlignLeft).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.historyList.SetCell(0, 1, tview.NewTableCell("Mode").SetAlign(tview.AlignCenter).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.historyList.SetCell(0, 2, tview.NewTableCell("Start").SetAlign(tview.AlignCenter).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.historyList.SetCell(0, 3, tview.NewTableCell("Duration").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.historyList.SetCell(0, 4, tview.NewTableCell("SR").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.historyList.SetCell(0, 5, tview.NewTableCell("TPS").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	for i, r := range m.runs {
		row := i + 1
		shortID := r.RunID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		dur := r.FinishedAt.Sub(r.StartedAt).Truncate(time.Second)
		m.historyList.SetCell(row, 0, tview.NewTableCell(shortID).SetAlign(tview.AlignLeft))
		m.historyList.SetCell(row, 1, tview.NewTableCell(r.Mode).SetAlign(tview.AlignCenter))
		m.historyList.SetCell(row, 2, tview.NewTableCell(r.StartedAt.Format("01-02 15:04")).SetAlign(tview.AlignCenter))
		m.historyList.SetCell(row, 3, tview.NewTableCell(dur.String()).SetAlign(tview.AlignRight))
		m.historyList.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%.1f%%", r.SuccessRate*100)).SetAlign(tview.AlignRight))
		m.historyList.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%.0f", r.AvgTPS)).SetAlign(tview.AlignRight))
	}
	if m.historyList.GetRowCount() > 1 {
		m.historyList.Select(1, 0)
	}
}

// RebuildStatsAndRequests 重建统计文本和请求列表。
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
		m.statsText.SetTitle(fmt.Sprintf(" Statistics [Running %d%%] ", pct))
	} else {
		m.statsText.SetTitle(" Statistics [Completed 100%] ")
	}

	m.statsText.SetText(formatStats(m.requests, m.totalReqs, isCompleted, 0))

	m.requestList.Clear()
	m.requestList.SetCell(0, 0, tview.NewTableCell("#").SetAlign(tview.AlignLeft).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 1, tview.NewTableCell("Status").SetAlign(tview.AlignCenter).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 2, tview.NewTableCell("Total").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 3, tview.NewTableCell("TTFT").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 4, tview.NewTableCell("TPS").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 5, tview.NewTableCell("PT").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 6, tview.NewTableCell("CT").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	m.requestList.SetCell(0, 7, tview.NewTableCell("CH").SetAlign(tview.AlignRight).SetSelectable(false).SetAttributes(tcell.AttrBold))
	for i, r := range m.requests {
		row := i + 1
		status := "✅"
		if !r.Success {
			status = "❌"
		}
		m.requestList.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("#%03d", r.Index)).SetAlign(tview.AlignLeft))
		m.requestList.SetCell(row, 1, tview.NewTableCell(status).SetAlign(tview.AlignCenter))
		m.requestList.SetCell(row, 2, tview.NewTableCell(r.TotalTime.Truncate(time.Millisecond).String()).SetAlign(tview.AlignRight))
		ttftText := r.TTFT.Truncate(time.Millisecond).String()
		if !r.Success {
			ttftText = "-"
		}
		m.requestList.SetCell(row, 3, tview.NewTableCell(ttftText).SetAlign(tview.AlignRight))
		m.requestList.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%.0f", r.TPS)).SetAlign(tview.AlignRight))
		m.requestList.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%d", r.PromptTokens)).SetAlign(tview.AlignRight))
		m.requestList.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%d", r.CompletionTokens)).SetAlign(tview.AlignRight))
		m.requestList.SetCell(row, 7, tview.NewTableCell(fmt.Sprintf("%.0f%%", r.CacheHitRate)).SetAlign(tview.AlignRight))
	}
	if m.requestList.GetRowCount() > 1 {
		m.requestList.Select(1, 0)
	}
}

// RefreshTasks 清空并重建任务列表（由 startUpdateLoop 调用）。
func (m *MainPage) RefreshTasks(tasks []types.TaskOverview) {
	row, _ := m.taskList.GetSelection()
	m.tasks = tasks
	m.RebuildTaskList()
	if row > 0 && row < m.taskList.GetRowCount() {
		m.taskList.Select(row, 0)
	} else if m.taskList.GetRowCount() > 1 {
		m.taskList.Select(1, 0)
	}
	m.updatePanelStyles()
}

// RefreshRuns 清空并重建历史列表（由 startUpdateLoop 调用）。
func (m *MainPage) RefreshRuns(runs []types.TaskRunSummary) {
	row, _ := m.historyList.GetSelection()
	m.runs = runs
	m.RebuildHistoryList()
	if row > 0 && row < m.historyList.GetRowCount() {
		m.historyList.Select(row, 0)
	} else if m.historyList.GetRowCount() > 1 {
		m.historyList.Select(1, 0)
	}
	m.updatePanelStyles()
}

// RefreshRequests 重建统计和请求列表（由 startUpdateLoop 调用）。
func (m *MainPage) RefreshRequests(reqs []types.RequestMetrics) {
	m.requests = reqs
	m.RebuildStatsAndRequests()
	m.updatePanelStyles()
}
