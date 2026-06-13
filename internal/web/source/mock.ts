export type TaskMode = 'standard' | 'turbo' | 'integrity'
export type RunStatus = 'running' | 'completed' | 'failed' | 'stopped'
export type RequestStatus = 'ok' | 'failed' | 'running' | 'queued'
export type PromptMode = 'text' | 'file' | 'generated' | 'raw'

export type PromptSpec = {
  mode: PromptMode
  label: string
  summary: string
  content: string
}

export type IntegrityCase = {
  id: string
  name: string
  capability: string
  prompt: string
  assertions: string[]
  timeout: string
  required: boolean
}

export type Task = {
  id: string
  name: string
  description: string
  mode: TaskMode
  protocol: string
  model: string
  endpoint: string
  concurrency: number
  requests: number
  updatedAt: string
  tags: string[]
  prompt?: string
  promptSpec?: PromptSpec
  standard?: {
    timeout: string
    stream: boolean
    thinking: boolean
    report: boolean
    log: boolean
  }
  turbo?: {
    levels: number[]
    initConcurrency: number
    maxConcurrency: number
    stepSize: number
    levelRequests: number
    minSuccessRate: number
    maxLatency: string
  }
  integrity?: {
    suite: string
    ruleFiles: string[]
    failFast: boolean
    caseTimeout: string
    cases: IntegrityCase[]
  }
}

export type RunRecord = {
  id: string
  taskId: string
  status: RunStatus
  startedAt: string
  duration: string
  requests: number
  success: number
  failed: number
  successRate: number
  errorRate: number
  avgTotalTime: string
  minTotalTime: string
  maxTotalTime: string
  stddevTotalTime: string
  avgTTFT: string
  minTTFT: string
  maxTTFT: string
  avgTPOT: string
  stddevTTFT: string
  avgTPS: number
  minTPS: number
  maxTPS: number
  stddevTPS: number
  rpm: number
  tpm: number
  totalThroughputTPS: number
  promptTokens: number
  cachedTokens: number
  outputTokens: number
  thinkingTokens: number
  cacheHitRate: number
  avgDNS: string
  avgConnect: string
  avgTLS: string
  targetIP: string
  summary: string
}

export type RequestDetail = {
  id: string
  runId: string
  index: number
  status: RequestStatus
  latency: string
  ttft: string
  tps: number
  promptTokens: number
  completionTokens: number
  cachedTokens: number
  dns: string
  connect: string
  tls: string
  targetIP: string
  request: string
  response: string
  error?: string
}

