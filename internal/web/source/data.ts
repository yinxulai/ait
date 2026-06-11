import type { LucideIcon } from 'lucide-react'
import { Activity, Bot, Gauge, History, Network, ShieldCheck, Zap } from 'lucide-react'

export type RunStatus = 'running' | 'done' | 'failed' | 'queued'

export type Task = {
  id: string
  name: string
  mode: 'standard' | 'turbo' | 'integrity'
  protocol: 'openai-responses' | 'openai-completions' | 'anthropic-messages'
  model: string
  endpoint: string
  requests: number
  concurrency: number
  stream: boolean
  successRate: number
  avgTTFT: string
  avgTPS: number
  cacheHitRate: number
  status: RunStatus
  updatedAt: string
}

export type RequestRow = {
  id: number
  status: 'ok' | 'failed' | 'running'
  total: string
  ttft: string
  tps: number
  tokens: number
  error?: string
}

export type Metric = {
  label: string
  value: string
  hint: string
  icon: LucideIcon
}

export const tasks: Task[] = [
  {
    id: 'task-chat-prod',
    name: 'chat-prod',
    mode: 'standard',
    protocol: 'openai-responses',
    model: 'gpt-4.1',
    endpoint: 'https://api.example.com/v1/responses',
    requests: 100,
    concurrency: 8,
    stream: true,
    successRate: 96.9,
    avgTTFT: '310ms',
    avgTPS: 82.4,
    cacheHitRate: 72,
    status: 'running',
    updatedAt: '刚刚',
  },
  {
    id: 'task-cache-turbo',
    name: 'cache-turbo',
    mode: 'turbo',
    protocol: 'anthropic-messages',
    model: 'claude-3-5-sonnet',
    endpoint: 'https://api.anthropic.com/v1/messages',
    requests: 300,
    concurrency: 16,
    stream: true,
    successRate: 99.2,
    avgTTFT: '420ms',
    avgTPS: 142.8,
    cacheHitRate: 88,
    status: 'done',
    updatedAt: '22:10',
  },
  {
    id: 'task-json-integrity',
    name: 'json-integrity',
    mode: 'integrity',
    protocol: 'openai-completions',
    model: 'gpt-4o-mini',
    endpoint: 'https://api.openai.com/v1/chat/completions',
    requests: 36,
    concurrency: 4,
    stream: false,
    successRate: 91.7,
    avgTTFT: '-',
    avgTPS: 64.1,
    cacheHitRate: 0,
    status: 'failed',
    updatedAt: '昨天',
  },
]

export const requests: RequestRow[] = [
  { id: 32, status: 'ok', total: '820ms', ttft: '310ms', tps: 84, tokens: 982 },
  { id: 31, status: 'ok', total: '790ms', ttft: '295ms', tps: 86.1, tokens: 1004 },
  { id: 30, status: 'failed', total: '1.2s', ttft: '-', tps: 0, tokens: 0, error: '429 rate limit' },
  { id: 29, status: 'running', total: '...', ttft: '...', tps: 0, tokens: 0 },
  { id: 28, status: 'ok', total: '870ms', ttft: '335ms', tps: 79.8, tokens: 944 },
]

export const metrics: Metric[] = [
  { label: '成功率', value: '96.9%', hint: '+2.1% vs 最近一次', icon: ShieldCheck },
  { label: '平均 TTFT', value: '310ms', hint: 'stream 首 token', icon: Gauge },
  { label: '输出 TPS', value: '82.4', hint: 'tokens / second', icon: Zap },
  { label: '缓存命中', value: '72.0%', hint: 'cached input tokens', icon: Network },
]

export const navItems = [
  { label: '任务', icon: Bot, active: true },
  { label: '运行', icon: Activity, active: false },
  { label: '历史', icon: History, active: false },
]
