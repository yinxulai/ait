package tui

import (
	"testing"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/tui/pages"
	"github.com/yinxulai/ait/internal/types"
)

// stubServer 是 server.Server 的测试桩，所有方法都返回零值。
type stubServer struct{}

func (s *stubServer) ListTasks() []server.TaskOverview                                     { return nil }
func (s *stubServer) GetTask(id string) (types.TaskDefinition, bool)                       { return types.TaskDefinition{}, false }
func (s *stubServer) CreateTask(cfg server.TaskConfig) (types.TaskDefinition, error)       { return types.TaskDefinition{}, nil }
func (s *stubServer) UpdateTask(id string, cfg server.TaskConfig) (types.TaskDefinition, error) {
	return types.TaskDefinition{}, nil
}
func (s *stubServer) DeleteTask(id string) error                                            { return nil }
func (s *stubServer) CopyTask(id string) (types.TaskDefinition, error)                    { return types.TaskDefinition{}, nil }
func (s *stubServer) StartRun(taskID string) (server.RunID, error)                        { return "", nil }
func (s *stubServer) StopRun(runID server.RunID) error                                     { return nil }
func (s *stubServer) GetRunState(runID server.RunID) (*server.RunState, bool)              { return nil, false }
func (s *stubServer) Subscribe(runID server.RunID) (<-chan server.Event, server.CancelFunc) {
	ch := make(chan server.Event)
	close(ch)
	return ch, func() {}
}
func (s *stubServer) GetHistory(taskID string, limit int) ([]types.TaskRunSummary, error) { return nil, nil }
func (s *stubServer) GenerateReport(runID server.RunID, fmt server.ReportFormat) (string, error) {
	return "", nil
}

// ─── NewModel ─────────────────────────────────────────────────────────────────

func TestNewModel_InitialState(t *testing.T) {
	m := NewModel(&stubServer{})
	if m == nil {
		t.Fatal("NewModel returned nil")
	}
	if m.view != viewTaskList {
		t.Errorf("initial view = %q, want %q", m.view, viewTaskList)
	}
}

// ─── Wizard: NewWizardState + BuildTaskConfig ──────────────────────────────────

func TestOpenWizard_NewTask_Defaults(t *testing.T) {
	m := NewModel(&stubServer{})
	m.wizard = pages.NewWizardState()
	if m.wizard == nil {
		t.Fatal("wizard should not be nil after NewWizardState")
	}
	if m.wizard.EditingID != "" {
		t.Errorf("new task wizard should have empty EditingID, got %q", m.wizard.EditingID)
	}
	if m.wizard.Concurrency <= 0 {
		t.Errorf("default concurrency should be positive, got %d", m.wizard.Concurrency)
	}
	if m.wizard.PromptMode != pages.PromptModeText {
		t.Errorf("default PromptMode = %q, want %q", m.wizard.PromptMode, pages.PromptModeText)
	}
}

func TestOpenWizard_EditTask_Populate(t *testing.T) {
	m := NewModel(&stubServer{})
	task := types.TaskDefinition{
		ID:   "task-123",
		Name: "my-task",
		Input: types.Input{
			Model:       "gpt-4",
			Protocol:    types.ProtocolOpenAICompletions,
			ApiKey:      "sk-test",
			Concurrency: 5,
			Count:       50,
			PromptMode:  pages.PromptModeText,
			PromptText:  "hello",
		},
	}
	m.wizard = pages.NewWizardStateEdit(&task)
	if m.wizard == nil {
		t.Fatal("wizard should not be nil")
	}
	if m.wizard.EditingID != "task-123" {
		t.Errorf("EditingID = %q, want %q", m.wizard.EditingID, "task-123")
	}
	if m.wizard.Model != "gpt-4" {
		t.Errorf("Model = %q, want %q", m.wizard.Model, "gpt-4")
	}
	if m.wizard.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", m.wizard.Concurrency)
	}
}

func TestBuildTaskInput_Standard(t *testing.T) {
	m := NewModel(&stubServer{})
	m.wizard = pages.NewWizardState()
	wz := m.wizard
	wz.Model = "gpt-4.1"
	wz.APIKey = "sk-test"
	wz.Concurrency = 8
	wz.Count = 120
	wz.PromptMode = pages.PromptModeText
	wz.PromptText = "hello"

	cfg := wz.BuildTaskConfig()
	inp := cfg.Input
	if inp.Model != "gpt-4.1" {
		t.Errorf("model = %q, want gpt-4.1", inp.Model)
	}
	if inp.Concurrency != 8 {
		t.Errorf("concurrency = %d, want 8", inp.Concurrency)
	}
	if inp.Count != 120 {
		t.Errorf("count = %d, want 120", inp.Count)
	}
	if inp.PromptMode != pages.PromptModeText || inp.PromptText != "hello" {
		t.Errorf("unexpected prompt config: mode=%q text=%q", inp.PromptMode, inp.PromptText)
	}
	if inp.Turbo {
		t.Error("turbo should be false in standard mode")
	}
}

func TestBuildTaskInput_Turbo(t *testing.T) {
	m := NewModel(&stubServer{})
	m.wizard = pages.NewWizardState()
	wz := m.wizard
	wz.Model = "claude-3-7-sonnet"
	wz.APIKey = "sk-ant"
	wz.Protocol = types.ProtocolAnthropicMessages
	wz.Turbo = true
	wz.InitConcurrency = 1
	wz.MaxConcurrency = 12
	wz.StepSize = 2
	wz.LevelRequests = 20
	wz.PromptMode = pages.PromptModeGenerated
	wz.PromptLength = 256

	cfg := wz.BuildTaskConfig()
	inp := cfg.Input
	if !inp.Turbo {
		t.Error("expected Turbo=true")
	}
	if inp.TurboConfig.MaxConcurrency != 12 {
		t.Errorf("MaxConcurrency = %d, want 12", inp.TurboConfig.MaxConcurrency)
	}
	if inp.PromptMode != pages.PromptModeGenerated || inp.PromptLength != 256 {
		t.Errorf("unexpected prompt config: mode=%q len=%d", inp.PromptMode, inp.PromptLength)
	}
	if inp.Protocol != types.ProtocolAnthropicMessages {
		t.Errorf("protocol = %q, want anthropic-messages", inp.Protocol)
	}
}
