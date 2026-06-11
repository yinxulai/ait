import { Save } from 'lucide-react'

import type { Task } from '@/data'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'

export function TaskEditor({ task }: { task: Task }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>创建 / 编辑任务</CardTitle>
        <CardDescription>覆盖基础信息、请求参数和运行模式三组字段。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <Tabs>
          <TabsList className="w-full sm:w-fit">
            <TabsTrigger data-state="active">1 基础</TabsTrigger>
            <TabsTrigger>2 请求</TabsTrigger>
            <TabsTrigger>3 模式</TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2 text-sm font-medium">
            任务名称
            <Input defaultValue={task.name} />
          </label>
          <label className="space-y-2 text-sm font-medium">
            模型
            <Input defaultValue={task.model} />
          </label>
          <label className="space-y-2 text-sm font-medium md:col-span-2">
            Endpoint
            <Input defaultValue={task.endpoint} />
          </label>
          <label className="space-y-2 text-sm font-medium">
            请求数
            <Input type="number" defaultValue={task.requests} />
          </label>
          <label className="space-y-2 text-sm font-medium">
            并发数
            <Input type="number" defaultValue={task.concurrency} />
          </label>
          <label className="space-y-2 text-sm font-medium md:col-span-2">
            Prompt
            <Textarea rows={4} defaultValue="请用简洁中文解释 AIT 的测试目标，并给出三个关键指标。" />
          </label>
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="outline">重置</Button>
          <Button><Save className="size-4" />保存任务</Button>
        </div>
      </CardContent>
    </Card>
  )
}
