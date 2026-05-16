package pages

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// ─── 文本工具 ─────────────────────────────────────────────────────────────────

// truncate 截断字符串（按可见列宽），超出部分显示 "…"。
func truncate(s string, maxW int) string {
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
		rw := utf8.RuneLen(r)
		if rw < 1 {
			rw = 1
		}
		if total+rw > maxW-1 {
			return string(runes[:i]) + "…"
		}
		total += rw
	}
	return s
}

// padRight 右侧补空格至 width（按可见列宽）。
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// wrapText 将文本按 maxW 宽度折行，返回行切片。
func wrapText(s string, maxW int) []string {
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
			end := maxW
			if end > len(runes) {
				end = len(runes)
			}
			result = append(result, string(runes[:end]))
			runes = runes[end:]
		}
	}
	return result
}

// dividerLine 生成全宽水平分隔线。
func dividerLine(st Styles, width int) string {
	if width <= 0 {
		return ""
	}
	return st.Divider.Render(strings.Repeat("─", width))
}

// ─── 时间格式化 ───────────────────────────────────────────────────────────────

// timeAgo 将时间转换为"N分钟前"/"刚刚"等人性化描述。
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	default:
		return t.Format("2006-01-02 15:04")
	}
}

// fmtDuration 格式化 Duration 为简短字符串（ms/s/min）。
func fmtDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms == 0 {
		return "0ms"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := float64(ms) / 1000
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	return fmt.Sprintf("%.0fm%.0fs", s/60, float64(int64(s)%60))
}

// ─── 布局工具 ─────────────────────────────────────────────────────────────────

// renderHeader 渲染顶部双行标题栏。
// 第一行：titleLeft（左）/ titleRight（右），紫色背景加粗
// 第二行：infoLeft（左）/ infoRight（右），较暗色背景
func renderHeader(st Styles, width int, titleLeft, titleRight, infoLeft, infoRight string) string {
	w := width
	if w < 1 {
		w = 80
	}
	// 第一行
	tl := " " + titleLeft
	tr := titleRight + " "
	tlW := lipgloss.Width(tl)
	trW := lipgloss.Width(tr)
	pad1 := w - tlW - trW
	if pad1 < 0 {
		pad1 = 0
	}
	line1 := tl + strings.Repeat(" ", pad1) + tr

	// 第二行
	il := "  " + infoLeft
	ir := infoRight + " "
	ilW := lipgloss.Width(il)
	irW := lipgloss.Width(ir)
	pad2 := w - ilW - irW
	if pad2 < 0 {
		pad2 = 0
	}
	line2 := il + strings.Repeat(" ", pad2) + ir

	return st.Header.Width(w).Render(line1) + "\n" +
		st.HeaderInfo.Width(w).Render(line2)
}

// renderFooter 渲染底部状态栏（单行，深色背景）。
func renderFooter(st Styles, width int, parts ...string) string {
	w := width
	if w < 1 {
		w = 80
	}
	var visible []string
	for _, p := range parts {
		if p != "" {
			visible = append(visible, p)
		}
	}
	line := "  " + strings.Join(visible, "   ")
	return st.Footer.Width(w).Render(line)
}

// dualColumnLayout 将左右两段文本排列为双栏，高度限定为 maxH。
// 中间用竖线 │ 隔开。
func dualColumnLayout(st Styles, left, right string, leftW, rightW, maxH int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	if len(leftLines) > maxH {
		leftLines = leftLines[:maxH]
	}
	if len(rightLines) > maxH {
		rightLines = rightLines[:maxH]
	}
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}

	sep := st.Divider.Render("│")
	var rows []string
	for i := 0; i < maxH; i++ {
		lLine := leftLines[i]
		rLine := rightLines[i]
		lW := lipgloss.Width(lLine)
		if lW < leftW {
			lLine += strings.Repeat(" ", leftW-lW)
		}
		rows = append(rows, lLine+sep+rLine)
	}
	return strings.Join(rows, "\n")
}

// progressBar 生成进度条字符串（filled=已完成比例 0.0-1.0）。
func progressBar(filled float64, width int) string {
	if width <= 0 {
		return ""
	}
	if filled < 0 {
		filled = 0
	}
	if filled > 1 {
		filled = 1
	}
	doneW := int(float64(width) * filled)
	emptyW := width - doneW
	done := strings.Repeat("█", doneW)
	empty := strings.Repeat("░", emptyW)
	return done + empty
}

// wrapIndex 循环索引（保证 0 ≤ result < count）。
func wrapIndex(idx, count int) int {
	if count <= 0 {
		return 0
	}
	return ((idx % count) + count) % count
}

// ─── 数据格式化 ───────────────────────────────────────────────────────────────

// maskAPIKey 遮蔽 API Key，只展示前 4 位和后 4 位。
func maskAPIKey(key string) string {
	r := []rune(key)
	if len(r) <= 8 {
		return strings.Repeat("•", len(r))
	}
	return string(r[:4]) + "••••••••" + string(r[len(r)-4:])
}

// shortProtocol 将协议名缩短为仪表盘友好的短名。
func shortProtocol(p string) string {
	switch p {
	case "openai-completions":
		return "completions"
	case "openai-responses":
		return "responses"
	case "anthropic-messages":
		return "messages"
	default:
		return p
	}
}

// promptSummary 返回 Prompt 的简短摘要文本。
func promptSummary(promptMode, promptText, promptFile string, promptLength int) string {
	switch promptMode {
	case "file":
		return "文件: " + promptFile
	case "generated":
		return fmt.Sprintf("生成 %d 字符", promptLength)
	default:
		if promptText != "" {
			r := []rune(promptText)
			if len(r) > 20 {
				return string(r[:20]) + "…"
			}
			return promptText
		}
		return "(未设置)"
	}
}

// boolLabel 将 bool 值转换为"开启"/"关闭"。
func boolLabel(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}

// labelValue 渲染一个 label:value 对。
func labelValue(st Styles, label, value string) string {
	return st.Label.Render(label) + "  " + st.Value.Render(value)
}
