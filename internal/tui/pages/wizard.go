package pages

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// Prompt 模式常量
const (
	PromptModeText      = "text"
	PromptModeFile      = "file"
	PromptModeGenerated = "generated"
)

// wizardStep 步骤枚举
type wizardStep int

const (
	wizardStep1 wizardStep = 0 // Step 1/3 · 基本信息
	wizardStep2 wizardStep = 1 // Step 2/3 · 测试参数
	wizardStep3 wizardStep = 2 // Step 3/3 · 确认保存
)

// WizardState 向导的完整状态。
type WizardState struct {
	Step      wizardStep
	EditingID string // 非空 = 编辑模式

	// Step 1: 基本信息
	Name        string
	Protocol    string // types.Protocol* 常量
	EndpointURL string
	APIKey      string
	Model       string

	// Step 2: 测试参数
	Turbo  bool
	Stream bool

	// 标准模式参数
	Concurrency int
	Count       int
	Timeout     int // 秒

	// Turbo 模式参数
	InitConcurrency int
	MaxConcurrency  int
	StepSize        int
	LevelRequests   int
	MinSuccessRate  float64 // 百分比，如 90

	// Prompt 配置
	PromptMode   string
	PromptText   string
	PromptFile   string
	PromptLength int

	// 当前活跃字段索引（Tab 切换）
	FieldIndex int
}

// NewWizardState 创建新建任务向导状态（使用默认值）。
func NewWizardState() *WizardState {
	return &WizardState{
		Step:           wizardStep1,
		Protocol:       types.ProtocolOpenAICompletions,
		Concurrency:    10,
		Count:          100,
		Timeout:        30,
		InitConcurrency: 1,
		MaxConcurrency: 50,
		StepSize:       2,
		LevelRequests:  30,
		MinSuccessRate: 90,
		PromptMode:     PromptModeText,
	}
}

// NewWizardStateEdit 创建编辑任务向导状态（预填任务数据）。
func NewWizardStateEdit(t *types.TaskDefinition) *WizardState {
	if t == nil {
		return NewWizardState()
	}
	inp := t.Input
	tc := inp.TurboConfig
	wz := &WizardState{
		Step:           wizardStep1,
		EditingID:      t.ID,
		Name:           t.Name,
		Protocol:       types.NormalizeProtocol(inp.Protocol),
		EndpointURL:    inp.EndpointURL,
		APIKey:         inp.ApiKey,
		Model:          inp.Model,
		Turbo:          inp.Turbo,
		Stream:         inp.Stream,
		Concurrency:    inp.Concurrency,
		Count:          inp.Count,
		Timeout:        int(inp.Timeout.Seconds()),
		InitConcurrency: tc.InitConcurrency,
		MaxConcurrency: tc.MaxConcurrency,
		StepSize:       tc.StepSize,
		LevelRequests:  tc.LevelRequests,
		MinSuccessRate: tc.MinSuccessRate * 100, // 转为百分比
		PromptMode:     inp.PromptMode,
		PromptText:     inp.PromptText,
		PromptFile:     inp.PromptFile,
		PromptLength:   inp.PromptLength,
	}
	if wz.PromptMode == "" {
		wz.PromptMode = PromptModeText
	}
	if wz.Concurrency == 0 {
		wz.Concurrency = 10
	}
	if wz.Count == 0 {
		wz.Count = 100
	}
	if wz.Timeout == 0 {
		wz.Timeout = 30
	}
	if wz.MinSuccessRate == 0 {
		wz.MinSuccessRate = 90
	}
	if wz.StepSize == 0 {
		wz.StepSize = 2
	}
	if wz.LevelRequests == 0 {
		wz.LevelRequests = 30
	}
	if wz.MaxConcurrency == 0 {
		wz.MaxConcurrency = 50
	}
	return wz
}

