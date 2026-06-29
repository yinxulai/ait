import { useState } from 'react'
import { CategoryScale, Chart as ChartJS, Filler, Legend as ChartLegend, LinearScale, LineElement, PointElement, Tooltip as ChartTooltip } from 'chart.js'

import { Card, CardContent } from '@/components/ui/card'
import { DashboardHeader, TaskSidebarContent } from '@/components/dashboard-layout'
import { RunDetail, TaskRunHistory } from '@/components/run-detail'
import { CreateTaskSheet, TaskOverview } from '@/components/task-editor'
import { useTaskWorkspace } from '@/hooks/use-task-workspace'
import { protocolOptions, taskMode } from '@/lib/task-utils'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Filler, ChartTooltip, ChartLegend)

function App() {
  const [taskDrawerOpen, setTaskDrawerOpen] = useState(false)
  const workspace = useTaskWorkspace()
  const { query, setQuery, taskList, taskNavItems, filteredTasks, protocols, selectedTask, selectedRun, selectedRunState, selectedRequest, runRequests, taskRuns, requestsByRun, startingTaskId, loadingMessage, errorMessage, chooseTaskById, chooseRun, createTask, startTaskRun, setSelectedRequestId } = workspace

  const standardTaskCount = taskList.filter((task) => taskMode(task) === 'standard').length
  const turboTaskCount = taskList.filter((task) => taskMode(task) === 'turbo').length
  const integrityTaskCount = taskList.filter((task) => taskMode(task) === 'integrity').length
  const protocolOptionsForCreate = protocols.length > 0 ? protocols.map((protocol) => protocol.id) : [...protocolOptions]
  function chooseTaskAndClose(taskId: string) {
    chooseTaskById(taskId)
    setTaskDrawerOpen(false)
  }

  return (
    <main className="min-h-screen bg-muted/30 text-foreground lg:h-screen lg:overflow-hidden">
      <div className="grid min-h-screen lg:h-full lg:min-h-0 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="hidden min-h-0 border-r bg-sidebar/95 lg:block">
          <TaskSidebarContent items={taskNavItems} totalTaskCount={taskList.length} query={query} onQueryChange={setQuery} onChooseTask={chooseTaskAndClose} />
        </aside>

        <section className="min-h-0 min-w-0 bg-background lg:overflow-y-auto">
          <DashboardHeader taskDrawerOpen={taskDrawerOpen} onTaskDrawerOpenChange={setTaskDrawerOpen} taskDrawerContent={<TaskSidebarContent items={taskNavItems} totalTaskCount={taskList.length} query={query} onQueryChange={setQuery} onChooseTask={chooseTaskAndClose} />} createTaskButton={<CreateTaskSheet onCreate={createTask} protocolOptions={protocolOptionsForCreate} protocols={protocols} variant="primary" />} filteredTaskCount={filteredTasks.length} totalTaskCount={taskList.length} standardTaskCount={standardTaskCount} turboTaskCount={turboTaskCount} integrityTaskCount={integrityTaskCount} />

          <div className="w-full space-y-4 p-4 sm:p-6">
            {errorMessage && <div className="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">{errorMessage}</div>}
            {!selectedTask && <Card className="rounded-2xl border bg-card shadow-none ring-0"><CardContent className="flex min-h-80 items-center justify-center p-6 text-sm text-muted-foreground">{loadingMessage || '暂无任务，请先创建任务。'}</CardContent></Card>}
            {selectedTask && <>
              <TaskOverview task={selectedTask} onCreate={createTask} onStartRun={startTaskRun} starting={startingTaskId === selectedTask.id} protocolOptions={protocolOptionsForCreate} protocols={protocols} />
              <TaskRunHistory runs={taskRuns} selectedRun={selectedRun} onChooseRun={chooseRun} samplesByRun={requestsByRun} />
              <RunDetail run={selectedRun} state={selectedRunState} requests={runRequests} selectedRequest={selectedRequest} onSelectRequest={setSelectedRequestId} />
            </>}
          </div>
        </section>
      </div>
    </main>
  )
}

export default App
