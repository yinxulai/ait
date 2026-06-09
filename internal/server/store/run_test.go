package store

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server/types"
)

func TestRunStore_MetadataOmitsPathIdentifiers(t *testing.T) {
	store := NewRunStore(t.TempDir())
	finishedAt := time.Now().UTC().Truncate(time.Second)

	meta := RunMetadata{
		RunID:      "run-1",
		TaskID:     "task-1",
		Mode:       "standard",
		Protocol:   "openai-completions",
		Model:      "test-model",
		Status:     "completed",
		StartedAt:  finishedAt.Add(-time.Second),
		FinishedAt: &finishedAt,
	}
	if err := store.SaveFinal(meta, RunResult{}); err != nil {
		t.Fatalf("SaveFinal: %v", err)
	}

	raw, err := os.ReadFile(store.MetadataPath(meta.TaskID, meta.RunID))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "run_id") || strings.Contains(text, "task_id") {
		t.Fatalf("expected run metadata to omit path identifiers, got %s", raw)
	}

	loaded, err := store.Load(meta.TaskID, meta.RunID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected stored run to load")
	}
	if loaded.Metadata.TaskID != meta.TaskID || loaded.Metadata.RunID != meta.RunID {
		t.Fatalf("expected identifiers to be reconstructed from path, got task=%q run=%q", loaded.Metadata.TaskID, loaded.Metadata.RunID)
	}
}

func TestRunStore_ResultOmitsDerivedSummaryFields(t *testing.T) {
	store := NewRunStore(t.TempDir())
	finishedAt := time.Now().UTC().Truncate(time.Second)

	if err := store.SaveFinal(RunMetadata{
		RunID:      "run-2",
		TaskID:     "task-2",
		Mode:       "standard",
		Status:     "completed",
		StartedAt:  finishedAt.Add(-time.Second),
		FinishedAt: &finishedAt,
	}, RunResult{
		ErrorSummary: "boom",
		StandardResult: &types.ReportData{
			TotalRequests: 4,
			SuccessRate:   75,
			AvgTPS:        12.5,
			AvgTTFT:       100 * time.Millisecond,
		},
	}); err != nil {
		t.Fatalf("SaveFinal: %v", err)
	}

	raw, err := os.ReadFile(store.ResultPath("task-2", "run-2"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for _, key := range []string{"done_reqs", "success_reqs", "failed_reqs", "success_rate", "avg_ttft", "avg_tps", "cache_hit_rate"} {
		if _, ok := payload[key]; ok {
			t.Fatalf("expected derived summary field %q to be omitted from result.json, got %s", key, raw)
		}
	}
	if _, ok := payload["standard_result"]; !ok {
		t.Fatalf("expected final report payload to remain in result.json, got %s", raw)
	}
	if _, ok := payload["error_summary"]; !ok {
		t.Fatalf("expected error_summary to remain in result.json, got %s", raw)
	}
}