// BuildTaskConfig 将向导状态转换为 server.TaskConfig。
func (wz *WizardState) BuildTaskConfig() server.TaskConfig {
	turboRate := wz.MinSuccessRate / 100 // 转回小数
	if turboRate <= 0 {
		turboRate = 0.9
	}
	return server.TaskConfig{
		Name: wz.Name,
		Input: types.Input{
			Protocol:    wz.Protocol,
			EndpointURL: wz.EndpointURL,
			ApiKey:      wz.APIKey,
			Model:       wz.Model,
			Concurrency: wz.Concurrency,
			Count:       wz.Count,
			Stream:      wz.Stream,
			Turbo:       wz.Turbo,
			TurboConfig: types.TurboConfig{
				InitConcurrency: wz.InitConcurrency,
				MaxConcurrency:  wz.MaxConcurrency,
				StepSize:        wz.StepSize,
				LevelRequests:   wz.LevelRequests,
				MinSuccessRate:  turboRate,
			},
			PromptMode:   wz.PromptMode,
			PromptText:   wz.PromptText,
			PromptFile:   wz.PromptFile,
			PromptLength: wz.PromptLength,
		},
	}
}

// fieldDef 向导字段定义
type fieldDef struct {
	kind  fieldKind
	label string
	// 获取当前值（字符串形式）
	get func(wz *WizardState) string
	// 设置文本值
	set func(wz *WizardState, v string)
	// 枚举/布尔切换
	toggle func(wz *WizardState, forward bool)
}

type fieldKind int

const (
	fieldText   fieldKind = iota // 自由文本输入
	fieldNumber                  // 数字
	fieldBool                    // 布尔开关
	fieldEnum                    // 枚举循环
)

// step1Fields 返回步骤1的字段列表。
func step1Fields() []fieldDef {
	protocols := []string{
		types.ProtocolOpenAICompletions,
		types.ProtocolOpenAIResponses,
		types.ProtocolAnthropicMessages,
	}
	return []fieldDef{
		{
			kind: fieldText, label: "任务名称",
			get: func(wz *WizardState) string { return wz.Name },
			set: func(wz *WizardState, v string) { wz.Name = v },
		},
		{
			kind: fieldEnum, label: "协议类型",
			get: func(wz *WizardState) string { return wz.Protocol },
			toggle: func(wz *WizardState, forward bool) {
				idx := 0
				for i, p := range protocols {
					if p == wz.Protocol {
						idx = i
						break
					}
				}
				if forward {
					idx = (idx + 1) % len(protocols)
				} else {
					idx = (idx - 1 + len(protocols)) % len(protocols)
				}
				wz.Protocol = protocols[idx]
				// 清空 endpoint，使其跟随协议默认值
				wz.EndpointURL = ""
			},
		},
		{
			kind: fieldText, label: "接口地址",
			get: func(wz *WizardState) string {
				if wz.EndpointURL != "" {
					return wz.EndpointURL
				}
				return types.DefaultEndpointURL(wz.Protocol)
			},
			set: func(wz *WizardState, v string) { wz.EndpointURL = v },
		},
		{
			kind: fieldText, label: "API 密钥",
			get: func(wz *WizardState) string { return wz.APIKey },
			set: func(wz *WizardState, v string) { wz.APIKey = v },
		},
		{
			kind: fieldText, label: "测试模型",
			get: func(wz *WizardState) string { return wz.Model },
			set: func(wz *WizardState, v string) { wz.Model = v },
		},
	}
}

