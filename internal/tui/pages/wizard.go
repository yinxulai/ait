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
	ScrollOff  int
}

// NewWizardState 创建新建任务向导状态（使用默认值）。
func NewWizardState() *WizardState {
	return &WizardState{
		Step:            wizardStep1,
		Protocol:        types.ProtocolOpenAICompletions,
		Concurrency:     10,
		Count:           100,
		Timeout:         30,
		InitConcurrency: 1,
		MaxConcurrency:  50,
		StepSize:        2,
		LevelRequests:   30,
		MinSuccessRate:  90,
		PromptMode:      PromptModeText,
	}
}

// NewWizardStateEdit 创建编辑任务向导状态（预填任务数据，零值字段沿用默认值）。
func NewWizardStateEdit(t *types.TaskDefinition) *WizardState {
	if t == nil {
		return NewWizardState()
	}
	wz := NewWizardState()
	inp := t.Input
	tc := inp.TurboConfig

	wz.EditingID = t.ID
	wz.Name = t.Name
	wz.Protocol = types.NormalizeProtocol(inp.Protocol)
	wz.EndpointURL = inp.EndpointURL
	wz.APIKey = inp.ApiKey
	wz.Model = inp.Model
	wz.Turbo = inp.Turbo
	wz.Stream = inp.Stream
	wz.PromptText = inp.PromptText
	wz.PromptFile = inp.PromptFile
	wz.PromptLength = inp.PromptLength
	if inp.PromptMode != "" {
		wz.PromptMode = inp.PromptMode
	}
	if inp.Concurrency > 0 {
		wz.Concurrency = inp.Concurrency
	}
	if inp.Count > 0 {
		wz.Count = inp.Count
	}
	if inp.Timeout > 0 {
		wz.Timeout = int(inp.Timeout.Seconds())
	}
	if tc.InitConcurrency > 0 {
		wz.InitConcurrency = tc.InitConcurrency
	}
	if tc.MaxConcurrency > 0 {
		wz.MaxConcurrency = tc.MaxConcurrency
	}
	if tc.StepSize > 0 {
		wz.StepSize = tc.StepSize
	}
	if tc.LevelRequests > 0 {
		wz.LevelRequests = tc.LevelRequests
	}
	if tc.MinSuccessRate > 0 {
		wz.MinSuccessRate = tc.MinSuccessRate * 100
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

// intField 构造一个整数输入字段（值 > 0 时才写入）。
func intField(label string, get func(*WizardState) int, set func(*WizardState, int)) fieldDef {
	return fieldDef{
		kind:  fieldNumber,
		label: label,
		get:   func(wz *WizardState) string { return strconv.Itoa(get(wz)) },
		set: func(wz *WizardState, v string) {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				set(wz, n)
			}
		},
	}
}

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
			intField("并发数", func(wz *WizardState) int { return wz.Concurrency }, func(wz *WizardState, n int) { wz.Concurrency = n }),
			intField("请求总数", func(wz *WizardState) int { return wz.Count }, func(wz *WizardState, n int) { wz.Count = n }),
			intField("超时(秒)", func(wz *WizardState) int { return wz.Timeout }, func(wz *WizardState, n int) { wz.Timeout = n }),
		)
	} else {
		fields = append(fields,
			intField("初始并发", func(wz *WizardState) int { return wz.InitConcurrency }, func(wz *WizardState, n int) { wz.InitConcurrency = n }),
			intField("最大并发", func(wz *WizardState) int { return wz.MaxConcurrency }, func(wz *WizardState, n int) { wz.MaxConcurrency = n }),
			intField("步进值", func(wz *WizardState) int { return wz.StepSize }, func(wz *WizardState, n int) { wz.StepSize = n }),
			intField("每级请求数", func(wz *WizardState) int { return wz.LevelRequests }, func(wz *WizardState, n int) { wz.LevelRequests = n }),
			fieldDef{
				kind:  fieldNumber,
				label: "最低成功率",
				get:   func(wz *WizardState) string { return fmt.Sprintf("%.0f", wz.MinSuccessRate) },
				set: func(wz *WizardState, v string) {
					if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 100 {
						wz.MinSuccessRate = f
					}
				},
			},
		)
	}

	// 流式模式：与测试模式无关，两种模式均可配置
	fields = append(fields, fieldDef{
		kind:   fieldBool,
		label:  "流式模式",
		get:    func(wz *WizardState) string { return boolLabel(wz.Stream) },
		toggle: func(wz *WizardState, _ bool) { wz.Stream = !wz.Stream },
	})

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
			wz.ScrollOff = 0
		case "up", "k":
			wz.ScrollOff--
		case "down", "j":
			wz.ScrollOff++
		case "pgup":
			wz.ScrollOff -= 5
		case "pgdown", " ":
			wz.ScrollOff += 5
		case "home":
			wz.ScrollOff = 0
		case "end":
			wz.ScrollOff = 1 << 30
		case "enter", "r":
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
			wz.ScrollOff = 0
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
					wz.ScrollOff = 0
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
					wz.ScrollOff = 0
				}
			}
		}

	case "enter":
		if wz.FieldIndex == maxField && int(wz.Step) < 2 {
			wz.Step++
			wz.FieldIndex = 0
			wz.ScrollOff = 0
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

// RenderWizard 渲染三步创建/编辑任务页。
func RenderWizard(wz *WizardState, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}
	if wz == nil {
		return renderTooSmall(st, width, height)
	}

	l := PageLayout{
		CtxItems:    wizardContextItems(wz.Step),
		FooterParts: []string{"[q] 退出", "◆ AIT  v0.1"},
	}

	content := buildWizardPageContent(wz, st, ContentWidth(width), l.ContentHeight(height))
	return l.Assemble(wrapPanel(st, content, width), st, width)
}

