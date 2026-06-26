package tui

import (
	"fmt"
	"strings"
	"time"

	ge "github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
	"github.com/yinxulai/ait/internal/tui/pages/shared"
)

// divider is a long horizontal rule — clipped by panel width via wordwrap.
const divider = "────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────"

var (
	cMuted  = shared.ColorMuted
	cFaint  = shared.ColorFaint
	cBody   = shared.ColorBody
	cGreen  = shared.ColorGreen
	cCyan   = shared.ColorCyan
	cRed    = shared.ColorRed
)

// ─── Sidebar ────────────────────────────────────────────────────────────────

func (a *App) renderSidebar() {
	p := prims.side
	p.Clear()

	for idx := range a.tasks {
		t := a.tasks[idx]
		statusIcon := shared.StatusIcon(a.isTaskRunning(t.TaskDefinition.ID), false)
		mainText := fmt.Sprintf("[%s]%s [white]%s",
			cBody, statusIcon, t.TaskDefinition.Name)

		secText := fmt.Sprintf("[%s]%s  %s",
			cMuted,
			shared.ModeBadge(t.TaskDefinition.Input.Turbo),
			shortStatus(t),
		)

		p.AddItem(mainText, secText, 0, nil)
	}

	if len(a.tasks) > 0 && a.taskCursor < p.GetItemCount() {
		p.SetCurrentItem(a.taskCursor)
	}
}

func shortStatus(t types.TaskOverview) string {
	if t.LatestRun == nil {
		return "[#505050]no runs"
	}
	switch t.LatestRun.Status {
	case "completed":
		return fmt.Sprintf("[green]✓ %.0f%%", t.LatestRun.SuccessRate*100)
	case "failed":
		return "[red]✗ failed"
	case "stopped":
	return "[yellow]⊘ stopped"
	case "running":
		return "[green]▶ running"
	default:
		return fmt.Sprintf("[gray]%s", t.LatestRun.Status)
	}
}

// ─── History panel ──────────────────────────────────────────────────────────

func (a *App) renderHistory() {
	p := prims.hist
	p.Clear()

	if a.navLevel == 0 || a.selectedHistory == nil {
		p.AddItem(fmt.Sprintf("[%s]← select a task", cFaint), "", 0, nil)
		return
	}

	if len(a.selectedHistory) == 0 {
		p.AddItem(fmt.Sprintf("[%s]  no runs yet", cFaint), "", 0, nil)
		return
	}

	for _, h := range a.selectedHistory {
		statusTag := shared.RunStatusTag(h.Status)
		succ := fmt.Sprintf("[white]%5.1f%%", h.SuccessRate*100)
		if h.SuccessRate >= 0.95 {
			succ = fmt.Sprintf("[green]%5.1f%%", h.SuccessRate*100)
		} else if h.SuccessRate < 0.5 {
			succ = fmt.Sprintf("[red]%5.1f%%", h.SuccessRate*100)
		}
		mainText := fmt.Sprintf(" %s  %s", statusTag, succ)

		ago := shared.FmtRelTime(h.StartedAt)
		secText := fmt.Sprintf("  [%s]%s  [%s]QPS %5.0f  [%s]TTFT %s",
			cMuted, ago,
			cCyan, h.AvgTPS,
			cMuted, shared.FmtDuration(h.AvgTTFT))

		p.AddItem(mainText, secText, 0, nil)
	}

	if a.historyCursor < p.GetItemCount() {
		p.SetCurrentItem(a.historyCursor)
	}
}

// ─── Content dispatcher ─────────────────────────────────────────────────────

func (a *App) renderContent() {
	switch a.navLevel {
	case 0:
		a.renderWelcome()
	case 1:
		a.renderTaskDetail()
	case 2:
		a.renderRunDetail()
	case 3:
		a.renderRequestDetail()
	}
}

// ─── Welcome (level 0) ──────────────────────────────────────────────────────

func (a *App) renderWelcome() {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n\n    [%s]AIT [white]— AI Testing Toolkit",
		shared.ColorPrimary))
	sb.WriteString(fmt.Sprintf("\n    [%s]v%s\n\n", cMuted, shared.AppVersion()))
	sb.WriteString(fmt.Sprintf("    [%s]↑↓ [%s]navigate    [%s]Enter [%s]select    [%s]q [%s]quit",
		cGreen, cBody,
		cGreen, cBody,
		cGreen, cBody))
	sb.WriteString(fmt.Sprintf("\n    [%s]Tab [%s]switch focus    [%s]F2 [%s]toggle language\n\n",
		cGreen, cBody,
		cGreen, cBody))
	sb.WriteString(fmt.Sprintf("    [%s]%d tasks loaded",
		cMuted, len(a.tasks)))
	prims.cont.SetText(sb.String()).SetTitle("").SetTitleColor(0)
}

