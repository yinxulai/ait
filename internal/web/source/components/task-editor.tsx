import { useEffect, useState } from 'react'
import { ClipboardList, Copy, FileJson, Gauge, ListChecks, Network, Play, Plus, Route, Settings2, ShieldCheck, Zap } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { Stepper } from '@/components/ui/stepper'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { BooleanToggle, CodeBlock, FormField, InlineSwitch, KeyValue, ModeIcon, NumberStat, OptionPicker } from '@/components/task-editor-shared'
import { cn } from '@/lib/utils'
import { createModeHint, createStepHint, defaultEndpoint, draftFromTask, formatDate, inputJsonFromDraft, isDraftValid, isStepValid, makeInitialDraft, maskSecret, modeLabel, nextStepLabel, promptFieldLabel, promptModeLabel, promptModeOptions, promptSpec, redactSecretInput, taskConcurrency, taskEndpoint, taskFromDraft, taskMode, taskModel, taskProtocol, taskRequests, toNumber, turboLevelsFromConfig, type TaskDraft } from '@/lib/task-utils'
import { listIntegritySuites, type IntegritySuite, type PromptMode, type ProtocolMeta, type Task, type TaskMode } from '@/api'

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

export function CreateTaskSheet({ onCreate, sourceTask, variant = 'create', protocolOptions, protocols }: CreateTaskSheetProps) {
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

export function TaskOverview({ task, onCreate, onStartRun, starting, protocolOptions, protocols }: { task: Task; onCreate: (draft: TaskDraft) => Promise<void> | void; onStartRun: (task: Task) => Promise<void> | void; starting: boolean; protocolOptions: string[]; protocols: ProtocolMeta[] }) {
  const mode = taskMode(task)
  return (
    <Card className="rounded-2xl border bg-card shadow-none ring-0">
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
          并发级别：{levels.length > 0 ? levels.join(' -> ') : '-'}
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

function TaskModeConfigPreview({ task }: { task: Task }) {
  return (
    <div>
      <div className="mb-3 flex items-center gap-2 text-sm font-medium"><Settings2 className="size-4" />类型配置</div>
      <ModeSpecificPanel task={task} />
    </div>
  )
}
