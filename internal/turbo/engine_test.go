package turbo

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/yinxulai/ait/internal/types"
)

type fakeRunner struct {
	report     *types.ReportData
	err        error
	stopCalled *atomic.Bool
	blockUntil <-chan struct{}
}

func (f *fakeRunner) Run() (*types.ReportData, error) {
	if f.blockUntil != nil {
		<-f.blockUntil
	}
	return f.report, f.err
}

func (f *fakeRunner) Stop() {
	if f.stopCalled != nil {
		f.stopCalled.Store(true)
	}
}

func TestEngineRunStopsOnLowSuccessRate(t *testing.T) {
	levels := map[int]*types.ReportData{
		1: {TotalRequests: 10, SuccessRate: 100, AvgTPS: 10, MaxTPS: 12, AvgTTFT: 50 * time.Millisecond, AvgTotalTime: 200 * time.Millisecond},
		2: {TotalRequests: 10, SuccessRate: 95, AvgTPS: 20, MaxTPS: 22, AvgTTFT: 60 * time.Millisecond, AvgTotalTime: 300 * time.Millisecond},
		3: {TotalRequests: 10, SuccessRate: 80, AvgTPS: 18, MaxTPS: 20, AvgTTFT: 80 * time.Millisecond, AvgTotalTime: 400 * time.Millisecond},
	}
	engine := New(func(input types.Input) (LevelRunner, error) {
		return &fakeRunner{report: levels[input.Concurrency]}, nil
	})

	result, err := engine.Run(types.Input{
		Protocol:    types.ProtocolOpenAIResponses,
		EndpointURL: "https://api.openai.com/v1/responses",
		Model:       "gpt-4.1",
		Count:       30,
		TurboConfig: types.TurboConfig{InitConcurrency: 1, MaxConcurrency: 5, StepSize: 1, LevelRequests: 10, MinSuccessRate: 0.9, MaxLatency: 10 * time.Second},
	})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if len(result.Levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(result.Levels))
	}
	if result.MaxStableConcurrency != 2 {
		t.Fatalf("expected MaxStableConcurrency 2, got %d", result.MaxStableConcurrency)
	}
	if result.StopReason != StopReasonLowSuccessRate {
		t.Fatalf("expected stop reason %s, got %s", StopReasonLowSuccessRate, result.StopReason)
	}
	if result.PeakTPS != 20 {
		t.Fatalf("expected PeakTPS 20, got %f", result.PeakTPS)
	}
	if result.Levels[2].Stable {
		t.Fatal("expected last level to be marked unstable")
	}
	if result.Levels[2].StopReason != StopReasonLowSuccessRate {
		t.Fatalf("expected level stop reason %s, got %s", StopReasonLowSuccessRate, result.Levels[2].StopReason)
	}
}

func TestEngineRunStopsOnHighLatency(t *testing.T) {
	engine := New(func(input types.Input) (LevelRunner, error) {
		report := &types.ReportData{TotalRequests: 10, SuccessRate: 100, AvgTPS: 10, MaxTPS: 15, AvgTTFT: 80 * time.Millisecond, AvgTotalTime: time.Duration(input.Concurrency) * time.Second}
		return &fakeRunner{report: report}, nil
	})

	result, err := engine.Run(types.Input{
		Protocol:    types.ProtocolAnthropicMessages,
		EndpointURL: "https://api.anthropic.com/v1/messages",
		Model:       "claude-3-7-sonnet",
		Count:       20,
		TurboConfig: types.TurboConfig{InitConcurrency: 1, MaxConcurrency: 5, StepSize: 1, LevelRequests: 10, MinSuccessRate: 0.9, MaxLatency: 2 * time.Second},
	})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if result.StopReason != StopReasonHighLatency {
		t.Fatalf("expected stop reason %s, got %s", StopReasonHighLatency, result.StopReason)
	}
	if result.MaxStableConcurrency != 2 {
		t.Fatalf("expected MaxStableConcurrency 2, got %d", result.MaxStableConcurrency)
	}
}

