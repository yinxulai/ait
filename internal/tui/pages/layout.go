package pages

import (
	"strings"

	"github.com/yinxulai/ait/internal/i18n"
)

// ── 尺寸常量 ──────────────────────────────────────────────────────────────────

const (
	// MinWidth / MinHeight：低于此值时显示"窗口过小"提示而非正常页面。
	MinWidth  = 40
	MinHeight = 12

	// chrome 各组成部分的行数（三行 AppHeader + 双行 Hotkeys）
	chromeHeaderH  = 3
	chromeHotkeysH = 2

	// panelBorderV 是单个面板的上下边框行数之和。
	panelBorderV = 2

	// appBorderV 是应用外层边框的上下行数之和。
	appBorderV = 2
	// appBorderH 是应用外层边框的左右列数之和。
	appBorderH = 2
)

// ── PageLayout ────────────────────────────────────────────────────────────────

// PageLayout 描述一个完整页面的共享 chrome（AppHeader + Hotkeys）。
// 各页面 Render 函数只提供标题、状态信息和底部 Hotkeys，Assemble 负责统一拼装。
type PageLayout struct {
	HeaderTitle     string
	HeaderSubtitle  string
	HeaderMeta      string
	HeaderInfoLeft  []string
	HeaderInfoRight []string
	Hotkeys         PageHotkeys
}

// PageHotkeys 描述页面底部统一的 Hotkeys 区域。
// Hotkeys 用于当前页快捷操作，Hints 用于返回、退出等全局提示。
type PageHotkeys struct {
	Hotkeys []HotkeyItem
	Hints   []HotkeyItem
}

// NewPageHotkeys 用于构建统一的页面 Hotkeys。
// 所有页面（帮助页除外）会自动在 Hotkeys 末尾追加 [?] 帮助提示。
func NewPageHotkeys(hotkeys []HotkeyItem, hints ...string) PageHotkeys {
	return PageHotkeys{
		Hotkeys: hotkeys,
		Hints:   HotkeyTexts(hints...),
	}
}

// NewPageHotkeysWithHelp 在 NewPageHotkeys 基础上自动追加 [?] 帮助和 [F2] 切换语言。
// 非帮助页使用此函数以统一显示帮助入口。
func NewPageHotkeysWithHelp(hotkeys []HotkeyItem, hints ...string) PageHotkeys {
	withHelp := append(hotkeys, HotkeyAction("f2", i18n.T(i18n.KToggleLang)), HotkeyAction("?", i18n.T(i18n.KHelp)))
	return PageHotkeys{
		Hotkeys: withHelp,
		Hints:   HotkeyTexts(hints...),
	}
}

// PageFrame 描述页面主内容区的统一尺寸。
// OuterWidth 是最外层内容面板总宽度，InnerWidth/InnerHeight 是面板内部可用区域。
type PageFrame struct {
	OuterWidth  int
	InnerWidth  int
	InnerHeight int
}

// PanelFrame 描述嵌套子面板的统一尺寸。
// OuterWidth 是含边框总宽度，InnerWidth 是面板内可用宽度。
type PanelFrame struct {
	OuterWidth int
	InnerWidth int
}

// ChromeHeight 返回 chrome 占用的总行数。
func (l PageLayout) ChromeHeight() int {
	return chromeHeaderH + chromeHotkeysH
}

// Frame 统一计算页面主内容区的外层与内层尺寸。
// 外层无边框，InnerWidth 等于 OuterWidth（内层子面板自行管理各自的边框）。
func (l PageLayout) Frame(totalW, totalH int) PageFrame {
	if totalW < 1 {
		totalW = 1
	}
	// 需要扣除应用外层边框占用的空间
	contentW := totalW - appBorderH
	if contentW < 1 {
		contentW = 1
	}
	return PageFrame{
		OuterWidth:  contentW,
		InnerWidth:  contentW,
		InnerHeight: l.ContentHeight(totalH),
	}
}

