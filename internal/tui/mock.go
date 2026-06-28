package tui

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/yinxulai/ait/internal/server/types"
)

// ─── 格式化函数 ────────────────────────────────────────────────────────────────

func formatTaskLine(t types.TaskOverview) (main, sec string) {
	main = t.Name
	if t.LatestRun != nil {
		sec = fmt.Sprintf("%s  | SR %.1f%%  | TPS %.0f",
			t.Input.RunMode(), t.LatestRun.SuccessRate*100, t.LatestRun.AvgTPS)
	} else {
		sec = t.Input.RunMode() + "  | -"
	}
	return
}

func formatRunLine(r types.TaskRunSummary) (main, sec string) {
	shortID := r.RunID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	main = shortID
	dur := r.FinishedAt.Sub(r.StartedAt).Truncate(time.Second)
	sec = fmt.Sprintf("%s | %s | %s | SR %.1f%% | TPS %.0f",
		r.Mode,
		r.StartedAt.Format("01-02 15:04"),
		dur,
		r.SuccessRate*100,
		r.AvgTPS,
	)
	return
}

func formatReqLine(r types.RequestMetrics) (main, sec string) {
	status := "✅"
	if !r.Success {
		status = "❌"
	}
	total := r.TotalTime.Truncate(time.Millisecond)
	ttft := r.TTFT.Truncate(time.Millisecond)
	if !r.Success {
		main = fmt.Sprintf("#%03d %s  %s  TTFT %s  TPS %.0f",
			r.Index, status, total, "-", r.TPS)
		sec = fmt.Sprintf("ERR: %s", r.ErrorMessage)
	} else {
		main = fmt.Sprintf("#%03d %s  %s  TTFT %s  TPS %.0f",
			r.Index, status, total, ttft, r.TPS)
		sec = fmt.Sprintf("PT %d  CT %d  CH %.0f%%",
			r.PromptTokens, r.CompletionTokens, r.CacheHitRate)
	}
	return
}