// ─── Task Detail (level 1) ──────────────────────────────────────────────────

func (a *App) renderTaskDetail() {
	if a.selectedTask == nil {
		prims.cont.SetText("[#505050]No task selected").SetTitle(" Error ")
		return
	}
	t := a.selectedTask

	var sb strings.Builder

	// ── Breadcrumb ──
	sb.WriteString(fmt.Sprintf("[%s]Tasks [%s]/ [white]%s\n", cMuted, cFaint, t.Name))
	sb.WriteString(fmt.Sprintf("[%s]%s\n\n", cFaint, divider))

	// ── Config ──
	pair := func(l, v string) string {
		return fmt.Sprintf("[%s]%s [white]%s", cBody, l, v)
	}
	items := []string{
		pair("Mode:", shared.ModeBadge(t.Input.Turbo)),
		pair("Protocol:", t.Input.Protocol),
		pair("Model:", t.Input.Model),
		pair("Endpoint:", t.Input.EndpointURL),
		pair("Concurrency:", fmt.Sprintf("%d", t.Input.Concurrency)),
		pair("Total Requests:", fmt.Sprintf("%d", t.Input.Count)),
		pair("Timeout:", t.Input.Timeout.String()),
		pair("Stream:", fmt.Sprintf("%v", t.Input.Stream)),
	}
	for i := 0; i < len(items); i += 2 {
		line := fmt.Sprintf("  %-40s", items[i])
		if i+1 < len(items) {
			line += fmt.Sprintf("  %s", items[i+1])
		}
		sb.WriteString(line + "\n")
	}

	if t.Input.Turbo {
		sb.WriteString(fmt.Sprintf("\n[%s]%s\n", cFaint, divider))
		sb.WriteString(fmt.Sprintf("[%s]── Turbo Ramp Config ──\n", cMuted))
		titems := []string{
			pair("Ramp:", fmt.Sprintf("%d → %d  (step: +%d, %d reqs/level)",
				t.Input.TurboConfig.InitConcurrency,
				t.Input.TurboConfig.MaxConcurrency,
				t.Input.TurboConfig.StepSize,
				t.Input.TurboConfig.LevelRequests)),
			pair("Min Success Rate:", fmt.Sprintf("%.0f%%", t.Input.TurboConfig.MinSuccessRate*100)),
		}
		for i := 0; i < len(titems); i += 2 {
			line := fmt.Sprintf("  %-40s", titems[i])
			if i+1 < len(titems) {
				line += fmt.Sprintf("  %s", titems[i+1])
			}
			sb.WriteString(line + "\n")
		}
	}

	sb.WriteString(fmt.Sprintf("\n[%s]%s\n", cFaint, divider))
	sb.WriteString(fmt.Sprintf("[%s]← select a run from the History panel\n", cMuted))

	prims.cont.SetText(sb.String()).
		SetTitle(fmt.Sprintf(" %s ", t.Name)).
		SetTitleColor(0)
}

// ─── Run Detail (level 2) ───────────────────────────────────────────────────

