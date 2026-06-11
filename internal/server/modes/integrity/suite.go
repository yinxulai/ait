package integrity

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	Version     string              `json:"version"`
	Suite       string              `json:"suite"`
	Description string              `json:"description"`
	Cases       []types.IntegrityCase `json:"cases"`       // 新：完整的测试用例
	Assertions  []types.Assertion     `json:"assertions"`  // 旧：兼容性保留
}

func LoadSuite(input types.Input) (types.IntegritySuite, error) {
	return LoadSuiteWithManager(input, nil)
}

func LoadSuiteWithManager(input types.Input, rulesManager *RulesManager) (types.IntegritySuite, error) {
	suite := BuiltinSuite(input.NormalizedProtocol(), input.Integrity.Suite)
	
	// 自动加载内置规则
	if rulesManager != nil {
		builtinFiles, err := rulesManager.GetRuleFiles(input.NormalizedProtocol(), suite.ID)
		if err == nil {
			for _, file := range builtinFiles {
				rules, err := LoadRuleFile(file)
				if err != nil {
					// 内置规则加载失败只警告，不中断
					slog.Warn("failed to load builtin rule, skipping", "file", file, "error", err)
					continue
				}
				// 跳过 suite 检查（内置规则可以使用 * 通配）
				suite = MergeCases(suite, rules.Cases, file)
				suite = MergeAssertions(suite, rules.Assertions, file)
			}
		}
	}
	
	// 加载用户自定义规则
	for _, file := range input.Integrity.RuleFiles {
		rules, err := LoadRuleFile(file)
		if err != nil {
			return types.IntegritySuite{}, err
		}
		if strings.TrimSpace(rules.Suite) != "" && rules.Suite != suite.ID && rules.Suite != "*" {
			return types.IntegritySuite{}, fmt.Errorf("rule file %q targets suite %q, current suite is %q", file, rules.Suite, suite.ID)
		}
		suite = MergeCases(suite, rules.Cases, file)
		suite = MergeAssertions(suite, rules.Assertions, file)
	}
	return suite, nil
}

func LoadRuleFile(path string) (RuleFile, error) {
	return LoadRuleFileWithContext(context.Background(), path)
}

func LoadRuleFileWithContext(ctx context.Context, path string) (RuleFile, error) {
	// 只支持本地文件路径
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
	
	// 设置 assertions 的 source（兼容旧格式）
	for i := range rules.Assertions {
		if strings.TrimSpace(rules.Assertions[i].Source) == "" {
			rules.Assertions[i].Source = path
		}
	}
	
	// 设置 cases 中 assertions 的 source（新格式）
	for i := range rules.Cases {
		for j := range rules.Cases[i].Assertions {
			if strings.TrimSpace(rules.Cases[i].Assertions[j].Source) == "" {
				rules.Cases[i].Assertions[j].Source = path
			}
			// 设置 CaseID（如果没有）
			if strings.TrimSpace(rules.Cases[i].Assertions[j].CaseID) == "" {
				rules.Cases[i].Assertions[j].CaseID = rules.Cases[i].ID
			}
		}
	}
	
	return rules, nil
}

// MergeCases 合并完整的测试用例到 suite
func MergeCases(suite types.IntegritySuite, cases []types.IntegrityCase, source string) types.IntegritySuite {
	if len(cases) == 0 {
		return suite
	}

	caseIndex := make(map[string]int, len(suite.Cases))
	for i := range suite.Cases {
		caseIndex[suite.Cases[i].ID] = i
	}

	for _, newCase := range cases {
		idx, exists := caseIndex[newCase.ID]
		if exists {
			// Case 已存在，合并 assertions
			existingCase := suite.Cases[idx]
			
			// 如果新 case 定义了 request，使用新的（优先级更高）
			if newCase.Request.Prompt != "" {
				existingCase.Request = newCase.Request
			}
			
			// 更新其他字段（如果新 case 有定义）
			if newCase.Name != "" {
				existingCase.Name = newCase.Name
			}
			if newCase.Description != "" {
				existingCase.Description = newCase.Description
			}
			if newCase.Category != "" {
				existingCase.Category = newCase.Category
			}
			if newCase.Capability != "" {
				existingCase.Capability = newCase.Capability
			}
			if newCase.TimeoutMS > 0 {
				existingCase.TimeoutMS = newCase.TimeoutMS
			}
			
			// 合并 assertions
			for _, assertion := range newCase.Assertions {
				if assertion.Source == "" {
					assertion.Source = source
				}
				if assertion.CaseID == "" {
					assertion.CaseID = newCase.ID
				}
				
				// 检查是否已存在相同 ID 的 assertion
				replaced := false
				if assertion.ID != "" {
					for j := range existingCase.Assertions {
						if existingCase.Assertions[j].ID == assertion.ID {
							existingCase.Assertions[j] = assertion
							replaced = true
							break
						}
					}
				}
				if !replaced {
					existingCase.Assertions = append(existingCase.Assertions, assertion)
				}
			}
			
			suite.Cases[idx] = existingCase
		} else {
			// 新的 case，直接添加
			for i := range newCase.Assertions {
				if newCase.Assertions[i].Source == "" {
					newCase.Assertions[i].Source = source
				}
				if newCase.Assertions[i].CaseID == "" {
					newCase.Assertions[i].CaseID = newCase.ID
				}
			}
			suite.Cases = append(suite.Cases, newCase)
			caseIndex[newCase.ID] = len(suite.Cases) - 1
		}
	}

	return suite
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
