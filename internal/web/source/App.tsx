import { useId, useMemo, useState } from 'react'
import { Activity, AlertTriangle, BarChart3, CheckCircle2, ChevronRight, Clock3, ClipboardList, Copy, Database, FileJson, Gauge, Hash, ListChecks, Network, Plus, Route, Search, Settings2, ShieldCheck, TrendingUp, XCircle, Zap } from 'lucide-react'
import { CategoryScale, Chart as ChartJS, Filler, Legend as ChartLegend, LinearScale, LineElement, PointElement, Tooltip as ChartTooltip } from 'chart.js'
import { Line } from 'react-chartjs-2'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Stepper } from '@/components/ui/stepper'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { getRunRequests, getTaskRuns, requestDetails, runs, tasks as mockTasks, type PromptMode, type RequestDetail, type RunRecord, type RunStatus, type Task, type TaskMode } from './mock'

const modeLabel: Record<TaskMode, string> = {
  standard: '标准压测',
  turbo: 'Turbo 爬坡',
  integrity: '完整性校验',
}

const statusLabel: Record<RunStatus, string> = {
  running: '运行中',
  completed: '已完成',
  failed: '失败',
  stopped: '已停止',
}

const statusStyle: Record<RunStatus, string> = {
  running: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-900 dark:bg-blue-950/40 dark:text-blue-300',
  completed: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-300',
  failed: 'border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300',
  stopped: 'border-zinc-200 bg-zinc-50 text-zinc-700 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300',
}

const promptModeLabel: Record<PromptMode, string> = {
  text: '直接输入文本',
  file: '从文件读取',
  generated: '按长度生成',
  raw: '原始请求 JSON',
}

const protocolLabel: Record<string, string> = {
  'openai-completions': 'OpenAI Completions 接口',
  'openai-responses': 'OpenAI Responses 接口',
  'anthropic-messages': 'Anthropic Messages 接口',
}

const promptFieldLabel: Record<string, string> = {
  prompt_file: 'Prompt 文件',
  prompt_length: 'Prompt 长度',
  prompt_text: 'Prompt 文本',
}

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, ChartTooltip, ChartLegend)

function App() {
  const [query, setQuery] = useState('')
  const [taskList, setTaskList] = useState<Task[]>(mockTasks)
  const [selectedTaskId, setSelectedTaskId] = useState(mockTasks[0]?.id ?? '')
  const [selectedRunId, setSelectedRunId] = useState(getTaskRuns(selectedTaskId)[0]?.id ?? '')
  const [selectedRequestId, setSelectedRequestId] = useState(getRunRequests(selectedRunId)[0]?.id ?? '')

  const filteredTasks = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return taskList
    return taskList.filter((task) => [task.name, task.description, task.model, task.protocol, ...task.tags].some((text) => text.toLowerCase().includes(keyword)))
  }, [query, taskList])

  const selectedTask = taskList.find((task) => task.id === selectedTaskId) ?? taskList[0]
  const taskRuns = selectedTask ? getTaskRuns(selectedTask.id) : []
  const selectedRun = taskRuns.find((run) => run.id === selectedRunId) ?? taskRuns[0]
  const runRequests = selectedRun ? getRunRequests(selectedRun.id) : []
  const selectedRequest = runRequests.find((request) => request.id === selectedRequestId) ?? runRequests[0]

  function chooseTask(task: Task) {
    const nextRuns = getTaskRuns(task.id)
    const nextRun = nextRuns[0]
    const nextRequest = nextRun ? getRunRequests(nextRun.id)[0] : undefined
    setSelectedTaskId(task.id)
    setSelectedRunId(nextRun?.id ?? '')
    setSelectedRequestId(nextRequest?.id ?? '')
  }

  function chooseRun(run: RunRecord) {
    const nextRequest = getRunRequests(run.id)[0]
    setSelectedRunId(run.id)
    setSelectedRequestId(nextRequest?.id ?? '')
  }

  function createTask(task: Task) {
    setTaskList((current) => [task, ...current])
    chooseTask(task)
  }

  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_top_left,var(--muted),transparent_34rem)] bg-background text-foreground">
      <div className="mx-auto flex min-h-screen max-w-[1680px] flex-col gap-4 p-3 sm:p-5 lg:p-6">
        <header className="rounded-3xl border bg-card/90 px-4 py-3 shadow-sm ring-1 ring-border/30 backdrop-blur sm:px-5">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex min-w-0 items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
                <Activity className="size-4" />
              </div>
              <div className="min-w-0">
                <h1 className="truncate text-lg font-semibold tracking-tight sm:text-xl">AIT 执行观测台</h1>
                <p className="truncate text-xs text-muted-foreground sm:text-sm">任务 → 执行记录 → 单次详情</p>
              </div>
            </div>
            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground sm:justify-end">
              <TopStat icon={<ListChecks className="size-3.5" />} label="任务" value={taskList.length.toString()} />
              <TopStat icon={<Clock3 className="size-3.5" />} label="执行" value={runs.length.toString()} />
              <TopStat icon={<Hash className="size-3.5" />} label="样本" value={requestDetails.length.toString()} />
            </div>
          </div>
        </header>

        {selectedTask && <section className="grid min-h-[calc(100vh-112px)] gap-4 lg:gap-5 xl:grid-cols-[320px_minmax(0,1fr)] 2xl:grid-cols-[340px_minmax(0,1fr)]">
          <Card className="overflow-hidden rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40 xl:sticky xl:top-6 xl:h-[calc(100vh-170px)]">
            <CardHeader className="space-y-4 p-4 pb-3 sm:p-5">
              <div>
                <CardTitle className="flex items-center gap-2 text-base"><ListChecks className="size-4" />任务列表</CardTitle>
                <CardDescription>选择任务后先看执行记录。</CardDescription>
              </div>
              <CreateTaskSheet onCreate={createTask} />
              <div className="relative">
                <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input value={query} onChange={(event) => setQuery(event.target.value)} className="h-10 rounded-2xl pl-9" placeholder="搜索任务 / 模型 / 协议 / 标签" />
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <ScrollArea className="h-auto max-h-[460px] xl:h-[calc(100vh-360px)] xl:max-h-none">
                <div className="space-y-2.5 p-4 pt-0 sm:p-5 sm:pt-0">
                  {filteredTasks.map((task) => {
                    const taskRuns = getTaskRuns(task.id)
                    const latestRun = taskRuns[0]
                    return (
                      <button key={task.id} type="button" onClick={() => chooseTask(task)} className={cn('group w-full rounded-2xl border bg-background/70 px-3.5 py-3 text-left transition hover:border-primary/30 hover:bg-accent/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring', selectedTask.id === task.id && 'border-primary bg-accent ring-1 ring-primary/10')}>
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0 flex-1">
                            <div className="flex min-w-0 items-center gap-2">
                              <ModeIcon mode={task.mode} className="size-3.5 shrink-0 text-muted-foreground" />
                              <div className="truncate font-medium leading-6">{task.name}</div>
                            </div>
                            <div className="mt-1 truncate text-xs text-muted-foreground">{modeLabel[task.mode]} · {task.model}</div>
                          </div>
                          <ChevronRight className="mt-1 size-4 shrink-0 text-muted-foreground/60 transition group-hover:translate-x-0.5 group-hover:text-foreground" />
                        </div>
                        <div className="mt-3 flex items-center justify-between gap-3 border-t border-border/60 pt-2.5 text-xs text-muted-foreground">
                          <span>{taskRuns.length} 次执行</span>
                          <span className="inline-flex items-center gap-2">
                            {latestRun && <span className={cn('tabular-nums', latestRun.status === 'failed' ? 'text-red-600' : 'text-emerald-600')}>{latestRun.successRate}%</span>}
                            <span>{task.updatedAt}</span>
                          </span>
                        </div>
                      </button>
                    )
                  })}
                </div>
              </ScrollArea>
            </CardContent>
          </Card>

          <div className="flex min-w-0 flex-col gap-4 lg:gap-5">
            <TaskOverview task={selectedTask} onCreate={createTask} />
            <TaskRunHistory runs={taskRuns} selectedRun={selectedRun} onChooseRun={chooseRun} />
            <RunDetail run={selectedRun} requests={runRequests} selectedRequest={selectedRequest} onSelectRequest={setSelectedRequestId} />
          </div>
        </section>}
      </div>
    </main>
  )
}

