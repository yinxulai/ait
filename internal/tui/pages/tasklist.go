package pages

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"charm.land/lipgloss/v2"
	lgtable "charm.land/lipgloss/v2/table"
	"github.com/yinxulai/ait/internal/i18n"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// TaskListState 任务列表页状态。
type TaskListState struct {
	Tasks    []types.TaskOverview
	Selected int
	Offset   int
	Visible  int
	// 运行中任务的进度（runID -> RunState 快照，由 Model 注入）
	ActiveRuns map[string]*server.RunState // taskID -> RunState
	// 删除二次确认
	ConfirmDelete bool
}

// NewTaskListState 创建初始任务列表状态。
func NewTaskListState() *TaskListState {
	return &TaskListState{
		ActiveRuns: make(map[string]*server.RunState),
	}
}

// CurrentTask 返回当前选中的任务。
func (s *TaskListState) CurrentTask() (types.TaskDefinition, bool) {
	if len(s.Tasks) == 0 || s.Selected < 0 || s.Selected >= len(s.Tasks) {
		return types.TaskDefinition{}, false
	}
	return s.Tasks[s.Selected].TaskDefinition, true
}

// IsTaskRunning 判断某任务是否正在运行。
func (s *TaskListState) IsTaskRunning(taskID string) bool {
	if rs, ok := s.ActiveRuns[taskID]; ok {
		return rs != nil && rs.Status == server.RunStatusRunning
	}
	return false
}

// latestRunAt 返回任务列表中最新一次运行时间（用于 header 显示）。
func (s *TaskListState) latestRunAt() *time.Time {
	var latest *time.Time
	for _, t := range s.Tasks {
		if t.LatestRun != nil && !t.LatestRun.FinishedAt.IsZero() {
			finishedAt := t.LatestRun.FinishedAt
			if latest == nil || finishedAt.After(*latest) {
				latest = &finishedAt
			}
		}
	}
	return latest
}

// HandleTaskListKey 处理任务列表页的按键，返回 tea.Cmd 和导航意图。
func HandleTaskListKey(s *TaskListState, msg tea.KeyMsg, client Client) (*TaskListState, tea.Cmd, NavAction) {
	nav := NavAction{}

	// 删除确认模式：拦截所有按键，只处理确认/取消
	if s.ConfirmDelete {
		switch msg.String() {
		case "y", "enter":
			s.ConfirmDelete = false
			if t, ok := s.CurrentTask(); ok {
				return s, client.DeleteTaskCmd(t.ID), nav
			}
		case "n", "esc", "q":
			s.ConfirmDelete = false
		case "ctrl+c":
			nav = NavAction{To: NavQuit}
		}
		return s, nil, nav
	}

	switch msg.String() {
	case "up", "k":
		if s.Selected > 0 {
			s.Selected--
		}
	case "down", "j":
		if s.Selected < len(s.Tasks)-1 {
			s.Selected++
		}

	case "a":
		nav = NavAction{To: NavWizard, EditTask: nil}

	case "p":
		nav = NavAction{To: NavProxy}

	case "e":
		if t, ok := s.CurrentTask(); ok {
			nav = NavAction{To: NavWizard, EditTask: &t}
		}

	case "y":
		if t, ok := s.CurrentTask(); ok {
			return s, client.CopyTaskCmd(t.ID), nav
		}

	case "d":
		if t, ok := s.CurrentTask(); ok && !s.IsTaskRunning(t.ID) {
			s.ConfirmDelete = true
		}

	case "enter":
		if t, ok := s.CurrentTask(); ok {
			nav = NavAction{To: NavTaskDetail, TaskID: t.ID}
		}

	case "r":
		if t, ok := s.CurrentTask(); ok && !s.IsTaskRunning(t.ID) {
			return s, client.StartRunCmd(t.ID), nav
		}

	case "s":
		if t, ok := s.CurrentTask(); ok {
			if rs, ok := s.ActiveRuns[t.ID]; ok && rs != nil {
				return s, client.StopRunCmd(rs.RunID), nav
			}
		}

	case "?":
		nav = NavAction{To: NavHelp}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}

	s.Offset = ensureVisibleOffset(s.Selected, len(s.Tasks), s.Offset, s.Visible)

	return s, nil, nav
}

// RenderTaskList 渲染任务列表页。

