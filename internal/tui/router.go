package tui

import "github.com/rivo/tview"

// PageRouter 管理页面栈和 overlay。
type PageRouter struct {
	pages   *tview.Pages
	stack   []Page
	overlay Page
}

// NewPageRouter 创建一个新的 PageRouter，绑定到给定的 tview.Pages。
func NewPageRouter(pages *tview.Pages) *PageRouter {
	return &PageRouter{pages: pages}
}

// NavigateTo 将 page 压入栈并切换到该页面。
func (r *PageRouter) NavigateTo(page Page) {
	if len(r.stack) > 0 {
		r.pages.HidePage(r.stack[len(r.stack)-1].Name())
	}
	r.stack = append(r.stack, page)
	r.pages.AddPage(page.Name(), page.Primitive(), true, true)
	r.pages.ShowPage(page.Name())
}

// GoBack 弹出栈顶页面，返回上一页。若栈中仅剩一页则不操作。
func (r *PageRouter) GoBack() {
	if len(r.stack) <= 1 {
		return
	}
	top := r.stack[len(r.stack)-1]
	r.pages.RemovePage(top.Name())
	r.stack = r.stack[:len(r.stack)-1]
	prev := r.stack[len(r.stack)-1]
	r.pages.ShowPage(prev.Name())
}

// ShowOverlay 在不影响栈的情况下显示 overlay 页面。
func (r *PageRouter) ShowOverlay(page Page) {
	if r.overlay != nil {
		r.pages.RemovePage(r.overlay.Name())
	}
	r.overlay = page
	r.pages.AddPage(page.Name(), page.Primitive(), true, true)
}

// HideOverlay 移除 overlay，恢复栈顶页面。
func (r *PageRouter) HideOverlay() {
	if r.overlay == nil {
		return
	}
	r.pages.RemovePage(r.overlay.Name())
	r.overlay = nil
	r.pages.ShowPage(r.stack[len(r.stack)-1].Name())
}

// ActivePage 返回当前活跃页面：overlay 优先，其次栈顶。
func (r *PageRouter) ActivePage() Page {
	if r.overlay != nil {
		return r.overlay
	}
	if len(r.stack) == 0 {
		return nil
	}
	return r.stack[len(r.stack)-1]
}

// StackDepth 返回当前栈深度。
func (r *PageRouter) StackDepth() int {
	return len(r.stack)
}
