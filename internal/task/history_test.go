package task

import (
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestAppendRunAndLoadHistoryNewestFirst(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	first := types.TaskRunSummary{RunID: "run-1", StartedAt: time.Unix(100, 0), FinishedAt: time.Unix(110, 0)}
	second := types.TaskRunSummary{RunID: "run-2", StartedAt: time.Unix(200, 0), FinishedAt: time.Unix(210, 0)}

	if err := AppendRun("task-1", first); err != nil {
		t.Fatalf("AppendRun(first) returned unexpected error: %v", err)
	}
	if err := AppendRun("task-1", second); err != nil {
		t.Fatalf("AppendRun(second) returned unexpected error: %v", err)
	}

	history, err := LoadHistory("task-1", 0)
	if err != nil {
		t.Fatalf("LoadHistory() returned unexpected error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(history))
	}
	if history[0].RunID != "run-2" || history[1].RunID != "run-1" {
		t.Fatalf("expected newest-first order, got %+v", history)
	}

	limited, err := LoadHistory("task-1", 1)
	if err != nil {
		t.Fatalf("LoadHistory(limit) returned unexpected error: %v", err)
	}
	if len(limited) != 1 || limited[0].RunID != "run-2" {
		t.Fatalf("unexpected limited history: %+v", limited)
	}
}
