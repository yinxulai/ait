package prompt

import (
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PromptSource 表示prompt的来源信息
type PromptSource struct {
	IsFile      bool     // 是否来自文件
	FilePaths   []string // 文件路径列表
	Contents    []string // prompt内容列表（仅用于非文件内容）
	DisplayText string   // 用于显示的文本
}

// LoadPrompts 解析prompt参数，只处理字符串内容
func LoadPrompts(promptArg string) (*PromptSource, error) {
	return &PromptSource{
		IsFile:      false,
		FilePaths:   nil,
		Contents:    []string{promptArg},
		DisplayText: promptArg,
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
		IsFile:      true,
		FilePaths:   []string{filePath},
		Contents:    nil, // 不预加载内容
		DisplayText: fmt.Sprintf("文件: %s (1个)", filePath),
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
		IsFile:      true,
		FilePaths:   filePaths,
		Contents:    nil, // 不预加载内容
		DisplayText: fmt.Sprintf("文件: %s (%d个)", pattern, len(filePaths)),
	}, nil
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
		if index < 0 || index >= len(ps.Contents) {
			return ps.GetRandomContent()
		}
		return ps.Contents[index]
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
		IsFile:      true,
		FilePaths:   filePaths,
		Contents:    nil, // 不预加载内容
		DisplayText: fmt.Sprintf("文件: %s (%d个)", pattern, len(filePaths)),
	}, nil
}
