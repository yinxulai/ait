package pages

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/yinxulai/ait/internal/server"
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
	return lipgloss.NewStyle().Width(width).Render(s)
}

// wrapText 将文本按 maxW 列宽折行，返回行切片（CJK 字符按 2 列宽计算）。
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

// renderProgressBar 渲染进度条行：prefix 固定在左，suffix 固定在右，中间弹性进度条。
func renderProgressBar(st Styles, prefix, suffix string, ratio float64, totalW int) string {
	barW := totalW - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	if barW < 5 {
		barW = 5
		maxSuffixW := maxInt(0, totalW-lipgloss.Width(prefix)-barW)
		suffix = truncate(suffix, maxSuffixW)
	}
	filled := int(ratio * float64(barW))
	barRendered := st.Ok.Render(strings.Repeat("█", filled)) +
		st.Muted.Render(strings.Repeat("░", barW-filled))
	return lipgloss.JoinHorizontal(lipgloss.Top, prefix, barRendered, suffix)
}

// dividerLine 生成全宽水平分隔线。
func dividerLine(st Styles, width int) string {
	if width <= 0 {
		return ""
	}
	return st.Divider.Render(strings.Repeat("─", width))
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

// fmtRelativeTime 将过去的时间格式化为「X 前」的简短形式。
func fmtRelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	if d < time.Minute {
		return "刚刚"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 30 {
		return fmt.Sprintf("%d 天前", days)
	}
	return t.Format("2006-01-02")
}

// ─── 布局工具 ─────────────────────────────────────────────────────────────────

// AppVersion 是展示在 AppHeader 中的版本字符串，由 SetAppVersion 在启动时设置。
var AppVersion = "dev"

// SetAppVersion 更新 AppHeader 中展示的版本字符串。
func SetAppVersion(v string) { AppVersion = v }

// renderHeader 渲染统一 AppHeader（三行）。
// 第一行：AIT ASCII 字符画 + 页面标题 | meta 徽章
// 第二行：AIT 字符画第二行 + 子标题 | 版本徽章
// 第三行：AIT 字符画第三行 + 左信息 chips | 右信息 chips
func renderHeader(st Styles, width int, title, subtitle, meta string, infoLeft, infoRight []string) string {
	w := width
	if w < 1 {
		w = 80
	}

	title = truncate(strings.TrimSpace(title), maxInt(12, w/2))
	subtitle = truncate(strings.TrimSpace(subtitle), maxInt(16, w/2))
	meta = truncate(strings.TrimSpace(meta), maxInt(10, w/4))
	if title == "" {
		title = "AIT"
	}

	// AIT ASCII 字符画（三行，粗体像素字体，实心彩色）
	//   A (10)     I (5)    T (10)
	//    ████       █████   ██████████
	//   ██  ██        █        ██
	//  ████████    █████        ██
	artA := [3]string{
		"   ████   ", // 可视宽 10
		"  ██  ██  ", // 可视宽 10
		" ████████ ", // 可视宽 10
	}
	artI := [3]string{
		"█████", // 可视宽 5
		"  █  ", // 可视宽 5
		"█████", // 可视宽 5
	}
	artT := [3]string{
		"██████████", // 可视宽 10
		"    ██    ", // 可视宽 10
		"    ██    ", // 可视宽 10
	}

	styleA := lipgloss.NewStyle().Foreground(colorPink).Bold(true)
	styleI := lipgloss.NewStyle().Foreground(colorGold).Bold(true)
	styleT := lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	styleSep := lipgloss.NewStyle().Foreground(colorPink)

	artRow := func(i int) string {
		return styleA.Render(artA[i]) + "  " + styleI.Render(artI[i]) + "  " + styleT.Render(artT[i])
	}
	vsep := styleSep.Render("┃")

	wideEnough := w >= 65 // 宽屏才展示 ASCII art

	// ── Line 1: [art row 0] │ title                          [meta badge] ────────
	right1 := ""
	if meta != "" {
		right1 = lipgloss.NewStyle().
			Background(colorCyan).Foreground(colorHeaderBg).Bold(true).Padding(0, 1).
			Render(meta) + " "
	}
	var left1 string
	if wideEnough {
		titleSeg := lipgloss.NewStyle().Foreground(colorWhite).Bold(true).Render(title)
		left1 = " " + artRow(0) + " " + vsep + " " + titleSeg
	} else {
		brandPill := lipgloss.NewStyle().Background(colorPink).Foreground(colorHeaderBg).Bold(true).Padding(0, 1).Render("AIT")
		titlePill := lipgloss.NewStyle().Foreground(colorWhite).Bold(true).Padding(0, 1).Render(title)
		left1 = " " + lipgloss.JoinHorizontal(lipgloss.Center, brandPill, titlePill)
	}
	line1 := renderChromeLine(st.Header, w, left1, right1)

	// ── Line 2: [art row 1] │ subtitle                    [version badge] ──────
	verBadge := lipgloss.NewStyle().
		Background(colorPurple).Foreground(colorWhite).Padding(0, 1).
		Render("v" + AppVersion) + " "
	var left2 string
	if wideEnough {
		subSeg := ""
		if subtitle != "" {
			subSeg = lipgloss.NewStyle().
				Background(colorHotkeysPrimaryBg).Foreground(colorWhite).Padding(0, 1).
				Render(subtitle)
		}
		left2 = " " + artRow(1) + " " + vsep + " " + subSeg
	} else {
		if subtitle != "" {
			left2 = " " + lipgloss.NewStyle().
				Background(colorHotkeysPrimaryBg).Foreground(colorWhite).Padding(0, 1).
				Render(subtitle)
		}
	}
	line2 := renderChromeLine(st.Header, w, left2, verBadge)

	// ── Line 3: [art row 2] │ infoLeft chips              infoRight chips ───
	var left3 string
	if wideEnough {
		artPart := " " + artRow(2) + " " + vsep
		artPartW := lipgloss.Width(artPart) + 1 // +1 为 art 与 pills 之间的分隔空格
		availW := maxInt(8, w-artPartW-2-maxInt(10, w/3))
		if leftPills := renderInfoPills(infoLeft, availW); leftPills != "" {
			left3 = artPart + " " + leftPills
		} else {
			left3 = artPart
		}
	} else {
		if leftPills := renderInfoPills(infoLeft, maxInt(8, w/3)); leftPills != "" {
			left3 = " " + leftPills
		}
	}
	right3 := ""
	if pills := renderInfoPills(infoRight, maxInt(10, w/3)); pills != "" {
		right3 = pills + " "
	}
	line3 := renderChromeLine(st.HeaderInfo, w, left3, right3)

	return line1 + "\n" + line2 + "\n" + line3
}

// renderHotkeys 渲染统一页面 Hotkeys。
// 第一行展示当前页快捷操作，第二行展示返回/退出等全局上下文与应用标识。
func renderHotkeys(st Styles, width int, hk PageHotkeys) string {
	w := width
	if w < 1 {
		w = 80
	}

	hkLine := renderPrimaryHotkeyItems(hk.Hotkeys, maxInt(8, w-4))
	line1 := renderChromeLine(st.HotkeysPrimary, w, " "+hkLine, "")

	appStamp := lipgloss.NewStyle().Foreground(colorPink).Bold(true).Render("AIT") +
		lipgloss.NewStyle().Foreground(colorMuted).Render("  终端 · "+time.Now().Format("15:04"))
	left2 := renderSecondaryHotkeyItems(hk.Hints, maxInt(8, w-lipgloss.Width(appStamp)-4))
	line2 := renderChromeLine(st.HotkeysSecondary, w, " "+left2, appStamp+" ")

	return line1 + "\n" + line2
}

func renderChromeLine(base lipgloss.Style, width int, left, right string) string {
	rightW := lipgloss.Width(right)
	spacerW := maxInt(0, width-lipgloss.Width(left)-rightW)
	spacer := lipgloss.NewStyle().Width(spacerW).Render("")
	return base.Width(width).Render(left + spacer + right)
}

func renderInfoPills(parts []string, maxW int) string {
	parts = nonEmptyParts(parts)
	if len(parts) == 0 {
		return ""
	}

	var rendered []string
	for _, part := range parts {
		rendered = append(rendered, lipgloss.NewStyle().
			Background(lipgloss.Color("239")).
			Foreground(colorHeaderFg).
			Padding(0, 1).
			Render(truncate(part, 28)))
	}
	return fitRenderedParts(rendered, " ", maxW)
}

func renderPrimaryHotkeyItems(items []HotkeyItem, maxW int) string {
	if len(items) == 0 {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("239")).
			Foreground(colorMuted).
			Padding(0, 1).
			Render("当前页暂无快捷操作")
	}

	var rendered []string
	for _, item := range items {
		if item.Key == "" && item.Desc == "" {
			if item.Text == "" {
				continue
			}
			rendered = append(rendered, lipgloss.NewStyle().
				Background(lipgloss.Color("239")).
				Foreground(colorWhite).
				Padding(0, 1).
				Render(item.Text))
			continue
		}
		keySeg := lipgloss.NewStyle().
			Background(colorGold).
			Foreground(colorHeaderBg).
			Bold(true).
			Padding(0, 1).
			Render(item.Key)
		descSeg := lipgloss.NewStyle().
			Background(lipgloss.Color("239")).
			Foreground(colorWhite).
			Padding(0, 1).
			Render(item.Desc)
		rendered = append(rendered, lipgloss.JoinHorizontal(lipgloss.Center, keySeg, descSeg))
	}
	return fitRenderedParts(rendered, "  ", maxW)
}

