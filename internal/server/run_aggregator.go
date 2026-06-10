package server

import (
	"time"

	"github.com/yinxulai/ait/internal/server/store"
	"github.com/yinxulai/ait/internal/server/types"
)

// RunAggregator 是运行状态、请求持久化和请求事件发布的统一入口。
type RunAggregator struct {
	server   *serverImpl
	active   *activeRun
	runID    RunID
	taskDef  types.TaskDefinition
	runStore *store.RunStore
}

func newRunAggregator(s *serverImpl, ar *activeRun, runID RunID, taskDef types.TaskDefinition, runStore *store.RunStore) *RunAggregator {
	return &RunAggregator{server: s, active: ar, runID: runID, taskDef: taskDef, runStore: runStore}
}

func (a *RunAggregator) MarkQueued(job RequestJob) {
	a.active.mu.Lock()
	if a.active.state.RequestStates == nil {
		a.active.state.RequestStates = make(map[int]RequestState)
	}
	a.active.state.RequestStates[job.Index] = RequestState{Index: job.Index, Status: RequestStatusQueued, Level: job.Level, CaseID: job.CaseID, QueuedAt: time.Now()}
	a.recountRequestStatesLocked()
	snap := a.active.snapshotState()
	a.active.mu.Unlock()
	a.server.bus.publishRunEvent(Event{RunID: a.runID, Kind: EventRequestQueued, Payload: snap})
}

func (a *RunAggregator) MarkStarted(job RequestJob) {
	now := time.Now()
	a.active.mu.Lock()
	state := a.active.state.RequestStates[job.Index]
	state.Index = job.Index
	state.Status = RequestStatusRunning
	state.Level = job.Level
	state.CaseID = job.CaseID
	state.StartedAt = &now
	if state.QueuedAt.IsZero() {
		state.QueuedAt = now
	}
	a.active.state.RequestStates[job.Index] = state
	a.recountRequestStatesLocked()
	snap := a.active.snapshotState()
	a.active.mu.Unlock()
	a.server.bus.publishRunEvent(Event{RunID: a.runID, Kind: EventRequestStarted, Payload: snap})
}

func (a *RunAggregator) MarkSkipped(job RequestJob) {
	now := time.Now()
	a.active.mu.Lock()
	if a.active.state.RequestStates == nil {
		a.active.state.RequestStates = make(map[int]RequestState)
	}
	state := a.active.state.RequestStates[job.Index]
	state.Index = job.Index
	state.Status = RequestStatusSkipped
	state.Level = job.Level
	state.CaseID = job.CaseID
	state.FinishedAt = &now
	if state.QueuedAt.IsZero() {
		state.QueuedAt = now
	}
	a.active.state.RequestStates[job.Index] = state
	a.recountRequestStatesLocked()
	snap := a.active.snapshotState()
	a.active.mu.Unlock()
	a.server.bus.publishRunEvent(Event{RunID: a.runID, Kind: EventRequestSkipped, Payload: snap})
}

func (a *RunAggregator) Complete(result RequestResult) *types.RequestMetrics {
	rm := mapRequestMetrics(result.Metrics, result.Job.Index, result.Err)
	rm.Level = result.Job.Level
	_ = a.runStore.AppendRequest(a.taskDef.ID, string(a.runID), *rm)

	now := time.Now()
	a.active.mu.Lock()
	if a.active.state.RequestStates == nil {
		a.active.state.RequestStates = make(map[int]RequestState)
	}
	requestState := a.active.state.RequestStates[result.Job.Index]
	requestState.Index = result.Job.Index
	requestState.Level = result.Job.Level
	requestState.CaseID = result.Job.CaseID
	requestState.FinishedAt = &now
	if requestState.QueuedAt.IsZero() {
		requestState.QueuedAt = now
	}
	if rm.Success {
		requestState.Status = RequestStatusSucceeded
	} else {
		requestState.Status = RequestStatusFailed
		requestState.ErrorMsg = rm.ErrorMessage
	}
	a.active.state.RequestStates[result.Job.Index] = requestState
	a.active.state.Requests = append(a.active.state.Requests, rm)
	a.active.state.DoneReqs++
	if rm.Success {
		a.active.state.SuccessReqs++
		a.active.tpsSum += rm.TPS
		a.active.ttftSum += rm.TTFT
		a.active.cacheSum += rm.CacheHitRate
		a.active.tokenSum += int64(rm.CompletionTokens)
	} else {
		a.active.state.FailedReqs++
	}
	if a.active.state.SuccessReqs > 0 {
		a.active.state.AvgTPS = a.active.tpsSum / float64(a.active.state.SuccessReqs)
		a.active.state.AvgTTFT = a.active.ttftSum / time.Duration(a.active.state.SuccessReqs)
		a.active.state.CacheHitRate = a.active.cacheSum / float64(a.active.state.SuccessReqs)
	}
	if a.active.state.DoneReqs > 0 {
		a.active.state.SuccessRate = float64(a.active.state.SuccessReqs) / float64(a.active.state.DoneReqs) * 100
	}
	if elapsed := time.Since(a.active.state.StartedAt).Minutes(); elapsed > 0 {
		a.active.state.RPM = float64(a.active.state.DoneReqs) / elapsed
		a.active.state.TPM = float64(a.active.tokenSum) / elapsed
	}
	a.recountRequestStatesLocked()
	snap := a.active.snapshotState()
	a.active.mu.Unlock()
	a.server.bus.publishRunEvent(Event{RunID: a.runID, Kind: EventRequestDone, Payload: snap})
	return rm
}

func (a *RunAggregator) recountRequestStatesLocked() {
	queued := 0
	running := 0
	skipped := 0
	for _, state := range a.active.state.RequestStates {
		switch state.Status {
		case RequestStatusQueued:
			queued++
		case RequestStatusRunning:
			running++
		case RequestStatusSkipped:
			skipped++
		}
	}
	a.active.state.QueuedReqs = queued
	a.active.state.RunningReqs = running
	a.active.state.SkippedReqs = skipped
}