//	╚══════════════════════════════╝
func RenderTaskList(s *TaskListState, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}

	var cbItems []HotkeyItem
	if s.ConfirmDelete {
		cbItems = []HotkeyItem{HotkeyAction("y/Enter", i18n.T(i18n.KConfirmDelete)), HotkeyAction("n/Esc", i18n.T(i18n.KCancel))}
	} else if t, ok := s.CurrentTask(); ok {
		if s.IsTaskRunning(t.ID) {
			cbItems = Hotkeys_TaskList_Running()
		} else {
			cbItems = Hotkeys_TaskList_Normal()
		}
	} else {
		cbItems = []HotkeyItem{HotkeyAction("a", i18n.T(i18n.KNewTask))}
	}
	runningCount := 0
	for _, rs := range s.ActiveRuns {
		if rs != nil && rs.Status == server.RunStatusRunning {
			runningCount++
		}
	}
	headerRight := []string{i18n.T(i18n.KNoRunHistory)}
	if latest := s.latestRunAt(); latest != nil {
		headerRight = []string{fmtRelativeTime(*latest)}
	}
	if t, ok := s.CurrentTask(); ok {
		headerRight = append([]string{truncate(t.Name, 22)}, headerRight...)
	}
	l := PageLayout{
		HeaderTitle:     i18n.T(i18n.KTaskCenter),
		HeaderSubtitle:  i18n.T(i18n.KTaskListSubtitle),
		HeaderMeta:      fmt.Sprintf("%d", len(s.Tasks)),
		HeaderInfoLeft:  []string{fmt.Sprintf("%s %d", i18n.T(i18n.KRunning), runningCount)},
		HeaderInfoRight: headerRight,
		Hotkeys:         NewPageHotkeysWithHelp(cbItems, i18n.T(i18n.KHintSelect), i18n.T(i18n.KHintNew), i18n.T(i18n.KHintQuit)),
	}
	frame := l.Frame(width, height)
	panel := NewPanelFrame(frame.OuterWidth)
	innerW := panel.InnerWidth
	innerH := PanelContentHeight(frame.InnerHeight)

	var content string
	if s.ConfirmDelete {
		content = buildTaskListConfirmContent(s, st, innerW, innerH)
	} else {
		content = buildTaskListContent(s, st, innerW, innerH)
	}
	return l.Assemble(panel.Wrap(st, content), st, width)
}