export const tasks: Task[] = [
  {
    id: 'task-prod-chat',
    name: '生产聊天接口基线',
    description: '固定并发批量请求，观察流式 TTFT、TPS、缓存命中和失败率。',
    mode: 'standard',
    protocol: 'openai-responses',
    model: 'gpt-4.1',
    endpoint: 'https://api.example.com/v1/responses',
    concurrency: 8,
    requests: 120,
    updatedAt: '今天 13:20',
    tags: ['prod', 'stream', 'baseline'],
    prompt: '解释缓存命中对大模型 API 性能指标的影响，输出三条结论。',
    promptSpec: { mode: 'text', label: 'prompt_text', summary: '直接使用文本 Prompt，按请求序号循环取内容。', content: '解释缓存命中对大模型 API 性能指标的影响，输出三条结论。' },
    standard: {
      timeout: '30s',
      stream: true,
      thinking: false,
      report: true,
      log: false,
    },
  },
  {
    id: 'task-cache-turbo',
    name: '缓存高并发爬坡',
    description: '逐级提高并发，寻找稳定并发上限和缓存收益拐点。',
    mode: 'turbo',
    protocol: 'anthropic-messages',
    model: 'claude-3-5-sonnet',
    endpoint: 'https://api.anthropic.com/v1/messages',
    concurrency: 4,
    requests: 360,
    updatedAt: '今天 10:42',
    tags: ['turbo', 'cache', 'capacity'],
    prompt: '使用 8000 token 共享上下文生成多条用户变体，观察缓存爬坡收益。',
    promptSpec: { mode: 'generated', label: 'prompt_length', summary: '按长度生成共享公共上下文与用户变体，观察缓存爬坡收益。', content: 'prompt_length: 8000\n公共上下文 + 多条用户变体由系统生成。' },
    turbo: {
      levels: [4, 8, 12, 16, 20, 24, 28, 32],
      initConcurrency: 4,
      maxConcurrency: 32,
      stepSize: 4,
      levelRequests: 45,
      minSuccessRate: 0.9,
      maxLatency: '10s',
    },
  },
  {
    id: 'task-json-integrity',
    name: 'JSON 协议完整性',
    description: '加载完整性测试集，逐个 case 发起请求并评估响应断言。',
    mode: 'integrity',
    protocol: 'openai-completions',
    model: 'gpt-4o-mini',
    endpoint: 'https://api.openai.com/v1/chat/completions',
    concurrency: 1,
    requests: 3,
    updatedAt: '昨天 21:08',
    tags: ['integrity', 'suite', 'cases'],
    integrity: {
      suite: 'openai-completions-smoke',
      ruleFiles: ['data/integrity/openai-completions.json'],
      failFast: true,
      caseTimeout: '30000ms',
      cases: [
        { id: 'basic-response-shape', name: '基础响应结构', capability: 'basic_request', prompt: 'Reply with a short greeting.', assertions: ['response.body exists', 'response.body.id exists', 'response.body.choices exists'], timeout: '30000ms', required: true },
        { id: 'json-object-shape', name: 'JSON 对象结构', capability: 'structured_output', prompt: 'Return a JSON object with name, price and currency.', assertions: ['response.body is valid JSON', 'currency exists', 'price is number'], timeout: '30000ms', required: true },
        { id: 'usage-metrics', name: '用量字段', capability: 'usage', prompt: 'Reply with one short sentence.', assertions: ['metrics.total_ms >= 0', 'usage exists'], timeout: '30000ms', required: false },
      ],
    },
  },
]

