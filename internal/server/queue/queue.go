package queue

import (
	"errors"
	"sync"
)

var (
	ErrClosed = errors.New("queue is closed")
	ErrFull   = errors.New("queue is full")
)

// Queue 是进程内 FIFO 队列。
//
// 它是轻量通用组件，用于运行队列、请求队列等内存调度场景。
// 队列只负责按提交顺序传递 item，不负责持久化、重试、优先级或业务状态更新。
type Queue[T any] struct {
	items chan T
	once  sync.Once
}

func New[T any](capacity int) *Queue[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &Queue[T]{items: make(chan T, capacity)}
}

// Enqueue 非阻塞入队；队列满时返回 ErrFull。
func (q *Queue[T]) Enqueue(item T) (err error) {
	if q == nil {
		return ErrClosed
	}
	defer func() {
		if recover() != nil {
			err = ErrClosed
		}
	}()
	select {
	case q.items <- item:
		return nil
	default:
		return ErrFull
	}
}

// EnqueueUntil 阻塞入队，直到 item 入队、done 关闭或队列关闭。
func (q *Queue[T]) EnqueueUntil(done <-chan struct{}, item T) (err error) {
	if q == nil {
		return ErrClosed
	}
	defer func() {
		if recover() != nil {
			err = ErrClosed
		}
	}()
	select {
	case <-done:
		return ErrClosed
	case q.items <- item:
		return nil
	}
}

// Items 返回只读消费通道。消费者按 FIFO 顺序 range 该通道。
func (q *Queue[T]) Items() <-chan T {
	if q == nil {
		return nil
	}
	return q.items
}

func (q *Queue[T]) Close() {
	if q == nil {
		return
	}
	q.once.Do(func() {
		close(q.items)
	})
}
