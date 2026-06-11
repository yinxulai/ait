import { Pause, Play, Square } from 'lucide-react'

import { requests, type Task } from '@/data'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { cn } from '@/lib/utils'

export function RunPanel({ task }: { task: Task }) {
  const progress = task.status === 'running' ? 32 : task.status === 'done' ? 100 : 42

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <CardTitle>运行详情</CardTitle>
              <Badge>{task.mode}</Badge>
              <Badge variant="outline">run_20260612</Badge>
            </div>
            <CardDescription className="mt-1">{task.endpoint}</CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm"><Play className="size-4" />启动</Button>
            <Button size="sm" variant="secondary"><Pause className="size-4" />后台</Button>
            <Button size="sm" variant="destructive"><Square className="size-4" />停止</Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <section className="rounded-xl border bg-muted/30 p-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-medium">running · 32/100 · elapsed=28s</p>
              <p className="mt-1 text-sm text-muted-foreground">
                success={task.successRate}% · failed=1 · ttft={task.avgTTFT} · tps={task.avgTPS} · cache={task.cacheHitRate}%
              </p>
            </div>
            <span className="text-2xl font-semibold tabular-nums">{progress}%</span>
          </div>
          <Progress value={progress} className="mt-4" />
        </section>

        <section>
          <div className="mb-3 flex items-center justify-between">
            <h3 className="font-semibold">请求</h3>
            <Button variant="ghost" size="sm">请求详情</Button>
          </div>
          <div className="overflow-hidden rounded-xl border">
            {requests.map((request) => (
              <div
                key={request.id}
                className="grid grid-cols-12 gap-3 border-b px-4 py-3 text-sm last:border-b-0"
              >
                <div className="col-span-3 font-medium sm:col-span-2">#{request.id.toString().padStart(3, '0')}</div>
                <div className={cn('col-span-3 sm:col-span-2', request.status === 'ok' && 'text-emerald-500', request.status === 'failed' && 'text-destructive', request.status === 'running' && 'text-blue-500')}>
                  {request.status}
                </div>
                <div className="col-span-6 text-muted-foreground sm:col-span-8">
                  {request.error ?? `total=${request.total} · ttft=${request.ttft} · tps=${request.tps} · tokens=${request.tokens}`}
                </div>
              </div>
            ))}
          </div>
        </section>
      </CardContent>
    </Card>
  )
}
