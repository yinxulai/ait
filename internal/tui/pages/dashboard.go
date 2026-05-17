package pages

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
)

// DashboardState 标准模式运行仪表盘页状态。
type DashboardState struct {
	RunID    server.RunID
	TaskID   string
	EventCh  <-chan server.Event   // nil = 已后台或已结束
	CancelFn server.CancelFunc
	RunState *server.RunState
	ReqSel   int       // 选中请求索引（-1 = 无选中）
	ReqOff   int       // 滚动偏移
	ReqVis   int       // 当前可见请求数
	BackNav  NavAction // 按 b/esc 时的返回目标；Zero = 返回任务列表
}

// NewDashboardState 创建仪表盘状态。
func NewDashboardState(runID server.RunID, taskID string) *DashboardState {
	return &DashboardState{
		RunID:  runID,
		TaskID: taskID,
		ReqSel: -1,
	}
}

// IsRunning 判断运行是否仍在进行。
func (d *DashboardState) IsRunning() bool {
	if d == nil || d.RunState == nil {
		return false
	}
	return d.RunState.Status == server.RunStatusRunning
}

// AdjustReqOffset 根据屏幕显示顺序调整列表可见窗口。
func (d *DashboardState) AdjustReqOffset(visH, total int) {
	if d == nil {
		return
	}
	if visH < 3 {
		visH = 3
	}
	if total <= 0 || d.ReqSel < 0 {
		d.ReqOff = 0
		return
	}
	sel := requestDisplayPos(d.ReqSel, total)
	off := d.ReqOff
	if sel < off {
		off = sel
	} else if sel >= off+visH {
		off = sel - visH + 1
	}
	d.ReqOff = clampInt(off, 0, maxInt(0, total-visH))
}

// HandleDashboardKey 处理仪表盘页按键。
func HandleDashboardKey(d *DashboardState, msg tea.KeyMsg, client Client) (*DashboardState, tea.Cmd, NavAction) {
	nav := NavAction{}
	if d == nil {
		return d, nil, NavAction{To: NavTaskList}
	}

	var reqs []*server.RequestMetrics
	if d.RunState != nil {
		reqs = d.RunState.Requests
	}

	switch msg.String() {
	case "up", "k":
		if len(reqs) == 0 {
			break
		}
		if d.ReqSel < 0 {
			// 无选中项：首次按键选中最新一条（显示列表顶部）
			d.ReqSel = requestIndexFromDisplayPos(0, len(reqs))
		} else {
			selPos := requestDisplayPos(d.ReqSel, len(reqs))
			if selPos <= 0 {
				selPos = len(reqs) - 1
			} else {
				selPos--
			}
			d.ReqSel = requestIndexFromDisplayPos(selPos, len(reqs))
		}
		d.AdjustReqOffset(d.ReqVis, len(reqs))

	case "down", "j":
		if len(reqs) == 0 {
			break
		}
		if d.ReqSel < 0 {
			// 无选中项：首次按键选中最新一条（显示列表顶部）
			d.ReqSel = requestIndexFromDisplayPos(0, len(reqs))
		} else {
			selPos := requestDisplayPos(d.ReqSel, len(reqs))
			if selPos < len(reqs)-1 {
				selPos++
			} else {
				selPos = 0
			}
			d.ReqSel = requestIndexFromDisplayPos(selPos, len(reqs))
		}
		d.AdjustReqOffset(d.ReqVis, len(reqs))

	case "enter":
		if d.ReqSel >= 0 && d.ReqSel < len(reqs) {
			nav = NavAction{To: NavReqDetail, ReqIndex: d.ReqSel}
		}

	case "s":
		if d.IsRunning() {
			return d, client.StopRunCmd(d.RunID), nav
		}

	case "b", "esc":
		if d.CancelFn != nil {
			d.CancelFn()
		}
		d.EventCh = nil
		d.CancelFn = nil
		if d.BackNav.To != NavNone {
			nav = d.BackNav
		} else {
			nav = NavAction{To: NavTaskList}
		}

	case "r":
		if d.RunState != nil && !d.IsRunning() {
			return d, client.GenerateReportCmd(d.RunID, server.ReportFormatJSON), nav
		}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}

	return d, nil, nav
}

