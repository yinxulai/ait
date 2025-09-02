package display

import (
	"fmt"
	"strings"
	"time"
)

// Colors 定义终端颜色
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

// PrintTitle 打印标题
func PrintTitle(title string) {
	fmt.Printf("\n%s%s=== %s ===%s\n\n", ColorBold, ColorCyan, title, ColorReset)
}

// PrintSection 打印章节
func PrintSection(section string) {
	fmt.Printf("%s%s%s%s\n", ColorBold, ColorYellow, section, ColorReset)
}

// PrintSuccess 打印成功信息
func PrintSuccess(message string) {
	fmt.Printf("%s✓ %s%s\n", ColorGreen, message, ColorReset)
}

// PrintError 打印错误信息
func PrintError(message string) {
	fmt.Printf("%s✗ %s%s\n", ColorRed, message, ColorReset)
}

// PrintWarning 打印警告信息
func PrintWarning(message string) {
	fmt.Printf("%s⚠ %s%s\n", ColorYellow, message, ColorReset)
}

// PrintInfo 打印信息
func PrintInfo(message string) {
	fmt.Printf("%sℹ %s%s\n", ColorBlue, message, ColorReset)
}

// ProgressBar 进度条结构
type ProgressBar struct {
	total   int
	current int
	width   int
	prefix  string
}

// NewProgressBar 创建新的进度条
func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		total:  total,
		width:  50,
		prefix: prefix,
	}
}

// Update 更新进度条
func (pb *ProgressBar) Update(current int) {
	pb.current = current
	pb.render()
}

// Finish 完成进度条
func (pb *ProgressBar) Finish() {
	pb.current = pb.total
	pb.render()
	fmt.Println()
}

// render 渲染进度条
func (pb *ProgressBar) render() {
	percent := float64(pb.current) / float64(pb.total)
	filled := int(percent * float64(pb.width))
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", pb.width-filled)
	
	fmt.Printf("\r%s%s %s[%s]%s %d/%d (%.1f%%)",
		ColorCyan, pb.prefix, ColorGreen, bar, ColorReset, pb.current, pb.total, percent*100)
}

// Table 表格结构
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable 创建新表格
func NewTable(headers []string) *Table {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	
	return &Table{
		headers: headers,
		widths:  widths,
	}
}

// AddRow 添加行
func (t *Table) AddRow(row []string) {
	for i, cell := range row {
		if i < len(t.widths) && len(cell) > t.widths[i] {
			t.widths[i] = len(cell)
		}
	}
	t.rows = append(t.rows, row)
}

// Render 渲染表格
func (t *Table) Render() {
	// 打印顶部边框
	t.printBorder()
	
	// 打印表头
	t.printRow(t.headers, ColorBold+ColorCyan)
	
	// 打印分隔线
	t.printSeparator()
	
	// 打印数据行
	for _, row := range t.rows {
		t.printRow(row, "")
	}
	
	// 打印底部边框
	t.printBorder()
}

// printBorder 打印边框
func (t *Table) printBorder() {
	fmt.Print("┌")
	for i, width := range t.widths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(t.widths)-1 {
			fmt.Print("┬")
		}
	}
	fmt.Println("┐")
}

// printSeparator 打印分隔线
func (t *Table) printSeparator() {
	fmt.Print("├")
	for i, width := range t.widths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(t.widths)-1 {
			fmt.Print("┼")
		}
	}
	fmt.Println("┤")
}

// printRow 打印行
func (t *Table) printRow(row []string, color string) {
	fmt.Print("│")
	for i, cell := range row {
		if i < len(t.widths) {
			fmt.Printf(" %s%-*s%s │", color, t.widths[i], cell, ColorReset)
		}
	}
	fmt.Println()
}

// FormatDuration 格式化时间
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%.0fns", float64(d.Nanoseconds()))
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.2fμs", float64(d.Nanoseconds())/1000)
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1000000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// FormatFloat 格式化浮点数
func FormatFloat(f float64, precision int) string {
	return fmt.Sprintf("%."+fmt.Sprintf("%d", precision)+"f", f)
}
