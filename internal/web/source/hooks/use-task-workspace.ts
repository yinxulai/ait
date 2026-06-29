import { useEffect, useEffectEvent, useMemo, useState } from 'react'

import { createTask as createTaskAPI, getRunRequests, getRunState, listProtocols, listTaskRuns, listTasks, startTaskRun as startTaskRunAPI, subscribeRunEvents, type ProtocolMeta, type RequestDetail, type RunState, type RunSummary, type Task, type TaskConfig } from '@/api'
import { isTerminalRunStatus, mergeStartedRun, runSummaryFromState, upsertRunSummary } from '@/lib/run-utils'
import { formatDate, formatPercent, inputJsonFromDraft, modeLabel, requestKey, taskEndpoint, taskMode, taskModel, taskProtocol, type TaskDraft } from '@/lib/task-utils'
import type { TaskNavItem } from '@/components/dashboard-layout'

export function useTaskWorkspace() {
  const [query, setQuery] = useState('')
  const [taskList, setTaskList] = useState<Task[]>([])
  const [runsByTask, setRunsByTask] = useState<Record<string, RunSummary[]>>({})
  const [statesByRun, setStatesByRun] = useState<Record<string, RunState>>({})
  const [requestsByRun, setRequestsByRun] = useState<Record<string, RequestDetail[]>>({})
  const [protocols, setProtocols] = useState<ProtocolMeta[]>([])
  const [selectedTaskId, setSelectedTaskId] = useState('')
  const [selectedRunId, setSelectedRunId] = useState('')
  const [selectedRequestId, setSelectedRequestId] = useState('')
  const [startingTaskId, setStartingTaskId] = useState('')
  const [loadingMessage, setLoadingMessage] = useState('加载任务中...')
  const [errorMessage, setErrorMessage] = useState('')

  useEffect(() => {
    let cancelled = false
    async function loadInitialData() {
      try {
        const [tasks, protocolMetas] = await Promise.all([listTasks(), listProtocols()])
        if (cancelled) return
        setTaskList(tasks)
        setProtocols(protocolMetas)
        setSelectedTaskId((current) => current || tasks[0]?.id || '')
        setErrorMessage('')
      } catch (error) {
        if (!cancelled) setErrorMessage(error instanceof Error ? error.message : '加载任务失败')
      } finally {
        if (!cancelled) setLoadingMessage('')
      }
    }
    loadInitialData()
    return () => { cancelled = true }
  }, [])

  useEffect(() => {
    if (!selectedTaskId) return
    let cancelled = false
    async function loadRuns() {
      try {
        const runList = await listTaskRuns(selectedTaskId)
        if (cancelled) return
        setRunsByTask((current) => ({ ...current, [selectedTaskId]: runList }))
        setSelectedRunId((current) => runList.some((run) => run.run_id === current) ? current : runList[0]?.run_id || '')
        setSelectedRequestId('')
        setErrorMessage('')
      } catch (error) {
        if (!cancelled) setErrorMessage(error instanceof Error ? error.message : '加载执行记录失败')
      }
    }
    loadRuns()
    return () => { cancelled = true }
  }, [selectedTaskId])

  useEffect(() => {
    if (!selectedRunId) return
    let cancelled = false
    async function loadRequests() {
      try {
        const state = await getRunState(selectedRunId).catch(() => undefined)
        const requestList = state?.requests?.length ? state.requests : await getRunRequests(selectedRunId)
        if (cancelled) return
        if (state) setStatesByRun((current) => ({ ...current, [selectedRunId]: state }))
        setRequestsByRun((current) => ({ ...current, [selectedRunId]: requestList }))
        setSelectedRequestId((current) => requestList.some((request) => requestKey(request) === current) ? current : requestKey(requestList[0]))
        setErrorMessage('')
      } catch (error) {
        if (!cancelled) setErrorMessage(error instanceof Error ? error.message : '加载请求样本失败')
      }
    }
    loadRequests()
    return () => { cancelled = true }
  }, [selectedRunId])

  const filteredTasks = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return taskList
    return taskList.filter((task) => [task.name, taskModel(task), taskProtocol(task), taskEndpoint(task), modeLabel[taskMode(task)]].some((text) => text.toLowerCase().includes(keyword)))
  }, [query, taskList])

  const selectedTask = taskList.find((task) => task.id === selectedTaskId) ?? taskList[0]
  const taskRuns = selectedTask ? runsByTask[selectedTask.id] ?? [] : []
  const selectedRun = taskRuns.find((run) => run.run_id === selectedRunId) ?? taskRuns[0]
  const selectedRunState = selectedRun ? statesByRun[selectedRun.run_id] : undefined
  const runRequests = selectedRun ? requestsByRun[selectedRun.run_id] ?? [] : []
  const selectedRequest = runRequests.find((request) => requestKey(request) === selectedRequestId) ?? runRequests[0]
  const streamRunId = selectedRun?.run_id ?? ''
  const streamRunStatus = selectedRun?.status
  const streamTaskId = selectedTask?.id ?? ''

  const handleRunState = useEffectEvent((state: RunState) => {
    const task = taskList.find((item) => item.id === state.task_id)
    if (task) updateRunFromState(state, task)
  })

  const handleRunStreamClose = useEffectEvent((taskId: string) => {
    refreshRunHistory(taskId)
  })

  const handleRunStreamError = useEffectEvent((runId: string) => {
    const task = taskList.find((item) => item.id === selectedTaskId)
    if (task) refreshSelectedRun(runId, task)
  })

  useEffect(() => {
    if (!streamRunId || !streamTaskId || !streamRunStatus || isTerminalRunStatus(streamRunStatus)) return

    const unsubscribe = subscribeRunEvents(streamRunId, handleRunState, () => handleRunStreamClose(streamTaskId), () => handleRunStreamError(streamRunId))

    return () => {
      unsubscribe()
    }
  }, [streamRunId, streamRunStatus, streamTaskId])

  const taskNavItems = useMemo<TaskNavItem[]>(() => filteredTasks.map((task) => {
    const runs = runsByTask[task.id] ?? []
    const latestRun = task.latest_run ?? runs[0]
    return {
      id: task.id,
      name: task.name,
      mode: taskMode(task),
      modeLabel: modeLabel[taskMode(task)],
      model: taskModel(task),
      updatedAt: formatDate(task.updated_at),
      runsCount: runs.length,
      successRate: latestRun ? formatPercent(latestRun.success_rate) : undefined,
      latestFailed: latestRun?.status === 'failed',
      active: selectedTask?.id === task.id,
    }
  }), [filteredTasks, runsByTask, selectedTask?.id])

  function chooseTask(task: Task) {
    setSelectedTaskId(task.id)
  }

  function chooseTaskById(taskId: string) {
    const task = taskList.find((item) => item.id === taskId)
    if (task) chooseTask(task)
  }

  function chooseRun(run: RunSummary) {
    setSelectedRunId(run.run_id)
  }

  async function createTask(draft: TaskDraft) {
    const config: TaskConfig = { name: draft.name.trim(), input: inputJsonFromDraft(draft) }
    const task = await createTaskAPI(config)
    setTaskList((current) => [task, ...current])
    setSelectedTaskId(task.id)
    setSelectedRunId('')
    setSelectedRequestId('')
  }

  async function startTaskRun(task: Task) {
    if (startingTaskId) return
    setStartingTaskId(task.id)
    try {
      const { run_id } = await startTaskRunAPI(task.id)
      const [state, runList] = await Promise.all([
        getRunState(run_id).catch(() => undefined),
        listTaskRuns(task.id).catch(() => []),
      ])
      const nextRuns = mergeStartedRun(runList, state, task)
      setRunsByTask((current) => ({ ...current, [task.id]: nextRuns }))
      setTaskList((current) => current.map((item) => item.id === task.id ? { ...item, latest_run: nextRuns[0] ?? item.latest_run } : item))
      setSelectedTaskId(task.id)
      setSelectedRunId(nextRuns.some((run) => run.run_id === run_id) ? run_id : nextRuns[0]?.run_id ?? '')
      setSelectedRequestId('')
      setErrorMessage('')
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : '启动任务失败')
    } finally {
      setStartingTaskId('')
    }
  }

  function updateRunFromState(state: RunState, task: Task) {
    setStatesByRun((current) => ({ ...current, [state.run_id]: state }))
    setRequestsByRun((current) => ({ ...current, [state.run_id]: state.requests ?? [] }))
    setSelectedRequestId((current) => state.requests?.some((request) => requestKey(request) === current) ? current : requestKey(state.requests?.[0]))

    const summary = runSummaryFromState(state, task)
    setRunsByTask((current) => ({
      ...current,
      [task.id]: upsertRunSummary(current[task.id] ?? [], summary),
    }))
    setTaskList((current) => current.map((item) => item.id === task.id ? { ...item, latest_run: summary } : item))
  }

  async function refreshSelectedRun(runId: string, task: Task) {
    const state = await getRunState(runId).catch(() => undefined)
    if (state) updateRunFromState(state, task)
  }

  async function refreshRunHistory(taskId: string) {
    const runList = await listTaskRuns(taskId).catch(() => undefined)
    if (runList) setRunsByTask((current) => ({ ...current, [taskId]: runList }))
  }

  return {
    query,
    setQuery,
    taskList,
    taskNavItems,
    filteredTasks,
    protocols,
    selectedTask,
    selectedRun,
    selectedRunState,
    selectedRequest,
    selectedRequestId,
    runRequests,
    taskRuns,
    requestsByRun,
    startingTaskId,
    loadingMessage,
    errorMessage,
    chooseTaskById,
    chooseRun,
    createTask,
    startTaskRun,
    setSelectedRequestId,
  }
}
