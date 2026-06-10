package assertion

import (
	"testing"

	"github.com/yinxulai/ait/internal/server/types"
)

func TestEvaluateAll_Operators(t *testing.T) {
	observation := map[string]any{
		"response": map[string]any{
			"body": map[string]any{
				"id":      "resp_123",
				"text":    "hello world",
				"status":  "completed",
				"scores":  []any{"basic", "usage"},
				"latency": float64(42),
				"nil":     nil,
			},
		},
		"metrics": map[string]any{
			"total_ms": 120,
		},
	}

	assertions := []types.Assertion{
		{ID: "exists", Path: "response.body.id", Op: "exists"},
		{ID: "nil-exists", Path: "response.body.nil", Op: "exists"},
		{ID: "not-exists", Path: "response.body.missing", Op: "not_exists"},
		{ID: "eq", Path: "response.body.status", Op: "eq", Value: "completed"},
		{ID: "contains-string", Path: "response.body.text", Op: "contains", Value: "world"},
		{ID: "contains-array", Path: "response.body.scores", Op: "contains", Value: "usage"},
		{ID: "matches", Path: "response.body.id", Op: "matches", Value: `^resp_\d+$`},
		{ID: "between", Path: "response.body.latency", Op: "between", Value: []any{float64(40), float64(50)}},
		{ID: "gte", Path: "metrics.total_ms", Op: "gte", Value: 0},
	}

	results, err := EvaluateAll(observation, assertions)
	if err != nil {
		t.Fatalf("EvaluateAll returned error: %v", err)
	}
	if len(results) != len(assertions) {
		t.Fatalf("expected %d results, got %d", len(assertions), len(results))
	}
	for _, result := range results {
		if !result.Passed {
			t.Fatalf("assertion %s should pass: %#v", result.AssertionID, result)
		}
	}
}

func TestEvaluateAll_ArrayPath(t *testing.T) {
	observation := map[string]any{
		"response": map[string]any{
			"body": map[string]any{
				"output": []any{
					map[string]any{"type": "message"},
				},
			},
		},
	}

	results, err := EvaluateAll(observation, []types.Assertion{
		{ID: "array-path", Path: "response.body.output[0].type", Op: "eq", Value: "message"},
	})
	if err != nil {
		t.Fatalf("EvaluateAll returned error: %v", err)
	}
	if !results[0].Passed {
		t.Fatalf("array path assertion should pass: %#v", results[0])
	}
}

func TestEvaluateAll_UnsupportedOp(t *testing.T) {
	_, err := EvaluateAll(map[string]any{}, []types.Assertion{{ID: "bad", Path: "x", Op: "unknown"}})
	if err == nil {
		t.Fatal("expected unsupported op error")
	}
}
