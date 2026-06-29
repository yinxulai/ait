import type { RunState, RunSummary, Task } from '@/api'
import { taskModel, taskProtocol } from '@/lib/task-utils'

export function mergeStartedRun(runs: RunSummary[], state: RunState | undefined, task: Task) {
  if (!state || runs.some((run) => run.run_id === state.run_id)) return runs
  return [runSummaryFromState(state, task), ...runs]
}

export function upsertRunSummary(runs: RunSummary[], summary: RunSummary) {
  const next = runs.filter((run) => run.run_id !== summary.run_id)
  return [summary, ...next]
}

export function isTerminalRunStatus(status: RunSummary['status']) {
  return status === 'completed' || status === 'failed' || status === 'stopped'
}

export function runSummaryFromState(state: RunState, task: Task): RunSummary {
  return {
    run_id: state.run_id,
    task_id: state.task_id,
    mode: state.mode,
    status: state.status,
    protocol: taskProtocol(task),
    model: taskModel(task),
    started_at: state.started_at,
    finished_at: state.finished_at ?? '',
    success_rate: state.success_rate,
    avg_ttft: state.avg_ttft,
    avg_tps: state.avg_tps,
    cache_hit_rate: state.cache_hit_rate,
    rpm: state.rpm,
    tpm: state.tpm,
    error_summary: state.error_msg,
  }
}
