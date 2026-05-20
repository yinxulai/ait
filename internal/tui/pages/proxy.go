package pages

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProxyConfigState 代理配置页面状态。
type ProxyConfigState struct {
	URL string // 当前编辑中的代理 URL
}

// NewProxyConfigState 创建代理配置页面状态，传入当前已保存的代理 URL。
func NewProxyConfigState(currentURL string) *ProxyConfigState {
	return &ProxyConfigState{URL: currentURL}
}

// HandleProxyConfigKey 处理代理配置页面的按键。
func HandleProxyConfigKey(s *ProxyConfigState, msg tea.KeyMsg, client Client) (*ProxyConfigState, tea.Cmd, NavAction) {
	nav := NavAction{}
	if s == nil {
		return s, nil, NavAction{To: NavTaskList}
	}

	switch msg.String() {
	case "esc":
		nav = NavAction{To: NavTaskList}

	case "enter":
		cmd := client.SaveProxyConfigCmd(s.URL)
		nav = NavAction{To: NavTaskList}
		return s, cmd, nav

	case "backspace":
		r := []rune(s.URL)
		if len(r) > 0 {
			s.URL = string(r[:len(r)-1])
		}

	case "ctrl+u":
		s.URL = ""

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}

	default:
		if len(msg.Runes) > 0 {
			s.URL += string(msg.Runes)
		}
	}

	return s, nil, nav
}

// RenderProxyConfig 渲染代理配置页面。
func RenderProxyConfig(s *ProxyConfigState, st Styles, width, height int) string {
	if TooSmall(width, height) {
		return renderTooSmall(st, width, height)
	}
	if s == nil {
		return renderTooSmall(st, width, height)
	}

	l := PageLayout{
		HeaderTitle:    "代理配置",
		HeaderSubtitle: "设置全局 HTTP 代理，适用于所有任务的请求。留空则使用系统环境变量或直连。",
		HeaderMeta:     "全局配置",
		Hotkeys:        NewPageHotkeys(Hotkeys_ProxyConfig(), "[Esc] 返回", "[q] 退出"),
	}
	frame := l.Frame(width, height)
	panel := NewPanelFrame(frame.OuterWidth)

	content := buildProxyConfigContent(s, st, panel.InnerWidth, PanelContentHeight(frame.InnerHeight))
	return l.Assemble(panel.Wrap(st, content), st, width)
}

func buildProxyConfigContent(s *ProxyConfigState, st Styles, contentW, maxH int) string {
	var lines []string

	lines = append(lines, st.SectionHead.Render("代理地址"))
	lines = append(lines, "")

	// 字段宽度（与 wizard renderWizardField 保持一致）
	fieldW := maxInt(10, contentW-19)
	displayURL := fitTail(s.URL, maxInt(1, fieldW-1)) + "█"
	renderedField := st.FieldActive.Width(fieldW).Render(st.Value.Render(displayURL))

	labelBlock := strings.Join([]string{
		strings.Repeat(" ", 15),
		lipgloss.NewStyle().Width(15).Render(st.Label.Render("代理地址")),
		strings.Repeat(" ", 15),
	}, "\n")
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, labelBlock, renderedField))
	lines = append(lines, "")

	hint := "示例: http://127.0.0.1:7890  或留空以直连"
	lines = append(lines, st.Muted.Render(truncate(hint, contentW)))
	lines = append(lines, "")
	lines = append(lines, st.Muted.Render(truncate("配置保存至 ~/.ait/config.json，重启无需重新输入。", contentW)))

	// 填充至 maxH
	for len(lines) < maxH {
		lines = append(lines, "")
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	return strings.Join(lines, "\n")
}
