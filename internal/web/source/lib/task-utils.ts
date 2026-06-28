import type { PromptMode, ProtocolMeta, RequestDetail, Task, TaskInput, TaskMode } from '../api'

export const modeLabel: Record<TaskMode, string> = {
  standard: '标准压测',
  turbo: 'Turbo 爬坡',
  integrity: '完整性校验',
}

export const promptModeLabel: Record<PromptMode, string> = {
  text: '直接输入文本',
  file: '从文件读取',
  generated: '按长度生成',
  raw: '原始请求 JSON',
}

export const protocolLabel: Record<string, string> = {
  'openai-completions': 'OpenAI Completions 接口',
  'openai-responses': 'OpenAI Responses 接口',
  'anthropic-messages': 'Anthropic Messages 接口',
}

export const promptFieldLabel: Record<string, string> = {
  prompt_file: 'Prompt 文件',
  prompt_length: 'Prompt 长度',
  prompt_text: 'Prompt 文本',
}

export const createModeHint: Record<TaskMode, string> = {
  standard: '固定并发与请求总数',
  turbo: '起始并发、最大并发、递增步长',
  integrity: '测试集与执行控制',
}

const draftName: Record<TaskMode, string> = {
  standard: '新建标准压测任务',
  turbo: '新建 Turbo 爬坡任务',
  integrity: '新建完整性校验任务',
}

const draftModel: Record<TaskMode, string> = {
  standard: 'gpt-4.1',
  turbo: 'claude-3-5-sonnet',
  integrity: 'gpt-4o-mini',
}

const draftProtocol: Record<TaskMode, string> = {
  standard: 'openai-responses',
  turbo: 'anthropic-messages',
  integrity: 'openai-completions',
}

const draftPrompt: Record<Exclude<TaskMode, 'integrity'>, string> = {
  standard: '解释缓存命中对大模型 API 性能指标的影响，输出三条结论。',
  turbo: '使用 8000 token 共享上下文生成多条用户变体，观察缓存爬坡收益。',
}

export const protocolOptions = ['openai-completions', 'openai-responses', 'anthropic-messages'] as const
export const promptModeOptions = ['text', 'file', 'generated', 'raw'] as const

export type TaskDraft = {
  mode: TaskMode
  name: string
  protocol: string
  endpoint: string
  apiKey: string
  model: string
  concurrency: number
  requests: number
  promptMode: PromptMode
  promptText: string
  promptFile: string
  promptLength: number
  timeout: string
  stream: boolean
  thinking: boolean
  report: boolean
  log: boolean
  turboInitConcurrency: number
  turboMaxConcurrency: number
  turboStepSize: number
  turboLevelRequests: number
  turboMinSuccessRate: number
  turboMaxLatency: string
  integritySuite: string
  integrityFailFast: boolean
  integrityCaseTimeout: string
}

export type PromptSpec = { mode: PromptMode; label: string; summary: string; content: string }

export function makeInitialDraft(mode: TaskMode, protocols: ProtocolMeta[] = []): TaskDraft {
  const protocol = draftProtocol[mode]
  return {
    mode,
    name: draftName[mode],
    protocol,
    endpoint: defaultEndpoint(protocol, protocols),
    apiKey: '',
    model: draftModel[mode],
    concurrency: mode === 'integrity' ? 1 : mode === 'turbo' ? 4 : 8,
    requests: mode === 'integrity' ? 1 : mode === 'turbo' ? 60 : 120,
    promptMode: mode === 'turbo' ? 'generated' : 'text',
    promptText: mode === 'integrity' ? '' : draftPrompt[mode],
    promptFile: '',
    promptLength: mode === 'turbo' ? 8000 : 1200,
    timeout: '30s',
    stream: true,
    thinking: false,
    report: true,
    log: false,
    turboInitConcurrency: 4,
    turboMaxConcurrency: 64,
    turboStepSize: 4,
    turboLevelRequests: 60,
    turboMinSuccessRate: 0.9,
    turboMaxLatency: '10s',
    integritySuite: defaultSuite(protocol),
    integrityFailFast: true,
    integrityCaseTimeout: '30000',
  }
}

