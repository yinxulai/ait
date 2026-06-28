package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// WarningPage 尺寸不足告警 overlay（v1）。
type WarningPage struct {
	root      *tview.Flex
	header    *tview.TextView
	footer    *tview.TextView
	textView  *tview.TextView
}

// NewWarningPage 创建告警页。
func NewWarningPage(version string) *WarningPage {
	p := &WarningPage{}

	p.header = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	p.header.SetText(fmt.Sprintf("◆ AIT  v%s", version))

	p.textView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	p.textView.SetText("Terminal too small.\nPlease resize to at least 120×16.")

	p.footer = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	p.footer.SetText("[::b]Esc=关闭提示[::-]")

	p.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(p.header, 1, 0, false).
		AddItem(p.textView, 0, 1, true).
		AddItem(p.footer, 1, 0, false)

	return p
}

// UpdateText 刷新尺寸提示文本。
func (p *WarningPage) UpdateText(w, h int) {
	p.textView.SetText(
		fmt.Sprintf(
			"Terminal too small (%dx%d).\nPlease resize to at least 120×16.",
			w, h,
		),
	)
}

// ─── Page 接口实现 ─────────────────────────────────────────────────────────────

func (p *WarningPage) Name() string                { return "warning" }
func (p *WarningPage) Primitive() tview.Primitive   { return p.root }
func (p *WarningPage) FocusTarget() tview.Primitive { return p.textView }
func (p *WarningPage) OnActivate()                  {}
func (p *WarningPage) OnDeactivate()                {}

func (p *WarningPage) HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		router.HideOverlay()
		return nil
	}
	return nil // 吃掉所有输入
}
