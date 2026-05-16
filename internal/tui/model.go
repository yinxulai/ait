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
	current         int
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
	program      *tea.Program
	runningTask  *types.TaskDefinition
	runStartedAt time.Time
	progress     types.StatsData
	runResult    *types.ReportData
	turboResult  *types.TurboResult
	activeRunner *runner.Runner
	activeTurbo  *turbo.Engine
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
	case runCompleteMsg:
		m.activeRunner = nil
		m.runResult = msg.result
		m.view = viewResult
		m.status = fmt.Sprintf("标准模式完成，共 %d 请求", msg.result.TotalRequests)
		m.persistStandardRun(msg.taskID, msg.result, msg.reportPaths)
		return m, nil
	case turboCompleteMsg:
		m.activeTurbo = nil
		m.turboResult = msg.result
		m.view = viewTurboResult
		m.status = fmt.Sprintf("Turbo 完成，最大稳定并发 %d", msg.result.MaxStableConcurrency)
		m.persistTurboRun(msg.taskID, msg.result)
		return m, nil
	case asyncErrorMsg:
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
		if _, ok := m.currentTask(); ok {
			m.reloadHistoryForSelectedTask()
			m.view = viewTaskDetail
		}
	case "r":
		if taskDef, ok := m.currentTask(); ok {
			m.startTaskRun(taskDef)
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
		m.startTaskRun(taskDef)
	}

	return m, nil
}

