package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/config"
	"github.com/yinxulai/ait/internal/report"
	"github.com/yinxulai/ait/internal/runner"
	"github.com/yinxulai/ait/internal/task"
	"github.com/yinxulai/ait/internal/turbo"
	"github.com/yinxulai/ait/internal/types"
)

type viewState string

const (
	viewTaskList    viewState = "task-list"
	viewTaskDetail  viewState = "task-detail"
	viewWizard      viewState = "wizard"
	viewDashboard   viewState = "dashboard"
	viewResult      viewState = "result"
	viewTurboResult viewState = "turbo-result"
)

const (
	modeStandard = "standard"
	modeTurbo    = "turbo"

	promptModeText      = "text"
	promptModeFile      = "file"
	promptModeGenerated = "generated"
)

var protocolOptions = []string{
	types.ProtocolOpenAICompletions,
	types.ProtocolOpenAIResponses,
	types.ProtocolAnthropicMessages,
}

var promptModeOptions = []string{promptModeText, promptModeFile, promptModeGenerated}

type fieldKind int

const (
	fieldText fieldKind = iota
	fieldSelect
	fieldToggle
)

type wizardField struct {
	key   string
	label string
	kind  fieldKind
}

type wizardState struct {
	editingTaskID   string
	createdAt       time.Time
	lastRunAt       *time.Time
	lastRunSummary  *types.TaskRunSummary
	fromView        viewState
	step            int // 0=基本信息 1=测试参数 2=确认保存
	fieldIndex      int // active field within current step
	input           textinput.Model
	values          map[string]string
	protocolIndex   int
	mode            string
	promptModeIndex int
	stream          bool
	thinking        bool
	report          bool
}

type Model struct {
	styles       styles
	store        *task.TaskStore
	config       *config.Config
	tasks        []types.TaskDefinition
	history      []types.TaskRunSummary
	selected     int
	view         viewState
	wizard       *wizardState
	width        int
	height       int
	status       string
	err          error
	program        *tea.Program
	runningTask    *types.TaskDefinition
	runningTaskID  string
	runStartedAt   time.Time
	progress     types.StatsData
	runResult    *types.ReportData
	turboResult  *types.TurboResult
	activeRunner *runner.Runner
	activeTurbo  *turbo.Engine
	requestLog   []string
}

func NewModel(store *task.TaskStore, cfg *config.Config) *Model {
	return &Model{
		styles: newStyles(),
		store:  store,
		config: cfg,
		tasks:  store.Tasks,
		view:   viewTaskList,
	}
}

func Run() error {
	store, err := task.LoadTasks()
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	model := NewModel(store, cfg)
	program := tea.NewProgram(model, tea.WithAltScreen())
	model.program = program
	_, err = program.Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case progressMsg:
		m.progress = msg.stats
		return m, nil
	case requestLogMsg:
		m.requestLog = append(m.requestLog, msg.entry)
		if len(m.requestLog) > 60 {
			m.requestLog = m.requestLog[len(m.requestLog)-60:]
		}
		return m, nil
	case runCompleteMsg:
		m.activeRunner = nil
		m.runningTaskID = ""
		m.runResult = msg.result
		if m.view == viewDashboard {
			m.view = viewResult
		}
		m.status = fmt.Sprintf("标准模式完成，共 %d 请求", msg.result.TotalRequests)
		m.persistStandardRun(msg.taskID, msg.result, msg.reportPaths)
		return m, nil
	case turboCompleteMsg:
		m.activeTurbo = nil
		m.runningTaskID = ""
		m.turboResult = msg.result
		if m.view == viewDashboard {
			m.view = viewTurboResult
		}
		m.status = fmt.Sprintf("Turbo 完成，最大稳定并发 %d", msg.result.MaxStableConcurrency)
		m.persistTurboRun(msg.taskID, msg.result)
		return m, nil
	case asyncErrorMsg:
		m.runningTaskID = ""
		m.err = msg.err
		m.status = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.view {
	case viewTaskList:
		return m.handleTaskListKey(msg)
	case viewTaskDetail:
		return m.handleTaskDetailKey(msg)
	case viewWizard:
		return m.handleWizardKey(msg)
	case viewDashboard:
		return m.handleDashboardKey(msg)
	case viewResult, viewTurboResult:
		if msg.String() == "b" || msg.String() == "esc" || msg.String() == "enter" {
			m.reloadHistoryForSelectedTask()
			m.view = viewTaskDetail
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) handleTaskListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(m.tasks)-1 {
			m.selected++
		}
	case "a":
		m.openWizard(nil)
	case "e":
		if taskDef, ok := m.currentTask(); ok {
			copyTask := taskDef
			m.openWizard(&copyTask)
		}
	case "y":
		if taskDef, ok := m.currentTask(); ok {
			copyTask := taskDef
			copyTask.ID = ""
			copyTask.Name = taskDef.Name + "-copy"
			m.openWizard(&copyTask)
		}
	case "d":
		if taskDef, ok := m.currentTask(); ok {
			if err := m.store.Delete(taskDef.ID); err != nil {
				m.err = err
				break
			}
			if err := m.store.Save(); err != nil {
				m.err = err
				break
			}
			m.tasks = m.store.Tasks
			if m.selected >= len(m.tasks) && m.selected > 0 {
				m.selected--
			}
			m.status = "任务已删除"
		}
	case "enter":
		if taskDef, ok := m.currentTask(); ok {
			if taskDef.ID == m.runningTaskID {
				m.view = viewDashboard
			} else {
				m.reloadHistoryForSelectedTask()
				m.view = viewTaskDetail
			}
		}
	case "r":
		if taskDef, ok := m.currentTask(); ok {
			if m.runningTaskID != "" {
				m.status = "已有任务正在运行中，请等待完成或进入仪表盘停止"
			} else {
				m.startTaskRun(taskDef)
			}
		}
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) handleTaskDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	taskDef, ok := m.currentTask()
	if !ok {
		m.view = viewTaskList
		return m, nil
	}

	switch msg.String() {
	case "b", "esc":
		m.view = viewTaskList
	case "e":
		copyTask := taskDef
		m.openWizard(&copyTask)
	case "d":
		if err := m.store.Delete(taskDef.ID); err != nil {
			m.err = err
			break
		}
		if err := m.store.Save(); err != nil {
			m.err = err
			break
		}
		m.tasks = m.store.Tasks
		if m.selected >= len(m.tasks) && m.selected > 0 {
			m.selected--
		}
		m.view = viewTaskList
	case "enter", "r":
		if m.runningTaskID != "" && m.runningTaskID != taskDef.ID {
			m.status = "已有任务正在运行中"
		} else {
			m.startTaskRun(taskDef)
			if m.runningTaskID == taskDef.ID {
				m.view = viewDashboard
			}
		}
	}

	return m, nil
}

