package pages

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"charm.land/lipgloss/v2"
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
		if t, ok := s.CurrentTask(); ok {
			return s, client.DeleteTaskCmd(t.ID), nav
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
	if t, ok := s.CurrentTask(); ok {
		if s.IsTaskRunning(t.ID) {
			cbItems = Hotkeys_TaskList_Running()
		} else {
			cbItems = Hotkeys_TaskList_Normal()
		}
	} else {
		cbItems = []HotkeyItem{HotkeyAction("a", "新建任务")}
	}
	runningCount := 0
	for _, rs := range s.ActiveRuns {
		if rs != nil && rs.Status == server.RunStatusRunning {
			runningCount++
		}
	}
	headerRight := []string{"暂无运行历史"}
	if latest := s.latestRunAt(); latest != nil {
		headerRight = []string{"最近运行 " + fmtRelativeTime(*latest)}
	}
	if t, ok := s.CurrentTask(); ok {
		headerRight = append([]string{"当前 " + truncate(t.Name, 22)}, headerRight...)
	}
	l := PageLayout{
		HeaderTitle:     "任务中心",
		HeaderSubtitle:  "创建任务、运行压测、查看执行记录与导出报告",
		HeaderMeta:      fmt.Sprintf("%d 个任务", len(s.Tasks)),
		HeaderInfoLeft:  []string{fmt.Sprintf("运行中 %d", runningCount)},
		HeaderInfoRight: headerRight,
		Hotkeys:         NewPageHotkeys(cbItems, "[↑↓] 选择", "[a] 新建", "[q] 退出"),
	}
	frame := l.Frame(width, height)
	panel := NewPanelFrame(frame.OuterWidth)
	content := buildTaskListContent(s, st, panel.InnerWidth, PanelContentHeight(frame.InnerHeight))
	return l.Assemble(panel.Wrap(st, content), st, width)
}

// buildTaskListContent 构建任务列表内容区（含表头 + 任务条目）。
func buildTaskListContent(s *TaskListState, st Styles, width, maxH int) string {
	var lines []string
	listTopLines := len(lines)

	// 列宽（gap=2 作为列间距内置到每个非末尾列的宽度中）
	const (
		modeW    = 9  // 7 + 2 gap
		protoW   = 20 // 10 + 2 gap
		lastRunW = 16 // 11 + 2 gap
		ttftW    = 16 // 10 + 2 gap
		tpsW     = 16  // 末尾列，无需额外 gap
	)
	fixedW := 2 + modeW + protoW + lastRunW + ttftW + tpsW
	nameW := maxInt(10, width-fixedW)

	// 表头：2 空格前缀与正文行对齐（cursor=2）
	header := renderTableHeader(st, width,
		lipgloss.JoinHorizontal(lipgloss.Top,
			tableCol(2, ""),
			tableCol(nameW, "任务名称"),
			tableCol(modeW, "模式"),
			tableCol(protoW, "协议"),
			tableCol(lastRunW, "上次运行"),
			tableCol(ttftW, "TTFT"),
			"TPS",
		))
	lines = append(lines, header)
	lines = append(lines, dividerLine(st, width))
	listMaxH := maxInt(3, maxH-listTopLines)
	s.Visible = listVisibleItems(listMaxH, 2)
	s.Offset = ensureVisibleOffset(s.Selected, len(s.Tasks), s.Offset, s.Visible)

	if len(s.Tasks) == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+st.Muted.Render("暂无任务  按 [a] 新建第一个任务"))
		// 补齐剩余行
		for len(lines) < maxH {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	start := s.Offset
	end := minInt(len(s.Tasks), start+s.Visible)
	for i := start; i < end; i++ {
		t := s.Tasks[i]

		isSel := i == s.Selected
		rs := s.ActiveRuns[t.ID]
		_, hasActiveRun := s.ActiveRuns[t.ID]

		// ── 指示符 ──
		prefix := tableCol(2, selectionMarker(isSel))

		// ── 模式（选中行禁用嵌套样式，避免重置整行背景）──
		modeText := "标准"
		var modeCol string
		if t.Input.Turbo {
			modeText = "Turbo"
			modeCol = tableCol(modeW, styleWhenNotSelected(isSel, lipgloss.NewStyle().Foreground(colorGold).Bold(true), modeText))
		} else {
			modeCol = tableCol(modeW, styleWhenNotSelected(isSel, lipgloss.NewStyle().Foreground(colorPurple), modeText))
		}

		// ── 协议 ──
		proto := tableCol(protoW, shortProtocol(t.Input.NormalizedProtocol()))

		// ── 任务名称 ──
		nameCol := tableCol(nameW, t.Name)

		// ── 上次运行时间 ──
		lastRunText := "─"
		if hasActiveRun || (t.LatestRun != nil && t.LatestRun.Status == string(server.RunStatusRunning)) {
			lastRunText = "运行中"
		} else if t.LatestRun != nil && !t.LatestRun.FinishedAt.IsZero() {
			lastRunText = fmtRelativeTime(t.LatestRun.FinishedAt)
		}
		lastRunStyle := st.Muted
		if hasActiveRun || (t.LatestRun != nil && t.LatestRun.Status == string(server.RunStatusRunning)) {
			lastRunStyle = st.Ok
		}
		lastRunCol := tableCol(lastRunW, styleWhenNotSelected(isSel, lastRunStyle, lastRunText))

		// ── TTFT ──
		ttftText := "─"
		if hasActiveRun && rs != nil && rs.AvgTTFT > 0 {
			ttftText = fmtDuration(rs.AvgTTFT)
		} else if !hasActiveRun && t.LatestRun != nil {
			ttftText = fmtDuration(t.LatestRun.AvgTTFT)
		}
		ttftCol := tableCol(ttftW, styleWhenNotSelected(isSel, st.Value, ttftText))

		// ── TPS ──
		tpsText := "─"
		if hasActiveRun && rs != nil && rs.AvgTPS > 0 {
			tpsText = fmt.Sprintf("%.1f", rs.AvgTPS)
		} else if !hasActiveRun && t.LatestRun != nil {
			if t.Input.Turbo && t.LatestRun.MaxStableConcurrency > 0 {
				tpsText = fmt.Sprintf("并发%d", t.LatestRun.MaxStableConcurrency)
			} else if !t.Input.Turbo {
				tpsText = fmt.Sprintf("%.1f", t.LatestRun.AvgTPS)
			}
		}
		tpsCol := styleWhenNotSelected(isSel, st.Value, tpsText)

		// ── 单行：名称 | 模式 | 协议 | 上次运行 | TTFT | TPS ──
		lines = append(lines, renderTableRow(st, width, isSel, lipgloss.JoinHorizontal(lipgloss.Top,
			prefix, nameCol, modeCol, proto, lastRunCol, ttftCol, tpsCol)))

		// ── 分隔线 ──
		if i < end-1 && len(lines) < maxH-1 {
			lines = append(lines, dividerLine(st, width))
		}
	}

	// 补齐剩余行
	for len(lines) < maxH {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
