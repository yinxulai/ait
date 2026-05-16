package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/types"
)

// historyState 任务详情页的历史数据。
type historyState struct {
	taskID  string
	history []types.TaskRunSummary
}

// ─── 渲染 ─────────────────────────────────────────────────────────────────────

func (m *Model) renderTaskDetail() string {
	if m.width == 0 {
		return "加载中..."
	}

	task, ok := m.taskList.currentTask()
	if !ok {
		return "请先选择任务"
	}

	header := m.renderHeader(
		"AIT  任务详情",
		task.Name,
	)

	var cbItems []contextBarItem
	if m.hist != nil {
		cbItems = contextBarItems_taskDetail(len(m.hist.history) > 0)
	} else {
		cbItems = contextBarItems_taskDetail(false)
	}
	contextBar := m.renderContextBar(cbItems)
	footer := m.renderFooter("[←/Esc] 返回", "[r] 运行", "[e] 编辑", "◆ AIT")

	cbH := 0
	if contextBar != "" {
		cbH = 1
	}
	contentH := m.height - 1 - cbH - 1
	if contentH < 4 {
		contentH = 4
	}

	leftW := (m.width - 4) * 55 / 100
	rightW := m.width - 4 - leftW

	leftContent := m.buildDetailLeft(task, contentH, leftW)
	rightContent := m.buildDetailRight(contentH)
	mid := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, contentH)

	parts := []string{header, mid}
	if contextBar != "" {
		parts = append(parts, contextBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func (m *Model) buildDetailLeft(task types.TaskDefinition, maxH, width int) string {
	inp := task.Input
	var lines []string

	lines = append(lines, m.styles.sectionHead.Render("基本配置"))
	lines = append(lines, "")
	lines = append(lines, row(m, "名称      ", task.Name))
	lines = append(lines, row(m, "创建时间  ", task.CreatedAt.Format("2006-01-02 15:04:05")))
	lines = append(lines, row(m, "更新时间  ", task.UpdatedAt.Format("2006-01-02 15:04:05")))
	if task.LastRunAt != nil {
		lines = append(lines, row(m, "上次运行  ", timeAgo(*task.LastRunAt)))
	}
	lines = append(lines, "")
	lines = append(lines, m.styles.sectionHead.Render("测试参数"))
	lines = append(lines, "")
	lines = append(lines, row(m, "协议      ", shortProtocol(inp.NormalizedProtocol())))
	lines = append(lines, row(m, "接口地址  ", truncate(inp.ResolvedEndpointURL(), width-20)))
	lines = append(lines, row(m, "API Key   ", maskAPIKey(inp.ApiKey)))
	lines = append(lines, row(m, "模型      ", inp.Model))

	modeStr := "标准"
	if inp.Turbo {
		modeStr = "Turbo (并发探测)"
	}
	lines = append(lines, row(m, "测试模式  ", modeStr))

	if inp.Turbo {
		tc := inp.TurboConfig
		lines = append(lines, row(m, "初始并发  ", fmt.Sprintf("%d", tc.InitConcurrency)))
		lines = append(lines, row(m, "最大并发  ", fmt.Sprintf("%d", tc.MaxConcurrency)))
		lines = append(lines, row(m, "步进大小  ", fmt.Sprintf("+%d", tc.StepSize)))
		lines = append(lines, row(m, "每级请求  ", fmt.Sprintf("%d", tc.LevelRequests)))
		lines = append(lines, row(m, "最低成功率", fmt.Sprintf("%.0f%%", tc.MinSuccessRate)))
	} else {
		lines = append(lines, row(m, "并发数    ", fmt.Sprintf("%d", inp.Concurrency)))
		lines = append(lines, row(m, "请求总数  ", fmt.Sprintf("%d", inp.Count)))
	}

	lines = append(lines, row(m, "流式输出  ", boolLabel(inp.Stream)))
	lines = append(lines, row(m, "Thinking  ", boolLabel(inp.Thinking)))
	lines = append(lines, "")
	lines = append(lines, m.styles.sectionHead.Render("Prompt 配置"))
	lines = append(lines, "")
	lines = append(lines, row(m, "模式      ", inp.PromptMode))
	lines = append(lines, row(m, "内容      ", truncate(promptSummary(inp), width-20)))

	if m.status != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render(m.status))
	}
	if m.err != nil {
		lines = append(lines, m.styles.errStyle.Render("错误: "+m.err.Error()))
	}

	return strings.Join(lines, "\n")
}

func (m *Model) buildDetailRight(maxH int) string {
	var lines []string
	lines = append(lines, m.styles.sectionHead.Render("运行历史"))
	lines = append(lines, "")

	if m.hist == nil || len(m.hist.history) == 0 {
		lines = append(lines, m.styles.muted.Render("  暂无历史记录"))
		lines = append(lines, "")
		lines = append(lines, m.styles.muted.Render("  按 [Enter] 或 [r] 开始第一次运行"))
		return strings.Join(lines, "\n")
	}

	for i, run := range m.hist.history {
		if len(lines) >= maxH-2 {
			break
		}
		statusIcon := m.styles.ok.Render("✓")
		if run.Status != "completed" {
			statusIcon = m.styles.errStyle.Render("✗")
		}
		elapsed := run.FinishedAt.Sub(run.StartedAt)
		lines = append(lines, fmt.Sprintf("%s #%d  %s",
			statusIcon, i+1, timeAgo(run.StartedAt)))
		lines = append(lines, fmt.Sprintf("   成功率 %.1f%%  TTFT %.0fms  TPS %.1f",
			run.SuccessRate, float64(run.AvgTTFT.Milliseconds()), run.AvgTPS))
		lines = append(lines, fmt.Sprintf("   耗时 %s  模式 %s",
			fmtDuration(elapsed), run.Mode))
		if run.ErrorSummary != "" {
			lines = append(lines, m.styles.errStyle.Render("   "+truncate(run.ErrorSummary, 36)))
		}
		if run.ReportJSONPath != "" {
			lines = append(lines, m.styles.muted.Render("   报告: "+truncate(run.ReportJSONPath, 32)))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// ─── 按键处理 ─────────────────────────────────────────────────────────────────

func (m *Model) handleTaskDetailKey(msg interface{ String() string }) tea.Cmd {
	switch msg.String() {
	case "left", "esc", "b":
		m.view = viewTaskList
		return nil

	case "enter", "r":
		if t, ok := m.taskList.currentTask(); ok {
			return m.startRunIfAllowed(t.ID, false)
		}
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func row(m *Model, label, value string) string {
	return m.styles.label.Render(label) + "  " + m.styles.value.Render(value)
}

func fmtDuration(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	s := float64(ms) / 1000
	if s < 60 {
		return fmt.Sprintf("%.1fs", s)
	}
	return fmt.Sprintf("%.0fm%.0fs", s/60, float64(int64(s)%60))
}
