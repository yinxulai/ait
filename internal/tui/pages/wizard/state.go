package wizard

import (
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// Prompt 模式常量
const (
	PromptModeText      = "text"
	PromptModeFile      = "file"
	PromptModeGenerated = "generated"
	PromptModeRaw       = "raw"
)

// Step 步骤枚举
type Step int

const (
	Step1 Step = 0 // Step 1/3 · 基本信息
	Step2 Step = 1 // Step 2/3 · 测试参数
	Step3 Step = 2 // Step 3/3 · 确认保存
)

// State 向导的完整状态。
type State struct {
	Step      Step
	EditingID string // 非空 = 编辑模式

	// Step 1: 基本信息
	Name        string
	Protocol    string // types.Protocol* 常量
	EndpointURL string
	APIKey      string
	Model       string

	// Step 2: 测试参数
	Turbo     bool
	Integrity bool
	Stream    bool

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

	// Integrity 模式参数
	IntegritySuite    string
	IntegrityFailFast bool

	// Prompt 配置
	PromptMode   string
	PromptText   string
	PromptFile   string
	PromptLength int

	// 当前活跃字段索引（Tab 切换）
	FieldIndex int
	ScrollOff  int
	Input      textinput.Model // 当前活跃文本字段的光标与编辑状态
}

// newTextInput 创建向导使用的 textinput，禁用光标闪烁。
func newTextInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.Focus()
	return ti
}

// NewState 创建新任务向导状态（零值 + 合理默认值）。
func NewState() *State {
	wz := &State{
		Step:            Step1,
		Protocol:        types.ProtocolOpenAICompletions,
		Concurrency:     10,
		Count:           100,
		Timeout:         30,
		InitConcurrency: 1,
		MaxConcurrency:  50,
		StepSize:        2,
		LevelRequests:   30,
		MinSuccessRate:  90.0,
		Stream:          true,
		PromptMode:      PromptModeGenerated,
		PromptLength:    4096,
		Input:           newTextInput(),
	}
	wz.LoadCurrentFieldInput()
	return wz
}

// NewStateEdit 创建编辑任务向导状态（预填任务数据，零值字段沿用默认值）。
func NewStateEdit(t *types.TaskDefinition) *State {
	if t == nil {
		return NewState()
	}
	wz := NewState()
	inp := t.Input
	tc := inp.TurboConfig

	wz.EditingID = t.ID
	wz.Name = t.Name
	wz.Protocol = types.NormalizeProtocol(inp.Protocol)
	wz.EndpointURL = inp.EndpointURL
	wz.APIKey = inp.ApiKey
	wz.Model = inp.Model
	wz.Turbo = inp.Turbo
	wz.Integrity = inp.Integrity.Enabled || inp.Integrity.Suite != ""
	wz.Stream = inp.Stream
	wz.PromptText = inp.PromptText
	wz.PromptFile = inp.PromptFile
	if inp.PromptLength > 0 {
		wz.PromptLength = inp.PromptLength
	}
	if inp.PromptMode != "" {
		wz.PromptMode = inp.PromptMode
	} else if inp.PromptFile != "" {
		wz.PromptMode = PromptModeFile
	} else if inp.PromptLength > 0 {
		wz.PromptMode = PromptModeGenerated
	} else {
		wz.PromptMode = PromptModeText
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
	ic := inp.Integrity
	if ic.Suite != "" {
		wz.IntegritySuite = ic.Suite
	}
	wz.IntegrityFailFast = ic.FailFast
	// 数据字段全部填充完毕后，重新加载当前字段（Name）到 Input
	wz.LoadCurrentFieldInput()
	return wz
}

// BuildTaskConfig 将向导状态转换为 server.TaskConfig。
func (wz *State) BuildTaskConfig() server.TaskConfig {
	turboRate := wz.MinSuccessRate / 100 // 转回小数
	if turboRate <= 0 {
		turboRate = 0.9
	}
	timeout := time.Duration(wz.Timeout) * time.Second
	if wz.Timeout <= 0 {
		timeout = 60 * time.Second
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
			Timeout:     timeout,
			Stream:      wz.Stream,
			Turbo:       wz.Turbo,
			TurboConfig: types.TurboConfig{
				InitConcurrency: wz.InitConcurrency,
				MaxConcurrency:  wz.MaxConcurrency,
				StepSize:        wz.StepSize,
				LevelRequests:   wz.LevelRequests,
				MinSuccessRate:  turboRate,
			},
			Integrity: types.IntegrityConfig{
				Enabled:   wz.Integrity,
				Suite:     wz.IntegritySuite,
				FailFast:  wz.IntegrityFailFast,
			},
			PromptMode:   wz.PromptMode,
			PromptText:   wz.PromptText,
			PromptFile:   wz.PromptFile,
			PromptLength: wz.PromptLength,
		},
	}
}

// ModeState 接口实现 - Standard 模式访问器
func (wz *State) GetConcurrency() int       { return wz.Concurrency }
func (wz *State) SetConcurrency(n int)      { wz.Concurrency = n }
func (wz *State) GetCount() int             { return wz.Count }
func (wz *State) SetCount(n int)            { wz.Count = n }
func (wz *State) GetTimeout() int           { return wz.Timeout }
func (wz *State) SetTimeout(n int)          { wz.Timeout = n }

// ModeState 接口实现 - Turbo 模式访问器
func (wz *State) GetInitConcurrency() int   { return wz.InitConcurrency }
func (wz *State) SetInitConcurrency(n int)  { wz.InitConcurrency = n }
func (wz *State) GetMaxConcurrency() int    { return wz.MaxConcurrency }
func (wz *State) SetMaxConcurrency(n int)   { wz.MaxConcurrency = n }
func (wz *State) GetStepSize() int          { return wz.StepSize }
func (wz *State) SetStepSize(n int)         { wz.StepSize = n }
func (wz *State) GetLevelRequests() int     { return wz.LevelRequests }
func (wz *State) SetLevelRequests(n int)    { wz.LevelRequests = n }
func (wz *State) GetMinSuccessRate() float64 { return wz.MinSuccessRate }
func (wz *State) SetMinSuccessRate(f float64) { wz.MinSuccessRate = f }

// ModeState 接口实现 - Integrity 模式访问器
func (wz *State) GetIntegritySuite() string      { return wz.IntegritySuite }
func (wz *State) SetIntegritySuite(s string)     { wz.IntegritySuite = s }
func (wz *State) GetIntegrityFailFast() bool     { return wz.IntegrityFailFast }
func (wz *State) SetIntegrityFailFast(b bool)    { wz.IntegrityFailFast = b }
