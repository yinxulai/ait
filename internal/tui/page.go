package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// panelIndex 表示主布局中当前激活的 panel。
type panelIndex int

const (
	panelTasks   panelIndex = iota // 左 panel — 任务列表
	panelHistory                   // 中 panel — 运行历史
	panelStats                     // 右 panel — 统计+请求
)

// Page 接口：每个页面是一个自包含的 Page 实现者。
type Page interface {
	// Name 返回页面唯一标识名。
	Name() string

	// Primitive 返回页面的根 tview.Primitive，供 Router 挂载。
	Primitive() tview.Primitive

	// FocusTarget 返回页面激活时应聚焦的 primitive。
	FocusTarget() tview.Primitive

	// HandleKey 处理按键事件。返回 nil 表示已消费，返回 event 表示透传。
	HandleKey(event *tcell.EventKey, router *PageRouter) *tcell.EventKey

	// OnActivate 页面被激活时调用（v1 空实现，v2 预留）。
	OnActivate()

	// OnDeactivate 页面被停用时调用（v1 空实现，v2 预留）。
	OnDeactivate()
}
