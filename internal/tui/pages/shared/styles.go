// Package shared provides reusable TUI styling primitives for all pages.
//
// Style philosophy: lazydocker-inspired — rounded borders,
// subdued colors, green accents for active elements, no frame on the bottom bar.
package shared

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ─── Color palette ──────────────────────────────────────────────────────────

const (
	ColorPrimary = "#00d7af" // teal accent
	ColorGreen   = "#5faf5f" // success, running
	ColorRed     = "#d75f5f" // error, failed
	ColorYellow  = "#d7af5f" // warning, turbo
	ColorMagenta = "#af5faf" // purple accent
	ColorCyan    = "#5fafaf" // info

	ColorHeading = "#d0d0d0" // bright text
	ColorBody    = "#b0b0b0" // body text
	ColorMuted   = "#707070" // dim text
	ColorFaint   = "#505050" // very dim (divider lines)
)

// ─── Width helpers ──────────────────────────────────────────────────────────

// StrWidth returns the visual column width of s (CJK = 2).
func StrWidth(s string) int {
	w := 0
	for _, r := range s {
		if r > 0x2E80 {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// PadRight pads s with spaces to visual width n.
func PadRight(s string, n int) string {
	pw := StrWidth(s)
	if pw >= n {
		return s
	}
	return s + strings.Repeat(" ", n-pw)
}

// Truncate truncates s to max visual width.
func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	w := 0
	runes := []rune(s)
	for i, r := range runes {
		rw := 1
		if r > 0x2E80 {
			rw = 2
		}
		if w+rw > max {
			return string(runes[:i]) + "…"
		}
		w += rw
	}
	return s
}

// ─── Formatting ─────────────────────────────────────────────────────────────

// FmtDuration formats a duration in compact form.
func FmtDuration(d time.Duration) string {
	if d < time.Microsecond {
		return "0s"
	}
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%.0fµs", float64(d.Nanoseconds())/1000)
	case d < time.Second:
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d < time.Hour:
		m := int(d.Minutes())
		s := d.Seconds() - float64(m*60)
		return fmt.Sprintf("%dm%ds", m, int(s))
	default:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", h, m)
	}
}

// FmtRelTime formats a time relative to now.
func FmtRelTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = -d
	}
	switch {
	case d < 2*time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// ─── Status helpers ─────────────────────────────────────────────────────────

// StatusIcon returns a single-char status indicator.
func StatusIcon(running, success bool) string {
	switch {
	case running:
		return "[green]●"
	case success:
		return "[green]✓"
	default:
		return "[red]✗"
	}
}

// RunStatusTag returns a color-coded run status tag.
func RunStatusTag(status string) string {
	switch strings.ToLower(status) {
	case "running":
		return "[green]▶ RUNNING"
	case "completed":
		return "[green]✓ DONE"
	case "failed":
		return "[red]✗ FAILED"
	case "stopped":
		return "[yellow]⊘ STOPPED"
	default:
		return "[gray]· " + strings.ToUpper(status)
	}
}

// ModeBadge returns a styled mode badge.
func ModeBadge(turbo bool) string {
	if turbo {
		return "[#d7af5f]TURBO"
	}
	return "[#af5faf]STD"
}

// ─── Panel helpers ──────────────────────────────────────────────────────────

// NewPanel creates a bordered text view with subdued colors.
func NewPanel(title string) *tview.TextView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWordWrap(true)
	tv.SetBorder(true).
		SetBorderColor(tcell.ColorGray).
		SetBorderPadding(0, 0, 1, 1)
	if title != "" {
		tv.SetTitle(" " + title + " ")
		tv.SetTitleColor(tcell.ColorGray)
	}
	return tv
}

// NewPanelFixed creates a non-scrollable bordered text view.
func NewPanelFixed(title string) *tview.TextView {
	tv := NewPanel(title)
	tv.SetScrollable(false)
	return tv
}

// ─── Table helpers ──────────────────────────────────────────────────────────

// ZebraRow returns a subtle alt-row background.
func ZebraRow(row int) tcell.Color {
	if row%2 == 0 {
		return tcell.ColorDefault
	}
	return tcell.NewRGBColor(24, 24, 40)
}

// TableHdr returns a styled table cell for headers.
func TableHdr(text string) *tview.TableCell {
	return tview.NewTableCell(text).
		SetTextColor(tcell.ColorAquaMarine).
		SetSelectable(false).
		SetAttributes(tcell.AttrBold)
}

// TableCell returns a colored data cell.
func TableCell(text string, c tcell.Color) *tview.TableCell {
	return tview.NewTableCell(text).SetTextColor(c)
}

// TableCellMuted returns a gray data cell.
func TableCellMuted(text string) *tview.TableCell {
	return NewTableCellGray(text)
}

// NewTableCellGray returns a cell with gray text.
func NewTableCellGray(text string) *tview.TableCell {
	return tview.NewTableCell(text).SetTextColor(tcell.ColorGray)
}

// NewTableCellGreen/Red/Yellow return status-colored cells.
func NewTableCellGreen(text string) *tview.TableCell {
	return tview.NewTableCell(text).SetTextColor(tcell.ColorGreen)
}
func NewTableCellRed(text string) *tview.TableCell {
	return tview.NewTableCell(text).SetTextColor(tcell.ColorRed)
}
func NewTableCellYellow(text string) *tview.TableCell {
	return tview.NewTableCell(text).SetTextColor(tcell.ColorYellow)
}

// ─── Progress bar ───────────────────────────────────────────────────────────

// ProgressBar returns a text progress bar of given width.
func ProgressBar(done, total int, width int) string {
	if total == 0 {
		return "[#505050]" + strings.Repeat("━", width)
	}
	ratio := float64(done) / float64(total)
	if ratio > 1 {
		ratio = 1
	}
	filled := int(math.Round(ratio * float64(width)))
	empty := width - filled
	return fmt.Sprintf("[green]%s[#333333]%s",
		strings.Repeat("━", filled), strings.Repeat("━", empty))
}

// ─── Version ────────────────────────────────────────────────────────────────

var appVersion = "dev"

// SetAppVersion sets the version string shown in the TUI.
func SetAppVersion(v string) { appVersion = v }

// AppVersion returns the version string.
func AppVersion() string { return appVersion }
