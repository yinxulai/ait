package turbo

import (
	"fmt"
	"sync"
	"time"

	runnerpkg "github.com/yinxulai/ait/internal/runner"
	"github.com/yinxulai/ait/internal/types"
)

const (
	StopReasonLowSuccessRate = "low_success_rate"
	StopReasonHighLatency    = "high_latency"
	StopReasonMaxConcurrency = "max_concurrency"
	StopReasonManual         = "manual"
)

type LevelRunner interface {
	Run() (*types.ReportData, error)
	Stop()
}

type RunnerFactory func(input types.Input) (LevelRunner, error)

type Engine struct {
	runnerFactory RunnerFactory
	now           func() time.Time
	mu            sync.Mutex
	currentRunner LevelRunner
	stopCh        chan struct{}
	stopOnce      sync.Once
}

func New(factory RunnerFactory) *Engine {
	return &Engine{
		runnerFactory: factory,
		now:           time.Now,
		stopCh:        make(chan struct{}),
	}
}

func DefaultRunnerFactory(taskID string) RunnerFactory {
	return func(input types.Input) (LevelRunner, error) {
		return runnerpkg.NewRunner(taskID, input)
	}
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopCh)
	})
	e.mu.Lock()
	runner := e.currentRunner
	e.mu.Unlock()
	if runner != nil {
		runner.Stop()
	}
}

func (e *Engine) Run(input types.Input) (*types.TurboResult, error) {
	if e.runnerFactory == nil {
		return nil, fmt.Errorf("turbo runnerFactory is required")
	}

	cfg := normalizeConfig(input.TurboConfig, input.Count)
	startedAt := e.now()
	result := &types.TurboResult{
		Config:      cfg,
		Levels:      []types.TurboLevelResult{},
		Model:       input.Model,
		Protocol:    input.NormalizedProtocol(),
		EndpointURL: input.ResolvedEndpointURL(),
		Timestamp:   startedAt.Format(time.RFC3339),
	}

	for concurrency := cfg.InitConcurrency; concurrency <= cfg.MaxConcurrency; concurrency += cfg.StepSize {
		select {
		case <-e.stopCh:
			result.StopReason = StopReasonManual
			result.ProbeDuration = time.Since(startedAt)
			return result, nil
		default:
		}

		levelInput := input
		levelInput.Turbo = false
		levelInput.Concurrency = concurrency
		levelInput.Count = cfg.LevelRequests

		levelRunner, err := e.runnerFactory(levelInput)
		if err != nil {
			return nil, err
		}

		e.mu.Lock()
		e.currentRunner = levelRunner
		e.mu.Unlock()

		report, err := levelRunner.Run()

		e.mu.Lock()
		e.currentRunner = nil
		e.mu.Unlock()

		if err != nil {
			return nil, err
		}

		level := buildLevelResult(report, concurrency)
		result.Levels = append(result.Levels, level)

		select {
		case <-e.stopCh:
			result.StopReason = StopReasonManual
			result.ProbeDuration = time.Since(startedAt)
			return result, nil
		default:
		}

		if level.SuccessRate < cfg.MinSuccessRate {
			result.Levels[len(result.Levels)-1].Stable = false
			result.Levels[len(result.Levels)-1].StopReason = StopReasonLowSuccessRate
			result.StopReason = StopReasonLowSuccessRate
			break
		}
		if level.AvgTotalTime > cfg.MaxLatency {
			result.Levels[len(result.Levels)-1].Stable = false
			result.Levels[len(result.Levels)-1].StopReason = StopReasonHighLatency
			result.StopReason = StopReasonHighLatency
			break
		}

		result.MaxStableConcurrency = concurrency
		if level.AvgTPS > result.PeakTPS {
			result.PeakTPS = level.AvgTPS
		}

		if concurrency+cfg.StepSize > cfg.MaxConcurrency {
			result.StopReason = StopReasonMaxConcurrency
		}
	}

	if result.StopReason == "" {
		result.StopReason = StopReasonMaxConcurrency
	}
	result.ProbeDuration = time.Since(startedAt)
	return result, nil
}

func buildLevelResult(report *types.ReportData, concurrency int) types.TurboLevelResult {
	successCount := int(float64(report.TotalRequests) * report.SuccessRate / 100)
	return types.TurboLevelResult{
		Concurrency:   concurrency,
		TotalRequests: report.TotalRequests,
		SuccessCount:  successCount,
		SuccessRate:   report.SuccessRate / 100,
		AvgTPS:        report.AvgTPS,
		PeakTPS:       report.MaxTPS,
		AvgTTFT:       report.AvgTTFT,
		CacheHitRate:  report.AvgCacheHitRate,
		AvgTotalTime:  report.AvgTotalTime,
		StdDevTPS:     report.StdDevTPS,
		Stable:        true,
	}
}

func normalizeConfig(cfg types.TurboConfig, fallbackLevelRequests int) types.TurboConfig {
	if cfg.InitConcurrency <= 0 {
		cfg.InitConcurrency = 1
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 50
	}
	if cfg.MaxConcurrency < cfg.InitConcurrency {
		cfg.MaxConcurrency = cfg.InitConcurrency
	}
	if cfg.StepSize <= 0 {
		cfg.StepSize = 2
	}
	if cfg.LevelRequests <= 0 {
		if fallbackLevelRequests > 0 {
			cfg.LevelRequests = fallbackLevelRequests
		} else {
			cfg.LevelRequests = 30
		}
	}
	if cfg.MinSuccessRate <= 0 {
		cfg.MinSuccessRate = 0.9
	}
	if cfg.MaxLatency <= 0 {
		cfg.MaxLatency = 10 * time.Second
	}
	return cfg
}