// step2Fields 返回步骤2的字段列表（根据 Turbo 模式动态变化）。
func step2Fields(turbo bool) []fieldDef {
	fields := []fieldDef{
		{
			kind: fieldBool, label: "测试模式",
			get: func(wz *WizardState) string {
				if wz.Turbo {
					return "Turbo 模式"
				}
				return "标准模式"
			},
			toggle: func(wz *WizardState, _ bool) { wz.Turbo = !wz.Turbo },
		},
	}

	if !turbo {
		fields = append(fields,
			fieldDef{
				kind: fieldNumber, label: "并发数",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.Concurrency) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.Concurrency = n
					}
				},
			},
			fieldDef{
				kind: fieldNumber, label: "请求总数",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.Count) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.Count = n
					}
				},
			},
			fieldDef{
				kind: fieldNumber, label: "超时(秒)",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.Timeout) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.Timeout = n
					}
				},
			},
			fieldDef{
				kind: fieldBool, label: "流式模式",
				get: func(wz *WizardState) string { return boolLabel(wz.Stream) },
				toggle: func(wz *WizardState, _ bool) { wz.Stream = !wz.Stream },
			},
		)
	} else {
		fields = append(fields,
			fieldDef{
				kind: fieldNumber, label: "初始并发",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.InitConcurrency) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.InitConcurrency = n
					}
				},
			},
			fieldDef{
				kind: fieldNumber, label: "最大并发",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.MaxConcurrency) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.MaxConcurrency = n
					}
				},
			},
			fieldDef{
				kind: fieldNumber, label: "步进值",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.StepSize) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.StepSize = n
					}
				},
			},
			fieldDef{
				kind: fieldNumber, label: "每级请求数",
				get: func(wz *WizardState) string { return strconv.Itoa(wz.LevelRequests) },
				set: func(wz *WizardState, v string) {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.LevelRequests = n
					}
				},
			},
			fieldDef{
				kind: fieldNumber, label: "最低成功率%",
				get: func(wz *WizardState) string { return fmt.Sprintf("%.0f", wz.MinSuccessRate) },
				set: func(wz *WizardState, v string) {
					if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 100 {
						wz.MinSuccessRate = f
					}
				},
			},
		)
	}

	// Prompt 字段（共用）
	promptModes := []string{PromptModeText, PromptModeFile, PromptModeGenerated}
	fields = append(fields,
		fieldDef{
			kind: fieldEnum, label: "输入方式",
			get: func(wz *WizardState) string {
				switch wz.PromptMode {
				case PromptModeFile:
					return "文件"
				case PromptModeGenerated:
					return "按长度生成"
				default:
					return "直接输入"
				}
			},
			toggle: func(wz *WizardState, forward bool) {
				idx := 0
				for i, m := range promptModes {
					if m == wz.PromptMode {
						idx = i
						break
					}
				}
				if forward {
					idx = (idx + 1) % len(promptModes)
				} else {
					idx = (idx - 1 + len(promptModes)) % len(promptModes)
				}
				wz.PromptMode = promptModes[idx]
			},
		},
	)

	// 根据 prompt 模式添加对应字段（在渲染时动态决定）
	fields = append(fields,
		fieldDef{
			kind: fieldText, label: "内容",
			get: func(wz *WizardState) string {
				switch wz.PromptMode {
				case PromptModeFile:
					return wz.PromptFile
				case PromptModeGenerated:
					return strconv.Itoa(wz.PromptLength)
				default:
					return wz.PromptText
				}
			},
			set: func(wz *WizardState, v string) {
				switch wz.PromptMode {
				case PromptModeFile:
					wz.PromptFile = v
				case PromptModeGenerated:
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						wz.PromptLength = n
					}
				default:
					wz.PromptText = v
				}
			},
		},
	)
	return fields
}

// HandleWizardKey 处理向导按键。
func HandleWizardKey(wz *WizardState, msg tea.KeyMsg, client Client) (*WizardState, tea.Cmd, NavAction) {
	nav := NavAction{}
	if wz == nil {
		return wz, nil, NavAction{To: NavTaskList}
	}

	// 当前步骤的字段列表
	var fields []fieldDef
	switch wz.Step {
	case wizardStep1:
		fields = step1Fields()
	case wizardStep2:
		fields = step2Fields(wz.Turbo)
	case wizardStep3:
		// Step 3 只有两个动作：保存、保存并运行
		switch msg.String() {
		case "esc":
			wz.Step = wizardStep2
			wz.FieldIndex = 0
		case "enter":
			// 保存任务
			cfg := wz.BuildTaskConfig()
			var cmd tea.Cmd
			if wz.EditingID != "" {
				cmd = client.UpdateTaskCmd(wz.EditingID, cfg)
			} else {
				cmd = client.CreateTaskCmd(cfg, true) // autoStart
			}
			nav = NavAction{To: NavTaskList}
			return wz, cmd, nav
		case "r":
			// 保存并运行（强制启动，忽略干扰检测）
			cfg := wz.BuildTaskConfig()
			var cmd tea.Cmd
			if wz.EditingID != "" {
				cmd = client.UpdateTaskCmd(wz.EditingID, cfg)
			} else {
				cmd = client.CreateTaskCmd(cfg, true)
			}
			nav = NavAction{To: NavTaskList}
			return wz, cmd, nav
		case "q", "ctrl+c":
			nav = NavAction{To: NavQuit}
		}
		return wz, nil, nav
	}

	maxField := len(fields) - 1

	switch msg.String() {
	case "esc":
		if wz.Step == wizardStep1 {
			nav = NavAction{To: NavTaskList}
		} else {
			wz.Step--
			wz.FieldIndex = 0
		}

	case "tab", "down", "j":
		if wz.FieldIndex < maxField {
			wz.FieldIndex++
		}

	case "shift+tab", "up", "k":
		if wz.FieldIndex > 0 {
			wz.FieldIndex--
		}

	case "left":
		if wz.FieldIndex < len(fields) {
			f := fields[wz.FieldIndex]
			if f.toggle != nil {
				f.toggle(wz, false)
				// 如果切换了 turbo 模式，重置 fieldIndex
				if f.label == "测试模式" {
					wz.FieldIndex = 0
				}
			}
		}

	case "right":
		if wz.FieldIndex < len(fields) {
			f := fields[wz.FieldIndex]
			if f.toggle != nil {
				f.toggle(wz, true)
				if f.label == "测试模式" {
					wz.FieldIndex = 0
				}
			}
		}

	case "enter":
		if wz.FieldIndex == maxField && int(wz.Step) < 2 {
			wz.Step++
			wz.FieldIndex = 0
		} else if wz.FieldIndex < maxField {
			wz.FieldIndex++
		}

	case "backspace":
		if wz.FieldIndex < len(fields) {
			f := fields[wz.FieldIndex]
			if f.set != nil && f.kind == fieldText {
				v := f.get(wz)
				r := []rune(v)
				if len(r) > 0 {
					f.set(wz, string(r[:len(r)-1]))
				}
			}
		}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}

	default:
		// 字符输入
		if len(msg.Runes) > 0 && wz.FieldIndex < len(fields) {
			f := fields[wz.FieldIndex]
			if f.set != nil && (f.kind == fieldText || f.kind == fieldNumber) {
				f.set(wz, f.get(wz)+string(msg.Runes))
			}
		}
	}

	return wz, nil, nav
}

