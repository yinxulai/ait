package tui

import "github.com/charmbracelet/lipgloss"

// Color palette — inspired by the Lip Gloss demo aesthetic:
// electric-purple header, vivid hot-pink brand, deep-plum panels, aqua accents.
const (
	colorHeaderBg  = lipgloss.Color("57")  // electric indigo — header background
	colorFooterBg  = lipgloss.Color("235") // near-black footer background
	colorPink      = lipgloss.Color("205") // vivid hot pink/magenta — brand primary
	colorCyan      = lipgloss.Color("86")  // bright aquamarine — table headers
	colorPurple    = lipgloss.Color("99")  // medium violet — badge, border
	colorPurpleDim = lipgloss.Color("60")  // slate purple — selected row bg
	colorPanelBg   = lipgloss.Color("55")  // deep plum — content card / panel bg
	colorGold      = lipgloss.Color("214") // amber — status badge
	colorGreen     = lipgloss.Color("78")  // vivid spring green — ok/success
	colorRed       = lipgloss.Color("204") // vivid rose-red — error/fail
	colorYellow    = lipgloss.Color("221") // warm yellow — metric values
	colorTeal      = lipgloss.Color("111") // periwinkle-teal — labels
	colorWhite     = lipgloss.Color("255") // bright white
	colorMuted     = lipgloss.Color("245") // muted gray
	colorDimBorder = lipgloss.Color("238") // dim border gray
	colorFieldBg   = lipgloss.Color("55")  // deep plum — active field bg
	colorDark      = lipgloss.Color("235") // near-black text on light badge
	colorHeaderFg  = lipgloss.Color("212") // light pink — header right text
)

type styles struct {
	header      lipgloss.Style
	footer      lipgloss.Style
	sectionHead lipgloss.Style
	tableHead   lipgloss.Style
	tableRow    lipgloss.Style
	tableRowSel lipgloss.Style
	label       lipgloss.Style
	value       lipgloss.Style
	muted       lipgloss.Style
	ok          lipgloss.Style
	errStyle    lipgloss.Style
	key         lipgloss.Style
	metricVal   lipgloss.Style
	dialog      lipgloss.Style
	fieldActive lipgloss.Style
	fieldIdle   lipgloss.Style
	cursor      lipgloss.Style
	// Badge styles
	badge     lipgloss.Style // AIT brand badge (purple)
	badgeAlt  lipgloss.Style // alternate badge (gold)
	tagTurbo  lipgloss.Style // "TURBO" mode inline tag
	tagStd    lipgloss.Style // "标准" mode inline tag
	// Log entry markers
	logOk  lipgloss.Style
	logErr lipgloss.Style
	// Wizard step indicators
	stepDone   lipgloss.Style
	stepActive lipgloss.Style
	stepTodo   lipgloss.Style
	// Primary action button
	btnPrimary lipgloss.Style
	// Divider / decorative line
	divider lipgloss.Style
	// Content panel (deep-plum background card, like the demo's purple paragraphs)
	panel lipgloss.Style
}

func newStyles() styles {
	return styles{
		// Header: deep indigo-purple background, white foreground
		header: lipgloss.NewStyle().
			Background(colorHeaderBg).
			Foreground(colorWhite),
		// Footer: dark near-black background
		footer: lipgloss.NewStyle().
			Background(colorFooterBg).
			Foreground(colorMuted),
		// Section headings: hot pink, bold
		sectionHead: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		// Table column headers: vivid cyan, bold
		tableHead: lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true),
		// Normal table rows: bright white
		tableRow: lipgloss.NewStyle().
			Foreground(colorWhite),
		// Selected table row: dim-purple bg, white text, bold
		tableRowSel: lipgloss.NewStyle().
			Background(colorPurpleDim).
			Foreground(colorWhite).
			Bold(true),
		// Property labels: soft teal, bold
		label: lipgloss.NewStyle().
			Foreground(colorTeal).
			Bold(true),
		// Property values: bright white
		value: lipgloss.NewStyle().
			Foreground(colorWhite),
		// Secondary/muted text: gray
		muted: lipgloss.NewStyle().
			Foreground(colorMuted),
		// Success indicator: bright green, bold
		ok: lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true),
		// Error indicator: red, bold
		errStyle: lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true),
		// Keyboard shortcut keys: hot pink, bold
		key: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		// Metric numeric values: yellow, bold
		metricVal: lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true),
		// Dialog/modal box: rounded border in hot pink, padded
		dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPink).
			Padding(1, 2),
		// Active wizard field: dark purple-blue bg, white text
		fieldActive: lipgloss.NewStyle().
			Background(colorFieldBg).
			Foreground(colorWhite),
		// Idle wizard field: muted gray text
		fieldIdle: lipgloss.NewStyle().
			Foreground(colorMuted),
		// Cursor/selection arrow: hot pink, bold
		cursor: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		// AIT brand badge: purple bg, white text, padded
		badge: lipgloss.NewStyle().
			Background(colorPurple).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 1),
		// Alternate badge: gold bg, dark text
		badgeAlt: lipgloss.NewStyle().
			Background(colorGold).
			Foreground(colorDark).
			Bold(true).
			Padding(0, 1),
		// TURBO mode tag: pink bg, dark text
		tagTurbo: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		// Standard mode tag: cyan text
		tagStd: lipgloss.NewStyle().
			Foreground(colorCyan),
		// Log ok: green
		logOk: lipgloss.NewStyle().
			Foreground(colorGreen),
		// Log error: red
		logErr: lipgloss.NewStyle().
			Foreground(colorRed),
		// Wizard step done: green checkmark
		stepDone: lipgloss.NewStyle().
			Foreground(colorGreen),
		// Wizard step active: pink, bold
		stepActive: lipgloss.NewStyle().
			Foreground(colorPink).
			Bold(true),
		// Wizard step todo: dim
		stepTodo: lipgloss.NewStyle().
			Foreground(colorMuted),
		// Primary action button: hot-pink bg, dark text, padded
		btnPrimary: lipgloss.NewStyle().
			Background(colorPink).
			Foreground(colorDark).
			Bold(true).
			Padding(0, 2),
		// Divider line: dim gray
		divider: lipgloss.NewStyle().
			Foreground(colorDimBorder),
		// Content panel: deep-plum background, white text, padded (like demo's purple paragraphs)
		panel: lipgloss.NewStyle().
			Background(colorPanelBg).
			Foreground(colorWhite).
			Padding(1, 2),
	}
}