func (a *App) renderRunDetail() {
	if a.dashRunState == nil {
		prims.cont.SetText("[#505050]No run selected").
			SetTitle(" Error ")
		return
	}
	rs := a.dashRunState

	var sb strings.Builder

	// ── Breadcrumb ──
	sb.WriteString(fmt.Sprintf("[%s]Run [white]%s\n",
		cMuted, rs.RunID))
	sb.WriteString(fmt.Sprintf("[%s]%s\n\n", cFaint, divider))

	// ── Metrics card ──
	elapsed := time.Since(rs.StartedAt)
	if rs.FinishedAt != nil {
		elapsed = rs.FinishedAt.Sub(rs.StartedAt)
	}
	tag := shared.RunStatusTag(string(rs.Status))
	prog := shared.ProgressBar(rs.DoneReqs, rs.TotalReqs, 40)

	p := func(l, v string) string {
		return fmt.Sprintf("[%s]%s [white]%s", cBody, l, v)
	}

	sb.WriteString(fmt.Sprintf("  %-35s  %s\n",
		p("Status:", tag),
		p("Elapsed:", shared.FmtDuration(elapsed))))
	sb.WriteString(fmt.Sprintf("  %-35s  %s\n",
		p("Mode:", shared.ModeBadge(rs.Mode == ge.ModeTurbo)),
		p("Progress:", fmt.Sprintf("%d/%d %s", rs.DoneReqs, rs.TotalReqs, prog))))

	sb.WriteString(fmt.Sprintf("\n  %-35s  %s\n",
		p("QPS:", fmt.Sprintf("%.0f", rs.AvgTPS)),
		p("RPM:", fmt.Sprintf("%.0f", rs.RPM))))
	sb.WriteString(fmt.Sprintf("  %-35s  %s\n",
		p("TPM:", fmt.Sprintf("%.0f", rs.TPM)),
		p("Success:", fmt.Sprintf("%.1f%% (%d/%d)", rs.SuccessRate*100, rs.SuccessReqs, rs.DoneReqs))))
	sb.WriteString(fmt.Sprintf("  %-35s\n",
		p("TTFT:", shared.FmtDuration(rs.AvgTTFT))))

	if rs.Mode == ge.ModeTurbo {
		if lvls, ok := rs.ModeState[ge.ModeStateKeyLevels]; ok {
			cl, _ := rs.ModeState[ge.ModeStateKeyCurrentLevel].(int)
			sb.WriteString(fmt.Sprintf("  %-35s\n", p("Turbo:", fmt.Sprintf("Level %d/%v", cl, lvls))))
		}
	}
	if rs.ErrorMsg != "" {
		sb.WriteString(fmt.Sprintf("\n  [red]%s\n", rs.ErrorMsg))
	}

	// ── Request list ──
	reqs := rs.Requests
	if reqs == nil {
		reqs = a.reqDetailRequests
	}
	sb.WriteString(fmt.Sprintf("\n[%s]%s\n", cFaint, divider))
	sb.WriteString(fmt.Sprintf("[%s]── Requests (%d) ──\n", cMuted, len(reqs)))
	if reqs == nil || len(reqs) == 0 {
		sb.WriteString(fmt.Sprintf("[%s]  No requests\n", cFaint))
	} else {
		maxShow := len(reqs)
		if maxShow > 200 {
			maxShow = 200
		}
		for i := 0; i < maxShow; i++ {
			r := reqs[i]
			cursor := " "
			if i == a.requestCursor {
				cursor = "[green]▶"
			}
			status := "[green]✓"
			if !r.Success {
				status = "[red]✗"
			}
			latStr := shared.FmtDuration(r.TotalTime)
			if !r.Success {
				latStr = "[red]" + latStr
			}
			ttftStr := shared.FmtDuration(r.TTFT)
			sb.WriteString(fmt.Sprintf("  %s %-3d %s  [%s]%-8s  [%s]TTFT %s  [%s]in:%d out:%d cached:%d\n",
				cursor, r.Index, status, cBody, latStr, cMuted, ttftStr, cMuted,
				r.PromptTokens, r.CompletionTokens, r.CachedTokens))
		}
		if len(reqs) > maxShow {
			sb.WriteString(fmt.Sprintf("[%s]  ... %d more\n", cFaint, len(reqs)-maxShow))
		}
	}

	title := fmt.Sprintf(" Run %s ", truncateRunID(string(rs.RunID)))
	prims.cont.SetText(sb.String()).SetTitle(title).SetTitleColor(0)
}

// ─── Request Detail (level 3) ───────────────────────────────────────────────