func TestEngineStopPropagatesToActiveRunner(t *testing.T) {
	stopCalled := &atomic.Bool{}
	blocker := make(chan struct{})
	engine := New(func(input types.Input) (LevelRunner, error) {
		return &fakeRunner{
			report:     &types.ReportData{TotalRequests: 1, SuccessRate: 100, AvgTPS: 1, MaxTPS: 1, AvgTotalTime: 10 * time.Millisecond},
			stopCalled: stopCalled,
			blockUntil: blocker,
		}, nil
	})

	resultCh := make(chan *types.TurboResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, err := engine.Run(types.Input{
			Protocol:    types.ProtocolOpenAICompletions,
			EndpointURL: "https://api.openai.com/v1/chat/completions",
			Model:       "gpt-4.1-mini",
			Count:       10,
			TurboConfig: types.TurboConfig{InitConcurrency: 1, MaxConcurrency: 3, StepSize: 1, LevelRequests: 1, MinSuccessRate: 0.9, MaxLatency: time.Second},
		})
		resultCh <- result
		errCh <- err
	}()

	time.Sleep(30 * time.Millisecond)
	engine.Stop()
	close(blocker)

	result := <-resultCh
	if err := <-errCh; err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if !stopCalled.Load() {
		t.Fatal("expected active runner Stop() to be called")
	}
	if result.StopReason != StopReasonManual {
		t.Fatalf("expected stop reason %s, got %s", StopReasonManual, result.StopReason)
	}
}

func TestNormalizeConfigUsesDefaults(t *testing.T) {
	cfg := normalizeConfig(types.TurboConfig{}, 12)
	if cfg.InitConcurrency != 1 || cfg.MaxConcurrency != 50 || cfg.StepSize != 2 {
		t.Fatalf("unexpected concurrency defaults: %+v", cfg)
	}
	if cfg.LevelRequests != 12 {
		t.Fatalf("expected fallback LevelRequests 12, got %d", cfg.LevelRequests)
	}
	if cfg.MinSuccessRate != 0.9 || cfg.MaxLatency != 10*time.Second {
		t.Fatalf("unexpected threshold defaults: %+v", cfg)
	}
}

func TestNormalizeConfigExported(t *testing.T) {
	cfg := NormalizeConfig(types.TurboConfig{}, 20)
	if cfg.LevelRequests != 20 {
		t.Fatalf("expected LevelRequests 20, got %d", cfg.LevelRequests)
	}
	if cfg.InitConcurrency != 1 || cfg.MaxConcurrency != 50 {
		t.Fatalf("unexpected defaults from exported NormalizeConfig: %+v", cfg)
	}
}

func TestEngineOnLevelDone_CalledPerLevel(t *testing.T) {
	var doneLevels []types.TurboLevelResult
	engine := New(func(input types.Input) (LevelRunner, error) {
		report := &types.ReportData{
			TotalRequests: 10, SuccessRate: 100,
			AvgTPS: float64(input.Concurrency) * 5, MaxTPS: float64(input.Concurrency) * 6,
			AvgTTFT: 50 * time.Millisecond, AvgTotalTime: 200 * time.Millisecond,
		}
		return &fakeRunner{report: report}, nil
	})
	engine.SetOnLevelDone(func(level types.TurboLevelResult) {
		doneLevels = append(doneLevels, level)
	})

	result, err := engine.Run(types.Input{
		Protocol:    types.ProtocolOpenAIResponses,
		EndpointURL: "https://api.example.com",
		Model:       "test-model",
		TurboConfig: types.TurboConfig{InitConcurrency: 1, MaxConcurrency: 3, StepSize: 1, LevelRequests: 10, MinSuccessRate: 0.9, MaxLatency: 10 * time.Second},
	})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if len(doneLevels) != 3 {
		t.Fatalf("expected OnLevelDone called 3 times, got %d", len(doneLevels))
	}
	if len(doneLevels) != len(result.Levels) {
		t.Fatalf("OnLevelDone calls (%d) != result.Levels (%d)", len(doneLevels), len(result.Levels))
	}
	for i, level := range doneLevels {
		if level.Concurrency != result.Levels[i].Concurrency {
			t.Fatalf("level[%d] concurrency mismatch: callback=%d result=%d", i, level.Concurrency, result.Levels[i].Concurrency)
		}
	}
}

