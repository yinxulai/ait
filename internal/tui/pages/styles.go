package pages

import "charm.land/lipgloss/v2"

// Color palette
var (
	colorHeaderBg           = lipgloss.Color("17")  // dark navy — refined header background
	colorHotkeysSecondaryBg = lipgloss.Color("235") // near-black secondary hotkeys background
	colorHotkeysPrimaryBg   = lipgloss.Color("237") // slightly lighter primary hotkeys background
	colorPink               = lipgloss.Color("205") // vivid hot pink/magenta — brand primary
	colorCyan               = lipgloss.Color("86")  // bright aquamarine — table headers
	colorPurple             = lipgloss.Color("99")  // medium violet
	colorPurpleDim          = lipgloss.Color("60")  // slate purple — selected row bg
	colorGreen              = lipgloss.Color("78")  // vivid spring green — ok/success
	colorRed                = lipgloss.Color("204") // vivid rose-red — error/fail
	colorYellow             = lipgloss.Color("221") // warm yellow — metric values
	colorTeal               = lipgloss.Color("111") // periwinkle-teal — labels
	colorWhite              = lipgloss.Color("255") // bright white
	colorMuted              = lipgloss.Color("245") // muted gray
	colorGold               = lipgloss.Color("214") // amber
	colorHeaderFg           = lipgloss.Color("248") // light gray — header info text
	colorDivider            = lipgloss.Color("241") // dim border gray — slightly more visible
)

// Styles 汇聚所有 TUI 样式，由 NewStyles() 初始化。
type Styles struct {
	Panel       lipgloss.Style
	Header      lipgloss.Style
	HeaderInfo  lipgloss.Style
	HotkeysSecondary lipgloss.Style
	HotkeysPrimary   lipgloss.Style
	PanelHead        lipgloss.Style
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
			Background(colorHotkeysPrimaryBg).
			Foreground(colorHeaderFg),
		HotkeysSecondary: lipgloss.NewStyle().
			Background(colorHotkeysSecondaryBg).
			Foreground(colorMuted),
		HotkeysPrimary: lipgloss.NewStyle().
			Background(colorHotkeysPrimaryBg).
			Foreground(colorWhite),
		PanelHead: lipgloss.NewStyle().
			Background(lipgloss.Color("234")).
			Foreground(colorPink).
			Bold(true),
		SectionHead: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		TableHead: lipgloss.NewStyle().
			Background(lipgloss.Color("234")).
			Foreground(colorCyan).
			Bold(true).
			Padding(0, 0),
		TableRow: lipgloss.NewStyle().
			Foreground(colorWhite).
			Padding(0, 0),
		TableRowSel: lipgloss.NewStyle().
			Background(colorPurpleDim).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 0),
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
		FieldActive: lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(colorWhite).
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorPink).
			Bold(true).
			Padding(0, 1),
		FieldIdle: lipgloss.NewStyle().
			Background(lipgloss.Color("234")).
			Foreground(colorWhite).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1),
		Cursor: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		TagTurbo: lipgloss.NewStyle().
			Foreground(colorGold).
			Bold(true).
			Padding(0, 1),
		TagStd: lipgloss.NewStyle().
			Foreground(colorPurple).
			Padding(0, 1),
		BtnPrimary: lipgloss.NewStyle().
			Background(colorPink).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 2),
		Divider: lipgloss.NewStyle().
			Foreground(colorDivider),
		Panel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorDivider),
	}
}
