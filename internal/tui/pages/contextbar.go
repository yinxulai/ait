package pages

// HotkeyItem 是底部 Hotkeys 区中的一个展示项。
type HotkeyItem struct {
	Key  string // 如 "Enter"、"r"、"↑↓"
	Desc string // 操作描述
	Text string // 纯文本提示
}

func HotkeyAction(key, desc string) HotkeyItem {
	return HotkeyItem{Key: key, Desc: desc}
}

func HotkeyText(text string) HotkeyItem {
	return HotkeyItem{Text: text}
}

func HotkeyTexts(texts ...string) []HotkeyItem {
	items := make([]HotkeyItem, 0, len(texts))
	for _, text := range texts {
		if text != "" {
			items = append(items, HotkeyText(text))
		}
	}
	return items
}

// ─── 各页面底部 Hotkeys 定义 ────────────────────────────────────────────────

// Hotkeys_TaskList_Normal 普通任务选中时的 Hotkeys。
func Hotkeys_TaskList_Normal() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", "查看详情"),
		HotkeyAction("r", "运行"),
		HotkeyAction("a", "新建任务"),
		HotkeyAction("e", "编辑"),
		HotkeyAction("d", "删除"),
		HotkeyAction("y", "复制"),
		HotkeyAction("p", "代理配置"),
	}
}

// Hotkeys_TaskList_Running 运行中任务选中时的 Hotkeys。
func Hotkeys_TaskList_Running() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", "查看详情"),
		HotkeyAction("s", "停止"),
		HotkeyAction("y", "复制"),
		HotkeyAction("p", "代理配置"),
	}
}

// Hotkeys_TaskDetail_NoHistory 任务详情页，无运行记录时。
func Hotkeys_TaskDetail_NoHistory() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("r", "运行"),
		HotkeyAction("e", "编辑"),
		HotkeyAction("y", "复制"),
		HotkeyAction("d", "删除"),
	}
}

// Hotkeys_TaskDetail_HasHistory 任务详情页，有运行记录且未运行时。
func Hotkeys_TaskDetail_HasHistory() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", "选择记录"),
		HotkeyAction("Enter", "查看运行详情"),
		HotkeyAction("r", "再次运行"),
		HotkeyAction("g", "导出 JSON 报告"),
		HotkeyAction("e", "编辑"),
		HotkeyAction("y", "复制任务"),
		HotkeyAction("d", "删除"),
	}
}

// Hotkeys_TaskDetail_Running 任务详情页，任务正在运行时。
func Hotkeys_TaskDetail_Running() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", "选择记录"),
		HotkeyAction("Enter", "进入运行中仪表盘"),
		HotkeyAction("g", "导出历史 JSON"),
		HotkeyAction("e", "编辑"),
		HotkeyAction("y", "复制任务"),
	}
}

// Hotkeys_Wizard_Step1 创建任务页，第 1 步。
func Hotkeys_Wizard_Step1() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Tab/↑↓", "切换字段"),
		HotkeyAction("←→", "切换协议"),
		HotkeyAction("Enter", "下一步"),
		HotkeyAction("Esc", "返回列表"),
	}
}

// Hotkeys_Wizard_Step2 创建任务页，第 2 步。
func Hotkeys_Wizard_Step2() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Tab/↑↓", "切换字段"),
		HotkeyAction("←→", "切换选项"),
		HotkeyAction("Enter", "下一步"),
		HotkeyAction("Esc", "返回上一步"),
	}
}

// Hotkeys_Wizard_Step3 创建任务页，第 3 步。
func Hotkeys_Wizard_Step3() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", "滚动"),
		HotkeyAction("PgUp/PgDn", "翻页"),
		HotkeyAction("Enter", "保存"),
		HotkeyAction("r", "保存并运行"),
		HotkeyAction("Esc", "返回修改"),
	}
}

// Hotkeys_Dashboard_Running_NoSel 标准仪表盘运行中，无选中请求时。
func Hotkeys_Dashboard_Running_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("s", "停止"),
		HotkeyAction("b/Esc", "返回列表"),
	}
}

// Hotkeys_Dashboard_Done_NoSel 标准仪表盘完成后，无选中请求时。
func Hotkeys_Dashboard_Done_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("r", "生成报告"),
		HotkeyAction("b/Esc", "返回列表"),
	}
}

// Hotkeys_Dashboard_Running_Sel 标准仪表盘运行中，已选中请求时。
func Hotkeys_Dashboard_Running_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", "查看请求详情"),
		HotkeyAction("↑↓", "选择请求"),
		HotkeyAction("s", "停止"),
	}
}

// Hotkeys_Dashboard_Done_Sel 标准仪表盘完成后，已选中请求时。
func Hotkeys_Dashboard_Done_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", "查看请求详情"),
		HotkeyAction("↑↓", "选择请求"),
	}
}

// Hotkeys_TurboDash_Running_NoSel Turbo 仪表盘运行中，无选中级别时。
func Hotkeys_TurboDash_Running_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("s", "停止"),
		HotkeyAction("b/Esc", "返回列表"),
	}
}

// Hotkeys_TurboDash_Done_NoSel Turbo 仪表盘完成后，无选中级别时。
func Hotkeys_TurboDash_Done_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("r", "生成报告"),
		HotkeyAction("b/Esc", "返回列表"),
	}
}

// Hotkeys_TurboDash_Running_Sel Turbo 仪表盘运行中，已选中级别时。
func Hotkeys_TurboDash_Running_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", "查看该级别请求"),
		HotkeyAction("↑↓", "选择"),
		HotkeyAction("s", "停止"),
	}
}

// Hotkeys_TurboDash_Done_Sel Turbo 仪表盘完成后，已选中级别时。
func Hotkeys_TurboDash_Done_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", "查看该级别请求"),
		HotkeyAction("↑↓", "选择"),
	}
}

// Hotkeys_ReqDetail 请求详情页。
func Hotkeys_ReqDetail() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("b/Esc", "返回仪表盘"),
		HotkeyAction("↑↓", "滚动"),
		HotkeyAction("←→", "上/下一条请求"),
	}
}

// Hotkeys_ProxyConfig 代理配置页。
func Hotkeys_ProxyConfig() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Tab/↑↓", "切换字段"),
		HotkeyAction("←→/Space", "切换类型"),
		HotkeyAction("Enter", "保存"),
		HotkeyAction("Ctrl+U", "清空"),
	}
}
