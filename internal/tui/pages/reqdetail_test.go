package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

func TestRenderReqDetailKeepsSameHeightForSuccessAndFailure(t *testing.T) {
	st := NewStyles()
	success := &ReqDetailState{
		RunID: server.RunID("run_success"),
		Requests: []*types.RequestMetrics{{
			Success:          true,
			TotalTime:        250 * time.Millisecond,
			TTFT:             80 * time.Millisecond,
			TPS:              12.5,
			PromptTokens:     64,
			CompletionTokens: 128,
			CachedTokens:     32,
			CacheHitRate:     0.5,
			DNSTime:          2 * time.Millisecond,
			ConnectTime:      3 * time.Millisecond,
			TLSTime:          4 * time.Millisecond,
			TargetIP:         "1.2.3.4",
			RequestBody:      "hello",
			ResponseBody:     strings.Repeat("ok ", 50),
		}},
	}
	failure := &ReqDetailState{
		RunID: server.RunID("run_failure"),
		Requests: []*types.RequestMetrics{{
			Success:      false,
			TotalTime:    250 * time.Millisecond,
			DNSTime:      2 * time.Millisecond,
			ConnectTime:  3 * time.Millisecond,
			TLSTime:      4 * time.Millisecond,
			RequestBody:  "hello",
			ResponseBody: "",
			ErrorMessage: "dial tcp:\nlookup api.example.com: no such host",
		}},
	}

	successLines := strings.Split(stripANSI(RenderReqDetail(success, "示例任务", st, 96, 30)), "\n")
	failureLines := strings.Split(stripANSI(RenderReqDetail(failure, "示例任务", st, 96, 30)), "\n")
	if len(successLines) != len(failureLines) {
		t.Fatalf("expected success/failure render heights to match, got %d vs %d", len(successLines), len(failureLines))
	}

	if strings.Contains(stripANSI(RenderReqDetail(failure, "示例任务", st, 96, 30)), "dial tcp:\n") {
		t.Fatalf("expected failure error summary to be normalized into a single visual line")
	}
}
