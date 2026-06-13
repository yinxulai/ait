package server

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/modes/integrity"
	"github.com/yinxulai/ait/internal/server/modes/turbo"
	"github.com/yinxulai/ait/internal/server/types"
)

// ProtocolMeta describes a model API protocol supported by AIT.
type ProtocolMeta struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	DefaultEndpointURL string `json:"default_endpoint_url"`
}

// ValidateTaskConfig validates and normalizes task configuration before it is persisted or executed.
func (s *serverImpl) ValidateTaskConfig(cfg TaskConfig) (TaskConfig, error) {
	cfg.Name = strings.TrimSpace(cfg.Name)
	if cfg.Name == "" {
		return TaskConfig{}, errors.New("name is required")
	}

	input := cfg.Input
	input.Mode = normalizeRunMode(input)
	input.Protocol = types.NormalizeProtocol(input.Protocol)
	if err := validateProtocol(input.Protocol); err != nil {
		return TaskConfig{}, err
	}
	if strings.TrimSpace(input.Model) == "" {
		return TaskConfig{}, errors.New("input.model is required")
	}

	switch input.RunMode() {
	case "standard":
		input.Turbo = false
		input.Integrity.Enabled = false
		if err := validatePrompt(input); err != nil {
			return TaskConfig{}, err
		}
		if input.Concurrency <= 0 {
			return TaskConfig{}, errors.New("input.concurrency must be greater than 0")
		}
		if input.Count <= 0 {
			return TaskConfig{}, errors.New("input.count must be greater than 0")
		}
	case "turbo":
		input.Turbo = true
		input.Integrity.Enabled = false
		if err := validatePrompt(input); err != nil {
			return TaskConfig{}, err
		}
		input.TurboConfig = turbo.NormalizeConfig(input.TurboConfig, input.Count)
		if input.TurboConfig.MaxConcurrency < input.TurboConfig.InitConcurrency {
			return TaskConfig{}, errors.New("turbo_config.max_concurrency must be greater than or equal to init_concurrency")
		}
	case "integrity":
		input.Turbo = false
		input.Integrity.Enabled = true
		if strings.TrimSpace(input.Integrity.Suite) == "" {
			return TaskConfig{}, errors.New("integrity.suite is required")
		}
		if _, err := s.GetIntegritySuite(input.Protocol, input.Integrity.Suite); err != nil {
			return TaskConfig{}, err
		}
	default:
		return TaskConfig{}, fmt.Errorf("unsupported input.mode: %s", input.RunMode())
	}

	cfg.Input = input
	return cfg, nil
}

// ListProtocols returns protocol metadata used by Web/TUI forms.
func (s *serverImpl) ListProtocols() []ProtocolMeta {
	return supportedProtocols()
}

// ListIntegritySuites returns available integrity suites for a protocol.
func (s *serverImpl) ListIntegritySuites(protocol string) ([]types.IntegritySuite, error) {
	protocol = types.NormalizeProtocol(protocol)
	if err := validateProtocol(protocol); err != nil {
		return nil, err
	}

	suiteIDs := map[string]struct{}{
		integrity.BuiltinSuite(protocol, "").ID: {},
	}
	if s.rulesManager != nil {
		if index := s.rulesManager.GetIndex(); index != nil {
			for _, rule := range index.Rules {
				if rule.Suite == "" || rule.Suite == "*" {
					continue
				}
				if rule.Protocol == protocol || rule.Protocol == "*" {
					suiteIDs[rule.Suite] = struct{}{}
				}
			}
		}
	}

	suites := make([]types.IntegritySuite, 0, len(suiteIDs))
	for suiteID := range suiteIDs {
		suite, err := s.GetIntegritySuite(protocol, suiteID)
		if err != nil {
			return nil, err
		}
		suites = append(suites, suite)
	}
	return suites, nil
}

// GetIntegritySuite loads one integrity suite and merges built-in/cached/custom rules when available.
func (s *serverImpl) GetIntegritySuite(protocol, suiteID string) (types.IntegritySuite, error) {
	protocol = types.NormalizeProtocol(protocol)
	if err := validateProtocol(protocol); err != nil {
		return types.IntegritySuite{}, err
	}
	input := types.Input{
		Protocol: protocol,
		Integrity: types.IntegrityConfig{
			Enabled: true,
			Suite:   strings.TrimSpace(suiteID),
		},
	}
	return integrity.LoadSuiteWithManager(input, s.rulesManager)
}

func normalizeRunMode(input types.Input) string {
	mode := strings.ToLower(strings.TrimSpace(input.Mode))
	if mode != "" {
		return mode
	}
	return input.RunMode()
}

func validatePrompt(input types.Input) error {
	if strings.TrimSpace(input.PromptText) == "" && strings.TrimSpace(input.PromptFile) == "" && input.PromptLength <= 0 {
		return errors.New("standard and turbo tasks require prompt_text, prompt_file or prompt_length")
	}
	return nil
}

func validateProtocol(protocol string) error {
	if _, err := client.NewClient(types.Input{Protocol: protocol, Model: "__validation__"}, nil); err != nil {
		return err
	}
	return nil
}

func supportedProtocols() []ProtocolMeta {
	return []ProtocolMeta{
		{ID: types.ProtocolOpenAICompletions, Name: "OpenAI Chat Completions", DefaultEndpointURL: types.DefaultEndpointURL(types.ProtocolOpenAICompletions)},
		{ID: types.ProtocolOpenAIResponses, Name: "OpenAI Responses", DefaultEndpointURL: types.DefaultEndpointURL(types.ProtocolOpenAIResponses)},
		{ID: types.ProtocolAnthropicMessages, Name: "Anthropic Messages", DefaultEndpointURL: types.DefaultEndpointURL(types.ProtocolAnthropicMessages)},
	}
}
