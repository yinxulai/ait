package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// ─── Prompt 模式常量 ──────────────────────────────────────────────────────────

const (
	promptModeText      = "text"
	promptModeFile      = "file"
	promptModeGenerated = "generated"
)

// ─── 向导状态 ─────────────────────────────────────────────────────────────────

type wizardStep int

const (
	wizardStepBasic    wizardStep = 0 // 基础配置（名称、模式、协议）
	wizardStepEndpoint wizardStep = 1 // 接口配置（URL、APIKey、模型）
	wizardStepPrompt   wizardStep = 2 // Prompt 配置（模式、内容、并发参数）
)

// wizardState 向导的完整状态。
type wizardState struct {
	step      wizardStep
	editingID string // 非空表示编辑模式，存放被编辑任务的 ID

	// Step 0
	name     string
	turbo    bool
	protocol string // types.Protocol* 常量

	// Step 1
	endpointURL string
	apiKey      string
	model       string
	stream      bool
	thinking    bool

	// Step 2 — Standard
	concurrency int
	count       int

	// Step 2 — Turbo
	initConcurrency int
	maxConcurrency  int
	stepSize        int
	levelRequests   int
	minSuccessRate  float64

	// Prompt
	promptMode   string
	promptText   string
	promptFile   string
	promptLength int

	// 当前活跃字段索引
	fieldIndex int
}

// openWizard 打开向导。task==nil 表示新建，非 nil 表示编辑。
func (m *Model) openWizard(task *types.TaskDefinition) {
	if task == nil {
		m.wizard = &wizardState{
			step:           wizardStepBasic,
			protocol:       types.ProtocolOpenAICompletions,
			concurrency:    10,
			count:          100,
			initConcurrency: 1,
			maxConcurrency: 50,
			stepSize:       5,
			levelRequests:  20,
			minSuccessRate: 95,
			promptMode:     promptModeText,
		}
	} else {
		inp := task.Input
		tc := inp.TurboConfig
		m.wizard = &wizardState{
			step:           wizardStepBasic,
			editingID:      task.ID,
			name:           task.Name,
			turbo:          inp.Turbo,
			protocol:       types.NormalizeProtocol(inp.Protocol),
			endpointURL:    inp.EndpointURL,
			apiKey:         inp.ApiKey,
			model:          inp.Model,
			stream:         inp.Stream,
			thinking:       inp.Thinking,
			concurrency:    inp.Concurrency,
			count:          inp.Count,
			initConcurrency: tc.InitConcurrency,
			maxConcurrency: tc.MaxConcurrency,
			stepSize:       tc.StepSize,
			levelRequests:  tc.LevelRequests,
			minSuccessRate: tc.MinSuccessRate,
			promptMode:     inp.PromptMode,
			promptText:     inp.PromptText,
			promptFile:     inp.PromptFile,
			promptLength:   inp.PromptLength,
		}
		if m.wizard.promptMode == "" {
			m.wizard.promptMode = promptModeText
		}
		if m.wizard.concurrency == 0 {
			m.wizard.concurrency = 10
		}
		if m.wizard.count == 0 {
			m.wizard.count = 100
		}
	}
	m.view = viewWizard
}

// ─── 按键处理 ─────────────────────────────────────────────────────────────────

func (m *Model) handleWizardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	wz := m.wizard
	if wz == nil {
		m.view = viewTaskList
		return m, nil
	}

	fields := m.wizardFields()
	maxField := len(fields) - 1

	switch msg.String() {
	case "esc":
		m.wizard = nil
		m.view = viewTaskList
		return m, nil

	case "tab", "down", "j":
		if wz.fieldIndex < maxField {
			wz.fieldIndex++
		}

	case "shift+tab", "up", "k":
		if wz.fieldIndex > 0 {
			wz.fieldIndex--
		}

	case "left", "right":
		// 布尔/枚举切换
		m.wizardToggleField(fields, wz.fieldIndex, msg.String() == "right")

	case "enter":
		// 如果在最后一个字段，或者按下 Enter 且是最后步骤，保存并运行
		if int(wz.step) == 2 && wz.fieldIndex == maxField {
			return m, m.saveWizard(true)
		}
		// 否则 Next / 保存
		if wz.fieldIndex == maxField {
			wz.step++
			wz.fieldIndex = 0
		} else {
			wz.fieldIndex++
		}

	case "ctrl+s":
		return m, m.saveWizard(false)

	case "ctrl+enter":
		if int(wz.step) == 2 {
			return m, m.saveWizard(true)
		}

	case "backspace":
		m.wizardBackspace(fields, wz.fieldIndex)

	default:
		// 字符输入
		if len(msg.Runes) > 0 {
			m.wizardInput(fields, wz.fieldIndex, string(msg.Runes))
		}
	}

	return m, nil
}