export const runs: RunRecord[] = [
  { id: 'run-prod-1042', taskId: 'task-prod-chat', status: 'running', startedAt: '13:24:08', duration: '01:42', requests: 120, success: 83, failed: 2, successRate: 96.5, errorRate: 2.4, avgTotalTime: '812ms', minTotalTime: '520ms', maxTotalTime: '1.21s', stddevTotalTime: '104ms', avgTTFT: '318ms', minTTFT: '248ms', maxTTFT: '612ms', avgTPOT: '12ms', stddevTTFT: '44ms', avgTPS: 84.2, minTPS: 0, maxTPS: 94.8, stddevTPS: 8.6, rpm: 468, tpm: 38200, totalThroughputTPS: 612.4, promptTokens: 812, cachedTokens: 544, outputTokens: 238, thinkingTokens: 0, cacheHitRate: 72.4, avgDNS: '4ms', avgConnect: '18ms', avgTLS: '29ms', targetIP: '34.117.12.8', summary: '当前运行整体稳定，出现少量 429 限流。' },
  { id: 'run-prod-1041', taskId: 'task-prod-chat', status: 'completed', startedAt: '12:18:31', duration: '02:34', requests: 120, success: 117, failed: 3, successRate: 97.5, errorRate: 2.5, avgTotalTime: '784ms', minTotalTime: '501ms', maxTotalTime: '1.08s', stddevTotalTime: '88ms', avgTTFT: '302ms', minTTFT: '232ms', maxTTFT: '540ms', avgTPOT: '11ms', stddevTTFT: '39ms', avgTPS: 86.8, minTPS: 72.4, maxTPS: 96.1, stddevTPS: 7.1, rpm: 452, tpm: 36120, totalThroughputTPS: 594.8, promptTokens: 798, cachedTokens: 525, outputTokens: 232, thinkingTokens: 0, cacheHitRate: 70.1, avgDNS: '4ms', avgConnect: '17ms', avgTLS: '28ms', targetIP: '34.117.12.8', summary: '基线通过，TTFT 比上一轮下降 5.3%。' },
  { id: 'run-prod-1040', taskId: 'task-prod-chat', status: 'failed', startedAt: '11:02:10', duration: '00:49', requests: 120, success: 35, failed: 6, successRate: 85.4, errorRate: 14.6, avgTotalTime: '1.12s', minTotalTime: '640ms', maxTotalTime: '2.40s', stddevTotalTime: '310ms', avgTTFT: '481ms', minTTFT: '310ms', maxTTFT: '1.10s', avgTPOT: '20ms', stddevTTFT: '122ms', avgTPS: 49.7, minTPS: 0, maxTPS: 71.2, stddevTPS: 18.2, rpm: 318, tpm: 21480, totalThroughputTPS: 356.2, promptTokens: 812, cachedTokens: 498, outputTokens: 204, thinkingTokens: 0, cacheHitRate: 64.8, avgDNS: '5ms', avgConnect: '23ms', avgTLS: '34ms', targetIP: '34.117.12.8', summary: '突发 429 导致提前失败。' },
  { id: 'run-cache-772', taskId: 'task-cache-turbo', status: 'completed', startedAt: '10:42:12', duration: '03:18', requests: 360, success: 356, failed: 4, successRate: 98.9, errorRate: 1.1, avgTotalTime: '690ms', minTotalTime: '488ms', maxTotalTime: '1.03s', stddevTotalTime: '76ms', avgTTFT: '412ms', minTTFT: '330ms', maxTTFT: '610ms', avgTPOT: '8ms', stddevTTFT: '41ms', avgTPS: 142.8, minTPS: 110.4, maxTPS: 168.9, stddevTPS: 12.7, rpm: 812, tpm: 74100, totalThroughputTPS: 1284.6, promptTokens: 1412, cachedTokens: 1204, outputTokens: 320, thinkingTokens: 0, cacheHitRate: 88.2, avgDNS: '6ms', avgConnect: '21ms', avgTLS: '35ms', targetIP: '18.64.22.18', summary: '稳定并发上限 48，level 6 开始成功率下降。' },
  { id: 'run-cache-771', taskId: 'task-cache-turbo', status: 'completed', startedAt: '09:55:47', duration: '03:26', requests: 300, success: 296, failed: 4, successRate: 98.7, errorRate: 1.3, avgTotalTime: '724ms', minTotalTime: '510ms', maxTotalTime: '1.14s', stddevTotalTime: '92ms', avgTTFT: '438ms', minTTFT: '352ms', maxTTFT: '690ms', avgTPOT: '9ms', stddevTTFT: '50ms', avgTPS: 137.4, minTPS: 104.6, maxTPS: 160.1, stddevTPS: 14.4, rpm: 790, tpm: 70240, totalThroughputTPS: 1198.1, promptTokens: 1412, cachedTokens: 1148, outputTokens: 306, thinkingTokens: 0, cacheHitRate: 84.9, avgDNS: '7ms', avgConnect: '22ms', avgTLS: '36ms', targetIP: '18.64.22.18', summary: '缓存命中率较低，峰值 TPS 低于新一轮。' },
  { id: 'run-json-319', taskId: 'task-json-integrity', status: 'failed', startedAt: '昨天 21:08', duration: '01:05', requests: 3, success: 2, failed: 1, successRate: 66.7, errorRate: 33.3, avgTotalTime: '540ms', minTotalTime: '420ms', maxTotalTime: '680ms', stddevTotalTime: '106ms', avgTTFT: '-', minTTFT: '-', maxTTFT: '-', avgTPOT: '9ms', stddevTTFT: '-', avgTPS: 61.4, minTPS: 0, maxTPS: 72.1, stddevTPS: 18.5, rpm: 220, tpm: 12600, totalThroughputTPS: 138.3, promptTokens: 146, cachedTokens: 0, outputTokens: 74, thinkingTokens: 0, cacheHitRate: 0, avgDNS: '4ms', avgConnect: '20ms', avgTLS: '28ms', targetIP: '104.18.7.192', summary: '1 个 case 未通过断言。' },
]

