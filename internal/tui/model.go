// Package tui implements the interactive terminal UI for AIT using tview.
//
// Design philosophy: lazydocker-inspired minimalism — rounded borders,
// subdued colors, bottom bar without frame, content-first layout.
//
// Layout structure (to be designed):
//   - Side panels: task list, run history
//   - Main panel: context-sensitive content (dashboard / detail / wizard)
//   - Bottom bar: keybindings + status
package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
	"github.com/yinxulai/ait/internal/tui/pages/shared"
)

// App is the top-level TUI application.  It holds all runtime state and
// async operations.  The layout (panels, flex, grid) is built by Layout()
// which is called once on startup and again when data changes.
type App struct {
	*tview.Application
	srv    server.Server
	pages  *tview.Pages
	status *tview.TextView // 1-line status bar at very bottom

	// ── State ────────────────────────────────────────────────────────────────
	mu sync.Mutex

	// Tasks (loaded once, refreshed after mutations)
	tasks   []types.TaskOverview
	taskMap map[string]*types.TaskOverview

	// Active runs: taskID → RunState (polled every 500ms while running)
	activeRuns map[string]*server.RunState

	// Selected task
	selectedTask    *types.TaskDefinition
	selectedHistory []types.TaskRunSummary

	// Dashboard (live run view)
	dashRunID    server.RunID
	dashRunState *server.RunState

	// Request detail
	reqDetailRequests []*types.RequestMetrics
	reqDetailIndex    int

	// ── Navigation ────────────────────────────────────────────────────────────
	// navLevel: 0=task list only, 1=task detail+history, 2=run detail, 3=request detail
	navLevel      int
	taskCursor    int // selected task in sidebar
	historyCursor int // selected run in history list
	requestCursor int // selected request in request list

	// Proxy
	proxyURL string

	// Wizard (3-step task creation/edit flow)
	wizard *WizardState

	// Language
	lang string // "zh" or "en"
}

// ─── Constructor ────────────────────────────────────────────────────────────

// NewApp creates an App with mock data for development.
func NewApp(srv server.Server) *App {
	a := &App{
		Application: tview.NewApplication(),
		srv:         srv,
		pages:       tview.NewPages(),
		activeRuns:  make(map[string]*server.RunState),
		taskMap:     make(map[string]*types.TaskOverview),
		lang:        "zh",
	}

	a.status = tview.NewTextView().
		SetDynamicColors(true).
		SetText("")

	a.SetInputCapture(a.globalKeys)

	// Load mock data instead of real server data
	mockData(a)

	return a
}

// Run starts the tview event loop, using mock data for development.
func Run(srv server.Server) error {
	app := NewApp(srv)
	_ = app.buildLayout()
	return app.Run()
}

// SetVersion stores the build version for display in the TUI.
func SetVersion(v string) { shared.SetAppVersion(v) }

// ─── Global keys ───────────────────────────────────────────────────────────

func (a *App) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF2:
		a.toggleLang()
		return nil
	case tcell.KeyCtrlC:
		a.Stop()
		return nil
	}
	switch event.Rune() {
	case 'q':
		a.Stop()
		return nil
	}
	return event
}

func (a *App) toggleLang() {
	if a.lang == "zh" {
		a.lang = "en"
	} else {
		a.lang = "zh"
	}
	a.rebuildLayout()
}

// rebuildLayout is called whenever state changes and the UI needs a full redraw.
func (a *App) rebuildLayout() { a.buildLayout() }

// ─── Status bar ─────────────────────────────────────────────────────────────

func (a *App) setStatus(text string) {
	a.status.SetText(" " + text)
}

// ─── Data loading (async → QueueUpdateDraw) ─────────────────────────────────

func (a *App) loadTasks() {
	go func() {
		tasks, err := a.srv.ListTasks()
		if err != nil {
			return
		}
		a.QueueUpdateDraw(func() {
			a.mu.Lock()
			a.tasks = tasks
			a.taskMap = make(map[string]*types.TaskOverview, len(tasks))
			for i := range tasks {
				a.taskMap[tasks[i].ID] = &tasks[i]
			}
			a.mu.Unlock()
			a.rebuildLayout()
		})
	}()
}

