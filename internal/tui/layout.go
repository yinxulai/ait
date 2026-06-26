package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// ─── Layout primitives (persistent) ─────────────────────────────────────────

// layout primitives hold the persistent tview widgets. They are created once
// in buildLayout() and their content is updated by render*() functions.
type layoutPrims struct {
	root  *tview.Flex   // vertical: main + bottombar
	main  *tview.Flex   // horizontal: tasks | history | content
	side  *tview.List    // task list sidebar
	hist  *tview.List    // run history sidebar (secondary menu for selected task)
	cont  *tview.TextView // main content area

	bottom     *tview.Flex       // bottom bar horizontal
	bottomKeys *tview.TextView   // keybinding hints
	bottomStat *tview.TextView   // status line
}

var prims layoutPrims

// ─── buildLayout ────────────────────────────────────────────────────────────

func (a *App) buildLayout() error {
	// One-time widget creation
	if prims.cont == nil {
		a.createWidgets()
	}

	// Render content into widgets
	a.renderAll()
	return nil
}

func (a *App) createWidgets() {
	// ── Sidebar: task list ──────────────────────────────────────────────
	prims.side = tview.NewList().
		ShowSecondaryText(false)
	prims.side.
		SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetTitle(" Tasks ").
		SetTitleColor(tcell.ColorGray).
		SetBorderPadding(0, 0, 1, 1)

	// ── History: run list for selected task ─────────────────────────────
	prims.hist = tview.NewList().
		ShowSecondaryText(false)
	prims.hist.
		SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetTitle(" History ").
		SetTitleColor(tcell.ColorGray).
		SetBorderPadding(0, 0, 1, 1)

	// ── Main content ────────────────────────────────────────────────────
	prims.cont = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	prims.cont.
		SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetBorderPadding(0, 0, 1, 1)

	// ── Bottom bar ──────────────────────────────────────────────────────
	prims.bottomKeys = tview.NewTextView().
		SetDynamicColors(true)
	prims.bottomStat = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)

	prims.bottom = tview.NewFlex().
		AddItem(prims.bottomKeys, 0, 3, false).
		AddItem(prims.bottomStat, 0, 1, false)

	// ── Main flex: tasks(2) | history(2) | content(6)  proportional ───
	prims.main = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(prims.side, 0, 2, true).
		AddItem(prims.hist, 0, 2, false).
		AddItem(prims.cont, 0, 6, false)

	// ── Root: main | bottom bar (1 line) ───────────────────────────────
	prims.root = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(prims.main, 0, 1, true).
		AddItem(prims.bottom, 1, 0, false)

	// Wire input capture per widget for navigation
	prims.side.SetInputCapture(a.sideKeys)
	prims.hist.SetInputCapture(a.historyKeys)
	prims.cont.SetInputCapture(a.contentKeys)

	a.SetRoot(prims.root, true)
}

// ─── Render dispatcher ──────────────────────────────────────────────────────

func (a *App) renderAll() {
	a.renderSidebar()
	a.renderHistory()
	a.renderContent()
	a.renderBottombar()
}

// ─── Sidebar input ──────────────────────────────────────────────────────────

func (a *App) sideKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyUp:
		if a.taskCursor > 0 {
			a.taskCursor--
		}
		a.renderAll()
		return nil
	case tcell.KeyDown:
		if a.taskCursor < len(a.tasks)-1 {
			a.taskCursor++
		}
		a.renderAll()
		return nil
	case tcell.KeyEnter:
		a.selectTask(a.tasks[a.taskCursor].TaskDefinition.ID)
		return nil
	case tcell.KeyTab:
		a.focusHistory()
		return nil
	}
	return event
}

// ─── History input (runs for selected task) ────────────────────────────────

func (a *App) historyKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyUp:
		if a.historyCursor > 0 {
			a.historyCursor--
		}
		a.renderAll()
		return nil
	case tcell.KeyDown:
		if a.historyCursor < len(a.selectedHistory)-1 {
			a.historyCursor++
		}
		a.renderAll()
		return nil
	case tcell.KeyEnter:
		if a.navLevel >= 1 && len(a.selectedHistory) > 0 {
			a.selectHistory()
		}
		return nil
	case tcell.KeyTab:
		a.focusContent()
		return nil
	case tcell.KeyLeft, tcell.KeyBackspace, tcell.KeyBackspace2:
		a.navLevel = 0
		a.focusSidebar()
		a.renderAll()
		return nil
	case tcell.KeyEscape:
		a.resetView()
		return nil
	}
	return event
}

