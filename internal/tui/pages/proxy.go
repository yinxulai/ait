package pages

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/i18n"
)

// 代理类型常量
const (
	ProxyTypeHTTP   = "http"
	ProxyTypeSOCKS5 = "socks5"
	ProxyTypeSSH    = "ssh"
)

var proxyTypes = []string{ProxyTypeHTTP, ProxyTypeSOCKS5, ProxyTypeSSH}

// ProxyConfigState 代理配置页面状态。
type ProxyConfigState struct {
	ProxyType  string // "http" | "socks5" | "ssh"
	FieldIndex int    // 0=代理类型, 1=代理地址
	input      textinput.Model
}

// NewProxyConfigState 创建代理配置页面状态，传入当前已保存的代理 URL。
func NewProxyConfigState(currentURL string) *ProxyConfigState {
	proxyType := ProxyTypeHTTP
	switch {
	case strings.HasPrefix(currentURL, "socks5://"):
		proxyType = ProxyTypeSOCKS5
	case strings.HasPrefix(currentURL, "ssh://"):
		proxyType = ProxyTypeSSH
	}

	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.SetValue(currentURL)
	ti.CursorEnd()
	ti.Focus()
	return &ProxyConfigState{ProxyType: proxyType, FieldIndex: 1, input: ti}
}

// proxyTypeLabel 返回代理类型的显示名。
func proxyTypeLabel(t string) string {
	switch t {
	case ProxyTypeSOCKS5:
		return "SOCKS5"
	case ProxyTypeSSH:
		return "SSH"
	default:
		return "HTTP"
	}
}

// proxyTypeHint 返回类型对应的示例 URL 提示。
func proxyTypeHint(t string) string {
	switch t {
	case ProxyTypeSOCKS5:
		return i18n.T(i18n.KExSOCKS5)
	case ProxyTypeSSH:
		return i18n.T(i18n.KExSSH)
	default:
		return i18n.T(i18n.KExHTTP)
	}
}

// cycleProxyType 循环切换代理类型，同时更新 URL 的 scheme 前缀。
func cycleProxyType(s *ProxyConfigState, forward bool) {
	idx := 0
	for i, t := range proxyTypes {
		if t == s.ProxyType {
			idx = i
			break
		}
	}
	if forward {
		idx = (idx + 1) % len(proxyTypes)
	} else {
		idx = (idx - 1 + len(proxyTypes)) % len(proxyTypes)
	}
	newType := proxyTypes[idx]

	// 更新 URL scheme 前缀
	url := s.input.Value()
	for _, t := range proxyTypes {
		scheme := t + "://"
		if strings.HasPrefix(url, scheme) {
			url = strings.TrimPrefix(url, scheme)
			break
		}
	}
	if url != "" {
		url = newType + "://" + url
	}
	s.input.SetValue(url)
	s.input.CursorEnd()

	s.ProxyType = newType
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
		if s.FieldIndex == 0 {
			// 在类型字段上按 Enter，切换到 URL 字段
			s.FieldIndex = 1
			return s, nil, nav
		}
		cmd := client.SaveProxyConfigCmd(s.input.Value())
		nav = NavAction{To: NavTaskList}
		return s, cmd, nav

	case "tab", "down", "j":
		if s.FieldIndex < 1 {
			s.FieldIndex++
		}

	case "shift+tab", "up", "k":
		if s.FieldIndex > 0 {
			s.FieldIndex--
		}

	case "left":
		if s.FieldIndex == 0 {
			cycleProxyType(s, false)
		} else {
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd, nav
		}

	case "right":
		if s.FieldIndex == 0 {
			cycleProxyType(s, true)
		} else {
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd, nav
		}

	case " ":
		if s.FieldIndex == 0 {
			cycleProxyType(s, true)
		} else {
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd, nav
		}

	case "?":
		nav = NavAction{To: NavHelp}

	case "q", "ctrl+c":
		nav = NavAction{To: NavQuit}

	default:
		if s.FieldIndex == 1 {
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			return s, cmd, nav
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
		HeaderTitle:    i18n.T(i18n.KProxyConfig),
		HeaderSubtitle: i18n.T(i18n.KProxySubtitle),
		HeaderMeta:     i18n.T(i18n.KGlobalConfig),
		Hotkeys:        NewPageHotkeysWithHelp(Hotkeys_ProxyConfig(), i18n.T(i18n.KHintGoBack), i18n.T(i18n.KHintQuit)),
	}
	frame := l.Frame(width, height)
	panel := NewPanelFrame(frame.OuterWidth)

	content := buildProxyConfigContent(s, st, panel.InnerWidth, PanelContentHeight(frame.InnerHeight))
	return l.Assemble(panel.Wrap(st, content), st, width)
}

func buildProxyConfigContent(s *ProxyConfigState, st Styles, contentW, maxH int) string {
	var lines []string

	appendBlock := func(block string) {
		for _, l := range strings.Split(block, "\n") {
			lines = append(lines, l)
		}
	}

	fieldW := maxInt(10, contentW-19)

	// 代理类型字段
	typeLabel := proxyTypeLabel(s.ProxyType)
	if s.FieldIndex == 0 {
		typeLabel = "‹ " + typeLabel + " ›"
	}
	typeLabel = truncate(typeLabel, maxInt(4, fieldW))
	typeFieldStyle := st.FieldIdle
	if s.FieldIndex == 0 {
		typeFieldStyle = st.FieldActive
	}
	typeLabelBlock := lipgloss.NewStyle().Width(15).Height(3).
		AlignVertical(lipgloss.Center).
		Render(st.Label.Render(i18n.T(i18n.KProxyType)))
	typeRendered := typeFieldStyle.Width(fieldW + 4).Render(st.Value.Render(typeLabel))
	appendBlock(lipgloss.JoinHorizontal(lipgloss.Top, typeLabelBlock, typeRendered))

	// 代理地址字段
	s.input.Width = fieldW
	urlFieldStyle := st.FieldIdle
	if s.FieldIndex == 1 {
		urlFieldStyle = st.FieldActive
	}
	var urlRendered string
	if s.FieldIndex == 1 {
		urlRendered = urlFieldStyle.Width(fieldW + 4).Render(s.input.View())
	} else {
		v := s.input.Value()
		if v == "" {
			urlRendered = urlFieldStyle.Width(fieldW + 4).Render(st.Muted.Render(i18n.T(i18n.KNotSet)))
		} else {
			urlRendered = urlFieldStyle.Width(fieldW + 4).Render(st.Value.Render(fitTail(v, fieldW)))
		}
	}
	urlLabelBlock := lipgloss.NewStyle().Width(15).Height(3).
		AlignVertical(lipgloss.Center).
		Render(st.Label.Render(i18n.T(i18n.KProxyURL)))
	appendBlock(lipgloss.JoinHorizontal(lipgloss.Top, urlLabelBlock, urlRendered))

	lines = append(lines, "")
	lines = append(lines, st.Muted.Render(truncate(proxyTypeHint(s.ProxyType), contentW)))
	lines = append(lines, "")
	lines = append(lines, st.Muted.Render(truncate(i18n.T(i18n.KProxySaveHint), contentW)))

	// 填充至 maxH
	for len(lines) < maxH {
		lines = append(lines, "")
	}
	if len(lines) > maxH {
		lines = lines[:maxH]
	}
	return strings.Join(lines, "\n")
}
