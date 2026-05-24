package pages

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	lgtable "charm.land/lipgloss/v2/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// TaskDetailState 任务详情页状态。
type TaskDetailState struct {
	Task    types.TaskDefinition
	History []types.TaskRunSummary
	BackNav NavAction
	// HistorySel 当前选中的历史记录索引（0 = 最近一次；若有正在运行的实例，0 = 运行中条目）
	HistorySel int
	HistoryOff int
	HistoryVis int
	// ActiveRun 当前正在运行的实例快照（nil = 无），由 model 注入
	ActiveRun *server.RunState
}

// NewTaskDetailState 创建初始任务详情状态。
func NewTaskDetailState(task types.TaskDefinition) *TaskDetailState {
	return &TaskDetailState{Task: task, BackNav: NavAction{To: NavTaskList}}
}

func taskDetailHistoryEntries(s *TaskDetailState) []types.TaskRunSummary {
	if s == nil || len(s.History) == 0 {
		return nil
	}
	if s.ActiveRun == nil {
		return s.History
	}
	if strings.TrimSpace(s.History[0].RunID) == strings.TrimSpace(string(s.ActiveRun.RunID)) {
		return s.History[1:]
	}
	return s.History
}

// HandleTaskDetailKey 处理任务详情页按键。
func HandleTaskDetailKey(s *TaskDetailState, msg tea.KeyMsg, client Client) (*TaskDetailState, tea.Cmd, NavAction) {
	nav := NavAction{}
	hasActive := s.ActiveRun != nil
	historyEntries := taskDetailHistoryEntries(s)
	effectiveLen := len(historyEntries)
	if hasActive {
		effectiveLen++
	}

	switch msg.String() {
	case "up", "k":
		if s.HistorySel > 0 {
			s.HistorySel--
		}

	case "down", "j":
		if s.HistorySel < effectiveLen-1 {
			s.HistorySel++
		}

	case "enter":
		if s.HistorySel >= 0 && s.HistorySel < effectiveLen {
			if hasActive && s.HistorySel == 0 {
				// 进入正在运行的仪表盘，直接导航，避免走 FromHistory 路径
				// （FromHistory 路径会覆盖 dash.BackNav，导致循环：dashboard ↔ taskdetail）
				if s.ActiveRun.Mode == "turbo" {
					nav = NavAction{To: NavTurboDash}
				} else {
					nav = NavAction{To: NavDashboard}
				}
			} else {
				histIdx := s.HistorySel
				if hasActive {
					histIdx--
				}
				if histIdx >= 0 && histIdx < len(historyEntries) {
					runID := strings.TrimSpace(historyEntries[histIdx].RunID)
					if runID != "" {
						sum := historyEntries[histIdx]
						nav = NavAction{To: NavRunDetail, RunID: server.RunID(runID), Summary: &sum}
					}
				}
			}
		}

	case "left", "esc", "b":
		if s.BackNav.To != NavNone {
			nav = s.BackNav
		} else {
			nav = NavAction{To: NavTaskList}
		}

	case "r":
		if s.ActiveRun == nil {
			return s, client.StartRunCmd(s.Task.ID), nav
		}

	case "g":
		if s.HistorySel >= 0 && s.HistorySel < effectiveLen {
			if hasActive && s.HistorySel == 0 {
				break // 正在运行中，无法导出报告
			}
			histIdx := s.HistorySel
			if hasActive {
				histIdx--
			}
			if histIdx >= 0 && histIdx < len(historyEntries) {
				if historyEntries[histIdx].Status == string(server.RunStatusRunning) {
					break
				}
				runID := strings.TrimSpace(historyEntries[histIdx].RunID)
				if runID != "" {
					return s, client.GenerateReportCmd(server.RunID(runID), server.ReportFormatJSON), nav
				}
			}
		}

	case "e":
		t := s.Task
		nav = NavAction{To: NavWizard, EditTask: &t}

	case "y":
		return s, client.CopyTaskCmd(s.Task.ID), nav

	case "d":
		return s, client.DeleteTaskCmd(s.Task.ID), nav

	case "?":
		nav = NavAction{To: NavHelp}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}
	s.HistoryOff = ensureVisibleOffset(s.HistorySel, effectiveLen, s.HistoryOff, s.HistoryVis)
	return s, nil, nav
}

