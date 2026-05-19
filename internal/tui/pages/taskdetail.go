package pages

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// TaskDetailState 任务详情页状态。
type TaskDetailState struct {
	Task    types.TaskDefinition
	History []types.TaskRunSummary
	// HistorySel 当前选中的历史记录索引（0 = 最近一次；若有正在运行的实例，0 = 运行中条目）
	HistorySel int
	HistoryOff int
	HistoryVis int
	// ActiveRun 当前正在运行的实例快照（nil = 无），由 model 注入
	ActiveRun *server.RunState
}

// NewTaskDetailState 创建初始任务详情状态。
func NewTaskDetailState(task types.TaskDefinition) *TaskDetailState {
	return &TaskDetailState{Task: task}
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
				// 进入正在运行的仪表盘
				nav = NavAction{To: NavRunDetail, RunID: s.ActiveRun.RunID}
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
		nav = NavAction{To: NavTaskList}

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

	var cbItems []ContextBarItem
	hasActive := s.ActiveRun != nil
	effectiveLen := len(taskDetailHistoryEntries(s))
	if hasActive {
		effectiveLen++
	}
	switch {
	case hasActive:
		cbItems = CtxBar_TaskDetail_Running()
	case effectiveLen > 0:
		cbItems = CtxBar_TaskDetail_HasHistory()
	default:
		cbItems = CtxBar_TaskDetail_NoHistory()
	}
	l := PageLayout{
		CtxItems:    cbItems,
		FooterParts: []string{"[b/Esc] 返回列表", "◆ AIT  v0.1"},
	}

	content := buildTaskDetailContent(s, st, t, inp, ContentWidth(width), l.ContentHeight(height))
	return l.Assemble(wrapPanel(st, content, width), st, width)
}

