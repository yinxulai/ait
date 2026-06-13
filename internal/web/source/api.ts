export type TaskMode = 'standard' | 'turbo' | 'integrity'
export type RunStatus = 'queued' | 'running' | 'completed' | 'failed' | 'stopped'
export type RequestStatus = 'ok' | 'failed' | 'running' | 'queued'
export type PromptMode = 'text' | 'file' | 'generated' | 'raw'

export type TurboConfig = {
  init_concurrency: number
  max_concurrency: number
  step_size: number
  level_requests: number
  min_success_rate: number
  max_latency: string
}

export type IntegrityCase = {
  id?: string
  name?: string
  capability?: string
  request?: { prompt?: string }
  assertions?: unknown[]
  required?: boolean
}

export type IntegrityConfig = {
  enabled?: boolean
  suite?: string
  fail_fast?: boolean
  case_timeout_ms?: number
  rule_files?: string[]
}

export type TaskInput = {
  mode: TaskMode
  protocol: string
  endpoint_url: string
  base_url?: string
  proxy_url?: string
  api_key?: string
  model: string
  concurrency?: number
  count?: number
  stream?: boolean
  thinking?: boolean
  turbo?: boolean
  turbo_config?: TurboConfig
  integrity?: IntegrityConfig
  prompt_mode?: PromptMode
  prompt_text?: string
  prompt_file?: string
  prompt_length?: number
  report?: boolean
  timeout?: string
  log?: boolean
}

export type Task = {
  id: string
  name: string
  mode: TaskMode
  input: TaskInput
  created_at: string
  updated_at: string
  latest_run?: RunSummary
}

export type TaskConfig = {
  name: string
  input: TaskInput
}

export type RunSummary = {
  run_id: string
  task_id: string
  mode: TaskMode
  status: RunStatus
  protocol: string
  model: string
  started_at: string
  finished_at: string
  success_rate: number
  avg_ttft: string
  avg_tps: number
  cache_hit_rate: number
  rpm?: number
  tpm?: number
  max_stable_concurrency?: number
  error_summary?: string
}

export type RunState = {
  run_id: string
  task_id: string
  status: RunStatus
  mode: TaskMode
  started_at: string
  finished_at?: string
  total_reqs: number
  queued_reqs: number
  running_reqs: number
  done_reqs: number
  success_reqs: number
  failed_reqs: number
  skipped_reqs: number
  avg_tps: number
  avg_ttft: string
  success_rate: number
  cache_hit_rate: number
  rpm: number
  tpm: number
  requests: RequestDetail[]
  mode_state?: unknown
  mode_result?: unknown
  error_msg?: string
}

export type RequestDetail = {
  index: number
  status: RequestStatus
  success: boolean
  total_time: string
  ttft: string
  tps: number
  prompt_tokens: number
  completion_tokens: number
  cached_tokens: number
  cache_hit_rate: number
  dns_time: string
  connect_time: string
  tls_time: string
  target_ip: string
  error_message?: string
  request_body?: string
  response_body?: string
  level?: number
}

export type ProtocolMeta = {
  id: string
  name: string
  default_endpoint_url: string
}

export type IntegritySuite = {
  id: string
  name?: string
  description?: string
  cases?: IntegrityCase[]
}

async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  })
  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`
    try {
      const body = await response.json() as { error?: string }
      if (body.error) message = body.error
    } catch {
      // ignore non-json errors
    }
    throw new Error(message)
  }
  return response.json() as Promise<T>
}

export async function listTasks() {
  const body = await requestJSON<{ tasks: Task[] }>('/api/tasks')
  return body.tasks
}

export async function createTask(config: TaskConfig) {
  return requestJSON<Task>('/api/tasks', { method: 'POST', body: JSON.stringify(config) })
}

export async function duplicateTask(taskId: string) {
  return requestJSON<Task>(`/api/tasks/${encodeURIComponent(taskId)}/duplicate`, { method: 'POST' })
}

export async function listTaskRuns(taskId: string, limit = 20) {
  const body = await requestJSON<{ runs: RunSummary[] }>(`/api/tasks/${encodeURIComponent(taskId)}/runs?limit=${limit}`)
  return body.runs
}

export async function getRunState(runId: string) {
  return requestJSON<RunState>(`/api/runs/${encodeURIComponent(runId)}`)
}

export async function getRunRequests(runId: string) {
  const body = await requestJSON<{ requests: RequestDetail[] }>(`/api/runs/${encodeURIComponent(runId)}/requests`)
  return body.requests
}

export async function listProtocols() {
  const body = await requestJSON<{ protocols: ProtocolMeta[] }>('/api/meta/protocols')
  return body.protocols
}

export async function listIntegritySuites(protocol: string) {
  const body = await requestJSON<{ suites: IntegritySuite[] }>(`/api/integrity/suites?protocol=${encodeURIComponent(protocol)}`)
  return body.suites
}
