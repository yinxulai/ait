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
	if (!strings.Contains(rendered, "AIT") && !strings.Contains(rendered, "████")) || !strings.Contains(rendered, "任务中心") {
		t.Fatalf("expected header brand/title in output: %q", rendered)
	}
	if !strings.Contains(rendered, "创建任务、运行压测") {
		t.Fatalf("expected header subtitle in output: %q", rendered)
	}
	if !strings.Contains(rendered, "查看详情") || !strings.Contains(rendered, "[q] 退出") {
		t.Fatalf("expected hotkey actions and global hints in output: %q", rendered)
	}
}

func TestPageLayoutFrameStructure(t *testing.T) {
	l := PageLayout{}
	frame := l.Frame(80, 30)

	// 所有尺寸必须为正
	if frame.OuterWidth <= 0 || frame.InnerWidth <= 0 || frame.InnerHeight <= 0 {
		t.Fatalf("frame dimensions must be positive: %#v", frame)
	}
	// InnerWidth 不应超过 OuterWidth
	if frame.InnerWidth > frame.OuterWidth {
		t.Fatalf("InnerWidth (%d) must not exceed OuterWidth (%d)", frame.InnerWidth, frame.OuterWidth)
	}
	// InnerHeight 应小于总高度（chrome 和边框占用后）
	if frame.InnerHeight >= 30 {
		t.Fatalf("InnerHeight (%d) must be less than total height (30)", frame.InnerHeight)
	}

	body := frame.InnerPanel()
	if body.OuterWidth <= 0 || body.InnerWidth <= 0 {
		t.Fatalf("panel dimensions must be positive: %#v", body)
	}
	if body.InnerWidth >= body.OuterWidth {
		t.Fatalf("panel InnerWidth (%d) must be less than OuterWidth (%d)", body.InnerWidth, body.OuterWidth)
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
