package pages

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// TaskListState 任务列表页状态。
type TaskListState struct {
	Tasks    []types.TaskDefinition
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
	return s.Tasks[s.Selected], true
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
		if t.LastRunAt != nil {
			if latest == nil || t.LastRunAt.After(*latest) {
				latest = t.LastRunAt
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
			if s.IsTaskRunning(t.ID) {
				if rs, ok := s.ActiveRuns[t.ID]; ok {
					nav = NavAction{To: NavDashboard, TaskID: t.ID, RunID: rs.RunID}
				}
			} else {
				nav = NavAction{To: NavTaskDetail, TaskID: t.ID}
			}
		}

	case "r":
		if t, ok := s.CurrentTask(); ok {
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
//
// 设计稿布局：
//
//	╔══ AIT  任务中心 ══════════════╗
//	║  ◆ AIT   已保存任务: N   最近运行: xxx ║
//	╠══════════════════════════════╣
//	║  任务名称   模式   协议   上次结果      ║
//	║  ─────────────────────────── ║
//	║ ▶ ◉ name   标准  responses  ✓ 98.5%  ║
//	║     model  并发10  请求200  ◉ 47/100  ║
//	║                              ║
//	╠══════════════════════════════╣
//	║  [Enter] 详情  [a] 新建  ...  ║ ← context bar
//	╠══════════════════════════════╣
//	║  [↑↓] 选择  [q] 退出  ◆ AIT  ║
//	╚══════════════════════════════╝
func RenderTaskList(s *TaskListState, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}

	lastRunStr := ""
	if lt := s.latestRunAt(); lt != nil {
		lastRunStr = "最近运行: " + lt.Format("2006-01-02 15:04")
	}
	var cbItems []ContextBarItem
	if t, ok := s.CurrentTask(); ok {
		if s.IsTaskRunning(t.ID) {
			cbItems = CtxBar_TaskList_Running()
		} else {
			cbItems = CtxBar_TaskList_Normal()
		}
	}
	l := PageLayout{
		TitleLeft:   "AIT  任务中心",
		InfoLeft:    fmt.Sprintf("已保存任务: %d   %s", len(s.Tasks), lastRunStr),
		CtxItems:    cbItems,
		FooterParts: []string{"[↑↓] 选择", "[a] 新建", "[q] 退出", "◆ AIT  v0.1"},
	}

	content := buildTaskListContent(s, st, ContentWidth(width), l.ContentHeight(height))
	return l.Assemble(wrapPanel(st, content, width), st, width)
}

// buildTaskListContent 构建任务列表内容区（含表头 + 任务条目）。
func buildTaskListContent(s *TaskListState, st Styles, width, maxH int) string {
	var lines []string
	showHero := width >= 60 && maxH >= 14
	if showHero {
		heroLines := renderWelcomeHero(st, width)
		lines = append(lines, heroLines...)
		lines = append(lines, dividerLine(st, width))
	}
	listTopLines := len(lines)

	// 列宽（合计 = nameW + modeW + protoW + 结果列）
	nameW := 28
	modeW := 8
	protoW := 14

	// 表头：2 空格前缀与正文行对齐（cursor=2）
	header := renderTableHeader(st, width,
		"  " + padRight("任务名称", nameW) +
			padRight("模式", modeW) +
			padRight("协议", protoW) +
			"上次结果")
	lines = append(lines, header)
	lines = append(lines, dividerLine(st, width))
	listMaxH := maxInt(3, maxH-listTopLines)
	s.Visible = listVisibleItems(listMaxH, 2)
	s.Offset = ensureVisibleOffset(s.Selected, len(s.Tasks), s.Offset, s.Visible)

	if len(s.Tasks) == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+st.Muted.Render("暂无任务  按 [a] 新建第一个任务"))
		return strings.Join(lines, "\n")
	}

	start := s.Offset
	end := minInt(len(s.Tasks), start+s.Visible)
	for i := start; i < end; i++ {
		t := s.Tasks[i]

		isRunning := s.IsTaskRunning(t.ID)
		isSel := i == s.Selected
		rs := s.ActiveRuns[t.ID]

		// ── 指示符 ──
		prefix := padRight(selectionMarker(isSel), 2)

		// ── 模式（选中行禁用嵌套样式，避免重置整行背景）──
		modeText := "标准"
		modeCol := padRight(modeText, modeW)
		if t.Input.Turbo {
			modeText = "Turbo"
			modeCol = padRight(styleWhenNotSelected(isSel, lipgloss.NewStyle().Foreground(colorGold).Bold(true), modeText), modeW)
		} else {
			modeCol = padRight(styleWhenNotSelected(isSel, lipgloss.NewStyle().Foreground(colorPurple), modeText), modeW)
		}

		// ── 协议 ──
		proto := padRight(shortProtocol(t.Input.NormalizedProtocol()), protoW)

		// ── 上次结果（选中行禁用嵌套样式，避免重置整行背景）──
		lastResultText := "从未运行"
		if t.LastRunSummary != nil {
			pct := t.LastRunSummary.SuccessRate
			if t.Input.Turbo {
				if t.LastRunSummary.MaxStableConcurrency > 0 {
					lastResultText = fmt.Sprintf("★ 并发%d", t.LastRunSummary.MaxStableConcurrency)
				}
			} else {
				switch {
				case pct >= 99:
					lastResultText = fmt.Sprintf("✓ %.1f%%", pct)
				case pct >= 90:
					lastResultText = fmt.Sprintf("%.1f%%", pct)
				default:
					lastResultText = fmt.Sprintf("✗ %.1f%%", pct)
				}
			}
		}
		if isRunning && rs != nil {
			lastResultText = fmt.Sprintf("◉ %d/%d  %.0f%%", rs.DoneReqs, rs.TotalReqs, rs.SuccessRate)
		}

		lastResult := lastResultText
		if isRunning && rs != nil {
			lastResult = styleWhenNotSelected(isSel, st.Ok, lastResultText)
		} else if t.LastRunSummary == nil {
			lastResult = styleWhenNotSelected(isSel, st.Muted, lastResultText)
		} else if t.Input.Turbo && t.LastRunSummary.MaxStableConcurrency > 0 {
			lastResult = styleWhenNotSelected(isSel, st.Ok, lastResultText)
		} else if !t.Input.Turbo {
			pct := t.LastRunSummary.SuccessRate
			switch {
			case pct >= 99:
				lastResult = styleWhenNotSelected(isSel, st.Ok, lastResultText)
			case pct >= 90:
				lastResult = styleWhenNotSelected(isSel, st.MetricVal, lastResultText)
			default:
				lastResult = styleWhenNotSelected(isSel, st.ErrStyle, lastResultText)
			}
		}

		// ── 任务名称（裁剪）──
		name := truncate(t.Name, nameW)
		namePad := nameW - lipgloss.Width(name)
		if namePad < 0 {
			namePad = 0
		}
		nameCol := name + strings.Repeat(" ", namePad)

		// ── 第一行 ──
		row1Content := nameCol + modeCol + proto + lastResult
		row1 := renderTableRow(st, width, isSel, prefix+row1Content)
		lines = append(lines, row1)

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
