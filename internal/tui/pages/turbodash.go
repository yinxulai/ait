package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	LevelSel int // 选中的级别索引（-1 = 无选中）
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
		if d.LevelSel <= 0 {
			d.LevelSel = len(levels) - 1
		} else {
			d.LevelSel--
		}

	case "down", "j":
		if len(levels) == 0 {
			break
		}
		if d.LevelSel < len(levels)-1 {
			d.LevelSel++
		} else {
			d.LevelSel = 0
		}

	case "enter":
		// 进入该级别的请求列表（使用标准仪表盘的请求详情，此处导航到 ReqDetail）
		if d.LevelSel >= 0 && d.LevelSel < len(levels) {
			nav = NavAction{To: NavReqDetail, ReqIndex: 0}
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
		if d.CancelFn != nil {
			d.CancelFn()
		}
		d.EventCh = nil
		d.CancelFn = nil
		nav = NavAction{To: NavTaskList}

	case "r":
		if d.RunState != nil && !d.IsRunning() {
			return d, client.GenerateReportCmd(d.RunID, server.ReportFormatJSON), nav
		}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}
	}

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
	if d == nil || width == 0 {
		return "加载中..."
	}
	rs := d.RunState

	// ── Header ──
	statusStr := "探测中"
	if rs != nil && rs.Status != server.RunStatusRunning {
		statusStr = st.Muted.Render(string(rs.Status))
	} else {
		statusStr = st.Ok.Render("探测中")
	}

	subtitle := "─"
	if rs != nil && len(rs.Levels) > 0 {
		curLevel := rs.CurrentLevel
		subtitle = fmt.Sprintf("◆ AIT   %s · 当前并发: %d   已完成 %d 级",
			"─", curLevel, len(rs.Levels))
	}

	header := renderHeader(st, width,
		"AIT  Turbo 探测 ─ "+truncate(taskName, 22),
		statusStr,
		subtitle,
		"",
	)

	// ── Context Bar ──
	var cbItems []ContextBarItem
	if d.LevelSel >= 0 && rs != nil && d.LevelSel < len(rs.Levels) {
		cbItems = CtxBar_TurboDash_Sel()
	} else {
		cbItems = CtxBar_TurboDash_NoSel()
	}
	ctxBar := RenderContextBar(st, width, cbItems)

	// ── Footer ──
	footer := renderFooter(st, width, "[s] 停止", "[b] 后台运行", "[m] 标记极限", "[r] 提前报告", "[q] 退出")

	// ── 计算高度 ──
	headerH := 2
	ctxBarH := 0
	if ctxBar != "" {
		ctxBarH = 1
	}
	footerH := 1
	splitH := 9
	progressH := 1
	divH := 3
	levelListH := height - headerH - ctxBarH - footerH - splitH - progressH - divH
	if levelListH < 3 {
		levelListH = 3
	}

	// ── 双栏（任务参数 ║ 当前级别指标）──
	leftW := (width - 2) * 45 / 100
	rightW := width - 2 - leftW - 1
	leftContent := buildTurboDashParams(rs, st, splitH-1, leftW)
	rightContent := buildTurboDashMetrics(rs, st, splitH-1, rightW)
	splitDiv := dividerLine(st, width)
	split := dualColumnLayout(st, leftContent, rightContent, leftW, rightW, splitH)

	// ── 进度条 ──
	progressLine := buildTurboProgressLine(rs, st, width)

	// ── 级别列表 ──
	levelDiv := dividerLine(st, width)
	levelList := buildLevelList(d, rs, st, width, levelListH)

	parts := []string{header, splitDiv, split, splitDiv, progressLine, levelDiv, levelList}
	if ctxBar != "" {
		parts = append(parts, ctxBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
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
		lines = append(lines, " "+labelValue(st, "成功率  ", st.MetricVal.Render(fmt.Sprintf("%.1f%%", rs.SuccessRate*100))))
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
	barW := 15
	barRendered := st.Ok.Render(strings.Repeat("█", int(ratio*float64(barW)))) +
		st.Muted.Render(strings.Repeat("░", barW-int(ratio*float64(barW))))

	levelTotal := len(rs.Levels)
	line := fmt.Sprintf(" 进度  %s  %d/%d  当前并发 %d   总进度: 已完成 %d/~? 级",
		barRendered, done, total, rs.CurrentLevel, levelTotal)
	return line
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

	// 表头
	lines = append(lines, " "+st.TableHead.Render(
		padRight("并发", 6)+padRight("成功率", 8)+padRight("TPS", 10)+
			padRight("TTFT", 10)+padRight("Cache", 8)+padRight("总耗时", 9)+"结论"))
	lines = append(lines, " "+st.Divider.Render(strings.Repeat("─", width-2)))

	for i, lv := range rs.Levels {
		if len(lines) >= maxH {
			break
		}
		isSel := i == d.LevelSel

		conclusion := st.Ok.Render("✓ 稳定")
		if !lv.Stable {
			conclusion = st.ErrStyle.Render("✗ 降级")
		}
		// 当前进行中的级别
		isCurrent := (i == len(rs.Levels)-1) && rs.Status == server.RunStatusRunning
		if isCurrent {
			conclusion = st.MetricVal.Render("🔄 进行中")
		}

		row := fmt.Sprintf(" %s%s%s%s%s%s%s",
			padRight(fmt.Sprintf("%d", lv.Concurrency), 6),
			padRight(fmt.Sprintf("%.1f%%", lv.SuccessRate*100), 8),
			padRight(fmt.Sprintf("%.1f", lv.AvgTPS), 10),
			padRight(fmtDuration(lv.AvgTTFT), 10),
			padRight(fmt.Sprintf("%.1f%%", lv.CacheHitRate*100), 8),
			padRight(fmtDuration(lv.AvgTotalTime), 9),
			conclusion,
		)

		cursorStr := "  "
		if isSel {
			cursorStr = "▶ "
		}

		var rendered string
		if isSel {
			rendered = st.TableRowSel.Render(cursorStr+row) +
				strings.Repeat(" ", max(0, width-len([]rune(cursorStr+row))-2))
		} else {
			rendered = "  " + st.TableRow.Render(row)
		}
		lines = append(lines, rendered)
	}

	for len(lines) < maxH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:maxH], "\n")
}
