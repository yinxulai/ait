package server

import (
	"context"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/modes/standard"
	"github.com/yinxulai/ait/internal/server/types"
)

type queuedLevelRunner struct {
	ctx        context.Context
	input      types.Input
	runID      RunID
	level      int
	client     client.ModelClient
	aggregator *RunAggregator
	results    []*client.ResponseMetrics
	stop       context.CancelFunc
}

func newQueuedLevelRunner(parent context.Context, runID RunID, input types.Input, modelClient client.ModelClient, aggregator *RunAggregator, level int) *queuedLevelRunner {
	ctx, cancel := context.WithCancel(parent)
	return &queuedLevelRunner{
		ctx:        ctx,
		input:      input,
		runID:      runID,
		level:      level,
		client:     modelClient,
		aggregator: aggregator,
		results:    make([]*client.ResponseMetrics, input.Count),
		stop:       cancel,
	}
}

func (r *queuedLevelRunner) Run() (*types.ReportData, error) {
	jobs := make([]RequestJob, 0, r.input.Count)
	for i := 0; i < r.input.Count; i++ {
		jobs = append(jobs, RequestJob{RunID: r.runID, Index: i, Input: r.input, Level: r.level})
	}
	start := time.Now()
	launched := RunRequestBatch(r.ctx, jobs, r.input.Concurrency, NewRequestExecutor(r.client), RequestQueueHooks{
		OnQueued:  r.aggregator.MarkQueued,
		OnStarted: r.aggregator.MarkStarted,
		OnSkipped: r.aggregator.MarkSkipped,
		OnDone: func(result RequestResult) {
			if result.Metrics != nil && result.Job.Index >= 0 && result.Job.Index < len(r.results) {
				r.results[result.Job.Index] = result.Metrics
			}
			rm := r.aggregator.Complete(result)
			if rm.Success {
				uploadRequest(r.aggregator.taskDef.ID, result.Metrics, r.input)
			}
		},
	})
	return standard.CalculateResult(r.input, r.results, time.Since(start), launched), nil
}

func (r *queuedLevelRunner) Stop() {
	if r.stop != nil {
		r.stop()
	}
}