func (m *Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.wizard == nil {
		return m, nil
	}

	// Step 2 (confirm): only action keys, no text input
	if m.wizard.step == 2 {
		switch msg.String() {
		case "esc":
			m.wizard.step = 1
			m.wizard.fieldIndex = len(m.wizardStepFields(1)) - 1
			m.refreshWizardInput()
		case "enter":
			if err := m.saveWizard(); err != nil {
				m.err = err
				m.status = err.Error()
			}
		case "r":
			if err := m.saveWizard(); err != nil {
				m.err = err
				m.status = err.Error()
				return m, nil
			}
			if taskDef, ok := m.currentTask(); ok {
				m.startTaskRun(taskDef)
				if m.runningTaskID == taskDef.ID {
					m.view = viewDashboard
				}
			}
		}
		return m, nil
	}

	fields := m.wizardStepFields(m.wizard.step)
	field := fields[m.wizard.fieldIndex]

	switch msg.String() {
	case "esc":
		if m.wizard.step > 0 {
			m.wizard.step--
			m.wizard.fieldIndex = len(m.wizardStepFields(m.wizard.step)) - 1
			m.refreshWizardInput()
		} else {
			m.view = m.wizard.fromView
			m.wizard = nil
		}
		return m, nil
	case "tab", "down", "j":
		if field.kind == fieldText {
			m.wizard.values[field.key] = m.wizard.input.Value()
		}
		m.advanceWizardField(1)
		return m, nil
	case "enter":
		if field.kind == fieldText {
			m.wizard.values[field.key] = m.wizard.input.Value()
		}
		if m.wizard.fieldIndex == len(fields)-1 {
			m.wizard.step++
			m.wizard.fieldIndex = 0
			if m.wizard.step < 2 {
				m.refreshWizardInput()
			}
		} else {
			m.wizard.fieldIndex++
			m.refreshWizardInput()
		}
		return m, nil
	case "shift+tab", "up", "k":
		if field.kind == fieldText {
			m.wizard.values[field.key] = m.wizard.input.Value()
		}
		m.advanceWizardField(-1)
		return m, nil
	case "left", "h":
		m.cycleWizardField(-1)
		return m, nil
	case "right", "l", "space":
		m.cycleWizardField(1)
		return m, nil
	}
	if field.kind == fieldText {
		var cmd tea.Cmd
		m.wizard.input, cmd = m.wizard.input.Update(msg)
		m.wizard.values[field.key] = m.wizard.input.Value()
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		if m.activeRunner != nil {
			m.activeRunner.Stop()
		}
		if m.activeTurbo != nil {
			m.activeTurbo.Stop()
		}
	case "b", "esc":
		// 返回列表，任务继续在后台运行
		m.view = viewTaskList
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) View() string {
	switch m.view {
	case viewTaskDetail:
		return m.renderTaskDetail()
	case viewWizard:
		return m.renderWizard()
	case viewDashboard:
		return m.renderDashboard()
	case viewResult:
		return m.renderResult()
	case viewTurboResult:
		return m.renderTurboResult()
	default:
		return m.renderTaskList()
	}
}

func (m *Model) renderTaskList() string {
	if m.width == 0 {
		return "加载中..."
	}
	lastRunStr := ""
	for _, t := range m.tasks {
		if t.LastRunAt != nil {
			lastRunStr = "最近: " + timeAgo(*t.LastRunAt)
			break
		}
	}
	header := m.renderHeader(
		"AIT  任务中心",
		fmt.Sprintf("已保存任务: %d  %s", len(m.tasks), lastRunStr),
	)
	footer := m.renderFooter(
		"[↑↓] 选择", "[Enter] 详情", "[a] 新建", "[r] 运行",
		"[e] 编辑", "[d] 删除", "[y] 复制", "[q] 退出",
	)
	contentH := m.height - 2
	if contentH < 4 {
		contentH = 4
	}
	panelH := contentH - 2
	leftW := (m.width - 4) * 57 / 100
	rightW := m.width - 4 - leftW
	leftContent := m.buildTaskListLeft(panelH, leftW)
	rightContent := m.buildTaskListRight(panelH)
	mid := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, panelH)
	return lipgloss.JoinVertical(lipgloss.Left, header, mid, footer)
}

