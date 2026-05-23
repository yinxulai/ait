package pages

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
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

	case "?":
		nav = NavAction{To: NavHelp}

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
	status := "失败"
	if r.Success {
		status = "成功"
	}

	l := PageLayout{
		HeaderTitle:     "请求详情",
		HeaderSubtitle:  "查看单次请求的耗时、网络阶段和完整报文",
		HeaderMeta:      truncate(string(s.RunID), 18),
		HeaderInfoLeft:  []string{fmt.Sprintf("请求 %d/%d", idx+1, len(s.Requests)), status},
		HeaderInfoRight: []string{fmt.Sprintf("缓存 %.0f%%", r.CacheHitRate*100), "耗时 " + fmtDuration(r.TotalTime)},
		Hotkeys:         NewPageHotkeysWithHelp(Hotkeys_ReqDetail(), "[b/Esc] 返回上一页", "[q] 退出"),
	}
	frame := l.Frame(width, height)
	bodyPanel := frame.InnerPanel()

	// ── 计算高度 ──
	splitOuterH := 9
	inputH := 5
	inputOuterH := inputH + panelBorderV
	outputOuterH := RemainingStackOuterHeight(frame.InnerHeight, splitOuterH, inputOuterH)
	outputH := PanelContentHeight(outputOuterH)

	// ── 双栏面板（性能指标 | 网络指标）──
	leftPanelFrame, rightPanelFrame := bodyPanel.Split(50, 24)
	leftContent := buildReqPerfPanel(r, st, PanelContentHeight(splitOuterH), leftPanelFrame.InnerWidth)
	rightContent := buildReqNetworkPanel(r, st, PanelContentHeight(splitOuterH), rightPanelFrame.InnerWidth)
	split := renderSplitPanels(st, leftPanelFrame, rightPanelFrame, leftContent, rightContent)

	// ── 输入区面板 ──
	inputSection := buildInputSection(r, st, bodyPanel.InnerWidth, inputH)
	inputPanelStr := bodyPanel.Wrap(st, inputSection)

	// ── 输出区面板 ──
	outputSection := buildOutputSection(r, s.ScrollY, st, bodyPanel.InnerWidth, outputH)
	outputPanelStr := bodyPanel.Wrap(st, outputSection)

	content := joinVerticalBlocks(split, inputPanelStr, outputPanelStr)
	return l.Assemble(frame.Wrap(st, content), st, width)
}

// buildReqPerfPanel 构建请求左侧性能指标面板。
func buildReqPerfPanel(r *types.RequestMetrics, st Styles, maxH, width int) string {
	lines := panelTitleLines(st, "性能指标", width, true)

	if r == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
		return finishPanelLines(lines, maxH)
	}

	statusStr := st.Ok.Render("✓ 成功")
	if !r.Success {
		statusStr = st.ErrStyle.Render("✗ 失败")
	}
	totalTime := "─"
	if r.TotalTime > 0 {
		totalTime = fmtDuration(r.TotalTime)
	}
	ttft := "─"
	if r.TTFT > 0 {
		ttft = fmtDuration(r.TTFT)
	}
	tps := "─"
	if r.TPS > 0 {
		tps = fmt.Sprintf("%.1f tok/s", r.TPS)
	}
	tokenSummary := fmt.Sprintf("%d in / %d out", r.PromptTokens, r.CompletionTokens)
	cacheSummary := fmt.Sprintf("%d tok (%.1f%%)", r.CachedTokens, r.CacheHitRate*100)
	errorSummary := "—"
	if !r.Success {
		errorSummary = normalizeInlineText(r.ErrorMessage)
		if errorSummary == "" {
			errorSummary = "请求失败"
		}
		errorSummary = truncate(errorSummary, maxInt(8, width-8))
	}

	lines = append(lines, " "+labelValue(st, "状态    ", statusStr))
	lines = append(lines, " "+labelValue(st, "总耗时  ", st.MetricVal.Render(totalTime)))
	lines = append(lines, " "+labelValue(st, "TTFT    ", st.MetricVal.Render(ttft)))
	lines = append(lines, " "+labelValue(st, "输出TPS ", st.MetricVal.Render(tps)))
	lines = append(lines, " "+labelValue(st, "Token    ", tokenSummary))
	if r.Success {
		lines = append(lines, " "+labelValue(st, "缓存    ", cacheSummary))
	} else {
		lines = append(lines, " "+st.ErrStyle.Render("错误: "+errorSummary))
	}

	return finishPanelLines(lines, maxH)
}

// buildReqNetworkPanel 构建请求右侧网络指标面板。
func buildReqNetworkPanel(r *types.RequestMetrics, st Styles, maxH, width int) string {
	lines := panelTitleLines(st, "网络指标", width, true)

	if r == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
		return finishPanelLines(lines, maxH)
	}

	lines = append(lines, " "+labelValue(st, "DNS      ", fmtDuration(r.DNSTime)))
	lines = append(lines, " "+labelValue(st, "TCP 连接 ", fmtDuration(r.ConnectTime)))
	lines = append(lines, " "+labelValue(st, "TLS 握手 ", fmtDuration(r.TLSTime)))
	if r.TargetIP != "" {
		lines = append(lines, " "+labelValue(st, "目标 IP  ", truncate(r.TargetIP, maxInt(4, width-12))))
	}

	return finishPanelLines(lines, maxH)
}

// buildInputSection 构建输入 (请求体) 区域。
func buildInputSection(r *types.RequestMetrics, st Styles, width, maxH int) string {
	lines := panelTitleLines(st, "请求体 (Request Body)", width, true)
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

	return finishPanelLines(lines, maxH)
}

// buildOutputSection 构建输出 (响应体) 区域。
func buildOutputSection(r *types.RequestMetrics, scrollY int, st Styles, width, maxH int) string {
	lines := panelTitleLines(st, "响应体 (Response Body)", width, true)
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

	return finishPanelLines(lines, maxH)
}
