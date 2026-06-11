package pages

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/tui/pages/shared"
)

// ===== 渲染相关函数 =====

var AppVersion = "dev"

func SetAppVersion(v string) {
	AppVersion = v
}

func padToDisplayWidth(s string, w int) string {
	actual := lipgloss.Width(s)
	if actual < w {
		return s + strings.Repeat(" ", w-actual)
	}
	return s
}

func renderProgressBar(st Styles, prefix, suffix string, ratio float64, totalW int) string {
	barW := totalW - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	if barW < 5 {
		barW = 5
		maxSuffixW := shared.MaxInt(0, totalW-lipgloss.Width(prefix)-barW)
		suffix = shared.Truncate(suffix, maxSuffixW)
	}
	filled := int(ratio * float64(barW))
	barRendered := st.Ok.Render(strings.Repeat("█", filled)) +
		st.Muted.Render(strings.Repeat("░", barW-filled))
	return lipgloss.JoinHorizontal(lipgloss.Top, prefix, barRendered, suffix)
}

func dividerLine(st Styles, width int) string {
	if width <= 0 {
		return ""
	}
	return st.Divider.Render(strings.Repeat("─", width))
}

func renderHeader(st Styles, width int, title, subtitle, meta string, infoLeft, infoRight []string) string {
	w := width
	if w < 1 {
		w = 80
	}

	title = shared.Truncate(strings.TrimSpace(title), shared.MaxInt(12, w/2))
	subtitle = shared.Truncate(strings.TrimSpace(subtitle), shared.MaxInt(16, w/2))
	meta = shared.Truncate(strings.TrimSpace(meta), shared.MaxInt(10, w/4))
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
		Render("v"+AppVersion) + " "
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
		availW := shared.MaxInt(8, w-artPartW-2-shared.MaxInt(10, w/3))
		if leftPills := renderInfoPills(infoLeft, availW); leftPills != "" {
			left3 = artPart + " " + leftPills
		} else {
			left3 = artPart
		}
	} else {
		if leftPills := renderInfoPills(infoLeft, shared.MaxInt(8, w/3)); leftPills != "" {
			left3 = " " + leftPills
		}
	}
	right3 := ""
	if pills := renderInfoPills(infoRight, shared.MaxInt(10, w/3)); pills != "" {
		right3 = pills + " "
	}
	line3 := renderChromeLine(st.HeaderInfo, w, left3, right3)

	return line1 + "\n" + line2 + "\n" + line3
}

func renderHotkeys(st Styles, width int, hk PageHotkeys) string {
	w := width
	if w < 1 {
		w = 80
	}

	hkLine := renderPrimaryHotkeyItems(hk.Hotkeys, shared.MaxInt(8, w-4))
	line1 := renderChromeLine(st.HotkeysPrimary, w, " "+hkLine, "")

	appStamp := lipgloss.NewStyle().Background(colorHotkeysSecondaryBg).Foreground(colorMuted).Render(time.Now().Format("2006-01-02 15:04:05")+"  ") +
		lipgloss.NewStyle().Background(colorHotkeysSecondaryBg).Foreground(colorPink).Bold(true).Render("github.com/yinxulai/ait") +
		lipgloss.NewStyle().Background(colorHotkeysSecondaryBg).Foreground(colorMuted).Render("  Powered by Alain")
	left2 := renderSecondaryHotkeyItems(hk.Hints, shared.MaxInt(8, w-lipgloss.Width(appStamp)-4))
	line2 := renderChromeLine(st.HotkeysSecondary, w, " "+left2, appStamp+" ")

	return line1 + "\n" + line2
}

func renderChromeLine(base lipgloss.Style, width int, left, right string) string {
	rightW := lipgloss.Width(right)
	spacerW := shared.MaxInt(0, width-lipgloss.Width(left)-rightW)
	spacer := lipgloss.NewStyle().Width(spacerW).Render("")
	return base.Width(width).Render(left + spacer + right)
}

func renderInfoPills(parts []string, maxW int) string {
	parts = shared.NonEmptyParts(parts)
	if len(parts) == 0 {
		return ""
	}

	var rendered []string
	for _, part := range parts {
		rendered = append(rendered, lipgloss.NewStyle().
			Background(lipgloss.Color("239")).
			Foreground(colorHeaderFg).
			Padding(0, 1).
			Render(shared.Truncate(part, 28)))
	}
	return fitRenderedParts(rendered, " ", maxW)
}

func renderPrimaryHotkeyItems(items []HotkeyItem, maxW int) string {
	if len(items) == 0 {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("239")).
			Foreground(colorMuted).
			Padding(0, 1).
			Render(i18n.T(i18n.KNoHotkeys))
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
	if len(parts) == 0 {
		return ""
	}
	sepW := lipgloss.Width(sep)
	totalW := 0
	for i, p := range parts {
		totalW += lipgloss.Width(p)
		if i > 0 {
			totalW += sepW
		}
	}
	if totalW <= maxW {
		return strings.Join(parts, sep)
	}
	var sb strings.Builder
	cur := 0
	for i, p := range parts {
		pw := lipgloss.Width(p)
		need := pw
		if i > 0 {
			need += sepW
		}
		if cur+need > maxW {
			break
		}
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(p)
		cur += need
	}
	return sb.String()
}

