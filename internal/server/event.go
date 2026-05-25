package server

import "sync"

// subscriber 持有单个订阅者的带缓冲事件通道。
type subscriber struct {
	ch chan Event
}

// eventBus 按 RunID 分组管理订阅者，负责事件的发布与通道生命周期管理。
type eventBus struct {
	mu          sync.Mutex
	subscribers map[RunID][]*subscriber
}

func newEventBus() *eventBus {
	return &eventBus{
		subscribers: make(map[RunID][]*subscriber),
	}
}

// Subscribe 注册对指定 RunID 的订阅，返回只读事件通道和取消函数。
// 取消函数调用后通道被关闭，range 循环自然退出。
// 通道容量为 64；若消费者处理过慢，后续事件将被丢弃（非阻塞发布）。
func (b *eventBus) Subscribe(runID RunID) (<-chan Event, CancelFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub := &subscriber{ch: make(chan Event, 64)}
	b.subscribers[runID] = append(b.subscribers[runID], sub)

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.removeLocked(runID, sub)
	}
	return sub.ch, cancel
}

// Publish 向该 RunID 的所有订阅者非阻塞地投递事件。
func (b *eventBus) Publish(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subscribers[event.RunID] {
		select {
		case sub.ch <- event:
		default:
			// 消费者过慢时丢弃，避免阻塞发布方
		}
	}
}

// CloseRun 关闭该 RunID 下所有订阅通道并清理条目。
// 必须在该 RunID 的最后一个 Publish 调用之后执行，以确保不丢失末尾事件。
func (b *eventBus) CloseRun(runID RunID) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sub := range b.subscribers[runID] {
		close(sub.ch)
	}
	delete(b.subscribers, runID)
}

// removeLocked 从订阅列表中移除 sub 并关闭其通道（已持锁时调用）。
func (b *eventBus) removeLocked(runID RunID, sub *subscriber) {
	subs := b.subscribers[runID]
	for i, s := range subs {
		if s == sub {
			b.subscribers[runID] = append(subs[:i], subs[i+1:]...)
			close(sub.ch)
			return
		}
	}
}
