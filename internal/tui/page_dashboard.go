package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
)

// dashboardState 仪表盘页的局部状态。
type dashboardState struct {
	runID    server.RunID
	taskID   string
	eventCh  <-chan server.Event // nil 表示已后台/已结束
	cancelFn server.CancelFunc
	runState *server.RunState
	reqSel   int // 选中请求的 index（-1 = 无选中）
	reqOff   int // 请求列表滚动偏移
}

// isRunning 判断仪表盘内的运行是否仍在进行中。
func (d *dashboardState) isRunning() bool {
	if d == nil || d.runState == nil {
		return false
	}
	return d.runState.Status == server.RunStatusRunning
}

// ─── 按键处理 ─────────────────────────────────────────────────────────────────

func (m *Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.dash == nil {
		m.view = viewTaskList
		return m, nil
	}
	d := m.dash
	var reqs []*server.RequestMetrics
	if d.runState != nil {
		reqs = d.runState.Requests
	}

	switch msg.String() {
	case "up", "k":
		if d.reqSel > 0 {
			d.reqSel--
		} else if len(reqs) > 0 {
			d.reqSel = len(reqs) - 1
		}
		m.adjustReqOffset()

	case "down", "j":
		if d.reqSel < len(reqs)-1 {
			d.reqSel++
		} else {
			d.reqSel = 0
		}
		m.adjustReqOffset()

	case "enter":
		if d.reqSel >= 0 && d.reqSel < len(reqs) {
			m.reqDetail = &reqDetailState{
				runID:    d.runID,
				requests: reqs,
				index:    d.reqSel,
			}
			m.view = viewReqDetail
		}

	case "s":
		if d.isRunning() {
			return m, m.client.StopRunCmd(d.runID)
		}

	case "b":
		// 后台运行：取消订阅，返回任务列表，保留 dash 状态
		if d.cancelFn != nil {
			d.cancelFn()
		}
		d.eventCh = nil
		d.cancelFn = nil
		m.view = viewTaskList
		m.status = fmt.Sprintf("运行 %s 已转入后台", d.runID)

	case "r":
		// 生成报告
		if d.runState != nil && !d.isRunning() {
			return m, m.client.GenerateReportCmd(d.runID, server.ReportFormatJSON)
		}

	case "left", "esc":
		if !d.isRunning() {
			// 运行已结束，直接返回任务详情
			if d.cancelFn != nil {
				d.cancelFn()
			}
			m.dash = nil
			m.view = viewTaskDetail
		}

	case "q":
		return m, tea.Quit
	}

	return m, nil
}

// adjustReqOffset 根据 reqSel 调整列表的可见窗口。
func (m *Model) adjustReqOffset() {
	if m.dash == nil {
		return
	}
	visH := m.height - 10
	if visH < 5 {
		visH = 5
	}
	sel := m.dash.reqSel
	off := m.dash.reqOff
	if sel < off {
		off = sel
	} else if sel >= off+visH {
		off = sel - visH + 1
	}
	m.dash.reqOff = off
}

// ─── 渲染 ─────────────────────────────────────────────────────────────────────

