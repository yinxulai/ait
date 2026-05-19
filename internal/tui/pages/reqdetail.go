package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// ReqDetailState 请求详情页状态。
type ReqDetailState struct {
	RunID    server.RunID
	Requests []*types.RequestMetrics
	Index    int       // 当前查看的请求索引
	ScrollY  int       // 输出区域滚动偏移
	BackNav  NavAction // 按 b/esc 时的返回目标
}

// NewReqDetailState 创建请求详情状态。
func NewReqDetailState(runID server.RunID, reqs []*types.RequestMetrics, index int) *ReqDetailState {
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
		if s.BackNav.To != NavNone {
			nav = s.BackNav
		} else {
			nav = NavAction{To: NavDashboard}
		}

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
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}
	if s == nil || len(s.Requests) == 0 {
		return renderTooSmall(st, width, height)
	}

	idx := s.Index
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.Requests) {
		idx = len(s.Requests) - 1
	}
	r := s.Requests[idx]

	l := PageLayout{
		CtxItems:    CtxBar_ReqDetail(),
		FooterParts: []string{"[q] 退出"},
	}

	// ── 计算高度 ──
	splitH := 9
	inputH := 5
	outputH := height - l.ChromeHeight() - splitH - inputH - 2 - 2 // -2 for input panel border, -2 for output panel border
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
	inputSection := buildInputSection(r, st, ContentWidth(width), inputH)
	inputPanelStr := wrapPanel(st, inputSection, width)

	// ── 输出区面板 ──
	outputSection := buildOutputSection(r, s.ScrollY, st, ContentWidth(width), outputH)
	outputPanelStr := wrapPanel(st, outputSection, width)

	content := strings.Join([]string{split, inputPanelStr, outputPanelStr}, "\n")
	return l.Assemble(content, st, width)
}

// buildReqPerfPanel 构建请求左侧性能指标面板。
func buildReqPerfPanel(r *types.RequestMetrics, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("性能指标"))
	lines = append(lines, "")

	if r == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
		for len(lines) < maxH {
			lines = append(lines, "")
		}
		return strings.Join(lines[:maxH], "\n")
	}

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
func buildReqNetworkPanel(r *types.RequestMetrics, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("网络指标"))
	lines = append(lines, "")

	if r == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
		for len(lines) < maxH {
			lines = append(lines, "")
		}
		return strings.Join(lines[:maxH], "\n")
	}

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

// buildInputSection 构建输入 (请求体) 区域。
func buildInputSection(r *types.RequestMetrics, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("请求体 (Request Body)"))
	lines = append(lines, " "+dividerLine(st, width-2))

	if r.RequestBody == "" {
		lines = append(lines, " "+st.Muted.Render("(未记录)"))
	} else {
		for _, l := range wrapText(r.RequestBody, width-3) {
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

// buildOutputSection 构建输出 (响应体) 区域。
func buildOutputSection(r *types.RequestMetrics, scrollY int, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("响应体 (Response Body)"))
	lines = append(lines, " "+dividerLine(st, width-2))

	if r.ResponseBody == "" {
		lines = append(lines, " "+st.Muted.Render("(未记录)"))
	} else {
		allLines := wrapText(r.ResponseBody, width-3)
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
