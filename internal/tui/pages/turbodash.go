package pages

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	lgtable "charm.land/lipgloss/v2/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// TurboDashState Turbo 模式仪表盘页状态。
type TurboDashState struct {
	RunID    server.RunID
	TaskID   string
	EventCh  <-chan server.Event
	CancelFn server.CancelFunc
	RunState *server.RunState
	ReqSel   int       // 选中请求索引（-1 = 无选中）
	ReqOff   int       // 滚动偏移
	ReqVis   int       // 当前可见请求数
	BackNav  NavAction // 按 b/esc 时的返回目标；Zero = 返回任务列表
}

// AdjustReqOffset 根据屏幕显示顺序调整列表可见窗口。
func (d *TurboDashState) AdjustReqOffset(visH, total int) {
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

// NewTurboDashState 创建 Turbo 仪表盘初始状态。
func NewTurboDashState(runID server.RunID, taskID string) *TurboDashState {
	return &TurboDashState{
		RunID:  runID,
		TaskID: taskID,
		ReqSel: -1,
	}
}

// IsRunning 判断是否仍在运行。
func (d *TurboDashState) IsRunning() bool {
	if d == nil {
		return false
	}
	return isRunStateRunning(d.RunState)
}

// HandleTurboDashKey 处理 Turbo 仪表盘按键。
func HandleTurboDashKey(d *TurboDashState, msg tea.KeyMsg, client Client) (*TurboDashState, tea.Cmd, NavAction) {
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
			d.ReqSel = requestIndexFromDisplayPos(len(reqs)-1, len(reqs))
		} else {
			selPos := requestDisplayPos(d.ReqSel, len(reqs))
			if selPos > 0 {
				selPos--
			} else {
				selPos = len(reqs) - 1
			}
			d.ReqSel = requestIndexFromDisplayPos(selPos, len(reqs))
		}

	case "down", "j":
		if len(reqs) == 0 {
			break
		}
		if d.ReqSel < 0 {
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
			return d, client.GenerateRunReportCmd(d.RunID, server.ReportFormatJSON), nav
		}

	case "?":
		nav = NavAction{To: NavHelp}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}
	d.AdjustReqOffset(d.ReqVis, len(reqs))

	return d, nil, nav
}

