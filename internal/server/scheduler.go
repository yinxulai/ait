package server

import (
	"fmt"

	"github.com/yinxulai/ait/internal/server/queue"
	"github.com/yinxulai/ait/internal/server/types"
)

const defaultRunQueueSize = 1024

// runQueueItem 是一次待调度运行的内存队列项。
type runQueueItem struct {
	RunID   RunID
	TaskID  string
	TaskDef types.TaskDefinition
	Input   types.Input
	Mode    string
}

// RunScheduler 负责按 FIFO 调度运行，并限制全局同时运行数量。
type RunScheduler struct {
	queue       *queue.Queue[runQueueItem]
	semaphore   chan struct{}
	dispatchRun func(runQueueItem)
}

func newRunScheduler(maxRunning int, dispatchRun func(runQueueItem)) *RunScheduler {
	if maxRunning <= 0 {
		maxRunning = 1
	}
	s := &RunScheduler{
		queue:       queue.New[runQueueItem](defaultRunQueueSize),
		semaphore:   make(chan struct{}, maxRunning),
		dispatchRun: dispatchRun,
	}
	go s.loop()
	return s
}

func (s *RunScheduler) Enqueue(item runQueueItem) error {
	if s == nil {
		return fmt.Errorf("run scheduler is not initialized")
	}
	if err := s.queue.Enqueue(item); err != nil {
		return fmt.Errorf("enqueue run: %w", err)
	}
	return nil
}

func (s *RunScheduler) loop() {
	for item := range s.queue.Items() {
		s.semaphore <- struct{}{}
		go func(item runQueueItem) {
			defer func() { <-s.semaphore }()
			s.dispatchRun(item)
		}(item)
	}
}
