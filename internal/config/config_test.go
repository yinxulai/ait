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
