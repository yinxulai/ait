package server

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/yinxulai/ait/internal/server/queue"
)

// RequestQueue 使用公共 FIFO queue 和 worker pool 执行一批请求。
type RequestQueue struct {
	queue *queue.Queue[RequestJob]
}

type RequestQueueHooks struct {
	OnQueued  func(RequestJob)
	OnStarted func(RequestJob)
	OnSkipped func(RequestJob)
	OnDone    func(RequestResult)
}

func NewRequestQueue(capacity int) *RequestQueue {
	return &RequestQueue{queue: queue.New[RequestJob](capacity)}
}

func RunRequestBatch(ctx context.Context, jobs []RequestJob, concurrency int, executor *RequestExecutor, hooks RequestQueueHooks) int {
	if concurrency <= 0 {
		concurrency = 1
	}
	if len(jobs) == 0 {
		return 0
	}

	requestQueue := NewRequestQueue(concurrency)
	var wg sync.WaitGroup
	var launched int64

	for workerID := 0; workerID < concurrency; workerID++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range requestQueue.queue.Items() {
				select {
				case <-ctx.Done():
					if hooks.OnSkipped != nil {
						hooks.OnSkipped(job)
					}
					continue
				default:
				}

				atomic.AddInt64(&launched, 1)
				if hooks.OnStarted != nil {
					hooks.OnStarted(job)
				}
				result := executor.Execute(ctx, job)
				if hooks.OnDone != nil {
					hooks.OnDone(result)
				}
			}
		}()
	}

	for _, job := range jobs {
		if hooks.OnQueued != nil {
			hooks.OnQueued(job)
		}
		if err := requestQueue.queue.EnqueueUntil(ctx.Done(), job); err != nil {
			if hooks.OnSkipped != nil {
				hooks.OnSkipped(job)
			}
			break
		}
	}
	requestQueue.queue.Close()
	wg.Wait()
	return int(atomic.LoadInt64(&launched))
}
