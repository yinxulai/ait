package wizard

import (
	"strconv"

	"github.com/yinxulai/ait/internal/i18n"
)

// GetPromptFields 返回 Prompt 配置字段（Standard 和 Turbo 模式共用）
func GetPromptFields() []fieldDef {
	promptModes := []string{PromptModeText, PromptModeFile, PromptModeGenerated, PromptModeRaw}
	
	return []fieldDef{
		{
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
		{
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
	}
}

// GetStreamField 返回 Stream 配置字段（Standard 和 Turbo 模式共用）
func GetStreamField() fieldDef {
	return fieldDef{
		kind:   fieldBool,
		label:  i18n.T(i18n.KWzStreamMode),
		get:    func(wz *State) string { return boolLabel(wz.Stream) },
		toggle: func(wz *State, _ bool) { wz.Stream = !wz.Stream },
	}
}
