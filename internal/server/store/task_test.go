package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server/types"
)

func TestTaskStore_CRUD(t *testing.T) {
	store := NewTaskStore(filepath.Join(t.TempDir(), "tasks"))

	created, err := store.Create(types.TaskDefinition{
		Name: "task-a",
		Input: types.Input{
			Protocol:    types.ProtocolOpenAICompletions,
			EndpointURL: "http://localhost:19999",
			Model:       "test-model",
			PromptMode:  "text",
			PromptText:  "hello",
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created task to have ID")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be populated")
	}

	loaded, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.Name != created.Name {
		t.Fatalf("Name: got %q, want %q", loaded.Name, created.Name)
	}

	tasks, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	raw, err := os.ReadFile(store.taskPath(created.ID))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(raw), "\"id\"") {
		t.Fatalf("expected task file to omit duplicated id field, got %s", raw)
	}

	time.Sleep(time.Millisecond)
	updated, err := store.Update(types.TaskDefinition{
		ID:        created.ID,
		Name:      "task-b",
		Input:     created.Input,
		CreatedAt: created.CreatedAt,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "task-b" {
		t.Fatalf("Name after update: got %q, want %q", updated.Name, "task-b")
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("expected UpdatedAt to advance, created=%v updated=%v", created.UpdatedAt, updated.UpdatedAt)
	}

	if err := store.Delete(created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(created.ID); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound after delete, got %v", err)
	}
	if err := store.Delete(created.ID); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound on second delete, got %v", err)
	}
}

func TestTaskViewStore_ListIncludesLatestRun(t *testing.T) {
	root := t.TempDir()
	tasks := NewTaskStore(filepath.Join(root, "tasks"))
	runs := NewRunStore(filepath.Join(root, "runs"))
	views := NewTaskViewStore(tasks, runs)

	taskWithRun, err := tasks.Create(types.TaskDefinition{
		Name:  "with-run",
		Input: types.Input{Protocol: types.ProtocolOpenAICompletions},
	})
	if err != nil {
		t.Fatalf("Create(with-run): %v", err)
	}
	_, err = tasks.Create(types.TaskDefinition{
		Name:  "no-run",
		Input: types.Input{Protocol: types.ProtocolOpenAICompletions},
	})
	if err != nil {
		t.Fatalf("Create(no-run): %v", err)
	}

	olderFinishedAt := time.Now().Add(-2 * time.Minute)
	newerFinishedAt := time.Now().Add(-time.Minute)
	if err := runs.SaveFinalRun(RunMetadata{
		RunID:      "run-older",
		TaskID:     taskWithRun.ID,
		Mode:       "standard",
		Status:     "completed",
		StartedAt:  olderFinishedAt.Add(-time.Second),
		FinishedAt: &olderFinishedAt,
	}, RunResult{}); err != nil {
		t.Fatalf("SaveFinalRun(older): %v", err)
	}
	if err := runs.SaveFinalRun(RunMetadata{
		RunID:      "run-newer",
		TaskID:     taskWithRun.ID,
		Mode:       "standard",
		Status:     "completed",
		StartedAt:  newerFinishedAt.Add(-time.Second),
		FinishedAt: &newerFinishedAt,
	}, RunResult{}); err != nil {
		t.Fatalf("SaveFinalRun(newer): %v", err)
	}

	overviews, err := views.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(overviews) != 2 {
		t.Fatalf("expected 2 task overviews, got %d", len(overviews))
	}

	seenRun := false
	seenNoRun := false
	for _, overview := range overviews {
		switch overview.Name {
		case "with-run":
			seenRun = true
			if overview.LatestRun == nil {
				t.Fatal("expected LatestRun for task with persisted runs")
			}
			if overview.LatestRun.RunID != "run-newer" {
				t.Fatalf("LatestRun.RunID: got %q, want %q", overview.LatestRun.RunID, "run-newer")
			}
		case "no-run":
			seenNoRun = true
			if overview.LatestRun != nil {
				t.Fatal("expected LatestRun to be nil when no persisted run exists")
			}
		}
	}
	if !seenRun || !seenNoRun {
		t.Fatalf("expected to see both task overviews, got %+v", overviews)
	}
}