func renderProgressBar(done, total int, width int) string {
	if total <= 0 {
		return strings.Repeat("░", width)
	}
	ratio := float64(done) / float64(total)
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func formatStats(reqs []types.RequestMetrics, totalReqs int, isCompleted bool, elapsed time.Duration) string {
	done := len(reqs)
	success := 0
	var sumTPS, sumCacheRate float64
	var ttfts []time.Duration
	var totalOutTokens int

	for _, r := range reqs {
		if r.Success {
			success++
		}
		sumTPS += r.TPS
		sumCacheRate += r.CacheHitRate
		ttfts = append(ttfts, r.TTFT)
		totalOutTokens += r.CompletionTokens
	}

	avgTPS := 0.0
	avgCacheRate := 0.0
	p50TTFT := time.Duration(0)
	p99TTFT := time.Duration(0)
	if done > 0 {
		avgTPS = sumTPS / float64(done)
		avgCacheRate = sumCacheRate / float64(done)
		sort.Slice(ttfts, func(i, j int) bool { return ttfts[i] < ttfts[j] })
		p50TTFT = ttfts[len(ttfts)*50/100]
		p99TTFT = ttfts[len(ttfts)*99/100]
	}
	successRate := 0.0
	if done > 0 {
		successRate = float64(success) / float64(done) * 100
	}

	// RPM / TPM
	rpm := 0.0
	tpm := 0.0
	if elapsed > 0 {
		rpm = float64(done) / elapsed.Minutes()
		tpm = float64(totalOutTokens) / elapsed.Minutes()
	}

	barWidth := 20
	pct := 100
	if !isCompleted && totalReqs > 0 {
		pct = done * 100 / totalReqs
	}
	bar := renderProgressBar(done, totalReqs, barWidth)

	// 时间标签
	timeLabel := ""
	if isCompleted && elapsed > 0 {
		timeLabel = fmt.Sprintf(" 耗时: %s", elapsed.Truncate(time.Second))
	} else if !isCompleted && elapsed > 0 && done > 0 && totalReqs > done {
		estimated := time.Duration(float64(elapsed)/float64(done)*float64(totalReqs-done)).Truncate(time.Second)
		timeLabel = fmt.Sprintf(" 预计剩余: %s", estimated)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s  %d%% (%d / %d)%s\n", bar, pct, done, totalReqs, timeLabel))
	sb.WriteString(strings.Repeat("─", 50) + "\n")
	sb.WriteString(fmt.Sprintf("总请求数: %d    成功率: %.1f%%\n", totalReqs, successRate))
	sb.WriteString(fmt.Sprintf("完成: %d  失败: %d\n", done, done-success))
	sb.WriteString(fmt.Sprintf("平均 TPS: %.1f     缓存命中率: %.1f%%\n", avgTPS, avgCacheRate))
	sb.WriteString(fmt.Sprintf("P50 TTFT: %s     P99 TTFT: %s\n",
		p50TTFT.Truncate(time.Millisecond), p99TTFT.Truncate(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("RPM: %.0f           TPM: %.0f\n", rpm, tpm))
	return sb.String()
}

// ─── 模拟数据生成 ──────────────────────────────────────────────────────────────

func generateTaskOverviews() []types.TaskOverview {
	now := time.Now()
	return []types.TaskOverview{
		{
			TaskDefinition: types.TaskDefinition{
				ID:   "task-001",
				Name: "高并发压测-gpt4",
				Input: types.Input{
					Mode:        "standard",
					Protocol:    "openai-completions",
					Model:       "gpt-4",
					Concurrency: 32,
					Count:       10000,
				},
				CreatedAt: now.Add(-24 * time.Hour),
				UpdatedAt: now.Add(-1 * time.Hour),
			},
			LatestRun: &types.TaskRunSummary{
				RunID:       "run-a1b2c3d4",
				TaskID:      "task-001",
				Mode:        "standard",
				Status:      "completed",
				Protocol:    "openai-completions",
				Model:       "gpt-4",
				StartedAt:   now.Add(-2 * time.Hour),
				FinishedAt:  now.Add(-1*time.Hour - 45*time.Minute),
				SuccessRate: 0.985,
				AvgTTFT:     120 * time.Millisecond,
				AvgTPS:      245,
				CacheHitRate: 0.452,
			},
		},
		{
			TaskDefinition: types.TaskDefinition{
				ID:   "task-002",
				Name: "Turbo 爬坡测试",
				Input: types.Input{
					Mode:     "turbo",
					Protocol: "openai-completions",
					Model:    "gpt-3.5-turbo",
				},
				CreatedAt: now.Add(-12 * time.Hour),
				UpdatedAt: now.Add(-30 * time.Minute),
			},
			LatestRun: &types.TaskRunSummary{
				RunID:                "run-e5f6g7h8",
				TaskID:               "task-002",
				Mode:                 "turbo",
				Status:               "completed",
				Protocol:             "openai-completions",
				Model:                "gpt-3.5-turbo",
				StartedAt:            now.Add(-3 * time.Hour),
				FinishedAt:           now.Add(-2*time.Hour - 30*time.Minute),
				SuccessRate:          0.978,
				AvgTTFT:              200 * time.Millisecond,
				AvgTPS:               512,
				CacheHitRate:         0.321,
				MaxStableConcurrency: 24,
			},
		},
		{
			TaskDefinition: types.TaskDefinition{
				ID:   "task-003",
				Name: "新建任务-未运行",
				Input: types.Input{
					Mode:     "standard",
					Protocol: "openai-responses",
					Model:    "gpt-4o",
				},
				CreatedAt: now.Add(-1 * time.Hour),
				UpdatedAt: now.Add(-1 * time.Hour),
			},
		},
	}
}

func generateRunSummaries() []types.TaskRunSummary {
	now := time.Now()
	return []types.TaskRunSummary{
		{
			RunID:       "run-a1b2c3d4",
			TaskID:      "task-001",
			Mode:        "standard",
			Status:      "completed",
			Protocol:    "openai-completions",
			Model:       "gpt-4",
			StartedAt:   now.Add(-2 * time.Hour),
			FinishedAt:  now.Add(-1*time.Hour - 45*time.Minute),
			SuccessRate: 0.985,
			AvgTTFT:     120 * time.Millisecond,
			AvgTPS:      245,
			CacheHitRate: 0.452,
		},
		{
			RunID:       "run-e5f6g7h8",
			TaskID:      "task-002",
			Mode:        "turbo",
			Status:      "completed",
			Protocol:    "openai-completions",
			Model:       "gpt-3.5-turbo",
			StartedAt:   now.Add(-3 * time.Hour),
			FinishedAt:  now.Add(-2*time.Hour - 30*time.Minute),
			SuccessRate: 0.978,
			AvgTTFT:     200 * time.Millisecond,
			AvgTPS:      512,
			CacheHitRate: 0.321,
		},
		{
			RunID:       "run-z9y8x7w6",
			TaskID:      "task-001",
			Mode:        "standard",
			Status:      "running",
			Protocol:    "openai-completions",
			Model:       "gpt-4",
			StartedAt:   now.Add(-5 * time.Minute),
			SuccessRate: 0.99,
			AvgTTFT:     115 * time.Millisecond,
			AvgTPS:      260,
			CacheHitRate: 0.48,
		},
		{
			RunID:       "run-v5u4t3s2",
			TaskID:      "task-001",
			Mode:        "standard",
			Status:      "completed",
			Protocol:    "openai-completions",
			Model:       "gpt-4",
			StartedAt:   now.Add(-5 * time.Hour),
			FinishedAt:  now.Add(-4*time.Hour - 50*time.Minute),
			SuccessRate: 0.972,
			AvgTTFT:     135 * time.Millisecond,
			AvgTPS:      230,
			CacheHitRate: 0.44,
		},
	}
}

func generateRequestMetrics(n int) []types.RequestMetrics {
	reqs := make([]types.RequestMetrics, n)
	for i := 0; i < n; i++ {
		success := rand.Intn(100) > 2 // 98% 成功率
		reqs[i] = types.RequestMetrics{
			Index:           i + 1,
			Success:         success,
			TotalTime:       time.Duration(150+rand.Intn(200)) * time.Millisecond,
			TTFT:            time.Duration(30+rand.Intn(100)) * time.Millisecond,
			TPS:             200 + rand.Float64()*500,
			PromptTokens:    1024 + rand.Intn(2048),
			CompletionTokens: 128 + rand.Intn(512),
			CachedTokens:    rand.Intn(512),
			CacheHitRate:    20 + rand.Float64()*60,
			DNSTime:         time.Duration(1+rand.Intn(5)) * time.Millisecond,
			ConnectTime:     time.Duration(2+rand.Intn(10)) * time.Millisecond,
			TLSTime:         time.Duration(5+rand.Intn(20)) * time.Millisecond,
			TargetIP:        "192.168.1.1",
		}
		if !success {
			reqs[i].ErrorMessage = "timeout after 30s"
			reqs[i].TotalTime = 30 * time.Second
			reqs[i].TTFT = 0
			reqs[i].TPS = 0
		}
	}
	return reqs
}

func generateRunRequests() map[int][]types.RequestMetrics {
	// 为 history panel 中每条 run 生成一组请求数据
	return map[int][]types.RequestMetrics{
		0: generateRequestMetrics(50),   // run-a1b2c3d4
		1: generateRequestMetrics(80),   // run-e5f6g7h8
		2: generateRequestMetrics(30),   // run-z9y8x7w6 (running)
		3: generateRequestMetrics(60),   // run-v5u4t3s2
	}
}
