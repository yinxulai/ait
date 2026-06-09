package client

import (
	"strings"

	"github.com/yinxulai/ait/internal/i18n"
)

// ErrorType represents the category of API errors
type ErrorType int

const (
	ErrUnknown ErrorType = iota
	ErrAuth
	ErrQuota
	ErrRateLimit
	ErrTimeout
	ErrNetwork
	ErrInvalidRequest
	ErrModelNotFound
	ErrServerError
)

// ClassifyError classifies an error message and returns its type
func ClassifyError(errMsg string) ErrorType {
	errLower := strings.ToLower(errMsg)

	// Authentication errors
	if containsAny(errLower, []string{"unauthorized", "invalid api key", "authentication failed", "api key not found", "401"}) {
		return ErrAuth
	}

	// Quota/billing errors
	if containsAny(errLower, []string{"exceeded quota", "insufficient quota", "billing", "payment required", "402"}) {
		return ErrQuota
	}

	// Rate limit errors
	if containsAny(errLower, []string{"rate limit", "too many requests", "429", "throttle"}) {
		return ErrRateLimit
	}

	// Timeout errors
	if containsAny(errLower, []string{"timeout", "deadline exceeded", "request timed out", "504"}) {
		return ErrTimeout
	}

	// Network errors
	if containsAny(errLower, []string{"connection refused", "connection reset", "no route to host", "dns failure", "network unreachable", "tls handshake"}) {
		return ErrNetwork
	}

	// Invalid request errors
	if containsAny(errLower, []string{"invalid request", "invalid parameter", "400", "validation error", "missing required field"}) {
		return ErrInvalidRequest
	}

	// Model not found
	if containsAny(errLower, []string{"model not found", "model does not exist", "model_id_invalid", "404"}) {
		return ErrModelNotFound
	}

	// Server errors
	if containsAny(errLower, []string{"500", "502", "503", "internal server error", "service unavailable", "bad gateway"}) {
		return ErrServerError
	}

	return ErrUnknown
}

// UserErrorHint returns a user-friendly hint for the given error
func UserErrorHint(errMsg string) string {
	errType := ClassifyError(errMsg)
	lang := i18n.Active()

	switch errType {
	case ErrAuth:
		if lang == i18n.EN {
			return "Authentication failed. Please check: 1) API key is correct, 2) API key has not expired, 3) API key has proper permissions."
		}
		return "认证失败。请检查：1) API Key 是否正确，2) API Key 是否过期，3) API Key 是否有相应权限。"

	case ErrQuota:
		if lang == i18n.EN {
			return "Quota exceeded. Please check your account balance or upgrade your plan."
		}
		return "配额超限。请检查账户余额或升级套餐。"

	case ErrRateLimit:
		if lang == i18n.EN {
			return "Rate limit exceeded. Please reduce concurrency or wait before retrying, or upgrade to higher rate limits."
		}
		return "请求频率超限。请降低并发数或稍后重试，或升级更高频率限制。"

	case ErrTimeout:
		if lang == i18n.EN {
			return "Request timed out. Please check: 1) Network connection, 2) Proxy settings, 3) Increase timeout value."
		}
		return "请求超时。请检查：1) 网络连接，2) Proxy 配置，3) 增加超时时间。"

	case ErrNetwork:
		if lang == i18n.EN {
			return "Network error. Please check: 1) Internet connection, 2) Proxy configuration, 3) Firewall settings."
		}
		return "网络错误。请检查：1) 网络连接，2) Proxy 配置，3) 防火墙设置。"

	case ErrInvalidRequest:
		if lang == i18n.EN {
			return "Invalid request. Please check the request parameters and try again."
		}
		return "请求无效。请检查请求参数后重试。"

	case ErrModelNotFound:
		if lang == i18n.EN {
			return "Model not found. Please check the model name is correct."
		}
		return "模型不存在。请检查模型名称是否正确。"

	case ErrServerError:
		if lang == i18n.EN {
			return "Server error. Please try again later or check the service status."
		}
		return "服务器错误。请稍后重试或检查服务状态。"

	default:
		if lang == i18n.EN {
			return "An error occurred. Please check the error message and try again."
		}
		return "发生错误。请检查错误信息后重试。"
	}
}

// EnhanceErrorMessage appends user-friendly hint to the original error message
func EnhanceErrorMessage(errMsg string) string {
	hint := UserErrorHint(errMsg)
	if hint == "" {
		return errMsg
	}
	return errMsg + "\nHint: " + hint
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
