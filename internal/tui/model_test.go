package tui

import (
	"testing"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// stubServer 是 server.Server 的测试桩，所有方法都返回零值。
type stubServer struct{}

func (s *stubServer) ListTasks() []types.TaskDefinition                                    { return nil }
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

// ─── Wizard: openWizard + buildTaskInput ──────────────────────────────────────

func TestOpenWizard_NewTask_Defaults(t *testing.T) {
	m := NewModel(&stubServer{})
	m.openWizard(nil)
	if m.wizard == nil {
		t.Fatal("wizard should not be nil after openWizard")
	}
	if m.wizard.editingID != "" {
		t.Errorf("new task wizard should have empty editingID, got %q", m.wizard.editingID)
	}
	if m.wizard.concurrency <= 0 {
		t.Errorf("default concurrency should be positive, got %d", m.wizard.concurrency)
	}
	if m.wizard.promptMode != promptModeText {
		t.Errorf("default promptMode = %q, want %q", m.wizard.promptMode, promptModeText)
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
			PromptMode:  promptModeText,
			PromptText:  "hello",
		},
	}
	m.openWizard(&task)
	if m.wizard == nil {
		t.Fatal("wizard should not be nil")
	}
	if m.wizard.editingID != "task-123" {
		t.Errorf("editingID = %q, want %q", m.wizard.editingID, "task-123")
	}
	if m.wizard.model != "gpt-4" {
		t.Errorf("model = %q, want %q", m.wizard.model, "gpt-4")
	}
	if m.wizard.concurrency != 5 {
		t.Errorf("concurrency = %d, want 5", m.wizard.concurrency)
	}
}

func TestBuildTaskInput_Standard(t *testing.T) {
	m := NewModel(&stubServer{})
	m.openWizard(nil)
	wz := m.wizard
	wz.model = "gpt-4.1"
	wz.apiKey = "sk-test"
	wz.concurrency = 8
	wz.count = 120
	wz.promptMode = promptModeText
	wz.promptText = "hello"

	inp := m.buildTaskInput()
	if inp.Model != "gpt-4.1" {
		t.Errorf("model = %q, want gpt-4.1", inp.Model)
	}
	if inp.Concurrency != 8 {
		t.Errorf("concurrency = %d, want 8", inp.Concurrency)
	}
	if inp.Count != 120 {
		t.Errorf("count = %d, want 120", inp.Count)
	}
	if inp.PromptMode != promptModeText || inp.PromptText != "hello" {
		t.Errorf("unexpected prompt config: mode=%q text=%q", inp.PromptMode, inp.PromptText)
	}
	if inp.Turbo {
		t.Error("turbo should be false in standard mode")
	}
}

func TestBuildTaskInput_Turbo(t *testing.T) {
	m := NewModel(&stubServer{})
	m.openWizard(nil)
	wz := m.wizard
	wz.model = "claude-3-7-sonnet"
	wz.apiKey = "sk-ant"
	wz.protocol = types.ProtocolAnthropicMessages
	wz.turbo = true
	wz.initConcurrency = 1
	wz.maxConcurrency = 12
	wz.stepSize = 2
	wz.levelRequests = 20
	wz.promptMode = promptModeGenerated
	wz.promptLength = 256

	inp := m.buildTaskInput()
	if !inp.Turbo {
		t.Error("expected Turbo=true")
	}
	if inp.TurboConfig.MaxConcurrency != 12 {
		t.Errorf("MaxConcurrency = %d, want 12", inp.TurboConfig.MaxConcurrency)
	}
	if inp.PromptMode != promptModeGenerated || inp.PromptLength != 256 {
		t.Errorf("unexpected prompt config: mode=%q len=%d", inp.PromptMode, inp.PromptLength)
	}
	if inp.Protocol != types.ProtocolAnthropicMessages {
		t.Errorf("protocol = %q, want anthropic-messages", inp.Protocol)
	}
}
