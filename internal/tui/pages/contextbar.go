package pages

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ContextBarItem 是 Context Bar 中的一个可操作项。
type ContextBarItem struct {
	Key  string // 如 "Enter"、"r"、"↑↓"
	Desc string // 操作描述
}

// RenderContextBar 渲染 Context Bar。
// 若 items 为空则返回空字符串（不占空间）。
func RenderContextBar(st Styles, width int, items []ContextBarItem) string {
	if len(items) == 0 {
		return ""
	}
	var parts []string
	for _, item := range items {
		parts = append(parts, "["+item.Key+"] "+item.Desc)
	}
	line := "  " + strings.Join(parts, "  ")
	if lipgloss.Width(line) > width {
		line = truncate(line, width)
	}
	return st.CtxBar.Width(width).Render(line)
}

// ─── 各页面 Context Bar 内容定义 ─────────────────────────────────────────────

// CtxBar_TaskList_Normal 普通任务选中时的 Context Bar。
func CtxBar_TaskList_Normal() []ContextBarItem {
	return []ContextBarItem{
		{Key: "Enter", Desc: "查看详情"},
		{Key: "r", Desc: "运行"},
		{Key: "e", Desc: "编辑"},
		{Key: "d", Desc: "删除"},
		{Key: "y", Desc: "复制"},
	}
}

// CtxBar_TaskList_Running 运行中任务选中时的 Context Bar。
func CtxBar_TaskList_Running() []ContextBarItem {
	return []ContextBarItem{
		{Key: "Enter", Desc: "进入仪表盘"},
		{Key: "s", Desc: "停止"},
		{Key: "y", Desc: "复制"},
	}
}

// CtxBar_TaskDetail_NoHistory 任务详情页，无运行记录时。
func CtxBar_TaskDetail_NoHistory() []ContextBarItem {
	return []ContextBarItem{
		{Key: "Enter/r", Desc: "运行"},
		{Key: "e", Desc: "编辑"},
		{Key: "y", Desc: "复制"},
		{Key: "d", Desc: "删除"},
	}
}

// CtxBar_TaskDetail_HasHistory 任务详情页，有运行记录时。
func CtxBar_TaskDetail_HasHistory() []ContextBarItem {
	return []ContextBarItem{
		{Key: "r", Desc: "生成报告"},
		{Key: "c", Desc: "复制摘要"},
		{Key: "Enter/r", Desc: "再次运行"},
		{Key: "e", Desc: "编辑"},
	}
}

// CtxBar_Dashboard_NoSel 标准仪表盘，无选中请求时。
func CtxBar_Dashboard_NoSel() []ContextBarItem {
	return []ContextBarItem{
		{Key: "s", Desc: "停止"},
		{Key: "b", Desc: "后台运行"},
		{Key: "r", Desc: "提前报告"},
	}
}

// CtxBar_Dashboard_Sel 标准仪表盘，已选中请求时。
func CtxBar_Dashboard_Sel() []ContextBarItem {
	return []ContextBarItem{
		{Key: "Enter", Desc: "查看请求详情"},
		{Key: "↑↓", Desc: "选择请求"},
		{Key: "s", Desc: "停止"},
	}
}

// CtxBar_TurboDash_NoSel Turbo 仪表盘，无选中级别时。
func CtxBar_TurboDash_NoSel() []ContextBarItem {
	return []ContextBarItem{
		{Key: "s", Desc: "停止"},
		{Key: "b", Desc: "后台运行"},
		{Key: "m", Desc: "标记极限"},
	}
}

// CtxBar_TurboDash_Sel Turbo 仪表盘，已选中已完成级别时。
func CtxBar_TurboDash_Sel() []ContextBarItem {
	return []ContextBarItem{
		{Key: "Enter", Desc: "查看该级别请求列表"},
		{Key: "↑↓", Desc: "选择"},
		{Key: "s", Desc: "停止"},
	}
}

// CtxBar_ReqDetail 请求详情页。
func CtxBar_ReqDetail() []ContextBarItem {
	return []ContextBarItem{
		{Key: "b/Esc", Desc: "返回仪表盘"},
		{Key: "←→", Desc: "上/下一条请求"},
	}
}