type TaskDraft = {
  mode: TaskMode
  name: string
  protocol: string
  endpoint: string
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
  integrityRuleFiles: string
  integrityFailFast: boolean
  integrityCaseTimeout: string
}

const createSteps = [
  { title: '任务类型', description: '选择创建模式' },
  { title: '基础信息', description: '名称与目标' },
  { title: '类型配置', description: '填写运行参数' },
  { title: '确认创建', description: '预览输入' },
] as const

type CreateTaskSheetProps = {
  onCreate: (task: Task) => void
  sourceTask?: Task
  variant?: 'create' | 'copy'
}

function CreateTaskSheet({ onCreate, sourceTask, variant = 'create' }: CreateTaskSheetProps) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<TaskDraft>(() => sourceTask ? draftFromTask(sourceTask) : makeInitialDraft('standard'))
  const canSubmit = isDraftValid(draft)

  function initialDraft() {
    return sourceTask ? draftFromTask(sourceTask) : makeInitialDraft('standard')
  }

  function patch(update: Partial<TaskDraft>) {
    setDraft((current) => ({ ...current, ...update }))
  }

  function changeMode(mode: TaskMode) {
    const next = makeInitialDraft(mode)
    setDraft((current) => ({ ...next, name: current.name || next.name }))
  }

  function reset() {
    setDraft(initialDraft())
  }

  function submit() {
    if (!canSubmit) return
    onCreate(taskFromDraft(`task-draft-${Date.now().toString(36)}`, draft))
    setOpen(false)
    reset()
  }

  return (
    <Dialog open={open} onOpenChange={(next) => { setOpen(next); if (next) setDraft(initialDraft()); if (!next) reset() }}>
      <DialogTrigger asChild>
        {variant === 'copy' ? (
          <Button variant="outline" size="sm" className="rounded-full">
            <Copy className="size-3.5" />复制为新任务
          </Button>
        ) : (
          <Button className="h-12 w-full justify-start rounded-2xl px-3.5 text-sm shadow-sm">
            <span className="flex size-7 items-center justify-center rounded-full bg-primary-foreground/15"><Plus className="size-4" /></span>
            创建新任务
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="flex max-h-[92vh] flex-col overflow-hidden p-0">
        <DialogHeader className="shrink-0 border-b bg-muted/30 p-0 px-8 py-5">
          <DialogTitle className="flex items-center gap-2 text-lg">{variant === 'copy' ? <Copy className="size-4" /> : <Plus className="size-4" />}{variant === 'copy' ? '复制任务' : '创建任务'}</DialogTitle>
          <DialogDescription>{variant === 'copy' && sourceTask ? `已基于「${sourceTask.name}」带入配置，可直接微调。` : '按任务类型分步填写最少必要配置。'}</DialogDescription>
        </DialogHeader>
        <Stepper key={`${open}-${sourceTask?.id ?? 'new'}`} steps={createSteps} rootClassName="flex min-h-0 flex-1 flex-col space-y-0" className="shrink-0 px-8 pt-6 pb-5" canAdvance={(current) => isStepValid(draft, current)}>
          {({ current, isFirst, isLast, canGoNext, next, previous }) => {
            const validationHint = createStepHint(draft, current)
            return (
              <>
                <div className="min-h-0 flex-1 overflow-y-auto px-8 pb-6">
                  {current === 0 && <CreateStepType draft={draft} onModeChange={changeMode} />}
                  {current === 1 && <CreateStepBasics draft={draft} onPatch={patch} />}
                  {current === 2 && <CreateStepModeConfig draft={draft} onPatch={patch} />}
                  {current === 3 && <CreateStepReview draft={draft} />}
                </div>
                <DialogFooter className="shrink-0 gap-3 border-t bg-muted/20 p-0 px-8 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-h-5 text-xs leading-5 text-muted-foreground">{!canGoNext && !isLast ? validationHint : isLast ? '确认无误后创建，当前原型只会写入本地 mock 列表。' : '可以继续下一步，也可以点击上方已完成步骤返回修改。'}</div>
                  <div className="flex shrink-0 gap-2">
                    <Button variant="outline" onClick={previous} disabled={isFirst}>上一步</Button>
                    {!isLast ? <Button onClick={next} disabled={!canGoNext}>{nextStepLabel(current)}</Button> : <Button onClick={submit} disabled={!canSubmit}>创建任务</Button>}
                  </div>
                </DialogFooter>
              </>
            )
          }}
        </Stepper>
      </DialogContent>
    </Dialog>
  )
}

function CreateStepType({ draft, onModeChange }: { draft: TaskDraft; onModeChange: (mode: TaskMode) => void }) {
  return (
    <section className="space-y-4">
      <div>
        <div className="text-sm font-medium">选择任务类型</div>
        <p className="mt-1 text-xs text-muted-foreground">先确定任务类型，再填写对应配置。</p>
      </div>
      <RadioGroup value={draft.mode} onValueChange={(mode) => onModeChange(mode as TaskMode)} className="grid gap-3">
        {(['standard', 'turbo', 'integrity'] as TaskMode[]).map((mode) => {
          const id = `create-mode-${mode}`
          return (
            <Label key={mode} htmlFor={id} className={cn('flex cursor-pointer items-start gap-3 rounded-2xl border bg-background/60 p-4 text-sm transition hover:bg-accent', draft.mode === mode && 'border-primary bg-accent ring-1 ring-primary/15')}>
              <RadioGroupItem id={id} value={mode} className="mt-1" />
              <span className="min-w-0">
                <span className="flex items-center gap-2 font-medium text-foreground"><ModeIcon mode={mode} className="size-4 text-muted-foreground" />{modeLabel[mode]}</span>
                <span className="mt-1 block text-xs leading-5 text-muted-foreground">{createModeHint[mode]}</span>
              </span>
            </Label>
          )
        })}
      </RadioGroup>
    </section>
  )
}

function CreateStepBasics({ draft, onPatch }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void }) {
  return (
    <div className="space-y-6">
      <section className="border-b pb-6">
        <div className="mb-5 flex items-center gap-2 text-sm font-medium"><ListChecks className="size-4" />任务信息</div>
        <div className="grid gap-4">
          <FormField label="任务名称" required description="用于在任务列表和执行记录中识别这次配置。"><Input value={draft.name} onChange={(event) => onPatch({ name: event.target.value })} /></FormField>
        </div>
      </section>

      <section>
        <div className="mb-5 flex items-center gap-2 text-sm font-medium"><Route className="size-4" />请求目标</div>
        <div className="grid gap-4">
          <FormField label="协议" required description="决定请求体结构和默认地址，切换后会自动带入对应接口地址。"><OptionPicker value={draft.protocol} options={protocolOptions} onChange={(protocol) => onPatch({ protocol, endpoint: defaultEndpoint(protocol) })} /></FormField>
          <FormField label="模型名称" required description="填写要压测或校验的模型标识，会写入最终请求配置。"><Input value={draft.model} onChange={(event) => onPatch({ model: event.target.value })} /></FormField>
          <FormField label="请求地址" required description="目标 API 的完整地址；如使用网关或代理，可在这里改为内部地址。"><Input value={draft.endpoint} onChange={(event) => onPatch({ endpoint: event.target.value })} /></FormField>
        </div>
      </section>
    </div>
  )
}