export function draftFromTask(task: Task): TaskDraft {
  const input = task.input
  const draft = makeInitialDraft(taskMode(task))
  const promptMode = input.prompt_mode ?? draft.promptMode
  return {
    ...draft,
    mode: taskMode(task),
    name: `${task.name} 副本`,
    protocol: input.protocol || draft.protocol,
    endpoint: input.endpoint_url || draft.endpoint,
    apiKey: input.api_key ?? '',
    model: input.model || draft.model,
    concurrency: input.concurrency ?? draft.concurrency,
    requests: input.count ?? draft.requests,
    promptMode,
    promptText: input.prompt_text ?? draft.promptText,
    promptFile: input.prompt_file ?? '',
    promptLength: input.prompt_length ?? draft.promptLength,
    timeout: input.timeout ?? draft.timeout,
    stream: input.stream ?? draft.stream,
    thinking: input.thinking ?? draft.thinking,
    report: input.report ?? draft.report,
    log: input.log ?? draft.log,
    turboInitConcurrency: input.turbo_config?.init_concurrency ?? draft.turboInitConcurrency,
    turboMaxConcurrency: input.turbo_config?.max_concurrency ?? draft.turboMaxConcurrency,
    turboStepSize: input.turbo_config?.step_size ?? draft.turboStepSize,
    turboLevelRequests: input.turbo_config?.level_requests ?? input.count ?? draft.turboLevelRequests,
    turboMinSuccessRate: input.turbo_config?.min_success_rate ?? draft.turboMinSuccessRate,
    turboMaxLatency: input.turbo_config?.max_latency ?? draft.turboMaxLatency,
    integritySuite: input.integrity?.suite ?? draft.integritySuite,
    integrityFailFast: input.integrity?.fail_fast ?? draft.integrityFailFast,
    integrityCaseTimeout: input.integrity?.case_timeout_ms ? String(input.integrity.case_timeout_ms) : draft.integrityCaseTimeout,
  }
}

export function taskFromDraft(id: string, draft: TaskDraft): Task {
  return {
    id,
    name: draft.name.trim() || draftName[draft.mode],
    mode: draft.mode,
    input: inputJsonFromDraft(draft),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
}

export function inputJsonFromDraft(draft: TaskDraft): TaskInput {
  const input: TaskInput = {
    mode: draft.mode,
    protocol: draft.protocol,
    endpoint_url: draft.endpoint.trim(),
    ...(draft.apiKey.trim() ? { api_key: draft.apiKey.trim() } : {}),
    model: draft.model.trim(),
    stream: draft.stream,
    report: draft.report,
    log: draft.log,
    ...(draft.timeout.trim() ? { timeout: draft.timeout.trim() } : {}),
  }

  if (draft.mode === 'integrity') {
    return {
      ...input,
      concurrency: 1,
      count: 0,
      integrity: {
        enabled: true,
        suite: draft.integritySuite.trim(),
        fail_fast: draft.integrityFailFast,
        case_timeout_ms: durationToMs(draft.integrityCaseTimeout),
      },
    }
  }

  const prompt = promptInputFromDraft(draft)
  if (draft.mode === 'turbo') {
    return {
      ...input,
      turbo: true,
      count: draft.turboLevelRequests,
      ...prompt,
      turbo_config: {
        init_concurrency: draft.turboInitConcurrency,
        max_concurrency: draft.turboMaxConcurrency,
        step_size: draft.turboStepSize,
        level_requests: draft.turboLevelRequests,
        min_success_rate: draft.turboMinSuccessRate,
        max_latency: draft.turboMaxLatency.trim(),
      },
    }
  }

  return {
    ...input,
    concurrency: draft.concurrency,
    count: draft.requests,
    thinking: draft.thinking,
    ...prompt,
  }
}

function promptInputFromDraft(draft: TaskDraft): Partial<TaskInput> {
  if (draft.promptMode === 'file') return { prompt_mode: 'file', prompt_file: draft.promptFile.trim() }
  if (draft.promptMode === 'generated') return { prompt_mode: 'generated', prompt_length: draft.promptLength }
  return { prompt_mode: draft.promptMode, prompt_text: draft.promptText }
}

export function promptSpec(input: TaskInput): PromptSpec | undefined {
  if (input.mode === 'integrity') return undefined
  if (input.prompt_mode === 'file') return { mode: 'file', label: 'prompt_file', summary: `从文件读取 Prompt：${input.prompt_file || '-'}`, content: input.prompt_file || '-' }
  if (input.prompt_mode === 'generated') return { mode: 'generated', label: 'prompt_length', summary: `按长度生成 ${input.prompt_length || 0} token Prompt。`, content: `Prompt 长度：${input.prompt_length || 0}` }
  if (input.prompt_mode === 'raw') return { mode: 'raw', label: 'prompt_text', summary: '原始 JSON 请求体。', content: input.prompt_text || '-' }
  return { mode: 'text', label: 'prompt_text', summary: '直接使用文本 Prompt。', content: input.prompt_text || '-' }
}

export function taskMode(task: Task) {
  return task.input.mode || task.mode
}

export function taskModel(task: Task) {
  return task.input.model || '-'
}

export function taskProtocol(task: Task) {
  return task.input.protocol || '-'
}

export function taskEndpoint(task: Task) {
  return task.input.endpoint_url || task.input.base_url || task.input.proxy_url || '-'
}

export function maskSecret(value?: string) {
  if (!value) return '未配置或已隐藏'
  if (value.length <= 8) return '••••••••'
  return `${value.slice(0, 4)}••••${value.slice(-4)}`
}

export function redactSecretInput(input: TaskInput): TaskInput {
  return input.api_key ? { ...input, api_key: maskSecret(input.api_key) } : input
}

export function taskConcurrency(task: Task) {
  return task.input.turbo_config?.init_concurrency ?? task.input.concurrency ?? 0
}

export function taskRequests(task: Task) {
  return task.input.count ?? 0
}

export function requestKey(request?: RequestDetail) {
  return request ? `${request.index}-${request.level ?? 0}` : ''
}

export function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

export function formatNumber(value?: number) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return Number.isInteger(value) ? value.toString() : value.toFixed(2)
}

