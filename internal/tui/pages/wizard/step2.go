package wizard

import (
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/tui/pages/wizard/modes"
)

// GetStep2Fields 返回步骤2的字段列表（根据模式动态变化）
func GetStep2Fields(wz *State) []fieldDef {
	// 1. 模式选择字段
	fields := []fieldDef{
		{
			kind: fieldBool, label: i18n.T(i18n.KWzTestMode),
			get: func(wz *State) string {
				return modes.GetMode(wz.Integrity, wz.Turbo).Label()
			},
			toggle: func(wz *State, _ bool) {
				// 循环切换：Standard → Turbo → Integrity → Standard
				if wz.Integrity {
					wz.Integrity = false
					wz.Turbo = false
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

	// 2. 添加当前模式的特有字段
	currentMode := modes.GetMode(wz.Integrity, wz.Turbo)
	modeFields := currentMode.Fields(wz)
	
	// 将 modes.FieldDef 转换为 fieldDef
	for _, mf := range modeFields {
		fields = append(fields, fieldDef{
			kind:               fieldKind(mf.Kind),
			label:              mf.Label,
			get:                func(state *State) string { return mf.Get(state) },
			getRaw:             func(state *State) string { 
				if mf.GetRaw != nil {
					return mf.GetRaw(state)
				}
				return mf.Get(state)
			},
			set:                func(state *State, v string) { mf.Set(state, v) },
			toggle:             func(state *State, forward bool) { 
				if mf.Toggle != nil {
					mf.Toggle(state, forward)
				}
			},
			password:           mf.Password,
			triggersFieldReset: mf.TriggersFieldReset,
		})
	}

	// 3. 如果当前模式不需要 Prompt 配置，直接返回
	if !currentMode.NeedsPrompt() {
		return fields
	}

	// 4. 添加 Stream 配置（Standard 和 Turbo 模式）
	if currentMode.NeedsStream() {
		fields = append(fields, GetStreamField())
	}

	// 5. 添加 Prompt 配置（Standard 和 Turbo 模式）
	fields = append(fields, GetPromptFields()...)

	return fields
}

// GetModeHint 返回当前模式的提示文本
func GetModeHint(wz *State) string {
	switch {
	case wz.Integrity:
		return i18n.T(i18n.KWzIntegrityModeLabel)
	case wz.Turbo:
		return i18n.T(i18n.KWzTurboModeLabel)
	default:
		return i18n.T(i18n.KWzSelectModeHint)
	}
}

// GetPromptHint 返回当前 Prompt 模式的提示文本
func GetPromptHint(wz *State) string {
	switch wz.PromptMode {
	case PromptModeText:
		return i18n.T(i18n.KWzHintDirect)
	case PromptModeFile:
		return i18n.T(i18n.KWzHintFile)
	case PromptModeGenerated:
		return i18n.T(i18n.KWzHintCacheToken)
	case PromptModeRaw:
		return i18n.T(i18n.KWzHintRaw)
	default:
		return ""
	}
}
