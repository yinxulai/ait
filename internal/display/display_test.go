package display

import (
	"testing"
)

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