// RenderWizard 渲染三步弹窗向导（overlay 覆盖在后台页面上）。
func RenderWizard(wz *WizardState, bgView string, st Styles, width, height int) string {
	if wz == nil {
		return bgView
	}

	// 暗化背景
	bgLines := strings.Split(bgView, "\n")
	for i, line := range bgLines {
		bgLines[i] = st.Muted.Render(line)
	}

	// 弹窗尺寸
	dialogW := width - 8
	if dialogW > 72 {
		dialogW = 72
	}
	if dialogW < 40 {
		dialogW = 40
	}

	var dialogLines []string

	stepTitles := []string{"1/3 · 基本信息", "2/3 · 测试参数", "3/3 · 确认保存"}
	stepTitle := stepTitles[int(wz.Step)]
	isEdit := wz.EditingID != ""
	action := "新建任务"
	if isEdit {
		action = "编辑任务"
	}
	dialogLines = append(dialogLines, st.SectionHead.Render(fmt.Sprintf("  %s  %s", action, stepTitle)))
	dialogLines = append(dialogLines, "")

	switch wz.Step {
	case wizardStep1:
		fields := step1Fields()
		for i, f := range fields {
			dialogLines = append(dialogLines, renderWizardField(st, f, wz, i == wz.FieldIndex, dialogW-4))
			dialogLines = append(dialogLines, "")
		}
		dialogLines = append(dialogLines, dividerLine(st, dialogW-4))
		hintStyle := st.Muted
		dialogLines = append(dialogLines, hintStyle.Render("  [Tab] 下一项  [↑↓] 切换协议  [Enter] 下一步  [Esc] 取消"))

	case wizardStep2:
		fields := step2Fields(wz.Turbo)
		for i, f := range fields {
			dialogLines = append(dialogLines, renderWizardField(st, f, wz, i == wz.FieldIndex, dialogW-4))
			dialogLines = append(dialogLines, "")
		}
		dialogLines = append(dialogLines, dividerLine(st, dialogW-4))
		dialogLines = append(dialogLines, st.Muted.Render("  [Tab] 下一项  [←→] 切换模式  [Enter] 下一步  [Esc] 返回"))

	case wizardStep3:
		dialogLines = append(dialogLines, renderStep3Summary(wz, st, dialogW-4)...)
		dialogLines = append(dialogLines, "")
		dialogLines = append(dialogLines, dividerLine(st, dialogW-4))
		dialogLines = append(dialogLines, st.Muted.Render("  [Enter] 保存任务   [r] 保存并运行   [Esc] 返回修改"))
	}

	// 构建弹窗框
	innerLines := dialogLines
	boxedLines := make([]string, len(innerLines))
	for i, l := range innerLines {
		lW := lipgloss.Width(l)
		pad := dialogW - 4 - lW
		if pad < 0 {
			pad = 0
		}
		boxedLines[i] = "  " + l + strings.Repeat(" ", pad)
	}

	// 用 lipgloss rounded border 包裹
	inner := strings.Join(boxedLines, "\n")
	box := st.Dialog.Width(dialogW).Render(inner)

	// 将弹窗叠加在背景中间
	boxLines := strings.Split(box, "\n")
	startRow := (height - len(boxLines)) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (width - dialogW) / 2
	if startCol < 0 {
		startCol = 0
	}

	for i, boxLine := range boxLines {
		row := startRow + i
		if row >= len(bgLines) {
			bgLines = append(bgLines, strings.Repeat(" ", width))
		}
		bgLine := []rune(bgLines[row])
		boxRunes := []rune(boxLine)
		// 替换对应列
		for j, r := range boxRunes {
			col := startCol + j
			if col < len(bgLine) {
				bgLine[col] = r
			}
		}
		bgLines[row] = string(bgLine)
	}

	return strings.Join(bgLines, "\n")
}

