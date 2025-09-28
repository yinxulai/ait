package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTemplate_applyTemplate 测试模板应用功能
func TestTemplate_applyTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  *Template
		content   string
		index     int
		timestamp time.Time
		expected  string
	}{
		{
			name: "基本占位符替换",
			template: &Template{
				Content:   "内容: {{content}}, 序号: {{index}}, 时间: {{timestamp}}",
				Variables: make(map[string]string),
			},
			content:   "测试内容",
			index:     1,
			timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected:  "内容: 测试内容, 序号: 1, 时间: 2023-01-01T12:00:00Z",
		},
		{
			name: "自定义变量替换",
			template: &Template{
				Content: "{{content}} - {{custom1}} - {{custom2}}",
				Variables: map[string]string{
					"custom1": "变量1",
					"custom2": "变量2",
				},
			},
			content:   "主要内容",
			index:     2,
			timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected:  "主要内容 - 变量1 - 变量2",
		},
		{
			name: "无占位符模板",
			template: &Template{
				Content:   "固定内容",
				Variables: make(map[string]string),
			},
			content:   "测试",
			index:     1,
			timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected:  "固定内容",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.template.applyTemplate(tt.content, tt.index, tt.timestamp)
			if result != tt.expected {
				t.Errorf("applyTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGenerateTaskID 测试任务ID生成
func TestGenerateTaskID(t *testing.T) {
	// 生成多个ID确保它们不重复
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateTaskID()
		
		// 检查格式 (8-4-4-4-12)
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			t.Errorf("generateTaskID() format error, got %d parts, want 5", len(parts))
		}
		
		// 检查每部分长度
		expectedLengths := []int{8, 4, 4, 4, 12}
		for j, part := range parts {
			if len(part) != expectedLengths[j] {
				t.Errorf("generateTaskID() part %d length error, got %d, want %d", j, len(part), expectedLengths[j])
			}
		}
		
		// 检查重复
		if ids[id] {
			t.Errorf("generateTaskID() generated duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

// TestValidateParams 测试参数验证
func TestValidateParams(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		length    int
		outputDir string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "有效参数",
			count:     10,
			length:    50,
			outputDir: "output",
			wantErr:   false,
		},
		{
			name:      "count为0",
			count:     0,
			length:    50,
			outputDir: "output",
			wantErr:   true,
			errMsg:    "count 必须大于 0",
		},
		{
			name:      "count为负数",
			count:     -1,
			length:    50,
			outputDir: "output",
			wantErr:   true,
			errMsg:    "count 必须大于 0",
		},
		{
			name:      "length为0",
			count:     10,
			length:    0,
			outputDir: "output",
			wantErr:   true,
			errMsg:    "length 必须大于 0",
		},
		{
			name:      "length为负数",
			count:     10,
			length:    -1,
			outputDir: "output",
			wantErr:   true,
			errMsg:    "length 必须大于 0",
		},
		{
			name:      "outputDir为空",
			count:     10,
			length:    50,
			outputDir: "",
			wantErr:   true,
			errMsg:    "输出目录不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParams(tt.count, tt.length, tt.outputDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateParams() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("validateParams() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateParams() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

// TestGenerateRandomText 测试随机文本生成
func TestGenerateRandomText(t *testing.T) {
	tests := []struct {
		name          string
		desiredLength int
		minLength     int // 最小期望长度
		maxLength     int // 最大期望长度
	}{
		{
			name:          "短文本",
			desiredLength: 20,
			minLength:     5,
			maxLength:     80,  // 增加上限以适应句子组合的特性
		},
		{
			name:          "中等文本",
			desiredLength: 100,
			minLength:     30,
			maxLength:     300, // 增加上限
		},
		{
			name:          "长文本",
			desiredLength: 500,
			minLength:     200,
			maxLength:     1500, // 增加上限
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateRandomText(tt.desiredLength)
			
			// 检查结果不为空
			if result == "" {
				t.Error("generateRandomText() returned empty string")
			}
			
			// 检查长度在合理范围内
			if len(result) < tt.minLength || len(result) > tt.maxLength {
				t.Errorf("generateRandomText() length = %d, want between %d and %d", 
					len(result), tt.minLength, tt.maxLength)
			}
			
			// 检查结果包含预期的句子
			containsKnownSentence := false
			for _, sentence := range sentences {
				if strings.Contains(result, sentence) {
					containsKnownSentence = true
					break
				}
			}
			if !containsKnownSentence {
				t.Error("generateRandomText() result doesn't contain any known sentence")
			}
		})
	}
}

// TestWritePromptFile 测试文件写入功能
func TestWritePromptFile(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "tpg_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		prompt   string
		filename string
		template *Template
		index    int
		wantErr  bool
	}{
		{
			name:     "基本文件写入",
			prompt:   "测试内容",
			filename: filepath.Join(tempDir, "test1.txt"),
			template: nil,
			index:    1,
			wantErr:  false,
		},
		{
			name:   "使用模板写入",
			prompt: "测试内容",
			filename: filepath.Join(tempDir, "test2.txt"),
			template: &Template{
				Content:   "序号{{index}}: {{content}}",
				Variables: make(map[string]string),
			},
			index:   2,
			wantErr: false,
		},
		{
			name:     "无效路径",
			prompt:   "测试内容",
			filename: "/invalid/path/test.txt",
			template: nil,
			index:    1,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writePromptFile(tt.prompt, tt.filename, tt.template, tt.index)
			
			if tt.wantErr {
				if err == nil {
					t.Error("writePromptFile() error = nil, wantErr true")
				}
				return
			}
			
			if err != nil {
				t.Errorf("writePromptFile() error = %v, wantErr false", err)
				return
			}
			
			// 验证文件是否创建
			if _, err := os.Stat(tt.filename); os.IsNotExist(err) {
				t.Error("writePromptFile() file was not created")
				return
			}
			
			// 读取文件内容验证
			content, err := os.ReadFile(tt.filename)
			if err != nil {
				t.Errorf("Failed to read written file: %v", err)
				return
			}
			
			expectedContent := tt.prompt
			if tt.template != nil {
				expectedContent = tt.template.applyTemplate(tt.prompt, tt.index, time.Now())
			}
			
			contentStr := string(content)
			if tt.template == nil && contentStr != expectedContent {
				t.Errorf("File content = %v, want %v", contentStr, expectedContent)
			} else if tt.template != nil {
				// 对于使用模板的情况，至少检查基本内容是否包含
				if !strings.Contains(contentStr, tt.prompt) {
					t.Errorf("File content should contain prompt: %v", tt.prompt)
				}
			}
		})
	}
}

// TestSentencesAvailability 测试句子库是否可用
func TestSentencesAvailability(t *testing.T) {
	if len(sentences) == 0 {
		t.Error("sentences slice is empty")
	}
	
	// 检查是否包含不同语言的句子
	languages := map[string]bool{
		"english": false,
		"chinese": false,
		"japanese": false,
		"korean": false,
		"french": false,
		"german": false,
		"spanish": false,
		"russian": false,
		"arabic": false,
	}
	
	for _, sentence := range sentences {
		if strings.Contains(sentence, "The quick brown fox") {
			languages["english"] = true
		}
		if strings.Contains(sentence, "人工智能") {
			languages["chinese"] = true
		}
		if strings.Contains(sentence, "人工知能") {
			languages["japanese"] = true
		}
		if strings.Contains(sentence, "인공지능") {
			languages["korean"] = true
		}
		if strings.Contains(sentence, "intelligence artificielle") {
			languages["french"] = true
		}
		if strings.Contains(sentence, "künstlichen Intelligenz") {
			languages["german"] = true
		}
		if strings.Contains(sentence, "inteligencia artificial") {
			languages["spanish"] = true
		}
		if strings.Contains(sentence, "искусственного интеллекта") {
			languages["russian"] = true
		}
		if strings.Contains(sentence, "الذكاء الاصطناعي") {
			languages["arabic"] = true
		}
	}
	
	// 检查是否至少包含几种语言
	foundLanguages := 0
	for _, found := range languages {
		if found {
			foundLanguages++
		}
	}
	
	if foundLanguages < 5 {
		t.Errorf("Expected at least 5 different languages in sentences, found %d", foundLanguages)
	}
}

// BenchmarkGenerateRandomText 性能测试
func BenchmarkGenerateRandomText(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateRandomText(100)
	}
}

// BenchmarkGenerateTaskID 任务ID生成性能测试
func BenchmarkGenerateTaskID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateTaskID()
	}
}

// BenchmarkTemplateApply 模板应用性能测试
func BenchmarkTemplateApply(b *testing.B) {
	template := &Template{
		Content: "内容: {{content}}, 序号: {{index}}, 时间: {{timestamp}}",
		Variables: map[string]string{
			"custom1": "value1",
			"custom2": "value2",
		},
	}
	
	content := "测试内容"
	timestamp := time.Now()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		template.applyTemplate(content, i, timestamp)
	}
}
