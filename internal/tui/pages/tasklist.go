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
	if width == 0 {
		return "加载中..."
	}

	// ── Header ──
	lastRunStr := ""
	if lt := s.latestRunAt(); lt != nil {
		lastRunStr = "最近运行: " + lt.Format("2006-01-02 15:04")
	}
	header := renderHeader(st, width,
		"AIT  任务中心",
		"",
		fmt.Sprintf("◆ AIT   已保存任务: %d   %s", len(s.Tasks), lastRunStr),
		"",
	)

	// ── Context Bar ──
	var cbItems []ContextBarItem
	if t, ok := s.CurrentTask(); ok {
		if s.IsTaskRunning(t.ID) {
			cbItems = CtxBar_TaskList_Running()
		} else {
			cbItems = CtxBar_TaskList_Normal()
		}
	}
	ctxBar := RenderContextBar(st, width, cbItems)

	// ── Footer ──
	footer := renderFooter(st, width, "[↑↓] 选择", "[a] 新建", "[y] 复制", "[q] 退出", "◆ AIT  v0.1")

	// ── 可用内容高度 ──
	headerH := 2
	ctxBarH := 0
	if ctxBar != "" {
		ctxBarH = 1
	}
	footerH := 1
	contentH := height - headerH - ctxBarH - footerH
	if contentH < 4 {
		contentH = 4
	}

	// ── 内容区 ──
	content := buildTaskListContent(s, st, width, contentH)

	parts := []string{header, content}
	if ctxBar != "" {
		parts = append(parts, ctxBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

// buildTaskListContent 构建任务列表内容区（含表头 + 任务条目）。
func buildTaskListContent(s *TaskListState, st Styles, width, maxH int) string {
	innerW := width - 2
	if innerW < 20 {
		innerW = 20
	}

	var lines []string

	// 表头行
	nameW := 28
	modeW := 8
	protoW := 14
	resultW := innerW - nameW - modeW - protoW - 4
	if resultW < 10 {
		resultW = 10
	}

	header := st.TableHead.Render(
		" " + padRight("任务名称", nameW) +
			padRight("模式", modeW) +
			padRight("协议", protoW) +
			"上次结果",
	)
	lines = append(lines, header)
	lines = append(lines, " "+st.Divider.Render(strings.Repeat("─", innerW-1)))

	if len(s.Tasks) == 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+st.Muted.Render("暂无任务  按 [a] 新建第一个任务"))
		return strings.Join(lines, "\n")
	}

	for i, t := range s.Tasks {
		if len(lines) >= maxH {
			break
		}

		isRunning := s.IsTaskRunning(t.ID)
		isSel := i == s.Selected
		rs := s.ActiveRuns[t.ID]

		// ── 指示符和运行中标记 ──
		cursor := "  "
		if isSel {
			cursor = "▶ "
		}
		runMark := "  "
		if isRunning {
			runMark = st.Ok.Render("◉") + " "
		}
		prefix := cursor + runMark

		// ── 模式标签 ──
		var modeTag string
		if t.Input.Turbo {
			modeTag = st.TagTurbo.Render("Turbo")
		} else {
			modeTag = st.TagStd.Render("标准 ")
		}
		modeTagW := lipgloss.Width(modeTag)
		modePad := modeW - modeTagW
		if modePad < 0 {
			modePad = 0
		}
		modeCol := modeTag + strings.Repeat(" ", modePad)

		// ── 协议 ──
		proto := padRight(shortProtocol(t.Input.NormalizedProtocol()), protoW)

		// ── 上次结果 ──
		lastResult := st.Muted.Render("从未运行")
		if t.LastRunSummary != nil {
			pct := t.LastRunSummary.SuccessRate
			if t.Input.Turbo {
				if t.LastRunSummary.MaxStableConcurrency > 0 {
					lastResult = st.Ok.Render(fmt.Sprintf("★ 并发%d", t.LastRunSummary.MaxStableConcurrency))
				}
			} else {
				switch {
				case pct >= 99:
					lastResult = st.Ok.Render(fmt.Sprintf("✓ %.1f%%", pct))
				case pct >= 90:
					lastResult = st.MetricVal.Render(fmt.Sprintf("%.1f%%", pct))
				default:
					lastResult = st.ErrStyle.Render(fmt.Sprintf("✗ %.1f%%", pct))
				}
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
		prefixW := lipgloss.Width(prefix)
		row1Content := nameCol + modeCol + proto + lastResult
		var row1 string
		if isSel {
			row1 = st.TableRowSel.Render(prefix+row1Content) + strings.Repeat(" ", max(0, width-prefixW-lipgloss.Width(row1Content)-2))
		} else {
			row1 = "  " + runMark + row1Content
		}
		lines = append(lines, row1)

		// ── 第二行（模型 + 参数 + 实时进度）──
		if len(lines) < maxH {
			indent := "     " // 5 空格缩进（对齐任务名）
			var params string
			if t.Input.Turbo {
				tc := t.Input.TurboConfig
				params = fmt.Sprintf("%s  %d→%d  步进+%d",
					truncate(t.Input.Model, 12),
					tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize)
				if t.LastRunSummary != nil {
					params += fmt.Sprintf("  上次: 峰值 TPS %.1f", t.LastRunSummary.AvgTPS)
				}
			} else {
				params = fmt.Sprintf("%s  并发%d  请求%d",
					truncate(t.Input.Model, 12),
					t.Input.Concurrency, t.Input.Count)
			}

			// 实时进度
			if isRunning && rs != nil {
				prog := fmt.Sprintf("  %s %d/%d  成功率 %.1f%%",
					st.Ok.Render("◉"), rs.DoneReqs, rs.TotalReqs, rs.SuccessRate*100)
				params += prog
			}

			row2 := indent + st.Muted.Render(truncate(params, width-7))
			lines = append(lines, row2)
		}

		// ── 空行分隔 ──
		if i < len(s.Tasks)-1 && len(lines) < maxH-1 {
			lines = append(lines, "")
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
