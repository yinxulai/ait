package config

import (
	"path/filepath"
	"testing"
)

func TestLoadReturnsDefaultWhenFileMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
	if cfg.SaveAPIKey {
		t.Fatal("expected SaveAPIKey to default to false")
	}
}

func TestConfigSaveAndLoadRoundTrip(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfg := &Config{
		SaveAPIKey:         true,
		LastSelectedTaskID: "task-1",
		DefaultProtocol:    "openai-responses",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() returned unexpected error: %v", err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "config.json"); path != want {
		t.Fatalf("expected config path %s, got %s", want, path)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if !loaded.SaveAPIKey || loaded.LastSelectedTaskID != "task-1" || loaded.DefaultProtocol != "openai-responses" {
		t.Fatalf("unexpected loaded config: %+v", loaded)
	}
}

func TestStoragePaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	tasksDir, err := TasksDir()
	if err != nil {
		t.Fatalf("TasksDir() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "tasks"); tasksDir != want {
		t.Fatalf("expected tasks dir %s, got %s", want, tasksDir)
	}

	runsDir, err := RunsDir()
	if err != nil {
		t.Fatalf("RunsDir() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "runs"); runsDir != want {
		t.Fatalf("expected runs dir %s, got %s", want, runsDir)
	}

	taskPath, err := TaskPath("task-1")
	if err != nil {
		t.Fatalf("TaskPath() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "tasks", "task-1.json"); taskPath != want {
		t.Fatalf("expected task path %s, got %s", want, taskPath)
	}

	runDir, err := RunDir("task-1", "run-1")
	if err != nil {
		t.Fatalf("RunDir() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "runs", "task-1", "run-1"); runDir != want {
		t.Fatalf("expected run dir %s, got %s", want, runDir)
	}

	runMetaPath, err := RunMetadataPath("task-1", "run-1")
	if err != nil {
		t.Fatalf("RunMetadataPath() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "runs", "task-1", "run-1", "run.json"); runMetaPath != want {
		t.Fatalf("expected run metadata path %s, got %s", want, runMetaPath)
	}

	runResultPath, err := RunResultPath("task-1", "run-1")
	if err != nil {
		t.Fatalf("RunResultPath() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "runs", "task-1", "run-1", "result.json"); runResultPath != want {
		t.Fatalf("expected run result path %s, got %s", want, runResultPath)
	}

	runRequestsPath, err := RunRequestsPath("task-1", "run-1")
	if err != nil {
		t.Fatalf("RunRequestsPath() returned unexpected error: %v", err)
	}
	if want := filepath.Join(homeDir, ".ait", "runs", "task-1", "run-1", "requests.jsonl"); runRequestsPath != want {
		t.Fatalf("expected run requests path %s, got %s", want, runRequestsPath)
	}
}
