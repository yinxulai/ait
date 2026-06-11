package wizard

import (
	"strings"

	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server/types"
)

// GetStep1Fields 返回步骤1的字段列表（基本信息）
func GetStep1Fields() []fieldDef {
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