func TestEngineOnLevelDone_UnstableLevelIncluded(t *testing.T) {
	var doneLevels []types.TurboLevelResult
	reports := map[int]*types.ReportData{
		1: {TotalRequests: 10, SuccessRate: 100, AvgTPS: 10, MaxTPS: 12, AvgTTFT: 50 * time.Millisecond, AvgTotalTime: 200 * time.Millisecond},
		2: {TotalRequests: 10, SuccessRate: 70, AvgTPS: 8, MaxTPS: 10, AvgTTFT: 80 * time.Millisecond, AvgTotalTime: 300 * time.Millisecond},
	}
	engine := New(func(input types.Input) (LevelRunner, error) {
		return &fakeRunner{report: reports[input.Concurrency]}, nil
	})
	engine.SetOnLevelDone(func(level types.TurboLevelResult) {
		doneLevels = append(doneLevels, level)
	})

	result, err := engine.Run(types.Input{
		TurboConfig: types.TurboConfig{InitConcurrency: 1, MaxConcurrency: 3, StepSize: 1, LevelRequests: 10, MinSuccessRate: 0.9, MaxLatency: 10 * time.Second},
	})
	if err != nil {
		t.Fatalf("Run() returned unexpected error: %v", err)
	}
	if len(doneLevels) != 2 {
		t.Fatalf("expected 2 OnLevelDone calls (stable+unstable), got %d", len(doneLevels))
	}
	// 最后一级应是不稳定的
	lastLevel := doneLevels[len(doneLevels)-1]
	if lastLevel.Stable {
		t.Fatal("expected last level to be unstable in OnLevelDone callback")
	}
	if lastLevel.StopReason != StopReasonLowSuccessRate {
		t.Fatalf("expected stop reason %s, got %s", StopReasonLowSuccessRate, lastLevel.StopReason)
	}
	if result.StopReason != StopReasonLowSuccessRate {
		t.Fatalf("expected result stop reason %s, got %s", StopReasonLowSuccessRate, result.StopReason)
	}
}

func TestEngineRunAllLevelsStable(t *testing.T) {
	engine := New(func(input types.Input) (LevelRunner, error) {
		report := &types.ReportData{
			TotalRequests: 5, SuccessRate: 100,
			AvgTPS: 10, MaxTPS: 12,
			AvgTTFT: 30 * time.Millisecond, AvgTotalTime: 100 * time.Millisecond,
		}
		return &fakeRunner{report: report}, nil
	})

	result, err := engine.Run(types.Input{
		TurboConfig: types.TurboConfig{InitConcurrency: 2, MaxConcurrency: 4, StepSize: 2, LevelRequests: 5, MinSuccessRate: 0.9, MaxLatency: 5 * time.Second},
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if len(result.Levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(result.Levels))
	}
	if result.MaxStableConcurrency != 4 {
		t.Fatalf("expected MaxStableConcurrency 4, got %d", result.MaxStableConcurrency)
	}
	if result.StopReason != StopReasonMaxConcurrency {
		t.Fatalf("expected stop reason %s, got %s", StopReasonMaxConcurrency, result.StopReason)
	}
	for i, level := range result.Levels {
		if !level.Stable {
			t.Fatalf("level[%d] expected stable but got unstable", i)
		}
	}
}

func TestEngineFactoryReceivesCorrectConcurrency(t *testing.T) {
	var concurrencies []int
	engine := New(func(input types.Input) (LevelRunner, error) {
		concurrencies = append(concurrencies, input.Concurrency)
		report := &types.ReportData{TotalRequests: 5, SuccessRate: 100, AvgTPS: 10, MaxTPS: 12, AvgTotalTime: 100 * time.Millisecond}
		return &fakeRunner{report: report}, nil
	})

	_, err := engine.Run(types.Input{
		TurboConfig: types.TurboConfig{InitConcurrency: 2, MaxConcurrency: 6, StepSize: 2, LevelRequests: 5, MinSuccessRate: 0.9, MaxLatency: 5 * time.Second},
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	expected := []int{2, 4, 6}
	if len(concurrencies) != len(expected) {
		t.Fatalf("expected factory called with %v, got %v", expected, concurrencies)
	}
	for i, c := range expected {
		if concurrencies[i] != c {
			t.Fatalf("concurrencies[%d]: expected %d, got %d", i, c, concurrencies[i])
		}
	}
}
