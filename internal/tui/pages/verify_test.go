package pages

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

func TestHeightCorrectness(t *testing.T) {
	st := NewStyles()

	// ReqDetail
	s := &ReqDetailState{
		RunID: server.RunID("test"),
		Requests: []*types.RequestMetrics{{
			Success: true, TotalTime: 250 * time.Millisecond,
			RequestBody: "hello", ResponseBody: strings.Repeat("ok ", 50),
		}},
	}
	fmt.Println("--- ReqDetail ---")
	for _, h := range []int{24, 26, 30, 40} {
		out := RenderReqDetail(s, "task", st, 80, h)
		got := strings.Count(out, "\n") + 1
		diff := got - h
		marker := "✓"
		if diff != 0 {
			marker = fmt.Sprintf("FAIL diff=%+d", diff)
		}
		fmt.Printf("height=%d → rendered=%d %s\n", h, got, marker)
		if diff != 0 {
			t.Errorf("ReqDetail height=%d: want %d lines, got %d", h, h, got)
		}
	}

	// Dashboard
	fmt.Println("--- Dashboard ---")
	ds := NewDashboardState("run1", "task1")
	for _, h := range []int{22, 26, 30, 40} {
		out := RenderDashboard(ds, "task", st, 80, h)
		got := strings.Count(out, "\n") + 1
		diff := got - h
		marker := "✓"
		if diff != 0 {
			marker = fmt.Sprintf("FAIL diff=%+d", diff)
		}
		fmt.Printf("height=%d → rendered=%d %s\n", h, got, marker)
		if diff != 0 {
			t.Errorf("Dashboard height=%d: want %d lines, got %d", h, h, got)
		}
	}

	// TaskList (empty tasks)
	fmt.Println("--- TaskList (empty) ---")
	ts := &TaskListState{}
	for _, h := range []int{24, 26, 30, 40} {
		out := RenderTaskList(ts, st, 80, h)
		got := strings.Count(out, "\n") + 1
		diff := got - h
		marker := "✓"
		if diff != 0 {
			marker = fmt.Sprintf("FAIL diff=%+d", diff)
		}
		fmt.Printf("height=%d → rendered=%d %s\n", h, got, marker)
		if diff != 0 {
			t.Errorf("TaskList height=%d: want %d lines, got %d", h, h, got)
		}
	}
}

func TestHeightWithCJKContent(t *testing.T) {
	st := NewStyles()
	// 模拟真实 LLM 响应：纯中文内容
	cjkBody := strings.Repeat("你好，我是一个大型语言模型，很高兴为你服务。", 10)
	s := &ReqDetailState{
		RunID: server.RunID("test"),
		Requests: []*types.RequestMetrics{{
			Success: true, TotalTime: 500 * time.Millisecond,
			RequestBody:  "请介绍一下自己",
			ResponseBody: cjkBody,
		}},
	}
	for _, h := range []int{24, 30, 40} {
		out := RenderReqDetail(s, "task", st, 80, h)
		got := strings.Count(out, "\n") + 1
		diff := got - h
		marker := "✓"
		if diff != 0 {
			marker = fmt.Sprintf("FAIL diff=%+d", diff)
		}
		t.Logf("CJK ReqDetail height=%d → rendered=%d %s", h, got, marker)
		if diff != 0 {
			t.Errorf("CJK content: height=%d want %d lines, got %d", h, h, got)
		}
	}
}
