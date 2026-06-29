import { ShieldCheck } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { formatDate } from '@/lib/task-utils'
import type { IntegrityAssertionResult, IntegrityCaseResult } from '@/api'

export type IntegritySnapshot = {
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

export function IntegrityRunDetail({ snapshot }: { snapshot: IntegritySnapshot }) {
  const selectedCase = snapshot.cases.find((item) => item.case_id === snapshot.currentCaseId) ?? snapshot.cases[0]

  return (
    <Card className="rounded-2xl border bg-card shadow-none ring-0">
      <CardContent className="p-4 sm:p-5">
        <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div className="flex items-center gap-2 text-sm font-medium"><ShieldCheck className="size-4" />完整性校验结果</div>
            <div className="mt-1 text-xs leading-5 text-muted-foreground">{snapshot.suiteId || '-'} · {snapshot.message || snapshot.status || '等待后端同步状态'}</div>
          </div>
          <Badge variant="outline">{snapshot.currentCaseId ? `当前 ${snapshot.currentCaseId}` : snapshot.status || 'suite'}</Badge>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <Metric label="总用例" value={String(snapshot.totalCases)} />
          <Metric label="通过用例" value={String(snapshot.passedCases)} />
          <Metric label="失败用例" value={String(snapshot.failedCases)} />
          <Metric label="跳过用例" value={String(snapshot.skippedCases)} />
        </div>
        <div className="mt-4 grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)]">
          <div className="space-y-2 pr-1 xl:max-h-130 xl:overflow-y-auto xl:scrollbar-gutter-stable">
            {snapshot.cases.map((item) => (
              <div key={item.case_id} className={cn('rounded-xl border bg-background/80 px-3 py-3 text-sm transition', item.case_id === selectedCase?.case_id && 'border-primary/40 bg-primary/5 shadow-xs')}>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate font-medium">{item.name || item.case_id}</div>
                    <div className="mt-1 text-xs text-muted-foreground">{item.capability || '未标注能力'} · {item.required ? '必需' : '可选'}</div>
                  </div>
                  <Badge variant="outline">{item.status}</Badge>
                </div>
                <div className="mt-2 text-xs text-muted-foreground">{item.duration || '-'}</div>
              </div>
            ))}
          </div>
          <div className="rounded-xl border bg-background/80 p-4 text-sm text-muted-foreground">
            {selectedCase ? (
              <>
                <div className="font-medium text-foreground">{selectedCase.name || selectedCase.case_id}</div>
                <div className="mt-1 text-xs">{selectedCase.case_id} · {selectedCase.capability || '未标注能力'} · {selectedCase.required ? '必需' : '可选'} · {selectedCase.duration || '-'}</div>
                <div className="mt-4 grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
                  <KV label="开始时间" value={formatDate(selectedCase.started_at)} />
                  <KV label="结束时间" value={formatDate(selectedCase.finished_at)} />
                  <KV label="断言总数" value={String(selectedCase.total_assertions)} />
                  <KV label="错误信息" value={selectedCase.error_message || '-'} />
                </div>
                <div className="mt-4 text-xs text-muted-foreground">该文件先把 integrity 独立出来，后续可以继续扩展 case 断言详情。</div>
              </>
            ) : (
              '完整性用例尚未产生结果。'
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return <div className="rounded-lg bg-muted/35 px-3 py-2"><div className="text-[11px] leading-4 text-muted-foreground">{label}</div><div className="mt-1 truncate text-sm font-medium">{value}</div></div>
}

function KV({ label, value }: { label: string; value: string }) {
  return <div className="rounded-lg bg-muted/35 px-3 py-2"><div className="text-[11px] leading-4 text-muted-foreground">{label}</div><div className="mt-1 truncate text-sm font-medium">{value}</div></div>
}
