package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type tableColumn struct {
	Header    string
	MaxWidth  int
	Expansion int
	Align     int
}

func newSelectableTable(title string) *tview.Table {
	table := tview.NewTable().
		SetSelectable(true, false).
		SetFixed(1, 0).
		SetSeparator(' ').
		SetWrapSelection(false, false)
	table.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	table.SetSelectedStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkBlue))
	return table
}

func selectedDataIndex(table *tview.Table) int {
	row, _ := table.GetSelection()
	return row - 1
}

func selectDataIndex(table *tview.Table, idx int) {
	if table.GetRowCount() <= 1 {
		table.Select(0, 0)
		return
	}
	if idx < 0 {
		idx = 0
	}
	maxIdx := table.GetRowCount() - 2
	if idx > maxIdx {
		idx = maxIdx
	}
	table.Select(idx+1, 0)
}

func setTableRows(table *tview.Table, columns []tableColumn, rows [][]string) {
	table.Clear()
	for col, column := range columns {
		header := tview.NewTableCell(column.Header).
			SetSelectable(false).
			SetStyle(tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)).
			SetAlign(column.Align).
			SetMaxWidth(column.MaxWidth).
			SetExpansion(column.Expansion)
		table.SetCell(0, col, header)
	}
	for rowIdx, row := range rows {
		for col, text := range row {
			if col >= len(columns) {
				continue
			}
			column := columns[col]
			cell := tview.NewTableCell(text).
				SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite)).
				SetAlign(column.Align).
				SetMaxWidth(column.MaxWidth).
				SetExpansion(column.Expansion)
			table.SetCell(rowIdx+1, col, cell)
		}
	}
	selectDataIndex(table, 0)
}

func taskColumns() []tableColumn {
	return []tableColumn{
		{Header: "Task", MaxWidth: 30, Expansion: 2, Align: tview.AlignLeft},
		{Header: "Mode", MaxWidth: 12, Expansion: 1, Align: tview.AlignLeft},
		{Header: "Endpoint", MaxWidth: 42, Expansion: 3, Align: tview.AlignLeft},
		{Header: "Load", MaxWidth: 24, Expansion: 2, Align: tview.AlignLeft},
	}
}

func runColumns() []tableColumn {
	return []tableColumn{
		{Header: "Run", MaxWidth: 12, Expansion: 1, Align: tview.AlignLeft},
		{Header: "Status", MaxWidth: 12, Expansion: 1, Align: tview.AlignLeft},
		{Header: "Time", MaxWidth: 24, Expansion: 2, Align: tview.AlignLeft},
		{Header: "Success", MaxWidth: 12, Expansion: 1, Align: tview.AlignLeft},
		{Header: "Throughput", MaxWidth: 32, Expansion: 3, Align: tview.AlignLeft},
	}
}

func requestColumns() []tableColumn {
	return []tableColumn{
		{Header: "Request", MaxWidth: 16, Expansion: 1, Align: tview.AlignLeft},
		{Header: "Latency", MaxWidth: 26, Expansion: 2, Align: tview.AlignLeft},
		{Header: "Throughput", MaxWidth: 32, Expansion: 3, Align: tview.AlignLeft},
		{Header: "Network", MaxWidth: 36, Expansion: 2, Align: tview.AlignLeft},
		{Header: "Problem", MaxWidth: 34, Expansion: 2, Align: tview.AlignLeft},
	}
}
