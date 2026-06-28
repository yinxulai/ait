package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server/types"
)

// RequestDetailPage 单条请求详情页（v1）。
type RequestDetailPage struct {
	req  *types.RequestMetrics
	root *tview.Flex
	tv   *tview.TextView
}

// NewRequestDetailPage 创建请求详情页。
func NewRequestDetailPage(req *types.RequestMetrics) *RequestDetailPage {
	p := &RequestDetailPage{req: req}

	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	statusIcon := "✅"
	statusText := "Success"
	if !req.Success {
		statusIcon = "❌"
		statusText = "Failed"
	}
	header.SetText(fmt.Sprintf("◆ Request Detail — #%03d %s %s", req.Index, statusIcon, statusText))

	p.tv = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	p.tv.SetBorder(true).SetTitle(" Metrics ")

	var content string
	content += fmt.Sprintf("Index:       %d\n", req.Index)
	content += fmt.Sprintf("Status:      %s\n", statusText)
	content += fmt.Sprintf("Total Time:  %s\n", req.TotalTime.Truncate(time.Millisecond))
	content += fmt.Sprintf("TTFT:        %s\n", req.TTFT.Truncate(time.Millisecond))
	content += fmt.Sprintf("TPS:         %.1f\n", req.TPS)
	content += fmt.Sprintf("Prompt Tokens:     %d\n", req.PromptTokens)
	content += fmt.Sprintf("Completion Tokens: %d\n", req.CompletionTokens)
	content += fmt.Sprintf("Cached Tokens:     %d\n", req.CachedTokens)
	content += fmt.Sprintf("Cache Hit Rate:    %.1f%%\n", req.CacheHitRate)
	content += fmt.Sprintf("DNS Time:          %s\n", req.DNSTime.Truncate(time.Millisecond))
	content += fmt.Sprintf("Connect Time:      %s\n", req.ConnectTime.Truncate(time.Millisecond))
	content += fmt.Sprintf("TLS Time:          %s\n", req.TLSTime.Truncate(time.Millisecond))
	content += fmt.Sprintf("Target IP:         %s\n", req.TargetIP)

	if req.ErrorMessage != "" {
		content += fmt.Sprintf("\nError: %s\n", req.ErrorMessage)
	}
	if req.RequestBody != "" {
		content += fmt.Sprintf("\n── Request Body ──\n%s\n", req.RequestBody)
	}
	if req.ResponseBody != "" {
		content += fmt.Sprintf("\n── Response Body ──\n%s\n", req.ResponseBody)
	}

	p.tv.SetText(content)

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	footer.SetText("[::b]Esc=返回[::-]")

	p.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(p.tv, 0, 1, true).
		AddItem(footer, 1, 0, false)

	return p
}

// ─── Page 接口实现 ─────────────────────────────────────────────────────────────

func (p *RequestDetailPage) Name() string                { return "request_detail" }
func (p *RequestDetailPage) Primitive() tview.Primitive   { return p.root }
func (p *RequestDetailPage) FocusTarget() tview.Primitive { return p.tv }
func (p *RequestDetailPage) OnActivate()                  {}
func (p *RequestDetailPage) OnDeactivate()                {}

func (p *RequestDetailPage) HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		router.GoBack()
		return nil
	}
	return event
}