// Wrap 保留接口兼容，外层不再添加边框，直接返回内容。
func (f PageFrame) Wrap(_ Styles, content string) string {
	return content
}

// InnerPanel 返回可用于嵌套子面板的统一 frame。
func (f PageFrame) InnerPanel() PanelFrame {
	return NewPanelFrame(f.InnerWidth)
}

// NewPanelFrame 创建一个统一的子面板尺寸描述。
func NewPanelFrame(outerW int) PanelFrame {
	if outerW < 1 {
		outerW = 1
	}
	return PanelFrame{OuterWidth: outerW, InnerWidth: ContentWidth(outerW)}
}

// Wrap 用统一子面板包裹内容。
func (f PanelFrame) Wrap(st Styles, content string) string {
	return wrapPanel(st, content, f.OuterWidth)
}

// Split 按比例拆分左右子面板宽度，避免各页重复手写宽度和最小值逻辑。
func (f PanelFrame) Split(leftPercent, minLeftOuter int) (PanelFrame, PanelFrame) {
	total := f.OuterWidth
	if total <= 1 {
		return NewPanelFrame(total), NewPanelFrame(1)
	}
	if leftPercent <= 0 || leftPercent >= 100 {
		leftPercent = 50
	}
	if minLeftOuter < 1 {
		minLeftOuter = 1
	}

	leftOuter := total * leftPercent / 100
	if leftOuter < minLeftOuter {
		leftOuter = minLeftOuter
	}
	if leftOuter >= total {
		leftOuter = total - 1
	}
	rightOuter := total - leftOuter
	if rightOuter < 1 {
		rightOuter = 1
		leftOuter = total - rightOuter
	}

	return NewPanelFrame(leftOuter), NewPanelFrame(rightOuter)
}

// ContentHeight 返回页面主内容区的可用行数（总高度 - chrome 行数 - 应用边框行数）。
// 外层不再有边框，故不扣除 panelBorderV。
func (l PageLayout) ContentHeight(totalH int) int {
	h := totalH - l.ChromeHeight() - appBorderV
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

// PanelContentHeight 将含边框高度转换为子面板可用内容高度。
func PanelContentHeight(outerH int) int {
	h := outerH - panelBorderV
	if h < 1 {
		h = 1
	}
	return h
}

// RemainingStackOuterHeight 计算纵向堆叠场景下，最后一个区块可用的外层高度。
// 会统一扣除前置区块自身高度。各块通过 strings.Join 拼接，总行数 = 各块行数之和，无需额外扣减分隔符。
func RemainingStackOuterHeight(totalH int, fixedOuterHeights ...int) int {
	remaining := totalH
	for _, h := range fixedOuterHeights {
		remaining -= h
	}
	minOuterHeight := panelBorderV + 1
	if remaining < minOuterHeight {
		remaining = minOuterHeight
	}
	return remaining
}

// joinVerticalBlocks 统一拼接纵向区块，避免各页自行处理空块和换行。
func joinVerticalBlocks(blocks ...string) string {
	var visible []string
	for _, block := range blocks {
		if block != "" {
			visible = append(visible, block)
		}
	}
	return strings.Join(visible, "\n")
}

// Assemble 拼装完整页面输出：header + content + hotkeys，最外层包裹应用边框。
func (l PageLayout) Assemble(content string, st Styles, width int) string {
	header := renderHeader(st, width-appBorderH, l.HeaderTitle, l.HeaderSubtitle, l.HeaderMeta, l.HeaderInfoLeft, l.HeaderInfoRight)
	hotkeys := renderHotkeys(st, width-appBorderH, l.Hotkeys)

	inner := strings.Join([]string{header, content, hotkeys}, "\n")
	// 包裹应用外层边框
	return st.AppBorder.Width(width).Render(inner)
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
	return st.Muted.Render(truncate(i18n.T(i18n.KWindowTooSmall), width))
}