func buildWizardPageContent(wz *WizardState, st Styles, width, maxH int) string {
	stepTitles := []string{"基本信息", "测试参数", "确认保存"}
	stepDescs := []string{
		"配置任务名称、模型协议和连接信息。",
		"选择压测模式，并补全并发与 Prompt 参数。",
		"保存前快速检查关键配置。",
	}
	action := "创建任务"
	if wz.EditingID != "" {
		action = "编辑任务"
	}
	stepTitle := stepTitles[int(wz.Step)]
	stepDesc := stepDescs[int(wz.Step)]

	titleLeft := st.SectionHead.Render(action)
	titleRight := st.Muted.Render(fmt.Sprintf("步骤 %d/3 · %s", int(wz.Step)+1, stepTitle))
	var topLines []string
	if lipgloss.Width(titleLeft)+lipgloss.Width(titleRight)+2 <= width {
		topLines = append(topLines, titleLeft+strings.Repeat(" ", width-lipgloss.Width(titleLeft)-lipgloss.Width(titleRight))+titleRight)
	} else {
		topLines = append(topLines, titleLeft, titleRight)
	}
	if maxH >= 8 {
		for _, line := range wrapText(stepDesc, width) {
			topLines = append(topLines, st.Muted.Render(line))
		}
	}
	if maxH >= 10 && width >= 46 {
		topLines = append(topLines, renderWizardStepStrip(wz.Step))
	}

	bottomCount := 1
	showBottomDivider := maxH >= 6
	if showBottomDivider {
		bottomCount = 2
	}

	// 为 body 保留最少 5 行空间
	minBodyH := 5
	availableForContent := maxH - bottomCount
	maxTopH := maxInt(1, availableForContent-minBodyH)

	// 限制 topLines 大小
	if len(topLines) > maxTopH {
		topLines = topLines[:maxTopH]
	}
	if maxH >= 6 {
		topLines = append(topLines, dividerLine(st, width))
	}

	bodyLines, focusLine := buildWizardBody(wz, st, width)
	bodyH := maxInt(1, availableForContent-len(topLines))
	offset := 0
	if wz.Step == wizardStep3 {
		offset = clampInt(wz.ScrollOff, 0, maxInt(0, len(bodyLines)-bodyH))
	} else if focusLine >= 0 {
		offset = ensureVisibleOffset(focusLine, len(bodyLines), 0, bodyH)
	}
	end := minInt(len(bodyLines), offset+bodyH)
	visibleBody := append([]string{}, bodyLines[offset:end]...)
	for len(visibleBody) < bodyH {
		visibleBody = append(visibleBody, "")
	}

	lines := append([]string{}, topLines...)
	lines = append(lines, visibleBody...)
	if showBottomDivider {
		lines = append(lines, dividerLine(st, width))
	}
	lines = append(lines, st.Muted.Render(truncate(wizardStatusText(wz, offset, end, len(bodyLines), bodyH), width)))

	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func buildWizardBody(wz *WizardState, st Styles, contentW int) ([]string, int) {
	var lines []string
	focusLine := -1

	// appendField 将字段渲染结果按行展开追加，因为 FieldActive/FieldIdle 带 Border
	// 会产生 3 行输出（顶部边框 + 内容 + 底部边框），必须逐行记录才能正确计算高度。
	appendField := func(rendered string, focused bool) {
		if focused {
			focusLine = len(lines)
		}
		for _, l := range strings.Split(rendered, "\n") {
			lines = append(lines, l)
		}
	}

	switch wz.Step {
	case wizardStep1:
		fields := step1Fields()
		for i, f := range fields {
			appendField(renderWizardField(st, f, wz, i == wz.FieldIndex, contentW), i == wz.FieldIndex)
		}

	case wizardStep2:
		fields := step2Fields(wz.Turbo)
		for i, f := range fields {
			if f.label == "输入方式" {
				lines = append(lines, "", st.Muted.Render("Prompt 配置"))
			}
			appendField(renderWizardField(st, f, wz, i == wz.FieldIndex, contentW), i == wz.FieldIndex)
		}

	case wizardStep3:
		lines = append(lines, renderStep3Summary(wz, st, contentW)...)
	}

	return lines, focusLine
}

// renderWizardField 渲染向导的一个字段行。
func renderWizardField(st Styles, f fieldDef, wz *WizardState, active bool, maxW int) string {
	var valueStr string

	if f.get != nil {
		valueStr = f.get(wz)
	}

	// API key 遮蔽显示
	if f.label == "API 密钥" && valueStr != "" {
		valueStr = maskAPIKey(valueStr)
	}

	// FieldActive/Idle: Width(n) = 内容区宽度（在 padding/border 之内）
	// 总渲染宽度 = n + padding(2) + border(2) = n + 4
	// Line1 = label(14) + space(1) + (n+4) = n + 19 ≤ maxW → n = maxW - 19
	fieldW := maxInt(10, maxW-19)
	valueStyle := st.Value
	if valueStr == "" && !active {
		valueStr = "未填写"
		valueStyle = st.Muted
	}

	// Width(fieldW) 是内容区宽度，padding 在其外侧叠加，文字区即为 fieldW
	// 激活时保留 1 列给光标 █，非激活可用满 fieldW
	if f.kind == fieldEnum || f.kind == fieldBool {
		if active {
			valueStr = "‹ " + valueStr + " ›"
		}
		valueStr = truncate(valueStr, maxInt(4, fieldW))
	} else {
		if active {
			valueStr = fitTail(valueStr, maxInt(1, fieldW-1)) + "█"
		} else {
			valueStr = fitTail(valueStr, maxInt(1, fieldW))
		}
	}

	fieldStyle := st.FieldIdle
	if active {
		fieldStyle = st.FieldActive
	}

	renderedValue := fieldStyle.Width(fieldW).Render(valueStyle.Render(valueStr))
	// 用 JoinHorizontal 而非字符串拼接：renderedValue 有 3 行（上边框/内容/下边框），
	// 直接 + 只有第一行有 label 前缀，后两行会从列 0 开始，导致布局混乱。
	// JoinHorizontal(Top, ...) 会将 label 块和 field 块按顶部对齐水平拼接，
	// label 块高度自动补齐到与 field 相同（3 行），布局整齐。
	labelBlock := lipgloss.NewStyle().Width(15).Render(st.Label.Render(wizardFieldLabel(f, wz)))
	return lipgloss.JoinHorizontal(lipgloss.Top, labelBlock, renderedValue)
}

// renderStep3Summary 渲染步骤3的确认内容。
func renderStep3Summary(wz *WizardState, st Styles, innerW int) []string {
	var lines []string
	addRow := func(label, value string, valueStyle lipgloss.Style) {
		appendWizardSummaryRow(&lines, st, label, value, innerW, valueStyle)
	}

	lines = append(lines, st.SectionHead.Render("配置概览"))
	addRow("任务名称", wizardFallback(wz.Name, "未命名任务"), st.Value)
	addRow("协议", wz.Protocol, st.Value)
	endpointDisplay := wz.EndpointURL
	if endpointDisplay == "" {
		endpointDisplay = types.DefaultEndpointURL(wz.Protocol)
	}
	addRow("接口地址", endpointDisplay, st.Value)
	addRow("API 密钥", wizardFallback(maskAPIKey(wz.APIKey), "未填写"), st.Value)
	addRow("测试模型", wizardFallback(wz.Model, "未填写"), st.Value)

	lines = append(lines, "", st.SectionHead.Render("执行参数"))
	if wz.Turbo {
		addRow("测试模式", "Turbo 模式", st.Value)
		addRow("并发爬坡", fmt.Sprintf("%d → %d · 步进 +%d · 每级 %d 请求",
			wz.InitConcurrency, wz.MaxConcurrency, wz.StepSize, wz.LevelRequests), st.Value)
		addRow("停止条件", fmt.Sprintf("成功率 < %.0f%%", wz.MinSuccessRate), st.Value)
	} else {
		addRow("测试模式", "标准模式", st.Value)
		addRow("并发数", strconv.Itoa(wz.Concurrency), st.Value)
		addRow("请求总数", strconv.Itoa(wz.Count), st.Value)
		addRow("超时", fmt.Sprintf("%ds", wz.Timeout), st.Value)
	}
	addRow("流式模式", boolLabel(wz.Stream), st.Value)

	lines = append(lines, "", st.SectionHead.Render("Prompt"))
	addRow("输入方式", wizardPromptModeLabel(wz.PromptMode), st.Value)
	promptDesc := promptSummary(wz.PromptMode, wz.PromptText, wz.PromptFile, wz.PromptLength)
	addRow("内容摘要", wizardFallback(promptDesc, "未填写"), st.Value)
	if wz.PromptMode == PromptModeText {
		addRow("字符数", strconv.Itoa(len([]rune(wz.PromptText))), st.Muted)
	} else if wz.PromptMode == PromptModeGenerated {
		addRow("目标长度", strconv.Itoa(wz.PromptLength), st.Muted)
	}

	lines = append(lines, "", st.Muted.Render("保存位置: ~/.ait/tasks.json"))

	return lines
}

func renderWizardStepStrip(step wizardStep) string {
	active := lipgloss.NewStyle().Background(colorPink).Foreground(colorWhite).Bold(true).Padding(0, 1)
	done := lipgloss.NewStyle().Background(colorCyan).Foreground(lipgloss.Color("233")).Bold(true).Padding(0, 1)
	idle := lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(colorMuted).Padding(0, 1)
	labels := []string{"1 基本信息", "2 测试参数", "3 确认保存"}
	parts := make([]string, 0, len(labels))
	for i, label := range labels {
		switch {
		case i < int(step):
			parts = append(parts, done.Render("✓ "+label))
		case i == int(step):
			parts = append(parts, active.Render(label))
		default:
			parts = append(parts, idle.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

func wizardFieldLabel(f fieldDef, wz *WizardState) string {
	if f.label != "内容" {
		return f.label
	}
	switch wz.PromptMode {
	case PromptModeFile:
		return "文件路径"
	case PromptModeGenerated:
		return "生成长度"
	default:
		return "Prompt"
	}
}

func wizardPromptModeLabel(mode string) string {
	switch mode {
	case PromptModeFile:
		return "文件"
	case PromptModeGenerated:
		return "按长度生成"
	default:
		return "直接输入"
	}
}

func wizardFallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func fitTail(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	runes := []rune(s)
	width := 0
	for i := len(runes) - 1; i >= 0; i-- {
		rw := lipgloss.Width(string(runes[i]))
		if width+rw > maxW-1 {
			return "…" + string(runes[i+1:])
		}
		width += rw
	}
	return s
}

func appendWizardSummaryRow(lines *[]string, st Styles, label, value string, width int, valueStyle lipgloss.Style) {
	labelW := 14
	contentW := maxInt(8, width-labelW-1)
	segments := wrapText(value, contentW)
	if len(segments) == 0 {
		segments = []string{""}
	}
	*lines = append(*lines, st.Label.Render(padRight(label, labelW))+" "+valueStyle.Render(segments[0]))
	indent := strings.Repeat(" ", labelW+1)
	for _, segment := range segments[1:] {
		*lines = append(*lines, indent+valueStyle.Render(segment))
	}
}

func wizardContextItems(step wizardStep) []ContextBarItem {
	switch step {
	case wizardStep1:
		return CtxBar_Wizard_Step1()
	case wizardStep2:
		return CtxBar_Wizard_Step2()
	default:
		return CtxBar_Wizard_Step3()
	}
}

func wizardStatusText(wz *WizardState, offset, end, scrollTotal, visible int) string {
	if wz.Step == wizardStep3 {
		if scrollTotal <= 0 {
			return "暂无确认项"
		}
		if scrollTotal > visible {
			return fmt.Sprintf("确认项 %d-%d/%d", offset+1, end, scrollTotal)
		}
		return fmt.Sprintf("共 %d 项待确认", scrollTotal)
	}
	var fieldTotal int
	switch wz.Step {
	case wizardStep1:
		fieldTotal = len(step1Fields())
	case wizardStep2:
		fieldTotal = len(step2Fields(wz.Turbo))
	}
	if fieldTotal <= 0 {
		return "暂无配置项"
	}
	return fmt.Sprintf("当前字段 %d/%d", wz.FieldIndex+1, fieldTotal)
}
