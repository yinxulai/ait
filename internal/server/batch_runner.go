package server

import (
	"context"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/modes/standard"
	"github.com/yinxulai/ait/internal/server/types"
)

// batchRunner 统一执行批量请求，替代 queuedCaseRunner（1请求）和 queuedLevelRunner（N请求）。
// 通过 constructor 选项化：.WithLevel(n) 或 .WithCaseID(id, idx) 决定行为。
type batchRunner struct {
	ctx        context.Context
	input      types.Input
	runID      RunID
	index      int
	caseID     string
	level      int
	client     client.ModelClient
	aggregator *RunAggregator
	results    []*client.ResponseMetrics
	stop       context.CancelFunc
}

func newBatchRunner(parent context.Context, runID RunID, input types.Input, modelClient client.ModelClient, aggregator *RunAggregator) *batchRunner {
	ctx, cancel := context.WithCancel(parent)
	return &batchRunner{ctx: ctx, input: input, runID: runID, client: modelClient, aggregator: aggregator, stop: cancel}
}

func (r *batchRunner) WithLevel(level int) *batchRunner {
	r.level = level
	return r
}

func (r *batchRunner) WithCaseID(caseID string, index int) *batchRunner {
	r.caseID = caseID
	r.index = index
	return r
}

func (r *batchRunner) Run() (*types.ReportData, error) {
	r.results = make([]*client.ResponseMetrics, r.input.Count)
	jobs := make([]RequestJob, 0, r.input.Count)
	for i := 0; i < r.input.Count; i++ {
		jobs = append(jobs, RequestJob{RunID: r.runID, Index: r.level*10000 + i, Input: r.input, Level: r.level})
	}
	start := time.Now()
	launched := RunRequestBatch(r.ctx, jobs, r.input.Concurrency, NewRequestExecutor(r.client), RequestQueueHooks{
		OnQueued:  r.aggregator.MarkQueued,
		OnStarted: r.aggregator.MarkStarted,
		OnSkipped: r.aggregator.MarkSkipped,
		OnDone: func(result RequestResult) {
			localIdx := result.Job.Index - r.level*10000
			if result.Metrics != nil && localIdx >= 0 && localIdx < len(r.results) {
				r.results[localIdx] = result.Metrics
			}
			rm := r.aggregator.Complete(result)
			if rm.Success {
				uploadRequest(r.aggregator.taskDef.ID, result.Metrics, r.input)
			}
		},
	})
	return standard.CalculateResult(r.input, r.results, time.Since(start), launched), nil
}

func (r *batchRunner) RunWithCallback(cb standard.RequestDoneCallback) (*types.ReportData, error) {
	job := RequestJob{RunID: r.runID, Index: r.index, Input: r.input, CaseID: r.caseID}
	var metrics *client.ResponseMetrics
	var resultErr error
	start := time.Now()
	launched := RunRequestBatch(r.ctx, []RequestJob{job}, 1, NewRequestExecutor(r.client), RequestQueueHooks{
		OnQueued:  r.aggregator.MarkQueued,
		OnStarted: r.aggregator.MarkStarted,
		OnSkipped: r.aggregator.MarkSkipped,
		OnDone: func(result RequestResult) {
			metrics = result.Metrics
			resultErr = result.Err
			if cb != nil {
				cb(result.Metrics, result.Job.Index, result.Err)
			}
		},
	})
	return standard.CalculateResult(r.input, []*client.ResponseMetrics{metrics}, time.Since(start), launched), resultErr
}

func (r *batchRunner) Stop() {
	if r.stop != nil {
		r.stop()
	}
}
