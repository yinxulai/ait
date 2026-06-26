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

func TestEvaluateAll_TemplateValue(t *testing.T) {
	obs := map[string]any{
		"requests": []any{
			map[string]any{
				"metrics": map[string]any{"total_ms": float64(100)},
			},
			map[string]any{
				"metrics": map[string]any{"total_ms": float64(50)},
			},
		},
	}

	// 单模板引用：lt + 模板值
	results, err := EvaluateAll(obs, []types.Assertion{
		{ID: "template-lt", Path: "requests[1].metrics.total_ms", Op: "lt", Value: "{{requests[0].metrics.total_ms}}"},
	})
	if err != nil {
		t.Fatalf("EvaluateAll error: %v", err)
	}
	if !results[0].Passed {
		t.Fatalf("expected template lt to pass: 50 < 100")
	}

	// 模板值引用不存在的路径：值为原始字符串，eq 不匹配
	results, err = EvaluateAll(obs, []types.Assertion{
		{ID: "template-missing", Path: "requests[0].metrics.total_ms", Op: "eq", Value: "{{requests[0].metrics.missing}}"},
	})
	if err != nil {
		t.Fatalf("EvaluateAll error: %v", err)
	}
	if results[0].Passed {
		t.Fatalf("expected missing template to fail eq against unresolved string")
	}
}

func TestEvaluateAll_MultiRequestObservation(t *testing.T) {
	obs := map[string]any{
		"task": map[string]any{"protocol": "openai-completions"},
		"case": map[string]any{"id": "cache-test"},
		"requests": []any{
			map[string]any{
				"index":    0,
				"metrics":  map[string]any{"cached_tokens": float64(0), "total_ms": float64(200)},
				"response": map[string]any{"body": map[string]any{"id": "resp-1"}},
			},
			map[string]any{
				"index":    1,
				"metrics":  map[string]any{"cached_tokens": float64(512), "total_ms": float64(80)},
				"response": map[string]any{"body": map[string]any{"id": "resp-2"}},
			},
		},
	}

	// 混合风格：显式 requests[N] 路径 + request_index 路径
	results, err := EvaluateAll(obs, []types.Assertion{
		// 旧风格：显式 requests[0] 路径（仍然支持）
		{ID: "req0.cached_zero", Path: "requests[0].metrics.cached_tokens", Op: "eq", Value: float64(0)},
		// 新风格：使用 request_index
		{ID: "req1.cached_nonzero", Path: "metrics.cached_tokens", RequestIndex: 1, Op: "gt", Value: float64(0)},
		// 跨请求模板比较
		{ID: "req1.faster", Path: "metrics.total_ms", RequestIndex: 1, Op: "lt", Value: "{{requests[0].metrics.total_ms}}"},
		{ID: "resp0.id", Path: "response.body.id", RequestIndex: 0, Op: "eq", Value: "resp-1"},
	})
	if err != nil {
		t.Fatalf("EvaluateAll error: %v", err)
	}
	for _, r := range results {
		if !r.Passed {
			t.Fatalf("assertion %q should pass: %#v", r.AssertionID, r)
		}
	}
}