func (a *App) renderRequestDetail() {
	reqs := a.reqDetailRequests
	if a.dashRunState != nil && a.dashRunState.Requests != nil {
		reqs = a.dashRunState.Requests
	}
	if reqs == nil || a.requestCursor >= len(reqs) {
		prims.cont.SetText("[#505050]No request selected").SetTitle(" Error ")
		return
	}
	r := reqs[a.requestCursor]

	var sb strings.Builder

	// ── Breadcrumb ──
	runID := string(a.dashRunID)
	sb.WriteString(fmt.Sprintf("[%s]Run [white]%s [%s]/ [%s]Req #%d/%d\n",
		cMuted, truncateRunID(runID),
		cFaint, cMuted, r.Index, len(reqs)))
	sb.WriteString(fmt.Sprintf("[%s]%s\n\n", cFaint, divider))

	// ── Metrics ──
	status := "[green]✓ SUCCESS"
	if !r.Success {
		status = "[red]✗ FAILED"
	}
	p := func(l, v string) string {
		return fmt.Sprintf("[%s]%s [white]%s", cBody, l, v)
	}

	sb.WriteString(fmt.Sprintf("  %-38s  %s\n",
		p("Status:", status),
		p("Total Latency:", shared.FmtDuration(r.TotalTime))))
	sb.WriteString(fmt.Sprintf("  %-38s  %s\n",
		p("TTFT:", shared.FmtDuration(r.TTFT)),
		p("TPS:", fmt.Sprintf("%.0f", r.TPS))))
	sb.WriteString(fmt.Sprintf("  %-38s  %s\n",
		p("Tokens:", fmt.Sprintf("in:%d out:%d cached:%d (%.0f%%)",
			r.PromptTokens, r.CompletionTokens, r.CachedTokens, r.CacheHitRate*100)),
		p("Target IP:", r.TargetIP)))
	sb.WriteString(fmt.Sprintf("  %-38s\n",
		p("Network:", fmt.Sprintf("DNS %s / TCP %s / TLS %s",
			shared.FmtDuration(r.DNSTime), shared.FmtDuration(r.ConnectTime), shared.FmtDuration(r.TLSTime)))))

	if r.ErrorMessage != "" {
		sb.WriteString(fmt.Sprintf("\n[%s]%s\n[red]%s\n", cFaint, divider, r.ErrorMessage))
	}

	// ── Body ──
	if r.RequestBody != "" {
		sb.WriteString(fmt.Sprintf("\n[%s]%s\n", cFaint, divider))
		sb.WriteString(fmt.Sprintf("[%s]── Request Body ──\n[%s]%s\n",
			cMuted, cBody, r.RequestBody))
	}
	if r.ResponseBody != "" {
		sb.WriteString(fmt.Sprintf("\n[%s]%s\n", cFaint, divider))
		sb.WriteString(fmt.Sprintf("[%s]── Response Body ──\n[%s]%s\n",
			cMuted, cBody, r.ResponseBody))
	}

	title := fmt.Sprintf(" Req #%d ", r.Index)
	prims.cont.SetText(sb.String()).SetTitle(title).SetTitleColor(0)
}

// ─── Bottom bar ─────────────────────────────────────────────────────────────

func (a *App) renderBottombar() {
	var keys []string
	var stat string

	switch a.navLevel {
	case 0:
		keys = []string{
			"[green]↑↓[white]:tasks",
			"[green]Enter[white]:select",
			"[green]n[white]:new",
			"[green]Tab[white]:history",
			"[green]F2[white]:lang",
			"[green]q[white]:quit",
		}
		stat = fmt.Sprintf("[%s]%d tasks · AIT v%s",
			cMuted, len(a.tasks), shared.AppVersion())
	case 1:
		keys = []string{
			"[green]↑↓[white]:runs",
			"[green]Enter[white]:inspect",
			"[green]←[white]:tasks",
			"[green]r[white]:start",
			"[green]Tab[white]:detail",
			"[green]q[white]:quit",
		}
		if a.selectedTask != nil {
			stat = fmt.Sprintf("[%s]%s · %s",
				cMuted, a.selectedTask.Name, shared.ModeBadge(a.selectedTask.Input.Turbo))
		}
	case 2:
		keys = []string{
			"[green]↑↓[white]:requests",
			"[green]Enter[white]:detail",
			"[green]←[white]:runs",
			"[green]g[white]:report",
			"[green]s[white]:stop",
			"[green]Tab[white]:tasks",
		}
		if a.dashRunState != nil {
			stat = fmt.Sprintf("[%s]Run %s · %s",
				cMuted, truncateRunID(string(a.dashRunState.RunID)),
				shared.RunStatusTag(string(a.dashRunState.Status)))
		}
	case 3:
		keys = []string{
			"[green]←→[white]:prev/next",
			"[green]←[white]:back",
			"[green]↑↓[white]:scroll",
			"[green]Tab[white]:tasks",
		}
		stat = fmt.Sprintf("[%s]Req #%d", cMuted, a.requestCursor+1)
	}

	keyStr := strings.Join(keys, fmt.Sprintf(" [%s]│ ", cFaint))
	prims.bottomKeys.SetText(" " + keyStr)
	if stat != "" {
		prims.bottomStat.SetText(stat + " ")
	} else {
		prims.bottomStat.SetText("")
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func truncateRunID(id string) string {
	if len(id) > 12 {
		return id[:12] + "..."
	}
	return id
}