func renderSecondaryHotkeyItems(items []HotkeyItem, maxW int) string {
	var parts []string
	for _, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" && (item.Key != "" || item.Desc != "") {
			switch {
			case item.Key != "" && item.Desc != "":
				text = "[" + item.Key + "] " + item.Desc
			case item.Key != "":
				text = item.Key
			default:
				text = item.Desc
			}
		}
		if text != "" {
			parts = append(parts, text)
		}
	}
	return fitPlainParts(parts, "   •   ", maxW)
}

func fitRenderedParts(parts []string, sep string, maxW int) string {
	visible := nonEmptyParts(parts)
	if len(visible) == 0 {
		return ""
	}
	if maxW <= 0 {
		return strings.Join(visible, sep)
	}

	var chosen []string
	used := 0
	sepW := lipgloss.Width(sep)
	for _, part := range visible {
		partW := lipgloss.Width(part)
		extra := partW
		if len(chosen) > 0 {
			extra += sepW
		}
		if len(chosen) > 0 && used+extra > maxW {
			break
		}
		chosen = append(chosen, part)
		used += extra
	}
	if len(chosen) == 0 {
		return visible[0]
	}
	return strings.Join(chosen, sep)
}

func fitPlainParts(parts []string, sep string, maxW int) string {
	visible := nonEmptyParts(parts)
	if len(visible) == 0 {
		return ""
	}
	if maxW <= 0 {
		return strings.Join(visible, sep)
	}

	var chosen []string
	used := 0
	sepW := lipgloss.Width(sep)
	for _, part := range visible {
		part = truncate(part, maxW)
		partW := lipgloss.Width(part)
		extra := partW
		if len(chosen) > 0 {
			extra += sepW
		}
		if len(chosen) > 0 && used+extra > maxW {
			break
		}
		chosen = append(chosen, part)
		used += extra
	}
	if len(chosen) == 0 {
		return truncate(visible[0], maxW)
	}
	return strings.Join(chosen, sep)
}

