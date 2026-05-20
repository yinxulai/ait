package pages

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

func TestTaskDetailHistoryEntries_SkipsActiveRunDuplicate(t *testing.T) {
	state := &TaskDetailState{
		ActiveRun: &server.RunState{RunID: "run-2"},
		History: []types.TaskRunSummary{
			{RunID: "run-2", Status: string(server.RunStatusRunning)},
			{RunID: "run-1", Status: string(server.RunStatusCompleted)},
		},
	}

	entries := taskDetailHistoryEntries(state)
	if len(entries) != 1 {
		t.Fatalf("expected 1 visible history entry, got %d", len(entries))
	}
	if entries[0].RunID != "run-1" {
		t.Fatalf("RunID: got %q, want %q", entries[0].RunID, "run-1")
	}
}

func TestHandleTaskDetailKey_UsesBackNav(t *testing.T) {
	state := &TaskDetailState{
		Task:    types.TaskDefinition{ID: "task-1", Name: "task"},
		BackNav: NavAction{To: NavDashboard},
	}

	_, _, nav := HandleTaskDetailKey(state, tea.KeyMsg{Type: tea.KeyEsc}, nil)
	if nav.To != NavDashboard {
		t.Fatalf("nav.To = %v, want %v", nav.To, NavDashboard)
	}
}