func (m *Model) buildTaskListLeft(maxH, width int) string {
	var lines []string
	lines = append(lines, m.styles.tableHead.Render(
		fmt.Sprintf("  %-28s %-9s %-14s %s", "任务名称", "模式", "协议", "上次结果"),
	))
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", width)))
	if len(m.tasks) == 0 {
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render("  暂无任务  按 [a] 新建"))
		return strings.Join(lines, "\n")
	}
	for i, t := range m.tasks {
		if len(lines) >= maxH-1 {
			break
		}
		// Mode: color-coded tag text with manual padding to 9 visual columns
		var modeRendered string
		if t.Input.Turbo {
			modeRendered = m.styles.tagTurbo.Render("Turbo")
		} else {
			modeRendered = m.styles.tagStd.Render("标准")
		}
		modePad := 9 - lipgloss.Width(modeRendered)
		if modePad < 0 {
			modePad = 0
		}
		modeCol := modeRendered + strings.Repeat(" ", modePad)
		proto := shortProtocol(t.Input.NormalizedProtocol())
		lastResult := m.styles.muted.Render("从未运行")
		if t.LastRunSummary != nil {
			pct := t.LastRunSummary.SuccessRate
			if pct >= 99 {
				lastResult = m.styles.ok.Render(fmt.Sprintf("%.1f%%", pct))
			} else if pct >= 90 {
				lastResult = m.styles.metricVal.Render(fmt.Sprintf("%.1f%%", pct))
			} else {
				lastResult = m.styles.errStyle.Render(fmt.Sprintf("%.1f%%", pct))
			}
		}
		nameStr := truncate(t.Name, 28)
		// Build row from parts so ANSI in modeCol doesn't break alignment
		nameCol := fmt.Sprintf("%-28s ", nameStr)
		protoCol := fmt.Sprintf("%-14s ", proto)
		mainRow := "  " + nameCol + modeCol + " " + protoCol + lastResult
		if i == m.selected {
			plainRow := "  " + nameCol + fmt.Sprintf("%-9s ", func() string {
				if t.Input.Turbo {
					return "Turbo"
				}
				return "标准"
			}()) + protoCol + lastResult
			lines = append(lines, m.styles.tableRowSel.Width(width).Render("▶"+plainRow[1:]))
		} else {
			lines = append(lines, mainRow)
		}
		var sub string
		if t.Input.Turbo {
			tc := t.Input.TurboConfig
			sub = fmt.Sprintf("     %s  %d→%d +%d 每级%d",
				truncate(t.Input.Model, 18),
				tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize, tc.LevelRequests)
		} else {
			sub = fmt.Sprintf("     %s  并发%d/请求%d",
				truncate(t.Input.Model, 20), t.Input.Concurrency, t.Input.Count)
		}
		if i == m.selected {
			lines = append(lines, m.styles.tableRowSel.Width(width).Render(sub))
		} else {
			lines = append(lines, m.styles.muted.Render(sub))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) buildTaskListRight(maxH int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("快捷操作"))
	lines = append(lines, "")
	lines = append(lines, " "+m.styles.key.Render("[a]")+"  新建任务")
	lines = append(lines, " "+m.styles.key.Render("[Enter]")+"  查看详情")
	lines = append(lines, " "+m.styles.key.Render("[r]")+"  直接运行选中任务")
	lines = append(lines, " "+m.styles.key.Render("[e]")+"  编辑  "+m.styles.key.Render("[d]")+"  删除  "+m.styles.key.Render("[y]")+"  复制")
	lines = append(lines, "")
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", 28)))
	lines = append(lines, "")
	lines = append(lines, m.styles.sectionHead.Render("最近执行"))
	lines = append(lines, "")
	count := 0
	for _, t := range m.tasks {
		if t.LastRunSummary == nil {
			continue
		}
		s := t.LastRunSummary
		statusIcon := m.styles.ok.Render("✓")
		if s.SuccessRate < 90 {
			statusIcon = m.styles.errStyle.Render("✗")
		}
		lines = append(lines, fmt.Sprintf(" %s %-16s %.1f%%  %.0f tok/s",
			statusIcon, truncate(t.Name, 16), s.SuccessRate, s.AvgTPS))
		count++
		if count >= 5 || len(lines) >= maxH-2 {
			break
		}
	}
	if count == 0 {
		lines = append(lines, m.styles.muted.Render("  暂无记录"))
	}
	if m.status != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render(m.status))
	}
	if m.err != nil {
		lines = append(lines, m.styles.errStyle.Render("错误: "+m.err.Error()))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderTaskDetail() string {
	if m.width == 0 {
		return "加载中..."
	}
	taskDef, ok := m.currentTask()
	if !ok {
		return m.styles.errStyle.Render("任务不存在")
	}
	updatedStr := ""
	if !taskDef.UpdatedAt.IsZero() {
		updatedStr = "更新: " + taskDef.UpdatedAt.Format("01-02 15:04")
	}
	lastRunStr := "从未运行"
	if taskDef.LastRunAt != nil {
		lastRunStr = "上次: " + timeAgo(*taskDef.LastRunAt)
	}
	header := m.renderHeader(
		"AIT  任务详情 — "+truncate(taskDef.Name, 24),
		updatedStr+"   "+lastRunStr,
	)
	footer := m.renderFooter("[Enter/r] 运行", "[e] 编辑", "[d] 删除", "[b] 返回")
	contentH := m.height - 2
	histH := 9
	topH := contentH - histH
	if topH < 6 {
		topH = 6
	}
	panelH := topH - 2
	leftW := (m.width - 4) * 57 / 100
	rightW := m.width - 4 - leftW
	leftContent := m.buildDetailLeft(taskDef, panelH, leftW)
	rightContent := m.buildDetailRight(taskDef)
	top := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, panelH)
	histPanelH := histH - 2
	histContent := m.buildHistoryContent(histPanelH, m.width-4)
	histPanel := lipgloss.NewStyle().
		Width(m.width - 2).Height(histPanelH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Render(histContent)
	return lipgloss.JoinVertical(lipgloss.Left, header, top, histPanel, footer)
}

func (m *Model) buildDetailLeft(t types.TaskDefinition, h, w int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("配置摘要"))
	lines = append(lines, "")
	maxURLLen := w - 14
	if maxURLLen < 20 {
		maxURLLen = 20
	}
	rows := [][2]string{
		{"协议", t.Input.NormalizedProtocol()},
		{"接口地址", truncate(t.Input.ResolvedEndpointURL(), maxURLLen)},
		{"模型", t.Input.Model},
	}
	if t.Input.Turbo {
		tc := t.Input.TurboConfig
		rows = append(rows,
			[2]string{"模式", "Turbo 模式"},
			[2]string{"爬坡", fmt.Sprintf("%d → %d  步进+%d  每级%d",
				tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize, tc.LevelRequests)},
			[2]string{"停止条件", fmt.Sprintf("成功率<%.0f%%  或延迟>%s",
				tc.MinSuccessRate*100, tc.MaxLatency)},
		)
	} else {
		rows = append(rows,
			[2]string{"模式", "标准模式"},
			[2]string{"并发", fmt.Sprintf("%d", t.Input.Concurrency)},
			[2]string{"请求数", fmt.Sprintf("%d", t.Input.Count)},
			[2]string{"超时", t.Input.Timeout.String()},
		)
	}
	rows = append(rows,
		[2]string{"流式", boolLabel(t.Input.Stream)},
		[2]string{"Prompt", promptSummary(t.Input)},
	)
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			m.styles.label.Render(fmt.Sprintf("%-8s", row[0])),
			m.styles.value.Render(row[1])))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) buildDetailRight(t types.TaskDefinition) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("最近一次结果"))
	lines = append(lines, "")
	if t.LastRunSummary == nil {
		lines = append(lines, m.styles.muted.Render("  从未运行"))
		return strings.Join(lines, "\n")
	}
	s := t.LastRunSummary
	statusStr := m.styles.ok.Render("✓ 完成")
	if s.SuccessRate < 90 {
		statusStr = m.styles.errStyle.Render("✗ 异常")
	}
	rows := [][2]string{
		{"状态", statusStr},
		{"成功率", fmt.Sprintf("%.1f%%", s.SuccessRate)},
		{"avg TTFT", s.AvgTTFT.Truncate(time.Millisecond).String()},
		{"avg TPS", fmt.Sprintf("%.1f tok/s", s.AvgTPS)},
		{"缓存命中", fmt.Sprintf("%.1f%%", s.CacheHitRate)},
	}
	if s.MaxStableConcurrency > 0 {
		rows = append(rows, [2]string{"最大稳定并发", fmt.Sprintf("%d", s.MaxStableConcurrency)})
	}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			m.styles.label.Render(fmt.Sprintf("%-10s", row[0])),
			row[1]))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) buildHistoryContent(maxH, width int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("最近运行记录")+"  "+
		m.styles.tableHead.Render(fmt.Sprintf("%-19s %-6s %-8s %-12s %-10s %-8s",
			"时间", "模式", "成功率", "TTFT", "TPS", "Cache")))
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", width-2)))
	if len(m.history) == 0 {
		lines = append(lines, m.styles.muted.Render("  暂无历史记录"))
		return strings.Join(lines, "\n")
	}
	for _, run := range m.history {
		if len(lines) >= maxH {
			break
		}
		status := m.styles.ok.Render("✓")
		if run.SuccessRate < 90 {
			status = m.styles.errStyle.Render("✗")
		}
		modeShort := run.Mode
		if len(modeShort) > 5 {
			modeShort = modeShort[:5]
		}
		lines = append(lines, fmt.Sprintf("  %s  %-19s %-6s %-8.1f%% %-12s %-10.1f %-8.1f%%",
			status,
			run.FinishedAt.Format("2006-01-02 15:04"),
			modeShort,
			run.SuccessRate,
			run.AvgTTFT.Truncate(time.Millisecond),
			run.AvgTPS,
			run.CacheHitRate))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderWizard() string {
	if m.width == 0 || m.wizard == nil {
		return "加载中..."
	}
	stepTitles := []string{"1/3 · 基本信息", "2/3 · 测试参数", "3/3 · 确认保存"}
	header := m.renderHeader("AIT  任务向导", "步骤 "+stepTitles[m.wizard.step])
	var footer string
	if m.wizard.step < 2 {
		footer = m.renderFooter("[Tab/↓] 下一项", "[↑] 上一项", "[←→] 切换选项", "[Enter] 下一步", "[Esc] 返回")
	} else {
		footer = m.renderFooter("[Enter] 保存任务", "[r] 保存并运行", "[Esc] 返回修改")
	}
	contentH := m.height - 2
	dialogW := m.width - 6
	if dialogW > 78 {
		dialogW = 78
	}
	if dialogW < 40 {
		dialogW = 40
	}
	dialogContentW := dialogW - 6 // -2 border -4 padding
	var content string
	switch m.wizard.step {
	case 0:
		content = m.renderWizardStep0(dialogContentW)
	case 1:
		content = m.renderWizardStep1(dialogContentW)
	case 2:
		content = m.renderWizardStep2(dialogContentW)
	}
	dialog := m.styles.dialog.Width(dialogContentW).Render(content)
	dialogH := lipgloss.Height(dialog)
	padTop := (contentH - dialogH) / 2
	if padTop < 0 {
		padTop = 0
	}
	centeredDialog := lipgloss.Place(m.width, contentH,
		lipgloss.Center, lipgloss.Top,
		strings.Repeat("\n", padTop)+dialog)
	return lipgloss.JoinVertical(lipgloss.Left, header, centeredDialog, footer)
}

func (m *Model) renderWizardStep0(w int) string {
	fields := m.wizardStepFields(0)
	var lines []string
	// Step indicator: ● ○ ○
	lines = append(lines, m.styles.stepActive.Render("●")+" "+
		m.styles.stepTodo.Render("○")+" "+
		m.styles.stepTodo.Render("○")+"  "+
		m.styles.sectionHead.Render("基本信息"))
	lines = append(lines, "")
	for i, field := range fields {
		active := i == m.wizard.fieldIndex
		lines = append(lines, m.renderWizardField(field, active))
		if field.key == "protocol" {
			for pi, p := range protocolOptions {
				bullet := "  ○ "
				if pi == m.wizard.protocolIndex {
					bullet = "  " + m.styles.ok.Render("●") + " "
				}
				lines = append(lines, "                  "+bullet+p)
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderWizardStep1(w int) string {
	fields := m.wizardStepFields(1)
	var lines []string
	// Step indicator: ✓ ● ○
	lines = append(lines, m.styles.stepDone.Render("✓")+" "+
		m.styles.stepActive.Render("●")+" "+
		m.styles.stepTodo.Render("○")+"  "+
		m.styles.sectionHead.Render("测试参数"))
	lines = append(lines, "")
	for i, field := range fields {
		active := i == m.wizard.fieldIndex
		lines = append(lines, m.renderWizardField(field, active))
		if field.key == "mode" {
			opts := []string{modeStandard, modeTurbo}
			labels := []string{"标准模式", "Turbo 模式"}
			for oi, opt := range opts {
				bullet := "  ○ "
				if opt == m.wizard.mode {
					bullet = "  " + m.styles.ok.Render("●") + " "
				}
				lines = append(lines, "                  "+bullet+labels[oi])
			}
		}
		if field.key == "prompt_mode" {
			pmLabels := []string{"直接输入", "文件路径", "按长度生成"}
			for pi, pl := range pmLabels {
				bullet := "  ○ "
				if pi == m.wizard.promptModeIndex {
					bullet = "  " + m.styles.ok.Render("●") + " "
				}
				lines = append(lines, "                  "+bullet+pl)
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderWizardStep2(w int) string {
	var lines []string
	// Step indicator: ✓ ✓ ●
	lines = append(lines, m.styles.stepDone.Render("✓")+" "+
		m.styles.stepDone.Render("✓")+" "+
		m.styles.stepActive.Render("●")+"  "+
		m.styles.sectionHead.Render("确认保存"))
	lines = append(lines, "")
	d, err := buildTaskDefinition(m.wizard)
	if err != nil {
		lines = append(lines, m.styles.errStyle.Render("配置有误: "+err.Error()))
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render("按 [Esc] 返回修改"))
		return strings.Join(lines, "\n")
	}
	rows := [][2]string{
		{"任务名称", d.Name},
		{"协议", d.Input.NormalizedProtocol()},
		{"接口地址", truncate(d.Input.ResolvedEndpointURL(), w-16)},
		{"API 密钥", maskAPIKey(d.Input.ApiKey)},
		{"测试模型", d.Input.Model},
	}
	if d.Input.Turbo {
		tc := d.Input.TurboConfig
		rows = append(rows,
			[2]string{"测试模式", "Turbo 模式"},
			[2]string{"并发爬坡", fmt.Sprintf("%d → %d  步进+%d  每级%d",
				tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize, tc.LevelRequests)},
			[2]string{"停止条件", fmt.Sprintf("成功率<%.0f%%  或延迟>%s",
				tc.MinSuccessRate*100, tc.MaxLatency)},
		)
	} else {
		rows = append(rows,
			[2]string{"测试模式", "标准模式"},
			[2]string{"并发/请求", fmt.Sprintf("%d / %d", d.Input.Concurrency, d.Input.Count)},
			[2]string{"超时", d.Input.Timeout.String()},
		)
	}
	rows = append(rows,
		[2]string{"流式", boolLabel(d.Input.Stream)},
		[2]string{"Prompt", promptSummary(d.Input)},
	)
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			m.styles.label.Render(fmt.Sprintf("%-10s", row[0])),
			row[1]))
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.ok.Render("  ▶ 按 [Enter] 保存，[r] 保存并立即运行"))
	return strings.Join(lines, "\n")
}

func (m *Model) renderWizardField(field wizardField, active bool) string {
	var val string
	if field.kind == fieldText && active {
		val = m.wizard.input.View()
	} else {
		val = m.displayWizardValue(field)
	}
	labelStr := fmt.Sprintf("%-12s", field.label)
	if active {
		return m.styles.cursor.Render("▶") + " " +
			m.styles.fieldActive.Render(labelStr) + "  " + val
	}
	return "  " + m.styles.fieldIdle.Render(labelStr) + "  " + m.styles.muted.Render(val)
}

func (m *Model) renderDashboard() string {
	if m.width == 0 {
		return "加载中..."
	}
	taskName, protocol, modelName := "", "", ""
	isTurbo := false
	totalReqs, concurrency := 0, 0
	if m.runningTask != nil {
		taskName = m.runningTask.Name
		protocol = shortProtocol(m.runningTask.Input.NormalizedProtocol())
		modelName = m.runningTask.Input.Model
		isTurbo = m.runningTask.Input.Turbo
		totalReqs = m.runningTask.Input.Count
		concurrency = m.runningTask.Input.Concurrency
		if isTurbo {
			totalReqs = m.runningTask.Input.TurboConfig.LevelRequests
			concurrency = m.runningTask.Input.TurboConfig.InitConcurrency
		}
	}
	title := "AIT  正在测试 — " + modelName
	if isTurbo {
		title = "AIT  Turbo 探测 — " + modelName
	}
	header := m.renderHeader(title,
		fmt.Sprintf("任务: %s  协议: %s", truncate(taskName, 20), protocol))
	footer := m.renderFooter("[s] 停止", "[q] 退出")
	contentH := m.height - 2
	logH := 7
	topH := contentH - logH
	if topH < 6 {
		topH = 6
	}
	panelH := topH - 2
	leftW := (m.width - 4) * 50 / 100
	rightW := m.width - 4 - leftW
	var leftContent, rightContent string
	if isTurbo {
		leftContent = m.buildTurboDashLeft(panelH)
		rightContent = m.buildTurboDashRight(panelH)
	} else {
		leftContent = m.buildStdDashLeft(panelH, totalReqs, concurrency)
		rightContent = m.buildStdDashRight(panelH)
	}
	top := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, panelH)
	logPanelH := logH - 2
	logContent := m.buildLogPanel(logPanelH, m.width-4)
	logPanel := lipgloss.NewStyle().
		Width(m.width - 2).Height(logPanelH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Render(logContent)
	return lipgloss.JoinVertical(lipgloss.Left, header, top, logPanel, footer)
}

func (m *Model) buildStdDashLeft(h, total, concurrency int) string {
	p := m.progress
	completed := p.CompletedCount
	failed := p.FailedCount
	elapsed := time.Duration(0)
	if !p.StartTime.IsZero() {
		elapsed = time.Since(p.StartTime)
	}
	var estRemaining string
	if completed > 0 && total > completed && elapsed > 0 {
		rate := float64(completed) / elapsed.Seconds()
		remaining := float64(total-completed) / rate
		estRemaining = "~" + time.Duration(remaining*float64(time.Second)).Truncate(time.Second).String()
	}
	barW := 20
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("进度"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s  %s  %d",
		m.styles.label.Render("完成"), progressBar(completed, total, barW), completed))
	lines = append(lines, fmt.Sprintf("  %s  %s  %d",
		m.styles.errStyle.Render("失败"), progressBarRed(failed, total, barW), failed))
	lines = append(lines, fmt.Sprintf("  %s  %s  %d",
		m.styles.muted.Render("总计"), progressBar(total, total, barW), total))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %-10s %s",
		m.styles.label.Render("已用时"),
		elapsed.Truncate(100*time.Millisecond)))
	if estRemaining != "" {
		lines = append(lines, fmt.Sprintf("  %-10s %s",
			m.styles.label.Render("预计剩余"),
			estRemaining))
	}
	lines = append(lines, fmt.Sprintf("  %-10s %d 活跃",
		m.styles.label.Render("并发槽"),
		concurrency))
	return strings.Join(lines, "\n")
}

func (m *Model) buildStdDashRight(h int) string {
	p := m.progress
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("实时指标"))
	lines = append(lines, "")
	successRate := 0.0
	if p.CompletedCount > 0 {
		successRate = float64(p.CompletedCount-p.FailedCount) / float64(p.CompletedCount) * 100
	}
	srBar := progressBar(int(successRate), 100, 16)
	lines = append(lines, fmt.Sprintf("  成功率  %s  %.1f%%", srBar, successRate))
	lines = append(lines, "")
	avgTPS := 0.0
	if len(p.OutputTokenCounts) > 0 && len(p.TotalTimes) > 0 {
		totalTokens := 0
		for _, tok := range p.OutputTokenCounts {
			totalTokens += tok
		}
		totalTimeS := 0.0
		for _, d := range p.TotalTimes {
			totalTimeS += d.Seconds()
		}
		if totalTimeS > 0 {
			avgTPS = float64(totalTokens) / totalTimeS
		}
	}
	avgTTFT := time.Duration(0)
	if len(p.TTFTs) > 0 {
		sum := time.Duration(0)
		for _, d := range p.TTFTs {
			sum += d
		}
		avgTTFT = sum / time.Duration(len(p.TTFTs))
	}
	avgTotal := time.Duration(0)
	if len(p.TotalTimes) > 0 {
		sum := time.Duration(0)
		for _, d := range p.TotalTimes {
			sum += d
		}
		avgTotal = sum / time.Duration(len(p.TotalTimes))
	}
	avgCache := 0.0
	if len(p.CacheHitRates) > 0 {
		sum := 0.0
		for _, r := range p.CacheHitRates {
			sum += r
		}
		avgCache = sum / float64(len(p.CacheHitRates)) * 100
	}
	rows := [][2]string{
		{"avg TPS", fmt.Sprintf("%.1f tok/s", avgTPS)},
		{"avg TTFT", avgTTFT.Truncate(time.Millisecond).String()},
		{"缓存命中率", fmt.Sprintf("%.1f%%", avgCache)},
		{"avg 总耗时", avgTotal.Truncate(time.Millisecond).String()},
	}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %-12s %s",
			m.styles.label.Render(row[0]),
			m.styles.metricVal.Render(row[1])))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) buildTurboDashLeft(h int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("Turbo 探测中"))
	lines = append(lines, "")
	elapsed := time.Since(m.runStartedAt)
	lines = append(lines, fmt.Sprintf("  %s  %s",
		m.styles.label.Render("已用时"),
		elapsed.Truncate(time.Second)))
	lines = append(lines, "")
	lines = append(lines, m.styles.muted.Render("  正在逐级探测最大稳定并发..."))
	lines = append(lines, m.styles.muted.Render("  完成后将自动显示结果"))
	return strings.Join(lines, "\n")
}

