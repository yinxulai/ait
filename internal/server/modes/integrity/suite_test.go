package integrity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yinxulai/ait/internal/server/types"
)

func TestLoadSuite_MergesRuleFiles(t *testing.T) {
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.json")
	ruleJSON := `{
		"version":"ait.integrity.rules/v1",
		"suite":"openai-responses-smoke",
		"assertions":[
			{"id":"responses.id.exists","case_id":"basic-response-shape","level":"warn","path":"response.body.custom_id","op":"exists"},
			{"id":"custom.assertion","case_id":"basic-response-shape","level":"error","path":"metrics.total_ms","op":"gte","value":0}
		]
	}`
	if err := os.WriteFile(rulePath, []byte(ruleJSON), 0o600); err != nil {
		t.Fatalf("write rule file: %v", err)
	}

	suite, err := LoadSuite(types.Input{
		Protocol: types.ProtocolOpenAIResponses,
		Integrity: types.IntegrityConfig{
			RuleFiles: []string{rulePath},
		},
	})
	if err != nil {
		t.Fatalf("LoadSuite returned error: %v", err)
	}
	if suite.ID != "openai-responses-smoke" {
		t.Fatalf("unexpected suite id: %s", suite.ID)
	}
	if len(suite.Cases) != 1 {
		t.Fatalf("expected one case, got %d", len(suite.Cases))
	}

	var replaced, custom bool
	for _, a := range suite.Cases[0].Assertions {
		if a.ID == "responses.id.exists" {
			replaced = a.Path == "response.body.custom_id" && a.Level == "warn"
		}
		if a.ID == "custom.assertion" {
			custom = a.Source == rulePath
		}
	}
	if !replaced {
		t.Fatal("expected rule assertion to replace builtin assertion with same id")
	}
	if !custom {
		t.Fatal("expected custom assertion to be appended with source")
	}
}

func TestMergeAssertions_AddsCustomCase(t *testing.T) {
	suite := BuiltinSuite(types.ProtocolOpenAICompletions, "")
	suite = MergeAssertions(suite, []types.Assertion{
		{ID: "custom.case.assertion", CaseID: "custom-case", Path: "response.body.id", Op: "exists"},
	}, "custom.json")

	for _, c := range suite.Cases {
		if c.ID == "custom-case" {
			if len(c.Assertions) != 1 {
				t.Fatalf("expected one assertion in custom case, got %d", len(c.Assertions))
			}
			return
		}
	}
	t.Fatal("expected custom case to be added")
}

func TestLoadRuleFile_RejectsUnsupportedVersion(t *testing.T) {
	dir := t.TempDir()
	rulePath := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(rulePath, []byte(`{"version":"bad/v1"}`), 0o600); err != nil {
		t.Fatalf("write rule file: %v", err)
	}
	if _, err := LoadRuleFile(rulePath); err == nil {
		t.Fatal("expected unsupported version error")
	}
}