// buildTaskDetailContent 构建任务详情内容区（左右双栏布局）。
// 左栏（40%）：配置摘要  右栏（60%）：历史运行记录
func buildTaskDetailContent(s *TaskDetailState, st Styles, t types.TaskDefinition, inp types.Input, width, maxH int) string {
	leftW := width * 4 / 10
	if leftW < 26 {
		leftW = 26
	}
	rightW := width - leftW - 1 // 1 列用于 │ 分隔符

	// ─── 左栏：配置摘要 ─────────────────────────────────────────
	var leftLines []string
	leftLines = append(leftLines, padRight(" "+st.SectionHead.Render("配置摘要"), leftW))
	leftLines = append(leftLines, padRight(st.Divider.Render(strings.Repeat("─", leftW)), leftW))
	leftLines = append(leftLines, padRight("", leftW))

	proto := inp.NormalizedProtocol()
	leftLines = append(leftLines, padRight(" "+st.Label.Render("协议")+"  "+st.Value.Render(proto), leftW))
	endpoint := truncate(inp.ResolvedEndpointURL(), leftW-8)
	leftLines = append(leftLines, padRight(" "+st.Label.Render("接口")+"  "+st.Value.Render(endpoint), leftW))
	leftLines = append(leftLines, padRight("", leftW))

	model := truncate(inp.Model, leftW-10)
	leftLines = append(leftLines, padRight(" "+st.Label.Render("模型")+"  "+st.Value.Render(model), leftW))
	modeStr := "标准模式"
	if inp.Turbo {
		modeStr = "Turbo 模式"
	}
	leftLines = append(leftLines, padRight(" "+st.Label.Render("模式")+"  "+st.Value.Render(modeStr), leftW))
	if inp.Turbo {
		tc := inp.TurboConfig
		leftLines = append(leftLines, padRight(" "+st.Label.Render("并发")+"  "+st.Value.Render(
			fmt.Sprintf("%d → %d", tc.InitConcurrency, tc.MaxConcurrency)), leftW))
		leftLines = append(leftLines, padRight(" "+st.Label.Render("步进")+"  "+st.Value.Render(
			fmt.Sprintf("+%d  每级%d请求", tc.StepSize, tc.LevelRequests)), leftW))
	} else {
		leftLines = append(leftLines, padRight(" "+st.Label.Render("并发")+"  "+st.Value.Render(
			fmt.Sprintf("%d", inp.Concurrency)), leftW))
		leftLines = append(leftLines, padRight(" "+st.Label.Render("请求")+"  "+st.Value.Render(
			fmt.Sprintf("%d", inp.Count)), leftW))
	}
	leftLines = append(leftLines, padRight("", leftW))
	leftLines = append(leftLines, padRight(" "+st.Label.Render("超时")+"  "+st.Value.Render(fmtDuration(inp.Timeout)), leftW))
	leftLines = append(leftLines, padRight(" "+st.Label.Render("流式")+"  "+st.Value.Render(boolLabel(inp.Stream)), leftW))
	prompt := promptSummary(inp.PromptMode, inp.PromptText, inp.PromptFile, inp.PromptLength)
	leftLines = append(leftLines, padRight(" "+st.Label.Render("Prompt")+"  "+st.Value.Render(truncate(prompt, leftW-12)), leftW))

	// ─── 右栏：历史运行记录 ─────────────────────────────────────
	var rightLines []string
	historyEntries := taskDetailHistoryEntries(s)
	rightLines = append(rightLines, padRight(" "+st.SectionHead.Render("历史运行记录"), rightW))
	rightLines = append(rightLines, padRight(st.Divider.Render(strings.Repeat("─", rightW)), rightW))

	hasActive := s.ActiveRun != nil
	effectiveLen := len(historyEntries)
	if hasActive {
		effectiveLen++
	}

	const (
		markW = 2
		statW = 2
		timeW = 17
		modeW = 7
		rateW = 8
		ttftW = 10
	)
	hdr := padRight("", markW) + padRight("", statW) + padRight("时间", timeW) + padRight("模式", modeW) +
		padRight("成功率", rateW) + padRight("TTFT", ttftW) + "TPS"
	rightLines = append(rightLines, padRight(renderTableHeader(st, rightW, hdr), rightW))
	rightLines = append(rightLines, padRight(st.Divider.Render(strings.Repeat("─", rightW)), rightW))

	if effectiveLen == 0 {
		rightLines = append(rightLines, padRight(" "+st.Muted.Render("暂无运行记录"), rightW))
	} else {
		// 始终为当前选中的历史条目显示详情面板
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
		tableMaxH := maxH - len(detailLines)
		if tableMaxH < 5 {
			allowedDetail := maxInt(0, maxH-5)
			if len(detailLines) > allowedDetail {
				detailLines = detailLines[:allowedDetail]
			}
			tableMaxH = maxH - len(detailLines)
		}
		s.HistoryVis = listVisibleItems(tableMaxH, 4)
		s.HistoryOff = ensureVisibleOffset(s.HistorySel, effectiveLen, s.HistoryOff, s.HistoryVis)
		start := s.HistoryOff
		end := minInt(effectiveLen, start+s.HistoryVis)

		// ── 历史列表 ──
		for idx := start; idx < end; idx++ {
			isSel := idx == s.HistorySel
			marker := selectionMarker(isSel)
			var row string

			if hasActive && idx == 0 {
				// 正在运行中的条目
				rs := s.ActiveRun
				modeShort := "标准"
				if rs.Mode == "turbo" {
					modeShort = "Turbo"
				}
				statusIcon := styleWhenNotSelected(isSel, st.Ok, "●")
				rateStr := "─"
				if rs.TotalReqs > 0 {
					rateStr = fmt.Sprintf("%.0f%%", rs.SuccessRate)
				}
				progStr := fmt.Sprintf("%d/%d 正在运行...", rs.DoneReqs, rs.TotalReqs)
				row = padRight(marker, markW) +
					padRight(statusIcon, statW) +
					padRight(rs.StartedAt.Format("2006-01-02 15:04"), timeW) +
					padRight(modeShort, modeW) +
					padRight(rateStr, rateW) +
					styleWhenNotSelected(isSel, st.Ok, progStr)
			} else {
				histIdx := idx
				if hasActive {
					histIdx--
				}
				run := historyEntries[histIdx]
				statusText := "✗"
				statusStyle := st.ErrStyle
				switch run.Status {
				case string(server.RunStatusRunning):
					statusText = "●"
					statusStyle = st.Ok
				case string(server.RunStatusCompleted):
					statusText = "✓"
					statusStyle = st.Ok
				case string(server.RunStatusStopped):
					statusText = "■"
					statusStyle = st.Muted
				}
				modeShort := "标准"
				if run.Mode == "turbo" {
					modeShort = "Turbo"
				}
				statusIcon := styleWhenNotSelected(isSel, statusStyle, statusText)
				row = padRight(marker, markW) +
					padRight(statusIcon, statW) +
					padRight(run.StartedAt.Format("2006-01-02 15:04"), timeW) +
					padRight(modeShort, modeW) +
					padRight(fmt.Sprintf("%.1f%%", run.SuccessRate), rateW) +
					padRight(fmtDuration(run.AvgTTFT), ttftW) +
					fmt.Sprintf("%.1f", run.AvgTPS)
			}
			rightLines = append(rightLines, padRight(renderTableRow(st, rightW, isSel, row), rightW))
			if idx < end-1 {
				rightLines = append(rightLines, padRight(st.Divider.Render(strings.Repeat("─", rightW)), rightW))
			}
		}
		rightLines = append(rightLines, detailLines...)
	}

	// ─── 合并双栏 ──────────────────────────────────────────────
	for len(leftLines) < maxH {
		leftLines = append(leftLines, padRight("", leftW))
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, padRight("", rightW))
	}
	sep := st.Divider.Render("│")
	var combined []string
	for i := 0; i < maxH; i++ {
		combined = append(combined, leftLines[i]+sep+rightLines[i])
	}
	return strings.Join(combined, "\n")
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
		finishedText = "进行中"
	}
	labelW := 8
	indent := " "
	gap := 4
	contentW := maxInt(12, width-lipgloss.Width(indent))
	useTwoCols := contentW >= 48

	statusText := sel.Status
	statusStyle := st.Value
	switch sel.Status {
	case "running":
		statusText = "运行中"
		statusStyle = st.Ok
	case "completed":
		statusText = "完成"
		statusStyle = st.Ok
	case "failed":
		statusText = "失败"
		statusStyle = st.ErrStyle
	case "stopped":
		statusText = "已停止"
		statusStyle = st.Muted
	}

	modeText := "标准"
	if sel.Mode == "turbo" {
		modeText = "Turbo"
	}

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
		padRight(" "+st.SectionHead.Render("记录详情"), width),
	}

	lines = appendPairRow(lines,
		"状态", statusText, statusStyle,
		"模式", modeText, st.Value,
	)
	lines = appendPairRow(lines,
		"开始", sel.StartedAt.Format("2006-01-02 15:04"), st.Value,
		"结束", finishedText, st.Value,
	)
	lines = appendPairRow(lines,
		"耗时", elapsedText, st.Value,
		"成功率", fmt.Sprintf("%.1f%%", sel.SuccessRate), st.Value,
	)
	lines = appendPairRow(lines,
		"TTFT", fmtDuration(sel.AvgTTFT), st.Value,
		"TPS", fmt.Sprintf("%.1f", sel.AvgTPS), st.MetricVal,
	)
	lines = appendSingleField(lines, "协议", shortProtocol(sel.Protocol), st.Value)
	lines = appendSingleField(lines, "模型", sel.Model, st.Value)
	if sel.CacheHitRate > 0 {
		lines = appendSingleField(lines, "缓存", fmt.Sprintf("%.1f%%", sel.CacheHitRate*100), st.Value)
	}
	if sel.ErrorSummary != "" {
		lines = append(lines, indent+st.Label.Render("错误摘要"))
		for _, seg := range wrapText(sel.ErrorSummary, maxInt(10, contentW-2)) {
			lines = append(lines, indent+"  "+st.ErrStyle.Render(seg))
		}
	}

	return lines
}