func fitPlainParts(parts []string, sep string, maxW int) string {
	if len(parts) == 0 {
		return ""
	}
	sepW := len(sep)
	totalW := 0
	for i, p := range parts {
		totalW += len(p)
		if i > 0 {
			totalW += sepW
		}
	}
	if totalW <= maxW {
		return strings.Join(parts, sep)
	}
	var sb strings.Builder
	cur := 0
	for i, p := range parts {
		pw := len(p)
		need := pw
		if i > 0 {
			need += sepW
		}
		if cur+need > maxW {
			break
		}
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(p)
		cur += need
	}
	return sb.String()
}

func applyColWidth(s lipgloss.Style, col int, colWidths []int) lipgloss.Style {
	if col < len(colWidths) && colWidths[col] > 0 {
		return s.Width(colWidths[col]).MaxWidth(colWidths[col])
	}
	return s.PaddingLeft(1).PaddingRight(1)
}

func appendRunMetricLines(lines []string, st Styles, rs *server.RunState) []string {
	lbls := []string{
		i18n.T(i18n.KSuccessRate),
		i18n.T(i18n.KAvgTPS),
		i18n.T(i18n.KAvgTTFT),
		i18n.T(i18n.KCacheHit),
		i18n.T(i18n.KRPM),
		i18n.T(i18n.KTPM),
	}
	lw := shared.MaxLabelWidth(lbls)
	lines = append(lines, " "+labelValue(st, lbls[0], st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate)), lw))
	lines = append(lines, " "+labelValue(st, lbls[1], st.MetricVal.Render(fmt.Sprintf("%.1f tok/s", rs.AvgTPS)), lw))
	lines = append(lines, " "+labelValue(st, lbls[2], st.MetricVal.Render(shared.FmtDuration(rs.AvgTTFT)), lw))
	lines = append(lines, " "+labelValue(st, lbls[3], st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.CacheHitRate*100)), lw))
	lines = append(lines, " "+labelValue(st, lbls[4], st.MetricVal.Render(fmt.Sprintf("%.0f req/min", rs.RPM)), lw))
	lines = append(lines, " "+labelValue(st, lbls[5], st.MetricVal.Render(fmt.Sprintf("%.0f tok/min", rs.TPM)), lw))
	return lines
}

func panelTitleLines(st Styles, title string, width int, compact bool) []string {
	var lines []string
	if compact {
		lines = append(lines, st.SectionHead.Render(shared.Truncate(title, width)))
	} else {
		lines = append(lines, st.SectionHead.Render(shared.Truncate(title, width)))
		lines = append(lines, dividerLine(st, width))
	}
	return lines
}

func finishPanelLines(lines []string, maxH int) string {
	if maxH < 1 {
		maxH = len(lines)
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	// 填充空行到maxH，保持高度一致
	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func renderSplitPanels(st Styles, leftFrame, rightFrame PanelFrame, leftContent, rightContent string) string {
	return leftContent + "\n" + rightContent
}

func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func labelValue(st Styles, label, value string, labelW ...int) string {
	l := label
	if len(labelW) > 0 && labelW[0] > 0 {
		l = padToDisplayWidth(label, labelW[0])
	}
	return st.Label.Render(l) + "  " + st.Value.Render(value)
}

func wrapPanel(st Styles, content string, outerW int) string {
	if outerW < 4 {
		return content
	}
	return st.Panel.Width(outerW).Render(content)
}

func ensureVisibleOffset(selected, count, offset, visible int) int {
	if count <= 0 {
		return 0
	}
	if visible < 1 {
		visible = 1
	}
	selected = clampInt(selected, 0, count-1)
	maxOffset := shared.MaxInt(0, count-visible)
	offset = clampInt(offset, 0, maxOffset)
	if selected < offset {
		offset = selected
	}
	if selected >= offset+visible {
		offset = selected - visible + 1
	}
	return clampInt(offset, 0, maxOffset)
}

func wrapIndex(idx, count int) int {
	if count <= 0 {
		return 0
	}
	return ((idx % count) + count) % count
}

func maskAPIKey(key string) string {
	r := []rune(key)
	if len(r) <= 8 {
		return strings.Repeat("•", len(r))
	}
	return string(r[:4]) + "••••••••" + string(r[len(r)-4:])
}

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

func promptSummary(promptMode, promptText, promptFile string, promptLength int) string {
	switch promptMode {
	case "file":
		return i18n.T(i18n.KFileSummaryPfx) + promptFile
	case "generated":
		return fmt.Sprintf(i18n.T(i18n.KWzGeneratedFmt), promptLength)
	case "raw":
		if promptText != "" {
			r := []rune(promptText)
			if len(r) > 20 {
				return "RAW: " + string(r[:20]) + "…"
			}
			return "RAW: " + promptText
		}
		return i18n.T(i18n.KNotSet)
	default:
		if promptText != "" {
			r := []rune(promptText)
			if len(r) > 20 {
				return string(r[:20]) + "…"
			}
			return promptText
		}
		return i18n.T(i18n.KNotSet)
	}
}

func boolLabel(b bool) string {
	if b {
		return i18n.T(i18n.KEnabled)
	}
	return i18n.T(i18n.KDisabled)
}