func (m *Model) buildTurboDashRight(h int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("探测状态"))
	lines = append(lines, "")
	lines = append(lines, "  "+m.styles.ok.Render("●")+"  测试运行中")
	lines = append(lines, m.styles.muted.Render("  等待完成..."))
	return strings.Join(lines, "\n")
}

func (m *Model) buildLogPanel(maxH, width int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("请求日志"))
	if len(m.requestLog) == 0 {
		lines = append(lines, m.styles.muted.Render("  等待请求完成..."))
		return strings.Join(lines, "\n")
	}
	start := 0
	if len(m.requestLog) > maxH-1 {
		start = len(m.requestLog) - (maxH - 1)
	}
	for _, entry := range m.requestLog[start:] {
		// Color log entries based on their leading status marker
		if strings.HasPrefix(entry, "✓") || strings.HasPrefix(entry, "✔") {
			lines = append(lines, "  "+m.styles.logOk.Render(entry))
		} else if strings.HasPrefix(entry, "✗") || strings.HasPrefix(entry, "✘") || strings.HasPrefix(entry, "ERR") {
			lines = append(lines, "  "+m.styles.logErr.Render(entry))
		} else {
			lines = append(lines, "  "+m.styles.muted.Render(entry))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderResult() string {
	if m.width == 0 {
		return "加载中..."
	}
	header := m.renderHeader("AIT  测试完成", "标准模式结果")
	footer := m.renderFooter("[b/Esc] 返回详情")
	if m.runResult == nil {
		return lipgloss.JoinVertical(lipgloss.Left, header,
			m.styles.errStyle.Render("结果为空"), footer)
	}
	r := m.runResult
	panelW := m.width - 4
	panelH := m.height - 4
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render(fmt.Sprintf("任务完成 — %s", r.Model)))
	lines = append(lines, "")
	rows := [][2]string{
		{"协议", r.Protocol},
		{"接口地址", truncate(r.EndpointURL, panelW-16)},
		{"模型", r.Model},
		{"成功率", fmt.Sprintf("%.1f%%", r.SuccessRate)},
		{"总请求数", fmt.Sprintf("%d", r.TotalRequests)},
		{"avg TTFT", r.AvgTTFT.Truncate(time.Millisecond).String()},
		{"avg TPS", fmt.Sprintf("%.2f tok/s", r.AvgTPS)},
		{"缓存命中率", fmt.Sprintf("%.1f%%", r.AvgCacheHitRate*100)},
		{"avg 总耗时", r.AvgTotalTime.Truncate(time.Millisecond).String()},
		{"总测试时长", r.TotalTime.Truncate(time.Second).String()},
	}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %s  %s",
			m.styles.label.Render(fmt.Sprintf("%-12s", row[0])),
			m.styles.value.Render(row[1])))
	}
	content := strings.Join(lines, "\n")
	panel := lipgloss.NewStyle().
		Width(panelW).Height(panelH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, panel, footer)
}