func nonEmptyParts(parts []string) []string {
	var visible []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			visible = append(visible, part)
		}
	}
	return visible
}

func runStatusText(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return "运行中"
	case "completed":
		return "已完成"
	case "failed":
		return "运行失败"
	case "stopped":
		return "已停止"
	case "":
		return "等待数据"
	default:
		return status
	}
}

// modeShortLabel 将运行模式字符串转换为短标签。
func modeShortLabel(mode string) string {
	if mode == "turbo" {
		return "Turbo"
	}
	return "标准"
}

// isRunStateRunning 判断 RunState 是否处于运行状态。
func isRunStateRunning(rs *server.RunState) bool {
	return rs != nil && rs.Status == server.RunStatusRunning
}

// applyColWidth 按列宽定义应用固定宽度或仅 padding，用于 lgtable StyleFunc 中的 aw 闭包。
// colWidths[col] > 0 时设固定总宽（含 padding），否则仅添加 padding。
func applyColWidth(s lipgloss.Style, col int, colWidths []int) lipgloss.Style {
	if col < len(colWidths) && colWidths[col] > 0 {
		return s.Width(colWidths[col]).Padding(0, 1)
	}
	return s.Padding(0, 1)
}

// appendRunMetricLines 向 lines 追加 6 行运行指标（成功率/TPS/TTFT/缓存命中/RPM/TPM）。
func appendRunMetricLines(lines []string, st Styles, rs *server.RunState) []string {
	lines = append(lines, " "+labelValue(st, "成功率  ", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate))))
	lines = append(lines, " "+labelValue(st, "TPS均值 ", st.MetricVal.Render(fmt.Sprintf("%.1f tok/s", rs.AvgTPS))))
	lines = append(lines, " "+labelValue(st, "TTFT均值", st.MetricVal.Render(fmtDuration(rs.AvgTTFT))))
	lines = append(lines, " "+labelValue(st, "缓存命中", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.CacheHitRate*100))))
	lines = append(lines, " "+labelValue(st, "RPM     ", st.MetricVal.Render(fmt.Sprintf("%.0f req/min", rs.RPM))))
	lines = append(lines, " "+labelValue(st, "TPM     ", st.MetricVal.Render(fmt.Sprintf("%.0f tok/min", rs.TPM))))
	return lines
}

