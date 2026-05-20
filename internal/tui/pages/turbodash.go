package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/types"
)

// TurboDashState Turbo 模式仪表盘页状态。
type TurboDashState struct {
	RunID    server.RunID
	TaskID   string
	EventCh  <-chan server.Event
	CancelFn server.CancelFunc
	RunState *server.RunState
	LevelSel int       // 选中的级别索引（-1 = 无选中）
	LevelOff int
	LevelVis int
	BackNav  NavAction // 按 b/esc 时的返回目标；Zero = 返回任务列表
}

// NewTurboDashState 创建 Turbo 仪表盘初始状态。
func NewTurboDashState(runID server.RunID, taskID string) *TurboDashState {
	return &TurboDashState{
		RunID:    runID,
		TaskID:   taskID,
		LevelSel: -1,
	}
}

// IsRunning 判断是否仍在运行。
func (d *TurboDashState) IsRunning() bool {
	if d == nil || d.RunState == nil {
		return false
	}
	return d.RunState.Status == server.RunStatusRunning
}

// HandleTurboDashKey 处理 Turbo 仪表盘按键。
func HandleTurboDashKey(d *TurboDashState, msg tea.KeyMsg, client Client) (*TurboDashState, tea.Cmd, NavAction) {
	nav := NavAction{}
	if d == nil {
		return d, nil, NavAction{To: NavTaskList}
	}

	var levels []types.TurboLevelResult
	if d.RunState != nil {
		levels = d.RunState.Levels
	}

	switch msg.String() {
	case "up", "k":
		if len(levels) == 0 {
			break
		}
		if d.LevelSel < 0 {
			// 首次按键：跳到最后一级（最新/最高并发），与 ↓ 保持一致
			d.LevelSel = len(levels) - 1
		} else if d.LevelSel <= 0 {
			d.LevelSel = len(levels) - 1
		} else {
			d.LevelSel--
		}

	case "down", "j":
		if len(levels) == 0 {
			break
		}
		if d.LevelSel < 0 {
			// 首次按键：跳到最后一级（最新/最高并发），与 ↑ 保持一致
			d.LevelSel = len(levels) - 1
		} else if d.LevelSel < len(levels)-1 {
			d.LevelSel++
		} else {
			d.LevelSel = 0
		}

	case "enter":
		// 进入该级别的请求列表，定位到该级别第一条请求
		if d.LevelSel >= 0 && d.LevelSel < len(levels) {
			startIdx := 0
			for j := 0; j < d.LevelSel; j++ {
				startIdx += levels[j].TotalRequests
			}
			nav = NavAction{To: NavReqDetail, ReqIndex: startIdx}
		}

	case "s":
		if d.IsRunning() {
			return d, client.StopRunCmd(d.RunID), nav
		}

	case "m":
		// 手动标记极限并停止
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
			return d, client.GenerateReportCmd(d.RunID, server.ReportFormatJSON), nav
		}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}
	d.LevelOff = ensureVisibleOffset(d.LevelSel, len(levels), d.LevelOff, d.LevelVis)

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
	hasSel := d.LevelSel >= 0 && rs != nil && d.LevelSel < len(rs.Levels)
	var cbItems []ContextBarItem
	switch {
	case hasSel && isRunning:
		cbItems = CtxBar_TurboDash_Running_Sel()
	case hasSel && !isRunning:
		cbItems = CtxBar_TurboDash_Done_Sel()
	case !hasSel && isRunning:
		cbItems = CtxBar_TurboDash_Running_NoSel()
	default:
		cbItems = CtxBar_TurboDash_Done_NoSel()
	}
	l := PageLayout{
		CtxItems:    cbItems,
		FooterParts: []string{"[b/Esc] 返回上一页", "[q] 退出"},
	}
	innerW := ContentWidth(width)
	innerH := l.ContentHeight(height)

	// ── 计算高度 ──
	splitH := 9
	progressPanel := 3
	levelListH := innerH - splitH - progressPanel - 2
	if levelListH < 3 {
		levelListH = 3
	}

	// ── 双栏面板（任务参数 | 当前级别指标）──
	leftW := innerW * 45 / 100
	rightW := innerW - leftW
	leftContent := buildTurboDashParams(rs, st, splitH-2, leftW-2)
	rightContent := buildTurboDashMetrics(rs, st, splitH-2, rightW-2)
	leftPanel := st.Panel.Width(leftW - 2).Render(leftContent)
	rightPanel := st.Panel.Width(rightW - 2).Render(rightContent)
	split := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// ── 进度条面板 ──
	progressLine := buildTurboProgressLine(rs, st, ContentWidth(innerW))
	progressPanelStr := wrapPanel(st, progressLine, innerW)

	// ── 级别列表面板 ──
	levelList := buildLevelList(d, rs, st, ContentWidth(innerW), levelListH)
	levelPanelStr := wrapPanel(st, levelList, innerW)

	content := strings.Join([]string{split, progressPanelStr, levelPanelStr}, "\n")
	return l.Assemble(wrapPanel(st, content, width), st, width)
}

