package tui

import (
	"time"

	"github.com/yinxulai/ait/internal/server"
	"github.com/yinxulai/ait/internal/server/types"
)

// taskHistories stores run histories per task ID (populated by mockData).
var taskHistories map[string][]types.TaskRunSummary

// mockData populates the App with realistic mock data for layout development.
func mockData(a *App) {
	now := time.Now()

	// ── Task 1: GPT-4 Standard Load Test ──────────────────────────────────
	t1 := types.TaskOverview{
		TaskDefinition: types.TaskDefinition{
			ID:        "task-001",
			Name:      "GPT-4 标准压测",
			CreatedAt: now.Add(-72 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
			Input: types.Input{
				Mode:        "standard",
				Protocol:    "openai-completions",
				EndpointURL: "https://api.openai.com/v1/chat/completions",
				Model:       "gpt-4",
				Concurrency: 20,
				Count:       500,
				Timeout:     30 * time.Second,
				Stream:      false,
			},
		},
	}
	t1h := []types.TaskRunSummary{
		{RunID: "run-001-1", TaskID: "task-001", Mode: "standard", Status: "completed", Protocol: "openai-completions", Model: "gpt-4", StartedAt: now.Add(-2 * time.Hour), FinishedAt: now.Add(-118*time.Minute - 30*time.Second), SuccessRate: 0.982, AvgTTFT: 320 * time.Millisecond, AvgTPS: 142, RPM: 8520, TPM: 1200000},
		{RunID: "run-001-2", TaskID: "task-001", Mode: "standard", Status: "completed", Protocol: "openai-completions", Model: "gpt-4", StartedAt: now.Add(-26 * time.Hour), FinishedAt: now.Add(-25*time.Hour - 42*time.Minute), SuccessRate: 0.951, AvgTTFT: 410 * time.Millisecond, AvgTPS: 118, RPM: 7080, TPM: 980000},
		{RunID: "run-001-3", TaskID: "task-001", Mode: "standard", Status: "failed", Protocol: "openai-completions", Model: "gpt-4", StartedAt: now.Add(-50 * time.Hour), FinishedAt: now.Add(-49*time.Hour - 10*time.Minute), SuccessRate: 0.23, AvgTTFT: 0, AvgTPS: 0, ErrorSummary: "429 rate limit exceeded after 230 requests"},
	}
	t1.LatestRun = &t1h[0]

	// ── Task 2: Claude Turbo Ramp Test ────────────────────────────────────
	t2 := types.TaskOverview{
		TaskDefinition: types.TaskDefinition{
			ID:        "task-002",
			Name:      "Claude Turbo 爬坡",
			CreatedAt: now.Add(-48 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
			Input: types.Input{
				Mode:        "turbo",
				Protocol:    "anthropic-messages",
				EndpointURL: "https://api.anthropic.com/v1/messages",
				Model:       "claude-3-opus",
				Concurrency: 10,
				Count:       2000,
				Timeout:     60 * time.Second,
				Stream:      true,
				Turbo:       true,
				TurboConfig: types.TurboConfig{
					InitConcurrency: 5,
					MaxConcurrency:  100,
					StepSize:        5,
					LevelRequests:   200,
					MinSuccessRate:  0.95,
				},
			},
		},
	}
	t2h := []types.TaskRunSummary{
		{RunID: "run-002-1", TaskID: "task-002", Mode: "turbo", Status: "completed", Protocol: "anthropic-messages", Model: "claude-3-opus", StartedAt: now.Add(-1 * time.Hour), FinishedAt: now.Add(-45 * time.Minute), SuccessRate: 0.967, AvgTTFT: 180 * time.Millisecond, AvgTPS: 210, RPM: 12600, TPM: 2800000, MaxStableConcurrency: 85},
		{RunID: "run-002-2", TaskID: "task-002", Mode: "turbo", Status: "stopped", Protocol: "anthropic-messages", Model: "claude-3-opus", StartedAt: now.Add(-24 * time.Hour), FinishedAt: now.Add(-23*time.Hour - 20*time.Minute), SuccessRate: 0.89, AvgTTFT: 250 * time.Millisecond, AvgTPS: 165, MaxStableConcurrency: 60},
	}
	t2.LatestRun = &t2h[0]

	// ── Task 3: Integrity Suite ───────────────────────────────────────────
	t3 := types.TaskOverview{
		TaskDefinition: types.TaskDefinition{
			ID:        "task-003",
			Name:      "API 一致性校验",
			CreatedAt: now.Add(-24 * time.Hour),
			UpdatedAt: now.Add(-30 * time.Minute),
			Input: types.Input{
				Mode:        "integrity",
				Protocol:    "openai-completions",
				EndpointURL: "https://api.openai.com/v1/chat/completions",
				Model:       "gpt-4o",
				Concurrency: 5,
				Count:       100,
				Timeout:     20 * time.Second,
				Stream:      false,
				Integrity:   types.IntegrityConfig{Enabled: true, Suite: "basic"},
			},
		},
	}
	t3h := []types.TaskRunSummary{
		{RunID: "run-003-1", TaskID: "task-003", Mode: "integrity", Status: "completed", Protocol: "openai-completions", Model: "gpt-4o", StartedAt: now.Add(-30 * time.Minute), FinishedAt: now.Add(-28 * time.Minute), SuccessRate: 1.0, AvgTTFT: 150 * time.Millisecond, AvgTPS: 45, CacheHitRate: 0.32, RPM: 2700},
	}
	t3.LatestRun = &t3h[0]

	// ── Task 4: more tasks ────────────────────────────────────────────────
	t4 := types.TaskOverview{
		TaskDefinition: types.TaskDefinition{
			ID:        "task-004",
			Name:      "GPT-4o Mini 流式测试",
			CreatedAt: now.Add(-10 * time.Hour),
			UpdatedAt: now.Add(-10 * time.Hour),
			Input: types.Input{
				Mode:        "standard",
				Protocol:    "openai-completions",
				EndpointURL: "https://api.openai.com/v1/chat/completions",
				Model:       "gpt-4o-mini",
				Concurrency: 50,
				Count:       1000,
				Timeout:     15 * time.Second,
				Stream:      true,
			},
		},
	}
	t4h := []types.TaskRunSummary{
		{RunID: "run-004-1", TaskID: "task-004", Mode: "standard", Status: "completed", Protocol: "openai-completions", Model: "gpt-4o-mini", StartedAt: now.Add(-8 * time.Hour), FinishedAt: now.Add(-7*time.Hour - 55*time.Minute), SuccessRate: 0.998, AvgTTFT: 85 * time.Millisecond, AvgTPS: 520, RPM: 31200, TPM: 4500000},
	}
	t4.LatestRun = &t4h[0]

	t5 := types.TaskOverview{
		TaskDefinition: types.TaskDefinition{
			ID:        "task-005",
			Name:      "Dev 本地调试",
			CreatedAt: now.Add(-1 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
			Input: types.Input{
				Mode:        "standard",
				Protocol:    "openai-completions",
				EndpointURL: "http://localhost:8080/v1/chat/completions",
				Model:       "llama-3-70b",
				Concurrency: 2,
				Count:       10,
				Timeout:     60 * time.Second,
				Stream:      true,
			},
		},
	}

	// ── Assemble ──────────────────────────────────────────────────────────
	a.tasks = []types.TaskOverview{t1, t2, t3, t4, t5}
	a.taskMap = map[string]*types.TaskOverview{
		"task-001": &a.tasks[0],
		"task-002": &a.tasks[1],
		"task-003": &a.tasks[2],
		"task-004": &a.tasks[3],
		"task-005": &a.tasks[4],
	}

	// Store histories for later lookup
	taskHistories = map[string][]types.TaskRunSummary{
		"task-001": t1h,
		"task-002": t2h,
		"task-003": t3h,
		"task-004": t4h,
	}

	a.selectedTask = &t1.TaskDefinition
	a.selectedHistory = t1h
	a.navLevel = 0

	// Active run simulation — task-002 is "running"
	rs := &server.RunState{
		Mode:       server.ModeTurbo,
		RunID:      "run-002-3",
		TaskID:     "task-002",
		Status:     server.RunStatusRunning,
		StartedAt:  now.Add(-3 * time.Minute),
		TotalReqs:  2000,
		QueuedReqs: 1200,
		DoneReqs:   600,
		SuccessReqs: 585,
		FailedReqs: 15,
		SkippedReqs: 0,
		RunningReqs: 200,
		AvgTPS:     210,
		AvgTTFT:    185 * time.Millisecond,
		SuccessRate: 0.975,
		CacheHitRate: 0.15,
		RPM:        12600,
		TPM:        2600000,
		ModeState: map[string]any{
			server.ModeStateKeyCurrentLevel: 5,
			server.ModeStateKeyLevels:       12,
			server.ModeStateKeyConfig: map[string]any{
				"initConcurrency": 5,
				"maxConcurrency":  100,
				"stepSize":        5,
			},
		},
	}
	a.activeRuns = map[string]*server.RunState{"task-002": rs}

	// Mock selected task detail (for task-001)
	a.selectedTask = &t1.TaskDefinition
	a.selectedHistory = t1h
	a.dashRunID = ""
	a.dashRunState = nil
	a.requestCursor = 0
	a.historyCursor = 0
}

// mockRequests returns realistic request metrics for demo.
func mockRequests(count int) []*types.RequestMetrics {
	reqs := make([]*types.RequestMetrics, count)
	for i := 0; i < count; i++ {
		success := i < count-2 // last 2 are failures
		ttft := time.Duration(80+int(float64(i%50)*2.5)) * time.Millisecond
		total := ttft + time.Duration(50+int(float64(i%30)*10))*time.Millisecond
		req := &types.RequestMetrics{
			Index:            i + 1,
			Success:          success,
			TotalTime:        total,
			TTFT:             ttft,
			TPS:              float64(500+int(float64(i%200)*3.5)),
			PromptTokens:     1200 + i%800,
			CompletionTokens: 300 + i%200,
			CachedTokens:     int(float64(1200+i%800) * 0.15),
			CacheHitRate:     0.15,
			DNSTime:          2 * time.Millisecond,
			ConnectTime:      5 * time.Millisecond,
			TLSTime:          12 * time.Millisecond,
			TargetIP:         "13.32.99.86",
		}
		if !success {
			req.ErrorMessage = "request timeout after 30s"
			req.TotalTime = 30 * time.Second
			req.TTFT = 0
			req.TPS = 0
			req.CompletionTokens = 0
		}
		reqs[i] = req
	}
	return reqs
}

// mockRunDetailRequests is a global store of mock requests per run for consistency.
var mockRunDetailRequests = map[string][]*types.RequestMetrics{
	"run-001-1": mockRequests(150),
	"run-001-2": mockRequests(120),
	"run-001-3": mockRequests(80),
	"run-002-1": mockRequests(200),
	"run-002-2": mockRequests(100),
	"run-003-1": mockRequests(60),
	"run-004-1": mockRequests(250),
}
