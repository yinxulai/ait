package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// ─── 包级版本 ──────────────────────────────────────────────────────────────────

var version string

// SetVersion 设置 TUI 显示的版本号。
func SetVersion(v string) { version = v }

// ─── 尺寸常量 ──────────────────────────────────────────────────────────────────

const (
	minTermWidth  = 100
	minTermHeight = 14
)

// ─── tuiState ──────────────────────────────────────────────────────────────────

type tuiState struct {
	app         *tview.Application
	router      *PageRouter
	srv         server.Server
	mainPage    *MainPage
	warningPage *WarningPage
	stopCh      chan struct{}
}

func (s *tuiState) stop() { close(s.stopCh) }

// ─── Simulator (v1) ───────────────────────────────────────────────────────────

// NewSimulator 返回三个只读 channel，用于 v1 模拟数据推送。
func NewSimulator() (
	<-chan []types.TaskOverview,
	<-chan []types.TaskRunSummary,
	<-chan []types.RequestMetrics,
) {
	tasksCh := make(chan []types.TaskOverview, 1)
	runsCh := make(chan []types.TaskRunSummary, 1)
	reqsCh := make(chan []types.RequestMetrics, 1)

	// 构造模拟数据
	tasks := generateTaskOverviews()
	runs := generateRunSummaries()
	reqs := generateRequestMetrics(50)

	tasksCh <- tasks
	runsCh <- runs
	reqsCh <- reqs

	return tasksCh, runsCh, reqsCh
}

// ─── 数据管道 ──────────────────────────────────────────────────────────────────

func (s *tuiState) startUpdateLoop(
	tasksCh <-chan []types.TaskOverview,
	runsCh <-chan []types.TaskRunSummary,
	reqsCh <-chan []types.RequestMetrics,
) {
	go func() {
		for {
			select {
			case tasks := <-tasksCh:
				s.app.QueueUpdateDraw(func() { s.mainPage.RefreshTasks(tasks) })
			case runs := <-runsCh:
				s.app.QueueUpdateDraw(func() { s.mainPage.RefreshRuns(runs) })
			case reqs := <-reqsCh:
				s.app.QueueUpdateDraw(func() { s.mainPage.RefreshRequests(reqs) })
			case <-s.stopCh:
				return
			}
		}
	}()
}

// ─── 按键 + 尺寸检测 ──────────────────────────────────────────────────────────

func (s *tuiState) setupKeybindings() {
	s.app.SetInputCapture(s.globalKeyHandler)
	s.app.SetBeforeDrawFunc(s.beforeDrawCheck)
}

func (s *tuiState) globalKeyHandler(event *tcell.EventKey) *tcell.EventKey {
	cur := s.router.ActivePage()
	if cur != nil {
		return cur.HandleKey(event, s.router)
	}
	return event
}

func (s *tuiState) beforeDrawCheck(screen tcell.Screen) bool {
	w, h := screen.Size()
	tooSmall := w < minTermWidth || h < minTermHeight
	overlayActive := s.router.ActivePage() != nil && s.router.ActivePage().Name() == "warning"

	if tooSmall && !overlayActive {
		s.warningPage.UpdateText(w, h)
		go func() {
			s.app.QueueUpdateDraw(func() {
				s.router.ShowOverlay(s.warningPage)
				s.app.SetFocus(s.warningPage.FocusTarget())
			})
		}()
	} else if !tooSmall && overlayActive {
		go func() {
			s.app.QueueUpdateDraw(func() {
				s.router.HideOverlay()
				cur := s.router.ActivePage()
				if cur != nil {
					s.app.SetFocus(cur.FocusTarget())
				}
			})
		}()
	}
	return true
}

// ─── Run 入口 ─────────────────────────────────────────────────────────────────

// Run 启动 TUI，srv 为 Server 接口引用。
func Run(srv server.Server) error {
	os.WriteFile("/tmp/ait-tui-start.log", []byte("TUI starting\n"), 0644)
	app := tview.NewApplication()
	pages := tview.NewPages()
	router := NewPageRouter(pages)

	// 1. 创建 MainPage（初始空壳，注入 srv 供操作按键使用）
	mp := NewMainPage(app, srv, version)

	// 2. 创建 Simulator（channel 管道）
	tasksCh, runsCh, reqsCh := NewSimulator()
	// v2: 用 srv.SubscribeRunEvents(runID) 订阅活跃 run 的事件流

	// 2.5 注入 runRequests + totalReqs（静态数据，不通过 channel）
	mp.SetRunRequests(generateRunRequests())
	mp.SetTotalReqs(10000)
	// 2.6 直接注入首屏初始数据，避免依赖异步 channel
	mp.PopulateData(generateTaskOverviews(), generateRunSummaries(), generateRequestMetrics(50), generateRunRequests(), 10000)

	// 3. 首次数据注入（Simulator 构造时已推入初始数据到 channel）
	state := &tuiState{
		app:         app,
		router:      router,
		srv:         srv,
		mainPage:    mp,
		warningPage: NewWarningPage(version),
		stopCh:      make(chan struct{}),
	}

	// 4. 启动更新循环（先启动，确保能消费初始数据）
	state.startUpdateLoop(tasksCh, runsCh, reqsCh)

	// 5. 加载主页面并设置根视图
	pages.AddPage(mp.Name(), mp.Primitive(), true, true)
	router.stack = append(router.stack, mp)
	app.SetRoot(mp.Primitive(), true)
	app.SetFocus(mp.FocusTarget())

	// 6. 注册按键 + 尺寸检测
	state.setupKeybindings()

	// 7. 运行
	os.WriteFile("/tmp/ait-tui-run.log", []byte("TUI running\n"), 0644)
	if err := app.Run(); err != nil {
		state.stop()
		_ = srv.Shutdown(5 * time.Second)
		return fmt.Errorf("tui: %w", err)
	}
	state.stop()
	_ = srv.Shutdown(5 * time.Second)
	return nil
}