// RenderTaskDetail 渲染任务详情页。
//
// 设计稿布局（全宽单列）：
//
//	╔══ AIT  任务详情 ─ name ══════════════╗
//	║  ◆ AIT   任务 ID: xxx   更新: xxx   刚刚 ║
//	╠══════════════════════════════════════╣
//	║  配置摘要                             ║
//	║  协议  xxx  接口  xxx                 ║
//	║  模型  xxx  模式  xxx  并发 N  请求 N ║
//	║  超时  xxx  流式  开启  Prompt  xxx   ║
//	╠══════════════════════════════════════╣
//	║  最近运行 ▼ 2026-05-16  ✓ 完成  100请求 ║
//	║  ── 指标表格 ──────────────────────── ║
//	╠══════════════════════════════════════╣
//	║  历史运行记录                          ║
//	║  ── 历史列表 ─────────────────────── ║
//	╠══════════════════════════════════════╣
//	║  [r] 生成报告  [c] 复制摘要  ...     ║  ← context bar
//	╠══════════════════════════════════════╣
//	║  [b/Esc] 返回列表  ◆ AIT  v0.1       ║
//	╚══════════════════════════════════════╝
func RenderTaskDetail(s *TaskDetailState, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}
	t := s.Task
	inp := t.Input

	var cbItems []HotkeyItem
	hasActive := s.ActiveRun != nil
	effectiveLen := len(taskDetailHistoryEntries(s))
	if hasActive {
		effectiveLen++
	}
	switch {
	case hasActive:
		cbItems = Hotkeys_TaskDetail_Running()
	case effectiveLen > 0:
		cbItems = Hotkeys_TaskDetail_HasHistory()
	default:
		cbItems = Hotkeys_TaskDetail_NoHistory()
	}
	modeStr := i18n.T(i18n.KStandardMode)
	if inp.Turbo {
		modeStr = i18n.T(i18n.KTurboMode)
	}
	headerRight := []string{i18n.T(i18n.KNoRunRecords)}
	historyCount := len(taskDetailHistoryEntries(s))
	if historyCount > 0 {
		headerRight = []string{fmt.Sprintf("%d", historyCount)}
	}
	if hasActive {
		headerRight = append([]string{i18n.T(i18n.KRunning)}, headerRight...)
	}
	l := PageLayout{
		HeaderTitle:     truncate(t.Name, 28),
		HeaderSubtitle:  i18n.T(i18n.KTaskDetailSubtitle),
		HeaderMeta:      i18n.T(i18n.KRecordDetails),
		HeaderInfoLeft:  []string{modeStr, inp.NormalizedProtocol()},
		HeaderInfoRight: headerRight,
		Hotkeys:         NewPageHotkeysWithHelp(cbItems, i18n.T(i18n.KHintGoBack), i18n.T(i18n.KHintQuit)),
	}
	frame := l.Frame(width, height)

	content := buildTaskDetailContent(s, st, t, inp, frame.InnerWidth, frame.InnerHeight)
	return l.Assemble(frame.Wrap(st, content), st, width)
}