func (m *Model) renderTurboResult() string {
	if m.width == 0 {
		return "加载中..."
	}
	header := m.renderHeader("AIT  Turbo 完成", "Turbo 模式结果")
	footer := m.renderFooter("[b/Esc] 返回详情")
	if m.turboResult == nil {
		return lipgloss.JoinVertical(lipgloss.Left, header,
			m.styles.errStyle.Render("Turbo 结果为空"), footer)
	}
	r := m.turboResult
	panelW := m.width - 4
	panelH := m.height - 4
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render(fmt.Sprintf(
		"Turbo 完成 — %s  最大稳定并发: %d  峰值 TPS: %.1f",
		r.Model, r.MaxStableConcurrency, r.PeakTPS)))
	lines = append(lines, "")
	lines = append(lines, m.styles.tableHead.Render(fmt.Sprintf(
		"  %-6s %-8s %-10s %-10s %-8s %-8s %s",
		"并发", "成功率", "TPS", "TTFT", "Cache", "总耗时", "状态")))
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", panelW-4)))
	for _, level := range r.Levels {
		status := m.styles.ok.Render("✓ 稳定")
		if !level.Stable {
			status = m.styles.errStyle.Render("✗ 不稳定")
		}
		marker := "  "
		if level.Concurrency == r.MaxStableConcurrency {
			marker = m.styles.cursor.Render("▶ ")
		}
		lines = append(lines, fmt.Sprintf("%s%-6d %-8.1f%% %-10.1f %-10s %-8.1f%% %-8s %s",
			marker,
			level.Concurrency,
			level.SuccessRate*100,
			level.AvgTPS,
			level.AvgTTFT.Truncate(time.Millisecond),
			level.CacheHitRate*100,
			level.AvgTotalTime.Truncate(time.Millisecond),
			status))
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.muted.Render("  停止原因: "+r.StopReason))
	content := strings.Join(lines, "\n")
	panel := lipgloss.NewStyle().
		Width(panelW).Height(panelH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurple).
		Render(content)
	return lipgloss.JoinVertical(lipgloss.Left, header, panel, footer)
}

