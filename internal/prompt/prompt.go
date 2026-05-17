package prompt

import (
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// PromptSource 表示prompt的来源信息
type PromptSource struct {
	IsFile         bool     // 是否来自文件
	FilePaths      []string // 文件路径列表
	Contents       []string // prompt内容列表（仅用于非文件内容）
	SystemContent  string   // 固定的系统消息内容（仅 generated 模式使用，用于触发前缀缓存）
	DisplayText    string   // 用于显示的文本
	ShouldTruncate bool     // 是否需要截断显示（对于已经包含长度信息的内容，不需要再次处理）
}

// LoadPrompts 解析prompt参数，只处理字符串内容
func LoadPrompts(promptArg string) (*PromptSource, error) {
	return &PromptSource{
		IsFile:         false,
		FilePaths:      nil,
		Contents:       []string{promptArg},
		DisplayText:    promptArg,
		ShouldTruncate: true,
	}, nil
}

// LoadPromptsFromFile 从文件路径加载prompt，支持单文件和通配符
func LoadPromptsFromFile(pathPattern string) (*PromptSource, error) {
	// 检查是否包含通配符
	if strings.Contains(pathPattern, "*") || strings.Contains(pathPattern, "?") || strings.Contains(pathPattern, "[") {
		// 使用glob模式匹配多个文件
		return loadMultipleFiles(pathPattern)
	} else {
		// 单个文件
		return loadSingleFile(pathPattern)
	}
}

// loadSingleFile 加载单个文件
func loadSingleFile(filePath string) (*PromptSource, error) {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	return &PromptSource{
		IsFile:         true,
		FilePaths:      []string{filePath},
		Contents:       nil, // 不预加载内容
		DisplayText:    fmt.Sprintf("文件: %s (1个)", filePath),
		ShouldTruncate: false, // 文件显示不需要截断
	}, nil
}

// loadMultipleFiles 使用glob模式加载多个文件
func loadMultipleFiles(pattern string) (*PromptSource, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob模式解析失败 %s: %v", pattern, err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("没有找到匹配的文件: %s", pattern)
	}

	var filePaths []string

	for _, match := range matches {
		// 检查是否为文件（跳过目录）
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}

		filePaths = append(filePaths, match)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("没有成功加载任何文件: %s", pattern)
	}

	return &PromptSource{
		IsFile:         true,
		FilePaths:      filePaths,
		Contents:       nil, // 不预加载内容
		DisplayText:    fmt.Sprintf("文件: %s (%d个)", pattern, len(filePaths)),
		ShouldTruncate: false, // 文件显示不需要截断
	}, nil
}

// GetSystemContent 返回系统消息内容（固定的大段上下文，用于前缀缓存）。
// 非 generated 模式返回空字符串，不影响原有请求结构。
func (ps *PromptSource) GetSystemContent() string {
	return ps.SystemContent
}

// GetRandomContent 随机获取一个prompt内容
func (ps *PromptSource) GetRandomContent() string {
	// 如果不是文件源，直接返回内容
	if !ps.IsFile {
		if len(ps.Contents) == 0 {
			return ""
		}
		if len(ps.Contents) == 1 {
			return ps.Contents[0]
		}
		
		// 使用当前时间和进程ID作为种子的随机数生成器
		r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))
		index := r.Intn(len(ps.Contents))
		return ps.Contents[index]
	}

	// 文件源：随机选择一个文件路径并读取内容
	if len(ps.FilePaths) == 0 {
		return ""
	}
	
	var filePath string
	if len(ps.FilePaths) == 1 {
		filePath = ps.FilePaths[0]
	} else {
		// 使用当前时间和进程ID作为种子的随机数生成器
		r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))
		index := r.Intn(len(ps.FilePaths))
		filePath = ps.FilePaths[index]
	}
	
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 读取文件失败 %s: %v\n", filePath, err)
		return ""
	}
	
	return string(content)
}

// GetContentByIndex 根据索引获取prompt内容
func (ps *PromptSource) GetContentByIndex(index int) string {
	// 如果不是文件源，直接返回内容
	if !ps.IsFile {
		if len(ps.Contents) == 0 {
			return ps.GetRandomContent()
		}
		if index < 0 {
			return ps.GetRandomContent()
		}
		// 用取模循环，确保多个请求在有限 Contents 上均匀分布
		return ps.Contents[index%len(ps.Contents)]
	}

	// 文件源：根据索引读取对应文件
	if index < 0 || index >= len(ps.FilePaths) {
		return ps.GetRandomContent()
	}
	
	filePath := ps.FilePaths[index]
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 读取文件失败 %s: %v\n", filePath, err)
		return ps.GetRandomContent()
	}
	
	return string(content)
}

