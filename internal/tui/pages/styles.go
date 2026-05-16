package pages

import "github.com/charmbracelet/lipgloss"

// Color palette
const (
	colorHeaderBg  = lipgloss.Color("57")  // electric indigo — header background
	colorFooterBg  = lipgloss.Color("235") // near-black footer background
	colorCtxBarBg  = lipgloss.Color("237") // slightly lighter than footer — context bar
	colorPink      = lipgloss.Color("205") // vivid hot pink/magenta — brand primary
	colorCyan      = lipgloss.Color("86")  // bright aquamarine — table headers
	colorPurple    = lipgloss.Color("99")  // medium violet
	colorPurpleDim = lipgloss.Color("60")  // slate purple — selected row bg
	colorGreen     = lipgloss.Color("78")  // vivid spring green — ok/success
	colorRed       = lipgloss.Color("204") // vivid rose-red — error/fail
	colorYellow    = lipgloss.Color("221") // warm yellow — metric values
	colorTeal      = lipgloss.Color("111") // periwinkle-teal — labels
	colorWhite     = lipgloss.Color("255") // bright white
	colorMuted     = lipgloss.Color("245") // muted gray
	colorGold      = lipgloss.Color("214") // amber
	colorHeaderFg  = lipgloss.Color("212") // light pink — header right text
	colorDivider   = lipgloss.Color("238") // dim border gray
)

// Styles 汇聚所有 TUI 样式，由 NewStyles() 初始化。
type Styles struct {
	AppBorder   lipgloss.Style
	Panel       lipgloss.Style
	Header      lipgloss.Style
	HeaderInfo  lipgloss.Style
	Footer      lipgloss.Style
	CtxBar      lipgloss.Style
	SectionHead lipgloss.Style
	TableHead   lipgloss.Style
	TableRow    lipgloss.Style
	TableRowSel lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Muted       lipgloss.Style
	Ok          lipgloss.Style
	ErrStyle    lipgloss.Style
	Key         lipgloss.Style
	MetricVal   lipgloss.Style
	Dialog      lipgloss.Style
	FieldActive lipgloss.Style
	FieldIdle   lipgloss.Style
	Cursor      lipgloss.Style
	TagTurbo    lipgloss.Style
	TagStd      lipgloss.Style
	BtnPrimary  lipgloss.Style
	Divider     lipgloss.Style
}

// NewStyles 创建并返回默认样式集合。
func NewStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Background(colorHeaderBg).
			Foreground(colorWhite).
			Bold(true),
		HeaderInfo: lipgloss.NewStyle().
			Background(colorHeaderBg).
			Foreground(colorHeaderFg),
		Footer: lipgloss.NewStyle().
			Background(colorFooterBg).
			Foreground(colorMuted),
		CtxBar: lipgloss.NewStyle().
			Background(colorCtxBarBg).
			Foreground(colorWhite),
		SectionHead: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		TableHead: lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true),
		TableRow: lipgloss.NewStyle().
			Foreground(colorWhite),
		TableRowSel: lipgloss.NewStyle().
			Background(colorPurpleDim).
			Foreground(colorWhite).
			Bold(true),
		Label: lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true),
		Value: lipgloss.NewStyle().
			Foreground(colorWhite),
		Muted: lipgloss.NewStyle().
			Foreground(colorMuted),
		Ok: lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true),
		ErrStyle: lipgloss.NewStyle().
			Foreground(colorRed),
		Key: lipgloss.NewStyle().
			Foreground(colorGold).
			Bold(true),
		MetricVal: lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true),
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(0, 1),
		FieldActive: lipgloss.NewStyle().
			Background(lipgloss.Color("55")).
			Foreground(colorWhite).
			Padding(0, 1),
		FieldIdle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorDivider).
			Padding(0, 1),
		Cursor: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		TagTurbo: lipgloss.NewStyle().
			Background(colorGold).
			Foreground(colorDivider).
			Bold(true).
			Padding(0, 1),
		TagStd: lipgloss.NewStyle().
			Background(colorPurple).
			Foreground(colorWhite).
			Padding(0, 1),
		BtnPrimary: lipgloss.NewStyle().
			Background(colorPink).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 2),
		Divider: lipgloss.NewStyle().
			Foreground(colorDivider),
		AppBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorDivider),
	}
}