func (m *Model) renderHeader(left, right string) string {
	if m.width == 0 {
		return ""
	}
	// Each part gets the same header background so the bar spans the full width.
	leftStyled := lipgloss.NewStyle().
		Background(colorHeaderBg).Bold(true).Foreground(colorPink).
		Render(" ◆ " + left)
	rightStyled := lipgloss.NewStyle().
		Background(colorHeaderBg).Foreground(colorHeaderFg).
		Render(right + " ")
	lw := lipgloss.Width(leftStyled)
	rw := lipgloss.Width(rightStyled)
	gap := m.width - lw - rw
	if gap < 0 {
		gap = 0
	}
	spacer := lipgloss.NewStyle().Background(colorHeaderBg).Render(strings.Repeat(" ", gap))
	return leftStyled + spacer + rightStyled
}

func (m *Model) renderFooter(hints ...string) string {
	if m.width == 0 {
		return ""
	}
	// Left: colored AIT brand badge
	leftBadge := lipgloss.NewStyle().
		Background(colorPurple).Foreground(colorWhite).Bold(true).
		Render(" ◆ AIT ")
	// Right: dim version badge
	rightBadge := lipgloss.NewStyle().
		Background(colorHeaderBg).Foreground(colorHeaderFg).
		Render(" v0.1 ")
	// Middle: key hints in pink on footer bg
	var parts []string
	for _, h := range hints {
		parts = append(parts, lipgloss.NewStyle().Foreground(colorPink).Render(h))
	}
	hintsStr := "  " + strings.Join(parts, "  ")
	lw := lipgloss.Width(leftBadge)
	rw := lipgloss.Width(rightBadge)
	hw := lipgloss.Width(hintsStr)
	gap := m.width - lw - rw - hw
	if gap < 0 {
		gap = 0
	}
	middle := lipgloss.NewStyle().
		Background(colorFooterBg).Foreground(colorMuted).
		Render(hintsStr + strings.Repeat(" ", gap))
	return leftBadge + middle + rightBadge
}

func (m *Model) dualColumnLayout(leftContent, rightContent string, leftW, rightW, h int) string {
	bc := colorPurple
	leftPane := lipgloss.NewStyle().
		Width(leftW).Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bc).
		Render(leftContent)
	rightPane := lipgloss.NewStyle().
		Width(rightW).Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bc).
		Render(rightContent)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func progressBar(current, total, width int) string {
	if total <= 0 || width <= 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width))
	}
	filled := current * width / total
	if filled > width {
		filled = width
	}
	bar := lipgloss.NewStyle().Foreground(colorGreen).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width-filled))
	return bar + empty
}

// progressBarRed renders a red-tinted progress bar for failure/error metrics.
func progressBarRed(current, total, width int) string {
	if total <= 0 || width <= 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width))
	}
	filled := current * width / total
	if filled > width {
		filled = width
	}
	bar := lipgloss.NewStyle().Foreground(colorRed).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width-filled))
	return bar + empty
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds 前", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm 前", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh 前", int(d.Hours()))
	}
	return t.Format("01-02 15:04")
}

func shortProtocol(p string) string {
	p = strings.ReplaceAll(p, "openai-", "")
	p = strings.ReplaceAll(p, "anthropic-", "")
	return p
}

func maskAPIKey(key string) string {
	if len(key) == 0 {
		return "(空)"
	}
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:4] + strings.Repeat("•", len(key)-8) + key[len(key)-4:]
}

func (m *Model) currentTask() (types.TaskDefinition, bool) {
	if len(m.tasks) == 0 || m.selected < 0 || m.selected >= len(m.tasks) {
		return types.TaskDefinition{}, false
	}
	return m.tasks[m.selected], true
}

func (m *Model) openWizard(existing *types.TaskDefinition) {
	state := newWizardState(existing, m.view, m.config)
	m.wizard = state
	m.view = viewWizard
	m.refreshWizardInput()
}

func newWizardState(existing *types.TaskDefinition, from viewState, cfg *config.Config) *wizardState {
	input := textinput.New()
	input.Width = 72
	input.Prompt = ""
	values := map[string]string{
		"name":                 "",
		"endpoint":             "",
		"apiKey":               "",
		"model":                "",
		"concurrency":          "5",
		"count":                "100",
		"timeout":              "30s",
		"turbo_init":           "1",
		"turbo_max":            "50",
		"turbo_step":           "2",
		"turbo_level_requests": "30",
		"turbo_min_success":    "0.9",
		"turbo_max_latency":    "10s",
		"prompt_value":         "你好，介绍一下你自己。",
	}
	state := &wizardState{
		fromView:        from,
		input:           input,
		values:          values,
		protocolIndex:   protocolIndex(cfg.DefaultProtocol),
		mode:            modeStandard,
		promptModeIndex: 0,
		stream:          true,
		thinking:        false,
		report:          true,
	}
	if existing != nil {
		state.editingTaskID = existing.ID
		state.createdAt = existing.CreatedAt
		state.lastRunAt = existing.LastRunAt
		state.lastRunSummary = existing.LastRunSummary
		state.values["name"] = existing.Name
		state.values["endpoint"] = existing.Input.ResolvedEndpointURL()
		state.values["apiKey"] = existing.Input.ApiKey
		state.values["model"] = existing.Input.Model
		state.protocolIndex = protocolIndex(existing.Input.NormalizedProtocol())
		state.stream = existing.Input.Stream
		state.thinking = existing.Input.Thinking
		state.report = existing.Input.Report
		if existing.Input.Turbo {
			state.mode = modeTurbo
			state.values["turbo_init"] = strconv.Itoa(existing.Input.TurboConfig.InitConcurrency)
			state.values["turbo_max"] = strconv.Itoa(existing.Input.TurboConfig.MaxConcurrency)
			state.values["turbo_step"] = strconv.Itoa(existing.Input.TurboConfig.StepSize)
			state.values["turbo_level_requests"] = strconv.Itoa(existing.Input.TurboConfig.LevelRequests)
			state.values["turbo_min_success"] = strconv.FormatFloat(existing.Input.TurboConfig.MinSuccessRate, 'f', -1, 64)
			state.values["turbo_max_latency"] = existing.Input.TurboConfig.MaxLatency.String()
		} else {
			state.values["concurrency"] = strconv.Itoa(existing.Input.Concurrency)
			state.values["count"] = strconv.Itoa(existing.Input.Count)
			if existing.Input.Timeout > 0 {
				state.values["timeout"] = existing.Input.Timeout.String()
			}
		}
		switch existing.Input.PromptMode {
		case promptModeFile:
			state.promptModeIndex = 1
			state.values["prompt_value"] = existing.Input.PromptFile
		case promptModeGenerated:
			state.promptModeIndex = 2
			state.values["prompt_value"] = strconv.Itoa(existing.Input.PromptLength)
		default:
			state.promptModeIndex = 0
			state.values["prompt_value"] = existing.Input.PromptText
		}
	}
	return state
}