// ─── 字段定义 ─────────────────────────────────────────────────────────────────

type fieldKind int

const (
	fieldText   fieldKind = iota // 自由文本输入
	fieldNumber                  // 数字
	fieldBool                    // 布尔开关
	fieldEnum                    // 枚举循环
	fieldAction                  // 动作按钮（保存/运行）
)

type wizardField struct {
	kind    fieldKind
	label   string
	getValue func(wz *wizardState) string
	setValue func(wz *wizardState, s string)
	options  []string // 仅 fieldEnum 使用
}

// wizardFields 根据当前步骤和 turbo 模式动态返回字段列表。
func (m *Model) wizardFields() []wizardField {
	wz := m.wizard
	if wz == nil {
		return nil
	}
	switch wz.step {
	case wizardStepBasic:
		return []wizardField{
			{kind: fieldText, label: "名称",
				getValue: func(wz *wizardState) string { return wz.name },
				setValue: func(wz *wizardState, s string) { wz.name = s }},
			{kind: fieldBool, label: "Turbo 模式",
				getValue: func(wz *wizardState) string { return boolLabel(wz.turbo) },
				setValue: func(wz *wizardState, s string) { wz.turbo = (s == "true") }},
			{kind: fieldEnum, label: "协议",
				options: []string{
					types.ProtocolOpenAICompletions,
					types.ProtocolOpenAIResponses,
					types.ProtocolAnthropicMessages,
				},
				getValue: func(wz *wizardState) string { return wz.protocol },
				setValue: func(wz *wizardState, s string) { wz.protocol = s }},
		}

	case wizardStepEndpoint:
		return []wizardField{
			{kind: fieldText, label: "接口地址 (可选)",
				getValue: func(wz *wizardState) string { return wz.endpointURL },
				setValue: func(wz *wizardState, s string) { wz.endpointURL = s }},
			{kind: fieldText, label: "API Key",
				getValue: func(wz *wizardState) string { return wz.apiKey },
				setValue: func(wz *wizardState, s string) { wz.apiKey = s }},
			{kind: fieldText, label: "模型",
				getValue: func(wz *wizardState) string { return wz.model },
				setValue: func(wz *wizardState, s string) { wz.model = s }},
			{kind: fieldBool, label: "流式输出",
				getValue: func(wz *wizardState) string { return boolLabel(wz.stream) },
				setValue: func(wz *wizardState, s string) { wz.stream = (s == "true") }},
			{kind: fieldBool, label: "Thinking 模式",
				getValue: func(wz *wizardState) string { return boolLabel(wz.thinking) },
				setValue: func(wz *wizardState, s string) { wz.thinking = (s == "true") }},
		}

	case wizardStepPrompt:
		base := []wizardField{
			{kind: fieldEnum, label: "Prompt 模式",
				options: []string{promptModeText, promptModeFile, promptModeGenerated},
				getValue: func(wz *wizardState) string { return wz.promptMode },
				setValue: func(wz *wizardState, s string) { wz.promptMode = s }},
		}
		switch wz.promptMode {
		case promptModeFile:
			base = append(base, wizardField{kind: fieldText, label: "文件路径",
				getValue: func(wz *wizardState) string { return wz.promptFile },
				setValue: func(wz *wizardState, s string) { wz.promptFile = s }})
		case promptModeGenerated:
			base = append(base, wizardField{kind: fieldNumber, label: "生成长度",
				getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.promptLength) },
				setValue: func(wz *wizardState, s string) {
					if n, err := strconv.Atoi(s); err == nil {
						wz.promptLength = n
					}
				}})
		default: // text
			base = append(base, wizardField{kind: fieldText, label: "Prompt 文本",
				getValue: func(wz *wizardState) string { return wz.promptText },
				setValue: func(wz *wizardState, s string) { wz.promptText = s }})
		}

		if wz.turbo {
			base = append(base,
				wizardField{kind: fieldNumber, label: "初始并发",
					getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.initConcurrency) },
					setValue: func(wz *wizardState, s string) {
						if n, err := strconv.Atoi(s); err == nil && n > 0 {
							wz.initConcurrency = n
						}
					}},
				wizardField{kind: fieldNumber, label: "最大并发",
					getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.maxConcurrency) },
					setValue: func(wz *wizardState, s string) {
						if n, err := strconv.Atoi(s); err == nil && n > 0 {
							wz.maxConcurrency = n
						}
					}},
				wizardField{kind: fieldNumber, label: "步进大小",
					getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.stepSize) },
					setValue: func(wz *wizardState, s string) {
						if n, err := strconv.Atoi(s); err == nil && n > 0 {
							wz.stepSize = n
						}
					}},
				wizardField{kind: fieldNumber, label: "每级请求数",
					getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.levelRequests) },
					setValue: func(wz *wizardState, s string) {
						if n, err := strconv.Atoi(s); err == nil && n > 0 {
							wz.levelRequests = n
						}
					}},
			)
		} else {
			base = append(base,
				wizardField{kind: fieldNumber, label: "并发数",
					getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.concurrency) },
					setValue: func(wz *wizardState, s string) {
						if n, err := strconv.Atoi(s); err == nil && n > 0 {
							wz.concurrency = n
						}
					}},
				wizardField{kind: fieldNumber, label: "请求总数",
					getValue: func(wz *wizardState) string { return fmt.Sprintf("%d", wz.count) },
					setValue: func(wz *wizardState, s string) {
						if n, err := strconv.Atoi(s); err == nil && n > 0 {
							wz.count = n
						}
					}},
			)
		}
		return base
	}
	return nil
}

