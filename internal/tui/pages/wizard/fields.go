package wizard

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server/types"
)

// fieldDef 向导字段定义
type fieldDef struct {
	kind  fieldKind
	label string
	// 获取当前值（字符串形式），用于显示；可能包含占位默认值
	get func(wz *State) string
	// 获取实际存储值（用于编辑操作）；若为 nil 则退回到 get
	getRaw func(wz *State) string
	// 设置文本值
	set func(wz *State, v string)
	// 枚举/布尔切换
	toggle func(wz *State, forward bool)
	// password 为 true 时以密码模式显示输入
	password bool
	// triggersFieldReset 为 true 时切换后重置字段列表索引
	triggersFieldReset bool
}

type fieldKind int

const (
	fieldText   fieldKind = iota // 自由文本输入
	fieldNumber                  // 数字
	fieldBool                    // 布尔开关
	fieldEnum                    // 枚举循环
)

// intField 构造一个整数输入字段（值 > 0 时才写入）。
func intField(label string, get func(*State) int, set func(*State, int)) fieldDef {
	return fieldDef{
		kind:  fieldNumber,
		label: label,
		get:   func(wz *State) string { return strconv.Itoa(get(wz)) },
		set: func(wz *State, v string) {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				set(wz, n)
			}
		},
	}
}

// stringField 构造一个字符串输入字段。
func stringField(label string, get func(*State) string, set func(*State, string)) fieldDef {
	return fieldDef{
		kind:  fieldText,
		label: label,
		get:   func(wz *State) string { return get(wz) },
		set:   func(wz *State, v string) { set(wz, v) },
	}
}

// boolLabel 返回布尔值的国际化标签。
func boolLabel(b bool) string {
	if b {
		return i18n.T(i18n.KEnabled)
	}
	return i18n.T(i18n.KDisabled)
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
			kind: fieldText, label: i18n.T(i18n.KWzTaskName),
			get: func(wz *State) string { return wz.Name },
			set: func(wz *State, v string) { wz.Name = v },
		},
		{
			kind: fieldEnum, label: i18n.T(i18n.KWzProtocol),
			get: func(wz *State) string { return wz.Protocol },
			toggle: func(wz *State, forward bool) {
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
				// 协议变化后直接填入默认 endpoint，避免用户看到空值。
				wz.EndpointURL = types.DefaultEndpointURL(wz.Protocol)
			},
		},
		{
			kind: fieldText, label: i18n.T(i18n.KWzEndpoint),
			get: func(wz *State) string {
				if wz.EndpointURL != "" {
					return wz.EndpointURL
				}
				return types.DefaultEndpointURL(wz.Protocol)
			},
			getRaw: func(wz *State) string {
				if strings.TrimSpace(wz.EndpointURL) != "" {
					return wz.EndpointURL
				}
				return types.DefaultEndpointURL(wz.Protocol)
			},
			set: func(wz *State, v string) { wz.EndpointURL = v },
		},
		{
			kind: fieldText, label: i18n.T(i18n.KWzAPIKey),
			get:      func(wz *State) string { return wz.APIKey },
			set:      func(wz *State, v string) { wz.APIKey = v },
			password: true,
		},
		{
			kind: fieldText, label: i18n.T(i18n.KWzTestModel),
			get: func(wz *State) string { return wz.Model },
			set: func(wz *State, v string) { wz.Model = v },
		},
	}
}

