"use client"

import { useState, type ReactNode } from "react"

import { cn } from "@/lib/utils"

type StepperItem = {
  title: string
  description?: string
}

type StepperState = {
  current: number
  isFirst: boolean
  isLast: boolean
  canGoNext: boolean
  next: () => void
  previous: () => void
  goTo: (step: number) => void
}

type StepperProps = {
  steps: readonly StepperItem[]
  className?: string
  rootClassName?: string
  defaultStep?: number
  canAdvance?: (step: number) => boolean
  children?: (state: StepperState) => ReactNode
}

function Stepper({ steps, className, rootClassName, defaultStep = 0, canAdvance = () => true, children }: StepperProps) {
  const [current, setCurrent] = useState(defaultStep)
  const isFirst = current === 0
  const isLast = current === steps.length - 1
  const canGoNext = canAdvance(current)

  function goTo(step: number) {
    setCurrent(Math.max(0, Math.min(steps.length - 1, step)))
  }

  function next() {
    if (!canGoNext) return
    goTo(current + 1)
  }

  function previous() {
    goTo(current - 1)
  }

  return (
    <div data-slot="stepper-root" className={cn("space-y-6", rootClassName)}>
      <nav data-slot="stepper" className={cn("w-full", className)} aria-label="步骤">
        <ol className="flex w-full items-start">
          {steps.map((step, index) => {
            const state = index < current ? "complete" : index === current ? "active" : "pending"
            return (
              <li key={step.title} data-state={state} aria-current={state === "active" ? "step" : undefined} className="relative flex flex-1 flex-col items-center px-2 text-center">
                {index < steps.length - 1 && (
                  <span
                    aria-hidden="true"
                    className={cn(
                      "absolute top-5 left-1/2 h-px w-full bg-border transition-colors",
                      index < current && "bg-primary"
                    )}
                  />
                )}
                <button
                  type="button"
                  disabled={index > current}
                  onClick={() => goTo(index)}
                  className="relative z-10 flex flex-col items-center disabled:cursor-default"
                  aria-label={`第 ${index + 1} 步：${step.title}`}
                >
                  <span
                    className={cn(
                      "flex size-10 items-center justify-center rounded-full border bg-background text-sm font-semibold text-muted-foreground transition-all",
                      state === "active" && "border-primary bg-primary text-primary-foreground shadow-sm ring-4 ring-primary/15",
                      state === "complete" && "border-primary bg-primary text-primary-foreground",
                      index <= current && "hover:ring-4 hover:ring-primary/10"
                    )}
                  >
                    {index + 1}
                  </span>
                  <span className="mt-3 max-w-32 text-sm font-medium text-foreground">{step.title}</span>
                  {step.description && <span className="mt-1 max-w-36 text-xs leading-4 text-muted-foreground">{step.description}</span>}
                </button>
              </li>
            )
          })}
        </ol>
      </nav>
      {children?.({ current, isFirst, isLast, canGoNext, next, previous, goTo })}
    </div>
  )
}

export { Stepper, type StepperItem, type StepperState }