// RenderTurboDash 渲染 Turbo 模式运行仪表盘。
//
// 设计稿布局：
//
//	╔══ AIT  Turbo 探测 ─ task-name ══════════╗
//	║  ◆ AIT   model · protocol · 1→50  步进+2  ║
//	╠══════════════════╦══════════════════════╣
//	║  任务参数           ║  当前级别实时指标 [并发=N] ║
//	║  ...               ║  ...                    ║
//	╠══════════════════╩══════════════════════╣
//	║  进度  ████░░  N/30  当前并发 N   总进度: 已完成N/~N级 ║
//	╠═════════════════════════════════════════╣
//	║  级别列表                                ║
//	║  并发  成功率  TPS  TTFT  Cache  总耗时  结论 ║
//	║  ...                                    ║
//	╠═════════════════════════════════════════╣
//	║  context bar                             ║
//	╠═════════════════════════════════════════╣
//	║  [s] 停止  [b] 后台  [m] 标记极限  [q] 退出 ║
//	╚═════════════════════════════════════════╝
func RenderTurboDash(d *TurboDashState, taskName string, st Styles, width, height int) string {
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
		cbItems = Hotkeys_TurboDash_Running_Sel()
	case hasSel && !isRunning:
		cbItems = Hotkeys_TurboDash_Done_Sel()
	case !hasSel && isRunning:
		cbItems = Hotkeys_TurboDash_Running_NoSel()
	default:
		cbItems = Hotkeys_TurboDash_Done_NoSel()
	}
	headerLeft := []string{i18n.T(i18n.KWaitingStatus)}
	headerRight := []string{}
	if rs != nil {
		headerLeft = []string{runStatusText(string(rs.Status)), fmt.Sprintf("%d/%d", rs.DoneReqs, rs.TotalReqs)}
		var levelNum int
		if d.IsRunning() {
			levelNum = len(rs.Levels) + 1
		} else {
			levelNum = len(rs.Levels)
		}
		if levelNum < 1 {
			levelNum = 1
		}
		headerRight = []string{fmt.Sprintf("%s %d", i18n.T(i18n.KColLevel), levelNum)}
		if len(rs.Levels) > 0 {
			headerRight = append(headerRight, fmt.Sprintf("%d", len(rs.Levels)))
		}
	}
	if d.TaskID != "" {
		headerRight = append(headerRight, truncate(d.TaskID, 14))
	}
	l := PageLayout{
		HeaderTitle:     i18n.T(i18n.KTurboMonitor),
		HeaderSubtitle:  i18n.T(i18n.KTurboSubtitle),
		HeaderMeta:      i18n.T(i18n.KTurboModeMeta),
		HeaderInfoLeft:  headerLeft,
		HeaderInfoRight: headerRight,
		Hotkeys:         NewPageHotkeysWithHelp(cbItems, i18n.T(i18n.KHintGoBack), i18n.T(i18n.KHintQuit)),
	}
	frame := l.Frame(width, height)
	bodyPanel := frame.InnerPanel()

	// ── 计算高度 ──
	splitOuterH := 9
	progressOuterH := 3
	levelOuterH := RemainingStackOuterHeight(frame.InnerHeight, splitOuterH, progressOuterH)
	levelListH := PanelContentHeight(levelOuterH)

	// ── 双栏面板（任务参数 | 当前级别指标）──
	leftPanelFrame, rightPanelFrame := bodyPanel.Split(45, 24)
	leftContent := buildTurboDashParams(rs, st, PanelContentHeight(splitOuterH), leftPanelFrame.InnerWidth)
	rightContent := buildTurboDashMetrics(rs, st, PanelContentHeight(splitOuterH), rightPanelFrame.InnerWidth)
	split := renderSplitPanels(st, leftPanelFrame, rightPanelFrame, leftContent, rightContent)

	// ── 进度条面板 ──
	progressLine := buildTurboProgressLine(rs, st, bodyPanel.InnerWidth)
	progressPanelStr := bodyPanel.Wrap(st, progressLine)

	// ── 请求列表面板 ──
	requestList := buildTurboRequestList(d, rs, st, bodyPanel.InnerWidth, levelListH)
	levelPanelStr := bodyPanel.Wrap(st, requestList)

	content := joinVerticalBlocks(split, progressPanelStr, levelPanelStr)
	return l.Assemble(frame.Wrap(st, content), st, width)
}

// buildTurboDashParams 构建 Turbo 仪表盘左侧任务参数面板。
func buildTurboDashParams(rs *server.RunState, st Styles, maxH, width int) string {
	lines := panelTitleLines(st, i18n.T(i18n.KConcurrency), width, false)

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render(i18n.T(i18n.KWaitingData)))
	} else {
		tc := rs.TurboConfig
		lbls := []string{i18n.T(i18n.KRamp), i18n.T(i18n.KPerLevel), i18n.T(i18n.KStopCondLabel)}
		lw := maxLabelWidth(lbls)
		lines = append(lines, " "+labelValue(st, lbls[0], fmt.Sprintf("%d→%d  +%d", tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize), lw))
		lines = append(lines, " "+labelValue(st, lbls[1], fmt.Sprintf("%d req", tc.LevelRequests), lw))
		lines = append(lines, " "+labelValue(st, lbls[2], fmt.Sprintf("%.0f%%", tc.MinSuccessRate*100), lw))
	}

	return finishPanelLines(lines, maxH)
}

// buildTurboDashMetrics 构建 Turbo 仪表盘右侧当前级别实时指标面板。
func buildTurboDashMetrics(rs *server.RunState, st Styles, maxH, width int) string {
	var lines []string

	curLevel := 0
	if rs != nil {
		curLevel = rs.CurrentLevel
	}
	lines = panelTitleLines(st, fmt.Sprintf(i18n.T(i18n.KTurboCurLevelFmt), curLevel), width, false)

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render(i18n.T(i18n.KWaitingData)))
	} else {
		lines = appendRunMetricLines(lines, st, rs)
	}

	return finishPanelLines(lines, maxH)
}