function CreateStepModeConfig({ draft, onPatch }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void }) {
  if (draft.mode === 'integrity') {
    return (
      <div className="space-y-6">
        <section className="border-b pb-6">
          <div className="mb-5 flex items-center gap-2 text-sm font-medium"><ShieldCheck className="size-4" />测试集来源</div>
          <div className="grid gap-4">
            <FormField label="测试集名称" required description="用于选择一组协议兼容性用例；完整性校验以测试集为核心。"><Input value={draft.integritySuite} onChange={(event) => onPatch({ integritySuite: event.target.value })} /></FormField>
            <FormField label="规则文件" required description="每行一个规则文件路径，用于描述测试用例、断言和必需能力。"><Textarea value={draft.integrityRuleFiles} onChange={(event) => onPatch({ integrityRuleFiles: event.target.value })} className="min-h-32" placeholder="data/integrity/openai-completions.json" /></FormField>
          </div>
        </section>
        <section>
          <div className="mb-5 flex items-center gap-2 text-sm font-medium"><Settings2 className="size-4" />执行控制</div>
          <div className="grid gap-4">
            <FormField label="单个用例超时" required description="单位为毫秒；单个用例超过该时间后视为超时。"><Input type="number" value={draft.integrityCaseTimeout} onChange={(event) => onPatch({ integrityCaseTimeout: event.target.value })} placeholder="30000" /></FormField>
            <BooleanToggle label="失败时立即停止" description="开启后任一必需用例失败就停止后续用例，适合快速发现阻塞问题。" value={draft.integrityFailFast} onChange={(value) => onPatch({ integrityFailFast: value })} />
          </div>
          <p className="mt-4 text-xs leading-5 text-muted-foreground">完整性校验不填写 Prompt；只选择测试集与执行控制。</p>
        </section>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PromptInputForm draft={draft} onPatch={onPatch} />
      {draft.mode === 'standard' ? <StandardConfigForm draft={draft} onPatch={onPatch} /> : <TurboConfigForm draft={draft} onPatch={onPatch} />}
      <RequestOptionsForm draft={draft} onPatch={onPatch} />
    </div>
  )
}

function PromptInputForm({ draft, onPatch }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void }) {
  return (
    <section className="border-b pb-6">
      <div className="mb-5 flex items-center gap-2 text-sm font-medium"><FileJson className="size-4" />Prompt 输入</div>
      <div className="grid gap-4">
        <FormField label="Prompt 来源" required description="标准压测和 Turbo 都需要 Prompt；可以直接输入、从文件读取、按长度生成，或提供完整请求 JSON。"><OptionPicker value={draft.promptMode} options={promptModeOptions} onChange={(promptMode) => onPatch({ promptMode: promptMode as PromptMode })} /></FormField>
        {draft.promptMode === 'generated' && <FormField label="Prompt 长度" required description="生成指定 token 规模的测试 Prompt，用于控制输入长度。"><Input type="number" value={draft.promptLength} onChange={(event) => onPatch({ promptLength: toNumber(event.target.value) })} /></FormField>}
        {draft.promptMode === 'file' && <FormField label="Prompt 文件" required description="填写本地 Prompt 文件路径，运行时从文件读取内容。"><Input value={draft.promptFile} onChange={(event) => onPatch({ promptFile: event.target.value })} /></FormField>}
        {(draft.promptMode === 'text' || draft.promptMode === 'raw') && <FormField label={draft.promptMode === 'raw' ? '原始请求 JSON' : 'Prompt 文本'} required description={draft.promptMode === 'raw' ? '用于覆盖完整请求体，适合需要自定义 messages、tools 或其他协议字段的场景。' : '直接作为请求 Prompt 内容，适合快速创建固定输入的压测任务。'}><Textarea value={draft.promptText} onChange={(event) => onPatch({ promptText: event.target.value })} className="min-h-72" /></FormField>}
      </div>
    </section>
  )
}

function StandardConfigForm({ draft, onPatch }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void }) {
  return (
    <section className="border-b pb-6">
      <div className="mb-5 flex items-center gap-2 text-sm font-medium"><Gauge className="size-4" />标准模式输入参数</div>
      <div className="grid gap-4">
        <FormField label="并发数" required description="同一时间最多运行的请求数量；用于观察固定压力下的性能表现。"><Input type="number" value={draft.concurrency} onChange={(event) => onPatch({ concurrency: toNumber(event.target.value) })} /></FormField>
        <FormField label="请求总数" required description="整个任务计划发送的请求数量，请求全部完成后任务结束。"><Input type="number" value={draft.requests} onChange={(event) => onPatch({ requests: toNumber(event.target.value) })} /></FormField>
      </div>
    </section>
  )
}

function RequestOptionsForm({ draft, onPatch }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void }) {
  return (
    <section>
      <div className="mb-5 flex items-center gap-2 text-sm font-medium"><Settings2 className="size-4" />通用请求选项</div>
      <div className="grid gap-4">
        <FormField label="请求超时" required description="单个请求允许的最长耗时，可使用 30s、1m 等时间写法。"><Input value={draft.timeout} onChange={(event) => onPatch({ timeout: event.target.value })} /></FormField>
        <BooleanToggle label="流式响应" description="开启后按流式接口统计首 token 时间、吞吐等指标。" value={draft.stream} onChange={(value) => onPatch({ stream: value })} />
        <BooleanToggle label="启用思考" description="适用于支持思考字段的模型；关闭时不额外请求 thinking 输出。" value={draft.thinking} onChange={(value) => onPatch({ thinking: value })} />
        <BooleanToggle label="生成报告" description="任务结束后输出可归档的报告数据，便于对比多次运行。" value={draft.report} onChange={(value) => onPatch({ report: value })} />
        <BooleanToggle label="记录日志" description="开启后保留更详细的请求过程信息，排查问题时更有用。" value={draft.log} onChange={(value) => onPatch({ log: value })} />
      </div>
    </section>
  )
}

