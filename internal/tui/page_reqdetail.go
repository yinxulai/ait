package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
)

// reqDetailState 请求详情页的状态。
type reqDetailState struct {
	runID    server.RunID
	requests []*server.RequestMetrics
	index    int
}

// ─── 按键处理 ─────────────────────────────────────────────────────────────────

func (m *Model) handleReqDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rd := m.reqDetail
	if rd == nil {
		m.view = viewDashboard
		return m, nil
	}

	switch msg.String() {
	case "left", "h":
		if rd.index > 0 {
			rd.index--
		} else {
			rd.index = len(rd.requests) - 1
		}
	case "right", "l":
		if rd.index < len(rd.requests)-1 {
			rd.index++
		} else {
			rd.index = 0
		}
	case "b", "esc", "backspace":
		m.view = viewDashboard
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// ─── 渲染 ─────────────────────────────────────────────────────────────────────

func (m *Model) renderReqDetail() string {
	rd := m.reqDetail
	if rd == nil || m.width == 0 {
		return "加载中..."
	}
	if len(rd.requests) == 0 {
		return "无请求数据"
	}

	idx := rd.index
	if idx < 0 {
		idx = 0
	}
	if idx >= len(rd.requests) {
		idx = len(rd.requests) - 1
	}
	r := rd.requests[idx]

	header := m.renderHeader(
		fmt.Sprintf("AIT  请求详情  #%d / %d", idx+1, len(rd.requests)),
		statusStr(m, r),
	)
	contextBar := m.renderContextBar(contextBarItems_reqdetail())
	footer := m.renderFooter("[←→] 切换", "[Esc] 返回仪表盘", "", "◆ AIT")

	cbH := 0
	if contextBar != "" {
		cbH = 1
	}
	contentH := m.height - 1 - cbH - 1
	if contentH < 4 {
		contentH = 4
	}

	leftW := (m.width - 4) * 50 / 100
	rightW := m.width - 4 - leftW

	leftContent := m.buildReqLeft(r, contentH, leftW)
	rightContent := m.buildReqRight(r, contentH)
	mid := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, contentH)

	parts := []string{header, mid}
	if contextBar != "" {
		parts = append(parts, contextBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func statusStr(m *Model, r *server.RequestMetrics) string {
	if r.Success {
		return m.styles.ok.Render("成功")
	}
	return m.styles.errStyle.Render("失败")
}

func (m *Model) buildReqLeft(r *server.RequestMetrics, maxH, width int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("时间指标"))
	lines = append(lines, "")
	lines = append(lines, row(m, "总耗时      ", fmt.Sprintf("%d ms", r.TotalTime.Milliseconds())))
	lines = append(lines, row(m, "TTFT        ", fmt.Sprintf("%d ms", r.TTFT.Milliseconds())))
	lines = append(lines, row(m, "TPS         ", m.styles.metricVal.Render(fmt.Sprintf("%.2f tok/s", r.TPS))))
	lines = append(lines, "")
	lines = append(lines, m.styles.sectionHead.Render("Token 统计"))
	lines = append(lines, "")
	lines = append(lines, row(m, "Prompt Tok  ", fmt.Sprintf("%d", r.PromptTokens)))
	lines = append(lines, row(m, "Output Tok  ", fmt.Sprintf("%d", r.CompletionTokens)))
	lines = append(lines, row(m, "缓存命中    ", fmt.Sprintf("%d tok (%.1f%%)", r.CachedTokens, r.CacheHitRate*100)))
	lines = append(lines, "")

	if r.ErrorMessage != "" {
		lines = append(lines, m.styles.sectionHead.Render("错误信息"))
		lines = append(lines, "")
		for _, part := range wrapText(r.ErrorMessage, width-4) {
			lines = append(lines, m.styles.errStyle.Render("  "+part))
		}
	}

	if r.PromptText != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.sectionHead.Render("Prompt"))
		lines = append(lines, "")
		for _, part := range wrapText(r.PromptText, width-4) {
			if len(lines) >= maxH-1 {
				break
			}
			lines = append(lines, "  "+part)
		}
	}
	return strings.Join(lines, "\n")
}

func (m *Model) buildReqRight(r *server.RequestMetrics, maxH int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("网络指标"))
	lines = append(lines, "")
	lines = append(lines, row(m, "目标 IP     ", r.TargetIP))
	lines = append(lines, row(m, "DNS 解析    ", fmt.Sprintf("%d ms", r.DNSTime.Milliseconds())))
	lines = append(lines, row(m, "TCP 连接    ", fmt.Sprintf("%d ms", r.ConnectTime.Milliseconds())))
	lines = append(lines, row(m, "TLS 握手    ", fmt.Sprintf("%d ms", r.TLSTime.Milliseconds())))
	lines = append(lines, "")

	if r.ResponseText != "" {
		lines = append(lines, m.styles.sectionHead.Render("Response"))
		lines = append(lines, "")
		for _, part := range wrapText(r.ResponseText, 40) {
			if len(lines) >= maxH-1 {
				break
			}
			lines = append(lines, "  "+part)
		}
	}
	return strings.Join(lines, "\n")
}

// wrapText 按宽度折行（简单按字节宽度，不处理 CJK）。
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var result []string
	runes := []rune(s)
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[:end]))
		runes = runes[end:]
	}
	return result
}
