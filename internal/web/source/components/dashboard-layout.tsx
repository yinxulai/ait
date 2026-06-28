import { Activity, BarChart3, ChevronRight, ListChecks, Menu, Route, Search, ShieldCheck, Zap } from 'lucide-react'
import type { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import type { TaskMode } from '@/api'

export type TaskNavItem = {
  id: string
  name: string
  mode: TaskMode
  modeLabel: string
  model: string
  updatedAt: string
  runsCount: number
  successRate?: string
  latestFailed: boolean
  active: boolean
}

type DashboardHeaderProps = {
  taskDrawerOpen: boolean
  onTaskDrawerOpenChange: (open: boolean) => void
  taskDrawerContent: ReactNode
  createTaskButton: ReactNode
  filteredTaskCount: number
  totalTaskCount: number
  standardTaskCount: number
  turboTaskCount: number
  integrityTaskCount: number
}

export function DashboardHeader({ taskDrawerOpen, onTaskDrawerOpenChange, taskDrawerContent, createTaskButton, filteredTaskCount, totalTaskCount, standardTaskCount, turboTaskCount, integrityTaskCount }: DashboardHeaderProps) {
  return (
    <header className="sticky top-0 z-20 border-b bg-background/90 px-4 py-3 backdrop-blur sm:px-6">
      <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
            <BarChart3 className="size-3.5" />Operations Dashboard
          </div>
          <h2 className="mt-1 text-xl font-semibold tracking-tight sm:text-2xl">AIT 执行观测台</h2>
        </div>
        <div className="flex flex-wrap items-center gap-2 xl:justify-end">
          <Sheet open={taskDrawerOpen} onOpenChange={onTaskDrawerOpenChange}>
            <SheetTrigger asChild>
              <Button variant="outline" size="sm" className="rounded-xl lg:hidden">
                <Menu className="size-3.5" />任务
              </Button>
            </SheetTrigger>
            <SheetContent side="right" className="w-[min(88vw,360px)]! max-w-none! gap-0 p-0">
              <SheetHeader className="sr-only">
                <SheetTitle>任务列表</SheetTitle>
                <SheetDescription>选择任务或搜索任务。</SheetDescription>
              </SheetHeader>
              {taskDrawerContent}
            </SheetContent>
          </Sheet>
          <div className="grid grid-cols-4 overflow-hidden rounded-2xl border bg-card p-1 shadow-xs">
            <HeaderMetric icon={<ListChecks className="size-3.5" />} label="全部" value={`${filteredTaskCount}/${totalTaskCount}`} />
            <HeaderMetric icon={<Route className="size-3.5" />} label="标准" value={String(standardTaskCount)} />
            <HeaderMetric icon={<Zap className="size-3.5" />} label="Turbo" value={String(turboTaskCount)} />
            <HeaderMetric icon={<ShieldCheck className="size-3.5" />} label="完整性" value={String(integrityTaskCount)} />
          </div>
          {createTaskButton}
        </div>
      </div>
    </header>
  )
}

type TaskSidebarContentProps = {
  items: TaskNavItem[]
  totalTaskCount: number
  query: string
  onQueryChange: (value: string) => void
  onChooseTask: (taskId: string) => void
}

export function TaskSidebarContent({ items, totalTaskCount, query, onQueryChange, onChooseTask }: TaskSidebarContentProps) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="border-b px-5 py-4">
        <div className="flex items-center gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-sidebar-primary text-sidebar-primary-foreground">
            <Activity className="size-4" />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-base font-semibold tracking-tight">AIT Dashboard</h1>
            <p className="truncate text-xs text-muted-foreground">任务执行与完整性校验</p>
          </div>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={query} onChange={(event) => onQueryChange(event.target.value)} className="h-10 rounded-xl bg-background pl-9" placeholder="搜索任务 / 模型 / 协议" />
        </div>
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>任务队列</span>
          <span>{items.length} / {totalTaskCount}</span>
        </div>
        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-2 pr-4">
            {items.map((task) => (
              <button key={task.id} type="button" onClick={() => onChooseTask(task.id)} className={cn('group w-full rounded-xl border border-transparent bg-background/70 px-3 py-3 text-left transition hover:border-sidebar-border hover:bg-sidebar-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring', task.active && 'border-sidebar-primary bg-sidebar-accent shadow-sm')}>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <TaskModeIcon mode={task.mode} className="size-3.5 shrink-0 text-muted-foreground" />
                      <div className="truncate text-sm font-medium leading-5">{task.name}</div>
                    </div>
                    <div className="mt-1 truncate text-xs text-muted-foreground">{task.modeLabel} · {task.model}</div>
                  </div>
                  <ChevronRight className="mt-0.5 size-4 shrink-0 text-muted-foreground/60 transition group-hover:translate-x-0.5 group-hover:text-foreground" />
                </div>
                <div className="mt-3 flex items-center justify-between gap-3 text-xs text-muted-foreground">
                  <span>{task.runsCount} 次执行</span>
                  <span className="inline-flex items-center gap-2">
                    {task.successRate && <span className={cn('tabular-nums', task.latestFailed ? 'text-red-600' : 'text-emerald-600')}>{task.successRate}</span>}
                    <span>{task.updatedAt}</span>
                  </span>
                </div>
              </button>
            ))}
          </div>
        </ScrollArea>
      </div>
    </div>
  )
}

function HeaderMetric({ icon, label, value }: { icon: ReactNode; label: string; value: string }) {
  return (
    <div className="flex min-w-[4.25rem] items-center justify-center gap-1.5 rounded-xl px-2.5 py-2 text-xs hover:bg-muted/70 sm:min-w-22 sm:justify-start sm:gap-2 sm:px-3">
      <span className="text-muted-foreground">{icon}</span>
      <span className="font-semibold tabular-nums text-foreground">{value}</span>
      <span className="hidden text-muted-foreground sm:inline">{label}</span>
    </div>
  )
}

function TaskModeIcon({ mode, className }: { mode: TaskMode; className?: string }) {
  if (mode === 'turbo') return <Zap className={className} />
  if (mode === 'integrity') return <ShieldCheck className={className} />
  return <Route className={className} />
}