func (m *Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	field := m.currentWizardField()
	switch msg.String() {
	case "esc":
		m.view = m.wizard.fromView
		m.wizard = nil
		return m, nil
	case "tab", "enter":
		if field.kind == fieldText {
			m.wizard.values[field.key] = m.wizard.input.Value()
		}
		if m.wizard.current == len(m.wizardFields())-1 {
			if err := m.saveWizard(); err != nil {
				m.err = err
				m.status = err.Error()
			}
			return m, nil
		}
		m.wizard.current++
		m.refreshWizardInput()
		return m, nil
	case "shift+tab", "up":
		if m.wizard.current > 0 {
			m.wizard.values[field.key] = m.wizard.input.Value()
			m.wizard.current--
			m.refreshWizardInput()
		}
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
	case "s", "q", "esc":
		if m.activeRunner != nil {
			m.activeRunner.Stop()
		}
		if m.activeTurbo != nil {
			m.activeTurbo.Stop()
		}
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
	var rows []string
	for i, taskDef := range m.tasks {
		mode := modeStandard
		if taskDef.Input.Turbo {
			mode = modeTurbo
		}
		summary := "从未运行"
		if taskDef.LastRunSummary != nil {
			summary = fmt.Sprintf("上次 %.1f%% · %.1f tok/s", taskDef.LastRunSummary.SuccessRate, taskDef.LastRunSummary.AvgTPS)
		}
		line := fmt.Sprintf("%s  %s  %s  %s", taskDef.Name, taskDef.Input.Model, mode, summary)
		if i == m.selected {
			line = m.styles.selected.Render("▶ " + line)
		} else {
			line = "  " + line
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = append(rows, m.styles.muted.Render("暂无任务，按 a 新建"))
	}

	content := []string{
		m.styles.title.Render("AIT 任务中心"),
		m.styles.subtitle.Render(fmt.Sprintf("已保存任务: %d", len(m.tasks))),
		m.styles.panel.Render(strings.Join(rows, "\n")),
		m.footer("[↑↓] 选择", "[Enter] 详情", "[a] 新建", "[r] 运行", "[e] 编辑", "[d] 删除", "[q] 退出"),
	}
	if m.status != "" {
		content = append(content, m.styles.muted.Render(m.status))
	}
	return strings.Join(content, "\n")
}

func (m *Model) renderTaskDetail() string {
	taskDef, ok := m.currentTask()
	if !ok {
		return m.styles.error.Render("任务不存在")
	}
	lastRun := "从未运行"
	if taskDef.LastRunAt != nil {
		lastRun = taskDef.LastRunAt.Format(time.RFC3339)
	}
	mode := modeStandard
	if taskDef.Input.Turbo {
		mode = modeTurbo
	}
	left := []string{
		fmt.Sprintf("名称: %s", taskDef.Name),
		fmt.Sprintf("协议: %s", taskDef.Input.NormalizedProtocol()),
		fmt.Sprintf("接口: %s", taskDef.Input.ResolvedEndpointURL()),
		fmt.Sprintf("模型: %s", taskDef.Input.Model),
		fmt.Sprintf("模式: %s", mode),
		fmt.Sprintf("Prompt: %s", promptSummary(taskDef.Input)),
		fmt.Sprintf("最近运行: %s", lastRun),
	}

	historyLines := []string{m.styles.label.Render("最近运行记录")}
	if len(m.history) == 0 {
		historyLines = append(historyLines, m.styles.muted.Render("暂无历史"))
	} else {
		for _, item := range m.history {
			historyLines = append(historyLines, fmt.Sprintf("%s  %s  %.1f%%  %.1f tok/s  cache %.1f%%", item.FinishedAt.Format("2006-01-02 15:04:05"), item.Mode, item.SuccessRate, item.AvgTPS, item.CacheHitRate))
		}
	}

	return strings.Join([]string{
		m.styles.title.Render("AIT 任务详情"),
		m.styles.panel.Render(strings.Join(left, "\n")),
		m.styles.panel.Render(strings.Join(historyLines, "\n")),
		m.footer("[Enter] 运行", "[e] 编辑", "[d] 删除", "[b] 返回"),
	}, "\n")
}

func (m *Model) renderWizard() string {
	fields := m.wizardFields()
	field := fields[m.wizard.current]
	var lines []string
	for i, f := range fields {
		marker := "  "
		if i == m.wizard.current {
			marker = "▶ "
		}
		lines = append(lines, marker+fmt.Sprintf("%s: %s", f.label, m.displayWizardValue(f)))
	}

	editor := ""
	if field.kind == fieldText {
		editor = m.styles.panel.Render(m.wizard.input.View())
	} else {
		editor = m.styles.panel.Render(m.displayWizardValue(field))
	}

	return strings.Join([]string{
		m.styles.title.Render("AIT 任务向导"),
		m.styles.subtitle.Render(fmt.Sprintf("步骤 %d/%d", m.wizard.current+1, len(fields))),
		m.styles.panel.Render(strings.Join(lines, "\n")),
		editor,
		m.footer("[Enter/Tab] 下一项或保存", "[←→/Space] 切换选项", "[Esc] 取消"),
	}, "\n")
}

func (m *Model) renderDashboard() string {
	title := "AIT 正在运行"
	if m.runningTask != nil && m.runningTask.Input.Turbo {
		title = "AIT Turbo 正在探测"
	}
	stats := []string{
		fmt.Sprintf("完成: %d", m.progress.CompletedCount),
		fmt.Sprintf("失败: %d", m.progress.FailedCount),
		fmt.Sprintf("运行时长: %s", m.progress.ElapsedTime.Truncate(100*time.Millisecond)),
	}
	if len(m.progress.CacheHitRates) > 0 {
		stats = append(stats, fmt.Sprintf("最近缓存命中率: %.1f%%", m.progress.CacheHitRates[len(m.progress.CacheHitRates)-1]*100))
	}
	return strings.Join([]string{
		m.styles.title.Render(title),
		m.styles.panel.Render(strings.Join(stats, "\n")),
		m.footer("[s] 停止"),
	}, "\n")
}

func (m *Model) renderResult() string {
	if m.runResult == nil {
		return m.styles.error.Render("结果为空")
	}
	result := m.runResult
	lines := []string{
		fmt.Sprintf("协议: %s", result.Protocol),
		fmt.Sprintf("接口: %s", result.EndpointURL),
		fmt.Sprintf("成功率: %.1f%%", result.SuccessRate),
		fmt.Sprintf("平均 TTFT: %s", result.AvgTTFT),
		fmt.Sprintf("平均 TPS: %.2f", result.AvgTPS),
		fmt.Sprintf("缓存命中率: %.1f%%", result.AvgCacheHitRate*100),
		fmt.Sprintf("平均总耗时: %s", result.AvgTotalTime),
	}
	return strings.Join([]string{
		m.styles.title.Render("AIT 标准模式结果"),
		m.styles.panel.Render(strings.Join(lines, "\n")),
		m.footer("[b] 返回详情"),
	}, "\n")
}

func (m *Model) renderTurboResult() string {
	if m.turboResult == nil {
		return m.styles.error.Render("Turbo 结果为空")
	}
	lines := []string{
		fmt.Sprintf("协议: %s", m.turboResult.Protocol),
		fmt.Sprintf("接口: %s", m.turboResult.EndpointURL),
		fmt.Sprintf("最大稳定并发: %d", m.turboResult.MaxStableConcurrency),
		fmt.Sprintf("峰值平均 TPS: %.2f", m.turboResult.PeakTPS),
		fmt.Sprintf("停止原因: %s", m.turboResult.StopReason),
	}
	for _, level := range m.turboResult.Levels {
		status := "✓"
		if !level.Stable {
			status = "✗"
		}
		lines = append(lines, fmt.Sprintf("%s 并发 %d  成功率 %.1f%%  avgTPS %.2f  cache %.1f%%", status, level.Concurrency, level.SuccessRate*100, level.AvgTPS, level.CacheHitRate*100))
	}
	return strings.Join([]string{
		m.styles.title.Render("AIT Turbo 结果"),
		m.styles.panel.Render(strings.Join(lines, "\n")),
		m.footer("[b] 返回详情"),
	}, "\n")
}

func (m *Model) footer(parts ...string) string {
	styled := make([]string, 0, len(parts))
	for _, part := range parts {
		styled = append(styled, m.styles.key.Render(part+"  "))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, styled...)
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

func (m *Model) wizardFields() []wizardField {
	fields := []wizardField{
		{key: "name", label: "任务名称", kind: fieldText},
		{key: "protocol", label: "协议类型", kind: fieldSelect},
		{key: "endpoint", label: "完整接口地址", kind: fieldText},
		{key: "apiKey", label: "API 密钥", kind: fieldText},
		{key: "model", label: "测试模型", kind: fieldText},
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
		wizardField{key: "prompt_mode", label: "Prompt 输入方式", kind: fieldSelect},
		wizardField{key: "prompt_value", label: promptValueLabel(m.wizard.promptModeIndex), kind: fieldText},
	)
	return fields
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
	return m.wizardFields()[m.wizard.current]
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
	m.wizard.current = min(m.wizard.current, len(m.wizardFields())-1)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	m.status = "任务已保存"
	m.wizard = nil
	m.view = viewTaskList
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