// renderWizardField 渲染向导的一个字段行。
func renderWizardField(st Styles, f fieldDef, wz *WizardState, active bool, maxW int) string {
	label := padRight(f.label, 12)
	var valueStr string

	if f.get != nil {
		valueStr = f.get(wz)
	}

	// API key 遮蔽显示
	if f.label == "API 密钥" && valueStr != "" {
		valueStr = maskAPIKey(valueStr)
	}

	var renderedValue string
	if active {
		if f.kind == fieldEnum || f.kind == fieldBool {
			renderedValue = st.Ok.Render("● " + valueStr)
		} else {
			renderedValue = st.FieldActive.Width(maxW - 14).Render(valueStr + "█") // 光标
		}
	} else {
		if f.kind == fieldEnum || f.kind == fieldBool {
			renderedValue = st.Muted.Render("○ " + valueStr)
		} else {
			renderedValue = st.FieldIdle.Width(maxW - 14).Render(valueStr)
		}
	}

	return "  " + st.Label.Render(label) + "  " + renderedValue
}

// renderStep3Summary 渲染步骤3的确认内容。
func renderStep3Summary(wz *WizardState, st Styles, innerW int) []string {
	var lines []string
	addRow := func(label, value string) {
		lines = append(lines, "  "+st.Label.Render(padRight(label, 12))+"  "+st.Value.Render(value))
	}

	addRow("任务名称", wz.Name)
	addRow("协议", wz.Protocol)
	endpointDisplay := wz.EndpointURL
	if endpointDisplay == "" {
		endpointDisplay = types.DefaultEndpointURL(wz.Protocol)
	}
	addRow("接口地址", truncate(endpointDisplay, innerW-20))
	addRow("API 密钥", maskAPIKey(wz.APIKey))
	addRow("测试模型", wz.Model)

	if wz.Turbo {
		addRow("测试模式", "Turbo 模式")
		addRow("并发爬坡", fmt.Sprintf("%d → %d  步进 +%d  每级 %d 请求",
			wz.InitConcurrency, wz.MaxConcurrency, wz.StepSize, wz.LevelRequests))
		addRow("停止条件", fmt.Sprintf("成功率 < %.0f%%", wz.MinSuccessRate))
	} else {
		addRow("测试模式", "标准模式")
		addRow("并发数", strconv.Itoa(wz.Concurrency))
		addRow("请求总数", strconv.Itoa(wz.Count))
		addRow("超时", fmt.Sprintf("%ds", wz.Timeout))
		addRow("流式模式", boolLabel(wz.Stream))
	}

	promptDesc := wz.PromptText
	if wz.PromptMode == PromptModeFile {
		promptDesc = "文件: " + wz.PromptFile
	} else if wz.PromptMode == PromptModeGenerated {
		promptDesc = fmt.Sprintf("生成 %d 字符", wz.PromptLength)
	}
	addRow("Prompt", truncate(promptDesc, innerW-20)+fmt.Sprintf(" (长度: %d)", len([]rune(wz.PromptText))))

	lines = append(lines, "")
	lines = append(lines, "  "+st.Muted.Render("保存任务到 ~/.ait/tasks.json  [✓]"))
	lines = append(lines, "")
	lines = append(lines, "  "+st.BtnPrimary.Render("▶  保存任务"))

	return lines
}