func (m *Model) renderDashboard() string {
	if m.dash == nil || m.width == 0 {
		return "加载中..."
	}
	d := m.dash
	rs := d.runState

	statusStr := "等待中"
	if rs != nil {
		switch rs.Status {
		case server.RunStatusRunning:
			statusStr = m.styles.ok.Render("运行中")
		case server.RunStatusCompleted:
			statusStr = m.styles.ok.Render("已完成")
		case server.RunStatusFailed:
			statusStr = m.styles.errStyle.Render("失败")
		case server.RunStatusStopped:
			statusStr = m.styles.muted.Render("已停止")
		}
	}
	header := m.renderHeader("AIT  仪表盘", statusStr)

	var cbItems []contextBarItem
	if d.reqSel >= 0 {
		cbItems = contextBarItems_dashboard_sel()
	} else {
		cbItems = contextBarItems_dashboard_nosel()
	}
	contextBar := m.renderContextBar(cbItems)

	var footerRight string
	if rs != nil && rs.TotalReqs > 0 {
		footerRight = fmt.Sprintf("%d/%d 请求", rs.DoneReqs, rs.TotalReqs)
	}
	footer := m.renderFooter("[s] 停止", "[b] 后台", "[r] 报告", footerRight)

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

	leftContent := m.buildDashLeft(contentH, leftW)
	rightContent := m.buildDashRight(contentH)
	mid := m.dualColumnLayout(leftContent, rightContent, leftW, rightW, contentH)

	parts := []string{header, mid}
	if contextBar != "" {
		parts = append(parts, contextBar)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func (m *Model) buildDashLeft(maxH, width int) string {
	d := m.dash
	rs := d.runState
	var lines []string

	// ── 汇总指标 ──
	lines = append(lines, m.styles.sectionHead.Render("实时指标"))
	lines = append(lines, "")

	if rs == nil {
		lines = append(lines, m.styles.muted.Render("  等待数据..."))
		return strings.Join(lines, "\n")
	}

	// 进度条
	pbW := width - 20
	if pbW < 10 {
		pbW = 10
	}
	pct := 0.0
	if rs.TotalReqs > 0 {
		pct = float64(rs.DoneReqs) * 100 / float64(rs.TotalReqs)
	}
	pb := progressBar(rs.DoneReqs, rs.TotalReqs, pbW)
	lines = append(lines, fmt.Sprintf("  进度  %s %5.1f%%", pb, pct))
	lines = append(lines, "")

	lines = append(lines, row(m, "总请求数    ", fmt.Sprintf("%d", rs.TotalReqs)))
	lines = append(lines, row(m, "已完成      ", fmt.Sprintf("%d", rs.DoneReqs)))
	lines = append(lines, row(m, "成功        ", m.styles.ok.Render(fmt.Sprintf("%d", rs.SuccessReqs))))
	lines = append(lines, row(m, "失败        ", m.styles.errStyle.Render(fmt.Sprintf("%d", rs.FailedReqs))))
	lines = append(lines, row(m, "成功率      ", fmt.Sprintf("%.2f%%", rs.SuccessRate)))
	lines = append(lines, "")
	lines = append(lines, m.styles.sectionHead.Render("性能指标"))
	lines = append(lines, "")
	lines = append(lines, row(m, "平均 TPS    ", m.styles.metricVal.Render(fmt.Sprintf("%.2f tok/s", rs.AvgTPS))))
	lines = append(lines, row(m, "平均 TTFT   ", m.styles.metricVal.Render(fmt.Sprintf("%.0f ms", float64(rs.AvgTTFT.Milliseconds())))))
	lines = append(lines, row(m, "缓存命中率  ", fmt.Sprintf("%.2f%%", rs.CacheHitRate)))

	if rs.Mode == "turbo" && len(rs.Levels) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.styles.sectionHead.Render("Turbo 并发探测"))
		lines = append(lines, "")
		for i, lv := range rs.Levels {
			sel := ""
			if i == rs.CurrentLevel {
				sel = m.styles.ok.Render("▶")
			} else {
				sel = " "
			}
			stableStr := m.styles.muted.Render("探测中")
			if lv.Stable {
				stableStr = m.styles.ok.Render("稳定")
			} else if lv.StopReason != "" {
				stableStr = m.styles.errStyle.Render("停止")
			}
			lines = append(lines, fmt.Sprintf("%s 并发%3d  TPS %5.1f  成功率 %5.1f%%  %s",
				sel, lv.Concurrency, lv.AvgTPS, lv.SuccessRate, stableStr))
		}
	}

	if rs.ErrorMsg != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.errStyle.Render("错误: "+truncate(rs.ErrorMsg, width-10)))
	}

	return strings.Join(lines, "\n")
}

func (m *Model) buildDashRight(maxH int) string {
	d := m.dash
	rs := d.runState
	var lines []string

	lines = append(lines, m.styles.sectionHead.Render("请求列表"))
	lines = append(lines, "")
	lines = append(lines, m.styles.tableHead.Render(
		fmt.Sprintf("  %-4s %-5s %8s %8s %8s %-6s", "#", "状态", "耗时", "TTFT", "TPS", "Token"),
	))
	lines = append(lines, m.styles.muted.Render(strings.Repeat("─", 56)))

	if rs == nil || len(rs.Requests) == 0 {
		lines = append(lines, m.styles.muted.Render("  暂无请求..."))
		return strings.Join(lines, "\n")
	}

	visH := maxH - len(lines) - 1
	if visH < 1 {
		visH = 1
	}

	start := d.reqOff
	if start < 0 {
		start = 0
	}
	end := start + visH
	if end > len(rs.Requests) {
		end = len(rs.Requests)
	}

	for i := start; i < end; i++ {
		r := rs.Requests[i]
		statusIcon := m.styles.ok.Render("✓")
		if !r.Success {
			statusIcon = m.styles.errStyle.Render("✗")
		}
		line := fmt.Sprintf("  %3d %s %7dms %7dms %7.1f %-6d",
			r.Index+1,
			statusIcon,
			r.TotalTime.Milliseconds(),
			r.TTFT.Milliseconds(),
			r.TPS,
			r.CompletionTokens,
		)
		if i == d.reqSel {
			lines = append(lines, m.styles.tableRowSel.Render(line))
		} else {
			lines = append(lines, line)
		}
	}

	// 滚动提示
	if len(rs.Requests) > visH {
		lines = append(lines, m.styles.muted.Render(
			fmt.Sprintf("  %d/%d 请求  [↑↓] 滚动", len(rs.Requests), len(rs.Requests))))
	}

	return strings.Join(lines, "\n")
}