// RenderDashboard 渲染标准模式运行仪表盘。
//
// 设计稿布局：
//
//	╔══ AIT  正在测试 ─ task-name ══════════╗
//	║  ◆ AIT   model · protocol · 并发: N · 请求: N ║
//	╠══════════════════╦══════════════════════╣
//	║  任务参数          ║  实时指标              ║
//	║  ...              ║  ...                  ║
//	╠══════════════════╩══════════════════════╣
//	║  进度  ████░░  N/N  已用: Xs  剩余: ~Xs  ║
//	╠═════════════════════════════════════════╣
//	║  请求列表                               ║
//	║  #  状态  总耗时  TTFT  Cache  Token  TPS║
//	║  ──────────────────────────────         ║
//	║  ...                                    ║
//	╠═════════════════════════════════════════╣
//	║  [Enter] 查看请求  [↑↓] 选择  [s] 停止  ║  ← context bar
//	╠═════════════════════════════════════════╣
//	║  [s] 停止  [b] 后台运行  [r] 报告  [q] 退出 ║
//	╚═════════════════════════════════════════╝
func RenderDashboard(d *DashboardState, taskName string, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}
	if d == nil {
		return renderTooSmall(st, width, height)
	}
	rs := d.RunState

	isRunning := d.IsRunning()
	hasSel := d.ReqSel >= 0 && rs != nil && d.ReqSel < len(rs.Requests)
	var cbItems []ContextBarItem
	switch {
	case hasSel && isRunning:
		cbItems = CtxBar_Dashboard_Running_Sel()
	case hasSel && !isRunning:
		cbItems = CtxBar_Dashboard_Done_Sel()
	case !hasSel && isRunning:
		cbItems = CtxBar_Dashboard_Running_NoSel()
	default:
		cbItems = CtxBar_Dashboard_Done_NoSel()
	}
	l := PageLayout{
		CtxItems:    cbItems,
		FooterParts: []string{"[b/Esc] 返回列表", "[q] 退出"},
	}

	// ── 计算高度 ──
	splitH := 9      // 双栏面板外部总高度（含面板边框）
	progressPanel := 3 // 进度条面板外部高度（1内容+2边框）
	reqListH := height - l.ChromeHeight() - splitH - progressPanel - 2 // -2 for req panel border
	if reqListH < 3 {
		reqListH = 3
	}

	// ── 双栏面板（任务参数 | 实时指标）──
	leftW := width * 45 / 100
	rightW := width - leftW
	leftContent := buildDashParamsPanel(d, rs, st, splitH-2, leftW-2)
	rightContent := buildDashMetricsPanel(rs, st, splitH-2, rightW-2)
	leftPanel := st.Panel.Width(leftW - 2).Render(leftContent)
	rightPanel := st.Panel.Width(rightW - 2).Render(rightContent)
	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// ── 进度条面板 ──
	progressLine := buildProgressLine(rs, st, ContentWidth(width))
	progressPanelStr := wrapPanel(st, progressLine, width)

	// ── 请求列表面板 ──
	reqList := buildRequestList(d, rs, st, ContentWidth(width), reqListH)
	reqPanelStr := wrapPanel(st, reqList, width)

	content := strings.Join([]string{split, progressPanelStr, reqPanelStr}, "\n")
	return l.Assemble(content, st, width)
}

// buildDashParamsPanel 构建左侧任务参数面板。
func buildDashParamsPanel(d *DashboardState, rs *server.RunState, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("任务参数"))
	lines = append(lines, "")

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
	} else {
		// 参数从 RunState 读取（实际可从 task 传入，此处用 RunState 已知信息展示）
		lines = append(lines, " "+labelValue(st, "进度", fmt.Sprintf("%d/%d", rs.DoneReqs, rs.TotalReqs)))
		lines = append(lines, " "+labelValue(st, "成功", fmt.Sprintf("%d", rs.SuccessReqs)))
		lines = append(lines, " "+labelValue(st, "失败", fmt.Sprintf("%d", rs.FailedReqs)))
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildDashMetricsPanel 构建右侧实时指标面板。
func buildDashMetricsPanel(rs *server.RunState, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("实时指标"))
	lines = append(lines, "")

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
	} else {
		lines = append(lines, " "+labelValue(st, "成功率  ",
			st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate))))
		lines = append(lines, " "+labelValue(st, "avg TPS ",
			st.MetricVal.Render(fmt.Sprintf("%.1f tok/s", rs.AvgTPS))))
		lines = append(lines, " "+labelValue(st, "avg TTFT",
			st.MetricVal.Render(fmtDuration(rs.AvgTTFT))))
		lines = append(lines, " "+labelValue(st, "缓存命中",
			st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.CacheHitRate*100))))
		lines = append(lines, " "+st.Muted.Render(fmt.Sprintf(" 成功: %d   失败: %d", rs.SuccessReqs, rs.FailedReqs)))
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildProgressLine 构建进度条行。
func buildProgressLine(rs *server.RunState, st Styles, width int) string {
	if rs == nil {
		return " 进度  " + st.Muted.Render("等待中...")
	}
	total := rs.TotalReqs
	done := rs.DoneReqs
	var ratio float64
	if total > 0 {
		ratio = float64(done) / float64(total)
	}
	prefix := " 进度  "
	elapsed := "─"
	if !rs.StartedAt.IsZero() {
		if rs.FinishedAt != nil {
			elapsed = fmtDuration(rs.FinishedAt.Sub(rs.StartedAt))
		} else {
			elapsed = fmtDuration(time.Since(rs.StartedAt))
		}
	}
	suffix := fmt.Sprintf("  %d / %d   %s", done, total, elapsed)

	barW := width - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	if barW < 5 {
		barW = 5
	}

	filled := int(ratio * float64(barW))
	barRendered := st.Ok.Render(strings.Repeat("█", filled)) +
		st.Muted.Render(strings.Repeat("░", barW-filled))

	return prefix + barRendered + suffix
}

