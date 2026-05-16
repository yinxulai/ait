package tui

import (
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/config"
	"github.com/yinxulai/ait/internal/types"
)

func TestBuildTaskDefinitionStandardMode(t *testing.T) {
	state := newWizardState(nil, viewTaskList, &config.Config{DefaultProtocol: types.ProtocolOpenAIResponses})
	state.values["name"] = "nightly-openai"
	state.values["endpoint"] = "https://api.openai.com/v1/responses"
	state.values["apiKey"] = "sk-test"
	state.values["model"] = "gpt-4.1"
	state.values["concurrency"] = "8"
	state.values["count"] = "120"
	state.values["timeout"] = "45s"
	state.values["prompt_value"] = "hello"

	taskDef, err := buildTaskDefinition(state)
	if err != nil {
		t.Fatalf("buildTaskDefinition() returned unexpected error: %v", err)
	}
	if taskDef.Input.Protocol != types.ProtocolOpenAIResponses {
		t.Fatalf("expected protocol %s, got %s", types.ProtocolOpenAIResponses, taskDef.Input.Protocol)
	}
	if taskDef.Input.EndpointURL != "https://api.openai.com/v1/responses" {
		t.Fatalf("unexpected endpoint: %s", taskDef.Input.EndpointURL)
	}
	if taskDef.Input.Concurrency != 8 || taskDef.Input.Count != 120 || taskDef.Input.Timeout != 45*time.Second {
		t.Fatalf("unexpected standard input fields: %+v", taskDef.Input)
	}
	if taskDef.Input.PromptMode != promptModeText || taskDef.Input.PromptText != "hello" {
		t.Fatalf("unexpected prompt fields: %+v", taskDef.Input)
	}
}

func TestBuildTaskDefinitionTurboMode(t *testing.T) {
	state := newWizardState(nil, viewTaskList, &config.Config{})
	state.mode = modeTurbo
	state.protocolIndex = 2
	state.promptModeIndex = 2
	state.values["name"] = "turbo-anthropic"
	state.values["endpoint"] = "https://api.anthropic.com/v1/messages"
	state.values["apiKey"] = "sk-ant"
	state.values["model"] = "claude-3-7-sonnet"
	state.values["turbo_init"] = "1"
	state.values["turbo_max"] = "12"
	state.values["turbo_step"] = "2"
	state.values["turbo_level_requests"] = "20"
	state.values["turbo_min_success"] = "0.92"
	state.values["turbo_max_latency"] = "6s"
	state.values["prompt_value"] = "256"

	taskDef, err := buildTaskDefinition(state)
	if err != nil {
		t.Fatalf("buildTaskDefinition() returned unexpected error: %v", err)
	}
	if !taskDef.Input.Turbo {
		t.Fatal("expected Turbo to be enabled")
	}
	if taskDef.Input.TurboConfig.MaxConcurrency != 12 || taskDef.Input.TurboConfig.MaxLatency != 6*time.Second {
		t.Fatalf("unexpected turbo config: %+v", taskDef.Input.TurboConfig)
	}
	if taskDef.Input.PromptMode != promptModeGenerated || taskDef.Input.PromptLength != 256 {
		t.Fatalf("unexpected generated prompt config: %+v", taskDef.Input)
	}
	if taskDef.Input.Protocol != types.ProtocolAnthropicMessages {
		t.Fatalf("expected anthropic protocol, got %s", taskDef.Input.Protocol)
	}
}
