import type { RequestDetail, RunState, RunSummary } from '@/api'

import { IntegrityRunDetail, type IntegritySnapshot } from '@/components/run-detail-integrity'
import { RunDetail as StandardRunDetail, TaskRunHistory as StandardTaskRunHistory } from '@/components/run-detail-standard'
import { TurboRunDetail } from '@/components/run-detail-turbo'

type RunDetailProps = {
  run?: RunSummary
  state?: RunState
  requests: RequestDetail[]
  selectedRequest?: RequestDetail
  onSelectRequest: (id: string) => void
}

type TaskRunHistoryProps = {
  runs: RunSummary[]
  selectedRun?: RunSummary
  onChooseRun: (run: RunSummary) => void
  samplesByRun: Record<string, RequestDetail[]>
}

export function TaskRunHistory(props: TaskRunHistoryProps) {
  return <StandardTaskRunHistory {...props} />
}

export function RunDetail({ run, state, requests, selectedRequest, onSelectRequest }: RunDetailProps) {
  if (!run) {
    return <StandardRunDetail run={undefined} requests={requests} selectedRequest={selectedRequest} onSelectRequest={onSelectRequest} />
  }

  const integrity = integritySnapshot(state)
  if (run.mode === 'integrity' && integrity) {
    return <IntegrityRunDetail snapshot={integrity} />
  }
  if (run.mode === 'turbo') {
    return <TurboRunDetail />
  }
  return <StandardRunDetail run={run} requests={requests} selectedRequest={selectedRequest} onSelectRequest={onSelectRequest} />
}

function integritySnapshot(state?: RunState): IntegritySnapshot | undefined {
  if (!state || state.mode !== 'integrity') return undefined
  const modeState = state.mode_state as import('@/api').IntegrityModeState | undefined
  const result = isIntegrityResult(state.mode_result) ? state.mode_result : undefined
  const suiteStatus = modeState?.suite_status
  const cases = result?.cases ?? modeState?.cases ?? []
  const assertions = result?.assertions ?? modeState?.assertion_results ?? flattenAssertions(cases)

  return {
    suiteId: result?.suite_id ?? suiteIdFromState(modeState),
    status: result?.status ?? suiteStatus?.phase ?? '',
    message: suiteStatus?.message ?? '',
    currentCaseId: modeState?.current_case_id ?? '',
    totalCases: result?.total_cases ?? suiteStatus?.case_count ?? cases.length,
    passedCases: result?.passed_cases ?? cases.filter((item) => item.status === 'passed').length,
    failedCases: result?.failed_cases ?? cases.filter((item) => item.status === 'failed').length,
    warnedCases: result?.warned_cases ?? cases.filter((item) => item.warned_assertions > 0).length,
    skippedCases: result?.skipped_cases ?? cases.filter((item) => item.status === 'skipped').length,
    requiredFailedCases: result?.required_failed_cases ?? cases.filter((item) => item.required && item.status === 'failed').length,
    totalAssertions: sumCases(cases, 'total_assertions', assertions.length),
    passedAssertions: sumCases(cases, 'passed_assertions', assertions.filter((item) => item.passed).length),
    failedAssertions: sumCases(cases, 'failed_assertions', assertions.filter((item) => !item.passed && item.level !== 'warn').length),
    warnedAssertions: sumCases(cases, 'warned_assertions', assertions.filter((item) => item.level === 'warn').length),
    cases,
    assertions,
  }
}

function isIntegrityResult(value: unknown): value is import('@/api').IntegrityResult {
  return Boolean(value && typeof value === 'object' && 'suite_id' in value && 'cases' in value)
}

function suiteIdFromState(modeState?: import('@/api').IntegrityModeState) {
  if (!modeState) return ''
  if (typeof modeState.suite === 'string') return modeState.suite
  return modeState.suite?.id ?? modeState.suite_status?.suite ?? ''
}

function flattenAssertions(cases: import('@/api').IntegrityCaseResult[]) {
  return cases.flatMap((item) => item.assertions ?? [])
}

function sumCases(cases: import('@/api').IntegrityCaseResult[], key: 'total_assertions' | 'passed_assertions' | 'failed_assertions' | 'warned_assertions', fallback: number) {
  if (cases.length === 0) return fallback
  return cases.reduce((sum, item) => sum + (item[key] || 0), 0)
}

export type { IntegritySnapshot, RunDetailProps, TaskRunHistoryProps }
