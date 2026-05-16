package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/types"
)

// taskListState 任务列表页的局部状态。
type taskListState struct {
	tasks    []types.TaskDefinition
	selected int
}

// currentTask 返回当前选中的任务。
func (s *taskListState) currentTask() (types.TaskDefinition, bool) {
	if len(s.tasks) == 0 || s.selected < 0 || s.selected >= len(s.tasks) {
		return types.TaskDefinition{}, false
	}
	return s.tasks[s.selected], true
}

// ─── 按键处理 ─────────────────────────────────────────────────────────────────

func (m *Model) handleTaskListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := &m.taskList

	switch msg.String() {
	case "up", "k":
		if s.selected > 0 {
			s.selected--
		}
	case "down", "j":
		if s.selected < len(s.tasks)-1 {
			s.selected++
		}

	case "a":
		// 新建任务 — 打开向导
		m.openWizard(nil)

	case "e":
		if t, ok := s.currentTask(); ok {
			m.openWizard(&t)
		}

	case "y":
		// 复制任务
		if t, ok := s.currentTask(); ok {
			return m, m.client.CopyTaskCmd(t.ID)
		}

	case "d":
		// 删除任务
		if t, ok := s.currentTask(); ok {
			return m, m.client.DeleteTaskCmd(t.ID)
		}

	case "enter":
		if t, ok := s.currentTask(); ok {
			// 如果是运行中任务，进入仪表盘
			if m.dash != nil && m.dash.runID != "" && m.dash.taskID == t.ID {
				m.view = viewDashboard
				return m, nil
			}
			// 否则进入任务详情并加载历史
			m.view = viewTaskDetail
			return m, m.client.LoadHistoryCmd(t.ID, 10)
		}

	case "r":
		if t, ok := s.currentTask(); ok {
			return m, m.startRunIfAllowed(t.ID, false)
		}

	case "s":
		// 停止当前运行中的任务（若选中的是运行中任务）
		if t, ok := s.currentTask(); ok {
			if m.dash != nil && m.dash.taskID == t.ID {
				return m, m.client.StopRunCmd(m.dash.runID)
			}
		}

	case "q":
		return m, tea.Quit
	}

	return m, nil
}

// ─── 渲染 ─────────────────────────────────────────────────────────────────────

