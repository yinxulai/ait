package modes

import (
	"github.com/yinxulai/ait/internal/i18n"
)

// ModeType 测试模式类型
type ModeType string

const (
	ModeStandard  ModeType = "standard"
	ModeTurbo     ModeType = "turbo"
	ModeIntegrity ModeType = "integrity"
)

// Mode 测试模式接口
type Mode interface {
	// Name 返回模式名称
	Name() string
	
	// Label 返回模式的国际化标签
	Label() string
	
	// Fields 返回该模式特有的字段定义
	Fields(state ModeState) []FieldDef
	
	// NeedsPrompt 是否需要 Prompt 配置
	NeedsPrompt() bool
	
	// NeedsStream 是否需要 Stream 配置
	NeedsStream() bool
}

// ModeState 模式状态访问接口
type ModeState interface {
	// Standard 模式
	GetConcurrency() int
	SetConcurrency(int)
	GetCount() int
	SetCount(int)
	GetTimeout() int
	SetTimeout(int)
	
	// Turbo 模式
	GetInitConcurrency() int
	SetInitConcurrency(int)
	GetMaxConcurrency() int
	SetMaxConcurrency(int)
	GetStepSize() int
	SetStepSize(int)
	GetLevelRequests() int
	SetLevelRequests(int)
	GetMinSuccessRate() float64
	SetMinSuccessRate(float64)
	
	// Integrity 模式
	GetIntegritySuite() string
	SetIntegritySuite(string)
	GetIntegrityFailFast() bool
	SetIntegrityFailFast(bool)
}

// FieldDef 字段定义（简化版，避免循环依赖）
type FieldDef struct {
	Kind               FieldKind
	Label              string
	Get                func(ModeState) string
	GetRaw             func(ModeState) string
	Set                func(ModeState, string)
	Toggle             func(ModeState, bool)
	Password           bool
	TriggersFieldReset bool
}

// FieldKind 字段类型
type FieldKind int

const (
	FieldText   FieldKind = iota // 自由文本输入
	FieldNumber                  // 数字
	FieldBool                    // 布尔开关
	FieldEnum                    // 枚举循环
)

// GetMode 根据状态返回当前激活的模式
func GetMode(isIntegrity, isTurbo bool) Mode {
	switch {
	case isIntegrity:
		return &IntegrityMode{}
	case isTurbo:
		return &TurboMode{}
	default:
		return &StandardMode{}
	}
}

// boolLabel 返回布尔值的国际化标签
func boolLabel(b bool) string {
	if b {
		return i18n.T(i18n.KEnabled)
	}
	return i18n.T(i18n.KDisabled)
}
