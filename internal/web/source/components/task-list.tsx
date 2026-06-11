import { CheckCircle2, CircleDashed, MoreHorizontal, XCircle } from 'lucide-react'

import type { Task } from '@/data'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'

const statusMeta = {
  running: { label: 'running', icon: CircleDashed, className: 'text-blue-500' },
  done: { label: 'done', icon: CheckCircle2, className: 'text-emerald-500' },
  failed: { label: 'failed', icon: XCircle, className: 'text-destructive' },
  queued: { label: 'queued', icon: CircleDashed, className: 'text-amber-500' },
}

export function TaskList({ tasks, selectedId, onSelect }: { tasks: Task[]; selectedId: string; onSelect: (id: string) => void }) {
  return (
    <Card className="h-full">
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div>
            <CardTitle>任务</CardTitle>
            <CardDescription>管理标准压测、Turbo 并发探测与 Integrity 校验任务。</CardDescription>
          </div>
          <Button variant="outline" size="sm">刷新</Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {tasks.map((task) => {
          const meta = statusMeta[task.status]
          return (
            <button
              key={task.id}
              onClick={() => onSelect(task.id)}
              className={cn(
                'w-full rounded-xl border p-4 text-left transition-all hover:border-primary/50 hover:bg-accent/40',
                selectedId === task.id && 'border-primary bg-primary/5 shadow-sm',
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="truncate font-semibold">{task.name}</h3>
                    <Badge variant="secondary">{task.mode}</Badge>
                    <Badge variant="outline">{task.protocol}</Badge>
                  </div>
                  <p className="mt-1 truncate text-sm text-muted-foreground">{task.model}</p>
                </div>
                <div className={cn('inline-flex items-center gap-1.5 text-xs font-medium', meta.className)}>
                  <meta.icon className="size-3.5" />
                  {meta.label}
                </div>
              </div>

              <div className="mt-4 grid grid-cols-2 gap-3 text-xs text-muted-foreground sm:grid-cols-4">
                <span>requests={task.requests}</span>
                <span>c={task.concurrency}</span>
                <span>ttft={task.avgTTFT}</span>
                <span>tps={task.avgTPS}</span>
              </div>
              <div className="mt-4 flex items-center gap-3">
                <Progress value={task.successRate} className="h-1.5" />
                <span className="text-xs tabular-nums text-muted-foreground">{task.successRate}%</span>
              </div>
            </button>
          )
        })}
        <Button variant="ghost" className="w-full justify-between text-muted-foreground">
          查看更多历史任务
          <MoreHorizontal className="size-4" />
        </Button>
      </CardContent>
    </Card>
  )
}