func protocolIndex(protocol string) int {
	for i, item := range protocolOptions {
		if item == types.NormalizeProtocol(protocol) {
			return i
		}
	}
	return 0
}

func (m *Model) wizardStepFields(step int) []wizardField {
	switch step {
	case 0:
		return []wizardField{
			{key: "name", label: "任务名称", kind: fieldText},
			{key: "protocol", label: "协议类型", kind: fieldSelect},
			{key: "endpoint", label: "完整接口地址", kind: fieldText},
			{key: "apiKey", label: "API 密钥", kind: fieldText},
			{key: "model", label: "测试模型", kind: fieldText},
		}
	case 1:
		fields := []wizardField{
			{key: "mode", label: "运行模式", kind: fieldSelect},
		}
		if m.wizard.mode == modeTurbo {
			fields = append(fields,
				wizardField{key: "turbo_init", label: "初始并发", kind: fieldText},
				wizardField{key: "turbo_max", label: "最大并发", kind: fieldText},
				wizardField{key: "turbo_step", label: "步进值", kind: fieldText},
				wizardField{key: "turbo_level_requests", label: "每级请求数", kind: fieldText},
				wizardField{key: "turbo_min_success", label: "最小成功率", kind: fieldText},
				wizardField{key: "turbo_max_latency", label: "最大平均延迟", kind: fieldText},
			)
		} else {
			fields = append(fields,
				wizardField{key: "concurrency", label: "并发数", kind: fieldText},
				wizardField{key: "count", label: "请求总数", kind: fieldText},
				wizardField{key: "timeout", label: "超时时间", kind: fieldText},
			)
		}
		fields = append(fields,
			wizardField{key: "stream", label: "流式模式", kind: fieldToggle},
			wizardField{key: "thinking", label: "Thinking 模式", kind: fieldToggle},
			wizardField{key: "report", label: "生成报告", kind: fieldToggle},
			wizardField{key: "prompt_mode", label: "Prompt 方式", kind: fieldSelect},
			wizardField{key: "prompt_value", label: promptValueLabel(m.wizard.promptModeIndex), kind: fieldText},
		)
		return fields
	default:
		return nil
	}
}

func (m *Model) advanceWizardField(delta int) {
	if m.wizard == nil {
		return
	}
	fields := m.wizardStepFields(m.wizard.step)
	next := m.wizard.fieldIndex + delta
	if next < 0 {
		if m.wizard.step > 0 {
			m.wizard.step--
			prevFields := m.wizardStepFields(m.wizard.step)
			m.wizard.fieldIndex = len(prevFields) - 1
			m.refreshWizardInput()
		}
		return
	}
	if next >= len(fields) {
		m.wizard.step++
		m.wizard.fieldIndex = 0
		if m.wizard.step < 2 {
			m.refreshWizardInput()
		}
		return
	}
	m.wizard.fieldIndex = next
	m.refreshWizardInput()
}

func promptValueLabel(promptModeIndex int) string {
	switch promptModeOptions[promptModeIndex] {
	case promptModeFile:
		return "Prompt 文件路径"
	case promptModeGenerated:
		return "Prompt 生成长度"
	default:
		return "Prompt 文本"
	}
}

func (m *Model) currentWizardField() wizardField {
	if m.wizard == nil {
		return wizardField{}
	}
	fields := m.wizardStepFields(m.wizard.step)
	if len(fields) == 0 || m.wizard.fieldIndex >= len(fields) {
		return wizardField{}
	}
	return fields[m.wizard.fieldIndex]
}

func (m *Model) refreshWizardInput() {
	field := m.currentWizardField()
	m.wizard.input.Blur()
	m.wizard.input.Focus()
	m.wizard.input.EchoMode = textinput.EchoNormal
	if field.key == "apiKey" {
		m.wizard.input.EchoMode = textinput.EchoPassword
	}
	m.wizard.input.SetValue(m.wizard.values[field.key])
	if field.key == "prompt_value" {
		m.wizard.input.Placeholder = promptValueLabel(m.wizard.promptModeIndex)
	} else {
		m.wizard.input.Placeholder = field.label
	}
	if field.kind != fieldText {
		m.wizard.input.SetValue("")
	}
}

func (m *Model) cycleWizardField(delta int) {
	if m.wizard == nil {
		return
	}
	field := m.currentWizardField()
	switch field.key {
	case "protocol":
		m.wizard.protocolIndex = wrapIndex(m.wizard.protocolIndex+delta, len(protocolOptions))
	case "mode":
		if m.wizard.mode == modeStandard {
			m.wizard.mode = modeTurbo
		} else {
			m.wizard.mode = modeStandard
		}
	case "prompt_mode":
		m.wizard.promptModeIndex = wrapIndex(m.wizard.promptModeIndex+delta, len(promptModeOptions))
		m.wizard.values["prompt_value"] = ""
	case "stream":
		m.wizard.stream = !m.wizard.stream
	case "thinking":
		m.wizard.thinking = !m.wizard.thinking
	case "report":
		m.wizard.report = !m.wizard.report
	default:
		return
	}
	// Clamp fieldIndex in case field count changed (e.g. mode switch)
	fields := m.wizardStepFields(m.wizard.step)
	if m.wizard.fieldIndex >= len(fields) && len(fields) > 0 {
		m.wizard.fieldIndex = len(fields) - 1
	}
	m.refreshWizardInput()
}

func (m *Model) displayWizardValue(field wizardField) string {
	switch field.key {
	case "protocol":
		return protocolOptions[m.wizard.protocolIndex]
	case "mode":
		return m.wizard.mode
	case "stream":
		return boolLabel(m.wizard.stream)
	case "thinking":
		return boolLabel(m.wizard.thinking)
	case "report":
		return boolLabel(m.wizard.report)
	case "prompt_mode":
		return promptModeOptions[m.wizard.promptModeIndex]
	default:
		return m.wizard.values[field.key]
	}
}

func boolLabel(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}

func wrapIndex(index, length int) int {
	if length == 0 {
		return 0
	}
	for index < 0 {
		index += length
	}
	return index % length
}

