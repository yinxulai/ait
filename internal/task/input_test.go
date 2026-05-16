package task

import (
	"testing"

	"github.com/yinxulai/ait/internal/types"
)

func TestHydrateInputTextMode(t *testing.T) {
	input, err := HydrateInput(types.Input{PromptMode: "text", PromptText: "hello"})
	if err != nil {
		t.Fatalf("HydrateInput(text) returned unexpected error: %v", err)
	}
	if input.PromptSource == nil || input.PromptSource.GetRandomContent() != "hello" {
		t.Fatal("expected PromptSource to be hydrated from PromptText")
	}
}

func TestHydrateInputGeneratedMode(t *testing.T) {
	input, err := HydrateInput(types.Input{PromptMode: "generated", PromptLength: 32})
	if err != nil {
		t.Fatalf("HydrateInput(generated) returned unexpected error: %v", err)
	}
	if input.PromptSource == nil || input.PromptSource.Count() != 1 {
		t.Fatal("expected generated PromptSource to be created")
	}
}

func TestHydrateInputRejectsInvalidMode(t *testing.T) {
	if _, err := HydrateInput(types.Input{PromptMode: "unknown"}); err == nil {
		t.Fatal("expected HydrateInput to reject unsupported prompt_mode")
	}
}
