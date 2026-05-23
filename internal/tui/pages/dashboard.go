package pages

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"charm.land/lipgloss/v2"
	lgtable "charm.land/lipgloss/v2/table"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
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

	var reqs []*types.RequestMetrics
	if d.RunState != nil {
		reqs = d.RunState.Requests
	}

	switch msg.String() {
	case "up", "k":
		if len(reqs) == 0 {
			break
		}
		if d.ReqSel < 0 {
			// 无选中项：首次向上按键选中最旧一条（显示列表底部）
			d.ReqSel = requestIndexFromDisplayPos(len(reqs)-1, len(reqs))
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
	var cbItems []HotkeyItem
	switch {
	case hasSel && isRunning:
		cbItems = Hotkeys_Dashboard_Running_Sel()
	case hasSel && !isRunning:
		cbItems = Hotkeys_Dashboard_Done_Sel()
	case !hasSel && isRunning:
		cbItems = Hotkeys_Dashboard_Running_NoSel()
	default:
		cbItems = Hotkeys_Dashboard_Done_NoSel()
	}
	headerLeft := []string{"等待数据"}
	headerRight := []string{}
	if rs != nil {
		headerLeft = []string{runStatusText(string(rs.Status)), fmt.Sprintf("完成 %d/%d", rs.DoneReqs, rs.TotalReqs)}
		headerRight = []string{fmt.Sprintf("成功率 %.1f%%", rs.SuccessRate)}
		if !rs.StartedAt.IsZero() {
			headerRight = append(headerRight, "开始 "+fmtRelativeTime(rs.StartedAt))
		}
	}
	if d.TaskID != "" {
		headerRight = append(headerRight, "任务 "+truncate(d.TaskID, 14))
	}
	l := PageLayout{
		HeaderTitle:     "标准运行监控",
		HeaderSubtitle:  "实时查看运行进度、吞吐和单请求明细",
		HeaderMeta:      "标准模式",
		HeaderInfoLeft:  headerLeft,
		HeaderInfoRight: headerRight,
		Hotkeys:         NewPageHotkeys(cbItems, "[b/Esc] 返回上一页", "[q] 退出"),
	}
	frame := l.Frame(width, height)
	bodyPanel := frame.InnerPanel()

	// ── 计算高度 ──
	splitOuterH := 7    // 双栏面板外部总高度（含面板边框）
	progressOuterH := 3 // 进度条面板外部高度（1内容+2边框）
	reqOuterH := RemainingStackOuterHeight(frame.InnerHeight, splitOuterH, progressOuterH)
	reqListH := PanelContentHeight(reqOuterH)

	// ── 双栏面板（任务参数 | 实时指标）──
	leftPanelFrame, rightPanelFrame := bodyPanel.Split(45, 24)
	leftContent := buildDashParamsPanel(d, rs, st, PanelContentHeight(splitOuterH), leftPanelFrame.InnerWidth)
	rightContent := buildDashMetricsPanel(rs, st, PanelContentHeight(splitOuterH), rightPanelFrame.InnerWidth)
	split := renderSplitPanels(st, leftPanelFrame, rightPanelFrame, leftContent, rightContent)

	// ── 进度条面板 ──
	progressLine := buildProgressLine(rs, st, bodyPanel.InnerWidth)
	progressPanelStr := bodyPanel.Wrap(st, progressLine)

	// ── 请求列表面板 ──
	reqList := buildRequestList(d, rs, st, bodyPanel.InnerWidth, reqListH)
	reqPanelStr := bodyPanel.Wrap(st, reqList)

	content := joinVerticalBlocks(split, progressPanelStr, reqPanelStr)
	return l.Assemble(frame.Wrap(st, content), st, width)
}

// buildDashParamsPanel 构建左侧任务参数面板。
func buildDashParamsPanel(d *DashboardState, rs *server.RunState, st Styles, maxH, width int) string {
	lines := panelTitleLines(st, "运行进度", width, false)

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
	} else {
		// 参数从 RunState 读取（实际可从 task 传入，此处用 RunState 已知信息展示）
		lines = append(lines, " "+labelValue(st, "进度", fmt.Sprintf("%d/%d", rs.DoneReqs, rs.TotalReqs)))
		lines = append(lines, " "+labelValue(st, "成功", fmt.Sprintf("%d", rs.SuccessReqs)))
		lines = append(lines, " "+labelValue(st, "失败", fmt.Sprintf("%d", rs.FailedReqs)))
	}

	return finishPanelLines(lines, maxH)
}