// buildTaskDetailContent 构建任务详情内容区（左右双栏布局）。
// 左栏（40%）：配置摘要  右栏（60%）：历史运行记录
func buildTaskDetailContent(s *TaskDetailState, st Styles, t types.TaskDefinition, inp types.Input, width, maxH int) string {
	bodyPanel := NewPanelFrame(width)
	leftPanelFrame, rightPanelFrame := bodyPanel.Split(40, 28)
	panelContentH := PanelContentHeight(maxH)

	// ─── 左栏：配置摘要 ─────────────────────────────────────────
	leftW := leftPanelFrame.InnerWidth
	leftLines := panelTitleLines(st, i18n.T(i18n.KProtocol), leftW, false)

	proto := inp.NormalizedProtocol()
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KProtocol))+"  "+st.Value.Render(proto), leftW))
	endpoint := truncate(inp.ResolvedEndpointURL(), leftW-8)
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KEndpoint))+"  "+st.Value.Render(endpoint), leftW))
	if inp.ProxyURL != "" {
		proxy := truncate(inp.ProxyURL, leftW-8)
		leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KProxy))+"  "+st.Value.Render(proxy), leftW))
	}
	leftLines = append(leftLines, padRight("", leftW))

	model := truncate(inp.Model, leftW-10)
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KModel))+"  "+st.Value.Render(model), leftW))
	modeStr := i18n.T(i18n.KStandardMode)
	if inp.Turbo {
		modeStr = i18n.T(i18n.KTurboMode)
	}
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KMode))+"  "+st.Value.Render(modeStr), leftW))
	if inp.Turbo {
		tc := inp.TurboConfig
		leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KConcurrency))+"  "+st.Value.Render(
			fmt.Sprintf("%d → %d", tc.InitConcurrency, tc.MaxConcurrency)), leftW))
		leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KStepLabel))+"  "+st.Value.Render(
			fmt.Sprintf("+%d  %d req", tc.StepSize, tc.LevelRequests)), leftW))
	} else {
		leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KConcurrency))+"  "+st.Value.Render(
			fmt.Sprintf("%d", inp.Concurrency)), leftW))
		leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KRequests))+"  "+st.Value.Render(
			fmt.Sprintf("%d", inp.Count)), leftW))
	}
	leftLines = append(leftLines, padRight("", leftW))
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KTimeout))+"  "+st.Value.Render(fmtDuration(inp.Timeout)), leftW))
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KStream))+"  "+st.Value.Render(boolLabel(inp.Stream)), leftW))
	prompt := promptSummary(inp.PromptMode, inp.PromptText, inp.PromptFile, inp.PromptLength)
	leftLines = append(leftLines, padRight(" "+st.Label.Render(i18n.T(i18n.KPromptLabel))+"  "+st.Value.Render(truncate(prompt, leftW-12)), leftW))
	leftContent := finishPanelLines(leftLines, panelContentH)

	// ─── 右栏：历史运行记录 ─────────────────────────────────────
	rightW := rightPanelFrame.InnerWidth
	rightTitle := panelTitleLines(st, i18n.T(i18n.KRunHistory), rightW, false) // 2 行
	historyEntries := taskDetailHistoryEntries(s)

	hasActive := s.ActiveRun != nil
	effectiveLen := len(historyEntries)
	if hasActive {
		effectiveLen++
	}

	if effectiveLen == 0 {
		rightLines := append(rightTitle, padRight(" "+st.Muted.Render(i18n.T(i18n.KNoRunRecords)), rightW))
		rightContent := finishPanelLines(rightLines, panelContentH)
		return renderSplitPanels(st, leftPanelFrame, rightPanelFrame, leftContent, rightContent)
	}

	// ── 计算 detailLines （选中条目详情）──
	var detailLines []string
	{
		histIdx := s.HistorySel
		if hasActive {
			if s.HistorySel == 0 {
				histIdx = -1 // 运行中条目无详情
			} else {
				histIdx--
			}
		}
		if histIdx >= 0 {
			detailLines = buildTaskHistoryDetailLines(historyEntries, histIdx, st, rightW)
		}
	}
	tableMaxH := panelContentH - len(detailLines)
	if tableMaxH < 5 {
		allowedDetail := maxInt(0, panelContentH-5)
		if len(detailLines) > allowedDetail {
			detailLines = detailLines[:allowedDetail]
		}
		tableMaxH = panelContentH - len(detailLines)
	}

	// ── 预计算每行数据──
	type histRow struct {
		statusText  string
		statusIsOk  bool
		statusMut bool
		statusIsMut bool
		time        string
		mode        string
		rate        string
		dur         string
		ttft        string
		tps         string
		rpm         string
		tpm         string
	}
	rowData := make([]histRow, effectiveLen)
	if hasActive {
		rs := s.ActiveRun
		modeShort := modeShortLabel(rs.Mode)
		rateStr := "─"
		if rs.TotalReqs > 0 {
			rateStr = fmt.Sprintf("%.1f%%", rs.SuccessRate)
		}
		rowData[0] = histRow{
			statusText: "●",
			statusIsOk: true,
			time:       rs.StartedAt.Format("2006-01-02 15:04"),
			mode:       modeShort,
			rate:       rateStr,
			dur:        "─",
			ttft:       "─",
			tps:        fmt.Sprintf("%d/%d %s", rs.DoneReqs, rs.TotalReqs, i18n.T(i18n.KRunning)),
			rpm:        "─",
			tpm:        "─",
		}
	}
	for histIdx := 0; histIdx < len(historyEntries); histIdx++ {
		rowIdx := histIdx
		if hasActive {
			rowIdx++
		}
		run := historyEntries[histIdx]
		statusText := "✗"
		statusIsOk := false
		statusMut := false
		switch run.Status {
		case string(server.RunStatusRunning):
			statusText = "●"
			statusIsOk = true
		case string(server.RunStatusCompleted):
			statusText = "✓"
			statusIsOk = true
		case string(server.RunStatusStopped):
			statusText = "■"
			statusMut = true
		}
		modeShort := modeShortLabel(run.Mode)
		durText := "─"
		if !run.FinishedAt.IsZero() {
			durText = fmtDuration(run.FinishedAt.Sub(run.StartedAt))
		}
		rowData[rowIdx] = histRow{
			statusText:  statusText,
			statusIsOk:  statusIsOk,
			statusIsMut: statusMut,
			time:        run.StartedAt.Format("2006-01-02 15:04"),
			mode:        modeShort,
			rate:        fmt.Sprintf("%.1f%%", run.SuccessRate),
			dur:         durText,
			ttft:        fmtDuration(run.AvgTTFT),
			tps:         fmt.Sprintf("%.1f", run.AvgTPS),
			rpm:         fmt.Sprintf("%.0f", run.RPM),
			tpm:         fmt.Sprintf("%.0f", run.TPM),
		}
	}

	// colWidths: 0 = 弹性列，>0 = 固定总宽
	// 动态列宽：取数据最小需求与表头显示宽+2的较大值，确保切换语言后不溢出
	hw := func(s string) int { return lipgloss.Width(s) + 2 }
	h3 := i18n.T(i18n.KMode)
	h4 := i18n.T(i18n.KSuccessRate)
	h5 := i18n.T(i18n.KElapsed)
	colWidths := []int{
		4,                       // 状态图标
		0,                       // 时间=flex
		maxInt(7, hw(h3)),       // 模式
		maxInt(8, hw(h4)),       // 成功率
		maxInt(7, hw(h5)),       // 耗时
		maxInt(7, hw("TTFT")),   // TTFT
		maxInt(7, hw("TPS")),    // TPS
		maxInt(6, hw("RPM")),    // RPM
		maxInt(6, hw("TPM")),    // TPM
	}
	sel := s.HistorySel
	tableH := tableMaxH - len(rightTitle)
	tbl := lgtable.New().
		Headers("", i18n.T(i18n.KTime), h3, h4, h5, "TTFT", "TPS", "RPM", "TPM").
		Width(rightW).
		Height(tableH).
		YOffset(s.HistoryOff).
		BorderTop(false).BorderBottom(false).
		BorderLeft(false).BorderRight(false).
		BorderHeader(true).BorderColumn(true).BorderRow(true).
		BorderStyle(lipgloss.NewStyle().Foreground(colorDivider)).
		StyleFunc(func(row, col int) lipgloss.Style {
			aw := func(s lipgloss.Style) lipgloss.Style { return applyColWidth(s, col, colWidths) }
			if row == lgtable.HeaderRow {
				return aw(st.TableHead)
			}
			if row < 0 || row >= len(rowData) {
				return aw(st.TableRow)
			}
			r := rowData[row]
			if row == sel {
				return aw(st.TableRowSel)
			}
			if col == 0 { // status icon
				if r.statusIsOk {
					return aw(st.Ok)
				}
				if r.statusIsMut {
					return aw(st.Muted)
				}
				return aw(st.ErrStyle)
			}
			if col >= 3 { // rate, dur, ttft, tps, rpm, tpm
				return aw(st.Value)
			}
			return aw(st.TableRow)
		})

	for _, r := range rowData {
		tbl.Row(r.statusText, r.time, r.mode, r.rate, r.dur, r.ttft, r.tps, r.rpm, r.tpm)
	}

	tableStr := tbl.String()
	s.HistoryVis = tbl.VisibleRows()
	if s.HistoryVis < 1 {
		s.HistoryVis = 1
	}
	s.HistoryOff = ensureVisibleOffset(s.HistorySel, effectiveLen, s.HistoryOff, s.HistoryVis)

	tableLines := strings.Split(tableStr, "\n")
	rightLines := append(rightTitle, tableLines...)
	rightLines = append(rightLines, detailLines...)
	rightContent := finishPanelLines(rightLines, panelContentH)
	return renderSplitPanels(st, leftPanelFrame, rightPanelFrame, leftContent, rightContent)
}