func (m *Model) renderTaskList() string {
	if m.width == 0 {
		return "加载中..."
	}
	s := &m.taskList

	lastRunStr := ""
	for _, t := range s.tasks {
		if t.LastRunAt != nil {
			lastRunStr = "最近: " + timeAgo(*t.LastRunAt)
			break
		}
	}
	header := m.renderHeader(
		"AIT  任务中心",
		fmt.Sprintf("已保存任务: %d  %s", len(s.tasks), lastRunStr),
	)

	// 决定 context bar 内容
	var cbItems []contextBarItem
	if t, ok := s.currentTask(); ok {
		isRunning := m.dash != nil && m.dash.taskID == t.ID && m.dash.isRunning()
		cbItems = contextBarItems_taskList(isRunning)
	}
	contextBar := m.renderContextBar(cbItems)
	footer := m.renderFooter("[↑↓] 选择", "[a] 新建", "[q] 退出", "◆ AIT v0.1")

	// 内容区高度 = 总高 - header(1) - contextbar - footer(1)
	cbH := 0
	if contextBar != "" {
		cbH = 1
	}
	contentH := m.height - 1 - cbH - 1
	if contentH < 4 {
		contentH = 4
	}

	leftW := (m.width - 4) * 65 / 100
	rightW := m.width - 4 - leftW

	leftContent := m.buildTaskListTable(contentH, leftW)
	rightContent := m.buildTaskListSidebar(contentH)
	mid := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, contentH)

	parts := []string{header, mid}
	if contextBar != "" {
		parts = append(parts, contextBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func (m *Model) buildTaskListTable(maxH, width int) string {
	s := &m.taskList
	var lines []string

	lines = append(lines, m.styles.tableHead.Render(
		fmt.Sprintf("  %-28s %-9s %-14s %s", "任务名称", "模式", "协议", "上次结果"),
	))
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", width)))

	if len(s.tasks) == 0 {
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render("  暂无任务  按 [a] 新建"))
		return strings.Join(lines, "\n")
	}

	for i, t := range s.tasks {
		if len(lines) >= maxH-1 {
			break
		}

		// 运行中标记
		runIndicator := " "
		if m.dash != nil && m.dash.taskID == t.ID && m.dash.isRunning() {
			runIndicator = m.styles.ok.Render("◉")
		}

		// 模式列（手动对齐 9 列宽）
		var modeRendered string
		if t.Input.Turbo {
			modeRendered = m.styles.tagTurbo.Render("Turbo")
		} else {
			modeRendered = m.styles.tagStd.Render("标准")
		}
		modePad := 9 - lipgloss.Width(modeRendered)
		if modePad < 0 {
			modePad = 0
		}
		modeCol := modeRendered + strings.Repeat(" ", modePad)

		proto := shortProtocol(t.Input.NormalizedProtocol())
		lastResult := m.styles.muted.Render("从未运行")
		if t.LastRunSummary != nil {
			pct := t.LastRunSummary.SuccessRate
			if pct >= 99 {
				lastResult = m.styles.ok.Render(fmt.Sprintf("✓ %.1f%%", pct))
			} else if pct >= 90 {
				lastResult = m.styles.metricVal.Render(fmt.Sprintf("%.1f%%", pct))
			} else {
				lastResult = m.styles.errStyle.Render(fmt.Sprintf("✗ %.1f%%", pct))
			}
		}

		nameStr := truncate(t.Name, 27)
		nameCol := fmt.Sprintf("%-27s ", nameStr)
		protoCol := fmt.Sprintf("%-14s ", proto)

		mainRow := runIndicator + " " + nameCol + modeCol + " " + protoCol + lastResult
		if i == s.selected {
			// 选中行：纯文本 + tableRowSel 背景
			plainMode := "标准"
			if t.Input.Turbo {
				plainMode = "Turbo"
			}
			plainRow := " ▶ " + nameCol + fmt.Sprintf("%-9s ", plainMode) + protoCol + lastResult
			lines = append(lines, m.styles.tableRowSel.Width(width).Render(plainRow))
		} else {
			lines = append(lines, mainRow)
		}

		// 二级子行：配置摘要
		var sub string
		if m.dash != nil && m.dash.taskID == t.ID && m.dash.isRunning() {
			rs := m.dash.runState
			if rs != nil {
				sub = fmt.Sprintf("     %s  ◉ %d/%d  成功率 %.1f%%",
					truncate(t.Input.Model, 18), rs.DoneReqs, rs.TotalReqs, rs.SuccessRate)
			}
		}
		if sub == "" {
			if t.Input.Turbo {
				tc := t.Input.TurboConfig
				sub = fmt.Sprintf("     %s  %d→%d 步进+%d 每级%d",
					truncate(t.Input.Model, 18),
					tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize, tc.LevelRequests)
			} else {
				sub = fmt.Sprintf("     %s  并发%d/请求%d",
					truncate(t.Input.Model, 20), t.Input.Concurrency, t.Input.Count)
			}
		}
		if i == s.selected {
			lines = append(lines, m.styles.tableRowSel.Width(width).Render(sub))
		} else {
			lines = append(lines, m.styles.muted.Render(sub))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) buildTaskListSidebar(maxH int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("快捷操作"))
	lines = append(lines, "")
	lines = append(lines, " "+m.styles.key.Render("[a]")+"  新建任务")
	lines = append(lines, " "+m.styles.key.Render("[Enter]")+"  查看详情 / 进仪表盘")
	lines = append(lines, " "+m.styles.key.Render("[r]")+"  运行选中任务")
	lines = append(lines, " "+m.styles.key.Render("[e]")+"  编辑  "+
		m.styles.key.Render("[d]")+"  删除  "+
		m.styles.key.Render("[y]")+"  复制")
	lines = append(lines, "")
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", 28)))
	lines = append(lines, "")
	lines = append(lines, m.styles.sectionHead.Render("最近执行"))
	lines = append(lines, "")

	count := 0
	for _, t := range m.taskList.tasks {
		if t.LastRunSummary == nil {
			continue
		}
		s := t.LastRunSummary
		icon := m.styles.ok.Render("✓")
		if s.SuccessRate < 90 {
			icon = m.styles.errStyle.Render("✗")
		}
		lines = append(lines, fmt.Sprintf(" %s %-16s %.1f%%  %.0f tok/s",
			icon, truncate(t.Name, 16), s.SuccessRate, s.AvgTPS))
		count++
		if count >= 5 || len(lines) >= maxH-2 {
			break
		}
	}
	if count == 0 {
		lines = append(lines, m.styles.muted.Render("  暂无记录"))
	}

	if m.status != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render(m.status))
	}
	if m.err != nil {
		lines = append(lines, m.styles.errStyle.Render("错误: "+m.err.Error()))
	}
	return strings.Join(lines, "\n")
}