// buildTurboDashParams 构建 Turbo 仪表盘左侧任务参数面板。
func buildTurboDashParams(rs *server.RunState, st Styles, maxH, width int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("任务参数"))
	lines = append(lines, "")

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
	} else {
		if rs.TurboResult != nil {
			tc := rs.TurboResult.Config
			lines = append(lines, " "+labelValue(st, "爬坡  ", fmt.Sprintf("%d→%d  步进+%d", tc.InitConcurrency, tc.MaxConcurrency, tc.StepSize)))
			lines = append(lines, " "+labelValue(st, "每级  ", fmt.Sprintf("%d 请求", tc.LevelRequests)))
			lines = append(lines, " "+labelValue(st, "停止  ", fmt.Sprintf("成功率 < %.0f%%", tc.MinSuccessRate*100)))
		}
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildTurboDashMetrics 构建 Turbo 仪表盘右侧当前级别实时指标面板。
func buildTurboDashMetrics(rs *server.RunState, st Styles, maxH, width int) string {
	var lines []string

	curLevel := 0
	if rs != nil {
		curLevel = rs.CurrentLevel
	}
	lines = append(lines, " "+st.SectionHead.Render(fmt.Sprintf("当前级别实时指标 [并发 = %d]", curLevel)))
	lines = append(lines, "")

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
	} else {
		lines = append(lines, " "+labelValue(st, "成功率  ", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate))))
		lines = append(lines, " "+labelValue(st, "TPS     ", st.MetricVal.Render(fmt.Sprintf("%.1f tok/s", rs.AvgTPS))))
		lines = append(lines, " "+labelValue(st, "TTFT    ", st.MetricVal.Render(fmtDuration(rs.AvgTTFT))))
		lines = append(lines, " "+labelValue(st, "Cache   ", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.CacheHitRate*100))))
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}

// buildTurboProgressLine 构建 Turbo 模式进度条行。
func buildTurboProgressLine(rs *server.RunState, st Styles, width int) string {
	if rs == nil {
		return " 进度  " + st.Muted.Render("等待中...")
	}
	total := rs.TotalReqs
	done := rs.DoneReqs
	var ratio float64
	if total > 0 {
		ratio = float64(done) / float64(total)
	}
	levelTotal := len(rs.Levels)
	prefix := " 进度  "
	suffix := fmt.Sprintf("  %d/%d  当前并发 %d   总进度: 已完成 %d/~? 级", done, total, rs.CurrentLevel, levelTotal)

	barW := width - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	if barW < 5 {
		barW = 5
	}

	filled := int(ratio * float64(barW))
	barRendered := st.Ok.Render(strings.Repeat("█", filled)) +
		st.Muted.Render(strings.Repeat("░", barW-filled))

	return prefix + barRendered + suffix
}

// buildLevelList 构建 Turbo 级别列表区域。
func buildLevelList(d *TurboDashState, rs *server.RunState, st Styles, width, maxH int) string {
	var lines []string
	lines = append(lines, " "+st.SectionHead.Render("级别列表"))

	if rs == nil || len(rs.Levels) == 0 {
		lines = append(lines, " "+st.Muted.Render("等待第一个级别完成..."))
		for len(lines) < maxH {
			lines = append(lines, "")
		}
		return strings.Join(lines, "\n")
	}

	// 列宽（header 与 content 行保持一致，前缀均为 2 字符）
	const (
		markW  = 2  // 选择标记列
		concW  = 6  // 并发数
		rateW  = 8  // 成功率
		tpsW   = 10 // TPS
		ttftW  = 10 // TTFT
		cacheW = 8  // Cache
		totW   = 9  // 总耗时
		// 结论: 余量
	)
	hdr := padRight("", markW) + padRight("并发", concW) + padRight("成功率", rateW) + padRight("TPS", tpsW) +
		padRight("TTFT", ttftW) + padRight("Cache", cacheW) + padRight("总耗时", totW) + "结论"
	lines = append(lines, renderTableHeader(st, width, hdr))
	lines = append(lines, dividerLine(st, width))
	d.LevelVis = listVisibleItems(maxH, 3)
	d.LevelOff = ensureVisibleOffset(d.LevelSel, len(rs.Levels), d.LevelOff, d.LevelVis)
	start := d.LevelOff
	end := minInt(len(rs.Levels), start+d.LevelVis)

	for i := start; i < end; i++ {
		lv := rs.Levels[i]
		isSel := i == d.LevelSel

		conclusionText := "✓ 稳定"
		if !lv.Stable {
			conclusionText = "✗ 降级"
		}
		isCurrent := (i == len(rs.Levels)-1) && rs.Status == server.RunStatusRunning
		if isCurrent {
			conclusionText = "🔄 进行中"
		}

		conclusion := conclusionText
		if isCurrent {
			conclusion = styleWhenNotSelected(isSel, st.MetricVal, conclusionText)
		} else if lv.Stable {
			conclusion = styleWhenNotSelected(isSel, st.Ok, conclusionText)
		} else {
			conclusion = styleWhenNotSelected(isSel, st.ErrStyle, conclusionText)
		}

		marker := selectionMarker(isSel)

		rowContent := padRight(marker, markW) +
			padRight(fmt.Sprintf("%d", lv.Concurrency), concW) +
			padRight(fmt.Sprintf("%.1f%%", lv.SuccessRate*100), rateW) +
			padRight(fmt.Sprintf("%.1f", lv.AvgTPS), tpsW) +
			padRight(fmtDuration(lv.AvgTTFT), ttftW) +
			padRight(fmt.Sprintf("%.1f%%", lv.CacheHitRate*100), cacheW) +
			padRight(fmtDuration(lv.AvgTotalTime), totW) +
			conclusion

		rendered := renderTableRow(st, width, isSel, rowContent)
		lines = append(lines, rendered)

		// 行间分隔线
		if i < end-1 && len(lines) < maxH-1 {
			lines = append(lines, dividerLine(st, width))
		}
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}
