package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
)

// ReqDetailState 请求详情页状态。
type ReqDetailState struct {
	RunID    server.RunID
	Requests []*server.RequestMetrics
	Index    int    // 当前查看的请求索引
	ScrollY  int    // 输出区域滚动偏移
}

// NewReqDetailState 创建请求详情状态。
func NewReqDetailState(runID server.RunID, reqs []*server.RequestMetrics, index int) *ReqDetailState {
	return &ReqDetailState{
		RunID:    runID,
		Requests: reqs,
		Index:    index,
	}
}

// HandleReqDetailKey 处理请求详情页按键。
func HandleReqDetailKey(s *ReqDetailState, msg tea.KeyMsg) (*ReqDetailState, NavAction) {
	nav := NavAction{}
	if s == nil {
		return s, NavAction{To: NavDashboard}
	}

	switch msg.String() {
	case "left", "h":
		if s.Index > 0 {
			s.Index--
		} else {
			s.Index = len(s.Requests) - 1
		}
		s.ScrollY = 0

	case "right", "l":
		if s.Index < len(s.Requests)-1 {
			s.Index++
		} else {
			s.Index = 0
		}
		s.ScrollY = 0

	case "up", "k":
		if s.ScrollY > 0 {
			s.ScrollY--
		}

	case "down", "j":
		s.ScrollY++

	case "b", "esc", "backspace":
		nav = NavAction{To: NavDashboard}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}

	return s, nav
}