// buildTaskListContent 构建任务列表内容区（含表头 + 任务条目）。
func buildTaskListContent(s *TaskListState, st Styles, width, maxH int) string {
	// ── 预计算每行数据（供 StyleFunc 闭包引用）──
	type taskRowData struct {
		name        string
		mode        string
		isTurbo     bool
		proto       string
		lastRun     string
		isRunning   bool
		rate        string
		cache       string
		ttft        string
		tps         string
		rpm         string
		tpm         string
	}

	sel := s.Selected
	rowData := make([]taskRowData, len(s.Tasks))
	for i, t := range s.Tasks {
		rs := s.ActiveRuns[t.ID]
		_, hasActiveRun := s.ActiveRuns[t.ID]

		modeText := i18n.T(i18n.KStandardMode)
		isTurbo := false
		if t.Input.Turbo {
			modeText = "Turbo"
			isTurbo = true
		}

		isRunning := hasActiveRun || (t.LatestRun != nil && t.LatestRun.Status == string(server.RunStatusRunning))
		lastRunText := "─"
		if isRunning {
			lastRunText = i18n.T(i18n.KRunning)
		} else if t.LatestRun != nil && !t.LatestRun.FinishedAt.IsZero() {
			lastRunText = fmtRelativeTime(t.LatestRun.FinishedAt)
		}

		rateText := "─"
		if hasActiveRun && rs != nil && rs.TotalReqs > 0 {
			rateText = fmt.Sprintf("%.1f%%", rs.SuccessRate)
		} else if !hasActiveRun && t.LatestRun != nil {
			rateText = fmt.Sprintf("%.1f%%", t.LatestRun.SuccessRate)
		}

		ttftText := "─"
		if hasActiveRun && rs != nil && rs.AvgTTFT > 0 {
			ttftText = fmtDuration(rs.AvgTTFT)
		} else if !hasActiveRun && t.LatestRun != nil {
			ttftText = fmtDuration(t.LatestRun.AvgTTFT)
		}

		tpsText := "─"
		if hasActiveRun && rs != nil && rs.AvgTPS > 0 {
			tpsText = fmt.Sprintf("%.1f", rs.AvgTPS)
		} else if !hasActiveRun && t.LatestRun != nil {
			if t.Input.Turbo && t.LatestRun.MaxStableConcurrency > 0 {
				tpsText = fmt.Sprintf(i18n.T(i18n.KConcFmt), t.LatestRun.MaxStableConcurrency)
			} else if !t.Input.Turbo {
				tpsText = fmt.Sprintf("%.1f", t.LatestRun.AvgTPS)
			}
		}

		cacheText := "─"
		if hasActiveRun && rs != nil && rs.CacheHitRate > 0 {
			cacheText = fmt.Sprintf("%.1f%%", rs.CacheHitRate*100)
		} else if !hasActiveRun && t.LatestRun != nil && t.LatestRun.CacheHitRate > 0 {
			cacheText = fmt.Sprintf("%.1f%%", t.LatestRun.CacheHitRate*100)
		}

		rpmText := "─"
		if hasActiveRun && rs != nil && rs.RPM > 0 {
			rpmText = fmt.Sprintf("%.0f", rs.RPM)
		} else if !hasActiveRun && t.LatestRun != nil && t.LatestRun.RPM > 0 {
			rpmText = fmt.Sprintf("%.0f", t.LatestRun.RPM)
		}

		tpmText := "─"
		if hasActiveRun && rs != nil && rs.TPM > 0 {
			tpmText = fmt.Sprintf("%.0f", rs.TPM)
		} else if !hasActiveRun && t.LatestRun != nil && t.LatestRun.TPM > 0 {
			tpmText = fmt.Sprintf("%.0f", t.LatestRun.TPM)
		}

		rowData[i] = taskRowData{
			name:      t.Name,
			mode:      modeText,
			isTurbo:   isTurbo,
			proto:     shortProtocol(t.Input.NormalizedProtocol()),
			lastRun:   lastRunText,
			isRunning: isRunning,
			rate:      rateText,
			cache:     cacheText,
			ttft:      ttftText,
			tps:       tpsText,
			rpm:       rpmText,
			tpm:       tpmText,
		}
	}

	// ── 构建 lipgloss/table ──
	// colWidths: 0 = 弹性列（占用剩余宽度），>0 = 固定总宽（包括两端各 1 字符 padding）
	colWidths := []int{0, 8, 22, 12, 8, 10, 10, 10, 8, 8} // 任务名称=flex, 模式, 协议, 上次运行, 成功率, 缓存命中, TTFT均值, TPS均值, RPM, TPM
	t := lgtable.New().
		Headers(i18n.T(i18n.KTaskName), i18n.T(i18n.KMode), i18n.T(i18n.KProtocol), i18n.T(i18n.KLastRun), i18n.T(i18n.KSuccessRate), i18n.T(i18n.KColCacheHit), i18n.T(i18n.KColAvgTTFT), i18n.T(i18n.KColAvgTPS), "RPM", "TPM").
		Width(width).
		Height(maxH).
		YOffset(s.Offset).
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
			switch col {
			case 1: // mode
				if r.isTurbo {
					return aw(lipgloss.NewStyle().Foreground(colorGold).Bold(true))
				}
				return aw(lipgloss.NewStyle().Foreground(colorPurple))
			case 3: // lastRun
				if r.isRunning {
					return aw(st.Ok)
				}
				return aw(st.Muted)
			case 4, 5, 6, 7, 8, 9: // rate, cache, ttft, tps, rpm, tpm
				return aw(st.Value)
			default:
				return aw(st.TableRow)
			}
		})

	for _, r := range rowData {
		t.Row(r.name, r.mode, r.proto, r.lastRun, r.rate, r.cache, r.ttft, r.tps, r.rpm, r.tpm)
	}

	tableStr := t.String()
	s.Visible = t.VisibleRows()
	if s.Visible < 1 {
		s.Visible = 1
	}
	s.Offset = ensureVisibleOffset(s.Selected, len(s.Tasks), s.Offset, s.Visible)

	// 空任务状态：在表头下方显示提示
	if len(s.Tasks) == 0 {
		tableLines := strings.Split(tableStr, "\n")
		for len(tableLines) < maxH-1 {
			tableLines = append(tableLines, "")
		}
		tableLines = append(tableLines, "  "+st.Muted.Render(i18n.T(i18n.KNoTasks)))
		if len(tableLines) > maxH {
			tableLines = tableLines[:maxH]
		}
		for len(tableLines) < maxH {
			tableLines = append(tableLines, "")
		}
		return strings.Join(tableLines, "\n")
	}

	// 补齐至 maxH
	tableLines := strings.Split(tableStr, "\n")
	for len(tableLines) < maxH {
		tableLines = append(tableLines, "")
	}
	if len(tableLines) > maxH {
		tableLines = tableLines[:maxH]
	}
	return strings.Join(tableLines, "\n")
}

// buildTaskListConfirmContent 渲染删除确认对话框内容。
func buildTaskListConfirmContent(s *TaskListState, st Styles, width, maxH int) string {
	var lines []string
	task, ok := s.CurrentTask()
	if !ok {
		return strings.Repeat("\n", maxH-1)
	}
	lines = append(lines, "")
	lines = append(lines, st.ErrStyle.Render("  "+i18n.T(i18n.KConfirmDeletePrompt)))
	lines = append(lines, "")
	lines = append(lines, "  "+st.Label.Render(i18n.T(i18n.KTaskName))+"  "+st.Value.Render(truncate(task.Name, maxInt(8, width-14))))
	lines = append(lines, "  "+st.Label.Render(i18n.T(i18n.KTaskID))+"  "+st.Muted.Render(task.ID))
	lines = append(lines, "")
	lines = append(lines, "  "+st.Muted.Render(i18n.T(i18n.KIrreversible)))
	lines = append(lines, "")
	lines = append(lines, "  "+st.Value.Render("[y / Enter]")+"  "+i18n.T(i18n.KConfirmDelete)+"       "+st.Value.Render("[n / Esc]")+"  "+i18n.T(i18n.KCancel))
	for len(lines) < maxH {
		lines = append(lines, "")
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	return strings.Join(lines, "\n")
}
