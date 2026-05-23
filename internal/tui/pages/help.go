package pages

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"charm.land/lipgloss/v2"
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
		HeaderTitle:    "帮助",
		HeaderSubtitle: "AIT — AI 接口压测工具概念说明与操作指南",
		HeaderMeta:     "帮助",
		Hotkeys:        NewPageHotkeys(Hotkeys_Help(), "[b/Esc] 返回", "[q] 退出"),
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
			title: "核心概念",
			items: []helpItem{
				{"任务 (Task)", "一组压测配置的集合，包含目标接口、模型、并发数、请求数等参数。任务可多次运行，每次运行独立记录结果。"},
				{"运行 (Run)", "任务的一次具体执行。每次运行产生独立的指标数据和请求记录，可导出为 JSON/CSV 报告。"},
				{"标准模式", "以固定并发数执行所有请求，适合衡量稳定负载下的接口性能。"},
				{"Turbo 模式", "自动从低并发逐步爬坡，找出接口在保持成功率要求下能承受的最大稳定并发数。"},
			},
		},
		{
			title: "性能指标",
			items: []helpItem{
				{"TPS", "Tokens Per Second，每秒输出 Token 数，衡量模型的文本生成速率。"},
				{"均值TPS", "本次运行中所有请求的 TPS 均值，反映整体吞吐水平。"},
				{"TTFT", "Time To First Token，从发送请求到收到第一个 Token 的耗时，衡量模型响应延迟。"},
				{"均值TTFT", "本次运行中所有请求的 TTFT 均值。"},
				{"成功率", "成功完成的请求数占总请求数的百分比。失败包括超时、HTTP 错误、模型返回错误等。"},
				{"缓存命中", "请求中使用了 KV 缓存（Prompt Cache）的比例。命中缓存可显著降低 TTFT 和推理成本。该指标为二值统计：单次请求若有任何 Token 命中缓存则计为命中。"},
				{"并发（Turbo）", "Turbo 模式下找到的最大稳定并发数，即在满足最低成功率要求的前提下能同时维持的请求数。"},
			},
		},
		{
			title: "协议支持",
			items: []helpItem{
				{"OpenAI", "兼容 OpenAI Chat Completions API（/v1/chat/completions），支持流式和非流式响应。"},
				{"Anthropic", "兼容 Anthropic Messages API（/v1/messages），支持流式和非流式响应。"},
			},
		},
		{
			title: "快捷键 — 全局",
			items: []helpItem{
				{"q / Ctrl+C", "退出程序。"},
				{"?", "打开此帮助页。"},
				{"b / Esc", "返回上一页。"},
			},
		},
		{
			title: "快捷键 — 任务列表",
			items: []helpItem{
				{"↑↓ / j k", "选择任务。"},
				{"Enter", "进入任务详情页。"},
				{"r", "立即运行选中任务。"},
				{"s", "停止正在运行的任务（仅任务运行中可用）。"},
				{"a", "新建任务（打开向导）。"},
				{"e", "编辑选中任务配置。"},
				{"d", "删除选中任务（需确认）。"},
				{"y", "复制选中任务（生成副本）。"},
				{"p", "打开代理配置页。"},
			},
		},
		{
			title: "快捷键 — 任务详情",
			items: []helpItem{
				{"↑↓ / j k", "在历史运行记录中选择条目。"},
				{"Enter", "查看选中运行的仪表盘；若任务正在运行，进入实时仪表盘。"},
				{"r", "再次运行该任务（无正在运行的实例时可用）。"},
				{"g", "将选中的历史运行导出为 JSON 报告。"},
				{"e", "编辑任务配置。"},
				{"y", "复制任务。"},
				{"d", "删除任务。"},
			},
		},
		{
			title: "快捷键 — 运行仪表盘",
			items: []helpItem{
				{"↑↓ / j k", "选择请求条目。"},
				{"Enter", "查看选中请求的详情（耗时、Token、响应体等）。"},
				{"s", "停止正在运行的任务。"},
				{"r", "生成 JSON 报告（运行结束后可用）。"},
				{"b / Esc", "返回任务详情页。"},
			},
		},
		{
			title: "报告导出",
			items: []helpItem{
				{"JSON 报告", "完整记录每次请求的所有指标、请求/响应体，适合程序化分析。"},
				{"CSV 报告", "表格形式的汇总数据，可直接在电子表格中打开。报告默认保存在当前工作目录。"},
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