// buildRequestList 构建请求列表区域。
func buildRequestList(d *DashboardState, rs *server.RunState, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("请求列表"))

	if rs == nil || len(rs.Requests) == 0 {
		msg := "等待请求..."
		if rs != nil && rs.Status != server.RunStatusRunning {
			msg = "无请求详情数据"
		}
		lines = append(lines, " "+st.Muted.Render(msg))
		for len(lines) < maxH {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	// 列宽（header 与 content 行保持一致，前缀均为 2 字符）
	const (
		markW  = 2  // 选择标记列
		idW    = 6  // "#1" 等
		statW  = 5  // "✓" / "✗" 加空白
		timeW  = 10 // 总耗时
		ttftW  = 10 // TTFT
		cacheW = 8  // Cache
		tokW   = 10 // Token
		// TPS: 余量
	)
	hdr := padRight("", markW) + padRight("#", idW) + padRight("状态", statW) + padRight("总耗时", timeW) +
		padRight("TTFT", ttftW) + padRight("Cache", cacheW) + padRight("Token", tokW) + "TPS"
	lines = append(lines, renderTableHeader(st, width, hdr))
	lines = append(lines, dividerLine(st, width))
	d.ReqVis = listVisibleItems(maxH, 3)
	d.AdjustReqOffset(d.ReqVis, len(rs.Requests))

	reqs := rs.Requests
	start := d.ReqOff
	end := minInt(len(reqs), start+d.ReqVis)
	for pos := start; pos < end; pos++ {
		i := requestIndexFromDisplayPos(pos, len(reqs))
		r := reqs[i]
		isSel := i == d.ReqSel

		statusText := "✓"
		if !r.Success {
			statusText = "✗"
		}
		totalText := fmtDuration(r.TotalTime)
		if !r.Success && r.ErrorMessage != "" {
			totalText = "timeout"
		}

		statusStr := statusText
		if r.Success {
			statusStr = styleWhenNotSelected(isSel, st.Ok, statusText)
		} else {
			statusStr = styleWhenNotSelected(isSel, st.ErrStyle, statusText)
		}
		totalStr := totalText
		if !r.Success && r.ErrorMessage != "" {
			totalStr = styleWhenNotSelected(isSel, st.ErrStyle, totalText)
		}

		marker := selectionMarker(isSel)

		rowContent := padRight(marker, markW) +
			padRight(fmt.Sprintf("#%d", r.Index+1), idW) +
			padRight(statusStr, statW) +
			padRight(totalStr, timeW) +
			padRight(fmtDuration(r.TTFT), ttftW) +
			padRight(fmt.Sprintf("%.0f%%", r.CacheHitRate*100), cacheW) +
			padRight(fmt.Sprintf("%dtok", r.CompletionTokens), tokW) +
			fmt.Sprintf("%.1f/s", r.TPS)

		rendered := renderTableRow(st, width, isSel, rowContent)
		lines = append(lines, rendered)

		// 行间分隔线
		if pos < end-1 && len(lines) < maxH-1 {
			lines = append(lines, dividerLine(st, width))
		}
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

func requestDisplayPos(reqIndex, total int) int {
	if total <= 0 {
		return 0
	}
	return clampInt(total-1-reqIndex, 0, total-1)
}

func requestIndexFromDisplayPos(pos, total int) int {
	if total <= 0 {
		return 0
	}
	return clampInt(total-1-pos, 0, total-1)
}
