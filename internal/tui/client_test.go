package tui

import (
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

func TestSummaryToRunState_RunningSummaryKeepsNilFinishedAt(t *testing.T) {
	state := summaryToRunState(&types.TaskRunSummary{
		RunID:       "run-1",
		TaskID:      "task-1",
		Status:      string(server.RunStatusRunning),
		StartedAt:   time.Unix(100, 0),
		AvgTPS:      12.5,
		SuccessRate: 50,
	})

	if state.Status != server.RunStatusRunning {
		t.Fatalf("Status: got %q, want %q", state.Status, server.RunStatusRunning)
	}
	if state.FinishedAt != nil {
		t.Fatal("expected FinishedAt to stay nil for running summary fallback")
	}
	if state.AvgTPS != 12.5 {
		t.Fatalf("AvgTPS: got %v, want %v", state.AvgTPS, 12.5)
	}
}
