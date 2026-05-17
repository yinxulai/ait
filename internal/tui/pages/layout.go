package pages

import "strings"

// ── 尺寸常量 ──────────────────────────────────────────────────────────────────

const (
	// MinWidth / MinHeight：低于此值时显示"窗口过小"提示而非正常页面。
	MinWidth  = 40
	MinHeight = 10

	// chrome 各组成部分的行数（仅保留单条合并底栏）
	chromeHeaderH = 0 // 顶部 header 已移除
	chromeFooterH = 1 // 单行底部状态栏（含上下文操作 + 全局导航，已合并）

	// panelBorderV 是单个面板的上下边框行数之和。
	panelBorderV = 2
)

// ── PageLayout ────────────────────────────────────────────────────────────────

// PageLayout 描述一个完整页面的 chrome（底部 ContextBar + Footer）。
// 各页面 Render 函数先构造 PageLayout，再调用 Assemble 拼装最终输出。
type PageLayout struct {
	TitleLeft   string
	TitleRight  string
	InfoLeft    string
	InfoRight   string
	CtxItems    []ContextBarItem
	FooterParts []string
}

// ChromeHeight 返回 chrome 占用的总行数（当前仅包含合并底栏）。
func (l PageLayout) ChromeHeight() int {
	return chromeHeaderH + chromeFooterH
}

// ContentHeight 返回单面板页面主内容区的可用行数
// （总高度 - chrome 行数 - 面板上下边框）。
func (l PageLayout) ContentHeight(totalH int) int {
	h := totalH - l.ChromeHeight() - panelBorderV
	if h < 2 {
		h = 2
	}
	return h
}

// ContentWidth 返回单面板页面内容区的可用列宽
// （总宽度 - 面板左右边框）。
func ContentWidth(totalW int) int {
	w := totalW - 2
	if w < 1 {
		w = 1
	}
	return w
}

// Assemble 拼装完整页面输出：
//
//	content
//	底栏（上下文操作 · 全局导航，合并为单行）
func (l PageLayout) Assemble(content string, st Styles, width int) string {
	// 将上下文操作与全局导航合并为单条底栏，用 · 分隔
	var barParts []string
	for _, item := range l.CtxItems {
		barParts = append(barParts, "["+item.Key+"] "+item.Desc)
	}
	if len(l.CtxItems) > 0 && len(l.FooterParts) > 0 {
		barParts = append(barParts, "·")
	}
	barParts = append(barParts, l.FooterParts...)
	footer := renderFooter(st, width, barParts...)

	return strings.Join([]string{content, footer}, "\n")
}

// ── 最小尺寸保护 ──────────────────────────────────────────────────────────────

// TooSmall 返回 true 当终端小于最小可用尺寸。
func TooSmall(width, height int) bool {
	return width < MinWidth || height < MinHeight
}

// renderTooSmall 返回终端过小时的简洁提示。
func renderTooSmall(st Styles, width, _ int) string {
	if width < 4 {
		return "..."
	}
	return st.Muted.Render(truncate("窗口过小 ↔ 请放大终端", width))
}