export const requestDetails: RequestDetail[] = [
  { id: 'req-080', runId: 'run-prod-1042', index: 80, status: 'ok', latency: '735ms', ttft: '282ms', tps: 90.2, promptTokens: 812, completionTokens: 228, cachedTokens: 544, dns: '4ms', connect: '18ms', tls: '29ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true, "input": "解释缓存命中对吞吐的影响" }', response: '缓存命中减少重复前缀计算，让首 token 更快返回。' },
  { id: 'req-081', runId: 'run-prod-1042', index: 81, status: 'ok', latency: '768ms', ttft: '291ms', tps: 88.9, promptTokens: 812, completionTokens: 236, cachedTokens: 544, dns: '4ms', connect: '18ms', tls: '29ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true, "input": "解释缓存命中对吞吐的影响" }', response: '重复上下文被缓存后，整体响应时间会更加稳定。' },
  { id: 'req-082', runId: 'run-prod-1042', index: 82, status: 'ok', latency: '824ms', ttft: '321ms', tps: 83.7, promptTokens: 812, completionTokens: 244, cachedTokens: 544, dns: '5ms', connect: '19ms', tls: '30ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true, "input": "解释缓存命中对吞吐的影响" }', response: '缓存收益通常体现在 TTFT 下降和 TPS 提升。' },
  { id: 'req-083', runId: 'run-prod-1042', index: 83, status: 'ok', latency: '776ms', ttft: '304ms', tps: 87.4, promptTokens: 812, completionTokens: 233, cachedTokens: 544, dns: '3ms', connect: '17ms', tls: '28ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true, "input": "解释缓存命中对吞吐的影响" }', response: '高缓存命中可以降低重复 prompt 的计算成本。' },
  { id: 'req-084', runId: 'run-prod-1042', index: 84, status: 'failed', latency: '1.21s', ttft: '-', tps: 0, promptTokens: 812, completionTokens: 0, cachedTokens: 0, dns: '5ms', connect: '22ms', tls: '31ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true }', response: '{ "error": { "type": "rate_limit_error" } }', error: '429 rate limit, retry after 2s' },
  { id: 'req-085', runId: 'run-prod-1042', index: 85, status: 'ok', latency: '790ms', ttft: '296ms', tps: 88.1, promptTokens: 812, completionTokens: 231, cachedTokens: 544, dns: '3ms', connect: '17ms', tls: '28ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true, "input": "解释缓存命中对吞吐的影响" }', response: '当请求复用相同上下文时，缓存可显著减少前置计算时间。' },
  { id: 'req-086', runId: 'run-prod-1042', index: 86, status: 'ok', latency: '812ms', ttft: '308ms', tps: 86.4, promptTokens: 812, completionTokens: 240, cachedTokens: 544, dns: '4ms', connect: '18ms', tls: '29ms', targetIP: '34.117.12.8', request: '{ "model": "gpt-4.1", "stream": true, "input": "解释缓存命中对吞吐的影响" }', response: '缓存命中减少重复前缀计算，降低首 token 延迟并提升整体吞吐。' },
  { id: 'req-240', runId: 'run-cache-772', index: 240, status: 'ok', latency: '752ms', ttft: '454ms', tps: 124.6, promptTokens: 1412, completionTokens: 304, cachedTokens: 1110, dns: '7ms', connect: '23ms', tls: '36ms', targetIP: '18.64.22.18', request: '{ "model": "claude-3-5-sonnet", "stream": true }', response: '- Stable cache reuse\n- Lower repeated-prefix cost' },
  { id: 'req-252', runId: 'run-cache-772', index: 252, status: 'ok', latency: '716ms', ttft: '430ms', tps: 138.4, promptTokens: 1412, completionTokens: 315, cachedTokens: 1168, dns: '6ms', connect: '21ms', tls: '35ms', targetIP: '18.64.22.18', request: '{ "model": "claude-3-5-sonnet", "stream": true }', response: '- Better throughput\n- Cache warm section reused' },
  { id: 'req-264', runId: 'run-cache-772', index: 264, status: 'ok', latency: '680ms', ttft: '404ms', tps: 151.7, promptTokens: 1412, completionTokens: 326, cachedTokens: 1204, dns: '6ms', connect: '21ms', tls: '34ms', targetIP: '18.64.22.18', request: '{ "model": "claude-3-5-sonnet", "stream": true }', response: '- Higher TPS\n- Similar output length' },
  { id: 'req-276', runId: 'run-cache-772', index: 276, status: 'ok', latency: '642ms', ttft: '386ms', tps: 165.8, promptTokens: 1412, completionTokens: 319, cachedTokens: 1220, dns: '6ms', connect: '20ms', tls: '34ms', targetIP: '18.64.22.18', request: '{ "model": "claude-3-5-sonnet", "stream": true }', response: '- Peak throughput\n- Cache hit rate improved' },
  { id: 'req-288', runId: 'run-cache-772', index: 288, status: 'failed', latency: '1.03s', ttft: '-', tps: 0, promptTokens: 1412, completionTokens: 0, cachedTokens: 0, dns: '8ms', connect: '28ms', tls: '42ms', targetIP: '18.64.22.18', request: '{ "model": "claude-3-5-sonnet", "stream": true }', response: '{ "error": "overloaded" }', error: 'overloaded at high concurrency' },
  { id: 'req-300', runId: 'run-cache-772', index: 300, status: 'ok', latency: '690ms', ttft: '410ms', tps: 146.2, promptTokens: 1412, completionTokens: 320, cachedTokens: 1204, dns: '6ms', connect: '21ms', tls: '35ms', targetIP: '18.64.22.18', request: '{ "model": "claude-3-5-sonnet", "stream": true }', response: '- Faster onboarding\n- Better observability\n- Lower latency for repeated prompts' },
  { id: 'case-034', runId: 'run-json-319', index: 34, status: 'ok', latency: '420ms', ttft: '-', tps: 72.1, promptTokens: 132, completionTokens: 61, cachedTokens: 0, dns: '4ms', connect: '19ms', tls: '27ms', targetIP: '104.18.7.192', request: '{ "messages": [{ "role": "user", "content": "Reply with a short greeting." }] }', response: 'Hello! How can I help?' },
  { id: 'case-035', runId: 'run-json-319', index: 35, status: 'ok', latency: '502ms', ttft: '-', tps: 66.8, promptTokens: 140, completionTokens: 72, cachedTokens: 0, dns: '4ms', connect: '20ms', tls: '28ms', targetIP: '104.18.7.192', request: '{ "messages": [{ "role": "user", "content": "Return a JSON object." }] }', response: '{ "name": "AIT", "price": 9.99, "currency": "USD" }' },
  { id: 'case-036', runId: 'run-json-319', index: 36, status: 'failed', latency: '540ms', ttft: '-', tps: 61.4, promptTokens: 146, completionTokens: 74, cachedTokens: 0, dns: '4ms', connect: '20ms', tls: '28ms', targetIP: '104.18.7.192', request: '{ "messages": [{ "role": "user", "content": "返回 JSON" }] }', response: '{ "name": "AIT", "price": 9.99 }', error: 'schema mismatch: missing currency' },
]

export function getTaskRuns(taskId: string) {
  return runs.filter((run) => run.taskId === taskId)
}

export function getRunRequests(runId: string) {
  return requestDetails.filter((request) => request.runId === runId)
}
