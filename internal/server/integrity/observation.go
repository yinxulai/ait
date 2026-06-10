package integrity

import (
	"encoding/json"
	"strings"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/types"
)

func BuildObservation(input types.Input, c types.IntegrityCase, metrics *client.ResponseMetrics, requestIndex int, err error) map[string]any {
	obs := map[string]any{
		"task": map[string]any{
			"protocol":    input.NormalizedProtocol(),
			"model":       input.Model,
			"endpoint":    input.ResolvedEndpointURL(),
			"stream":      input.Stream,
			"concurrency": input.Concurrency,
			"count":       input.Count,
			"prompt": map[string]any{
				"mode":   input.PromptMode,
				"length": input.PromptLength,
			},
		},
		"case": map[string]any{
			"id":         c.ID,
			"category":   c.Category,
			"capability": c.Capability,
			"required":   c.Required,
		},
		"request": map[string]any{
			"index": requestIndex,
		},
	}

	if metrics == nil {
		message := ""
		if err != nil {
			message = err.Error()
		}
		obs["response"] = map[string]any{
			"body":        nil,
			"parse_error": message,
			"error":       message,
		}
		obs["metrics"] = map[string]any{}
		obs["network"] = map[string]any{}
		return obs
	}

	requestBody, requestParseError := parseJSON(metrics.RequestBody)
	responseBody, responseParseError := parseResponseBody(metrics.ResponseBody)
	response := map[string]any{
		"body":        responseBody,
		"parse_error": responseParseError,
		"error":       metrics.ErrorMessage,
	}
	obs["request"] = map[string]any{
		"index":       requestIndex,
		"body":        requestBody,
		"parse_error": requestParseError,
	}
	obs["response"] = response
	obs["metrics"] = map[string]any{
		"total_ms":      float64(metrics.TotalTime.Milliseconds()),
		"ttft_ms":       float64(metrics.TimeToFirstToken.Milliseconds()),
		"tps":           tokensPerSecond(metrics),
		"input_tokens":  float64(metrics.PromptTokens),
		"output_tokens": float64(metrics.CompletionTokens),
		"cached_tokens": float64(metrics.CachedInputTokens),
	}
	obs["network"] = map[string]any{
		"dns_ms":     float64(metrics.DNSTime.Milliseconds()),
		"connect_ms": float64(metrics.ConnectTime.Milliseconds()),
		"tls_ms":     float64(metrics.TLSHandshakeTime.Milliseconds()),
		"target_ip":  metrics.TargetIP,
	}
	return obs
}

func parseResponseBody(body string) (any, any) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil, "empty response body"
	}
	if strings.HasPrefix(trimmed, "data:") {
		return parseStreamEvents(trimmed), nil
	}
	return parseJSON(trimmed)
}

func parseJSON(body string) (any, any) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err.Error()
	}
	return value, nil
}

func parseStreamEvents(body string) []any {
	events := []any{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if data == "" || data == "[DONE]" {
			continue
		}
		value, _ := parseJSON(data)
		if value != nil {
			events = append(events, value)
		}
	}
	return events
}

func tokensPerSecond(metrics *client.ResponseMetrics) float64 {
	if metrics == nil || metrics.TotalTime <= 0 || metrics.CompletionTokens <= 0 {
		return 0
	}
	return float64(metrics.CompletionTokens) / metrics.TotalTime.Seconds()
}