// RenderReqDetail 渲染请求详情页。
//
// 设计稿布局：
//
//	╔══ AIT  请求详情 - task-name  #N ════════╗
//	║  ◆ AIT   任务: name  请求 N/total  ✓ 成功 ║
//	╠══════════════════╦══════════════════════╣
//	║  性能指标           ║  网络指标              ║
//	║  状态  ✓ 成功      ║  DNS       1.2ms      ║
//	║  总耗时 245ms      ║  TCP 连接  2.1ms      ║
//	║  TTFT  82ms       ║  TLS 握手  8.4ms      ║
//	║  TPS   12.3/s     ║                       ║
//	║  输入Token  64     ║                       ║
//	║  输出Token  128    ║                       ║
//	║  缓存命中  100%    ║                       ║
//	╠══════════════════╩══════════════════════╣
//	║  输入 (Prompt)                           ║
//	║  ──────────────────────────────────      ║
//	║  你好，介绍一下你自己。                   ║
//	╠═════════════════════════════════════════╣
//	║  输出 (Response)                         ║
//	║  ──────────────────────────────────      ║
//	║  你好！我是 Claude...                    ║
//	║  （↑↓ 滚动查看完整内容）                  ║
//	╠═════════════════════════════════════════╣
//	║  [b/Esc] 返回仪表盘  [↑↓] 滚动  [←→] 上/下一条 ║
//	╚═════════════════════════════════════════╝
func RenderReqDetail(s *ReqDetailState, taskName string, st Styles, width, height int) string {
	if s == nil || width == 0 {
		return "加载中..."
	}
	if len(s.Requests) == 0 {
		return "无请求数据"
	}

	idx := s.Index
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.Requests) {
		idx = len(s.Requests) - 1
	}
	r := s.Requests[idx]

	// ── Header ──
	statusStr := st.Ok.Render("✓ 成功")
	if !r.Success {
		statusStr = st.ErrStyle.Render("✗ 失败")
	}
	header := renderHeader(st, width,
		fmt.Sprintf("AIT  请求详情 - %s  #%d", truncate(taskName, 20), idx+1),
		statusStr,
		fmt.Sprintf("◆ AIT   任务: %s  请求 %d / %d",
			truncate(taskName, 20), idx+1, len(s.Requests)),
		"",
	)

	// ── Context Bar ──
	ctxBar := RenderContextBar(st, width, CtxBar_ReqDetail())

	// ── Footer ──
	footer := renderFooter(st, width, "[b/Esc] 返回仪表盘", "[↑↓] 滚动", "[←→] 上/下一条请求")

	// ── 计算高度 ──
	// 布局：header(2) + split面板(splitH) + 输入面板(inputH+2) + 输出面板(outputH+2) + ctxBarH + footer(1)
	headerH := 2
	ctxBarH := 0
	if ctxBar != "" {
		ctxBarH = 1
	}
	footerH := 1
	splitH := 9
	inputH := 5
	outputH := height - headerH - ctxBarH - footerH - splitH - inputH - 2 - 2 // -2 for input panel border, -2 for output panel border
	if outputH < 4 {
		outputH = 4
	}

	// ── 双栏面板（性能指标 | 网络指标）──
	leftW := width * 50 / 100
	rightW := width - leftW
	leftContent := buildReqPerfPanel(r, st, splitH-2, leftW-2)
	rightContent := buildReqNetworkPanel(r, st, splitH-2, rightW-2)
	leftPanel := st.Panel.Width(leftW - 2).Render(leftContent)
	rightPanel := st.Panel.Width(rightW - 2).Render(rightContent)
	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// ── 输入区面板 ──
	inputSection := buildInputSection(r, st, width-2, inputH)
	inputPanelStr := wrapPanel(st, inputSection, width)

	// ── 输出区面板 ──
	outputSection := buildOutputSection(r, s.ScrollY, st, width-2, outputH)
	outputPanelStr := wrapPanel(st, outputSection, width)

	parts := []string{header, split, inputPanelStr, outputPanelStr}
	if ctxBar != "" {
		parts = append(parts, ctxBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

// buildReqPerfPanel 构建请求左侧性能指标面板。
func buildReqPerfPanel(r *server.RequestMetrics, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("性能指标"))
	lines = append(lines, "")

	statusStr := st.Ok.Render("✓ 成功")
	if !r.Success {
		statusStr = st.ErrStyle.Render("✗ 失败")
	}
	lines = append(lines, " "+labelValue(st, "状态    ", statusStr))

	if r.Success {
		lines = append(lines, " "+labelValue(st, "总耗时  ", st.MetricVal.Render(fmtDuration(r.TotalTime))))
		lines = append(lines, " "+labelValue(st, "TTFT    ", st.MetricVal.Render(fmtDuration(r.TTFT))))
		lines = append(lines, " "+labelValue(st, "输出TPS ", st.MetricVal.Render(fmt.Sprintf("%.1f tok/s", r.TPS))))
		lines = append(lines, " "+labelValue(st, "输入Token", fmt.Sprintf("%d", r.PromptTokens)))
		lines = append(lines, " "+labelValue(st, "输出Token", fmt.Sprintf("%d", r.CompletionTokens)))
		lines = append(lines, " "+labelValue(st, "缓存命中", fmt.Sprintf("%d tok (%.1f%%)", r.CachedTokens, r.CacheHitRate*100)))
	} else {
		if r.ErrorMessage != "" {
			lines = append(lines, " "+st.ErrStyle.Render("错误: "+truncate(r.ErrorMessage, width-8)))
		}
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildReqNetworkPanel 构建请求右侧网络指标面板。
func buildReqNetworkPanel(r *server.RequestMetrics, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("网络指标"))
	lines = append(lines, "")
	lines = append(lines, " "+labelValue(st, "DNS      ", fmtDuration(r.DNSTime)))
	lines = append(lines, " "+labelValue(st, "TCP 连接 ", fmtDuration(r.ConnectTime)))
	lines = append(lines, " "+labelValue(st, "TLS 握手 ", fmtDuration(r.TLSTime)))
	if r.TargetIP != "" {
		lines = append(lines, " "+labelValue(st, "目标 IP  ", r.TargetIP))
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildInputSection 构建输入 (Prompt) 区域。
func buildInputSection(r *server.RequestMetrics, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("输入 (Prompt)"))
	lines = append(lines, " "+dividerLine(st, width-2))

	if r.PromptText == "" {
		lines = append(lines, " "+st.Muted.Render("(未记录)"))
	} else {
		for _, l := range wrapText(r.PromptText, width-3) {
			if len(lines) >= maxH-1 {
				break
			}
			lines = append(lines, " "+l)
		}
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildOutputSection 构建输出 (Response) 区域。
func buildOutputSection(r *server.RequestMetrics, scrollY int, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("输出 (Response)"))
	lines = append(lines, " "+dividerLine(st, width-2))

	if r.ResponseText == "" {
		lines = append(lines, " "+st.Muted.Render("(未记录)"))
	} else {
		allLines := wrapText(r.ResponseText, width-3)
		if scrollY >= len(allLines) {
			scrollY = len(allLines) - 1
		}
		if scrollY < 0 {
			scrollY = 0
		}
		for _, l := range allLines[scrollY:] {
			if len(lines) >= maxH-1 {
				break
			}
			lines = append(lines, " "+l)
		}
		if len(allLines) > maxH-3 {
			lines = append(lines, " "+st.Muted.Render("（↑↓ 滚动查看完整内容）"))
		}
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}
