import type { Metric } from '@/data'
import { Card, CardContent } from '@/components/ui/card'

export function MetricCard({ metric }: { metric: Metric }) {
  return (
    <Card className="overflow-hidden border-border/70 bg-card/80 py-0">
      <CardContent className="p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm text-muted-foreground">{metric.label}</p>
            <p className="mt-2 text-3xl font-semibold tracking-tight">{metric.value}</p>
          </div>
          <div className="rounded-xl bg-primary/10 p-2 text-primary">
            <metric.icon className="size-5" />
          </div>
        </div>
        <p className="mt-4 text-xs text-muted-foreground">{metric.hint}</p>
      </CardContent>
    </Card>
  )
}
