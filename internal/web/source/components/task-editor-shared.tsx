import { type ReactNode, useId } from 'react'
import { Gauge, ShieldCheck, Zap } from 'lucide-react'

import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'
import { promptModeLabel, protocolLabel } from '@/lib/task-utils'
import { type PromptMode, type TaskMode } from '@/api'

export function FormField({ label, description, required, children }: { label: string; description?: string; required?: boolean; children: ReactNode }) {
  return (
    <div className="grid gap-1.5 text-sm">
      <Label className="flex items-center gap-1">{label}{required && <span className="text-destructive">*</span>}</Label>
      {description && <p className="text-xs leading-5 text-muted-foreground">{description}</p>}
      {children}
    </div>
  )
}

export function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 rounded-xl bg-muted/50 px-3 py-2">
      <div className="text-[11px] text-muted-foreground">{label}</div>
      <div className="truncate text-sm font-medium" title={value}>{value}</div>
    </div>
  )
}

export function BooleanToggle({ label, description, value, onChange }: { label: string; description?: string; value: boolean; onChange: (value: boolean) => void }) {
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

export function NumberStat({ label, value, compact }: { label: string; value: string; compact?: boolean }) {
  return (
    <div>
      <div className={cn('font-semibold tracking-tight', compact ? 'text-base' : 'text-2xl')}>{value}</div>
      <div className="text-xs text-muted-foreground">{label}</div>
    </div>
  )
}

export function InlineSwitch({ label, enabled }: { label: string; enabled: boolean }) {
  return (
    <div className="flex items-center justify-between rounded-2xl bg-muted/50 px-4 py-3 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className={cn('rounded-full px-2 py-0.5 text-xs font-medium', enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300' : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-900 dark:text-zinc-300')}>{enabled ? '开启' : '关闭'}</span>
    </div>
  )
}

export function CodeBlock({ label, value, icon }: { label: string; value: string; icon?: ReactNode }) {
  const isJson = value.trim().startsWith('{') || value.trim().startsWith('[')
  const prettyValue = (() => {
    if (!isJson) return value
    try {
      return JSON.stringify(JSON.parse(value), null, 2)
    } catch {
      return value
    }
  })()

  return (
    <div className="overflow-hidden rounded-xl bg-muted/25">
      <div className="flex items-center gap-1.5 border-b bg-background/80 px-3 py-2 text-xs font-medium text-muted-foreground">{icon}{label}</div>
      <pre className="max-h-44 overflow-auto bg-muted/55 px-3 py-3 text-xs leading-5 text-foreground/90"><code>{prettyValue}</code></pre>
    </div>
  )
}

export function OptionPicker<T extends string>({ value, options, onChange }: { value: T; options: readonly T[]; onChange: (value: T) => void }) {
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

export function ModeIcon({ mode, className }: { mode: TaskMode; className?: string }) {
  const Icon = mode === 'standard' ? Gauge : mode === 'turbo' ? Zap : ShieldCheck
  return <Icon className={className} />
}

function optionDisplayName(option: string) {
  return promptModeLabel[option as PromptMode] ?? protocolLabel[option] ?? option
}
