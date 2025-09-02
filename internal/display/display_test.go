package display

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// captureOutput 捕获标准输出，用于测试
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintTitle(t *testing.T) {
	output := captureOutput(func() {
		PrintTitle("Test Title")
	})

	if !strings.Contains(output, "Test Title") {
		t.Errorf("PrintTitle() output should contain 'Test Title', got: %s", output)
	}

	if !strings.Contains(output, "===") {
		t.Errorf("PrintTitle() output should contain '===', got: %s", output)
	}
}

func TestPrintSection(t *testing.T) {
	output := captureOutput(func() {
		PrintSection("Test Section")
	})

	if !strings.Contains(output, "Test Section") {
		t.Errorf("PrintSection() output should contain 'Test Section', got: %s", output)
	}
}

func TestPrintSuccess(t *testing.T) {
	output := captureOutput(func() {
		PrintSuccess("Test success message")
	})

	if !strings.Contains(output, "Test success message") {
		t.Errorf("PrintSuccess() output should contain message, got: %s", output)
	}

	if !strings.Contains(output, "✓") {
		t.Errorf("PrintSuccess() output should contain checkmark, got: %s", output)
	}
}

func TestPrintError(t *testing.T) {
	output := captureOutput(func() {
		PrintError("Test error message")
	})

	if !strings.Contains(output, "Test error message") {
		t.Errorf("PrintError() output should contain message, got: %s", output)
	}

	if !strings.Contains(output, "✗") {
		t.Errorf("PrintError() output should contain X mark, got: %s", output)
	}
}

func TestPrintWarning(t *testing.T) {
	output := captureOutput(func() {
		PrintWarning("Test warning message")
	})

	if !strings.Contains(output, "Test warning message") {
		t.Errorf("PrintWarning() output should contain message, got: %s", output)
	}

	if !strings.Contains(output, "⚠") {
		t.Errorf("PrintWarning() output should contain warning symbol, got: %s", output)
	}
}

func TestPrintInfo(t *testing.T) {
	output := captureOutput(func() {
		PrintInfo("Test info message")
	})

	if !strings.Contains(output, "Test info message") {
		t.Errorf("PrintInfo() output should contain message, got: %s", output)
	}

	if !strings.Contains(output, "ℹ") {
		t.Errorf("PrintInfo() output should contain info symbol, got: %s", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "milliseconds",
			duration: 150 * time.Millisecond,
			want:     "150.00ms",
		},
		{
			name:     "seconds",
			duration: 2500 * time.Millisecond,
			want:     "2.50s",
		},
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			want:     "500.00μs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("FormatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision int
		want      string
	}{
		{
			name:      "two decimal places",
			value:     123.456789,
			precision: 2,
			want:      "123.46",
		},
		{
			name:      "zero decimal places",
			value:     123.456789,
			precision: 0,
			want:      "123",
		},
		{
			name:      "four decimal places",
			value:     123.456789,
			precision: 4,
			want:      "123.4568",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFloat(tt.value, tt.precision)
			if got != tt.want {
				t.Errorf("FormatFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTable(t *testing.T) {
	headers := []string{"Header1", "Header2", "Header3"}
	table := NewTable(headers)

	if table == nil {
		t.Error("NewTable() should not return nil")
	}

	if len(table.headers) != len(headers) {
		t.Errorf("NewTable() headers length = %v, want %v", len(table.headers), len(headers))
	}

	for i, header := range headers {
		if table.headers[i] != header {
			t.Errorf("NewTable() header[%d] = %v, want %v", i, table.headers[i], header)
		}
	}
}

func TestTable_AddRow(t *testing.T) {
	table := NewTable([]string{"Col1", "Col2"})
	row := []string{"Value1", "Value2"}

	table.AddRow(row)

	if len(table.rows) != 1 {
		t.Errorf("AddRow() table.rows length = %v, want %v", len(table.rows), 1)
	}

	if len(table.rows[0]) != len(row) {
		t.Errorf("AddRow() row length = %v, want %v", len(table.rows[0]), len(row))
	}
}

func TestNewProgressBar(t *testing.T) {
	total := 100
	prefix := "Test Progress"
	pb := NewProgressBar(total, prefix)

	if pb == nil {
		t.Error("NewProgressBar() should not return nil")
	}

	if pb.total != total {
		t.Errorf("NewProgressBar() total = %v, want %v", pb.total, total)
	}

	if pb.prefix != prefix {
		t.Errorf("NewProgressBar() prefix = %v, want %v", pb.prefix, prefix)
	}

	if pb.current != 0 {
		t.Errorf("NewProgressBar() current = %v, want %v", pb.current, 0)
	}
}
