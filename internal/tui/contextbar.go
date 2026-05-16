package tui

import (
	"fmt"
	"strings"
)

// contextBarItem 描述 Context Bar 中的一个可用操作。
type contextBarItem struct {
	key  string
	desc string
}

// renderContextBar 渲染 Context Bar：紧贴 Footer 上方的动态操作提示行。
// 若 items 为空则返回空字符串（不占空间）。
func (m *Model) renderContextBar(items []contextBarItem) string {
	if len(items) == 0 {
		return ""
	}
	var parts []string
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s %s",
			m.styles.key.Render("["+item.key+"]"),
			m.styles.muted.Render(item.desc),
		))
	}
	bar := "  " + strings.Join(parts, "  ")
	barW := m.width
	if barW < 1 {
		barW = 80
	}
	return m.styles.footer.Width(barW).Render(bar)
}

// contextBarItems_taskList 返回任务列表页的 Context Bar 内容。
func contextBarItems_taskList(isRunning bool) []contextBarItem {
	if isRunning {
		return []contextBarItem{
			{"Enter", "进入仪表盘"},
			{"s", "停止"},
			{"y", "复制"},
		}
	}
	return []contextBarItem{
		{"Enter", "查看详情"},
		{"r", "运行"},
		{"e", "编辑"},
		{"d", "删除"},
		{"y", "复制"},
	}
}

// contextBarItems_taskDetail 返回任务详情页的 Context Bar 内容。
func contextBarItems_taskDetail(hasHistory bool) []contextBarItem {
	if hasHistory {
		return []contextBarItem{
			{"r", "生成报告"},
			{"c", "复制摘要"},
			{"Enter", "再次运行"},
			{"e", "编辑"},
		}
	}
	return []contextBarItem{
		{"Enter", "运行"},
		{"e", "编辑"},
		{"y", "复制"},
		{"d", "删除"},
	}
}

// contextBarItems_dashboard_nosel 仪表盘无选中请求。
func contextBarItems_dashboard_nosel() []contextBarItem {
	return []contextBarItem{
		{"s", "停止"},
		{"b", "后台运行"},
		{"r", "提前报告"},
	}
}

// contextBarItems_dashboard_sel 仪表盘有选中请求。
func contextBarItems_dashboard_sel() []contextBarItem {
	return []contextBarItem{
		{"Enter", "查看请求详情"},
		{"↑↓", "选择请求"},
		{"s", "停止"},
	}
}

// contextBarItems_reqdetail 请求详情页。
func contextBarItems_reqdetail() []contextBarItem {
	return []contextBarItem{
		{"b/Esc", "返回仪表盘"},
		{"←→", "上/下一条请求"},
	}
}