// Count 返回prompt内容的数量
func (ps *PromptSource) Count() int {
	if ps.IsFile {
		return len(ps.FilePaths)
	}
	return len(ps.Contents)
}

// LoadPromptsFromPattern 递归加载目录下匹配模式的文件
func LoadPromptsFromPattern(pattern string) (*PromptSource, error) {
	var filePaths []string

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if d.IsDir() {
			return nil
		}

		// 检查是否匹配模式
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if matched {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历目录失败: %v", err)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("没有找到匹配的文件: %s", pattern)
	}

	return &PromptSource{
		IsFile:         true,
		FilePaths:      filePaths,
		Contents:       nil, // 不预加载内容
		DisplayText:    fmt.Sprintf("文件: %s (%d个)", pattern, len(filePaths)),
		ShouldTruncate: false, // 文件显示不需要截断
	}, nil
}

// GeneratePromptByLength 根据指定长度生成prompt内容
// 生成的内容是有意义的文本片段，而不是随机字符
func GeneratePromptByLength(length int) string {
	if length <= 0 {
		return ""
	}

	// 使用一段可重复的测试文本作为基础内容
	baseText := "这是一段用于性能测试的文本内容。人工智能技术的发展正在改变我们的生活方式，从自然语言处理到计算机视觉，从机器学习到深度学习，各种技术不断涌现。大语言模型的出现更是让AI应用达到了新的高度，能够理解和生成人类语言，完成各种复杂的任务。测试不同长度的输入对于评估模型性能至关重要，可以帮助我们了解模型在处理不同规模数据时的表现。"

	// 计算需要重复的次数
	baseLen := utf8.RuneCountInString(baseText)
	if length <= baseLen {
		// 如果需要的长度小于基础文本，直接截取
		runes := []rune(baseText)
		return string(runes[:length])
	}

	// 需要重复多次基础文本
	var builder strings.Builder
	builder.Grow(length * 3) // 预分配足够的空间（考虑UTF-8编码）

	currentLen := 0
	for currentLen < length {
		if currentLen > 0 {
			builder.WriteString(" ") // 添加分隔符
			currentLen++
		}

		remaining := length - currentLen
		if remaining >= baseLen {
			builder.WriteString(baseText)
			currentLen += baseLen
		} else {
			// 最后一部分，只取需要的长度
			runes := []rune(baseText)
			builder.WriteString(string(runes[:remaining]))
			currentLen += remaining
		}
	}

	return builder.String()
}

// LoadPromptByLength 创建指定长度的 PromptSource。
//
// 为了让测试中部分请求满足前缀缓存条件（Prefix Cache），内容被拆分为两部分：
//   - SystemContent（约 90% 长度）：固定不变的大段上下文，作为 system 消息发送；
//     同一批次所有请求共享相同的 system 消息，API 侧命中前缀缓存后可大幅降低延迟。
//   - Contents（user 消息候选列表）：多条短问题，每个请求按 index 取模轮流使用，
//     既保证请求内容有差异，又确保 system 前缀不变以触发缓存。
func LoadPromptByLength(length int) (*PromptSource, error) {
	if length <= 0 {
		return nil, fmt.Errorf("prompt 长度必须大于 0")
	}

	// 90% 作为 system 消息（固定，供缓存命中）
	systemLen := length * 9 / 10
	if systemLen < 1 {
		systemLen = 1
	}
	systemContent := GeneratePromptByLength(systemLen)
	actualSystemLen := utf8.RuneCountInString(systemContent)

	// 短而多样的 user 消息，各请求轮流使用（保证差异 + 共享 system 前缀）
	userQuestions := []string{
		"请帮我总结一下上述内容的核心要点。",
		"根据以上信息，有什么值得特别关注的地方？",
		"上述内容中最重要的信息是什么？",
		"请对以上内容进行简短分析。",
		"上述内容的主要主题是什么，请概括。",
		"从以上内容中能得出哪些结论？",
		"以上内容有哪些值得深入探讨的点？",
		"请提炼上述内容的关键信息。",
		"对以上内容你有什么看法？",
		"上述内容对实际应用有什么启示？",
	}

	return &PromptSource{
		IsFile:         false,
		FilePaths:      nil,
		Contents:       userQuestions,
		SystemContent:  systemContent,
		DisplayText:    fmt.Sprintf("生成内容 (系统消息: %d 字符，轮换用户问题 x%d)", actualSystemLen, len(userQuestions)),
		ShouldTruncate: false,
	}, nil
}