func (a *App) loadTaskDetail(taskID string) {
	go func() {
		task, err := a.srv.GetTask(taskID)
		if err != nil {
			a.QueueUpdateDraw(func() { a.setStatus(fmt.Sprintf("[red]%v", err)) })
			return
		}
		history, _ := a.srv.ListTaskRunHistory(taskID, 20)
		// Check for active run
		if rs, ok := a.srv.GetRunState(server.RunID(taskID)); ok {
			a.QueueUpdateDraw(func() { a.activeRuns[taskID] = rs })
		}
		a.QueueUpdateDraw(func() {
			a.selectedTask = &task
			a.selectedHistory = history
			a.rebuildLayout()
		})
	}()
}

func (a *App) isTaskRunning(taskID string) bool {
	rs, ok := a.activeRuns[taskID]
	return ok && rs != nil && rs.Status == server.RunStatusRunning
}

// ─── Async operations ───────────────────────────────────────────────────────

func (a *App) startRunAsync(taskID string) {
	a.setStatus("Starting run...")
	go func() {
		runID, err := a.srv.StartRun(taskID)
		if err != nil {
			a.QueueUpdateDraw(func() { a.setStatus(fmt.Sprintf("[red]%v", err)) })
			return
		}
		events, cancel := a.srv.SubscribeRunEvents(runID)

		go func() {
			defer cancel()
			tick := time.NewTicker(500 * time.Millisecond)
			defer tick.Stop()
			for range tick.C {
				rs, ok := a.srv.GetRunState(runID)
				if !ok {
					continue
				}
				a.QueueUpdateDraw(func() {
					a.activeRuns[taskID] = rs
					if runID == a.dashRunID {
						a.dashRunState = rs
					}
				})
				if rs.Status != server.RunStatusRunning && rs.Status != server.RunStatusQueued {
					a.QueueUpdateDraw(func() { a.rebuildLayout() })
					return
				}
			}
		}()

		// drain events to keep channel alive
		go func() {
			for range events {
			}
		}()

		a.QueueUpdateDraw(func() {
			a.setStatus(fmt.Sprintf("[green]Run started: %s", runID))
			a.loadTasks()
		})
	}()
}

func (a *App) stopRunAsync() {
	a.setStatus("Stopping...")
	go func() {
		if err := a.srv.StopRun(a.dashRunID); err != nil {
			a.QueueUpdateDraw(func() { a.setStatus(fmt.Sprintf("[red]%v", err)) })
			return
		}
		a.QueueUpdateDraw(func() { a.setStatus("[yellow]Run stopped") })
	}()
}

func (a *App) duplicateTaskAsync(taskID string) {
	a.setStatus("Duplicating...")
	go func() {
		if _, err := a.srv.DuplicateTask(taskID); err != nil {
			a.QueueUpdateDraw(func() { a.setStatus(fmt.Sprintf("[red]%v", err)) })
			return
		}
		a.QueueUpdateDraw(func() {
			a.setStatus("[green]Task duplicated")
			a.loadTasks()
		})
	}()
}

func (a *App) deleteTaskAsync(taskID string) {
	a.setStatus("Deleting...")
	go func() {
		if err := a.srv.DeleteTask(taskID); err != nil {
			a.QueueUpdateDraw(func() { a.setStatus(fmt.Sprintf("[red]%v", err)) })
			return
		}
		a.QueueUpdateDraw(func() {
			a.setStatus("[green]Task deleted")
			a.loadTasks()
		})
	}()
}

func (a *App) generateReportAsync(runID server.RunID) {
	a.setStatus("Generating report...")
	go func() {
		path, err := a.srv.GenerateRunReport(runID, server.ReportFormatJSON)
		if err != nil {
			a.QueueUpdateDraw(func() { a.setStatus(fmt.Sprintf("[red]%v", err)) })
			return
		}
		a.QueueUpdateDraw(func() { a.setStatus("[green]Report: " + path) })
	}()
}

// ─── Modals ─────────────────────────────────────────────────────────────────

func (a *App) confirmDeleteTask(taskID string) {
	modal := tview.NewModal().
		SetText("Delete this task?\n\nThis action cannot be undone.").
		AddButtons([]string{"Cancel", "Delete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Delete" {
				a.deleteTaskAsync(taskID)
			}
			a.pages.RemovePage("modal")
			a.rebuildLayout()
		})
	a.pages.AddPage("modal", modal, true, true)
}

// ─── Layout builder ────────────────────────────────────────────────────────
// buildLayout is implemented in layout.go.
// It constructs the full TUI layout and sets it as root.
