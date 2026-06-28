import { useEffect, useId, useMemo, useState } from 'react'
import { Activity, AlertTriangle, BarChart3, CheckCircle2, ClipboardList, Clock3, Copy, Database, FileJson, Gauge, ListChecks, Network, Play, Plus, Route, Settings2, ShieldCheck, TrendingUp, XCircle, Zap } from 'lucide-react'
import { CategoryScale, Chart as ChartJS, Filler, Legend as ChartLegend, LinearScale, LineElement, PointElement, Tooltip as ChartTooltip } from 'chart.js'
import { Line } from 'react-chartjs-2'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { DashboardHeader, TaskSidebarContent, type TaskNavItem } from '@/components/dashboard-layout'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Stepper } from '@/components/ui/stepper'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { createModeHint, createStepHint, defaultEndpoint, draftFromTask, formatDate, formatNumber, formatPercent, inputJsonFromDraft, isDraftValid, isStepValid, makeInitialDraft, maskSecret, modeLabel, nextStepLabel, promptFieldLabel, promptModeLabel, promptModeOptions, promptSpec, protocolLabel, protocolOptions, redactSecretInput, requestKey, taskConcurrency, taskEndpoint, taskFromDraft, taskMode, taskModel, taskProtocol, taskRequests, toNumber, turboLevelsFromConfig, type TaskDraft } from '@/lib/task-utils'
import { createTask as createTaskAPI, getRunRequests, getRunState, listIntegritySuites, listProtocols, listTaskRuns, listTasks, startTaskRun as startTaskRunAPI, type IntegrityAssertionResult, type IntegrityCaseResult, type IntegrityModeState, type IntegrityResult, type IntegritySuite, type PromptMode, type ProtocolMeta, type RequestDetail, type RunState, type RunStatus, type RunSummary, type Task, type TaskConfig, type TaskMode } from './api'

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

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, ChartTooltip, ChartLegend)

