package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

func TestCorePagesRenderExpectedContent(t *testing.T) {
	st := NewStyles()

	reqDetail := &ReqDetailState{
		RunID: server.RunID("test"),
		Requests: []*types.RequestMetrics{{
			Success: true, TotalTime: 250 * time.Millisecond,
			RequestBody: "hello", ResponseBody: strings.Repeat("ok ", 50),
		}},
	}
	out := stripANSI(RenderReqDetail(reqDetail, "task", st, 80, 24))
	for _, want := range []string{"查看请求详情", "hello", "请求体"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ReqDetail should render %q in output", want)
		}
	}

	dashboard := NewDashboardState("run1", "task1")
	out = stripANSI(RenderDashboard(dashboard, "task", st, 80, 22))
	for _, want := range []string{"task1", "等待"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Dashboard should render %q in output", want)
		}
	}

	tasks := &TaskListState{}
	out = stripANSI(RenderTaskList(tasks, st, 80, 24))
	for _, want := range []string{"任务中心", "运行中"} {
		if !strings.Contains(out, want) {
			t.Fatalf("TaskList should render %q in output", want)
		}
	}
}

func TestReqDetailRendersCJKContent(t *testing.T) {
	st := NewStyles()
	cjkBody := strings.Repeat("你好，我是一个大型语言模型，很高兴为你服务。", 10)
	s := &ReqDetailState{
		RunID: server.RunID("test"),
		Requests: []*types.RequestMetrics{{
			Success: true, TotalTime: 500 * time.Millisecond,
			RequestBody:  "请介绍一下自己",
			ResponseBody: cjkBody,
		}},
	}
	out := stripANSI(RenderReqDetail(s, "task", st, 80, 24))
	for _, want := range []string{"请介绍一下自己", "请求体"} {
		if !strings.Contains(out, want) {
			t.Fatalf("ReqDetail should render CJK content %q in output", want)
		}
	}
}
