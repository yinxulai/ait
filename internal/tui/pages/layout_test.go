package pages

import (
	"strings"
	"testing"
)

func TestPageLayoutAssembleRendersSharedChrome(t *testing.T) {
	st := NewStyles()
	l := PageLayout{
		HeaderTitle:     "任务中心",
		HeaderSubtitle:  "创建任务、运行压测、查看执行记录与导出报告",
		HeaderMeta:      "2 个任务",
		HeaderInfoLeft:  []string{"运行中 1"},
		HeaderInfoRight: []string{"最近运行 1 分钟前"},
		Hotkeys: NewPageHotkeys([]HotkeyItem{
			HotkeyAction("Enter", "查看详情"),
		}, "[q] 退出"),
	}

	rendered := stripANSI(l.Assemble("content", st, 80))
	lines := strings.Split(rendered, "\n")
	if len(lines) < 6 {
		t.Fatalf("expected shared chrome to add header/hotkeys lines, got %d lines", len(lines))
	}
	if !strings.Contains(rendered, "AIT") || !strings.Contains(rendered, "任务中心") {
		t.Fatalf("expected header brand/title in output: %q", rendered)
	}
	if !strings.Contains(rendered, "创建任务、运行压测") {
		t.Fatalf("expected header subtitle in output: %q", rendered)
	}
	if !strings.Contains(rendered, "查看详情") || !strings.Contains(rendered, "[q] 退出") {
		t.Fatalf("expected hotkey actions and global hints in output: %q", rendered)
	}
}

func TestPageLayoutFrameCalculatesNestedPanelSizes(t *testing.T) {
	l := PageLayout{}
	frame := l.Frame(80, 30)
	if frame.OuterWidth != 80 || frame.InnerWidth != 80 || frame.InnerHeight != 25 {
		t.Fatalf("unexpected page frame: %#v", frame)
	}

	body := frame.InnerPanel()
	if body.OuterWidth != 80 || body.InnerWidth != 78 {
		t.Fatalf("unexpected inner panel frame: %#v", body)
	}
}

func TestRemainingStackOuterHeightAccountsForJoinGaps(t *testing.T) {
	totalHeight := 24
	remaining := RemainingStackOuterHeight(totalHeight, 9, 3)
	if remaining != 12 {
		t.Fatalf("expected remaining outer height 12, got %d", remaining)
	}

	// strings.Join 拼接时总行数 = 各块行数之和，分隔符 \n 不额外增加行数
	used := 9 + 3 + remaining
	if used != totalHeight {
		t.Fatalf("expected stacked blocks to fit exactly, used %d of %d", used, totalHeight)
	}

	if PanelContentHeight(remaining) != 10 {
		t.Fatalf("expected remaining content height 10, got %d", PanelContentHeight(remaining))
	}
}