function App() {
  const [query, setQuery] = useState('')
  const [taskList, setTaskList] = useState<Task[]>([])
  const [runsByTask, setRunsByTask] = useState<Record<string, RunSummary[]>>({})
  const [statesByRun, setStatesByRun] = useState<Record<string, RunState>>({})
  const [requestsByRun, setRequestsByRun] = useState<Record<string, RequestDetail[]>>({})
  const [protocols, setProtocols] = useState<ProtocolMeta[]>([])
  const [selectedTaskId, setSelectedTaskId] = useState('')
  const [selectedRunId, setSelectedRunId] = useState('')
  const [selectedRequestId, setSelectedRequestId] = useState('')
  const [startingTaskId, setStartingTaskId] = useState('')
  const [taskDrawerOpen, setTaskDrawerOpen] = useState(false)
  const [loadingMessage, setLoadingMessage] = useState('加载任务中...')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    let cancelled = false
    async function loadInitialData() {
      try {
        const [tasks, protocolMetas] = await Promise.all([listTasks(), listProtocols()])
        if (cancelled) return
        setTaskList(tasks)
        setProtocols(protocolMetas)
        setSelectedTaskId((current) => current || tasks[0]?.id || '')
        setErrorMessage('')
      } catch (error) {
        if (!cancelled) setErrorMessage(error instanceof Error ? error.message : '加载任务失败')
      } finally {
        if (!cancelled) setLoadingMessage('')
      }
    }
    loadInitialData()
    return () => { cancelled = true }
  }, [])

  useEffect(() => {
    if (!selectedTaskId) return
    let cancelled = false
    async function loadRuns() {
      try {
        const runList = await listTaskRuns(selectedTaskId)
        if (cancelled) return
        setRunsByTask((current) => ({ ...current, [selectedTaskId]: runList }))
        setSelectedRunId((current) => runList.some((run) => run.run_id === current) ? current : runList[0]?.run_id || '')
        setSelectedRequestId('')
        setErrorMessage('')
      } catch (error) {
        if (!cancelled) setErrorMessage(error instanceof Error ? error.message : '加载执行记录失败')
      }
    }
    loadRuns()
    return () => { cancelled = true }
  }, [selectedTaskId])

  useEffect(() => {
    if (!selectedRunId) return
    let cancelled = false
    async function loadRequests() {
      try {
        const state = await getRunState(selectedRunId).catch(() => undefined)
        const requestList = state?.requests?.length ? state.requests : await getRunRequests(selectedRunId)
        if (cancelled) return
        if (state) setStatesByRun((current) => ({ ...current, [selectedRunId]: state }))
        setRequestsByRun((current) => ({ ...current, [selectedRunId]: requestList }))
        setSelectedRequestId((current) => requestList.some((request) => requestKey(request) === current) ? current : requestKey(requestList[0]))
        setErrorMessage('')
      } catch (error) {
        if (!cancelled) setErrorMessage(error instanceof Error ? error.message : '加载请求样本失败')
      }
    }
    loadRequests()
    return () => { cancelled = true }
  }, [selectedRunId])

  const filteredTasks = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return taskList
    return taskList.filter((task) => [task.name, taskModel(task), taskProtocol(task), taskEndpoint(task), modeLabel[taskMode(task)]].some((text) => text.toLowerCase().includes(keyword)))
  }, [query, taskList])

  const selectedTask = taskList.find((task) => task.id === selectedTaskId) ?? taskList[0]
  const taskRuns = selectedTask ? runsByTask[selectedTask.id] ?? [] : []
  const selectedRun = taskRuns.find((run) => run.run_id === selectedRunId) ?? taskRuns[0]
  const selectedRunState = selectedRun ? statesByRun[selectedRun.run_id] : undefined
  const runRequests = selectedRun ? requestsByRun[selectedRun.run_id] ?? [] : []
  const selectedRequest = runRequests.find((request) => requestKey(request) === selectedRequestId) ?? runRequests[0]
  const standardTaskCount = taskList.filter((task) => taskMode(task) === 'standard').length
  const turboTaskCount = taskList.filter((task) => taskMode(task) === 'turbo').length
  const integrityTaskCount = taskList.filter((task) => taskMode(task) === 'integrity').length
  const protocolOptionsForCreate = protocols.length > 0 ? protocols.map((protocol) => protocol.id) : [...protocolOptions]
  const taskNavItems = useMemo<TaskNavItem[]>(() => filteredTasks.map((task) => {
    const runs = runsByTask[task.id] ?? []
    const latestRun = task.latest_run ?? runs[0]
    return {
      id: task.id,
      name: task.name,
      mode: taskMode(task),
      modeLabel: modeLabel[taskMode(task)],
      model: taskModel(task),
      updatedAt: formatDate(task.updated_at),
      runsCount: runs.length,
      successRate: latestRun ? formatPercent(latestRun.success_rate) : undefined,
      latestFailed: latestRun?.status === 'failed',
      active: selectedTask?.id === task.id,
    }
  }), [filteredTasks, runsByTask, selectedTask?.id])

  function chooseTask(task: Task) {
    setSelectedTaskId(task.id)
    setTaskDrawerOpen(false)
  }

  function chooseTaskById(taskId: string) {
    const task = taskList.find((item) => item.id === taskId)
    if (task) chooseTask(task)
  }

  function chooseRun(run: RunSummary) {
    setSelectedRunId(run.run_id)
  }

  async function createTask(draft: TaskDraft) {
    const config: TaskConfig = { name: draft.name.trim(), input: inputJsonFromDraft(draft) }
    const task = await createTaskAPI(config)
    setTaskList((current) => [task, ...current])
    setSelectedTaskId(task.id)
    setSelectedRunId('')
    setSelectedRequestId('')
  }

  async function startTaskRun(task: Task) {
    if (startingTaskId) return
    setStartingTaskId(task.id)
    try {
      const { run_id } = await startTaskRunAPI(task.id)
      const [state, runList] = await Promise.all([
        getRunState(run_id).catch(() => undefined),
        listTaskRuns(task.id).catch(() => []),
      ])
      const nextRuns = mergeStartedRun(runList, state, task)
      setRunsByTask((current) => ({ ...current, [task.id]: nextRuns }))
      setTaskList((current) => current.map((item) => item.id === task.id ? { ...item, latest_run: nextRuns[0] ?? item.latest_run } : item))
      setSelectedTaskId(task.id)
      setSelectedRunId(nextRuns.some((run) => run.run_id === run_id) ? run_id : nextRuns[0]?.run_id ?? '')
      setSelectedRequestId('')
      setErrorMessage('')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '启动任务失败')
    } finally {
      setStartingTaskId('')
    }
  }

  return (
    <main className="min-h-screen bg-muted/30 text-foreground lg:h-screen lg:overflow-hidden">
      <div className="grid min-h-screen lg:h-full lg:min-h-0 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="hidden min-h-0 border-r bg-sidebar/95 lg:block">
          <TaskSidebarContent items={taskNavItems} totalTaskCount={taskList.length} query={query} onQueryChange={setQuery} onChooseTask={chooseTaskById} />
        </aside>

        <section className="min-h-0 min-w-0 bg-background/80 lg:overflow-y-auto">
          <DashboardHeader taskDrawerOpen={taskDrawerOpen} onTaskDrawerOpenChange={setTaskDrawerOpen} taskDrawerContent={<TaskSidebarContent items={taskNavItems} totalTaskCount={taskList.length} query={query} onQueryChange={setQuery} onChooseTask={chooseTaskById} />} createTaskButton={<CreateTaskSheet onCreate={createTask} protocolOptions={protocolOptionsForCreate} protocols={protocols} variant="primary" />} filteredTaskCount={filteredTasks.length} totalTaskCount={taskList.length} standardTaskCount={standardTaskCount} turboTaskCount={turboTaskCount} integrityTaskCount={integrityTaskCount} />

          <div className="mx-auto max-w-375 space-y-4 p-4 sm:p-6">
            {errorMessage && <div className="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">{errorMessage}</div>}
            {!selectedTask && <Card className="rounded-2xl bg-card shadow-sm"><CardContent className="flex min-h-80 items-center justify-center p-6 text-sm text-muted-foreground">{loadingMessage || '暂无任务，请先创建任务。'}</CardContent></Card>}
            {selectedTask && <>
              <TaskOverview task={selectedTask} onCreate={createTask} onStartRun={startTaskRun} starting={startingTaskId === selectedTask.id} protocolOptions={protocolOptionsForCreate} protocols={protocols} />
              <TaskRunHistory runs={taskRuns} selectedRun={selectedRun} onChooseRun={chooseRun} samplesByRun={requestsByRun} />
              <RunDetail run={selectedRun} state={selectedRunState} requests={runRequests} selectedRequest={selectedRequest} onSelectRequest={setSelectedRequestId} />
            </>}
          </div>
        </section>
      </div>
    </main>
  )
}

