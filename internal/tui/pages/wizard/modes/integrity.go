package modes

import (
	"github.com/yinxulai/ait/internal/i18n"
)

// IntegrityMode Integrity 完整性验证模式
type IntegrityMode struct{}

func (m *IntegrityMode) Name() string {
	return string(ModeIntegrity)
}

func (m *IntegrityMode) Label() string {
	return i18n.T(i18n.KIntegrityMode)
}

func (m *IntegrityMode) NeedsPrompt() bool {
	return false // Integrity 模式由测试套件提供请求内容
}

func (m *IntegrityMode) NeedsStream() bool {
	return false // Integrity 模式不支持流式配置
}

func (m *IntegrityMode) Fields(state ModeState) []FieldDef {
	return []FieldDef{
		{
			Kind:  FieldText,
			Label: i18n.T(i18n.KWzIntegritySuite),
			Get: func(s ModeState) string {
				return s.GetIntegritySuite()
			},
			Set: func(s ModeState, v string) {
				s.SetIntegritySuite(v)
			},
		},
		{
			Kind:  FieldBool,
			Label: i18n.T(i18n.KWzFailFast),
			Get: func(s ModeState) string {
				return boolLabel(s.GetIntegrityFailFast())
			},
			Toggle: func(s ModeState, _ bool) {
				s.SetIntegrityFailFast(!s.GetIntegrityFailFast())
			},
		},
	}
}
