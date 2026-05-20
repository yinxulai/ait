package pages

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinxulai/ait/internal/server"
)

var ansiRE = regexp.MustCompile("\\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestRenderWizardField_CentersLabelOnInputRow(t *testing.T) {
	st := NewStyles()
	wz := NewWizardState()
	wz.Name = "demo"
	field := step1Fields()[0]

	rendered := stripANSI(renderWizardField(st, field, wz, true, 80))
	lines := strings.Split(rendered, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if strings.Contains(lines[0], "任务名称") {
		t.Fatalf("label should not be rendered on top border line: %q", lines[0])
	}
	if !strings.Contains(lines[1], "任务名称") {
		t.Fatalf("label should be rendered on the input content line: %q", lines[1])
	}
}

func TestHandleDashboardKey_BackPreservesSubscription(t *testing.T) {
	called := false
	ch := make(chan server.Event)
	d := &DashboardState{
		EventCh:  ch,
		CancelFn: func() { called = true },
		BackNav:  NavAction{To: NavTaskDetail},
	}

	_, _, nav := HandleDashboardKey(d, tea.KeyMsg{Type: tea.KeyEsc}, nil)
	if nav.To != NavTaskDetail {
		t.Fatalf("nav.To = %v, want %v", nav.To, NavTaskDetail)
	}
	if called {
		t.Fatal("CancelFn should not be called when returning to previous page")
	}
	if d.EventCh != ch {
		t.Fatal("EventCh should be preserved when returning to previous page")
	}
	if d.CancelFn == nil {
		t.Fatal("CancelFn should remain set when returning to previous page")
	}
	close(ch)
}
