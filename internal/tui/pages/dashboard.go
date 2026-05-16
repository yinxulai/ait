package pages

import (
	"fmt"
	"strings"

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
	ReqSel   int // 选中请求索引（-1 = 无选中）
	ReqOff   int // 滚动偏移
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

// AdjustReqOffset 根据 ReqSel 调整列表可见窗口。
func (d *DashboardState) AdjustReqOffset(visH int) {
	if d == nil {
		return
	}
	if visH < 3 {
		visH = 3
	}
	sel := d.ReqSel
	off := d.ReqOff
	if sel < 0 {
		return
	}
	if sel < off {
		off = sel
	} else if sel >= off+visH {
		off = sel - visH + 1
	}
	d.ReqOff = off
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
		if d.ReqSel <= 0 {
			d.ReqSel = len(reqs) - 1
		} else {
			d.ReqSel--
		}
		d.AdjustReqOffset(10)

	case "down", "j":
		if len(reqs) == 0 {
			break
		}
		if d.ReqSel < len(reqs)-1 {
			d.ReqSel++
		} else {
			d.ReqSel = 0
		}
		d.AdjustReqOffset(10)

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
		nav = NavAction{To: NavTaskList}

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
	if d == nil || width == 0 {
		return "加载中..."
	}
	rs := d.RunState

	// ── Header ──
	statusStr := "等待中"
	if rs != nil {
		switch rs.Status {
		case server.RunStatusRunning:
			statusStr = st.Ok.Render("运行中")
		case server.RunStatusCompleted:
			statusStr = st.Ok.Render("已完成")
		case server.RunStatusFailed:
			statusStr = st.ErrStyle.Render("失败")
		case server.RunStatusStopped:
			statusStr = st.Muted.Render("已停止")
		}
	}

	subtitle := "─"
	if rs != nil {
		subtitle = fmt.Sprintf("◆ AIT   %s · %s · 并发: %d · 请求: %d",
			"─", "─", 0, rs.TotalReqs)
	}

	header := renderHeader(st, width,
		"AIT  正在测试 ─ "+truncate(taskName, 25),
		statusStr,
		subtitle,
		"",
	)

	// ── Context Bar ──
	var cbItems []ContextBarItem
	if d.ReqSel >= 0 && rs != nil && d.ReqSel < len(rs.Requests) {
		cbItems = CtxBar_Dashboard_Sel()
	} else {
		cbItems = CtxBar_Dashboard_NoSel()
	}
	ctxBar := RenderContextBar(st, width, cbItems)

	// ── Footer ──
	footer := renderFooter(st, width, "[s] 停止", "[b] 后台运行", "[r] 提前报告", "[q] 退出")

	// ── 计算高度 ──
	headerH := 2
	ctxBarH := 0
	if ctxBar != "" {
		ctxBarH = 1
	}
	footerH := 1
	splitH := 9  // 上方双栏区域高度
	progressH := 1 // 进度条行高
	divH := 3   // 分隔线总行数（3条分隔线各占1行）
	reqListH := height - headerH - ctxBarH - footerH - splitH - progressH - divH
	if reqListH < 3 {
		reqListH = 3
	}

	// ── 双栏（任务参数 ║ 实时指标）──
	leftW := (width - 2) * 45 / 100
	rightW := width - 2 - leftW - 1 // -1 for separator │
	leftContent := buildDashParamsPanel(d, rs, st, splitH-1, leftW)
	rightContent := buildDashMetricsPanel(rs, st, splitH-1, rightW)
	splitDiv := dividerLine(st, width)
	split := dualColumnLayout(st, leftContent, rightContent, leftW, rightW, splitH)

	// ── 进度条 ──
	progressLine := buildProgressLine(rs, st, width)

	// ── 请求列表 ──
	reqDiv := dividerLine(st, width)
	reqList := buildRequestList(d, rs, st, width, reqListH)

	parts := []string{header, splitDiv, split, splitDiv, progressLine, reqDiv, reqList}
	if ctxBar != "" {
		parts = append(parts, ctxBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
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
			st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate*100))))
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
	barW := 20
	bar := progressBar(ratio, barW)
	barRendered := st.Ok.Render(strings.Repeat("█", int(ratio*float64(barW)))) +
		st.Muted.Render(strings.Repeat("░", barW-int(ratio*float64(barW))))

	elapsed := ""
	if !rs.StartedAt.IsZero() {
		// elapsed time display
		elapsed = "─"
	}

	line := fmt.Sprintf(" 进度  %s  %d / %d   %s",
		barRendered, done, total, elapsed)
	_ = bar
	if lipgloss.Width(line) > width {
		line = truncate(line, width)
	}
	return line
}

// buildRequestList 构建请求列表区域。
func buildRequestList(d *DashboardState, rs *server.RunState, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("请求列表"))

	if rs == nil || len(rs.Requests) == 0 {
		lines = append(lines, " "+st.Muted.Render("等待请求..."))
		for len(lines) < maxH {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	// 表头
	lines = append(lines, " "+st.TableHead.Render(
		padRight("#", 6)+padRight("状态", 6)+padRight("总耗时", 10)+
			padRight("TTFT", 10)+padRight("Cache", 8)+padRight("输出Token", 10)+"TPS"))
	lines = append(lines, " "+st.Divider.Render(strings.Repeat("─", width-2)))

	reqs := rs.Requests
	// 倒序展示（最新在上方）
	off := d.ReqOff
	for i := len(reqs) - 1 - off; i >= 0; i-- {
		if len(lines) >= maxH {
			break
		}
		r := reqs[i]
		isSel := i == d.ReqSel

		statusStr := st.Ok.Render("✓")
		if !r.Success {
			statusStr = st.ErrStyle.Render("✗")
		}
		totalTime := fmtDuration(r.TotalTime)
		if !r.Success && r.ErrorMessage != "" {
			totalTime = st.ErrStyle.Render("timeout")
		}
		ttft := fmtDuration(r.TTFT)
		cache := fmt.Sprintf("%.0f%%", r.CacheHitRate*100)
		tok := fmt.Sprintf("%dtok", r.CompletionTokens)
		tps := fmt.Sprintf("%.1f/s", r.TPS)

		row := fmt.Sprintf(" %s %s  %s  %s  %s  %s  %s",
			padRight(fmt.Sprintf("#%d", r.Index+1), 5),
			statusStr,
			padRight(totalTime, 9),
			padRight(ttft, 9),
			padRight(cache, 7),
			padRight(tok, 9),
			tps,
		)

		var rendered string
		cursorStr := "  "
		if isSel {
			cursorStr = "▶ "
		}
		if isSel {
			rendered = st.TableRowSel.Render(cursorStr+row) +
				strings.Repeat(" ", max(0, width-lipgloss.Width(cursorStr+row)-2))
		} else {
			rendered = "  " + st.TableRow.Render(row)
		}
		lines = append(lines, rendered)
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}