export function formatPercent(value?: number) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return `${Math.round(value)}%`
}

export function turboLevelsFromConfig(config?: TaskInput['turbo_config']) {
  if (!config) return []
  const levels: number[] = []
  const stepSize = Math.max(1, config.step_size || 1)
  for (let value = Math.max(1, config.init_concurrency); value <= Math.max(config.init_concurrency, config.max_concurrency); value += stepSize) levels.push(value)
  return levels
}

export function defaultEndpoint(protocol: string, protocols: ProtocolMeta[] = []) {
  const meta = protocols.find((item) => item.id === protocol)
  if (meta?.default_endpoint_url) return meta.default_endpoint_url
  if (protocol === 'openai-responses') return 'https://api.openai.com/v1/responses'
  if (protocol === 'anthropic-messages') return 'https://api.anthropic.com/v1/messages'
  return 'https://api.openai.com/v1/chat/completions'
}

function defaultSuite(protocol: string) {
  if (protocol === 'openai-responses') return 'openai-responses-smoke'
  if (protocol === 'anthropic-messages') return 'anthropic-messages-smoke'
  return 'openai-completions-smoke'
}

export function nextStepLabel(step: number) {
  if (step === 0) return '填写基础信息'
  if (step === 1) return '填写类型配置'
  if (step === 2) return '检查并确认'
  return '下一步'
}

export function createStepHint(draft: TaskDraft, step: number) {
  if (step === 1) return '请先填写任务名称、协议、模型名称和请求地址。'
  if (step === 2 && draft.mode === 'integrity') return '请选择当前协议已加载的测试集，并填写单个用例超时。'
  if (step === 2 && draft.mode === 'turbo') return '请确认 Prompt、并发爬坡参数、请求超时和停止条件均已填写。'
  if (step === 2) return '请确认 Prompt、并发数、请求总数和请求超时均已填写。'
  return '请先完成当前步骤。'
}

export function isStepValid(draft: TaskDraft, step: number): boolean {
  if (step === 0) return Boolean(draft.mode)
  if (step === 1) return isBasicConfigValid(draft)
  if (step === 2) return isModeConfigValid(draft)
  return isDraftValid(draft)
}

export function isDraftValid(draft: TaskDraft): boolean {
  return isBasicConfigValid(draft) && isModeConfigValid(draft)
}

function isBasicConfigValid(draft: TaskDraft): boolean {
  return Boolean(draft.name.trim() && draft.protocol.trim() && draft.endpoint.trim() && draft.model.trim())
}

function isModeConfigValid(draft: TaskDraft): boolean {
  if (draft.mode === 'integrity') return Boolean(draft.integritySuite.trim() && durationToMs(draft.integrityCaseTimeout) > 0)
  if (draft.mode === 'turbo' && (draft.turboInitConcurrency <= 0 || draft.turboMaxConcurrency <= 0 || draft.turboStepSize <= 0 || draft.turboLevelRequests <= 0 || draft.turboMinSuccessRate <= 0 || !draft.turboMaxLatency.trim())) return false
  if (draft.mode === 'standard' && (draft.concurrency <= 0 || draft.requests <= 0)) return false
  if (!draft.timeout.trim()) return false
  if (draft.promptMode === 'file') return Boolean(draft.promptFile.trim())
  if (draft.promptMode === 'generated') return draft.promptLength > 0
  return Boolean(draft.promptText.trim())
}

export function toNumber(value: string) {
  return Math.max(0, Number.parseInt(value, 10) || 0)
}

function durationToMs(value: string) {
  const normalized = value.trim()
  const amount = Number.parseFloat(normalized)
  if (!Number.isFinite(amount)) return 0
  if (normalized.endsWith('ms')) return Math.round(amount)
  if (normalized.endsWith('s')) return Math.round(amount * 1000)
  return Math.round(amount)
}
