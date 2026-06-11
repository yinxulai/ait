package shared

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// Truncate 截断字符串（按可见列宽），超出部分显示 "…"。
func Truncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w <= maxW {
		return s
	}
	// 按 rune 截断
	runes := []rune(s)
	total := 0
	for i, r := range runes {
		rw := lipgloss.Width(string(r))
		if total+rw > maxW-1 {
			return string(runes[:i]) + "…"
		}
		total += rw
	}
	return s
}

// PadRight 右侧补空格至 width（按可见列宽）。
func PadRight(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

// PadToDisplayWidth 用空格将 s 右侧填充至 w 显示列宽。
// 使用 runewidth 精确测量 CJK（2列）与 ASCII（1列）字符。
func PadToDisplayWidth(s string, w int) string {
	cur := runewidth.StringWidth(s)
	if cur >= w {
		return s
	}
	return s + strings.Repeat(" ", w-cur)
}

// WrapText 将文本按 maxW 列宽折行，返回行切片（CJK 字符按 2 列宽计算）。
func WrapText(s string, maxW int) []string {
	if maxW <= 0 {
		return []string{s}
	}
	var result []string
	for _, line := range strings.Split(s, "\n") {
		runes := []rune(line)
		if len(runes) == 0 {
			result = append(result, "")
			continue
		}
		for len(runes) > 0 {
			colW := 0
			end := 0
			for end < len(runes) {
				rw := lipgloss.Width(string(runes[end]))
				if colW+rw > maxW {
					break
				}
				colW += rw
				end++
			}
			if end == 0 {
				// 单个字符超宽（如极窄终端）——强制取一个避免死循环
				end = 1
			}
			result = append(result, string(runes[:end]))
			runes = runes[end:]
		}
	}
	return result
}

// NormalizeInlineText 移除文本中的换行符，将其转换为空格（用于单行显示）。
func NormalizeInlineText(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", " "), "\n", " ")
}

// MaxLabelWidth 返回一组标签中最大的显示列宽。
func MaxLabelWidth(labels []string) int {
	max := 0
	for _, l := range labels {
		if w := runewidth.StringWidth(l); w > max {
			max = w
		}
	}
	return max
}

// NonEmptyParts 过滤掉空字符串，返回非空部分。
func NonEmptyParts(parts []string) []string {
	var result []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			result = append(result, p)
		}
	}
	return result
}