function TurboConfigForm({ draft, onPatch }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void }) {
  return (
    <section className="border-b pb-6">
      <div className="mb-5 flex items-center gap-2 text-sm font-medium"><Zap className="size-4" />Turbo 爬坡配置</div>
      <div className="grid gap-4">
        <FormField label="起始并发" required description="爬坡测试的第一档并发数，从这个压力开始逐级增加。"><Input type="number" value={draft.turboInitConcurrency} onChange={(event) => onPatch({ turboInitConcurrency: toNumber(event.target.value) })} /></FormField>
        <FormField label="最大并发" required description="爬坡上限；达到该并发或触发停止条件后不再继续升档。"><Input type="number" value={draft.turboMaxConcurrency} onChange={(event) => onPatch({ turboMaxConcurrency: toNumber(event.target.value) })} /></FormField>
        <FormField label="每级递增" required description="每完成一级后增加的并发数量，例如 4 表示 4、8、12 这样递增。"><Input type="number" value={draft.turboStepSize} onChange={(event) => onPatch({ turboStepSize: toNumber(event.target.value) })} /></FormField>
        <FormField label="每级请求数" required description="每个并发级别发送的请求数，用于评估该压力档是否稳定。"><Input type="number" value={draft.turboLevelRequests} onChange={(event) => onPatch({ turboLevelRequests: toNumber(event.target.value) })} /></FormField>
        <FormField label="最低成功率" required description="低于该成功率时认为当前压力档不可接受；1 表示 100%。"><Input type="number" step="0.01" value={draft.turboMinSuccessRate} onChange={(event) => onPatch({ turboMinSuccessRate: Number.parseFloat(event.target.value) || 0 })} /></FormField>
        <FormField label="最大延迟" required description="当前压力档允许的最大平均延迟，可使用 1s、800ms 等时间写法。"><Input value={draft.turboMaxLatency} onChange={(event) => onPatch({ turboMaxLatency: event.target.value })} /></FormField>
      </div>
    </section>
  )
}

function CreateStepReview({ draft }: { draft: TaskDraft }) {
  const task = taskFromDraft('preview', draft)
  return (
    <div className="space-y-6">
      <DraftTaskPreview task={task} />
      <section>
        <div className="mb-2 flex items-center gap-2 text-sm font-medium"><FileJson className="size-4" />将要提交的 Input</div>
        <p className="mb-4 text-xs leading-5 text-muted-foreground">这里展示的是最终会进入任务配置的真实结构，因此保留实际字段名，便于与 CLI 或配置文件对应。</p>
        <CodeBlock label="TaskConfig.Input" value={JSON.stringify(inputJsonFromDraft(draft), null, 2)} icon={<FileJson className="size-3.5" />} />
      </section>
    </div>
  )
}

