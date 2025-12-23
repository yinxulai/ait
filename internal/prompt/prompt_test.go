package prompt

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestGeneratePromptByLength(t *testing.T) {
	tests := []struct {
		name     string
		length   int
		validate func(result string) bool
	}{
		{
			name:   "零长度",
			length: 0,
			validate: func(result string) bool {
				return result == ""
			},
		},
		{
			name:   "负数长度",
			length: -10,
			validate: func(result string) bool {
				return result == ""
			},
		},
		{
			name:   "短文本(50字符)",
			length: 50,
			validate: func(result string) bool {
				actualLen := utf8.RuneCountInString(result)
				return actualLen == 50
			},
		},
		{
			name:   "中等长度(200字符)",
			length: 200,
			validate: func(result string) bool {
				actualLen := utf8.RuneCountInString(result)
				return actualLen == 200
			},
		},
		{
			name:   "长文本(1000字符)",
			length: 1000,
			validate: func(result string) bool {
				actualLen := utf8.RuneCountInString(result)
				return actualLen == 1000
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePromptByLength(tt.length)
			if !tt.validate(result) {
				actualLen := utf8.RuneCountInString(result)
				t.Errorf("GeneratePromptByLength(%d) 返回长度 = %d, 验证失败", tt.length, actualLen)
			}

			// 验证生成的内容不是空白字符
			if tt.length > 0 && strings.TrimSpace(result) == "" {
				t.Errorf("GeneratePromptByLength(%d) 生成了空白内容", tt.length)
			}
		})
	}
}

func TestLoadPromptByLength(t *testing.T) {
	tests := []struct {
		name      string
		length    int
		wantError bool
	}{
		{
			name:      "有效长度",
			length:    100,
			wantError: false,
		},
		{
			name:      "零长度应该返回错误",
			length:    0,
			wantError: true,
		},
		{
			name:      "负数长度应该返回错误",
			length:    -10,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := LoadPromptByLength(tt.length)

			if tt.wantError {
				if err == nil {
					t.Errorf("LoadPromptByLength(%d) 期望返回错误，但没有错误", tt.length)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadPromptByLength(%d) 返回错误: %v", tt.length, err)
				return
			}

			if source == nil {
				t.Errorf("LoadPromptByLength(%d) 返回 nil source", tt.length)
				return
			}

			// 验证 PromptSource 的属性
			if source.IsFile {
				t.Errorf("LoadPromptByLength 不应该设置 IsFile = true")
			}

			if len(source.Contents) != 1 {
				t.Errorf("LoadPromptByLength 应该返回 1 个内容，实际返回 %d 个", len(source.Contents))
			}

			// 验证内容长度
			content := source.GetRandomContent()
			actualLen := utf8.RuneCountInString(content)
			if actualLen != tt.length {
				t.Errorf("LoadPromptByLength(%d) 返回内容长度 = %d", tt.length, actualLen)
			}

			// 验证 DisplayText
			if source.DisplayText == "" {
				t.Errorf("LoadPromptByLength 应该设置 DisplayText")
			}
		})
	}
}

func TestPromptSourceWithGeneratedContent(t *testing.T) {
	length := 500
	source, err := LoadPromptByLength(length)
	if err != nil {
		t.Fatalf("LoadPromptByLength 失败: %v", err)
	}

	// 测试多次调用 GetRandomContent 应该返回相同的内容
	content1 := source.GetRandomContent()
	content2 := source.GetRandomContent()

	if content1 != content2 {
		t.Errorf("GetRandomContent 在单内容源时应该返回相同内容")
	}

	// 测试 Count 方法
	if source.Count() != 1 {
		t.Errorf("Count() = %d, 期望 1", source.Count())
	}
}

func TestGeneratePromptByLengthQuality(t *testing.T) {
	// 测试生成的内容质量
	length := 300
	content := GeneratePromptByLength(length)

	// 验证不包含重复的分隔符
	if strings.Contains(content, "  ") {
		t.Errorf("生成的内容包含连续的空格")
	}

	// 验证包含中文字符
	hasChineseChar := false
	for _, r := range content {
		if r >= 0x4e00 && r <= 0x9fff {
			hasChineseChar = true
			break
		}
	}
	if !hasChineseChar {
		t.Errorf("生成的内容应该包含中文字符")
	}

	// 验证内容不以空格开头或结尾
	if strings.HasPrefix(content, " ") || strings.HasSuffix(content, " ") {
		t.Errorf("生成的内容不应该以空格开头或结尾")
	}
}