// wizardToggleField 处理 ←→ 按键的枚举/布尔切换。
func (m *Model) wizardToggleField(fields []wizardField, idx int, forward bool) {
	wz := m.wizard
	if idx >= len(fields) {
		return
	}
	f := fields[idx]
	switch f.kind {
	case fieldBool:
		cur := f.getValue(wz) == "开启"
		if forward {
			f.setValue(wz, boolVal(!cur))
		} else {
			f.setValue(wz, boolVal(cur))
		}
	case fieldEnum:
		cur := f.getValue(wz)
		i := 0
		for j, o := range f.options {
			if o == cur {
				i = j
				break
			}
		}
		if forward {
			i = (i + 1) % len(f.options)
		} else {
			i = (i - 1 + len(f.options)) % len(f.options)
		}
		f.setValue(wz, f.options[i])
	}
}

func boolVal(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// wizardInput 追加字符到文本/数字字段。
func (m *Model) wizardInput(fields []wizardField, idx int, s string) {
	wz := m.wizard
	if idx >= len(fields) {
		return
	}
	f := fields[idx]
	if f.kind == fieldText {
		cur := f.getValue(wz)
		f.setValue(wz, cur+s)
	} else if f.kind == fieldNumber {
		cur := f.getValue(wz)
		if len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
			f.setValue(wz, cur+s)
		}
	}
}

// wizardBackspace 删除文本/数字字段末尾字符。
func (m *Model) wizardBackspace(fields []wizardField, idx int) {
	wz := m.wizard
	if idx >= len(fields) {
		return
	}
	f := fields[idx]
	if f.kind == fieldText || f.kind == fieldNumber {
		cur := []rune(f.getValue(wz))
		if len(cur) > 0 {
			f.setValue(wz, string(cur[:len(cur)-1]))
		}
	}
}

// ─── 保存 ─────────────────────────────────────────────────────────────────────

// buildTaskInput 从 wizardState 构建 types.Input。
func (m *Model) buildTaskInput() types.Input {
	wz := m.wizard
	inp := types.Input{
		Protocol:    wz.protocol,
		EndpointURL: wz.endpointURL,
		ApiKey:      wz.apiKey,
		Model:       wz.model,
		Stream:      wz.stream,
		Thinking:    wz.thinking,
		Turbo:       wz.turbo,
		PromptMode:  wz.promptMode,
		PromptText:  wz.promptText,
		PromptFile:  wz.promptFile,
		PromptLength: wz.promptLength,
	}
	if wz.turbo {
		inp.TurboConfig = types.TurboConfig{
			InitConcurrency: wz.initConcurrency,
			MaxConcurrency:  wz.maxConcurrency,
			StepSize:        wz.stepSize,
			LevelRequests:   wz.levelRequests,
			MinSuccessRate:  wz.minSuccessRate,
		}
	} else {
		inp.Concurrency = wz.concurrency
		inp.Count = wz.count
	}
	return inp
}