// step2Fields 返回步骤2的字段列表（根据模式动态变化）。
func step2Fields(wz *State) []fieldDef {
	fields := []fieldDef{
		{
			kind: fieldBool, label: i18n.T(i18n.KWzTestMode),
			get: func(wz *State) string {
				switch {
				case wz.Integrity:
					return i18n.T(i18n.KIntegrityMode)
				case wz.Turbo:
					return i18n.T(i18n.KTurboMode)
				default:
					return i18n.T(i18n.KWzStandardMode)
				}
			},
			toggle: func(wz *State, _ bool) {
				// 循环切换：Standard → Turbo → Integrity → Standard
				if wz.Integrity {
					wz.Integrity = false
					// 回到 Standard
				} else if wz.Turbo {
					wz.Turbo = false
					wz.Integrity = true
				} else {
					wz.Turbo = true
				}
			},
			triggersFieldReset: true,
		},
	}

	// 根据模式添加对应字段
	switch {
	case wz.Integrity:
		fields = append(fields,
			stringField(i18n.T(i18n.KWzIntegritySuite), func(wz *State) string { return wz.IntegritySuite }, func(wz *State, v string) { wz.IntegritySuite = v }),
			fieldDef{
				kind:   fieldBool,
				label:  i18n.T(i18n.KWzFailFast),
				get:    func(wz *State) string { return boolLabel(wz.IntegrityFailFast) },
				toggle: func(wz *State, _ bool) { wz.IntegrityFailFast = !wz.IntegrityFailFast },
			},
		)
	case wz.Turbo:
		fields = append(fields,
			intField(i18n.T(i18n.KWzInitConc), func(wz *State) int { return wz.InitConcurrency }, func(wz *State, n int) { wz.InitConcurrency = n }),
			intField(i18n.T(i18n.KWzMaxConc), func(wz *State) int { return wz.MaxConcurrency }, func(wz *State, n int) { wz.MaxConcurrency = n }),
			intField(i18n.T(i18n.KWzStepSize), func(wz *State) int { return wz.StepSize }, func(wz *State, n int) { wz.StepSize = n }),
			intField(i18n.T(i18n.KWzLevelReqs), func(wz *State) int { return wz.LevelRequests }, func(wz *State, n int) { wz.LevelRequests = n }),
			fieldDef{
				kind:  fieldNumber,
				label: i18n.T(i18n.KWzMinSuccessRate),
				get:   func(wz *State) string { return fmt.Sprintf("%.0f", wz.MinSuccessRate) },
				set: func(wz *State, v string) {
					if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 100 {
						wz.MinSuccessRate = f
					}
				},
			},
		)
	default: // Standard
		fields = append(fields,
			intField(i18n.T(i18n.KWzConcurrency), func(wz *State) int { return wz.Concurrency }, func(wz *State, n int) { wz.Concurrency = n }),
			intField(i18n.T(i18n.KWzTotalRequests), func(wz *State) int { return wz.Count }, func(wz *State, n int) { wz.Count = n }),
			intField(i18n.T(i18n.KWzTimeoutSecs), func(wz *State) int { return wz.Timeout }, func(wz *State, n int) { wz.Timeout = n }),
		)
	}

	// Integrity 模式不需要 Prompt 和 Stream 配置（由测试套件决定）
	if wz.Integrity {
		return fields
	}

	// 流式模式：Standard 和 Turbo 模式可配置
	fields = append(fields, fieldDef{
		kind:   fieldBool,
		label:  i18n.T(i18n.KWzStreamMode),
		get:    func(wz *State) string { return boolLabel(wz.Stream) },
		toggle: func(wz *State, _ bool) { wz.Stream = !wz.Stream },
	})

	// Prompt 字段（Standard 和 Turbo 模式共用）
	promptModes := []string{PromptModeText, PromptModeFile, PromptModeGenerated, PromptModeRaw}
	fields = append(fields,
		fieldDef{
			kind: fieldEnum, label: i18n.T(i18n.KWzInputMode),
			get: func(wz *State) string {
				switch wz.PromptMode {
				case PromptModeFile:
					return i18n.T(i18n.KWzInputFile)
				case PromptModeGenerated:
					return i18n.T(i18n.KWzInputGenerated)
				case PromptModeRaw:
					return i18n.T(i18n.KWzInputRaw)
				default:
					return i18n.T(i18n.KWzInputDirect)
				}
			},
			toggle: func(wz *State, forward bool) {
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
				if wz.PromptMode == PromptModeGenerated && wz.PromptLength <= 0 {
					wz.PromptLength = 4096
				}
			},
		},
	)

	// 根据 prompt 模式添加对应字段（在渲染时动态决定）
	fields = append(fields,
		fieldDef{
			kind: fieldText, label: i18n.T(i18n.KWzPromptContent),
			get: func(wz *State) string {
				switch wz.PromptMode {
				case PromptModeFile:
					return wz.PromptFile
				case PromptModeGenerated:
					return strconv.Itoa(wz.PromptLength)
				default:
					return wz.PromptText
				}
			},
			set: func(wz *State, v string) {
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

// WizardFieldLabel 返回字段的标签（根据模式动态变化）。
func wizardFieldLabel(f fieldDef, wz *State) string {
	if f.label != i18n.T(i18n.KWzPromptContent) {
		return f.label
	}
	switch wz.PromptMode {
	case PromptModeFile:
		return i18n.T(i18n.KWzFileSummary)
	case PromptModeGenerated:
		return i18n.T(i18n.KWzRAWBody)
	case PromptModeRaw:
		return i18n.T(i18n.KWzJSONBody)
	default:
		return i18n.T(i18n.KWzPromptLabelShort)
	}
}

// loadInputForField 将字段的当前值加载到 wz.Input，并将光标移到末尾。
func loadInputForField(wz *State, f fieldDef) {
	rawVal := ""
	if f.getRaw != nil {
		rawVal = f.getRaw(wz)
	} else if f.get != nil {
		rawVal = f.get(wz)
	}
	if f.password {
		wz.Input.EchoMode = textinput.EchoPassword
	} else {
		wz.Input.EchoMode = textinput.EchoNormal
	}
	wz.Input.SetValue(rawVal)
	wz.Input.CursorEnd()
}

// LoadCurrentFieldInput 根据当前 Step/FieldIndex 重新加载 Input。
// 在字段切换或步骤切换后调用。
func (wz *State) LoadCurrentFieldInput() {
	var fields []fieldDef
	switch wz.Step {
	case Step1:
		fields = step1Fields()
	case Step2:
		fields = step2Fields(wz)
	default:
		return
	}
	if wz.FieldIndex < len(fields) {
		f := fields[wz.FieldIndex]
		if f.kind == fieldText || f.kind == fieldNumber {
			loadInputForField(wz, f)
		}
	}
}
