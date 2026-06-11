package prompt

import (
	"fmt"
	"io/fs"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

var generatedCommonSeeds = []string{
	"公共消息1：以下内容描述一个固定的评测背景，所有请求都共享这段上下文，以便模拟前缀缓存命中。",
	"公共消息2：请基于同一组系统约束、相同的领域设定和一致的输出风格进行分析，不要改变整体语境。",
	"公共消息3：你正在参与稳定负载测试，公共上下文应尽量保持一致，只有用户问题会发生变化。",
}

var generatedUserSeeds = []string{
	"随机用户消息1：请提炼以上背景里的三个关键结论，并说明它们为什么重要。",
	"随机用户消息2：基于上述共享信息，总结最值得关注的风险点与应对方向。",
	"随机用户消息3：请用简洁结构归纳核心要点，并指出其中最有价值的一条。",
	"随机用户消息4：结合以上上下文，说明该场景在实际落地时应优先验证哪些指标。",
	"随机用户消息5：请从性能、稳定性和可观测性三个角度给出简短分析。",
	"随机用户消息6：在不改变公共背景的前提下，概括最可能影响结果判断的因素。",
}

// PromptSource 表示prompt的来源信息
type PromptSource struct {
	IsFile         bool     // 是否来自文件
	FilePaths      []string // 文件路径列表
	Contents       []string // prompt内容列表（仅用于非文件内容）
	SystemContent  string   // 可选的系统消息内容；为空时表示不额外发送 system 消息
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

// GetSystemContent 返回系统消息内容；为空时不发送额外的 system 消息。
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
		slog.Warn("failed to read prompt file", "path", filePath, "error", err)
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
		slog.Warn("failed to read prompt file, falling back to random", "path", filePath, "error", err)
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

func splitGeneratedPromptLengths(total int) (commonLen, userLen int) {
	if total <= 0 {
		return 0, 0
	}

	if total <= 24 {
		return 0, total
	}

	commonLen = total * 9 / 10
	userLen = total - commonLen

	if userLen < 24 {
		userLen = 24
		if total < userLen {
			userLen = total
		}
		commonLen = total - userLen
	}

	if commonLen < 0 {
		commonLen = 0
	}
	if userLen < 0 {
		userLen = 0
	}

	return commonLen, userLen
}

func splitBudget(total, parts int) []int {
	if parts <= 0 {
		return nil
	}
	budgets := make([]int, parts)
	base := total / parts
	rest := total % parts
	for i := 0; i < parts; i++ {
		budgets[i] = base
		if i < rest {
			budgets[i]++
		}
	}
	return budgets
}

func truncateToRunes(text string, length int) string {
	if length <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= length {
		return text
	}
	return string(runes[:length])
}

func composeSizedText(seed string, target int) string {
	if target <= 0 {
		return ""
	}

	seed = strings.TrimSpace(seed)
	if utf8.RuneCountInString(seed) >= target {
		return truncateToRunes(seed, target)
	}

	builder := strings.Builder{}
	builder.WriteString(seed)
	currentLen := utf8.RuneCountInString(seed)

	if currentLen < target {
		remaining := target - currentLen
		if remaining > 0 {
			builder.WriteString(GeneratePromptByLength(remaining))
		}
	}

	return truncateToRunes(builder.String(), target)
}

func buildGeneratedCommonPrompt(target int) string {
	if target <= 0 {
		return ""
	}

	messageCount := len(generatedCommonSeeds)
	separatorLen := 2 * (messageCount - 1)
	if target <= separatorLen+messageCount {
		return composeSizedText(generatedCommonSeeds[0], target)
	}

	bodyBudget := target - separatorLen
	budgets := splitBudget(bodyBudget, messageCount)
	parts := make([]string, 0, messageCount)
	for i, budget := range budgets {
		parts = append(parts, composeSizedText(generatedCommonSeeds[i], budget))
	}

	return truncateToRunes(strings.Join(parts, "\n\n"), target)
}

func buildGeneratedUserPrompts(target int) []string {
	if target <= 0 {
		return []string{""}
	}

	contents := make([]string, 0, len(generatedUserSeeds))
	for _, seed := range generatedUserSeeds {
		contents = append(contents, composeSizedText(seed, target))
	}
	return contents
}

// LoadPromptByLength 创建指定长度的 PromptSource。
//
// generated 模式会构造一段共享公共前缀和多条用户问题变体：
//   - SystemContent: 共享的公共消息，所有请求保持一致，用于模拟缓存命中前缀。
//   - Contents: 多条不同的用户消息，请求按索引轮换，模拟公共前缀下的随机用户提问。
//
// 单次请求的总 prompt 长度仍与传入的 length 保持一致。
func LoadPromptByLength(length int) (*PromptSource, error) {
	if length <= 0 {
		return nil, fmt.Errorf("prompt 长度必须大于 0")
	}
	commonLen, userLen := splitGeneratedPromptLengths(length)
	systemContent := buildGeneratedCommonPrompt(commonLen)
	contents := buildGeneratedUserPrompts(userLen)

	return &PromptSource{
		IsFile:         false,
		FilePaths:      nil,
		Contents:       contents,
		SystemContent:  systemContent,
		DisplayText:    fmt.Sprintf("生成内容 (公共消息 %d 字符, 用户变体 x%d, 单次总长 %d 字符)", utf8.RuneCountInString(systemContent), len(contents), length),
		ShouldTruncate: false,
	}, nil
}