function TaskRunHistory({ runs, selectedRun, onChooseRun }: { runs: RunRecord[]; selectedRun?: RunRecord; onChooseRun: (run: RunRecord) => void }) {
  return (
    <Card className="min-h-0 rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40">
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
        <ExecutionTrend runs={runs} />
        <div className="overflow-x-auto rounded-2xl border bg-background/70">
          <Table className="min-w-[940px]">
            <TableHeader>
              <TableRow>
                <TableHead>开始时间</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>完成</TableHead>
                <TableHead>成功率</TableHead>
                <TableHead>错误率</TableHead>
                <TableHead>总耗时</TableHead>
                <TableHead>TTFT</TableHead>
                <TableHead>TPS</TableHead>
                <TableHead>RPM</TableHead>
                <TableHead>缓存</TableHead>
                <TableHead>Token</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {runs.map((run) => (
                <TableRow key={run.id} onClick={() => onChooseRun(run)} className={cn('cursor-pointer transition-colors', selectedRun?.id === run.id && 'bg-accent')}>
                  <TableCell>
                    <div className="font-medium">{run.startedAt}</div>
                    <div className="mt-1 text-xs text-muted-foreground">耗时 {run.duration}</div>
                  </TableCell>
                  <TableCell><StatusBadge status={run.status} /></TableCell>
                  <TableCell className="tabular-nums">{run.success + run.failed}/{run.requests}</TableCell>
                  <TableCell><RunRate value={run.successRate} /></TableCell>
                  <TableCell className={cn('tabular-nums', run.errorRate > 5 ? 'text-red-600' : 'text-muted-foreground')}>{run.errorRate}%</TableCell>
                  <TableCell>{run.avgTotalTime}</TableCell>
                  <TableCell>{run.avgTTFT}</TableCell>
                  <TableCell className="tabular-nums">{run.avgTPS}</TableCell>
                  <TableCell className="tabular-nums">{run.rpm}</TableCell>
                  <TableCell className="tabular-nums">{run.cacheHitRate}%</TableCell>
                  <TableCell className="text-xs text-muted-foreground">in {run.promptTokens} / out {run.outputTokens}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}

function DraftTaskPreview({ task }: { task: Task }) {
  return (
    <div className="space-y-4 rounded-3xl border bg-background/70 p-4 shadow-xs">
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-muted text-muted-foreground"><ModeIcon mode={task.mode} className="size-5" /></div>
        <div className="min-w-0">
          <div className="font-medium">{task.name}</div>
          <div className="mt-1 text-sm leading-5 text-muted-foreground">{task.description}</div>
        </div>
      </div>
      <div className="grid gap-2 text-sm sm:grid-cols-2">
        <KeyValue label="模型名称" value={task.model} />
        <KeyValue label="协议类型" value={task.protocol} />
        <KeyValue label={task.mode === 'integrity' ? '测试集' : 'Prompt'} value={task.mode === 'integrity' ? task.integrity?.suite ?? '-' : `${promptModeLabel[task.promptSpec!.mode]} · ${task.promptSpec!.summary}`} />
        <KeyValue label="请求地址" value={task.endpoint} />
      </div>
      <TaskModeConfigPreview task={task} />
    </div>
  )
}

function TopStat({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="inline-flex items-center gap-1.5 rounded-full border bg-background/70 px-3 py-1.5 shadow-xs">
      <span className="text-muted-foreground">{icon}</span>
      <span className="font-semibold text-foreground">{value}</span>
      <span>{label}</span>
    </div>
  )
}

function TaskOverview({ task, onCreate }: { task: Task; onCreate: (task: Task) => void }) {
  return (
    <Card className="rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40">
      <CardHeader className="p-4 sm:p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Badge className="gap-1.5"><ModeIcon mode={task.mode} className="size-3.5" />{modeLabel[task.mode]}</Badge>
              <Badge variant="outline" className="gap-1.5"><Network className="size-3.5" />{task.protocol}</Badge>
            </div>
            <CardTitle className="text-xl sm:text-2xl">{task.name}</CardTitle>
            <CardDescription className="mt-2 max-w-2xl leading-6">{task.description}</CardDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2 lg:justify-end">
            {task.tags.map((tag) => <Badge key={tag} variant="outline">{tag}</Badge>)}
            <CreateTaskSheet onCreate={onCreate} sourceTask={task} variant="copy" />
            <TaskConfigSheet task={task} />
          </div>
        </div>
      </CardHeader>
      <CardContent className="grid gap-2 p-4 pt-0 text-sm sm:grid-cols-2 lg:grid-cols-4 sm:p-5 sm:pt-0">
        <KeyValue label="模型名称" value={task.model} />
        <KeyValue label="协议类型" value={task.protocol} />
        <KeyValue label={task.mode === 'turbo' ? '每级请求 × 级数' : '请求总数'} value={task.requests.toString()} />
        <KeyValue label={task.mode === 'turbo' ? '起始并发' : '并发数'} value={task.concurrency.toString()} />
      </CardContent>
    </Card>
  )
}

function TaskConfigSheet({ task }: { task: Task }) {
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="outline" size="sm" className="rounded-full">
          <Settings2 className="size-3.5" />请求配置
        </Button>
      </SheetTrigger>
      <SheetContent className="!w-[min(96vw,1040px)] !max-w-none overflow-y-auto">
        <SheetHeader className="border-b bg-muted/30 px-6 py-5">
          <SheetTitle className="flex items-center gap-2"><Settings2 className="size-4" />请求配置</SheetTitle>
          <SheetDescription>这些参数用于定位任务输入，不占用主执行视图。</SheetDescription>
        </SheetHeader>
        <div className="space-y-5 px-6">
          <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
            <div className="mb-3 flex items-center gap-2 text-sm font-medium"><Route className="size-4" />请求目标</div>
            <div className="grid gap-2 text-sm sm:grid-cols-2">
              <KeyValue label="请求地址" value={task.endpoint} />
              <KeyValue label="模型名称" value={task.model} />
              <KeyValue label="协议类型" value={task.protocol} />
              <KeyValue label="任务类型" value={modeLabel[task.mode]} />
            </div>
          </div>
          <TaskModeDetails task={task} />
        </div>
      </SheetContent>
    </Sheet>
  )
}

function TaskModeDetails({ task }: { task: Task }) {
  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
      <ModeSpecificPanel task={task} />
      {task.mode !== 'integrity' && <PromptPanel task={task} />}
    </div>
  )
}

function ModeSpecificPanel({ task }: { task: Task }) {
  if (task.mode === 'turbo') {
    const levels = task.turbo?.levels ?? []
    return (
      <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-sm font-medium"><Zap className="size-4" />Turbo 配置</div>
          <Badge variant="outline">{levels.length} 个级别</Badge>
        </div>
        <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
          <KeyValue label="起始并发" value={(task.turbo?.initConcurrency ?? '-').toString()} />
          <KeyValue label="最大并发" value={(task.turbo?.maxConcurrency ?? '-').toString()} />
          <KeyValue label="每级递增" value={(task.turbo?.stepSize ?? '-').toString()} />
          <KeyValue label="每级请求数" value={(task.turbo?.levelRequests ?? '-').toString()} />
          <KeyValue label="最低成功率" value={(task.turbo?.minSuccessRate ?? '-').toString()} />
          <KeyValue label="最大延迟" value={task.turbo?.maxLatency ?? '-'} />
        </div>
        <div className="mt-4 rounded-2xl bg-muted/50 px-4 py-3 text-xs leading-5 text-muted-foreground">
          并发级别：{levels.length > 0 ? levels.join(' → ') : '-'}
        </div>
      </div>
    )
  }

  if (task.mode === 'integrity') {
    const cases = task.integrity?.cases ?? []
    return (
      <div className="xl:col-span-2 rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2 text-sm font-medium"><ClipboardList className="size-4" />完整性测试集</div>
          <Badge variant="outline">{cases.length} 个用例</Badge>
        </div>
        <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4">
          <KeyValue label="测试集名称" value={task.integrity?.suite ?? '-'} />
          <KeyValue label="规则文件" value={task.integrity?.ruleFiles.join(', ') ?? '-'} />
          <InlineSwitch label="失败时立即停止" enabled={task.integrity?.failFast ?? false} />
          <KeyValue label="单个用例超时" value={task.integrity?.caseTimeout ?? '-'} />
        </div>
        <div className="mt-4 overflow-hidden rounded-2xl border bg-background/70">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用例</TableHead>
                <TableHead>能力</TableHead>
                <TableHead>断言</TableHead>
                <TableHead>必需</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cases.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>
                    <div className="font-medium">{item.name}</div>
                    <div className="mt-1 max-w-[360px] truncate text-xs text-muted-foreground" title={item.prompt}>{item.prompt}</div>
                  </TableCell>
                  <TableCell>{item.capability}</TableCell>
                  <TableCell>{item.assertions.length}</TableCell>
                  <TableCell>{item.required ? '是' : '否'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
      <div className="mb-4 flex items-center gap-2 text-sm font-medium"><Gauge className="size-4" />标准模式输入参数</div>
      <div className="grid gap-4 lg:grid-cols-[220px_minmax(0,1fr)]">
        <div className="rounded-2xl bg-muted/50 p-4">
          <div className="text-xs text-muted-foreground">执行队列</div>
          <div className="mt-3 grid grid-cols-2 gap-4">
            <NumberStat label="并发数" value={task.concurrency.toString()} />
            <NumberStat label="请求总数" value={task.requests.toString()} />
          </div>
          <p className="mt-4 text-xs leading-5 text-muted-foreground">按请求总数生成请求，最多同时运行指定并发数。</p>
        </div>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
          <KeyValue label="请求超时" value={task.standard?.timeout ?? '-'} />
          <InlineSwitch label="流式响应" enabled={task.standard?.stream ?? false} />
          <InlineSwitch label="启用思考" enabled={task.standard?.thinking ?? false} />
          <InlineSwitch label="生成报告" enabled={task.standard?.report ?? false} />
          <InlineSwitch label="记录日志" enabled={task.standard?.log ?? false} />
          <KeyValue label="Prompt 来源" value={task.promptSpec ? promptModeLabel[task.promptSpec.mode] : '-'} />
        </div>
      </div>
    </div>
  )
}

function PromptPanel({ task }: { task: Task }) {
  const prompt = task.promptSpec
  if (!prompt) return null

  return (
    <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2 text-sm font-medium"><FileJson className="size-4" />Prompt 输入</div>
          <div className="mt-1 text-xs text-muted-foreground">标准和 Turbo 使用 Prompt；完整性校验使用测试集 case。</div>
        </div>
        <Badge variant="outline">{promptModeLabel[prompt.mode]}</Badge>
      </div>
      <div className="mt-4 rounded-2xl bg-muted/50 px-4 py-3 text-sm text-muted-foreground">{prompt.summary}</div>
      <Tabs defaultValue="summary" className="mt-4">
        <TabsList className="grid w-full grid-cols-2 sm:w-72">
          <TabsTrigger value="summary">结构</TabsTrigger>
          <TabsTrigger value="content">内容</TabsTrigger>
        </TabsList>
        <TabsContent value="summary" className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
          <KeyValue label="配置项" value={promptFieldLabel[prompt.label] ?? prompt.label} />
          <KeyValue label="输入方式" value={promptModeLabel[prompt.mode]} />
        </TabsContent>
        <TabsContent value="content" className="mt-4">
          <CodeBlock label="Prompt 内容" value={prompt.content} icon={<FileJson className="size-3.5" />} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-xl bg-muted/50 px-3 py-2">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      <div className="truncate text-sm font-medium" title={value}>{value}</div>
    </div>
  )
}

function FormField({ label, description, required, children }: { label: string; description?: string; required?: boolean; children: React.ReactNode }) {
  return (
    <div className="grid gap-1.5 text-sm">
      <Label className="flex items-center gap-1">{label}{required && <span className="text-destructive">*</span>}</Label>
      {description && <p className="text-xs leading-5 text-muted-foreground">{description}</p>}
      {children}
    </div>
  )
}

function OptionPicker<T extends string>({ value, options, onChange }: { value: T; options: readonly T[]; onChange: (value: T) => void }) {
  return (
    <Select value={value} onValueChange={(next) => onChange(next as T)}>
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => <SelectItem key={option} value={option}>{optionDisplayName(option)}</SelectItem>)}
      </SelectContent>
    </Select>
  )
}

function optionDisplayName(option: string) {
  return promptModeLabel[option as PromptMode] ?? protocolLabel[option] ?? option
}

function BooleanToggle({ label, description, value, onChange }: { label: string; description?: string; value: boolean; onChange: (value: boolean) => void }) {
  const id = useId()

  return (
    <div className="flex min-h-12 items-center justify-between gap-4 rounded-2xl border bg-background px-3 py-2 text-sm">
      <div className="grid gap-0.5">
        <Label htmlFor={id} className="text-sm">{label}</Label>
        {description && <p className="text-xs leading-5 text-muted-foreground">{description}</p>}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="text-xs text-muted-foreground">{value ? '开启' : '关闭'}</span>
        <Switch id={id} checked={value} onCheckedChange={onChange} />
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

function InlineSwitch({ label, enabled }: { label: string; enabled: boolean }) {
  return (
    <div className="flex items-center justify-between rounded-2xl bg-muted/50 px-4 py-3 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn('rounded-full px-2 py-0.5 text-xs font-medium', enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-900 dark:text-zinc-300')}>{enabled ? '开启' : '关闭'}</span>
    </div>
  )
}

function ExecutionTrend({ runs }: { runs: RunRecord[] }) {
  if (runs.length === 0) return null

  const latestRun = runs[0]
  const samples = getRunRequests(latestRun.id).sort((a, b) => a.index - b.index)
  const completed = latestRun.success + latestRun.failed
  const maxTps = Math.max(...samples.map((request) => request.tps), 1)
  const maxLatency = Math.max(...samples.map((request) => parseDurationMs(request.latency)), 1)
  const maxTTFT = Math.max(...samples.map((request) => parseDurationMs(request.ttft)), 1)
  const maxOutputTokens = Math.max(...samples.map((request) => request.completionTokens), 1)
  const maxNetwork = Math.max(...samples.map((request) => requestNetworkMs(request)), 1)
  const cacheRates = samples.map((request) => request.promptTokens > 0 ? Math.round((request.cachedTokens / request.promptTokens) * 100) : 0)
  const maxCache = Math.max(...cacheRates, 1)
  const failedSamples = samples.filter((request) => request.status === 'failed').length

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
      <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div>
            <div className="flex items-center gap-2 text-sm font-medium"><TrendingUp className="size-4" />最新执行样本曲线</div>
            <div className="mt-1 text-xs text-muted-foreground">{latestRun.id} 内部请求的吞吐、延迟、TTFT、Token、网络和缓存变化。</div>
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
          <NumberStat compact label="完成" value={`${completed}/${latestRun.requests}`} />
          <NumberStat compact label="成功率" value={`${latestRun.successRate}%`} />
          <NumberStat compact label="错误率" value={`${latestRun.errorRate}%`} />
          <NumberStat compact label="平均耗时" value={latestRun.avgTotalTime} />
          <NumberStat compact label="TTFT" value={latestRun.avgTTFT} />
          <NumberStat compact label="TPS" value={latestRun.avgTPS.toString()} />
          <NumberStat compact label="RPM" value={latestRun.rpm.toString()} />
          <NumberStat compact label="缓存" value={`${latestRun.cacheHitRate}%`} />
        </div>
      </div>
    </div>
  )
}

function SampleLineChart({ samples, cacheRates, maxTps, maxLatency, maxTTFT, maxOutputTokens, maxNetwork, maxCache }: { samples: RequestDetail[]; cacheRates: number[]; maxTps: number; maxLatency: number; maxTTFT: number; maxOutputTokens: number; maxNetwork: number; maxCache: number }) {
  const labels = samples.map((request) => `#${request.index}`)
  const datasets = [
    chartDataset('TPS', samples.map((request) => normalizeChartValue(request.tps, maxTps)), '#18181b', 3, true),
    chartDataset('总耗时', samples.map((request) => normalizeChartValue(parseDurationMs(request.latency), maxLatency)), '#71717a'),
    chartDataset('TTFT', samples.map((request) => normalizeChartValue(parseDurationMs(request.ttft), maxTTFT)), '#0ea5e9'),
    chartDataset('输出 Token', samples.map((request) => normalizeChartValue(request.completionTokens, maxOutputTokens)), '#8b5cf6'),
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
  return parseDurationMs(request.dns) + parseDurationMs(request.connect) + parseDurationMs(request.tls)
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

function RunRate({ value }: { value: number }) {
  return (
    <div className="flex min-w-28 items-center gap-2">
      <div className="h-1.5 flex-1 rounded-full bg-muted"><div className="h-full rounded-full bg-primary" style={{ width: `${value}%` }} /></div>
      <span className="w-10 text-right text-sm tabular-nums">{value}%</span>
    </div>
  )
}

function RunDetail({ run, requests, selectedRequest, onSelectRequest }: { run?: RunRecord; requests: RequestDetail[]; selectedRequest?: RequestDetail; onSelectRequest: (id: string) => void }) {
  if (!run) {
    return <Card className="rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40"><CardContent className="flex min-h-60 items-center justify-center p-6 text-sm text-muted-foreground"><Clock3 className="mr-2 size-4" />当前任务暂无执行记录。</CardContent></Card>
  }

  return (
    <Card className="rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40">
      <CardHeader className="p-4 sm:p-5">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <CardTitle className="flex items-center gap-2 text-base"><FileJson className="size-4" />执行详情</CardTitle>
            <CardDescription className="truncate">{run.id} · {run.startedAt} · {run.summary}</CardDescription>
          </div>
          <StatusBadge status={run.status} />
        </div>
      </CardHeader>
      <CardContent className="space-y-5 p-4 pt-0 sm:p-5 sm:pt-0">
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <KpiCard icon={<CheckCircle2 className="size-4" />} label="成功率" value={`${run.successRate}%`} sub={`${run.success} 成功 / ${run.failed} 失败`} />
          <KpiCard icon={<Gauge className="size-4" />} label="平均总耗时" value={run.avgTotalTime} sub={`TTFT ${run.avgTTFT} · TPOT ${run.avgTPOT}`} />
          <KpiCard icon={<TrendingUp className="size-4" />} label="平均 TPS" value={run.avgTPS.toString()} sub={`RPM ${run.rpm} · TPM ${run.tpm}`} />
          <KpiCard icon={<Database className="size-4" />} label="缓存命中" value={`${run.cacheHitRate}%`} sub={`${run.cachedTokens} cached tokens`} />
        </div>

        <div className="rounded-2xl border bg-background/70 p-4">
          <div className="mb-2 flex items-center justify-between text-sm">
            <span className="font-medium">完成进度</span>
            <span className="text-muted-foreground">{run.success + run.failed} / {run.requests}</span>
          </div>
          <Progress value={Math.round(((run.success + run.failed) / run.requests) * 100)} />
        </div>

        <Tabs defaultValue="overview">
          <TabsList className="grid w-full grid-cols-2 lg:w-[320px]">
            <TabsTrigger value="overview">指标概览</TabsTrigger>
            <TabsTrigger value="requests">请求样本</TabsTrigger>
          </TabsList>
          <TabsContent value="overview" className="mt-4 grid gap-3 lg:grid-cols-3">
            <CompactMetricList title="延迟分布" icon={<Clock3 className="size-4" />} items={[
              ['总耗时', `${run.minTotalTime} / ${run.avgTotalTime} / ${run.maxTotalTime}`],
              ['TTFT', `${run.minTTFT} / ${run.avgTTFT} / ${run.maxTTFT}`],
              ['TPOT', run.avgTPOT],
              ['标准差', `total ${run.stddevTotalTime} · ttft ${run.stddevTTFT}`],
            ]} />
            <CompactMetricList title="吞吐与 Token" icon={<TrendingUp className="size-4" />} items={[
              ['TPS', `${run.minTPS} / ${run.avgTPS} / ${run.maxTPS}`],
              ['总吞吐 TPS', run.totalThroughputTPS.toString()],
              ['Token', `in ${run.promptTokens} · out ${run.outputTokens}`],
              ['thinking', run.thinkingTokens.toString()],
            ]} />
            <CompactMetricList title="网络" icon={<Network className="size-4" />} items={[
              ['DNS', run.avgDNS],
              ['Connect', run.avgConnect],
              ['TLS', run.avgTLS],
              ['Target IP', run.targetIP],
            ]} />
          </TabsContent>
          <TabsContent value="requests" className="mt-4 grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
            <div className="space-y-2">
              {requests.map((request) => (
                <button key={request.id} type="button" onClick={() => onSelectRequest(request.id)} className={cn('flex w-full items-center justify-between rounded-2xl border bg-background/70 p-3 text-left text-sm hover:bg-accent', selectedRequest?.id === request.id && 'border-primary bg-accent')}>
                  <span className="font-medium">#{request.index} · {request.id}</span>
                  <span className="flex items-center gap-2 text-muted-foreground">{request.status === 'failed' ? <XCircle className="size-4 text-red-500" /> : <CheckCircle2 className="size-4 text-emerald-500" />}{request.latency}</span>
                </button>
              ))}
            </div>
            {selectedRequest && <RequestPanel request={selectedRequest} />}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

function KpiCard({ icon, label, value, sub }: { icon: React.ReactNode; label: string; value: string; sub: string }) {
  return (
    <div className="rounded-2xl border bg-background/75 p-4 shadow-xs ring-1 ring-border/20">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">{icon}{label}</div>
      <div className="mt-2 text-2xl font-semibold tracking-tight tabular-nums">{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground" title={sub}>{sub}</div>
    </div>
  )
}

function CompactMetricList({ title, icon, items }: { title: string; icon: React.ReactNode; items: Array<[string, string]> }) {
  return (
    <div className="rounded-2xl border bg-background/75 p-4 shadow-xs ring-1 ring-border/20">
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

function RequestPanel({ request }: { request: RequestDetail }) {
  return (
    <div className="rounded-2xl border bg-background p-4">
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="font-medium">请求详情 #{request.index}</div>
        {request.error ? <Badge className="bg-red-600"><AlertTriangle className="size-3" />{request.error}</Badge> : <Badge className="bg-emerald-600">OK</Badge>}
      </div>
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <CompactMetricList title="本次指标" icon={<Gauge className="size-4" />} items={[
          ['延迟', `${request.latency} · TTFT ${request.ttft}`],
          ['TPS', request.tps.toString()],
          ['Token', `in ${request.promptTokens} · out ${request.completionTokens} · cached ${request.cachedTokens}`],
          ['网络', `DNS ${request.dns} · Conn ${request.connect} · TLS ${request.tls}`],
          ['Target IP', request.targetIP],
        ]} />
        <div className="space-y-3">
          <CodeBlock label="请求内容" value={request.request} icon={<Network className="size-3.5" />} />
          <CodeBlock label="响应内容" value={request.response} icon={<FileJson className="size-3.5" />} />
        </div>
      </div>
    </div>
  )
}

function CodeBlock({ label, value, icon }: { label: string; value: string; icon?: React.ReactNode }) {
  return (
    <div>
      <div className="mb-1 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">{icon}{label}</div>
      <pre className="max-h-36 overflow-auto rounded-xl border bg-muted/70 p-3 text-xs leading-5"><code>{value}</code></pre>
    </div>
  )
}

function StatusBadge({ status }: { status: RunStatus }) {
  const Icon = status === 'running' ? Activity : status === 'completed' ? CheckCircle2 : status === 'failed' ? XCircle : Clock3
  return <Badge variant="outline" className={cn('gap-1', statusStyle[status])}><Icon className="size-3" />{statusLabel[status]}</Badge>
}

function ModeIcon({ mode, className }: { mode: TaskMode; className?: string }) {
  const Icon = mode === 'standard' ? Gauge : mode === 'turbo' ? Zap : ShieldCheck
  return <Icon className={className} />
}

function TaskModeConfigPreview({ task }: { task: Task }) {
  return (
    <div>
      <div className="mb-3 flex items-center gap-2 text-sm font-medium"><Settings2 className="size-4" />类型配置</div>
      <ModeSpecificPanel task={task} />
    </div>
  )
}

const createModeHint: Record<TaskMode, string> = {
  standard: '固定并发与请求总数',
  turbo: '起始并发、最大并发、递增步长',
  integrity: '测试集与规则文件',
}

const draftName: Record<TaskMode, string> = {
  standard: '新建标准压测任务',
  turbo: '新建 Turbo 爬坡任务',
  integrity: '新建完整性校验任务',
}

const draftDescription: Record<TaskMode, string> = {
  standard: '固定并发性能基线任务。',
  turbo: '并发爬坡压力任务。',
  integrity: '协议完整性校验任务。',
}

const draftTags: Record<TaskMode, string[]> = {
  standard: ['standard', 'mock'],
  turbo: ['turbo', 'mock'],
  integrity: ['integrity', 'mock'],
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

const protocolOptions = ['openai-completions', 'openai-responses', 'anthropic-messages'] as const
const promptModeOptions = ['text', 'file', 'generated', 'raw'] as const

function makeInitialDraft(mode: TaskMode): TaskDraft {
  return {
    mode,
    name: draftName[mode],
    protocol: draftProtocol[mode],
    endpoint: defaultEndpoint(draftProtocol[mode]),
    model: draftModel[mode],
    concurrency: mode === 'integrity' ? 1 : mode === 'turbo' ? 4 : 8,
    requests: mode === 'integrity' ? 3 : mode === 'turbo' ? 360 : 120,
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
    integritySuite: defaultSuite(draftProtocol[mode]),
    integrityRuleFiles: 'data/integrity/openai-completions.json',
    integrityFailFast: true,
    integrityCaseTimeout: '30000',
  }
}

function draftFromTask(task: Task): TaskDraft {
  const draft = makeInitialDraft(task.mode)
  const promptMode = task.promptSpec?.mode ?? draft.promptMode
  return {
    ...draft,
    mode: task.mode,
    name: `${task.name} 副本`,
    protocol: task.protocol,
    endpoint: task.endpoint,
    model: task.model,
    concurrency: task.concurrency,
    requests: task.requests,
    promptMode,
    promptText: promptMode === 'generated' ? draft.promptText : task.promptSpec?.content ?? task.prompt ?? draft.promptText,
    promptFile: promptMode === 'file' ? task.promptSpec?.content ?? '' : '',
    promptLength: promptMode === 'generated' ? numberFromText(task.promptSpec?.content ?? '', draft.promptLength) : draft.promptLength,
    timeout: task.standard?.timeout ?? draft.timeout,
    stream: task.standard?.stream ?? true,
    thinking: task.standard?.thinking ?? false,
    report: task.standard?.report ?? true,
    log: task.standard?.log ?? false,
    turboInitConcurrency: task.turbo?.initConcurrency ?? task.turbo?.levels[0] ?? draft.turboInitConcurrency,
    turboMaxConcurrency: task.turbo?.maxConcurrency ?? draft.turboMaxConcurrency,
    turboStepSize: task.turbo?.stepSize ?? inferStepSize(task.turbo?.levels) ?? draft.turboStepSize,
    turboLevelRequests: task.turbo?.levelRequests ?? (task.turbo ? Math.max(1, Math.round(task.requests / Math.max(task.turbo.levels.length, 1))) : draft.turboLevelRequests),
    turboMinSuccessRate: task.turbo?.minSuccessRate ?? draft.turboMinSuccessRate,
    turboMaxLatency: task.turbo?.maxLatency ?? draft.turboMaxLatency,
    integritySuite: task.integrity?.suite ?? draft.integritySuite,
    integrityRuleFiles: task.integrity?.ruleFiles.join('\n') ?? draft.integrityRuleFiles,
    integrityFailFast: task.integrity?.failFast ?? draft.integrityFailFast,
    integrityCaseTimeout: task.integrity?.caseTimeout ? String(durationToMs(task.integrity.caseTimeout)) : draft.integrityCaseTimeout,
  }
}

function taskFromDraft(id: string, draft: TaskDraft): Task {
  const base = {
    id,
    name: draft.name.trim() || draftName[draft.mode],
    description: draftDescription[draft.mode],
    mode: draft.mode,
    protocol: draft.protocol,
    model: draft.model.trim(),
    endpoint: draft.endpoint.trim(),
    concurrency: draft.mode === 'turbo' ? draft.turboInitConcurrency : draft.concurrency,
    requests: draft.mode === 'turbo' ? draft.turboLevelRequests * turboLevelsFromDraft(draft).length : draft.requests,
    updatedAt: '刚刚',
    tags: draftTags[draft.mode],
  }

  if (draft.mode === 'integrity') return { ...base, integrity: { suite: draft.integritySuite.trim(), ruleFiles: parseList(draft.integrityRuleFiles), failFast: draft.integrityFailFast, caseTimeout: `${durationToMs(draft.integrityCaseTimeout)}ms`, cases: draftIntegrityCases(draft) } }

  const promptSpec = promptSpecFromDraft(draft)
  if (draft.mode === 'turbo') return { ...base, prompt: promptSpec.content, promptSpec, turbo: { levels: turboLevelsFromDraft(draft), initConcurrency: draft.turboInitConcurrency, maxConcurrency: draft.turboMaxConcurrency, stepSize: draft.turboStepSize, levelRequests: draft.turboLevelRequests, minSuccessRate: draft.turboMinSuccessRate, maxLatency: draft.turboMaxLatency.trim() } }
  return { ...base, prompt: promptSpec.content, promptSpec, standard: { timeout: draft.timeout.trim(), stream: draft.stream, thinking: draft.thinking, report: draft.report, log: draft.log } }
}

function inputJsonFromDraft(draft: TaskDraft) {
  const input = {
    mode: draft.mode,
    protocol: draft.protocol,
    endpoint_url: draft.endpoint.trim(),
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
        rule_files: parseList(draft.integrityRuleFiles),
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

function promptInputFromDraft(draft: TaskDraft) {
  if (draft.promptMode === 'file') return { prompt_mode: 'file', prompt_file: draft.promptFile.trim() }
  if (draft.promptMode === 'generated') return { prompt_mode: 'generated', prompt_length: draft.promptLength }
  return { prompt_mode: draft.promptMode, prompt_text: draft.promptText }
}

function promptSpecFromDraft(draft: TaskDraft) {
  if (draft.promptMode === 'file') return { mode: 'file' as const, label: 'prompt_file', summary: `从文件读取 Prompt：${draft.promptFile}`, content: draft.promptFile }
  if (draft.promptMode === 'generated') return { mode: 'generated' as const, label: 'prompt_length', summary: `按长度生成 ${draft.promptLength} token Prompt。`, content: `Prompt 长度：${draft.promptLength}` }
  if (draft.promptMode === 'raw') return { mode: 'raw' as const, label: 'prompt_text', summary: '原始 JSON 请求体。', content: draft.promptText }
  return { mode: 'text' as const, label: 'prompt_text', summary: '直接使用文本 Prompt。', content: draft.promptText }
}

function defaultEndpoint(protocol: string) {
  if (protocol === 'openai-responses') return 'https://api.openai.com/v1/responses'
  if (protocol === 'anthropic-messages') return 'https://api.anthropic.com/v1/messages'
  return 'https://api.openai.com/v1/chat/completions'
}

function defaultSuite(protocol: string) {
  if (protocol === 'openai-responses') return 'openai-responses-smoke'
  if (protocol === 'anthropic-messages') return 'anthropic-messages-smoke'
  return 'openai-completions-smoke'
}

function nextStepLabel(step: number) {
  if (step === 0) return '填写基础信息'
  if (step === 1) return '填写类型配置'
  if (step === 2) return '检查并确认'
  return '下一步'
}

function createStepHint(draft: TaskDraft, step: number) {
  if (step === 1) return '请先填写任务名称、协议、模型名称和请求地址。'
  if (step === 2 && draft.mode === 'integrity') return '请补充测试集名称、规则文件和单个用例超时。'
  if (step === 2 && draft.mode === 'turbo') return '请确认 Prompt、并发爬坡参数、请求超时和停止条件均已填写。'
  if (step === 2) return '请确认 Prompt、并发数、请求总数和请求超时均已填写。'
  return '请先完成当前步骤。'
}

function isStepValid(draft: TaskDraft, step: number): boolean {
  if (step === 0) return Boolean(draft.mode)
  if (step === 1) return isBasicConfigValid(draft)
  if (step === 2) return isModeConfigValid(draft)
  return isDraftValid(draft)
}

function isDraftValid(draft: TaskDraft): boolean {
  return isBasicConfigValid(draft) && isModeConfigValid(draft)
}

function isBasicConfigValid(draft: TaskDraft): boolean {
  return Boolean(draft.name.trim() && draft.protocol.trim() && draft.endpoint.trim() && draft.model.trim())
}

function isModeConfigValid(draft: TaskDraft): boolean {
  if (draft.mode === 'integrity') return Boolean(draft.integritySuite.trim() && draft.integrityRuleFiles.trim() && durationToMs(draft.integrityCaseTimeout) > 0)
  if (draft.mode === 'turbo' && (draft.turboInitConcurrency <= 0 || draft.turboMaxConcurrency <= 0 || draft.turboStepSize <= 0 || draft.turboLevelRequests <= 0 || draft.turboMinSuccessRate <= 0 || !draft.turboMaxLatency.trim())) return false
  if (draft.mode === 'standard' && (draft.concurrency <= 0 || draft.requests <= 0)) return false
  if (!draft.timeout.trim()) return false
  if (draft.promptMode === 'file') return Boolean(draft.promptFile.trim())
  if (draft.promptMode === 'generated') return draft.promptLength > 0
  return Boolean(draft.promptText.trim())
}

function parseList(value: string) {
  return value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean)
}

function toNumber(value: string) {
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

function numberFromText(value: string, fallback: number) {
  const match = value.match(/\d+/)
  return match ? Number.parseInt(match[0], 10) : fallback
}

function inferStepSize(levels?: number[]) {
  if (!levels || levels.length < 2) return undefined
  return Math.max(1, levels[1] - levels[0])
}

function turboLevelsFromDraft(draft: TaskDraft) {
  const levels: number[] = []
  const stepSize = Math.max(1, draft.turboStepSize)
  for (let value = Math.max(1, draft.turboInitConcurrency); value <= Math.max(draft.turboInitConcurrency, draft.turboMaxConcurrency); value += stepSize) levels.push(value)
  return levels
}

function draftIntegrityCases(draft: TaskDraft) {
  return [
    { id: 'basic-response-shape', name: '基础响应结构', capability: 'basic_request', prompt: 'Reply with a short greeting.', assertions: ['response.body exists', 'response.body.choices exists'], timeout: draft.integrityCaseTimeout, required: true },
    { id: 'usage-metrics', name: '用量字段', capability: 'usage', prompt: 'Reply with one short sentence.', assertions: ['metrics.total_ms >= 0', 'usage exists'], timeout: draft.integrityCaseTimeout, required: false },
  ]
}

export default App