const createSteps = [
  { title: '任务类型', description: '选择创建模式' },
  { title: '基础信息', description: '名称与目标' },
  { title: '类型配置', description: '填写运行参数' },
  { title: '确认创建', description: '预览输入' },
] as const

type CreateTaskSheetProps = {
  onCreate: (draft: TaskDraft) => Promise<void> | void
  sourceTask?: Task
  variant?: 'create' | 'copy' | 'primary'
  protocolOptions: string[]
  protocols: ProtocolMeta[]
}

function CreateTaskSheet({ onCreate, sourceTask, variant = 'create', protocolOptions, protocols }: CreateTaskSheetProps) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<TaskDraft>(() => sourceTask ? draftFromTask(sourceTask) : makeInitialDraft('standard', protocols))
  const [submitting, setSubmitting] = useState(false)
  const [integritySuites, setIntegritySuites] = useState<IntegritySuite[]>([])
  const [integritySuitesLoading, setIntegritySuitesLoading] = useState(false)
  const [integritySuitesError, setIntegritySuitesError] = useState('')
  const canSubmit = isDraftValid(draft) && (draft.mode !== 'integrity' || (!integritySuitesLoading && !integritySuitesError && integritySuites.length > 0))

  useEffect(() => {
    if (!open || draft.mode !== 'integrity' || !draft.protocol) return
    let cancelled = false
    const protocol = draft.protocol
    async function loadSuites() {
      await Promise.resolve()
      if (cancelled) return
      setIntegritySuitesLoading(true)
      setIntegritySuitesError('')
      try {
        const suites = await listIntegritySuites(protocol)
        if (cancelled) return
        setIntegritySuites(suites)
        setDraft((current) => {
          if (current.mode !== 'integrity' || current.protocol !== protocol) return current
          if (suites.some((suite) => suite.id === current.integritySuite)) return current
          return { ...current, integritySuite: suites[0]?.id ?? '' }
        })
      } catch (error) {
        if (cancelled) return
        setIntegritySuites([])
        setIntegritySuitesError(error instanceof Error ? error.message : '加载测试集失败')
      } finally {
        if (!cancelled) setIntegritySuitesLoading(false)
      }
    }
    loadSuites()
    return () => { cancelled = true }
  }, [open, draft.mode, draft.protocol])

  function initialDraft() {
    return sourceTask ? draftFromTask(sourceTask) : makeInitialDraft('standard', protocols)
  }

  function patch(update: Partial<TaskDraft>) {
    setDraft((current) => ({ ...current, ...update }))
  }

  function changeMode(mode: TaskMode) {
    const next = makeInitialDraft(mode, protocols)
    setDraft((current) => ({ ...next, name: current.name || next.name }))
  }

  function reset() {
    setDraft(initialDraft())
    setSubmitting(false)
  }

  async function submit() {
    if (!canSubmit || submitting) return
    setSubmitting(true)
    try {
      await onCreate(draft)
      setOpen(false)
      reset()
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(next) => { setOpen(next); if (next) setDraft(initialDraft()); if (!next) reset() }}>
      <DialogTrigger asChild>
        {variant === 'copy' ? (
          <Button variant="outline" size="sm" className="rounded-full">
            <Copy className="size-3.5" />复制为新任务
          </Button>
        ) : variant === 'primary' ? (
          <Button size="sm" className="h-10 rounded-2xl bg-foreground px-4 font-medium text-background shadow-sm hover:bg-foreground/90">
            <Plus className="size-3.5" />创建任务
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
        <Stepper key={`${open}-${sourceTask?.id ?? 'new'}`} steps={createSteps} rootClassName="flex min-h-0 flex-1 flex-col space-y-0" className="shrink-0 px-8 pt-6 pb-5" canAdvance={(current) => isStepValid(draft, current) && (current !== 2 || draft.mode !== 'integrity' || (!integritySuitesLoading && !integritySuitesError && integritySuites.length > 0))}>
          {({ current, isFirst, isLast, canGoNext, next, previous }) => {
            const validationHint = current === 2 && draft.mode === 'integrity' && integritySuitesLoading
              ? '正在加载当前协议的测试集。'
              : current === 2 && draft.mode === 'integrity' && integritySuitesError
                ? integritySuitesError
                : createStepHint(draft, current)
            return (
              <>
                <div className="min-h-0 flex-1 overflow-y-auto px-8 pb-6">
                  {current === 0 && <CreateStepType draft={draft} onModeChange={changeMode} />}
                  {current === 1 && <CreateStepBasics draft={draft} onPatch={patch} protocolOptions={protocolOptions} protocols={protocols} />}
                  {current === 2 && <CreateStepModeConfig draft={draft} onPatch={patch} integritySuites={integritySuites} integritySuitesLoading={integritySuitesLoading} integritySuitesError={integritySuitesError} />}
                  {current === 3 && <CreateStepReview draft={draft} />}
                </div>
                <DialogFooter className="shrink-0 gap-3 border-t bg-muted/20 p-0 px-8 py-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-h-5 text-xs leading-5 text-muted-foreground">{!canGoNext && !isLast ? validationHint : isLast ? '确认无误后将调用后端接口创建真实任务。' : '可以继续下一步，也可以点击上方已完成步骤返回修改。'}</div>
                  <div className="flex shrink-0 gap-2">
                    <Button variant="outline" onClick={previous} disabled={isFirst || submitting}>上一步</Button>
                    {!isLast ? <Button onClick={next} disabled={!canGoNext}>{nextStepLabel(current)}</Button> : <Button onClick={submit} disabled={!canSubmit || submitting}>{submitting ? '创建中...' : '创建任务'}</Button>}
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

function CreateStepBasics({ draft, onPatch, protocolOptions, protocols }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void; protocolOptions: string[]; protocols: ProtocolMeta[] }) {
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
          <FormField label="协议" required description="决定请求体结构和默认地址，切换后会自动带入对应接口地址。"><OptionPicker value={draft.protocol} options={protocolOptions} onChange={(protocol) => onPatch({ protocol, endpoint: defaultEndpoint(protocol, protocols) })} /></FormField>
          <FormField label="模型名称" required description="填写要压测或校验的模型标识，会写入最终请求配置。"><Input value={draft.model} onChange={(event) => onPatch({ model: event.target.value })} /></FormField>
          <FormField label="请求地址" required description="目标 API 的完整地址；如使用网关或代理，可在这里改为内部地址。"><Input value={draft.endpoint} onChange={(event) => onPatch({ endpoint: event.target.value })} /></FormField>
          <FormField label="API Key" description="可选；OpenAI/Anthropic 等云服务通常需要，本地或已鉴权代理可留空。"><Input type="password" autoComplete="off" value={draft.apiKey} onChange={(event) => onPatch({ apiKey: event.target.value })} placeholder="sk-..." /></FormField>
        </div>
      </section>
    </div>
  )
}

function CreateStepModeConfig({ draft, onPatch, integritySuites, integritySuitesLoading, integritySuitesError }: { draft: TaskDraft; onPatch: (update: Partial<TaskDraft>) => void; integritySuites: IntegritySuite[]; integritySuitesLoading: boolean; integritySuitesError: string }) {
  if (draft.mode === 'integrity') {
    const selectedSuite = integritySuites.find((suite) => suite.id === draft.integritySuite)
    return (
      <div className="space-y-6">
        <section className="border-b pb-6">
          <div className="mb-5 flex items-center gap-2 text-sm font-medium"><ShieldCheck className="size-4" />测试集来源</div>
          <div className="grid gap-4">
            <FormField label="测试集" required description="测试集来自当前已加载的完整性规则，不能手动填写不存在的名称。">
              <Select value={draft.integritySuite} onValueChange={(integritySuite) => onPatch({ integritySuite })} disabled={integritySuitesLoading || integritySuites.length === 0}>
                <SelectTrigger><SelectValue placeholder={integritySuitesLoading ? '加载测试集中...' : '选择测试集'} /></SelectTrigger>
                <SelectContent>
                  {integritySuites.map((suite) => <SelectItem key={suite.id} value={suite.id}>{suite.name || suite.id} · {suite.cases?.length ?? 0} 个用例</SelectItem>)}
                </SelectContent>
              </Select>
              {integritySuitesError ? <p className="mt-2 text-xs text-destructive">{integritySuitesError}</p> : null}
              {!integritySuitesLoading && !integritySuitesError && integritySuites.length === 0 ? <p className="mt-2 text-xs text-destructive">当前协议没有已加载的测试集，请先等待规则加载完成或检查规则缓存。</p> : null}
              {selectedSuite ? <p className="mt-2 text-xs leading-5 text-muted-foreground">{selectedSuite.description || selectedSuite.id}，包含 {selectedSuite.cases?.length ?? 0} 个用例。</p> : null}
            </FormField>
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
        <p className="mb-4 text-xs leading-5 text-muted-foreground">这里展示的是最终会进入任务配置的真实结构，因此保留实际字段名；敏感字段会脱敏显示。</p>
        <CodeBlock label="TaskConfig.Input" value={JSON.stringify(redactSecretInput(inputJsonFromDraft(draft)), null, 2)} icon={<FileJson className="size-3.5" />} />
      </section>
    </div>
  )
}

function TaskRunHistory({ runs, selectedRun, onChooseRun, samplesByRun }: { runs: RunSummary[]; selectedRun?: RunSummary; onChooseRun: (run: RunSummary) => void; samplesByRun: Record<string, RequestDetail[]> }) {
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
        <ExecutionTrend runs={runs} samplesByRun={samplesByRun} />
        {runs.length === 0 ? (
          <div className="rounded-2xl border border-dashed bg-background/70 px-4 py-10 text-center text-sm text-muted-foreground">这个任务还没有执行记录。点击上方“开始运行”后，这里会显示每次执行的结果。</div>
        ) : (
          <div className="overflow-x-auto rounded-2xl border bg-background/70">
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

function DraftTaskPreview({ task }: { task: Task }) {
  const prompt = promptSpec(task.input)
  return (
    <div className="space-y-4 rounded-3xl border bg-background/70 p-4 shadow-xs">
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-2xl bg-muted text-muted-foreground"><ModeIcon mode={taskMode(task)} className="size-5" /></div>
        <div className="min-w-0">
          <div className="font-medium">{task.name}</div>
          <div className="mt-1 text-sm leading-5 text-muted-foreground">{modeLabel[taskMode(task)]} · {taskModel(task)}</div>
        </div>
      </div>
      <div className="grid gap-2 text-sm sm:grid-cols-2">
        <KeyValue label="模型名称" value={taskModel(task)} />
        <KeyValue label="协议类型" value={taskProtocol(task)} />
        <KeyValue label={taskMode(task) === 'integrity' ? '测试集' : 'Prompt'} value={taskMode(task) === 'integrity' ? task.input.integrity?.suite ?? '-' : prompt ? `${promptModeLabel[prompt.mode]} · ${prompt.summary}` : '-'} />
        <KeyValue label="请求地址" value={taskEndpoint(task)} />
        <KeyValue label="API Key" value={maskSecret(task.input.api_key)} />
      </div>
      <TaskModeConfigPreview task={task} />
    </div>
  )
}

function TaskOverview({ task, onCreate, onStartRun, starting, protocolOptions, protocols }: { task: Task; onCreate: (draft: TaskDraft) => Promise<void> | void; onStartRun: (task: Task) => Promise<void> | void; starting: boolean; protocolOptions: string[]; protocols: ProtocolMeta[] }) {
  const mode = taskMode(task)
  return (
    <Card className="rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40">
      <CardHeader className="p-4 sm:p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="min-w-0">
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Badge className="gap-1.5"><ModeIcon mode={mode} className="size-3.5" />{modeLabel[mode]}</Badge>
              <Badge variant="outline" className="gap-1.5"><Network className="size-3.5" />{taskProtocol(task)}</Badge>
            </div>
            <CardTitle className="text-xl sm:text-2xl">{task.name}</CardTitle>
            <CardDescription className="mt-2 max-w-2xl leading-6">创建于 {formatDate(task.created_at)}，最近更新 {formatDate(task.updated_at)}</CardDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2 lg:justify-end">
            <Button size="sm" className="rounded-full" onClick={() => onStartRun(task)} disabled={starting}>
              <Play className="size-3.5" />{starting ? '启动中...' : '开始运行'}
            </Button>
            <CreateTaskSheet onCreate={onCreate} sourceTask={task} variant="copy" protocolOptions={protocolOptions} protocols={protocols} />
            <TaskConfigSheet task={task} />
          </div>
        </div>
      </CardHeader>
      <CardContent className="grid gap-2 p-4 pt-0 text-sm sm:grid-cols-2 lg:grid-cols-4 sm:p-5 sm:pt-0">
        <KeyValue label="模型名称" value={taskModel(task)} />
        <KeyValue label="协议类型" value={taskProtocol(task)} />
        <KeyValue label={mode === 'turbo' ? '每级请求' : mode === 'integrity' ? '测试集' : '请求总数'} value={mode === 'integrity' ? task.input.integrity?.suite ?? '-' : String(taskRequests(task))} />
        <KeyValue label={mode === 'turbo' ? '起始并发' : '并发数'} value={String(taskConcurrency(task))} />
      </CardContent>
    </Card>
  )
}

function mergeStartedRun(runs: RunSummary[], state: RunState | undefined, task: Task) {
  if (!state || runs.some((run) => run.run_id === state.run_id)) return runs
  return [runSummaryFromState(state, task), ...runs]
}

function runSummaryFromState(state: RunState, task: Task): RunSummary {
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

function TaskConfigSheet({ task }: { task: Task }) {
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="outline" size="sm" className="rounded-full">
          <Settings2 className="size-3.5" />请求配置
        </Button>
      </SheetTrigger>
      <SheetContent className="w-[min(96vw,1040px)]! max-w-none! overflow-y-auto">
        <SheetHeader className="border-b bg-muted/30 px-6 py-5">
          <SheetTitle className="flex items-center gap-2"><Settings2 className="size-4" />请求配置</SheetTitle>
          <SheetDescription>这些参数来自后端任务配置。</SheetDescription>
        </SheetHeader>
        <div className="space-y-5 px-6">
          <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
            <div className="mb-3 flex items-center gap-2 text-sm font-medium"><Route className="size-4" />请求目标</div>
            <div className="grid gap-2 text-sm sm:grid-cols-2">
              <KeyValue label="请求地址" value={taskEndpoint(task)} />
              <KeyValue label="模型名称" value={taskModel(task)} />
              <KeyValue label="协议类型" value={taskProtocol(task)} />
              <KeyValue label="任务类型" value={modeLabel[taskMode(task)]} />
              <KeyValue label="API Key" value={maskSecret(task.input.api_key)} />
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
      {taskMode(task) !== 'integrity' && <PromptPanel task={task} />}
    </div>
  )
}

function ModeSpecificPanel({ task }: { task: Task }) {
  const mode = taskMode(task)
  const input = task.input
  if (mode === 'turbo') {
    const cfg = input.turbo_config
    const levels = turboLevelsFromConfig(cfg)
    return (
      <div className="rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-sm font-medium"><Zap className="size-4" />Turbo 配置</div>
          <Badge variant="outline">{levels.length} 个级别</Badge>
        </div>
        <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
          <KeyValue label="起始并发" value={String(cfg?.init_concurrency ?? '-')} />
          <KeyValue label="最大并发" value={String(cfg?.max_concurrency ?? '-')} />
          <KeyValue label="每级递增" value={String(cfg?.step_size ?? '-')} />
          <KeyValue label="每级请求数" value={String(cfg?.level_requests ?? input.count ?? '-')} />
          <KeyValue label="最低成功率" value={String(cfg?.min_success_rate ?? '-')} />
          <KeyValue label="最大延迟" value={cfg?.max_latency ?? '-'} />
        </div>
        <div className="mt-4 rounded-2xl bg-muted/50 px-4 py-3 text-xs leading-5 text-muted-foreground">
          并发级别：{levels.length > 0 ? levels.join(' → ') : '-'}
        </div>
      </div>
    )
  }

  if (mode === 'integrity') {
    const integrity = input.integrity
    return (
      <div className="xl:col-span-2 rounded-2xl border bg-background/70 p-4 shadow-xs">
        <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2 text-sm font-medium"><ClipboardList className="size-4" />完整性测试集</div>
          <Badge variant="outline">suite</Badge>
        </div>
        <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
          <KeyValue label="测试集名称" value={integrity?.suite ?? '-'} />
          <InlineSwitch label="失败时立即停止" enabled={integrity?.fail_fast ?? false} />
          <KeyValue label="单个用例超时" value={integrity?.case_timeout_ms ? `${integrity.case_timeout_ms}ms` : '-'} />
        </div>
        <div className="mt-4 rounded-2xl bg-muted/50 px-4 py-3 text-xs leading-5 text-muted-foreground">完整性校验由后端根据协议和测试集加载用例；任务配置本身只保存 suite 与执行控制。</div>
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
            <NumberStat label="并发数" value={String(input.concurrency ?? 0)} />
            <NumberStat label="请求总数" value={String(input.count ?? 0)} />
          </div>
          <p className="mt-4 text-xs leading-5 text-muted-foreground">按请求总数生成请求，最多同时运行指定并发数。</p>
        </div>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
          <KeyValue label="请求超时" value={input.timeout ?? '-'} />
          <InlineSwitch label="流式响应" enabled={input.stream ?? false} />
          <InlineSwitch label="启用思考" enabled={input.thinking ?? false} />
          <InlineSwitch label="生成报告" enabled={input.report ?? false} />
          <InlineSwitch label="记录日志" enabled={input.log ?? false} />
          <KeyValue label="Prompt 来源" value={promptSpec(input) ? promptModeLabel[promptSpec(input)!.mode] : '-'} />
        </div>
      </div>
    </div>
  )
}

function PromptPanel({ task }: { task: Task }) {
  const prompt = promptSpec(task.input)
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

function RunRate({ value }: { value: number }) {
  return (
    <div className="flex min-w-28 items-center gap-2">
      <div className="h-1.5 flex-1 rounded-full bg-muted"><div className="h-full rounded-full bg-primary" style={{ width: `${value}%` }} /></div>
      <span className="w-10 text-right text-sm tabular-nums">{value}%</span>
    </div>
  )
}

function RunDetail({ run, state, requests, selectedRequest, onSelectRequest }: { run?: RunSummary; state?: RunState; requests: RequestDetail[]; selectedRequest?: RequestDetail; onSelectRequest: (id: string) => void }) {
  if (!run) {
    return <Card className="rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40"><CardContent className="flex min-h-60 items-center justify-center p-6 text-sm text-muted-foreground"><Clock3 className="mr-2 size-4" />当前任务暂无执行记录。</CardContent></Card>
  }

  const integrity = integritySnapshot(state)
  const successCount = requests.filter((request) => request.success).length
  const failedCount = requests.filter((request) => request.status === 'failed' || !request.success).length
  const doneCount = successCount + failedCount
  const progress = requests.length > 0 ? Math.round((doneCount / requests.length) * 100) : Math.round(run.success_rate || 0)

  return (
    <Card className="rounded-3xl bg-card/95 shadow-sm ring-1 ring-border/40">
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
          <div className="space-y-4">
            <section className="rounded-2xl border bg-background/70 p-4">
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

          <section className="min-w-0 rounded-2xl border bg-muted/25 p-4 ring-1 ring-border/20">
            <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <div className="flex items-center gap-2 text-sm font-medium"><Network className="size-4" />请求样本检查</div>
                <div className="mt-1 text-xs leading-5 text-muted-foreground">按请求样本查看耗时、Token、网络与原始请求响应。</div>
              </div>
              <Badge variant="outline" className="w-fit bg-background">{requests.length} 个样本</Badge>
            </div>
            <div className="grid min-h-0 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
              <div className="space-y-2 rounded-2xl border bg-background/75 p-2 xl:max-h-140 xl:overflow-y-auto xl:scrollbar-gutter-stable">
                {requests.length === 0 ? <div className="rounded-xl bg-muted/50 px-3 py-6 text-center text-sm text-muted-foreground">暂无请求样本。</div> : requests.map((request) => {
                  const key = requestKey(request)
                  const active = selectedRequest && requestKey(selectedRequest) === key
                  return (
                    <button key={key} type="button" onClick={() => onSelectRequest(key)} className={cn('flex w-full items-center justify-between gap-3 rounded-xl border border-transparent bg-transparent px-3 py-2.5 text-left text-sm hover:bg-muted/70', active && 'border-border bg-muted shadow-xs')}>
                      <span className="min-w-0">
                        <span className="block truncate font-medium">#{request.index}{request.case_id ? ` · ${request.case_id}` : ''}</span>
                        <span className="mt-0.5 block truncate text-xs text-muted-foreground">{request.total_time} · TTFT {request.ttft}</span>
                      </span>
                      {request.status === 'failed' || !request.success ? <XCircle className="size-4 shrink-0 text-red-500" /> : <CheckCircle2 className="size-4 shrink-0 text-emerald-500" />}
                    </button>
                  )
                })}
              </div>
              {selectedRequest && <RequestPanel request={selectedRequest} compact />}
            </div>
          </section>
        </div>
      </CardContent>
    </Card>
  )
}

function InlineMetric({ icon, label, value, tone = 'neutral' }: { icon: React.ReactNode; label: string; value: string; tone?: 'neutral' | 'success' | 'danger' }) {
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
    <div className={cn('rounded-2xl border bg-background p-4 shadow-xs', compact && 'mt-4 xl:mt-0')}>
      <div className="mb-4 flex items-center justify-between gap-3">
        <div>
          <div className="font-medium">请求详情 #{request.index}</div>
          <div className="mt-1 text-xs text-muted-foreground">状态、耗时、Token 与网络阶段明细</div>
        </div>
        {request.error_message ? <Badge className="bg-red-600"><AlertTriangle className="size-3" />{request.error_message}</Badge> : <Badge className="bg-emerald-600">OK</Badge>}
      </div>
      <div className={cn('grid gap-4', !compact && 'xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]')}>
        <div>
          <div className="mb-2 flex items-center gap-2 text-sm font-medium"><Gauge className="size-4" />本次指标</div>
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {metricItems.map(([label, value]) => <RequestMetricTile key={label} label={label} value={value} />)}
          </div>
        </div>
        <div className="space-y-3">
          <CodeBlock label="请求内容" value={request.request_body || '-'} icon={<Network className="size-3.5" />} />
          <CodeBlock label="响应内容" value={request.response_body || request.error_message || '-'} icon={<FileJson className="size-3.5" />} />
        </div>
      </div>
    </div>
  )
}

function RequestMetricTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-lg bg-muted/65 px-3 py-2.5 ring-1 ring-border/30">
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

export default App
