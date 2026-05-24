package pages

import "github.com/yinxulai/ait/internal/i18n"

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
		HotkeyAction("Enter", i18n.T(i18n.KViewDetails)),
		HotkeyAction("r", i18n.T(i18n.KRun)),
		HotkeyAction("a", i18n.T(i18n.KNewTask)),
		HotkeyAction("e", i18n.T(i18n.KEdit)),
		HotkeyAction("d", i18n.T(i18n.KDelete)),
		HotkeyAction("y", i18n.T(i18n.KCopy)),
		HotkeyAction("p", i18n.T(i18n.KProxyConfig)),
	}
}

// Hotkeys_TaskList_Running 运行中任务选中时的 Hotkeys。
func Hotkeys_TaskList_Running() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", i18n.T(i18n.KViewDetails)),
		HotkeyAction("s", i18n.T(i18n.KStop)),
		HotkeyAction("y", i18n.T(i18n.KCopy)),
		HotkeyAction("p", i18n.T(i18n.KProxyConfig)),
	}
}

// Hotkeys_TaskDetail_NoHistory 任务详情页，无运行记录时。
func Hotkeys_TaskDetail_NoHistory() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("r", i18n.T(i18n.KRun)),
		HotkeyAction("e", i18n.T(i18n.KEdit)),
		HotkeyAction("y", i18n.T(i18n.KCopyTask)),
		HotkeyAction("d", i18n.T(i18n.KDelete)),
	}
}

// Hotkeys_TaskDetail_HasHistory 任务详情页，有运行记录且未运行时。
func Hotkeys_TaskDetail_HasHistory() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", i18n.T(i18n.KSelectRecord)),
		HotkeyAction("Enter", i18n.T(i18n.KViewRunDetails)),
		HotkeyAction("r", i18n.T(i18n.KRunAgain)),
		HotkeyAction("g", i18n.T(i18n.KExportJSONReport)),
		HotkeyAction("e", i18n.T(i18n.KEdit)),
		HotkeyAction("y", i18n.T(i18n.KCopyTask)),
		HotkeyAction("d", i18n.T(i18n.KDelete)),
	}
}

// Hotkeys_TaskDetail_Running 任务详情页，任务正在运行时。
func Hotkeys_TaskDetail_Running() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", i18n.T(i18n.KSelectRecord)),
		HotkeyAction("Enter", i18n.T(i18n.KGoToLiveDash)),
		HotkeyAction("g", i18n.T(i18n.KExportHistoryJSON)),
		HotkeyAction("e", i18n.T(i18n.KEdit)),
		HotkeyAction("y", i18n.T(i18n.KCopyTask)),
	}
}

// Hotkeys_Wizard_Step1 创建任务页，第 1 步。
func Hotkeys_Wizard_Step1() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Tab/↑↓", i18n.T(i18n.KSwitchField)),
		HotkeyAction("←→", i18n.T(i18n.KSwitchProtocol)),
		HotkeyAction("Enter", i18n.T(i18n.KNextStep)),
		HotkeyAction("Esc", i18n.T(i18n.KBackToList)),
	}
}

// Hotkeys_Wizard_Step2 创建任务页，第 2 步。
func Hotkeys_Wizard_Step2() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Tab/↑↓", i18n.T(i18n.KSwitchField)),
		HotkeyAction("←→", i18n.T(i18n.KSwitchOption)),
		HotkeyAction("Enter", i18n.T(i18n.KNextStep)),
		HotkeyAction("Esc", i18n.T(i18n.KGoBack)),
	}
}

// Hotkeys_Wizard_Step3 创建任务页，第 3 步。
func Hotkeys_Wizard_Step3() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", i18n.T(i18n.KScroll)),
		HotkeyAction("PgUp/PgDn", i18n.T(i18n.KPageTurn)),
		HotkeyAction("Enter", i18n.T(i18n.KSave)),
		HotkeyAction("r", i18n.T(i18n.KSaveAndRun)),
		HotkeyAction("Esc", i18n.T(i18n.KBackToEdit)),
	}
}