// ─── Content input ──────────────────────────────────────────────────────────

func (a *App) contentKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyUp:
		a.contentMoveUp()
		a.renderAll()
		return nil
	case tcell.KeyDown:
		a.contentMoveDown()
		a.renderAll()
		return nil
	case tcell.KeyEnter:
		a.contentEnter()
		return nil
	case tcell.KeyLeft, tcell.KeyBackspace, tcell.KeyBackspace2:
		a.contentBack()
		return nil
	case tcell.KeyTab:
		a.focusSidebar()
		return nil
	case tcell.KeyEscape:
		a.resetView()
		return nil
	}
	return event
}

// ─── Navigation actions ─────────────────────────────────────────────────────

func (a *App) selectTask(taskID string) {
	a.loadTaskDetailMock(taskID)
	a.navLevel = 1
	a.historyCursor = 0
	a.requestCursor = 0
	a.renderAll()
	a.focusHistory()
}

func (a *App) selectHistory() {
	if len(a.selectedHistory) == 0 {
		return
	}
	a.navLevel = 2
	a.requestCursor = 0
	h := a.selectedHistory[a.historyCursor]
	a.reqDetailRequests = mockRunDetailRequests[h.RunID]
	a.dashRunID = server.RunID(h.RunID)
	a.dashRunState = a.summaryToRunState(h)
	a.renderAll()
	a.focusContent()
}

func (a *App) contentMoveUp() {
	switch a.navLevel {
	case 2:
		if a.requestCursor > 0 {
			a.requestCursor--
		}
	}
}

func (a *App) contentMoveDown() {
	switch a.navLevel {
	case 2:
		if rs := a.dashRunState; rs != nil && rs.Requests != nil {
			if a.requestCursor < len(rs.Requests)-1 {
				a.requestCursor++
			}
		} else if reqs := a.reqDetailRequests; reqs != nil {
			if a.requestCursor < len(reqs)-1 {
				a.requestCursor++
			}
		}
	}
}

func (a *App) contentEnter() {
	switch a.navLevel {
	case 2:
		// Enter on a request → go to request detail
		a.navLevel = 3
		a.renderAll()
	}
}

func (a *App) contentBack() {
	switch a.navLevel {
	case 1:
		// Back to task list
		a.navLevel = 0
		a.focusSidebar()
		a.renderAll()
	case 2:
		// Back to history panel
		a.focusHistory()
		a.renderAll()
	case 3:
		a.navLevel = 2
		a.renderAll()
	}
}

func (a *App) resetView() {
	a.navLevel = 0
	a.selectedTask = nil
	a.selectedHistory = nil
	a.dashRunID = ""
	a.dashRunState = nil
	a.reqDetailRequests = nil
	a.historyCursor = 0
	a.requestCursor = 0
	a.focusSidebar()
	a.renderAll()
}

// ─── Focus helpers ──────────────────────────────────────────────────────────

func (a *App) focusSidebar() {
	a.SetFocus(prims.side)
}

func (a *App) focusHistory() {
	a.SetFocus(prims.hist)
}

func (a *App) focusContent() {
	a.SetFocus(prims.cont)
}

// ─── Mock task detail loading ───────────────────────────────────────────────

func (a *App) loadTaskDetailMock(taskID string) {
	if to, ok := a.taskMap[taskID]; ok {
		a.selectedTask = &to.TaskDefinition
	}
	if h, ok := taskHistories[taskID]; ok {
		a.selectedHistory = h
	} else {
		a.selectedHistory = nil
	}
}

// summaryToRunState converts a TaskRunSummary to a minimal RunState for display.
func (a *App) summaryToRunState(s types.TaskRunSummary) *server.RunState {
	return &server.RunState{
		Mode:       server.RunMode(s.Mode),
		RunID:      server.RunID(s.RunID),
		TaskID:     s.TaskID,
		Status:     server.RunStatus(s.Status),
		StartedAt:  s.StartedAt,
		FinishedAt: &s.FinishedAt,
		SuccessRate: s.SuccessRate,
		AvgTTFT:    s.AvgTTFT,
		AvgTPS:     s.AvgTPS,
		CacheHitRate: s.CacheHitRate,
		RPM:        s.RPM,
		TPM:        s.TPM,
		Requests:   mockRunDetailRequests[s.RunID],
		ModeState:  map[string]any{},
	}
}
