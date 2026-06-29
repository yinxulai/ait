import { Activity, BarChart3, CheckCircle2, Clock3, Database, FileJson, Gauge, Network, ShieldCheck, TrendingUp, XCircle } from 'lucide-react'
import type { ReactNode } from 'react'
import { Line } from 'react-chartjs-2'
import useEmblaCarousel from 'embla-carousel-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'
import { formatDate, formatNumber, formatPercent, requestKey } from '@/lib/task-utils'
import type { IntegrityAssertionResult, IntegrityCaseResult, IntegrityModeState, IntegrityResult, RequestDetail, RunState, RunStatus, RunSummary } from '@/api'

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

export function TaskRunHistory({ runs, selectedRun, onChooseRun, samplesByRun }: TaskRunHistoryProps) {
  return (
    <Card className="min-h-0 rounded-2xl border bg-card shadow-none ring-0">
      <CardHeader className="p-4 pb-3 sm:p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-base"><Clock3 className="size-4" />执行记录</CardTitle>
            <CardDescription>任务最重要的信息。先看趋势，再选择某次执行。</CardDescription>
          </div>
          <Badge variant="outline" className="w-fit gap-1.5"><BarChart3 className="size-3.5" />{runs.length} 次执行</Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 p-4 pt-0 sm:p-5 sm:pt-0">
        <ExecutionTrend runs={runs} samplesByRun={samplesByRun} />
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

export function RunDetail({ run, state, requests, selectedRequest, onSelectRequest }: RunDetailProps) {
  if (!run) {
    return <Card className="rounded-2xl border bg-card shadow-none ring-0"><CardContent className="flex min-h-60 items-center justify-center p-6 text-sm text-muted-foreground"><Clock3 className="mr-2 size-4" />当前任务暂无执行记录。</CardContent></Card>
  }

  const integrity = integritySnapshot(state)
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
            {integrity && <Badge variant="outline"><ShieldCheck className="size-3" />{integrity.suiteId || 'integrity'}</Badge>}
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 p-4 pt-0 sm:p-5 sm:pt-0">
        <div className="space-y-4">
          <section className="space-y-4 rounded-xl bg-muted/25 p-4">
            <div className="mb-3 flex items-center justify-between gap-3 text-sm">
              <span className="font-medium">执行完成度</span>
              <span className="text-muted-foreground">{doneCount} / {requests.length}</span>
            </div>
            <Progress value={progress} />
            <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <InlineMetric icon={<CheckCircle2 className="size-4" />} label={integrity ? '用例' : '成功率'} value={integrity ? `${integrity.passedCases}/${integrity.totalCases}` : formatPercent(run.success_rate)} tone={failedCount > 0 || (integrity?.failedCases ?? 0) > 0 ? 'danger' : 'success'} />
              <InlineMetric icon={<Gauge className="size-4" />} label="TTFT" value={run.avg_ttft || '-'} />
              <InlineMetric icon={<TrendingUp className="size-4" />} label="TPS" value={formatNumber(run.avg_tps)} />
              <InlineMetric icon={<Database className="size-4" />} label={integrity ? '断言' : '缓存'} value={integrity ? `${integrity.passedAssertions}/${integrity.totalAssertions}` : formatPercent(run.cache_hit_rate)} />
            </div>
          </section>

          {integrity && <IntegrityRunPanel snapshot={integrity} />}

          <section className="grid gap-3 lg:grid-cols-3">
            <CompactMetricList title="运行摘要" icon={<Clock3 className="size-4" />} items={[
              ['开始时间', formatDate(run.started_at)],
              ['结束时间', formatDate(run.finished_at)],
              ['状态', statusLabel[run.status]],
              ['错误摘要', run.error_summary || '-'],
            ]} />
            <CompactMetricList title="吞吐与速度" icon={<TrendingUp className="size-4" />} items={[
              ['平均 TPS', formatNumber(run.avg_tps)],
              ['RPM', formatNumber(run.rpm)],
              ['TPM', formatNumber(run.tpm)],
              ['稳定并发', String(run.max_stable_concurrency || '-')],
            ]} />
            <CompactMetricList title="样本质量" icon={<Network className="size-4" />} items={[
              ['总样本', String(requests.length)],
              ['成功', String(successCount)],
              ['失败', String(failedCount)],
              ['缓存命中', formatPercent(run.cache_hit_rate)],
            ]} />
          </section>
        </div>

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

function CompactMetricList({ title, icon, items }: { title: string; icon: ReactNode; items: Array<[string, string]> }) {
  return (
    <div className="rounded-xl bg-muted/25 p-4">
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
        <div className="rounded-xl bg-muted/25 p-3">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium"><Gauge className="size-4" />本次指标</div>
          <div className="grid gap-2 sm:grid-cols-2 2xl:grid-cols-3">
            {metricItems.map(([label, value]) => <RequestMetricTile key={label} label={label} value={value} />)}
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

function RequestMetricTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg bg-background/80 px-3 py-2.5">
      <div className="text-[11px] leading-4 text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-sm font-semibold tabular-nums" title={value}>{value}</div>
    </div>
  )
}

type IntegritySnapshot = {
  suiteId: string
  status: string
  message: string
  currentCaseId: string
  totalCases: number
  passedCases: number
  failedCases: number
  warnedCases: number
  skippedCases: number
  requiredFailedCases: number
  totalAssertions: number
  passedAssertions: number
  failedAssertions: number
  warnedAssertions: number
  cases: IntegrityCaseResult[]
  assertions: IntegrityAssertionResult[]
}

function IntegrityRunPanel({ snapshot }: { snapshot: IntegritySnapshot }) {
  return (
    <div className="rounded-2xl border bg-background/70 p-4">
      <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex items-center gap-2 text-sm font-medium"><ShieldCheck className="size-4" />完整性校验结果</div>
          <div className="mt-1 text-xs leading-5 text-muted-foreground">{snapshot.suiteId || '-'} · {snapshot.message || snapshot.status || '等待后端同步状态'}</div>
        </div>
        <Badge variant="outline">{snapshot.currentCaseId ? `当前 ${snapshot.currentCaseId}` : snapshot.status || 'suite'}</Badge>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <NumberStat compact label="总用例" value={String(snapshot.totalCases)} />
        <NumberStat compact label="通过用例" value={String(snapshot.passedCases)} />
        <NumberStat compact label="失败用例" value={String(snapshot.failedCases)} />
        <NumberStat compact label="必需失败" value={String(snapshot.requiredFailedCases)} />
      </div>
      <div className="mt-4 space-y-2 pr-3 xl:max-h-130 xl:overflow-y-auto xl:scrollbar-gutter-stable">
        {snapshot.cases.length === 0 ? <div className="rounded-xl bg-muted/50 px-3 py-3 text-xs text-muted-foreground">完整性用例尚未产生结果。</div> : snapshot.cases.map((item) => (
          <div key={item.case_id} className="rounded-xl border bg-background/80 px-3 py-3 text-sm">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="truncate font-medium">{item.name || item.case_id}</div>
                <div className="mt-1 text-xs text-muted-foreground">{item.capability || '未标注能力'} · {item.duration || '-'}</div>
              </div>
              <Badge variant="outline" className={cn(item.status === 'passed' && 'border-emerald-200 bg-emerald-50 text-emerald-700', item.status === 'failed' && 'border-red-200 bg-red-50 text-red-700')}>{item.status}</Badge>
            </div>
            <div className="mt-3 grid gap-2 text-xs sm:grid-cols-4">
              <KeyValue label="断言" value={String(item.total_assertions)} />
              <KeyValue label="通过" value={String(item.passed_assertions)} />
              <KeyValue label="失败" value={String(item.failed_assertions)} />
              <KeyValue label="警告" value={String(item.warned_assertions)} />
            </div>
            {item.error_message && <div className="mt-3 rounded-lg bg-destructive/10 px-3 py-2 text-xs text-destructive">{item.error_message}</div>}
          </div>
        ))}
      </div>
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
    <div className="flex min-w-0 items-center gap-3 rounded-xl bg-muted/45 px-3 py-3">
      <div className={cn('flex size-9 shrink-0 items-center justify-center rounded-lg bg-background text-muted-foreground', tone === 'success' && 'text-emerald-600', tone === 'danger' && 'text-red-600')}>{icon}</div>
      <div className="min-w-0">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="truncate text-lg font-semibold tabular-nums" title={value}>{value}</div>
      </div>
    </div>
  )
}

function NumberStat({ label, value, compact }: { label: string; value: string; compact?: boolean }) {
  return (
    <div>
      <div className={cn('font-semibold tracking-tight', compact ? 'text-base' : 'text-2xl')}>{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg bg-muted/35 px-3 py-2">
      <div className="text-[11px] leading-4 text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-sm font-medium" title={value}>{value}</div>
    </div>
  )
}

function ExecutionTrend({ runs, samplesByRun }: { runs: RunSummary[]; samplesByRun: Record<string, RequestDetail[]> }) {
  if (runs.length === 0) return null

  const latestRun = runs[0]
  const samples = [...(samplesByRun[latestRun.run_id] ?? [])].sort((a, b) => a.index - b.index)
  const maxTps = Math.max(...samples.map((request) => request.tps), 1)
  const maxLatency = Math.max(...samples.map((request) => parseDurationMs(request.total_time)), 1)
  const maxTTFT = Math.max(...samples.map((request) => parseDurationMs(request.ttft)), 1)
  const maxOutputTokens = Math.max(...samples.map((request) => request.completion_tokens), 1)
  const maxNetwork = Math.max(...samples.map((request) => requestNetworkMs(request)), 1)
  const cacheRates = samples.map((request) => request.cache_hit_rate || (request.prompt_tokens > 0 ? Math.round((request.cached_tokens / request.prompt_tokens) * 100) : 0))
  const maxCache = Math.max(...cacheRates, 1)
  const failedSamples = samples.filter((request) => request.status === 'failed').length

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
      <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <div className="flex items-center gap-2 text-sm font-medium"><TrendingUp className="size-4" />最新执行样本曲线</div>
            <div className="mt-1 text-xs text-muted-foreground">{latestRun.run_id} 内部请求的吞吐、延迟、TTFT、Token、网络和缓存变化。</div>
          </div>
          <Badge variant="outline" className="gap-1.5"><BarChart3 className="size-3.5" />{samples.length} 个样本</Badge>
        </div>
        {samples.length > 0 ? <SampleLineChart samples={samples} cacheRates={cacheRates} maxTps={maxTps} maxLatency={maxLatency} maxTTFT={maxTTFT} maxOutputTokens={maxOutputTokens} maxNetwork={maxNetwork} maxCache={maxCache} /> : <div className="flex h-44 items-center justify-center rounded-2xl bg-muted/40 text-sm text-muted-foreground">最新执行暂无请求样本。</div>}
        <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
          <Legend color="bg-primary/75" label="TPS" />
          <Legend color="bg-muted-foreground/45" label="总耗时" />
          <Legend color="bg-sky-500/70" label="TTFT" />
          <Legend color="bg-violet-500/70" label="输出 Token" />
          <Legend color="bg-amber-500/70" label="网络" />
          <Legend color="bg-emerald-500/70" label="缓存命中" />
          <span className="ml-auto">失败样本 {failedSamples}</span>
        </div>
      </div>
      <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-3 text-sm font-medium">最近表现</div>
        <div className="grid grid-cols-2 gap-3">
          <NumberStat compact label="成功率" value={formatPercent(latestRun.success_rate)} />
          <NumberStat compact label="状态" value={statusLabel[latestRun.status]} />
          <NumberStat compact label="TTFT" value={latestRun.avg_ttft || '-'} />
          <NumberStat compact label="TPS" value={formatNumber(latestRun.avg_tps)} />
          <NumberStat compact label="RPM" value={formatNumber(latestRun.rpm)} />
          <NumberStat compact label="TPM" value={formatNumber(latestRun.tpm)} />
          <NumberStat compact label="缓存" value={formatPercent(latestRun.cache_hit_rate)} />
          <NumberStat compact label="稳定并发" value={String(latestRun.max_stable_concurrency || '-')} />
        </div>
      </div>
    </div>
  )
}

function SampleLineChart({ samples, cacheRates, maxTps, maxLatency, maxTTFT, maxOutputTokens, maxNetwork, maxCache }: { samples: RequestDetail[]; cacheRates: number[]; maxTps: number; maxLatency: number; maxTTFT: number; maxOutputTokens: number; maxNetwork: number; maxCache: number }) {
  const labels = samples.map((request) => `#${request.index}`)
  const datasets = [
    chartDataset('TPS', samples.map((request) => normalizeChartValue(request.tps, maxTps)), '#18181b', 3, true),
    chartDataset('总耗时', samples.map((request) => normalizeChartValue(parseDurationMs(request.total_time), maxLatency)), '#71717a'),
    chartDataset('TTFT', samples.map((request) => normalizeChartValue(parseDurationMs(request.ttft), maxTTFT)), '#0ea5e9'),
    chartDataset('输出 Token', samples.map((request) => normalizeChartValue(request.completion_tokens, maxOutputTokens)), '#8b5cf6'),
    chartDataset('网络', samples.map((request) => normalizeChartValue(requestNetworkMs(request), maxNetwork)), '#f59e0b'),
    chartDataset('缓存命中', samples.map((_, index) => normalizeChartValue(cacheRates[index], maxCache)), '#10b981'),
  ]

  return (
    <div className="h-56 overflow-hidden rounded-2xl bg-muted/40 p-3">
      <Line
        data={{ labels, datasets }}
        options={{
          responsive: true,
          maintainAspectRatio: false,
          interaction: { mode: 'index', intersect: false },
          plugins: {
            legend: { display: false },
            tooltip: { backgroundColor: '#18181b', borderColor: '#27272a', borderWidth: 1, padding: 10, cornerRadius: 12 },
          },
          scales: {
            x: { border: { display: false }, grid: { display: false }, ticks: { color: '#71717a', font: { size: 11 } } },
            y: { display: false, min: 0, max: 100, grid: { color: 'rgba(113,113,122,0.18)', drawTicks: false } },
          },
        }}
      />
    </div>
  )
}

function chartDataset(label: string, data: number[], color: string, width = 2, points = false) {
  return {
    label,
    data,
    borderColor: color,
    backgroundColor: color,
    borderWidth: width,
    tension: 0.35,
    pointRadius: points ? 3 : 0,
    pointHoverRadius: 5,
  }
}

function normalizeChartValue(value: number, max: number) {
  return Math.round((Math.max(0, value) / Math.max(max, 1)) * 100)
}

function requestNetworkMs(request: RequestDetail) {
  return parseDurationMs(request.dns_time) + parseDurationMs(request.connect_time) + parseDurationMs(request.tls_time)
}

function parseDurationMs(value: string) {
  const normalized = value.trim()
  if (!normalized || normalized === '-') return 0
  const amount = Number.parseFloat(normalized)
  if (Number.isNaN(amount)) return 0
  if (normalized.endsWith('ms')) return amount
  if (normalized.endsWith('s')) return amount * 1000
  return amount
}

function Legend({ color, label }: { color: string; label: string }) {
  return <span className="inline-flex items-center gap-1.5"><span className={cn('size-2 rounded-full', color)} />{label}</span>
}

function integritySnapshot(state?: RunState): IntegritySnapshot | undefined {
  if (!state || state.mode !== 'integrity') return undefined
  const modeState = state.mode_state as IntegrityModeState | undefined
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

function isIntegrityResult(value: unknown): value is IntegrityResult {
  return Boolean(value && typeof value === 'object' && 'suite_id' in value && 'cases' in value)
}

function suiteIdFromState(modeState?: IntegrityModeState) {
  if (!modeState) return ''
  if (typeof modeState.suite === 'string') return modeState.suite
  return modeState.suite?.id ?? modeState.suite_status?.suite ?? ''
}

function flattenAssertions(cases: IntegrityCaseResult[]) {
  return cases.flatMap((item) => item.assertions ?? [])
}

function sumCases(cases: IntegrityCaseResult[], key: 'total_assertions' | 'passed_assertions' | 'failed_assertions' | 'warned_assertions', fallback: number) {
  if (cases.length === 0) return fallback
  return cases.reduce((sum, item) => sum + (item[key] || 0), 0)
}

export type { IntegritySnapshot, RunDetailProps, TaskRunHistoryProps }