// buildTurboProgressLine 构建 Turbo 模式进度条行。
func buildTurboProgressLine(rs *server.RunState, st Styles, width int) string {
	if rs == nil {
		return " " + padToDisplayWidth(i18n.T(i18n.KProgress), 4) + "  " + st.Muted.Render(i18n.T(i18n.KWaitingDots))
	}
	total := rs.TotalReqs
	done := rs.DoneReqs
	var ratio float64
	if total > 0 {
		ratio = float64(done) / float64(total)
	}
	levelDone := len(rs.Levels)
	var levelTotalStr string
	cfg := rs.TurboConfig
	if cfg.StepSize > 0 {
		expected := (cfg.MaxConcurrency-cfg.InitConcurrency)/cfg.StepSize + 1
		levelTotalStr = fmt.Sprintf("%d/%d", levelDone, expected)
	} else {
		levelTotalStr = fmt.Sprintf("%d", levelDone)
	}
	suffix := fmt.Sprintf(i18n.T(i18n.KTurboDashSuffix), done, total, rs.CurrentLevel, levelTotalStr)
	return renderProgressBar(st, " "+padToDisplayWidth(i18n.T(i18n.KProgress), 4)+"  ", suffix, ratio, width)
}

// buildTurboRequestList 构建 Turbo 模式请求列表区域。
func buildTurboRequestList(d *TurboDashState, rs *server.RunState, st Styles, width, maxH int) string {
	titleLines := panelTitleLines(st, i18n.T(i18n.KRequests), width, true)

	if rs == nil || len(rs.Requests) == 0 {
		msg := i18n.T(i18n.KWaitingData)
		if rs != nil && rs.Status != server.RunStatusRunning {
			msg = i18n.T(i18n.KNoRunRecords)
		}
		titleLines = append(titleLines, " "+st.Muted.Render(msg))
		return finishPanelLines(titleLines, maxH)
	}

	// ── 预计算每行数据（按展示顺序，最新在前）──
	type reqRow struct {
		success bool
		errMsg  string
		id      string
		status  string
		level   string
		total   string
		ttft    string
		cache   string
		ptok    string
		ctok    string
		tps     string
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
			level:   fmt.Sprintf("%d", r.Level),
			total:   totalText,
			ttft:    fmtDuration(r.TTFT),
			cache:   fmt.Sprintf("%dtok", r.CachedTokens),
			ptok:    fmt.Sprintf("%dtok", r.PromptTokens),
			ctok:    fmt.Sprintf("%dtok", r.CompletionTokens),
			tps:     fmt.Sprintf("%.1f/s", r.TPS),
		}
	}

	selDisplayPos := requestDisplayPos(d.ReqSel, len(reqs))

	// colWidths: 0 = 弹性列（占用剩余宽度），>0 = 固定总宽
	colWidths := []int{6, 5, 6, 0, 8, 10, 12, 12, 10} // #, 状态, 级别, 总耗时=flex, TTFT, Cache, 输入, 输出, TPS
	tableH := maxH - len(titleLines)
	tbl := lgtable.New().
		Headers("#", i18n.T(i18n.KStatus), i18n.T(i18n.KColLevel), i18n.T(i18n.KTotalTime), "TTFT", "Cache", i18n.T(i18n.KColInput), i18n.T(i18n.KColOutput), "TPS").
		Width(width).
		Height(tableH).
		YOffset(d.ReqOff).
		BorderTop(false).BorderBottom(false).
		BorderLeft(false).BorderRight(false).
		BorderHeader(true).BorderColumn(true).BorderRow(true).
		BorderStyle(lipgloss.NewStyle().Foreground(colorDivider)).
		StyleFunc(func(row, col int) lipgloss.Style {
			aw := func(s lipgloss.Style) lipgloss.Style { return applyColWidth(s, col, colWidths) }
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
			case 3: // total
				if !r.success && r.errMsg != "" {
					return aw(st.ErrStyle)
				}
				return aw(st.Value)
			case 4, 5, 6, 7, 8: // ttft, cache, ptok, ctok, tps
				return aw(st.Value)
			default:
				return aw(st.TableRow)
			}
		})

	for _, r := range reqRows {
		tbl.Row(r.id, r.status, r.level, r.total, r.ttft, r.cache, r.ptok, r.ctok, r.tps)
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
