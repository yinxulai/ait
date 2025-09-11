package display

import (
	"testing"
)

func TestTruncatePrompt(t *testing.T) {
	// 测试用例 1: 短提示词
	t.Run("Short prompt", func(t *testing.T) {
		result := truncatePrompt("你好")
		expected := "你好 (长度: 2)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	// 测试用例 2: 长提示词
	t.Run("Long prompt", func(t *testing.T) {
		longPrompt := "这是一个非常长的测试提示词，用于测试truncatePrompt函数的截断功能和长度显示"
		result := truncatePrompt(longPrompt)
		expected := "这是一个非常长的测试提示词，用于测试truncatePrompt函数的截断功能和长度显示 (长度: 44)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	// 测试用例 2.5: 超长提示词（需要截断）
	t.Run("Very long prompt that needs truncation", func(t *testing.T) {
		veryLongPrompt := "这是一个非常长的测试提示词，用于测试truncatePrompt函数的截断功能和长度显示，这个字符串超过五十个字符所以会被截断处理"
		result := truncatePrompt(veryLongPrompt)
		expected := "这是一个非常长的测试提示词，用于测试truncatePrompt函数的截断功能和长度显示，这个... (长度: 65)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	// 测试用例 3: 恰好50字符的提示词
	t.Run("Exactly 50 characters", func(t *testing.T) {
		promptWith50Chars := "这个测试字符串包含恰好五十个字符用于测试边界条件是否能够正确处理各种情况的测试案例增加字符再加五十个"
		result := truncatePrompt(promptWith50Chars)
		expected := "这个测试字符串包含恰好五十个字符用于测试边界条件是否能够正确处理各种情况的测试案例增加字符再加五十个 (长度: 50)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	// 测试用例 4: 空字符串
	t.Run("Empty string", func(t *testing.T) {
		result := truncatePrompt("")
		expected := " (长度: 0)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	// 测试用例 5: 混合中英文
	t.Run("Mixed Chinese and English", func(t *testing.T) {
		mixedPrompt := "Hello 世界 123"
		result := truncatePrompt(mixedPrompt)
		expected := "Hello 世界 123 (长度: 12)"
		if result != expected {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})
}

func TestShowErrorsReport(t *testing.T) {
	displayer := New()

	// 测试用例 1: 空错误列表
	t.Run("Empty errors", func(t *testing.T) {
		var errors []*string
		displayer.ShowErrorsReport(errors)
		// 应该不输出任何内容，函数应该直接返回
	})

	// 测试用例 2: 单个错误
	t.Run("Single error", func(t *testing.T) {
		error1 := "API 连接超时"
		errors := []*string{&error1}
		displayer.ShowErrorsReport(errors)
	})

	// 测试用例 3: 多个不同错误
	t.Run("Multiple different errors", func(t *testing.T) {
		error1 := "API 连接超时"
		error2 := "认证失败: 无效的 API 密钥"
		error3 := "模型不存在或无法访问"
		errors := []*string{&error1, &error2, &error3}
		displayer.ShowErrorsReport(errors)
	})

	// 测试用例 4: 重复错误统计
	t.Run("Duplicate errors counting", func(t *testing.T) {
		error1 := "API 连接超时"
		error2 := "认证失败: 无效的 API 密钥"
		error3 := "API 连接超时" // 重复错误
		error4 := "认证失败: 无效的 API 密钥" // 重复错误
		error5 := "API 连接超时" // 又一个重复错误
		errors := []*string{&error1, &error2, &error3, &error4, &error5}
		displayer.ShowErrorsReport(errors)
	})

	// 测试用例 5: 包含长错误消息
	t.Run("Long error messages", func(t *testing.T) {
		longError := "这是一个非常长的错误消息，用于测试当错误消息超过100个字符时的截断功能。这个错误消息包含了大量的细节信息，比如堆栈跟踪、请求参数、响应状态码等等。"
		shortError := "短错误消息"
		errors := []*string{&longError, &shortError, &longError} // 包含重复的长错误
		displayer.ShowErrorsReport(errors)
	})

	// 测试用例 6: 包含nil指针
	t.Run("With nil pointers", func(t *testing.T) {
		error1 := "正常错误消息"
		errors := []*string{&error1, nil, &error1, nil}
		displayer.ShowErrorsReport(errors)
	})
}
