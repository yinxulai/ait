package tui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	title    lipgloss.Style
	subtitle lipgloss.Style
	panel    lipgloss.Style
	selected lipgloss.Style
	muted    lipgloss.Style
	error    lipgloss.Style
	ok       lipgloss.Style
	key      lipgloss.Style
	label    lipgloss.Style
	value    lipgloss.Style
}

func newStyles() styles {
	border := lipgloss.RoundedBorder()
	return styles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1),
		subtitle: lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Padding(0, 1),
		panel:    lipgloss.NewStyle().Border(border).BorderForeground(lipgloss.Color("62")).Padding(1, 2),
		selected: lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Bold(true),
		muted:    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		error:    lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true),
		ok:       lipgloss.NewStyle().Foreground(lipgloss.Color("85")).Bold(true),
		key:      lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		label:    lipgloss.NewStyle().Foreground(lipgloss.Color("151")).Bold(true),
		value:    lipgloss.NewStyle().Foreground(lipgloss.Color("255")),
	}
}
