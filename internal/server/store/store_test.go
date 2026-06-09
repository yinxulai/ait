package store

import (
	"path/filepath"
	"testing"
	"time"
)

type testPayload struct {
	Value string `json:"value"`
}

func TestDebouncedJSONStore_LoadFlushesPendingWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	debounced := NewDebouncedJSONStore[testPayload](path, 200*time.Millisecond)

	if err := debounced.Save(testPayload{Value: "pending"}); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	plain := NewJSONStore[testPayload](path)
	loaded, err := plain.Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if loaded.Value != "pending" {
		t.Fatalf("Value: got %q, want %q", loaded.Value, "pending")
	}
	if err := debounced.Flush(); err != nil {
		t.Fatalf("Flush() returned unexpected error: %v", err)
	}
}

func TestDebouncedJSONStore_CoalescesWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store := NewDebouncedJSONStore[testPayload](path, 40*time.Millisecond)

	if err := store.Save(testPayload{Value: "first"}); err != nil {
		t.Fatalf("Save(first) returned unexpected error: %v", err)
	}
	if err := store.Save(testPayload{Value: "second"}); err != nil {
		t.Fatalf("Save(second) returned unexpected error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	loaded, err := NewJSONStore[testPayload](path).Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if loaded.Value != "second" {
		t.Fatalf("Value: got %q, want %q", loaded.Value, "second")
	}
}

func TestJSONStore_ImmediateSaveOverridesPendingDebouncedWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	debounced := NewDebouncedJSONStore[testPayload](path, 60*time.Millisecond)
	plain := NewJSONStore[testPayload](path)

	if err := debounced.Save(testPayload{Value: "stale"}); err != nil {
		t.Fatalf("debounced Save() returned unexpected error: %v", err)
	}
	if err := plain.Save(testPayload{Value: "final"}); err != nil {
		t.Fatalf("plain Save() returned unexpected error: %v", err)
	}

	time.Sleep(120 * time.Millisecond)

	loaded, err := plain.Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if loaded.Value != "final" {
		t.Fatalf("Value: got %q, want %q", loaded.Value, "final")
	}
}