// startRunIfAllowed 根据是否已有运行中任务决定是否启动新运行。
// forceStart=true 表示无论是否有其他任务都启动（用于向导 [r] 保存并运行）。
func (m *Model) startRunIfAllowed(taskID string, forceStart bool) tea.Cmd {
	if !forceStart && m.dash != nil && m.dash.isRunning() {
		m.status = fmt.Sprintf("已有任务 %q 在运行中，多任务并行可能影响网络指标",
			m.dash.taskID)
		return nil
	}
	return m.client.StartRunCmd(taskID)
}

// ─── 共享渲染工具 ─────────────────────────────────────────────────────────────

// 这些函数被多个 page_*.go 使用，统一放在此文件。

func progressBar(current, total, width int) string {
	if total <= 0 || width <= 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width))
	}
	filled := current * width / total
	if filled > width {
		filled = width
	}
	bar := lipgloss.NewStyle().Foreground(colorGreen).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width-filled))
	return bar + empty
}

func progressBarRed(current, total, width int) string {
	if total <= 0 || width <= 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width))
	}
	filled := current * width / total
	if filled > width {
		filled = width
	}
	bar := lipgloss.NewStyle().Foreground(colorRed).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(strings.Repeat("░", width-filled))
	return bar + empty
}

// isRunningTask 判断任务是否当前正在运行。
func (m *Model) isRunningTask(taskID string) bool {
	return m.dash != nil && m.dash.taskID == taskID && m.dash.isRunning()
}

// ─── 工具函数 ─────────────────────────────────────────────────────────────────

func truncate(s string, n int) string {
	if n <= 0 || len([]rune(s)) <= n {
		return s
	}
	r := []rune(s)
	if n <= 3 {
		return string(r[:n])
	}
	return string(r[:n-3]) + "..."
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds 前", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm 前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh 前", int(d.Hours()))
	}
	return t.Format("01-02 15:04")
}

func shortProtocol(p string) string {
	p = strings.ReplaceAll(p, "openai-", "")
	p = strings.ReplaceAll(p, "anthropic-", "")
	return p
}

func boolLabel(v bool) string {
	if v {
		return "开启"
	}
	return "关闭"
}

func promptSummary(input types.Input) string {
	switch input.PromptMode {
	case promptModeFile:
		return input.PromptFile
	case promptModeGenerated:
		return fmt.Sprintf("长度 %d", input.PromptLength)
	default:
		if len([]rune(input.PromptText)) > 48 {
			return string([]rune(input.PromptText)[:45]) + "..."
		}
		return input.PromptText
	}
}

func maskAPIKey(key string) string {
	if len(key) == 0 {
		return "(空)"
	}
	if len(key) <= 8 {
		return strings.Repeat("•", len(key))
	}
	return key[:4] + strings.Repeat("•", len(key)-8) + key[len(key)-4:]
}

func wrapIndex(index, length int) int {
	if length == 0 {
		return 0
	}
	for index < 0 {
		index += length
	}
	return index % length
}
