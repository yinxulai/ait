import { Activity, GitBranch, Settings } from 'lucide-react'

import { navItems } from '@/data'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-background text-foreground">
      <aside className="fixed inset-y-0 left-0 z-30 hidden w-72 border-r bg-sidebar/80 px-4 py-5 backdrop-blur xl:block">
        <div className="flex items-center gap-3 px-2">
          <div className="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-lg shadow-primary/20">
            <Activity className="size-5" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">AI Benchmark</p>
            <h1 className="text-xl font-semibold tracking-tight">AIT Console</h1>
          </div>
        </div>

        <nav className="mt-8 space-y-1">
          {navItems.map((item) => (
            <button
              key={item.label}
              className={cn(
                'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors',
                item.active
                  ? 'bg-sidebar-primary text-sidebar-primary-foreground shadow-sm'
                  : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              )}
            >
              <item.icon className="size-4" />
              {item.label}
            </button>
          ))}
        </nav>

        <div className="absolute inset-x-4 bottom-5 rounded-xl border bg-card p-4 text-sm shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">服务状态</span>
            <span className="inline-flex items-center gap-1.5 text-emerald-500">
              <span className="size-2 rounded-full bg-emerald-500" />
              在线
            </span>
          </div>
          <p className="mt-3 text-xs leading-5 text-muted-foreground">
            当前为前端静态实现，后续可对接 Go Server 的任务、运行和事件 API。
          </p>
        </div>
      </aside>

      <div className="xl:pl-72">
        <header className="sticky top-0 z-20 border-b bg-background/85 backdrop-blur">
          <div className="flex h-16 items-center justify-between px-4 sm:px-6 lg:px-8">
            <div>
              <p className="text-xs font-medium uppercase tracking-[0.24em] text-muted-foreground">AIT Web UI</p>
              <h2 className="text-base font-semibold sm:text-lg">模型性能测试工作台</h2>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" className="hidden sm:inline-flex">
                <GitBranch className="size-4" />
                yinxulai/ait
              </Button>
              <Button variant="ghost" size="icon" aria-label="设置">
                <Settings className="size-4" />
              </Button>
            </div>
          </div>
        </header>

        <main className="px-4 py-6 sm:px-6 lg:px-8">{children}</main>
      </div>
    </div>
  )
}
