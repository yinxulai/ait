package integrity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yinxulai/ait/internal/server/types"
)

const (
	DefaultRuleVersion = "ait.integrity.rules/v1"
	DefaultSuiteID     = "openai-completions-smoke"
)

type RuleFile struct {
	Version    string            `json:"version"`
	Suite      string            `json:"suite"`
	Assertions []types.Assertion `json:"assertions"`
}

func LoadSuite(input types.Input) (types.IntegritySuite, error) {
	suite := BuiltinSuite(input.NormalizedProtocol(), input.Integrity.Suite)
	for _, file := range input.Integrity.RuleFiles {
		rules, err := LoadRuleFile(file)
		if err != nil {
			return types.IntegritySuite{}, err
		}
		if strings.TrimSpace(rules.Suite) != "" && rules.Suite != suite.ID {
			return types.IntegritySuite{}, fmt.Errorf("rule file %q targets suite %q, current suite is %q", file, rules.Suite, suite.ID)
		}
		suite = MergeAssertions(suite, rules.Assertions, file)
	}
	return suite, nil
}

func LoadRuleFile(path string) (RuleFile, error) {
	path = expandHome(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return RuleFile{}, fmt.Errorf("load integrity rule file %q: %w", path, err)
	}
	var rules RuleFile
	if err := json.Unmarshal(data, &rules); err != nil {
		return RuleFile{}, fmt.Errorf("parse integrity rule file %q: %w", path, err)
	}
	if rules.Version != "" && rules.Version != DefaultRuleVersion {
		return RuleFile{}, fmt.Errorf("unsupported integrity rule version %q in %q", rules.Version, path)
	}
	for i := range rules.Assertions {
		if strings.TrimSpace(rules.Assertions[i].Source) == "" {
			rules.Assertions[i].Source = path
		}
	}
	return rules, nil
}

func MergeAssertions(suite types.IntegritySuite, assertions []types.Assertion, source string) types.IntegritySuite {
	caseIndex := make(map[string]int, len(suite.Cases))
	for i := range suite.Cases {
		caseIndex[suite.Cases[i].ID] = i
	}

	for _, a := range assertions {
		if a.Source == "" {
			a.Source = source
		}
		caseID := a.CaseID
		if caseID == "" && len(suite.Cases) > 0 {
			caseID = suite.Cases[0].ID
		}
		idx, ok := caseIndex[caseID]
		if !ok {
			suite.Cases = append(suite.Cases, types.IntegrityCase{
				ID:         caseID,
				Name:       caseID,
				Category:   "custom",
				Capability: "custom",
				Required:   true,
			})
			idx = len(suite.Cases) - 1
			caseIndex[caseID] = idx
		}
		caseAssertions := suite.Cases[idx].Assertions
		replaced := false
		if a.ID != "" {
			for j := range caseAssertions {
				if caseAssertions[j].ID == a.ID {
					caseAssertions[j] = a
					replaced = true
					break
				}
			}
		}
		if !replaced {
			caseAssertions = append(caseAssertions, a)
		}
		suite.Cases[idx].Assertions = caseAssertions
	}
	return suite
}

func BuiltinSuite(protocol, requested string) types.IntegritySuite {
	protocol = types.NormalizeProtocol(protocol)
	id := strings.TrimSpace(requested)
	if id == "" {
		switch protocol {
		case types.ProtocolOpenAIResponses:
			id = "openai-responses-smoke"
		case types.ProtocolAnthropicMessages:
			id = "anthropic-messages-smoke"
		default:
			id = DefaultSuiteID
		}
	}

	suite := types.IntegritySuite{
		Version:      "ait.integrity/v1",
		ID:           id,
		Name:         id,
		Description:  "Built-in smoke integrity suite",
		Protocols:    []string{protocol},
		Capabilities: []string{"basic_request", "usage"},
	}

	caseDef := types.IntegrityCase{
		ID:         "basic-response-shape",
		Name:       "基础响应结构",
		Category:   "protocol",
		Capability: "basic_request",
		Required:   true,
		Request:    types.IntegrityRequest{Prompt: "Reply with a short greeting.", Stream: false},
		TimeoutMS:  30000,
		Assertions: baseAssertions(protocol),
	}
	suite.Cases = []types.IntegrityCase{caseDef}
	return suite
}

func baseAssertions(protocol string) []types.Assertion {
	common := []types.Assertion{
		{ID: "response.body.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body", Op: "exists", Message: "响应体必须是可解析的 JSON 对象。", Source: "builtin"},
		{ID: "metrics.total_ms.gte_zero", CaseID: "basic-response-shape", Level: "warn", Path: "metrics.total_ms", Op: "gte", Value: float64(0), Message: "请求耗时必须可用。", Source: "builtin"},
	}
	switch protocol {
	case types.ProtocolOpenAIResponses:
		return append(common,
			types.Assertion{ID: "responses.id.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body.id", Op: "exists", Message: "Responses 响应体必须包含 id 字段。", Source: "builtin"},
			types.Assertion{ID: "responses.output.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body.output", Op: "exists", Message: "Responses 响应体必须包含 output 字段。", Source: "builtin"},
		)
	case types.ProtocolAnthropicMessages:
		return append(common,
			types.Assertion{ID: "anthropic.id.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body.id", Op: "exists", Message: "Anthropic 响应体必须包含 id 字段。", Source: "builtin"},
			types.Assertion{ID: "anthropic.content.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body.content", Op: "exists", Message: "Anthropic 响应体必须包含 content 字段。", Source: "builtin"},
		)
	default:
		return append(common,
			types.Assertion{ID: "chat.id.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body.id", Op: "exists", Message: "响应体必须包含 id 字段。", Source: "builtin"},
			types.Assertion{ID: "chat.choices.exists", CaseID: "basic-response-shape", Level: "error", Path: "response.body.choices", Op: "exists", Message: "Chat Completions 响应体必须包含 choices 字段。", Source: "builtin"},
		)
	}
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
