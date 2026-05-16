package task

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

func TestLoadTasksReturnsEmptyStoreWhenMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store, err := LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() returned unexpected error: %v", err)
	}
	if len(store.Tasks) != 0 {
		t.Fatalf("expected no tasks, got %d", len(store.Tasks))
	}
}

func TestTaskStoreUpsertSaveAndReload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	store := &TaskStore{}
	task := types.TaskDefinition{
		ID:   "task-1",
		Name: "nightly-openai",
		Input: types.Input{
			Protocol:    types.ProtocolOpenAIResponses,
			EndpointURL: "https://api.openai.com/v1/responses",
			Model:       "gpt-4.1",
		},
	}
	store.Upsert(task)
	if err := store.Save(); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	loaded, err := LoadTasks()
	if err != nil {
		t.Fatalf("LoadTasks() returned unexpected error: %v", err)
	}
	if len(loaded.Tasks) != 1 || loaded.Tasks[0].ID != "task-1" {
		t.Fatalf("unexpected loaded tasks: %+v", loaded.Tasks)
	}

	firstUpdatedAt := loaded.Tasks[0].UpdatedAt
	time.Sleep(10 * time.Millisecond)
	task.Name = "nightly-openai-updated"
	loaded.Upsert(task)
	if len(loaded.Tasks) != 1 {
		t.Fatalf("expected one task after update, got %d", len(loaded.Tasks))
	}
	if loaded.Tasks[0].Name != "nightly-openai-updated" {
		t.Fatalf("expected updated task name, got %s", loaded.Tasks[0].Name)
	}
	if !loaded.Tasks[0].UpdatedAt.After(firstUpdatedAt) {
		t.Fatalf("expected UpdatedAt to advance after Upsert")
	}
}

func TestTaskStoreDelete(t *testing.T) {
	store := &TaskStore{Tasks: []types.TaskDefinition{{ID: "task-1"}, {ID: "task-2"}}}
	if err := store.Delete("task-1"); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}
	if len(store.Tasks) != 1 || store.Tasks[0].ID != "task-2" {
		t.Fatalf("unexpected tasks after delete: %+v", store.Tasks)
	}
	if err := store.Delete("missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}
