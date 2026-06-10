package server

import (
	"context"
	"time"

	"github.com/yinxulai/ait/internal/server/client"
	"github.com/yinxulai/ait/internal/server/modes/standard"
	"github.com/yinxulai/ait/internal/server/types"
)

type queuedCaseRunner struct {
	ctx        context.Context
	input      types.Input
	runID      RunID
	index      int
	caseID     string
	client     client.ModelClient
	aggregator *RunAggregator
	stop       context.CancelFunc
}

func newQueuedCaseRunner(parent context.Context, runID RunID, input types.Input, modelClient client.ModelClient, aggregator *RunAggregator, index int, caseID string) *queuedCaseRunner {
	ctx, cancel := context.WithCancel(parent)
	return &queuedCaseRunner{ctx: ctx, input: input, runID: runID, index: index, caseID: caseID, client: modelClient, aggregator: aggregator, stop: cancel}
}

func (r *queuedCaseRunner) RunWithCallback(cb standard.RequestDoneCallback) (*types.ReportData, error) {
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

func (r *queuedCaseRunner) Stop() {
	if r.stop != nil {
		r.stop()
	}
}