// saveWizard 保存或创建任务。autoStart=true 表示成功后立刻运行。
func (m *Model) saveWizard(autoStart bool) tea.Cmd {
	wz := m.wizard
	if wz == nil {
		return nil
	}
	cfg := server.TaskConfig{
		Name:  wz.name,
		Input: m.buildTaskInput(),
	}
	m.wizard = nil
	m.view = viewTaskList

	if wz.editingID != "" {
		return m.client.UpdateTaskCmd(wz.editingID, cfg)
	}
	return m.client.CreateTaskCmd(cfg, autoStart)
}

// ─── 渲染 ─────────────────────────────────────────────────────────────────────

func (m *Model) renderWizard() string {
	wz := m.wizard
	if wz == nil {
		return ""
	}

	title := "新建任务"
	if wz.editingID != "" {
		title = "编辑任务"
	}

	steps := []string{"基础配置", "接口配置", "参数配置"}
	var stepParts []string
	for i, s := range steps {
		switch {
		case i < int(wz.step):
			stepParts = append(stepParts, m.styles.stepDone.Render("✓ "+s))
		case i == int(wz.step):
			stepParts = append(stepParts, m.styles.stepActive.Render("▶ "+s))
		default:
			stepParts = append(stepParts, m.styles.stepTodo.Render("○ "+s))
		}
	}
	stepLine := strings.Join(stepParts, "  ")

	fields := m.wizardFields()
	var fieldLines []string
	for i, f := range fields {
		label := fmt.Sprintf("%-16s", f.label)
		val := f.getValue(wz)
		if f.kind == fieldBool {
			if wz.turbo && f.label == "Turbo 模式" {
				val = m.styles.ok.Render("开启")
			} else if !wz.turbo && f.label == "Turbo 模式" {
				val = m.styles.muted.Render("关闭")
			}
		}
		// mask API key display
		if f.label == "API Key" && val != "" {
			val = maskAPIKey(val)
		}

		var line string
		if i == wz.fieldIndex {
			cursor := m.styles.cursor.Render("▶")
			labelS := m.styles.sectionHead.Render(label)
			valS := m.styles.fieldActive.Render(" " + val + " " + m.styles.cursor.Render("_"))
			line = cursor + " " + labelS + "  " + valS
		} else {
			labelS := m.styles.label.Render(label)
			valS := m.styles.value.Render(val)
			line = "  " + labelS + "  " + valS
		}
		fieldLines = append(fieldLines, line)
	}

	// 底部操作提示
	var hints []string
	hints = append(hints, m.styles.key.Render("[↑↓/Tab]")+" 切换字段")
	hints = append(hints, m.styles.key.Render("[←→]")+" 切换选项")
	hints = append(hints, m.styles.key.Render("[Ctrl+S]")+" 保存")
	if int(wz.step) == 2 {
		hints = append(hints, m.styles.key.Render("[Ctrl+Enter]")+" 保存并运行")
	} else {
		hints = append(hints, m.styles.key.Render("[Enter]")+" 下一步")
	}
	hints = append(hints, m.styles.key.Render("[Esc]")+" 取消")
	hintLine := strings.Join(hints, "  ")

	dialogW := m.width * 70 / 100
	if dialogW < 60 {
		dialogW = 60
	}
	if dialogW > m.width-4 {
		dialogW = m.width - 4
	}

	inner := fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s",
		m.styles.sectionHead.Render(title),
		stepLine,
		strings.Join(fieldLines, "\n"),
		hintLine,
	)

	dialog := m.styles.dialog.Width(dialogW).Render(inner)
	totalH := m.height
	dialogRendered := dialog

	// 垂直居中
	dialogH := strings.Count(dialogRendered, "\n") + 1
	padTop := (totalH - dialogH) / 2
	if padTop < 0 {
		padTop = 0
	}
	topPad := strings.Repeat("\n", padTop)

	// 水平居中（外层宽度补齐）
	dialogLineW := lipgloss.Width(strings.Split(dialogRendered, "\n")[0])
	leftPad := (m.width - dialogLineW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	paddedLines := make([]string, 0)
	for _, l := range strings.Split(dialogRendered, "\n") {
		paddedLines = append(paddedLines, strings.Repeat(" ", leftPad)+l)
	}

	return topPad + strings.Join(paddedLines, "\n")
}
