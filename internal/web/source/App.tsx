import { useMemo, useState } from 'react'
import { Plus, Search } from 'lucide-react'

import { metrics, tasks } from '@/data'
import { AppShell } from '@/components/shell'
import { MetricCard } from '@/components/metric-card'
import { RunPanel } from '@/components/run-panel'
import { TaskEditor } from '@/components/task-editor'
import { TaskList } from '@/components/task-list'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

function App() {
  const [selectedId, setSelectedId] = useState(tasks[0].id)
  const selectedTask = useMemo(() => tasks.find((task) => task.id === selectedId) ?? tasks[0], [selectedId])

  return (
    <AppShell>
      <div className="mx-auto flex max-w-7xl flex-col gap-6">
        <section className="flex flex-col gap-4 rounded-2xl border bg-linear-to-br from-card via-card to-primary/5 p-5 shadow-sm lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-sm font-medium text-primary">实时仪表盘</p>
            <h2 className="mt-2 text-2xl font-semibold tracking-tight sm:text-3xl">批量测试 AI 模型性能指标</h2>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              使用 OpenAI / Anthropic 协议配置测试任务，观察成功率、TTFT、TPS、缓存命中与请求级网络耗时。
            </p>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input className="pl-9 sm:w-64" placeholder="搜索任务、模型或协议" />
            </div>
            <Button><Plus className="size-4" />新建任务</Button>
          </div>
        </section>

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {metrics.map((metric) => (
            <MetricCard key={metric.label} metric={metric} />
          ))}
        </section>

        <section className="grid gap-6 xl:grid-cols-[420px_1fr]">
          <TaskList tasks={tasks} selectedId={selectedId} onSelect={setSelectedId} />
          <div className="space-y-6">
            <RunPanel task={selectedTask} />
            <TaskEditor task={selectedTask} />
          </div>
        </section>
      </div>
    </AppShell>
  )
}

export default App