// buildDashMetricsPanel 构建右侧实时指标面板。
func buildDashMetricsPanel(rs *server.RunState, st Styles, maxH, width int) string {
	lines := panelTitleLines(st, "实时指标", width, false)

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
	}

	return finishPanelLines(lines, maxH)
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
		// 压缩 suffix 确保进度行总宽度不超过 width，防止 lipgloss 折行
		maxSuffixW := maxInt(0, width-lipgloss.Width(prefix)-barW)
		suffix = truncate(suffix, maxSuffixW)
	}

	filled := int(ratio * float64(barW))
	barRendered := st.Ok.Render(strings.Repeat("█", filled)) +
		st.Muted.Render(strings.Repeat("░", barW-filled))

	return lipgloss.JoinHorizontal(lipgloss.Top, prefix, barRendered, suffix)
}

// buildRequestList 构建请求列表区域。
func buildRequestList(d *DashboardState, rs *server.RunState, st Styles, width, maxH int) string {
	titleLines := panelTitleLines(st, "请求列表", width, true)

	if rs == nil || len(rs.Requests) == 0 {
		msg := "等待请求..."
		if rs != nil && rs.Status != server.RunStatusRunning {
			msg = "无请求详情数据"
		}
		titleLines = append(titleLines, " "+st.Muted.Render(msg))
		return finishPanelLines(titleLines, maxH)
	}

	// ── 预计算每行数据（按展示顺序，最新在前）──
	type reqRow struct {
		success  bool
		errMsg   string
		id       string
		status   string
		total    string
		ttft     string
		cache    string
		ptok     string
		ctok     string
		tps      string
	}
	reqs := rs.Requests
	reqRows := make([]reqRow, len(reqs))
	for pos := 0; pos < len(reqs); pos++ {
		i := requestIndexFromDisplayPos(pos, len(reqs))
		r := reqs[i]
		statusText := "✓"
		if !r.Success {
			statusText = "✗"
		}
		totalText := fmtDuration(r.TotalTime)
		if !r.Success && r.ErrorMessage != "" {
			totalText = r.ErrorMessage
		}
		reqRows[pos] = reqRow{
			success: r.Success,
			errMsg:  r.ErrorMessage,
			id:      fmt.Sprintf("#%d", len(reqs)-pos),
			status:  statusText,
			total:   totalText,
			ttft:    fmtDuration(r.TTFT),
			cache:   fmt.Sprintf("%dtok", r.CachedTokens),
			ptok:    fmt.Sprintf("%dtok", r.PromptTokens),
			ctok:    fmt.Sprintf("%dtok", r.CompletionTokens),
			tps:     fmt.Sprintf("%.1f/s", r.TPS),
		}
	}

	// 将 d.ReqSel（绝对索引）转换为展示位置
	selDisplayPos := requestDisplayPos(d.ReqSel, len(reqs))

	// colWidths: 0 = 弹性列（占用剩余宽度），>0 = 固定总宽
	colWidths := []int{6, 8, 0, 8, 10, 12, 12, 10} // #, 状态, 总耗时=flex, TTFT, Cache, 输入, 输出, TPS
	tableH := maxH - len(titleLines)
	tbl := lgtable.New().
		Headers("#", "状态", "总耗时", "TTFT", "Cache", "输入", "输出", "TPS").
		Width(width).
		Height(tableH).
		YOffset(d.ReqOff).
		BorderTop(false).BorderBottom(false).
		BorderLeft(false).BorderRight(false).
		BorderHeader(true).BorderColumn(true).BorderRow(true).
		BorderStyle(lipgloss.NewStyle().Foreground(colorDivider)).
		StyleFunc(func(row, col int) lipgloss.Style {
			aw := func(s lipgloss.Style) lipgloss.Style {
				if col < len(colWidths) && colWidths[col] > 0 {
					return s.Width(colWidths[col]).Padding(0, 1)
				}
				return s.Padding(0, 1)
			}
			if row == lgtable.HeaderRow {
				return aw(st.TableHead)
			}
			if row < 0 || row >= len(reqRows) {
				return aw(st.TableRow)
			}
			r := reqRows[row]
			if row == selDisplayPos {
				return aw(st.TableRowSel)
			}
			switch col {
			case 1: // status
				if r.success {
					return aw(st.Ok)
				}
				return aw(st.ErrStyle)
			case 2: // total
				if !r.success && r.errMsg != "" {
					return aw(st.ErrStyle)
				}
				return aw(st.Value)
			case 3, 4, 5, 6, 7: // ttft, cache, ptok, ctok, tps
				return aw(st.Value)
			default:
				return aw(st.TableRow)
			}
		})

	for _, r := range reqRows {
		tbl.Row(r.id, r.status, r.total, r.ttft, r.cache, r.ptok, r.ctok, r.tps)
	}

	tableStr := tbl.String()
	d.ReqVis = tbl.VisibleRows()
	if d.ReqVis < 1 {
		d.ReqVis = 1
	}
	d.AdjustReqOffset(d.ReqVis, len(reqs))

	tableLines := strings.Split(tableStr, "\n")
	result := append(titleLines, tableLines...)
	return finishPanelLines(result, maxH)
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
