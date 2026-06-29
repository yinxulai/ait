import { Activity, CheckCircle2, Clock3, Database, FileJson, Gauge, Network, TrendingUp, XCircle } from 'lucide-react'
import type { ReactNode } from 'react'
import useEmblaCarousel from 'embla-carousel-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'
import { formatDate, formatNumber, formatPercent, requestKey } from '@/lib/task-utils'
import type { RequestDetail, RunStatus, RunSummary } from '@/api'

const statusLabel: Record<RunStatus, string> = {
  queued: '排队中',
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  stopped: '已停止',
}

const statusStyle: Record<RunStatus, string> = {
  queued: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300',
  running: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-900 dark:bg-blue-950/40 dark:text-blue-300',
  completed: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300',
  failed: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300',
  stopped: 'border-zinc-200 bg-zinc-50 text-zinc-700 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300',
}

export type RunDetailProps = {
  run?: RunSummary
  requests: RequestDetail[]
  selectedRequest?: RequestDetail
  onSelectRequest: (id: string) => void
}

export type TaskRunHistoryProps = {
  runs: RunSummary[]
  selectedRun?: RunSummary
  onChooseRun: (run: RunSummary) => void
  samplesByRun: Record<string, RequestDetail[]>
}

export function TaskRunHistory({ runs, selectedRun, onChooseRun, samplesByRun }: TaskRunHistoryProps) {
  return (
    <Card className="min-h-0 rounded-2xl border bg-card shadow-none ring-0">
      <CardHeader className="p-4 pb-3 sm:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base"><Clock3 className="size-4" />执行记录</CardTitle>
            <CardDescription>任务最重要的信息。先看趋势，再选择某次执行。</CardDescription>
          </div>
          <Badge variant="outline" className="w-fit gap-1.5"><TrendingUp className="size-3.5" />{runs.length} 次执行</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 p-4 pt-0 sm:p-5 sm:pt-0">
        <RunHistorySummary runs={runs} samplesByRun={samplesByRun} />
        {runs.length === 0 ? (
          <div className="rounded-xl border border-dashed bg-muted/25 px-4 py-10 text-center text-sm text-muted-foreground">这个任务还没有执行记录。点击上方“开始运行”后，这里会显示每次执行的结果。</div>
        ) : (
          <div className="overflow-x-auto rounded-xl bg-muted/20">
            <Table className="min-w-190">
              <TableHeader>
                <TableRow>
                  <TableHead>开始时间</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>成功率</TableHead>
                  <TableHead>TTFT</TableHead>
                  <TableHead>TPS</TableHead>
                  <TableHead>RPM</TableHead>
                  <TableHead>TPM</TableHead>
                  <TableHead>缓存</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((run) => (
                  <TableRow key={run.run_id} onClick={() => onChooseRun(run)} className={cn('cursor-pointer transition-colors', selectedRun?.run_id === run.run_id && 'bg-accent')}>
                    <TableCell>
                      <div className="font-medium">{formatDate(run.started_at)}</div>
                      <div className="mt-1 text-xs text-muted-foreground">{run.run_id}</div>
                    </TableCell>
                    <TableCell><StatusBadge status={run.status} /></TableCell>
                    <TableCell><RunRate value={run.success_rate} /></TableCell>
                    <TableCell>{run.avg_ttft || '-'}</TableCell>
                    <TableCell className="tabular-nums">{formatNumber(run.avg_tps)}</TableCell>
                    <TableCell className="tabular-nums">{formatNumber(run.rpm)}</TableCell>
                    <TableCell className="tabular-nums">{formatNumber(run.tpm)}</TableCell>
                    <TableCell className="tabular-nums">{formatPercent(run.cache_hit_rate)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function RunDetail({ run, requests, selectedRequest, onSelectRequest }: RunDetailProps) {
  if (!run) {
    return <Card className="rounded-2xl border bg-card shadow-none ring-0"><CardContent className="flex min-h-60 items-center justify-center p-6 text-sm text-muted-foreground"><Clock3 className="mr-2 size-4" />当前任务暂无执行记录。</CardContent></Card>
  }

  const successCount = requests.filter((request) => request.success).length
  const failedCount = requests.filter((request) => request.status === 'failed' || !request.success).length
  const doneCount = successCount + failedCount
  const progress = requests.length > 0 ? Math.round((doneCount / requests.length) * 100) : Math.round(run.success_rate || 0)

  return (
    <Card className="rounded-2xl border bg-card shadow-none ring-0">
      <CardHeader className="p-4 sm:p-5">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="min-w-0 space-y-2">
            <CardTitle className="flex items-center gap-2 text-base"><FileJson className="size-4" />选中执行</CardTitle>
            <CardDescription className="break-all leading-5">{run.run_id}</CardDescription>
            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
              <span>{formatDate(run.started_at)}</span>
              <span>·</span>
              <span>{run.protocol}</span>
              <span>·</span>
              <span>{run.model}</span>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2 xl:justify-end">
            <StatusBadge status={run.status} />
            <Badge variant="outline">{requests.length} 个请求样本</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 p-4 pt-0 sm:p-5 sm:pt-0">
        <section className="space-y-4 rounded-xl bg-muted/35 p-4">
          <div className="mb-3 flex items-center justify-between gap-3 text-sm">
            <span className="font-medium">执行完成度</span>
            <span className="text-muted-foreground">{doneCount} / {requests.length}</span>
          </div>
          <Progress value={progress} />
          <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <InlineMetric icon={<CheckCircle2 className="size-4" />} label="成功率" value={formatPercent(run.success_rate)} tone={failedCount > 0 ? 'danger' : 'success'} />
            <InlineMetric icon={<Gauge className="size-4" />} label="TTFT" value={run.avg_ttft || '-'} />
            <InlineMetric icon={<TrendingUp className="size-4" />} label="TPS" value={formatNumber(run.avg_tps)} />
            <InlineMetric icon={<Database className="size-4" />} label="缓存" value={formatPercent(run.cache_hit_rate)} />
          </div>
        </section>

        <section className="grid gap-3 lg:grid-cols-3">
          <CompactMetricList title="运行摘要" icon={<Clock3 className="size-4" />} variant="solid" items={[
            ['开始时间', formatDate(run.started_at)],
            ['结束时间', formatDate(run.finished_at)],
            ['状态', statusLabel[run.status]],
            ['错误摘要', run.error_summary || '-'],
          ]} />
          <CompactMetricList title="吞吐与速度" icon={<TrendingUp className="size-4" />} variant="soft" items={[
            ['平均 TPS', formatNumber(run.avg_tps)],
            ['RPM', formatNumber(run.rpm)],
            ['TPM', formatNumber(run.tpm)],
            ['稳定并发', String(run.max_stable_concurrency || '-')],
          ]} />
          <CompactMetricList title="样本质量" icon={<Network className="size-4" />} variant="outline" items={[
            ['总样本', String(requests.length)],
            ['成功', String(successCount)],
            ['失败', String(failedCount)],
            ['缓存命中', formatPercent(run.cache_hit_rate)],
          ]} />
        </section>

        <section className="min-w-0 rounded-xl bg-muted/25 p-4">
          <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div className="flex items-center gap-2 text-sm font-medium"><Network className="size-4" />请求样本检查</div>
              <div className="mt-1 text-xs leading-5 text-muted-foreground">按请求样本查看耗时、Token、网络与原始请求响应。</div>
            </div>
            <Badge variant="outline" className="w-fit bg-background">{requests.length} 个样本</Badge>
          </div>
          <div className="space-y-4">
            {requests.length === 0 ? (
              <div className="rounded-xl bg-background px-3 py-6 text-center text-sm text-muted-foreground">暂无请求样本。</div>
            ) : (
              <RequestSampleSelector requests={requests} selectedRequest={selectedRequest} onSelectRequest={onSelectRequest} />
            )}
            {selectedRequest && <RequestPanel request={selectedRequest} compact />}
          </div>
        </section>
      </CardContent>
    </Card>
  )
}

function RequestSampleSelector({ requests, selectedRequest, onSelectRequest }: { requests: RequestDetail[]; selectedRequest?: RequestDetail; onSelectRequest: (id: string) => void }) {
  const selectedKey = selectedRequest ? requestKey(selectedRequest) : ''
  const [emblaRef, emblaApi] = useEmblaCarousel({ align: 'start', dragFree: true, containScroll: 'trimSnaps' })

  function handleSelect(key: string) {
    onSelectRequest(key)
    const index = requests.findIndex((request) => requestKey(request) === key)
    if (index >= 0) emblaApi?.scrollTo(index)
  }

  function moveNext() {
    if (!emblaApi) return
    if (emblaApi.canScrollNext()) {
      emblaApi.scrollNext()
      return
    }
    emblaApi.scrollTo(0)
  }

  return (
    <div className="space-y-3 rounded-xl bg-background p-3">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="text-xs leading-5 text-muted-foreground">选择样本后，下方详情会展示对应请求和响应内容。</div>
        <Select value={selectedKey} onValueChange={handleSelect}>
          <SelectTrigger className="w-full md:w-72">
            <SelectValue placeholder="选择请求样本" />
          </SelectTrigger>
          <SelectContent>
            {requests.map((request) => {
              const key = requestKey(request)
              return <SelectItem key={key} value={key}>{sampleLabel(request)}</SelectItem>
            })}
          </SelectContent>
        </Select>
      </div>
      <div className="overflow-hidden pb-1" ref={emblaRef}>
        <div className="flex gap-2">
          {requests.map((request) => {
            const key = requestKey(request)
            const active = selectedKey === key
            const failed = request.status === 'failed' || !request.success
            const Icon = failed ? XCircle : CheckCircle2

            return (
              <button key={key} type="button" onClick={() => handleSelect(key)} className={cn('embla__slide flex shrink-0 basis-42 items-center gap-2 rounded-xl border bg-muted/35 px-3 py-2 text-left text-sm transition hover:bg-muted/70 sm:basis-52', active ? 'border-primary/40 bg-primary/10 text-foreground shadow-xs' : 'border-transparent text-muted-foreground')}>
                <Icon className={cn('size-4 shrink-0', failed ? 'text-red-500' : 'text-emerald-500')} />
                <span className="min-w-0">
                  <span className="block truncate font-medium">#{request.index}{request.case_id ? ` · ${request.case_id}` : ''}</span>
                  <span className="mt-0.5 block truncate text-xs">{request.total_time} · TTFT {request.ttft}</span>
                </span>
              </button>
            )
          })}
        </div>
      </div>
      <div className="flex items-center justify-between gap-2">
        <div className="text-xs text-muted-foreground">可拖拽滚动，也可以点击单项自动切换到对应样本。</div>
        <Button type="button" variant="outline" size="sm" onClick={moveNext}>下一项</Button>
      </div>
    </div>
  )
}

function sampleLabel(request: RequestDetail) {
  return `#${request.index}${request.case_id ? ` · ${request.case_id}` : ''} · ${request.total_time} · TTFT ${request.ttft}`
}

function CompactMetricList({ title, icon, items, variant }: { title: string; icon: ReactNode; items: Array<[string, string]>; variant: 'solid' | 'soft' | 'outline' }) {
  const panelStyle = {
    solid: 'bg-background',
    soft: 'bg-muted/25',
    outline: 'bg-muted/45',
  }[variant]

  return (
    <div className={cn('rounded-xl p-4', panelStyle)}>
      <div className="mb-3 flex items-center gap-2 text-sm font-medium">{icon}{title}</div>
      <div className="divide-y divide-border/60 text-sm">
        {items.map(([label, value]) => (
          <div key={label} className="flex min-w-0 items-center justify-between gap-4 py-2 first:pt-0 last:pb-0">
            <span className="shrink-0 text-muted-foreground">{label}</span>
            <span className="truncate text-right font-medium" title={value}>{value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

function RequestPanel({ request, compact = false }: { request: RequestDetail; compact?: boolean }) {
  const metricItems = [
    ['Case ID', request.case_id || '-'],
    ['总耗时', request.total_time || '-'],
    ['TTFT', request.ttft || '-'],
    ['TPS', formatNumber(request.tps)],
    ['输入 Token', formatNumber(request.prompt_tokens)],
    ['输出 Token', formatNumber(request.completion_tokens)],
    ['缓存 Token', formatNumber(request.cached_tokens)],
    ['Target IP', request.target_ip || '-'],
    ['DNS', request.dns_time || '-'],
    ['Connect', request.connect_time || '-'],
    ['TLS', request.tls_time || '-'],
  ]

  return (
    <div className={cn('rounded-xl bg-background p-4', compact && 'min-h-0 xl:mt-0')}>
      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <div className="font-medium">请求详情 #{request.index}</div>
          <div className="mt-1 text-xs text-muted-foreground">状态、耗时、Token 与网络阶段明细</div>
        </div>
        {request.status === 'failed' || !request.success ? <Badge variant="outline" className="rounded-full border-red-200 bg-red-50 px-3 py-1 text-red-700">失败</Badge> : null}
      </div>
      <div className={cn('grid gap-4', compact ? 'xl:grid-cols-[minmax(0,1fr)]' : 'xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]')}>
        <div className="rounded-xl bg-muted/35 p-3">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium"><Gauge className="size-4" />本次指标</div>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-6">
            {metricItems.map(([label, value], index) => <RequestMetricTile key={label} label={label} value={value} variant={index % 3} />)}
          </div>
        </div>
        <div className="space-y-4">
          <CodeBlock label="请求内容" value={request.request_body || '-'} icon={<Network className="size-3.5" />} />
          <CodeBlock label="响应内容" value={request.response_body || request.error_message || '-'} icon={<FileJson className="size-3.5" />} />
        </div>
      </div>
    </div>
  )
}

function RequestMetricTile({ label, value, variant }: { label: string; value: string; variant: number }) {
  const tileStyle = ['bg-background', 'bg-muted/20', 'bg-muted/45'][variant]

  return (
    <div className={cn('min-w-0 rounded-lg px-3 py-2.5', tileStyle)}>
      <div className="text-[11px] leading-4 text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-sm font-semibold tabular-nums" title={value}>{value}</div>
    </div>
  )
}

function CodeBlock({ label, value, icon }: { label: string; value: string; icon?: ReactNode }) {
  const isJson = value.trim().startsWith('{') || value.trim().startsWith('[')
  const prettyValue = (() => {
    if (!isJson) return value
    try {
      return JSON.stringify(JSON.parse(value), null, 2)
    } catch {
      return value
    }
  })()

  return (
    <div className="overflow-hidden rounded-xl bg-muted/25">
      <div className="flex items-center gap-1.5 border-b bg-background/80 px-3 py-2 text-xs font-medium text-muted-foreground">{icon}{label}</div>
      <pre className="max-h-44 overflow-auto bg-muted/55 px-3 py-3 text-xs leading-5 text-foreground/90"><code>{prettyValue}</code></pre>
    </div>
  )
}

function StatusBadge({ status }: { status: RunStatus }) {
  const Icon = status === 'running' ? Activity : status === 'completed' ? CheckCircle2 : status === 'failed' ? XCircle : Clock3
  return <Badge variant="outline" className={cn('gap-1', statusStyle[status])}><Icon className="size-3" />{statusLabel[status]}</Badge>
}

function RunRate({ value }: { value: number }) {
  return (
    <div className="flex min-w-28 items-center gap-2">
      <div className="h-1.5 flex-1 rounded-full bg-muted"><div className="h-full rounded-full bg-primary" style={{ width: `${value}%` }} /></div>
      <span className="w-10 text-right text-sm tabular-nums">{value}%</span>
    </div>
  )
}

function InlineMetric({ icon, label, value, tone = 'neutral' }: { icon: ReactNode; label: string; value: string; tone?: 'neutral' | 'success' | 'danger' }) {
  return (
    <div className="flex min-w-0 items-center gap-3 rounded-xl bg-background px-3 py-3">
      <div className={cn('flex size-9 shrink-0 items-center justify-center rounded-lg bg-background text-muted-foreground', tone === 'success' && 'text-emerald-600', tone === 'danger' && 'text-red-600')}>{icon}</div>
      <div className="min-w-0">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="truncate text-lg font-semibold tabular-nums" title={value}>{value}</div>
      </div>
    </div>
  )
}

function RunHistorySummary({ runs, samplesByRun }: { runs: RunSummary[]; samplesByRun: Record<string, RequestDetail[]> }) {
  const completedRuns = runs.filter((run) => run.status === 'completed')
  const failedRuns = runs.filter((run) => run.status === 'failed')
  const latestRun = runs[0]
  const latestSamples = latestRun ? samplesByRun[latestRun.run_id]?.length || 0 : 0
  const averageSuccessRate = runs.length > 0 ? Math.round(runs.reduce((sum, run) => sum + run.success_rate, 0) / runs.length) : 0
  const bestStableConcurrency = Math.max(0, ...runs.map((run) => run.max_stable_concurrency || 0))

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <HistoryStat label="历史执行" value={String(runs.length)} helper={`${completedRuns.length} 次完成 / ${failedRuns.length} 次失败`} variant="solid" />
      <HistoryStat label="平均成功率" value={formatPercent(averageSuccessRate)} helper="按历史执行均值计算" variant="soft" />
      <HistoryStat label="最新样本数" value={String(latestSamples)} helper={latestRun ? latestRun.run_id : '暂无执行'} variant="outline" />
      <HistoryStat label="最佳稳定并发" value={bestStableConcurrency > 0 ? String(bestStableConcurrency) : '-'} helper="来自历史执行摘要" variant="soft" />
    </div>
  )
}

function HistoryStat({ label, value, helper, variant }: { label: string; value: string; helper: string; variant: 'solid' | 'soft' | 'outline' }) {
  const panelStyle = {
    solid: 'bg-background',
    soft: 'bg-muted/25',
    outline: 'bg-muted/45',
  }[variant]

  return (
    <div className={cn('rounded-xl p-3', panelStyle)}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-2xl font-semibold tabular-nums" title={value}>{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground" title={helper}>{helper}</div>
    </div>
  )
}
