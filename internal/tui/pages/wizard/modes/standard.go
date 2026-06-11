package modes

import (
	"strconv"

	"github.com/yinxulai/ait/internal/i18n"
)

// StandardMode Standard 标准压测模式
type StandardMode struct{}

func (m *StandardMode) Name() string {
	return string(ModeStandard)
}

func (m *StandardMode) Label() string {
	return i18n.T(i18n.KWzStandardMode)
}

func (m *StandardMode) NeedsPrompt() bool {
	return true
}

func (m *StandardMode) NeedsStream() bool {
	return true
}

func (m *StandardMode) Fields(state ModeState) []FieldDef {
	return []FieldDef{
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzConcurrency),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetConcurrency())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetConcurrency(n)
				}
			},
		},
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzTotalRequests),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetCount())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetCount(n)
				}
			},
		},
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzTimeoutSecs),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetTimeout())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetTimeout(n)
				}
			},
		},
	}
}
