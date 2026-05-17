package pages

import (
	"fmt"
	"strings"
	"time"

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
		rw := lipgloss.Width(string(r))
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
// 第一行：◆ brand（粉色）│ 页面标题（青色），深色背景
// 第二行：infoLeft（左）/ infoRight（右），较暗色背景
func renderHeader(st Styles, width int, titleLeft, titleRight, infoLeft, infoRight string) string {
	w := width
	if w < 1 {
		w = 80
	}

	// Line 1: avoid nested Render() fragments to prevent ANSI reset from breaking background.
	brand := titleLeft
	pageTitle := ""
	if idx := strings.Index(titleLeft, "  "); idx >= 0 {
		brand = titleLeft[:idx]
		pageTitle = strings.TrimSpace(titleLeft[idx:])
	}

	brandSeg := lipgloss.NewStyle().
		Background(colorHeaderBg).
		Foreground(colorPink).
		Bold(true).
		Render(" ◆ " + brand)
	sepSeg := lipgloss.NewStyle().
		Background(colorHeaderBg).
		Foreground(colorDivider).
		Render("  │  ")
	titleSeg := lipgloss.NewStyle().
		Background(colorHeaderBg).
		Foreground(colorCyan).
		Bold(true).
		Render(pageTitle)

	left1 := brandSeg
	if pageTitle != "" {
		left1 += sepSeg + titleSeg
	}
	right1 := ""
	if titleRight != "" {
		right1 = lipgloss.NewStyle().
			Background(colorHeaderBg).
			Foreground(colorMuted).
			Render(titleRight + " ")
	}
	left1W := lipgloss.Width(left1)
	right1W := lipgloss.Width(right1)
	pad1 := w - left1W - right1W
	if pad1 < 0 {
		pad1 = 0
	}
	padSeg := lipgloss.NewStyle().
		Background(colorHeaderBg).
		Render(strings.Repeat(" ", pad1))
	line1 := left1 + padSeg + right1

	// ─ Line 2: info bar ─
	il := "  " + infoLeft
	ir := infoRight + " "
	ilW := lipgloss.Width(il)
	irW := lipgloss.Width(ir)
	pad2 := w - ilW - irW
	if pad2 < 0 {
		pad2 = 0
	}
	line2 := st.HeaderInfo.Width(w).Render(il + strings.Repeat(" ", pad2) + ir)

	return line1 + "\n" + line2
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

// renderTableHeader 统一渲染列表表头。
func renderTableHeader(st Styles, width int, row string) string {
	return st.TableHead.Width(width).Render(row)
}

// renderTableRow 统一渲染列表行（选中/未选中）。
func renderTableRow(st Styles, width int, isSel bool, row string) string {
	if isSel {
		return st.TableRowSel.Width(width).Render(row)
	}
	return st.TableRow.Width(width).Render(row)
}

// minInt 返回两个整数中的较小值。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt 返回两个整数中的较大值。
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// clampInt 将 v 约束在 [low, high] 区间内。
func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

// listVisibleItems 计算在给定高度下可自然滚动的列表项数量。
// staticLines 是列表项区域前的固定行数（如 section/header/divider）。
func listVisibleItems(maxLines, staticLines int) int {
	visible := (maxLines - staticLines + 1) / 2
	if visible < 1 {
		return 1
	}
	return visible
}

// ensureVisibleOffset 让 selected 始终位于 offset/visible 定义的可视窗口内。
func ensureVisibleOffset(selected, count, offset, visible int) int {
	if count <= 0 {
		return 0
	}
	if visible < 1 {
		visible = 1
	}
	selected = clampInt(selected, 0, count-1)
	maxOffset := maxInt(0, count-visible)
	offset = clampInt(offset, 0, maxOffset)
	if selected < offset {
		offset = selected
	}
	if selected >= offset+visible {
		offset = selected - visible + 1
	}
	return clampInt(offset, 0, maxOffset)
}

// selectionMarker 返回统一的选中标记列内容。
func selectionMarker(isSel bool) string {
	if isSel {
		return "▶"
	}
	return ""
}

// styleWhenNotSelected 仅在未选中时应用局部样式，避免重置选中行背景。
func styleWhenNotSelected(isSel bool, style lipgloss.Style, text string) string {
	if isSel {
		return text
	}
	return style.Render(text)
}

// renderWelcomeHero 渲染任务中心顶部的品牌欢迎区。
func renderWelcomeHero(st Styles, width int) []string {
	if width < 42 {
		return nil
	}

	art := []string{
		"    _    ___ _____",
		"   / \\  |_ _|_   _|",
		"  / _ \\  | |  | |  ",
		" / ___ \\ | |  | |  ",
		"/_/   \\_\\___| |_|  ",
	}
	artStyles := []lipgloss.Style{
		lipgloss.NewStyle().Foreground(colorPink).Bold(true),
		lipgloss.NewStyle().Foreground(colorCyan).Bold(true),
		lipgloss.NewStyle().Foreground(colorGold).Bold(true),
		lipgloss.NewStyle().Foreground(colorTeal).Bold(true),
		lipgloss.NewStyle().Foreground(colorPurple).Bold(true),
	}

	type heroTextLine struct {
		style lipgloss.Style
		text  string
	}
	intro := []heroTextLine{
		{style: st.SectionHead, text: "AI 模型性能测试工作台"},
		{style: st.Value, text: "批量压测 OpenAI / Anthropic 协议模型，聚焦 TTFT、TPS、缓存与网络指标。"},
		{style: st.Muted, text: "从任务中心出发：创建任务、直接运行、查看执行记录、导出报告。"},
		{style: st.Muted, text: "[a] 新建任务   [Enter] 查看详情/进入仪表盘   [r] 立即运行"},
	}

	artW := 0
	for _, line := range art {
		artW = maxInt(artW, lipgloss.Width(line))
	}

	if width >= 76 {
		gap := 3
		rightW := maxInt(18, width-artW-gap)
		wrapped := make([]string, 0, 8)
		for i, line := range intro {
			segments := wrapText(line.text, rightW)
			if len(segments) == 0 {
				segments = []string{""}
			}
			for _, segment := range segments {
				wrapped = append(wrapped, line.style.Render(segment))
			}
			if i == 0 {
				wrapped = append(wrapped, "")
			}
		}

		total := maxInt(len(art), len(wrapped))
		lines := make([]string, 0, total)
		for i := 0; i < total; i++ {
			left := strings.Repeat(" ", artW)
			if i < len(art) {
				left = artStyles[i].Render(art[i])
			}
			right := ""
			if i < len(wrapped) {
				right = wrapped[i]
			}
			lines = append(lines, padRight(left, artW)+strings.Repeat(" ", gap)+right)
		}
		return lines
	}

	lines := make([]string, 0, len(art)+len(intro)+1)
	for i, line := range art {
		lines = append(lines, artStyles[i].Render(line))
	}
	lines = append(lines, "")
	for _, line := range intro {
		for _, segment := range wrapText(line.text, width) {
			lines = append(lines, line.style.Render(segment))
		}
	}
	return lines
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

// wrapPanel 用带边框的 Panel 包裹内容，outerW 为包含边框的总宽度。
func wrapPanel(st Styles, content string, outerW int) string {
	if outerW < 4 {
		return content
	}
	return st.Panel.Width(outerW - 2).Render(content)
}