// buildMetricRow 构建指标表格一行。
func buildMetricRow(st Styles, name, minV, avgV, stdV, maxV string) string {
	return "  " + st.Label.Render(padRight(name, 16)) +
		st.Value.Render(padRight(minV, 10)) +
		st.MetricVal.Render(padRight(avgV, 10)) +
		st.Muted.Render(padRight(stdV, 10)) +
		st.Value.Render(maxV)
}

// TaskDetailFromMsg 从消息中提取 TaskDetailState 的帮助函数，
// 供 model.go 在 HistoryLoadedMsg 处理时使用。
func UpdateTaskDetailHistory(s *TaskDetailState, history []types.TaskRunSummary, autoExpand bool) *TaskDetailState {
	if s == nil {
		return s
	}
	s.History = history
	effectiveLen := len(taskDetailHistoryEntries(s))
	if s.ActiveRun != nil {
		effectiveLen++
	}
	if effectiveLen == 0 {
		s.HistorySel = 0
		s.HistoryOff = 0
	} else {
		if s.HistorySel < 0 {
			s.HistorySel = 0
		}
		if s.HistorySel >= effectiveLen {
			s.HistorySel = effectiveLen - 1
		}
		s.HistoryOff = ensureVisibleOffset(s.HistorySel, effectiveLen, s.HistoryOff, s.HistoryVis)
	}
	if autoExpand && len(taskDetailHistoryEntries(s)) > 0 {
		// autoExpand 参数保留接口兼容性，展开行为已由渲染层自动处理
		_ = autoExpand
	}
	return s
}