func panelTitleLines(st Styles, title string, width int, compact bool) []string {
	var rendered string
	if width > 0 {
		// 截断标题防止超宽后被 lipgloss 折行
		rendered = st.PanelHead.Width(width).Padding(0, 0, 0, 1).Render(truncate(title, maxInt(1, width-1)))
	} else {
		rendered = st.PanelHead.Render(" " + title)
	}
	lines := []string{rendered}
	if !compact {
		lines = append(lines, "")
	}
	return lines
}

func finishPanelLines(lines []string, maxH int) string {
	if maxH < 1 {
		maxH = 1
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

func renderSplitPanels(st Styles, leftFrame, rightFrame PanelFrame, leftContent, rightContent string) string {
	leftPanel := leftFrame.Wrap(st, leftContent)
	rightPanel := rightFrame.Wrap(st, rightContent)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
}

func normalizeInlineText(s string) string {
	replacer := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ")
	return strings.Join(strings.Fields(replacer.Replace(s)), " ")
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
		return "openai-completions"
	case "openai-responses":
		return "openai-responses"
	case "anthropic-messages":
		return "anthropic-messages"
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
	case "raw":
		if promptText != "" {
			r := []rune(promptText)
			if len(r) > 20 {
				return "RAW: " + string(r[:20]) + "…"
			}
			return "RAW: " + promptText
		}
		return "(未设置)"
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
// lipgloss v2: Width(n) = 外部总宽度（含 border），Panel 有 1 字符宽边框。
func wrapPanel(st Styles, content string, outerW int) string {
	if outerW < 4 {
		return content
	}
	return st.Panel.Width(outerW).Render(content)
}