// Hotkeys_Dashboard_Running_NoSel 标准仪表盘运行中，无选中请求时。
func Hotkeys_Dashboard_Running_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("s", i18n.T(i18n.KStop)),
		HotkeyAction("b/Esc", i18n.T(i18n.KBackToList)),
	}
}

// Hotkeys_Dashboard_Done_NoSel 标准仪表盘完成后，无选中请求时。
func Hotkeys_Dashboard_Done_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("r", i18n.T(i18n.KGenerateReport)),
		HotkeyAction("b/Esc", i18n.T(i18n.KBackToList)),
	}
}

// Hotkeys_Dashboard_Running_Sel 标准仪表盘运行中，已选中请求时。
func Hotkeys_Dashboard_Running_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", i18n.T(i18n.KViewRequest)),
		HotkeyAction("↑↓", i18n.T(i18n.KSelectRequest)),
		HotkeyAction("s", i18n.T(i18n.KStop)),
	}
}

// Hotkeys_Dashboard_Done_Sel 标准仪表盘完成后，已选中请求时。
func Hotkeys_Dashboard_Done_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", i18n.T(i18n.KViewRequest)),
		HotkeyAction("↑↓", i18n.T(i18n.KSelectRequest)),
	}
}

// Hotkeys_TurboDash_Running_NoSel Turbo 仪表盘运行中，无选中级别时。
func Hotkeys_TurboDash_Running_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("s", i18n.T(i18n.KStop)),
		HotkeyAction("b/Esc", i18n.T(i18n.KBackToList)),
	}
}

// Hotkeys_TurboDash_Done_NoSel Turbo 仪表盘完成后，无选中级别时。
func Hotkeys_TurboDash_Done_NoSel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("r", i18n.T(i18n.KGenerateReport)),
		HotkeyAction("b/Esc", i18n.T(i18n.KBackToList)),
	}
}

// Hotkeys_TurboDash_Running_Sel Turbo 仪表盘运行中，已选中级别时。
func Hotkeys_TurboDash_Running_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", i18n.T(i18n.KViewLevelReqs)),
		HotkeyAction("↑↓", i18n.T(i18n.KSelectItem)),
		HotkeyAction("s", i18n.T(i18n.KStop)),
	}
}

// Hotkeys_TurboDash_Done_Sel Turbo 仪表盘完成后，已选中级别时。
func Hotkeys_TurboDash_Done_Sel() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Enter", i18n.T(i18n.KViewLevelReqs)),
		HotkeyAction("↑↓", i18n.T(i18n.KSelectItem)),
	}
}

// Hotkeys_ReqDetail 请求详情页。
func Hotkeys_ReqDetail() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("b/Esc", i18n.T(i18n.KBackToDash)),
		HotkeyAction("↑↓", i18n.T(i18n.KScroll)),
		HotkeyAction("←→", i18n.T(i18n.KPrevNextReq)),
	}
}

// Hotkeys_ProxyConfig 代理配置页。
func Hotkeys_ProxyConfig() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("Tab/↑↓", i18n.T(i18n.KSwitchField)),
		HotkeyAction("←→/Space", i18n.T(i18n.KSwitchType)),
		HotkeyAction("Enter", i18n.T(i18n.KSave)),
		HotkeyAction("Ctrl+U", i18n.T(i18n.KClear)),
	}
}

// Hotkeys_Help 帮助页。
func Hotkeys_Help() []HotkeyItem {
	return []HotkeyItem{
		HotkeyAction("↑↓", i18n.T(i18n.KScroll)),
		HotkeyAction("PgUp/PgDn", i18n.T(i18n.KPageTurn)),
		HotkeyAction("g/G", i18n.T(i18n.KTopBottom)),
	}
}
