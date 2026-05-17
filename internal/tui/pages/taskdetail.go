package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/types"
)

// TaskDetailState 任务详情页状态。
type TaskDetailState struct {
	Task    types.TaskDefinition
	History []types.TaskRunSummary
	// LatestExpanded 控制最近一次运行是否展开（运行结束后自动置 true）
	LatestExpanded bool
}

// NewTaskDetailState 创建初始任务详情状态。
func NewTaskDetailState(task types.TaskDefinition) *TaskDetailState {
	return &TaskDetailState{Task: task}
}

// HandleTaskDetailKey 处理任务详情页按键。
func HandleTaskDetailKey(s *TaskDetailState, msg tea.KeyMsg, client Client) (*TaskDetailState, tea.Cmd, NavAction) {
	nav := NavAction{}
	switch msg.String() {
	case "left", "esc", "b":
		nav = NavAction{To: NavTaskList}

	case "enter", "r":
		return s, client.StartRunCmd(s.Task.ID), nav

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

	updatedStr := timeAgo(t.UpdatedAt)
	var cbItems []ContextBarItem
	if len(s.History) > 0 {
		cbItems = CtxBar_TaskDetail_HasHistory()
	} else {
		cbItems = CtxBar_TaskDetail_NoHistory()
	}
	l := PageLayout{
		TitleLeft: "AIT  任务详情 ─ " + truncate(t.Name, 30),
		InfoLeft: fmt.Sprintf("◆ AIT   任务 ID: %s   更新: %s   %s",
			truncate(t.ID, 10), t.UpdatedAt.Format("2006-01-02 15:04"), updatedStr),
		CtxItems:    cbItems,
		FooterParts: []string{"[b/Esc] 返回列表", "[r] 运行", "[e] 编辑", "◆ AIT  v0.1"},
	}

	content := buildTaskDetailContent(s, st, t, inp, ContentWidth(width), l.ContentHeight(height))
	return l.Assemble(wrapPanel(st, content, width), st, width)
}

// buildTaskDetailContent 构建任务详情内容区。
func buildTaskDetailContent(s *TaskDetailState, st Styles, t types.TaskDefinition, inp types.Input, width, maxH int) string {
	innerW := width - 2
	if innerW < 10 {
		innerW = 10
	}

	var lines []string

	// ─── 配置摘要 ─────────────────────────────────────────────
	lines = append(lines, "  "+st.SectionHead.Render("配置摘要"))
	lines = append(lines, "  "+dividerLine(st, innerW-2))

	// 行1：协议 + 接口
	proto := inp.NormalizedProtocol()
	endpoint := truncate(inp.ResolvedEndpointURL(), innerW-30)
	lines = append(lines, "  "+
		st.Label.Render("协议")+"  "+st.Value.Render(proto)+
		"    "+st.Label.Render("接口")+"  "+st.Value.Render(endpoint))

	// 行2：模型 + 模式 + 并发 + 请求
	modeStr := "标准模式"
	if inp.Turbo {
		modeStr = "Turbo 模式"
	}
	if inp.Turbo {
		tc := inp.TurboConfig
		lines = append(lines, "  "+
			st.Label.Render("模型")+"  "+st.Value.Render(inp.Model)+
			"    "+st.Label.Render("模式")+"  "+st.Value.Render(modeStr)+
			"    "+st.Label.Render("并发爬坡")+"  "+
			st.Value.Render(fmt.Sprintf("%d → %d  步进+%d  每级%d请求",
				tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize, tc.LevelRequests)))
	} else {
		lines = append(lines, "  "+
			st.Label.Render("模型")+"  "+st.Value.Render(inp.Model)+
			"    "+st.Label.Render("模式")+"  "+st.Value.Render(modeStr)+
			"    "+st.Label.Render("并发")+"  "+st.Value.Render(fmt.Sprintf("%d", inp.Concurrency))+
			"    "+st.Label.Render("请求")+"  "+st.Value.Render(fmt.Sprintf("%d", inp.Count)))
	}

	// 行3：超时 + 流式 + Prompt
	prompt := promptSummary(inp.PromptMode, inp.PromptText, inp.PromptFile, inp.PromptLength)
	lines = append(lines, "  "+
		st.Label.Render("超时")+"  "+st.Value.Render(fmtDuration(inp.Timeout))+
		"    "+st.Label.Render("流式")+"  "+st.Value.Render(boolLabel(inp.Stream))+
		"    "+st.Label.Render("Prompt")+"  "+st.Value.Render(truncate(prompt, innerW-50)))

	lines = append(lines, "")

	// ─── 最近运行 ──────────────────────────────────────────────
	if len(s.History) > 0 {
		latest := s.History[0]
		statusStr := "✓ 完成"
		if latest.Status != "completed" {
			statusStr = "✗ " + latest.Status
		}
		elapsed := latest.FinishedAt.Sub(latest.StartedAt)
		expandMark := "▼"
		if !s.LatestExpanded {
			expandMark = "▶"
		}
		lines = append(lines, fmt.Sprintf("  %s  %s 最近运行 %s  %s  %d 请求  耗时 %s",
			st.SectionHead.Render("最近运行"),
			st.Ok.Render(expandMark),
			latest.StartedAt.Format("2006-01-02 15:04"),
			st.Ok.Render(statusStr),
			0, // 请求总数（运行摘要中需要补充该字段，暂用 0）
			fmtDuration(elapsed),
		))
		lines = append(lines, "  "+dividerLine(st, innerW-2))

		if s.LatestExpanded && len(lines) < maxH-10 {
			// 指标表格
			lines = append(lines, "  "+st.TableHead.Render(
				padRight("指标", 16)+padRight("最小值", 10)+padRight("平均值", 10)+padRight("标准差", 10)+"最大值"))
			lines = append(lines, "  "+st.Divider.Render(strings.Repeat("─", innerW-2)))

			if len(lines) < maxH {
				lines = append(lines, buildMetricRow(st, "TTFT",
					fmtDuration(latest.AvgTTFT), fmtDuration(latest.AvgTTFT), "─", "─"))
			}
			if len(lines) < maxH {
				lines = append(lines, buildMetricRow(st, "输出 TPS",
					"─", fmt.Sprintf("%.1f", latest.AvgTPS), "─", "─"))
			}
			if len(lines) < maxH {
				lines = append(lines, buildMetricRow(st, "成功率",
					"─", fmt.Sprintf("%.1f%%", latest.SuccessRate*100), "─", "─"))
			}
			if latest.CacheHitRate > 0 && len(lines) < maxH {
				lines = append(lines, buildMetricRow(st, "缓存命中率",
					"─", fmt.Sprintf("%.1f%%", latest.CacheHitRate*100), "─", "─"))
			}
			if latest.ErrorSummary != "" && len(lines) < maxH {
				lines = append(lines, "  "+st.ErrStyle.Render("错误  "+truncate(latest.ErrorSummary, innerW-10)))
			}
		}
		lines = append(lines, "")
	}

	// ─── 历史运行记录 ──────────────────────────────────────────
	if len(lines) < maxH-4 {
		lines = append(lines, "  "+st.SectionHead.Render("历史运行记录"))
		lines = append(lines, "  "+st.TableHead.Render(
			padRight("时间", 20)+padRight("模式", 8)+padRight("成功率", 8)+
				padRight("TTFT", 10)+padRight("TPS", 10)+"Cache"))
		lines = append(lines, "  "+st.Divider.Render(strings.Repeat("─", innerW-2)))

		for _, run := range s.History {
			if len(lines) >= maxH-1 {
				break
			}
			statusIcon := st.Ok.Render("✓")
			if run.Status != "completed" {
				statusIcon = st.ErrStyle.Render("✗")
			}
			modeShort := "标准"
			if run.Mode == "turbo" {
				modeShort = "Turbo"
			}
			cacheStr := "─"
			if run.CacheHitRate > 0 {
				cacheStr = fmt.Sprintf("%.1f%%", run.CacheHitRate*100)
			}
			row := fmt.Sprintf(" %s %s  %s  %s  %s  %s  %s",
				statusIcon,
				run.StartedAt.Format("2006-01-02 15:04"),
				padRight(modeShort, 6),
				padRight(fmt.Sprintf("%.1f%%", run.SuccessRate*100), 7),
				padRight(fmtDuration(run.AvgTTFT), 9),
				padRight(fmt.Sprintf("%.1f", run.AvgTPS), 9),
				cacheStr,
			)
			lines = append(lines, "  "+st.TableRow.Render(row))
		}
	}

	// 补齐剩余高度
	for len(lines) < maxH {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
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
	if autoExpand && len(history) > 0 {
		s.LatestExpanded = true
	}
	return s
}