func buildTaskDefinition(state *wizardState) (types.TaskDefinition, error) {
	protocol := protocolOptions[state.protocolIndex]
	input := types.Input{
		Protocol:    protocol,
		EndpointURL: strings.TrimSpace(state.values["endpoint"]),
		ApiKey:      strings.TrimSpace(state.values["apiKey"]),
		Model:       strings.TrimSpace(state.values["model"]),
		Stream:      state.stream,
		Thinking:    state.thinking,
		Report:      state.report,
		PromptMode:  promptModeOptions[state.promptModeIndex],
	}

	switch input.PromptMode {
	case promptModeFile:
		input.PromptFile = strings.TrimSpace(state.values["prompt_value"])
	case promptModeGenerated:
		length, err := strconv.Atoi(strings.TrimSpace(state.values["prompt_value"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid prompt length: %w", err)
		}
		input.PromptLength = length
	default:
		input.PromptText = state.values["prompt_value"]
	}

	if state.mode == modeTurbo {
		initConcurrency, err := strconv.Atoi(strings.TrimSpace(state.values["turbo_init"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid turbo init concurrency: %w", err)
		}
		maxConcurrency, err := strconv.Atoi(strings.TrimSpace(state.values["turbo_max"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid turbo max concurrency: %w", err)
		}
		stepSize, err := strconv.Atoi(strings.TrimSpace(state.values["turbo_step"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid turbo step size: %w", err)
		}
		levelRequests, err := strconv.Atoi(strings.TrimSpace(state.values["turbo_level_requests"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid turbo level requests: %w", err)
		}
		minSuccessRate, err := strconv.ParseFloat(strings.TrimSpace(state.values["turbo_min_success"]), 64)
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid turbo min success rate: %w", err)
		}
		maxLatency, err := time.ParseDuration(strings.TrimSpace(state.values["turbo_max_latency"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid turbo max latency: %w", err)
		}
		input.Turbo = true
		input.Count = levelRequests
		input.Concurrency = initConcurrency
		input.TurboConfig = types.TurboConfig{
			InitConcurrency: initConcurrency,
			MaxConcurrency:  maxConcurrency,
			StepSize:        stepSize,
			LevelRequests:   levelRequests,
			MinSuccessRate:  minSuccessRate,
			MaxLatency:      maxLatency,
		}
	} else {
		concurrency, err := strconv.Atoi(strings.TrimSpace(state.values["concurrency"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid concurrency: %w", err)
		}
		count, err := strconv.Atoi(strings.TrimSpace(state.values["count"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid count: %w", err)
		}
		timeout, err := time.ParseDuration(strings.TrimSpace(state.values["timeout"]))
		if err != nil {
			return types.TaskDefinition{}, fmt.Errorf("invalid timeout: %w", err)
		}
		input.Concurrency = concurrency
		input.Count = count
		input.Timeout = timeout
	}

	validatedInput, err := task.HydrateInput(input)
	if err != nil {
		return types.TaskDefinition{}, err
	}
	validatedInput.PromptSource = nil

	now := time.Now()
	createdAt := state.createdAt
	if createdAt.IsZero() {
		createdAt = now
	}

	return types.TaskDefinition{
		ID:             state.editingTaskID,
		Name:           strings.TrimSpace(state.values["name"]),
		Input:          validatedInput,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
		LastRunAt:      state.lastRunAt,
		LastRunSummary: state.lastRunSummary,
	}, nil
}

func (m *Model) saveWizard() error {
	taskDef, err := buildTaskDefinition(m.wizard)
	if err != nil {
		return err
	}
	if taskDef.ID == "" {
		taskDef.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	m.store.Upsert(taskDef)
	if err := m.store.Save(); err != nil {
		return err
	}
	m.tasks = m.store.Tasks
	for i, item := range m.tasks {
		if item.ID == taskDef.ID {
			m.selected = i
			break
		}
	}
	m.reloadHistoryForSelectedTask()
	m.status = "任务已保存"
	m.wizard = nil
	m.view = viewTaskDetail
	return nil
}

func (m *Model) reloadHistoryForSelectedTask() {
	taskDef, ok := m.currentTask()
	if !ok {
		m.history = nil
		return
	}
	history, err := task.LoadHistory(taskDef.ID, 5)
	if err != nil {
		m.err = err
		m.history = nil
		return
	}
	m.history = history
}

func (m *Model) startTaskRun(taskDef types.TaskDefinition) {
	input, err := task.HydrateInput(taskDef.Input)
	if err != nil {
		m.err = err
		return
	}
	m.runningTask = &taskDef
	m.runStartedAt = time.Now()
	m.progress = types.StatsData{}
	m.runResult = nil
	m.turboResult = nil
	m.view = viewDashboard

	if input.Turbo {
		engine := turbo.New(turbo.DefaultRunnerFactory(taskDef.ID))
		m.activeTurbo = engine
		go func() {
			result, err := engine.Run(input)
			if err != nil {
				m.program.Send(asyncErrorMsg{err: err})
				return
			}
			m.program.Send(turboCompleteMsg{taskID: taskDef.ID, result: result})
		}()
		return
	}

	runnerInstance, err := runner.NewRunner(taskDef.ID, input)
	if err != nil {
		m.err = err
		return
	}
	m.activeRunner = runnerInstance
	go func() {
		result, err := runnerInstance.RunWithProgress(func(stats types.StatsData) {
			m.program.Send(progressMsg{stats: stats})
		})
		if err != nil {
			m.program.Send(asyncErrorMsg{err: err})
			return
		}
		paths, err := generateReports(result, input.Report)
		if err != nil {
			m.program.Send(asyncErrorMsg{err: err})
			return
		}
		m.program.Send(runCompleteMsg{taskID: taskDef.ID, result: result, reportPaths: paths})
	}()
}

func generateReports(result *types.ReportData, enabled bool) ([]string, error) {
	if !enabled || result == nil {
		return nil, nil
	}
	manager := report.NewReportManager()
	return manager.GenerateReports([]types.ReportData{*result}, []string{"json", "csv"})
}

func (m *Model) persistStandardRun(taskID string, result *types.ReportData, reportPaths []string) {
	taskDef, ok := m.store.Get(taskID)
	if !ok {
		return
	}
	finishedAt := time.Now()
	summary := &types.TaskRunSummary{
		RunID:        fmt.Sprintf("run_%d", finishedAt.UnixNano()),
		TaskID:       taskID,
		Mode:         modeStandard,
		Status:       "completed",
		Protocol:     result.Protocol,
		Model:        result.Model,
		StartedAt:    m.runStartedAt,
		FinishedAt:   finishedAt,
		SuccessRate:  result.SuccessRate,
		AvgTTFT:      result.AvgTTFT,
		AvgTPS:       result.AvgTPS,
		CacheHitRate: result.AvgCacheHitRate * 100,
	}
	for _, path := range reportPaths {
		switch filepath.Ext(path) {
		case ".json":
			summary.ReportJSONPath = path
		case ".csv":
			summary.ReportCSVPath = path
		}
	}
	taskDef.LastRunAt = &finishedAt
	taskDef.LastRunSummary = summary
	m.store.Upsert(taskDef)
	_ = m.store.Save()
	_ = task.AppendRun(taskID, *summary)
	m.tasks = m.store.Tasks
	m.reloadHistoryForSelectedTask()
}

func (m *Model) persistTurboRun(taskID string, result *types.TurboResult) {
	taskDef, ok := m.store.Get(taskID)
	if !ok {
		return
	}
	finishedAt := time.Now()
	latestSuccessRate := 0.0
	latestCacheHitRate := 0.0
	if len(result.Levels) > 0 {
		lastLevel := result.Levels[len(result.Levels)-1]
		latestSuccessRate = lastLevel.SuccessRate * 100
		latestCacheHitRate = lastLevel.CacheHitRate * 100
	}
	summary := &types.TaskRunSummary{
		RunID:                fmt.Sprintf("run_%d", finishedAt.UnixNano()),
		TaskID:               taskID,
		Mode:                 modeTurbo,
		Status:               result.StopReason,
		Protocol:             result.Protocol,
		Model:                result.Model,
		StartedAt:            m.runStartedAt,
		FinishedAt:           finishedAt,
		SuccessRate:          latestSuccessRate,
		AvgTPS:               result.PeakTPS,
		CacheHitRate:         latestCacheHitRate,
		MaxStableConcurrency: result.MaxStableConcurrency,
	}
	taskDef.LastRunAt = &finishedAt
	taskDef.LastRunSummary = summary
	m.store.Upsert(taskDef)
	_ = m.store.Save()
	_ = task.AppendRun(taskID, *summary)
	m.tasks = m.store.Tasks
	m.reloadHistoryForSelectedTask()
}

func promptSummary(input types.Input) string {
	switch input.PromptMode {
	case promptModeFile:
		return input.PromptFile
	case promptModeGenerated:
		return fmt.Sprintf("长度 %d", input.PromptLength)
	default:
		if len(input.PromptText) > 48 {
			return input.PromptText[:48] + "..."
		}
		return input.PromptText
	}
}
