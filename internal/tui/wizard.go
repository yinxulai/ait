package tui

// WizardState holds the transient state for the 3-step task creation/edit wizard.
// It is populated by the wizard form and consumed by CreateTask/UpdateTask.
type WizardState struct {
	Step   int    // 0=Mode, 1=Configure, 2=Review
	EditID string // non-empty when editing an existing task

	// Step 1: Mode
	Turbo bool

	// Step 2: Configuration
	Name        string
	Protocol    string
	EndpointURL string
	APIKey      string
	Model       string

	// ── Load parameters ──
	Concurrency int
	Count       int
	TimeoutSec  int
	Stream      bool

	// ── Turbo-specific ──
	InitConc   int
	MaxConc    int
	StepSize   int
	LevelReqs  int
	StopMinSR  float64 // minimum success rate to stop ramping

	// ── Integrity-specific ──
	IntegritySuite string
}

// newWizardState creates a fresh wizard with defaults.
func newWizardState() *WizardState {
	return &WizardState{
		Step:        0,
		Concurrency: 10,
		Count:       100,
		TimeoutSec:  30,
		InitConc:    1,
		MaxConc:     50,
		StepSize:    5,
		LevelReqs:   100,
		StopMinSR:   0.95,
	}
}

// newWizardStateEdit creates a wizard pre-populated from an existing task.
func newWizardStateEdit(task *TaskDefinition) *WizardState {
	ws := newWizardState()
	ws.EditID = task.ID
	ws.Name = task.Name
	ws.Protocol = task.Input.Protocol
	ws.EndpointURL = task.Input.EndpointURL
	ws.APIKey = task.Input.APIKey
	ws.Model = task.Input.Model
	ws.Turbo = task.Input.Turbo
	ws.Concurrency = task.Input.Concurrency
	ws.Count = task.Input.Count
	ws.TimeoutSec = int(task.Input.Timeout)
	ws.Stream = task.Input.Stream
	if task.Input.Turbo {
		ws.InitConc = task.Input.TurboConfig.InitConcurrency
		ws.MaxConc = task.Input.TurboConfig.MaxConcurrency
		ws.StepSize = task.Input.TurboConfig.StepSize
	}
	return ws
}

// TaskDefinition shadows server/types.TaskDefinition for the wizard.
// (The real type is in server/types; this is a local copy to break import cycles.)
type TaskDefinition struct {
	ID    string
	Name  string
	Input TaskInput
}

// TaskInput shadows server/types.Input.
type TaskInput struct {
	Protocol     string
	EndpointURL  string
	APIKey       string
	Model        string
	Concurrency  int
	Count        int
	Timeout      Duration
	Stream       bool
	Turbo        bool
	TurboConfig  TurboConfig
	IntegritySuite string
}

// Duration wraps time.Duration for JSON handling.
type Duration = int64 // seconds

// TurboConfig holds turbo ramp parameters.
type TurboConfig struct {
	InitConcurrency int
	MaxConcurrency  int
	StepSize        int
}
