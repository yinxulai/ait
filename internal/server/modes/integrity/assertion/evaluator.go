package assertion

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/yinxulai/ait/internal/server/types"
)

type Evaluation struct {
	Assertion types.Assertion
	Passed    bool
	Actual    any
	Found     bool
	Message   string
}

func EvaluateAll(observation map[string]any, assertions []types.Assertion) ([]types.AssertionResult, error) {
	results := make([]types.AssertionResult, 0, len(assertions))
	for _, a := range assertions {
		eval, err := Evaluate(observation, a)
		if err != nil {
			return nil, err
		}
		results = append(results, types.AssertionResult{
			AssertionID:  eval.Assertion.ID,
			Level:        normalizeLevel(eval.Assertion.Level),
			Passed:       eval.Passed,
			RequestIndex: eval.Assertion.RequestIndex,
			Path:         eval.Assertion.Path,
			Op:           eval.Assertion.Op,
			Expected:     eval.Assertion.Value,
			Actual:       eval.Actual,
			Message:      eval.Message,
		})
	}
	return results, nil
}

func Evaluate(observation map[string]any, a types.Assertion) (Evaluation, error) {
	path := a.Path
	// 如果 observation 是多请求合并结构（包含 requests 数组），
	// 且 path 不以 requests[ 开头，则根据 RequestIndex 自动加前缀
	if !strings.HasPrefix(path, "requests[") {
		if _, hasRequests := observation["requests"]; hasRequests {
			path = fmt.Sprintf("requests[%d].%s", a.RequestIndex, path)
		}
	}
	actual, found, err := ResolvePath(observation, path)
	if err != nil {
		return Evaluation{}, err
	}

	op := strings.ToLower(strings.TrimSpace(a.Op))
	if op == "" {
		op = "exists"
	}

	// 解析 Value 中的 {{path}} 模板引用
	resolvedValue := resolveTemplateValue(observation, a.Value)

	passed, err := evaluateOp(op, actual, found, resolvedValue)
	if err != nil {
		return Evaluation{}, fmt.Errorf("assertion %q: %w", a.ID, err)
	}

	message := a.Message
	if message == "" && !passed {
		message = fmt.Sprintf("assertion %q failed: %s %s", a.ID, a.Path, op)
	}

	return Evaluation{Assertion: a, Passed: passed, Actual: actual, Found: found, Message: message}, nil
}

func ResolvePath(root any, path string) (any, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return root, true, nil
	}

	current := root
	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			return nil, false, fmt.Errorf("invalid empty path segment in %q", path)
		}
		name, indexes, err := parseSegment(segment)
		if err != nil {
			return nil, false, err
		}
		if name != "" {
			m, ok := current.(map[string]any)
			if !ok {
				return nil, false, nil
			}
			var exists bool
			current, exists = m[name]
			if !exists {
				return nil, false, nil
			}
		}
		for _, idx := range indexes {
			arr, ok := current.([]any)
			if !ok || idx < 0 || idx >= len(arr) {
				return nil, false, nil
			}
			current = arr[idx]
		}
	}
	return current, true, nil
}

func parseSegment(segment string) (string, []int, error) {
	nameEnd := strings.Index(segment, "[")
	if nameEnd == -1 {
		return segment, nil, nil
	}

	name := segment[:nameEnd]
	rest := segment[nameEnd:]
	indexes := []int{}
	for rest != "" {
		if !strings.HasPrefix(rest, "[") {
			return "", nil, fmt.Errorf("invalid path segment %q", segment)
		}
		end := strings.Index(rest, "]")
		if end < 0 {
			return "", nil, fmt.Errorf("invalid path segment %q", segment)
		}
		idx, err := strconv.Atoi(rest[1:end])
		if err != nil {
			return "", nil, fmt.Errorf("invalid array index in %q", segment)
		}
		indexes = append(indexes, idx)
		rest = rest[end+1:]
	}
	return name, indexes, nil
}

func evaluateOp(op string, actual any, found bool, expected any) (bool, error) {
	switch op {
	case "exists":
		return found, nil
	case "not_exists":
		return !found, nil
	case "eq":
		return found && valuesEqual(actual, expected), nil
	case "neq":
		return !found || !valuesEqual(actual, expected), nil
	case "in":
		if !found {
			return false, nil
		}
		values, ok := expected.([]any)
		if !ok {
			return false, fmt.Errorf("in expects array value")
		}
		for _, value := range values {
			if valuesEqual(actual, value) {
				return true, nil
			}
		}
		return false, nil
	case "contains":
		if !found {
			return false, nil
		}
		return contains(actual, expected)
	case "matches":
		if !found {
			return false, nil
		}
		pattern, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("matches expects string value")
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, err
		}
		return re.MatchString(fmt.Sprint(actual)), nil
	case "gt", "gte", "lt", "lte":
		if !found {
			return false, nil
		}
		actualNum, ok := asFloat(actual)
		if !ok {
			return false, nil
		}
		expectedNum, ok := asFloat(expected)
		if !ok {
			return false, nil
		}
		switch op {
		case "gt":
			return actualNum > expectedNum, nil
		case "gte":
			return actualNum >= expectedNum, nil
		case "lt":
			return actualNum < expectedNum, nil
		default:
			return actualNum <= expectedNum, nil
		}
	case "between":
		if !found {
			return false, nil
		}
		actualNum, ok := asFloat(actual)
		if !ok {
			return false, nil
		}
		bounds, ok := expected.([]any)
		if !ok || len(bounds) != 2 {
			return false, fmt.Errorf("between expects [min, max] value")
		}
		min, ok1 := asFloat(bounds[0])
		max, ok2 := asFloat(bounds[1])
		if !ok1 || !ok2 {
			return false, nil
		}
		return actualNum >= min && actualNum <= max, nil
	default:
		return false, fmt.Errorf("unsupported op %q", op)
	}
}

func normalizeLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return "error"
	}
	return level
}

func valuesEqual(a, b any) bool {
	if af, ok := asFloat(a); ok {
		if bf, ok := asFloat(b); ok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

func contains(actual, expected any) (bool, error) {
	switch value := actual.(type) {
	case string:
		needle, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("contains expects string value for string actual")
		}
		return strings.Contains(value, needle), nil
	case []any:
		for _, item := range value {
			if valuesEqual(item, expected) {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, nil
	}
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

var templateValueRe = regexp.MustCompile(`\{\{(.+?)\}\}`)

// resolveTemplateValue 解析 Value 中的 {{path}} 模板引用，用 observation 中对应路径的值替换。
// 如果 Value 本身就是一个模板字符串（如 "{{requests[0].metrics.total_ms}}"），则直接返回解析后的值（保留其类型）。
// 如果 Value 是包含模板的复合字符串，则进行字符串替换。
// 如果 Value 不是字符串或不含模板，则原样返回。
func resolveTemplateValue(observation map[string]any, value any) any {
	s, ok := value.(string)
	if !ok {
		return value
	}
	matches := templateValueRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return value
	}
	// 如果整个 Value 就是单个 {{path}}，直接返回值本身（保留类型）
	match := templateValueRe.FindStringSubmatch(s)
	if match != nil && match[0] == s {
		resolved, found, _ := ResolvePath(observation, match[1])
		if found {
			return resolved
		}
		return value
	}
	// 复合模板字符串，逐个替换
	result := templateValueRe.ReplaceAllStringFunc(s, func(match string) string {
		key := match[2 : len(match)-2]
		resolved, found, _ := ResolvePath(observation, key)
		if found {
			return fmt.Sprintf("%v", resolved)
		}
		return match
	})
	return result
}
