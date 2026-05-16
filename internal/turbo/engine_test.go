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
