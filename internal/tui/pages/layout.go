package pages

import "strings"

// ── 尺寸常量 ──────────────────────────────────────────────────────────────────

const (
	// MinWidth / MinHeight：低于此值时显示"窗口过小"提示而非正常页面。
	MinWidth  = 40
	MinHeight = 10

	// chrome 各组成部分的行数
	chromeHeaderH = 2 // 双行标题栏
	chromeFooterH = 1 // 单行底部状态栏
	chromeCtxBarH = 2 // ContextBar (1) + 分隔线 (1)

	// panelBorderV 是单个面板的上下边框行数之和。
	panelBorderV = 2
)

// ── PageLayout ────────────────────────────────────────────────────────────────

// PageLayout 描述一个完整页面的 chrome（顶部标题栏、底部 ContextBar + 分隔线 + Footer）。
// 各页面 Render 函数先构造 PageLayout，再调用 Assemble 拼装最终输出，
// 从而消除页面间重复的 header/footer/ctxbar 组装逻辑。
type PageLayout struct {
	TitleLeft   string
	TitleRight  string
	InfoLeft    string
	InfoRight   string
	CtxItems    []ContextBarItem
	FooterParts []string
}

// ChromeHeight 返回 chrome 占用的总行数
// （header + 若有 ctxbar 则含 ctxbar+分隔线 + footer）。
func (l PageLayout) ChromeHeight() int {
	h := chromeHeaderH + chromeFooterH
	if len(l.CtxItems) > 0 {
		h += chromeCtxBarH
	}
	return h
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
//	header
//	content          ← 由调用方构建（单面板或多面板）
//	[ctxbar]         ← 仅当 CtxItems 非空时输出
//	[divider]        ← ctxbar 与 footer 之间的分隔线
//	footer
func (l PageLayout) Assemble(content string, st Styles, width int) string {
	header := renderHeader(st, width, l.TitleLeft, l.TitleRight, l.InfoLeft, l.InfoRight)
	footer := renderFooter(st, width, l.FooterParts...)

	parts := []string{header, content}
	if len(l.CtxItems) > 0 {
		ctxBar := RenderContextBar(st, width, l.CtxItems)
		parts = append(parts, ctxBar, dividerLine(st, width))
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
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