func buildTaskHistoryDetailLines(history []types.TaskRunSummary, histIdx int, st Styles, width int) []string {
	if histIdx < 0 || histIdx >= len(history) {
		return nil
	}
	sel := history[histIdx]
	elapsed := sel.FinishedAt.Sub(sel.StartedAt)
	elapsedText := fmtDuration(elapsed)
	finishedText := sel.FinishedAt.Format("2006-01-02 15:04")
	if sel.FinishedAt.IsZero() {
		elapsedText = fmtDuration(time.Since(sel.StartedAt))
		finishedText = i18n.T(i18n.KRunning)
	}
	labelW := maxLabelWidth([]string{
		i18n.T(i18n.KStatus), i18n.T(i18n.KMode), i18n.T(i18n.KStart), i18n.T(i18n.KEnd),
		i18n.T(i18n.KElapsed), i18n.T(i18n.KSuccessRate), "TTFT", "TPS", "RPM", "TPM",
		i18n.T(i18n.KProtocol), i18n.T(i18n.KModel), i18n.T(i18n.KCache), i18n.T(i18n.KErrorSummary),
	})
	indent := " "
	gap := 4
	contentW := maxInt(12, width-lipgloss.Width(indent))
	useTwoCols := contentW >= 48

	statusText := runStatusText(sel.Status)
	statusStyle := st.Value
	switch sel.Status {
	case "running":
		statusStyle = st.Ok
	case "completed":
		statusStyle = st.Ok
	case "failed":
		statusStyle = st.ErrStyle
	case "stopped":
		statusStyle = st.Muted
	}

	modeText := modeShortLabel(sel.Mode)

	renderCell := func(label, value string, valueStyle lipgloss.Style, cellW int) string {
		prefix := st.Label.Render(padRight(label, labelW))
		available := maxInt(6, cellW-labelW-2)
		return prefix + "  " + valueStyle.Render(truncate(value, available))
	}

	appendSingleField := func(lines []string, label, value string, valueStyle lipgloss.Style) []string {
		valueW := maxInt(10, contentW-labelW-2)
		segments := wrapText(value, valueW)
		if len(segments) == 0 {
			segments = []string{""}
		}
		lines = append(lines, indent+st.Label.Render(padRight(label, labelW))+"  "+valueStyle.Render(segments[0]))
		contIndent := strings.Repeat(" ", lipgloss.Width(indent)+labelW+2)
		for _, seg := range segments[1:] {
			lines = append(lines, contIndent+valueStyle.Render(seg))
		}
		return lines
	}

	appendPairRow := func(lines []string, leftLabel, leftValue string, leftStyle lipgloss.Style, rightLabel, rightValue string, rightStyle lipgloss.Style) []string {
		if !useTwoCols {
			lines = appendSingleField(lines, leftLabel, leftValue, leftStyle)
			return appendSingleField(lines, rightLabel, rightValue, rightStyle)
		}
		leftW := (contentW - gap) / 2
		rightW := contentW - gap - leftW
		row := indent + padRight(renderCell(leftLabel, leftValue, leftStyle, leftW), leftW) + strings.Repeat(" ", gap) +
			renderCell(rightLabel, rightValue, rightStyle, rightW)
		return append(lines, row)
	}

	lines := []string{
		padRight(st.Divider.Render(strings.Repeat("─", width)), width),
		padRight(" "+st.SectionHead.Render(i18n.T(i18n.KRecordDetails)), width),
	}

	lines = appendPairRow(lines,
		i18n.T(i18n.KStatus), statusText, statusStyle,
		i18n.T(i18n.KMode), modeText, st.Value,
	)
	lines = appendPairRow(lines,
		i18n.T(i18n.KStart), sel.StartedAt.Format("2006-01-02 15:04"), st.Value,
		i18n.T(i18n.KEnd), finishedText, st.Value,
	)
	lines = appendPairRow(lines,
		i18n.T(i18n.KElapsed), elapsedText, st.Value,
		i18n.T(i18n.KSuccessRate), fmt.Sprintf("%.1f%%", sel.SuccessRate), st.Value,
	)
	lines = appendPairRow(lines,
		"TTFT", fmtDuration(sel.AvgTTFT), st.Value,
		"TPS", fmt.Sprintf("%.1f", sel.AvgTPS), st.MetricVal,
	)
	lines = appendPairRow(lines,
		"RPM", fmt.Sprintf("%.0f req/min", sel.RPM), st.MetricVal,
		"TPM", fmt.Sprintf("%.0f tok/min", sel.TPM), st.MetricVal,
	)
	lines = appendSingleField(lines, i18n.T(i18n.KProtocol), shortProtocol(sel.Protocol), st.Value)
	lines = appendSingleField(lines, i18n.T(i18n.KModel), sel.Model, st.Value)
	if sel.CacheHitRate > 0 {
		lines = appendSingleField(lines, i18n.T(i18n.KCache), fmt.Sprintf("%.1f%%", sel.CacheHitRate*100), st.Value)
	}
	if sel.ErrorSummary != "" {
		lines = append(lines, indent+st.Label.Render(i18n.T(i18n.KErrorSummary)))
		for _, seg := range wrapText(sel.ErrorSummary, maxInt(10, contentW-2)) {
			lines = append(lines, indent+"  "+st.ErrStyle.Render(seg))
		}
	}

	return lines
}
