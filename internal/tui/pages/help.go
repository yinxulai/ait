package pages

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"charm.land/lipgloss/v2"
	"github.com/yinxulai/ait/internal/i18n"
)

// HelpState 帮助页状态。
type HelpState struct {
	ScrollY  int
	BackNav  NavAction // 按 b/Esc 时的返回目标
}

// NewHelpState 创建帮助页状态。
func NewHelpState(backNav NavAction) *HelpState {
	return &HelpState{BackNav: backNav}
}

// HandleHelpKey 处理帮助页按键。
func HandleHelpKey(s *HelpState, msg tea.KeyMsg) (*HelpState, NavAction) {
	nav := NavAction{}
	if s == nil {
		return s, NavAction{To: NavTaskList}
	}

	lines := buildHelpLines(s, 9999, 9999) // 仅用于计算总行数
	totalLines := len(lines)

	switch msg.String() {
	case "b", "esc", "q", "?":
		if s.BackNav.To != NavNone {
			nav = s.BackNav
		} else {
			nav = NavAction{To: NavTaskList}
		}

	case "ctrl+c":
		nav = NavAction{To: NavQuit}

	case "up", "k":
		if s.ScrollY > 0 {
			s.ScrollY--
		}

	case "down", "j":
		if s.ScrollY < totalLines-1 {
			s.ScrollY++
		}

	case "pgup":
		s.ScrollY -= 10
		if s.ScrollY < 0 {
			s.ScrollY = 0
		}

	case "pgdown":
		s.ScrollY += 10
		if s.ScrollY >= totalLines {
			s.ScrollY = maxInt(0, totalLines-1)
		}

	case "home", "g":
		s.ScrollY = 0

	case "end", "G":
		s.ScrollY = maxInt(0, totalLines-1)
	}

	return s, nav
}

// RenderHelp 渲染帮助页面。
func RenderHelp(s *HelpState, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}
	if s == nil {
		s = &HelpState{}
	}

	l := PageLayout{
		HeaderTitle:    i18n.T(i18n.KHelpTitle),
		HeaderSubtitle: i18n.T(i18n.KHelpSubtitle),
		HeaderMeta:     i18n.T(i18n.KHelpMeta),
		Hotkeys:        NewPageHotkeys(Hotkeys_Help(), i18n.T(i18n.KHintEscBack), i18n.T(i18n.KHintQuit)),
	}
	frame := l.Frame(width, height)
	panel := NewPanelFrame(frame.OuterWidth)
	content := buildHelpContent(s, st, panel.InnerWidth, PanelContentHeight(frame.InnerHeight))
	return l.Assemble(panel.Wrap(st, content), st, width)
}

// ─── 内容构建 ─────────────────────────────────────────────────────────────────

// helpSection 表示帮助页的一个章节。
type helpSection struct {
	title string
	items []helpItem
}

type helpItem struct {
	term string // 概念名称或快捷键
	desc string // 说明
}

