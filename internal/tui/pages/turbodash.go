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
	var cbItems []HotkeyItem
	switch {
	case hasSel && isRunning:
		cbItems = Hotkeys_TurboDash_Running_Sel()
	case hasSel && !isRunning:
		cbItems = Hotkeys_TurboDash_Done_Sel()
	case !hasSel && isRunning:
		cbItems = Hotkeys_TurboDash_Running_NoSel()
	default:
		cbItems = Hotkeys_TurboDash_Done_NoSel()
	}
	headerLeft := []string{"等待数据"}
	headerRight := []string{}
	if rs != nil {
		headerLeft = []string{runStatusText(string(rs.Status)), fmt.Sprintf("完成 %d/%d", rs.DoneReqs, rs.TotalReqs)}
		currentLevel := rs.CurrentLevel + 1
		if currentLevel < 1 {
			currentLevel = 1
		}
		headerRight = []string{fmt.Sprintf("级别 %d", currentLevel)}
		if len(rs.Levels) > 0 {
			headerRight = append(headerRight, fmt.Sprintf("已探测 %d 档", len(rs.Levels)))
		}
	}
	if d.TaskID != "" {
		headerRight = append(headerRight, "任务 "+truncate(d.TaskID, 14))
	}
	l := PageLayout{
		HeaderTitle:     "Turbo 探测监控",
		HeaderSubtitle:  "观察并发爬坡过程、级别指标与稳定区间",
		HeaderMeta:      "Turbo 模式",
		HeaderInfoLeft:  headerLeft,
		HeaderInfoRight: headerRight,
		Hotkeys:         NewPageHotkeys(cbItems, "[b/Esc] 返回上一页", "[q] 退出"),
	}
	frame := l.Frame(width, height)
	bodyPanel := frame.InnerPanel()

	// ── 计算高度 ──
	splitOuterH := 9
	progressOuterH := 3
	levelOuterH := RemainingStackOuterHeight(frame.InnerHeight, splitOuterH, progressOuterH)
	levelListH := PanelContentHeight(levelOuterH)

	// ── 双栏面板（任务参数 | 当前级别指标）──
	leftPanelFrame, rightPanelFrame := bodyPanel.Split(45, 24)
	leftContent := buildTurboDashParams(rs, st, PanelContentHeight(splitOuterH), leftPanelFrame.InnerWidth)
	rightContent := buildTurboDashMetrics(rs, st, PanelContentHeight(splitOuterH), rightPanelFrame.InnerWidth)
	split := renderSplitPanels(st, leftPanelFrame, rightPanelFrame, leftContent, rightContent)

	// ── 进度条面板 ──
	progressLine := buildTurboProgressLine(rs, st, bodyPanel.InnerWidth)
	progressPanelStr := bodyPanel.Wrap(st, progressLine)

	// ── 级别列表面板 ──
	levelList := buildLevelList(d, rs, st, bodyPanel.InnerWidth, levelListH)
	levelPanelStr := bodyPanel.Wrap(st, levelList)

	content := joinVerticalBlocks(split, progressPanelStr, levelPanelStr)
	return l.Assemble(frame.Wrap(st, content), st, width)
}

// buildTurboDashParams 构建 Turbo 仪表盘左侧任务参数面板。
func buildTurboDashParams(rs *server.RunState, st Styles, maxH, width int) string {
	lines := panelTitleLines(st, "任务参数", width, false)

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

	return finishPanelLines(lines, maxH)
}

// buildTurboDashMetrics 构建 Turbo 仪表盘右侧当前级别实时指标面板。
func buildTurboDashMetrics(rs *server.RunState, st Styles, maxH, width int) string {
	var lines []string

	curLevel := 0
	if rs != nil {
		curLevel = rs.CurrentLevel
	}
	lines = panelTitleLines(st, fmt.Sprintf("当前级别实时指标 [并发 = %d]", curLevel), width, false)

	if rs == nil {
		lines = append(lines, " "+st.Muted.Render("等待数据..."))
	} else {
		lines = append(lines, " "+labelValue(st, "成功率  ", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate))))
		lines = append(lines, " "+labelValue(st, "TPS     ", st.MetricVal.Render(fmt.Sprintf("%.1f tok/s", rs.AvgTPS))))
		lines = append(lines, " "+labelValue(st, "TTFT    ", st.MetricVal.Render(fmtDuration(rs.AvgTTFT))))
		lines = append(lines, " "+labelValue(st, "Cache   ", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.CacheHitRate*100))))
	}

	return finishPanelLines(lines, maxH)
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
	lines := panelTitleLines(st, "级别列表", width, true)

	if rs == nil || len(rs.Levels) == 0 {
		lines = append(lines, " "+st.Muted.Render("等待第一个级别完成..."))
		return finishPanelLines(lines, maxH)
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

	return finishPanelLines(lines, maxH)
}
