package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// StandardDashboardPage 标准模式运行仪表盘（v1）。
type StandardDashboardPage struct {
	srv     server.Server
	run     *types.TaskRunSummary
	reqs    []types.RequestMetrics
	total   int
	root    *tview.Flex
	rTable  *tview.Table
	statsTv *tview.TextView
}

// NewStandardDashboardPage 创建标准模式仪表盘。
func NewStandardDashboardPage(
	srv server.Server,
	run *types.TaskRunSummary,
	reqs []types.RequestMetrics,
	totalReqs int,
) *StandardDashboardPage {
	p := &StandardDashboardPage{
		srv:   srv,
		run:   run,
		reqs:  reqs,
		total: totalReqs,
	}

	// Header
	header := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	shortID := run.RunID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	isCompleted := run.Status == "completed" || run.Status == "failed" || run.Status == "stopped"
	statusLabel := fmt.Sprintf("Completed %d%%", 100)
	if !isCompleted {
		statusLabel = "Running..."
	}
	header.SetText(fmt.Sprintf("Run Dashboard - %s [%s]", shortID, statusLabel))

	// 统计文本
	p.statsTv = tview.NewTextView().
		SetDynamicColors(true)
	elapsed := time.Since(run.StartedAt)
	if isCompleted {
		elapsed = run.FinishedAt.Sub(run.StartedAt)
	}
	p.statsTv.SetText(formatStats(reqs, totalReqs, isCompleted, elapsed))

	// 请求表
	p.rTable = newSelectableTable(" Requests ")
	p.rebuildRequestTable()

	// Footer
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	footer.SetText("[::b]s=停止  g=报告  ↑↓=选择  Esc=返回[::-]")

	// 布局
	content := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(p.statsTv, 0, 4, false).
		AddItem(p.rTable, 0, 6, true)

	p.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(footer, 1, 0, false)

	return p
}

// SetRequests 更新请求列表（由外部轮询刷新）。
func (p *StandardDashboardPage) SetRequests(reqs []types.RequestMetrics, totalReqs int) {
	p.reqs = reqs
	p.total = totalReqs
	isCompleted := p.run.Status == "completed" || p.run.Status == "failed" || p.run.Status == "stopped"
	elapsed := time.Since(p.run.StartedAt)
	if isCompleted {
		elapsed = p.run.FinishedAt.Sub(p.run.StartedAt)
	}
	p.statsTv.SetText(formatStats(reqs, totalReqs, isCompleted, elapsed))
	oldIdx := selectedDataIndex(p.rTable)
	p.rebuildRequestTable()
	selectDataIndex(p.rTable, oldIdx)
}

func (p *StandardDashboardPage) rebuildRequestTable() {
	rows := make([][]string, 0, len(p.reqs))
	for _, r := range p.reqs {
		rows = append(rows, formatReqRow(r))
	}
	setTableRows(p.rTable, requestColumns(), rows)
}

// ─── Page 接口实现 ─────────────────────────────────────────────────────────────

func (p *StandardDashboardPage) Name() string                 { return "standard_dashboard" }
func (p *StandardDashboardPage) Primitive() tview.Primitive   { return p.root }
func (p *StandardDashboardPage) FocusTarget() tview.Primitive { return p.rTable }
func (p *StandardDashboardPage) OnActivate()                  {}
func (p *StandardDashboardPage) OnDeactivate()                {}

func (p *StandardDashboardPage) HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		router.GoBack()
		return nil
	}
	switch event.Rune() {
	case 's':
		go func() { _ = p.srv.StopRun(server.RunID(p.run.RunID)) }()
		return nil
	}
	return event // ↑↓ 等由 tview Table 原生处理
}
