package modes

import (
	"fmt"
	"strconv"

	"github.com/yinxulai/ait/internal/i18n"
)

// TurboMode Turbo 渐进式压测模式
type TurboMode struct{}

func (m *TurboMode) Name() string {
	return string(ModeTurbo)
}

func (m *TurboMode) Label() string {
	return i18n.T(i18n.KTurboMode)
}

func (m *TurboMode) NeedsPrompt() bool {
	return true
}

func (m *TurboMode) NeedsStream() bool {
	return true
}

func (m *TurboMode) Fields(state ModeState) []FieldDef {
	return []FieldDef{
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzInitConc),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetInitConcurrency())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetInitConcurrency(n)
				}
			},
		},
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzMaxConc),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetMaxConcurrency())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetMaxConcurrency(n)
				}
			},
		},
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzStepSize),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetStepSize())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetStepSize(n)
				}
			},
		},
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzLevelReqs),
			Get: func(s ModeState) string {
				return strconv.Itoa(s.GetLevelRequests())
			},
			Set: func(s ModeState, v string) {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					s.SetLevelRequests(n)
				}
			},
		},
		{
			Kind:  FieldNumber,
			Label: i18n.T(i18n.KWzMinSuccessRate),
			Get: func(s ModeState) string {
				return fmt.Sprintf("%.0f", s.GetMinSuccessRate())
			},
			Set: func(s ModeState, v string) {
				if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 100 {
					s.SetMinSuccessRate(f)
				}
			},
		},
	}
}
