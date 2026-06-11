package server

import (
	"context"
	"fmt"
	"sync"

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

	// 优雅关闭支持
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newRunScheduler(maxRunning int, dispatchRun func(runQueueItem)) *RunScheduler {
	if maxRunning <= 0 {
		maxRunning = 1
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &RunScheduler{
		queue:       queue.New[runQueueItem](defaultRunQueueSize),
		semaphore:   make(chan struct{}, maxRunning),
		dispatchRun: dispatchRun,
		ctx:         ctx,
		cancel:      cancel,
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
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			// 优雅关闭：处理完队列中的剩余项后退出
			s.queue.Close()
			return
		case item, ok := <-s.queue.Items():
			if !ok {
				// 队列已关闭
				return
			}
			s.semaphore <- struct{}{}
			s.wg.Add(1)
			go func(item runQueueItem) {
				defer func() {
					<-s.semaphore
					s.wg.Done()
				}()
				s.dispatchRun(item)
			}(item)
		}
	}
}

// Shutdown 优雅关闭调度器，等待所有运行完成。
func (s *RunScheduler) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}

	// 1. 发送取消信号
	s.cancel()

	// 2. 等待所有运行完成（带超时）
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