func helpContent() []helpSection {
	return []helpSection{
		{
			title: i18n.T(i18n.KHelpSecConcepts),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermTask), i18n.T(i18n.KHelpDescTask)},
				{i18n.T(i18n.KHelpTermRun), i18n.T(i18n.KHelpDescRun)},
				{i18n.T(i18n.KHelpTermStandard), i18n.T(i18n.KHelpDescStandard)},
				{i18n.T(i18n.KHelpTermTurboMode), i18n.T(i18n.KHelpDescTurboMode)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecMetrics),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermTPS), i18n.T(i18n.KHelpDescTPS)},
				{i18n.T(i18n.KHelpTermAvgTPS), i18n.T(i18n.KHelpDescAvgTPS)},
				{i18n.T(i18n.KHelpTermTTFT), i18n.T(i18n.KHelpDescTTFT)},
				{i18n.T(i18n.KHelpTermAvgTTFT), i18n.T(i18n.KHelpDescAvgTTFT)},
				{i18n.T(i18n.KHelpTermSuccessRate), i18n.T(i18n.KHelpDescSuccessRate)},
				{i18n.T(i18n.KHelpTermCacheHit), i18n.T(i18n.KHelpDescCacheHit)},
				{i18n.T(i18n.KHelpTermConcurrencyTurbo), i18n.T(i18n.KHelpDescConcurrencyTurbo)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecProtocols),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermOpenAI), i18n.T(i18n.KHelpDescOpenAI)},
				{i18n.T(i18n.KHelpTermAnthropic), i18n.T(i18n.KHelpDescAnthropic)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecGlobal),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermQuit), i18n.T(i18n.KHelpDescQuit)},
				{i18n.T(i18n.KHelpTermQuestionMark), i18n.T(i18n.KHelpDescQuestionMark)},
				{i18n.T(i18n.KHelpTermBack), i18n.T(i18n.KHelpDescBack)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecTaskList),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermSelectTask), i18n.T(i18n.KHelpDescSelectTask)},
				{i18n.T(i18n.KHelpTermEnterDetail), i18n.T(i18n.KHelpDescEnterDetail)},
				{i18n.T(i18n.KHelpTermRunTask), i18n.T(i18n.KHelpDescRunTask)},
				{i18n.T(i18n.KHelpTermStopTask), i18n.T(i18n.KHelpDescStopTask)},
				{i18n.T(i18n.KHelpTermNewTask), i18n.T(i18n.KHelpDescNewTask)},
				{i18n.T(i18n.KHelpTermEditTask), i18n.T(i18n.KHelpDescEditTask)},
				{i18n.T(i18n.KHelpTermDeleteTask), i18n.T(i18n.KHelpDescDeleteTask)},
				{i18n.T(i18n.KHelpTermCopyTask), i18n.T(i18n.KHelpDescCopyTask)},
				{i18n.T(i18n.KHelpTermProxy), i18n.T(i18n.KHelpDescProxy)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecTaskDetail),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermSelectHistory), i18n.T(i18n.KHelpDescSelectHistory)},
				{i18n.T(i18n.KHelpTermEnterDash), i18n.T(i18n.KHelpDescEnterDash)},
				{i18n.T(i18n.KHelpTermRunAgain), i18n.T(i18n.KHelpDescRunAgain)},
				{i18n.T(i18n.KHelpTermExport), i18n.T(i18n.KHelpDescExport)},
				{i18n.T(i18n.KHelpTermEditConfig), i18n.T(i18n.KHelpDescEditConfig)},
				{i18n.T(i18n.KHelpTermCopyTask2), i18n.T(i18n.KHelpDescCopyTask2)},
				{i18n.T(i18n.KHelpTermDeleteTask2), i18n.T(i18n.KHelpDescDeleteTask2)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecDashboard),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermSelectReq), i18n.T(i18n.KHelpDescSelectReq)},
				{i18n.T(i18n.KHelpTermViewReq), i18n.T(i18n.KHelpDescViewReq)},
				{i18n.T(i18n.KHelpTermStopDash), i18n.T(i18n.KHelpDescStopDash)},
				{i18n.T(i18n.KHelpTermGenerateReport), i18n.T(i18n.KHelpDescGenerateReport)},
				{i18n.T(i18n.KHelpTermBackDash), i18n.T(i18n.KHelpDescBackDash)},
			},
		},
		{
			title: i18n.T(i18n.KHelpSecExport),
			items: []helpItem{
				{i18n.T(i18n.KHelpTermJSONReport), i18n.T(i18n.KHelpDescJSONReport)},
				{i18n.T(i18n.KHelpTermCSVReport), i18n.T(i18n.KHelpDescCSVReport)},
			},
		},
	}
}

func buildHelpLines(s *HelpState, contentW, _ int) []string {
	sections := helpContent()
	var lines []string
	for _, sec := range sections {
		lines = append(lines, "  "+sec.title)
		lines = append(lines, "")
		for _, item := range sec.items {
			lines = append(lines, "    "+item.term)
			// 简单 wrap desc
			wrapped := wrapText(item.desc, maxInt(20, contentW-6))
			for _, l := range wrapped {
				lines = append(lines, "      "+l)
			}
			lines = append(lines, "")
		}
	}
	return lines
}

func buildHelpContent(s *HelpState, st Styles, contentW, maxH int) string {
	sections := helpContent()
	termW := 16 // 概念名/快捷键列宽

	var rawLines []string
	for _, sec := range sections {
		// 章节标题
		rawLines = append(rawLines, st.SectionHead.Render("  "+sec.title))
		rawLines = append(rawLines, "")
		for _, item := range sec.items {
			// term 列
			termStr := st.Label.Render(padRight(item.term, termW))
			// desc 第一行与 term 同行，后续行缩进
			descW := maxInt(20, contentW-termW-4)
			wrapped := wrapText(item.desc, descW)
			if len(wrapped) == 0 {
				wrapped = []string{""}
			}
			// 第一行：term + desc[0]
			firstLine := "  " + termStr + "  " + wrapped[0]
			rawLines = append(rawLines, firstLine)
			// 后续行缩进对齐
			indent := strings.Repeat(" ", 2+lipgloss.Width(termStr)+2)
			for _, seg := range wrapped[1:] {
				rawLines = append(rawLines, indent+seg)
			}
		}
		rawLines = append(rawLines, "")
	}

	// 应用滚动
	if s.ScrollY >= len(rawLines) {
		s.ScrollY = maxInt(0, len(rawLines)-1)
	}
	visible := rawLines
	if s.ScrollY > 0 {
		visible = rawLines[s.ScrollY:]
	}

	// 填充至 maxH
	if len(visible) > maxH {
		visible = visible[:maxH]
	}
	for len(visible) < maxH {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n")
}
